package processors

import (
	"oak-compiler/ast"
	"oak-compiler/ast/parsed"
	"oak-compiler/common"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	KwModule   = "module"
	KwImport   = "import"
	KwAs       = "as"
	KwExposing = "exposing"
	KwInfix    = "infix"
	KwAlias    = "alias"
	KwData     = "data"
	KwDef      = "def"
	KwHidden   = "hidden"
	KwExtern   = "extern"
	KwLeft     = "left"
	KwRight    = "right"
	KwNon      = "non"
	KwIf       = "if"
	KwThen     = "then"
	KwElse     = "else"
	KwLet      = "let"
	KwIn       = "in"
	KwSelect   = "select"
	KwCase     = "case"

	SeqComment          = "//"
	SeqCommentStart     = "/*"
	SeqCommentEnd       = "*/"
	SeqExposingAll      = "*"
	SeqParenthesisOpen  = "("
	SeqParenthesisClose = ")"
	SeqBracketsOpen     = "["
	SeqBracketsClose    = "]"
	SeqBracesOpen       = "{"
	SeqBracesClose      = "}"
	SeqComma            = ","
	SeqColon            = ":"
	SeqEqual            = "="
	SeqBar              = "|"
	SeqUnderscore       = "_"
	SeqDot              = "."
	SeqMinus            = "-"
	SeqLambda           = "\\("
	SeqLambdaBind       = "->"
	SeqCaseBind         = "->"
	SeqInfixChars       = "!#$%&*+-/:;<=>?^|~`"

	SmbNewLine     = '\n'
	SmbQuoteString = '"'
	SmbQuoteChar   = '\''
	SmbEscape      = '\\'
)

// - void skip*() skips sequence if it can, returns nothing, does not set error.
// - * read*() reads something, returns NULL if cannot, does not set error. eats all trailing whitespace and comments.
// - bool parse(..., *out) parses something, can set error (returns false in that case) if failed in a middle of parsing,
//      in other case returns true. sets `out` to NULL if nothing read. eats all trailing whitespace and comments.

type Source struct {
	filePath string
	cursor   uint64
	text     []rune
}

func loc(src *Source, cursor uint64) ast.Location {
	return ast.Location{FilePath: src.filePath, Position: cursor}
}

func setErrorSource(src Source, msg string) {
	panic(common.Error{
		Location: ast.Location{
			FilePath: src.filePath,
			Position: src.cursor,
		},
		Message: msg,
	})
}

func isOk(src *Source) bool {
	return src.cursor < uint64(len(src.text))
}

func isIdentChar(c rune, first *bool, qualified bool) bool {
	wasFirst := *first
	*first = false

	if unicode.IsLetter(c) {
		return true
	}
	if !wasFirst {
		if ('_' == c) || ('`' == c) || unicode.IsDigit(c) {
			return true
		}
		if qualified {
			if '.' == c {
				*first = true
				return true
			}
		}
	}
	return false
}

func isInfixChar(c rune) bool {
	for _, x := range SeqInfixChars {
		if c == x {
			return true
		}
	}
	return false
}

func readSequence(src *Source, value string) *string {
	start := src.cursor
	for _, c := range []rune(value) {
		if !isOk(src) || src.text[src.cursor] != c {
			src.cursor = start
			return nil
		}
		src.cursor++
	}
	return &value
}

func skipWhiteSpace(src *Source) {
	for isOk(src) && unicode.IsSpace(src.text[src.cursor]) {
		src.cursor++
	}
}

func skipComment(src *Source) {
	if !isOk(src) {
		return
	}

	skipWhiteSpace(src)
	if nil != readSequence(src, SeqComment) {

		for isOk(src) && SmbNewLine != src.text[src.cursor] {
			src.cursor++
		}
		src.cursor++ //skip SMB_NEW_LINE
	} else if nil != readSequence(src, SeqCommentStart) {
		level := 1
		for isOk(src) {
			if nil != readSequence(src, SeqCommentStart) {
				level++
			} else if nil != readSequence(src, SeqCommentEnd) {
				level--
				if 0 == level {
					break
				}
			}
			src.cursor++
		}
		if 0 != level {
			return
		}
	} else {
		return
	}

	skipWhiteSpace(src)
	skipComment(src)
}

func readIdentifier(src *Source, qualified bool) *ast.QualifiedIdentifier {
	start := src.cursor
	first := true
	for isOk(src) && isIdentChar(src.text[src.cursor], &first, qualified) {
		src.cursor++
	}

	if start != src.cursor {
		end := src.cursor
		skipComment(src)
		result := ast.QualifiedIdentifier(src.text[start:end])
		return &result
	}

	src.cursor = start
	return nil
}

func parseInt(src *Source) *int64 {
	if !isOk(src) {
		return nil
	}

	pos := src.cursor

	strValue, base := readIntegerPart(src, true)

	if strValue == "" {
		src.cursor = pos
		return nil
	}

	value, err := strconv.ParseInt(strValue, base, 64)
	if err != nil {
		setErrorSource(*src, "failed to parse integer: "+err.Error())
	}

	skipComment(src)
	return &value
}

func parseFloat(src *Source) *float64 {
	if !isOk(src) {
		return nil
	}
	pos := src.cursor

	first, _ := readIntegerPart(src, false)
	if first == "" {
		return nil
	}

	if readSequence(src, ".") != nil {
		second, base := readIntegerPart(src, false)
		if base == 0 {
			return nil
		}
		first += "." + second
	} else if readSequence(src, "e") != nil || readSequence(src, "E") != nil {
		var sign string
		if readSequence(src, "-") != nil {
			sign = "-"
		} else if readSequence(src, "+") != nil {
			sign = "+"
		} else {
			return nil
		}
		second, base := readIntegerPart(src, false)
		if base == 0 {
			return nil
		}
		first += "e" + sign + second
	}

	if isOk(src) && (unicode.IsLetter(src.text[src.cursor]) || unicode.IsNumber(src.text[src.cursor])) {
		src.cursor = pos
		return nil
	}
	skipComment(src)

	value, err := strconv.ParseFloat(first, 64)
	if err != nil {
		setErrorSource(*src, "failed to parse float: "+err.Error())
	}
	return &value
}

var kNumBin = []rune{'0', '1'}
var kNumOct = []rune{'0', '1', '2', '3', '4', '5', '6', '7'}
var kNumDec = []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
var kNumHex = []rune{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F'}

func readIntegerPart(src *Source, allowBases bool) (string, int) {
	if !isOk(src) {
		return "", 0
	}

	base := 10
	if allowBases {
		if nil != readSequence(src, "0x") || nil != readSequence(src, "0X") {
			base = 16
		} else if nil != readSequence(src, "0b") || nil != readSequence(src, "0B") {
			base = 2
		} else if nil != readSequence(src, "0o") || nil != readSequence(src, "0O") {
			base = 8
		}
	}

	var value []rune
	var nums []rune
	switch base {
	case 2:
		nums = kNumBin
		break
	case 8:
		nums = kNumOct
		break
	case 10:
		nums = kNumDec
		break
	case 16:
		nums = kNumHex
		break
	}
	for {
		if nil != readSequence(src, "_") {
			continue
		}
		if isOk(src) && slices.Contains(nums, src.text[src.cursor]) {
			value = append(value, src.text[src.cursor])
			src.cursor++
		} else {
			break
		}
	}

	if len(value) == 0 {
		if base == 8 {
			return "0", 10
		}
		return "", 0
	}

	return string(value), base
}

func readExact(src *Source, value string) bool {
	if nil != readSequence(src, value) {
		skipComment(src)
		return true
	}
	return false
}

func parseChar(src *Source) *rune {
	if !isOk(src) {
		return nil
	}

	if SmbQuoteChar != src.text[src.cursor] {
		return nil
	}
	src.cursor++
	if !isOk(src) {
		setErrorSource(*src, "character is not closed before end of file")
	}

	src.cursor++

	if !isOk(src) || SmbQuoteChar != src.text[src.cursor] {
		setErrorSource(*src, "expected "+string(SmbQuoteChar)+"here")
	}
	src.cursor++

	r := src.text[src.cursor-1]
	skipComment(src)
	return &r
}

func parseString(src *Source) *string {

	if !isOk(src) {
		return nil
	}

	start := src.cursor

	if SmbQuoteString != src.text[src.cursor] {
		return nil
	}

	skipNextQuote := true
	for {
		if !isOk(src) {
			setErrorSource(*src, "string is not closed before the end of file")
		}
		if SmbQuoteString == src.text[src.cursor] && !skipNextQuote {
			break
		}
		src.cursor++
		skipNextQuote = SmbEscape == src.text[src.cursor]
	}
	src.cursor++
	str := string(src.text[start+1 : src.cursor-1])
	skipComment(src)
	return &str
}

func parseNumber(src *Source) (iValue *int64, fValue *float64) {
	pos := src.cursor
	fv := parseFloat(src)
	fvPos := src.cursor

	src.cursor = pos
	iv := parseInt(src)

	if fv == nil {
		return iv, nil
	}
	if iv == nil {
		src.cursor = fvPos
		return nil, fv
	}

	if src.cursor != fvPos {
		src.cursor = fvPos
		return nil, fv
	}

	return iv, fv
}

func parseConst(src *Source) ast.ConstValue {
	r := parseChar(src)
	if nil != r {
		return ast.CChar{Value: *r}
	}

	s := parseString(src)
	if nil != s {
		return ast.CString{Value: *s}
	}

	i, f := parseNumber(src)

	if f != nil {
		return ast.CFloat{Value: *f}
	}
	if i != nil {
		return ast.CInt{Value: *i}
	}

	return nil
}

func parseInfixIdentifier(src *Source, withParenthesis bool) *ast.InfixIdentifier {
	if !isOk(src) {
		return nil
	}

	cursor := src.cursor

	if withParenthesis && !readExact(src, SeqParenthesisOpen) {
		return nil
	}

	start := src.cursor
	for isInfixChar(src.text[src.cursor]) {
		src.cursor++
	}
	end := src.cursor

	if end-start == 0 {
		src.cursor = cursor
		return nil
	}

	if withParenthesis && !readExact(src, SeqParenthesisClose) {
		setErrorSource(*src, "expected `)` here")
	}

	if 0 == end-start {
		src.cursor = cursor
		return nil
	}
	result := ast.InfixIdentifier(src.text[start:end])

	skipComment(src)

	return &result
}

func parseTypeParamNames(src *Source) []ast.Identifier {
	if !readExact(src, SeqBracketsOpen) {
		return nil
	}

	var result []ast.Identifier
	for {
		name := readIdentifier(src, false)
		if nil == name {
			setErrorSource(*src, "expected variable type name here")
		} else if !unicode.IsLower([]rune(*name)[0]) {
			setErrorSource(*src, "type parameter name should start with lowercase letter")
		} else {
			result = append(result, ast.Identifier(*name))
		}

		if readExact(src, SeqComma) {
			continue
		}
		if readExact(src, SeqBracketsClose) {
			break
		}
		setErrorSource(*src, "expected `,` or `]` here")
	}

	return result
}

func parseType(src *Source) parsed.Type {
	cursor := src.cursor

	//signature/tuple/unit
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return parsed.TUnit{Location: loc(src, cursor)}
		}

		var items []parsed.Type

		for {

			type_ := parseType(src)
			if nil == type_ {
				setErrorSource(*src, "expected type here")
			}
			items = append(items, type_)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `)` here")
		}

		if readExact(src, SeqColon) {
			ret := parseType(src)
			if nil == ret {
				setErrorSource(*src, "expected return type here")
			}
			return parsed.TFunc{Location: loc(src, cursor), Return: ret, Params: items}
		} else {
			if 1 == len(items) {
				return items[0]
			} else {
				return parsed.TTuple{Location: loc(src, cursor), Items: items}
			}
		}
	}

	//record
	if readExact(src, SeqBracesOpen) {
		recCursor := src.cursor
		ext := readIdentifier(src, true)
		if nil != ext && !readExact(src, SeqBar) {
			ext = nil
			src.cursor = recCursor
		}

		fields := map[ast.Identifier]parsed.Type{}

		for {
			name := readIdentifier(src, false)
			if nil == name {
				setErrorSource(*src, "expected field name here")
			}
			if !readExact(src, SeqColon) {
				setErrorSource(*src, "expected `:` here")
			}
			type_ := parseType(src)
			if nil == type_ {
				setErrorSource(*src, "expected field type here")
			}

			if _, ok := fields[ast.Identifier(*name)]; ok {
				setErrorSource(*src, "field with this name has already declared for the record")
			}
			fields[ast.Identifier(*name)] = type_

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `}` here")
		}

		return parsed.TRecord{Location: loc(src, cursor), Fields: fields}
	}

	if name := readIdentifier(src, true); nil != name {
		if unicode.IsLower([]rune(*name)[0]) {
			return parsed.TTypeParameter{Location: loc(src, cursor), Name: ast.Identifier(*name)}
		} else {
			var typeParams []parsed.Type
			if readExact(src, SeqBracketsOpen) {
				for {
					type_ := parseType(src)
					if nil == type_ {
						setErrorSource(*src, "expected type parameter here")
					}
					typeParams = append(typeParams, type_)

					if readExact(src, SeqComma) {
						continue
					}
					if readExact(src, SeqBracketsClose) {
						break
					}
					setErrorSource(*src, "expected `,` or `]`  here")
				}
			}

			return parsed.TNamed{Location: loc(src, cursor), Name: *name, Args: typeParams}
		}
	}
	return nil
}

func parsePattern(src *Source) parsed.Pattern {
	cursor := src.cursor

	//tuple/unit
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return finishParsePattern(src, parsed.PConst{
				Location: loc(src, cursor),
				Value:    ast.CUnit{},
			})
		}
		var items []parsed.Pattern
		for {
			item := parsePattern(src)
			if nil == item {
				setErrorSource(*src, "expected tuple item pattern here")
			}
			items = append(items, item)
			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `)` here")
		}
		if 1 == len(items) {
			return finishParsePattern(src, items[0])
		}
		return finishParsePattern(src, parsed.PTuple{
			Location: loc(src, cursor),
			Items:    items,
		})
	}

	//record
	if readExact(src, SeqBracesOpen) {
		var fields []parsed.PRecordField
		for {
			fieldCursor := src.cursor
			name := readIdentifier(src, false)
			if nil == name {
				setErrorSource(*src, "expected record field name here")
			}
			fields = append(fields, parsed.PRecordField{
				Location: loc(src, fieldCursor),
				Name:     ast.Identifier(*name),
			})

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `}` here")
		}

		return finishParsePattern(src, parsed.PRecord{
			Location: loc(src, cursor),
			Fields:   fields,
		})
	}

	//list
	if readExact(src, SeqBracketsOpen) {
		if readExact(src, SeqBracketsClose) {
			return finishParsePattern(src, parsed.PList{Location: loc(src, cursor), Items: nil})
		}

		var items []parsed.Pattern
		for {
			p := parsePattern(src)
			if nil == p {
				setErrorSource(*src, "expected list item pattern here")
			}
			items = append(items, p)
			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracketsClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `}` here")
		}

		return finishParsePattern(src, parsed.PList{Location: loc(src, cursor), Items: items})
	}

	//union
	name := readIdentifier(src, true)
	if nil != name && unicode.IsUpper([]rune(*name)[0]) {
		var items []parsed.Pattern
		if readExact(src, SeqParenthesisOpen) {
			for {
				item := parsePattern(src)
				if nil == item {
					setErrorSource(*src, "expected option value pattern here")
				}
				items = append(items, item)
				if readExact(src, SeqComma) {
					continue
				}
				if readExact(src, SeqParenthesisClose) {
					break
				}
				setErrorSource(*src, "expected `,` or `)` here")
			}
		}
		return finishParsePattern(src, parsed.PDataValue{Location: loc(src, cursor), Name: *name, Values: items})
	} else {
		src.cursor = cursor
	}

	name = readIdentifier(src, false)
	if nil != name && unicode.IsLower([]rune(*name)[0]) {
		return finishParsePattern(src, parsed.PNamed{Location: loc(src, cursor), Name: ast.Identifier(*name)})
	} else {
		src.cursor = cursor
	}

	//anything
	if readExact(src, SeqUnderscore) {
		return finishParsePattern(src, parsed.PAny{Location: loc(src, cursor)})
	}

	const_ := parseConst(src)
	if nil != const_ {
		return finishParsePattern(src, parsed.PConst{Location: loc(src, cursor), Value: const_})
	}

	return nil
}

func finishParsePattern(src *Source, pattern parsed.Pattern) parsed.Pattern {
	cursor := src.cursor

	if readExact(src, SeqColon) {
		type_ := parseType(src)
		if nil == type_ {
			setErrorSource(*src, "expected type here")
		}
		return finishParsePattern(src, pattern.WithType(type_))
	}

	if readExact(src, KwAs) {
		name := readIdentifier(src, false)
		if nil == name {
			setErrorSource(*src, "expected pattern alias name here")
		}
		return finishParsePattern(src,
			parsed.PAlias{Location: loc(src, cursor), Alias: ast.Identifier(*name), Nested: pattern})
	}

	if readExact(src, SeqBar) {
		tail := parsePattern(src)
		if nil == tail {
			setErrorSource(*src, "expected list tail pattern here")
		}

		return finishParsePattern(src, parsed.PCons{Location: loc(src, cursor), Head: pattern, Tail: pattern})
	}

	return pattern
}

func parseSignature(src *Source) ([]parsed.Pattern, parsed.Type) {
	if !readExact(src, SeqParenthesisOpen) {
		return nil, nil
	}

	var patterns []parsed.Pattern
	var ret parsed.Type

	for {
		pattern := parsePattern(src)
		if nil == pattern {
			setErrorSource(*src, "expected pattern here")
		}
		patterns = append(patterns, pattern)

		if readExact(src, SeqComma) {
			continue
		}
		if readExact(src, SeqParenthesisClose) {
			break
		}
		setErrorSource(*src, "expected `,` or `)` here")
	}
	if readExact(src, SeqColon) {
		ret = parseType(src)
		if nil == ret {
			setErrorSource(*src, "expected return type here")
		}
	}

	return patterns, ret
}

func parseExpression(src *Source) parsed.Expression {
	cursor := src.cursor

	//const
	const_ := parseConst(src)
	if nil != const_ {
		return finishParseExpression(src, parsed.Const{Location: loc(src, cursor), Value: const_})
	}

	//list
	if readExact(src, SeqBracketsOpen) {
		var items []parsed.Expression
		if !readExact(src, SeqBracketsClose) {
			for {
				item := parseExpression(src)
				if nil == item {
					setErrorSource(*src, "expected list item expression here")
				}
				items = append(items, item)

				if readExact(src, SeqComma) {
					continue
				}
				if readExact(src, SeqBracketsClose) {
					break
				}
				setErrorSource(*src, "expected `,` or `]` here")
			}
		}
		return finishParseExpression(src, parsed.List{Location: loc(src, cursor), Items: items})
	}

	//infix value
	infix := parseInfixIdentifier(src, true)
	if nil != infix {
		return finishParseExpression(src, parsed.InfixVar{Location: loc(src, cursor), Infix: *infix})
	}

	//negate
	if readExact(src, SeqMinus) {
		nested := parseExpression(src)
		if nil == nested {
			setErrorSource(*src, "expected expression here")
		}

		return finishParseExpression(src, parsed.Negate{Location: loc(src, cursor), Nested: nested})
	}

	//lambda
	if readExact(src, SeqLambda) {
		src.cursor = cursor + 1

		patterns, ret := parseSignature(src)
		if nil == patterns {
			setErrorSource(*src, "expected lambda signature here")
		}

		if !readExact(src, SeqLambdaBind) {
			setErrorSource(*src, "expected `->` here")
		}

		body := parseExpression(src)
		if nil == body {
			setErrorSource(*src, "expected lambda expression body here")
		}
		return finishParseExpression(src,
			parsed.Lambda{Location: loc(src, cursor), Params: patterns, Body: body, Return: ret})
	}

	//if
	if readExact(src, KwIf) {
		condition := parseExpression(src)
		if nil == condition {
			setErrorSource(*src, "expected condition expression here")
		}
		if !readExact(src, KwThen) {
			setErrorSource(*src, "expected `then` here")
		}
		positive := parseExpression(src)
		if nil == positive {
			setErrorSource(*src, "expected positive branch expression here")
		}
		if !readExact(src, KwElse) {
			setErrorSource(*src, "expected `else` here")
		}
		negative := parseExpression(src)
		if nil == negative {
			setErrorSource(*src, "expected negative branch expression here")
		}
		return finishParseExpression(src,
			parsed.If{Location: loc(src, cursor), Condition: condition, Positive: positive, Negative: negative})
	}

	//let
	if readExact(src, KwLet) {
		defCursor := src.cursor
		name := readIdentifier(src, false)
		typeCursor := src.cursor
		patterns, ret := parseSignature(src)
		var def parsed.Definition
		if nil != patterns {
			if !readExact(src, SeqEqual) {
				setErrorSource(*src, "expected `=` here")
			}
			body := parseExpression(src)
			if nil == body {
				setErrorSource(*src, "expected function body here")
			}
			def = parsed.Definition{
				Pattern: parsed.PNamed{
					Location: loc(src, defCursor),
					Name:     ast.Identifier(*name),
				},
				Expression: parsed.Lambda{
					Params: patterns,
					Return: ret,
					Body:   body,
				},
				Type: parsed.TFunc{
					Location: loc(src, typeCursor),
					Params:   common.Map(func(x parsed.Pattern) parsed.Type { return x.GetType() }, patterns),
					Return:   ret,
				},
			}
		} else {
			src.cursor = defCursor
			pattern := parsePattern(src)
			if nil == pattern {
				setErrorSource(*src, "expected pattern here")
			}
			if !readExact(src, SeqEqual) {
				setErrorSource(*src, "expected `=` here")
			}
			expr := parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected expression here")
			}
			def = parsed.Definition{
				Pattern:    pattern,
				Expression: expr,
			}
		}

		preLet := src.cursor
		if readExact(src, KwLet) {
			src.cursor = preLet
		} else if !readExact(src, KwIn) {
			setErrorSource(*src, "expected `let` or `in` here")
		}

		body := parseExpression(src)
		if nil == body {
			setErrorSource(*src, "expected expression here")
		}
		return finishParseExpression(src, parsed.Let{Location: loc(src, cursor), Definition: def, Body: body})
	}

	//select
	if readExact(src, KwSelect) {
		condition := parseExpression(src)
		if nil == condition {
			setErrorSource(*src, "expected select condition expression here")
		}

		var cases []parsed.SelectCase

		for {
			caseCursor := src.cursor
			if !readExact(src, KwCase) {
				break
			}

			pattern := parsePattern(src)
			if nil == pattern {
				setErrorSource(*src, "expected pattern here")
			}

			if !readExact(src, SeqCaseBind) {
				setErrorSource(*src, "expected `->` here")
			}

			expr := parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected case expression here here")
			}
			cases = append(cases, parsed.SelectCase{Location: loc(src, caseCursor), Pattern: pattern, Expression: expr})
		}

		if 0 == len(cases) {
			setErrorSource(*src, "expected case expression here here")
		}
		return finishParseExpression(src, parsed.Select{Location: loc(src, cursor), Condition: condition, Cases: cases})
	}

	//accessor
	if readExact(src, SeqDot) {
		name := readIdentifier(src, false) //TODO: make nested access
		if nil == name {
			setErrorSource(*src, "expected accessor name here")
		}
		return parsed.Accessor{Location: loc(src, cursor), FieldName: ast.Identifier(*name)}
	}

	//record / update
	if readExact(src, SeqBracesOpen) {
		recCursor := src.cursor

		name := readIdentifier(src, true)
		if nil != name && !readExact(src, SeqBar) {
			src.cursor = recCursor
		}

		var fields []parsed.RecordField
		for {
			fieldCursor := src.cursor

			fieldName := readIdentifier(src, true)
			if nil == fieldName {
				setErrorSource(*src, "expected field name here")
			}
			if !readExact(src, SeqEqual) {
				setErrorSource(*src, "expected `=` here")
			}
			expr := parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected record field value expression here")
			}
			fields = append(fields, parsed.RecordField{
				Location: loc(src, fieldCursor),
				Name:     ast.Identifier(*fieldName),
				Value:    expr,
			})

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `}` here")
		}

		if nil == name {
			return finishParseExpression(src, parsed.Record{Location: loc(src, cursor), Fields: fields})
		} else {
			return finishParseExpression(src,
				parsed.Update{Location: loc(src, cursor), RecordName: *name, Fields: fields})
		}
	}

	//tuple / void / precedence
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return finishParseExpression(src, parsed.Const{Location: loc(src, cursor), Value: ast.CUnit{}})
		}

		var items []parsed.Expression
		for {
			expr := parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected expression here")
			}
			items = append(items, expr)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `)` here")
		}

		if 1 == len(items) {
			return finishParseExpression(src, items[0])
		} else {
			return finishParseExpression(src, parsed.Tuple{Location: loc(src, cursor), Items: items})
		}
	}

	name := readIdentifier(src, true)
	if nil != name {
		return finishParseExpression(src, parsed.Var{Location: loc(src, cursor), Name: *name})
	}

	return nil
}

func finishParseExpression(src *Source, expr parsed.Expression) parsed.Expression {
	cursor := src.cursor

	infixOp := parseInfixIdentifier(src, false)
	if nil != infixOp {
		final := parseExpression(src)
		if nil == final {
			setErrorSource(*src, "expected second operand expression of binary expression here")
		}

		return parsed.BinOp{Location: loc(src, cursor), Infix: *infixOp, Left: expr, Right: final}
	}

	if readExact(src, SeqParenthesisOpen) {
		var items []parsed.Expression
		for {
			item := parseExpression(src)
			if nil == item {
				setErrorSource(*src, "expected function argument expression here")
			}
			items = append(items, item)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			setErrorSource(*src, "expected `,` or `)` here")
		}
		return finishParseExpression(src, parsed.Call{Location: loc(src, cursor), Func: expr, Args: items})
	}

	if readExact(src, SeqDot) {
		name := readIdentifier(src, false)
		if nil == name {
			setErrorSource(*src, "expected field name here")
		}
		return finishParseExpression(src, parsed.Access{
			Location:  loc(src, cursor),
			Record:    expr,
			FieldName: ast.Identifier(*name)})
	}
	return expr
}

func parseDataValue(src *Source) parsed.DataTypeValue {
	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	var types []parsed.Type

	name := readIdentifier(src, false)

	if nil == name {
		setErrorSource(*src, "expected option name here")
	}
	if readExact(src, SeqParenthesisOpen) {
		for {
			type_ := parseType(src)
			if nil == type_ {
				setErrorSource(*src, "expected option value type here")
			}
			types = append(types, type_)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
		}
	}

	return parsed.DataTypeValue{
		Location: loc(src, cursor),
		Name:     ast.Identifier(*name),
		Params:   types,
		Hidden:   hidden,
	}
}

func parseImport(src *Source) *parsed.Import {
	if !readExact(src, KwImport) {
		return nil
	}

	cursor := src.cursor
	exposingAll := false
	var alias *ast.QualifiedIdentifier
	var exposing []string
	path := parseString(src)

	if nil == path {
		setErrorSource(*src, "expected module path string here")
	}

	if readExact(src, KwAs) {
		alias = readIdentifier(src, false)
		if nil == alias {
			setErrorSource(*src, "expected alias name here")
		}
	}

	if readExact(src, KwExposing) {
		exposingAll = readExact(src, SeqExposingAll)
		if !exposingAll {
			if !readExact(src, SeqParenthesisOpen) {
				setErrorSource(*src, "expected `(`")
			}

			for {
				id := readIdentifier(src, false)
				if nil == id {
					inf := parseInfixIdentifier(src, true)
					if nil == inf {
						setErrorSource(*src, "expected definition/infix name here")
					} else {
						exposing = append(exposing, string(*inf))
					}

				} else {
					exposing = append(exposing, string(*id))
				}

				if readExact(src, SeqComma) {
					continue
				}
				if readExact(src, SeqParenthesisClose) {
					break
				}
				setErrorSource(*src, "expected `,` or `)`")
			}
		}
	}
	return &parsed.Import{
		Location:    loc(src, cursor),
		Path:        *path,
		Alias:       (*ast.Identifier)(alias),
		ExposingAll: exposingAll,
		Exposing:    exposing,
	}
}

func parseInfixFn(src *Source) *parsed.Infix {
	if !readExact(src, KwInfix) {
		return nil
	}
	cursor := src.cursor
	hidden := readExact(src, KwHidden)

	name := parseInfixIdentifier(src, true)
	if nil == name {
		setErrorSource(*src, "expected infix statement name here")
	}
	if !readExact(src, SeqColon) {
		setErrorSource(*src, "expected `:` here")
	}
	if !readExact(src, SeqParenthesisOpen) {
		setErrorSource(*src, "expected `(` here")
	}
	var assoc parsed.Associativity
	if readExact(src, KwLeft) {
		assoc = parsed.Left
	} else if readExact(src, KwRight) {
		assoc = parsed.Right
	} else if readExact(src, KwNon) {
		assoc = parsed.None
	} else {
		setErrorSource(*src, "expected `left`, `right` or `non` here")
	}

	precedence := parseInt(src)
	if precedence == nil {
		setErrorSource(*src, "expected precedence (integer number) here")
	}

	if !readExact(src, SeqParenthesisClose) {
		setErrorSource(*src, "expected `)` here")
	}
	if !readExact(src, SeqEqual) {
		setErrorSource(*src, "expected `=` here")
	}

	aliasCursor := src.cursor
	alias := readIdentifier(src, false)
	if nil == alias {
		setErrorSource(*src, "expected definition name here")
	}
	return &parsed.Infix{
		Location:      loc(src, cursor),
		Hidden:        hidden,
		Name:          *name,
		Associativity: assoc,
		Precedence:    int(*precedence),
		AliasLocation: loc(src, aliasCursor),
		Alias:         ast.Identifier(*alias),
	}
}

func parseAlias(src *Source) *parsed.Alias {
	if !readExact(src, KwAlias) {
		return nil
	}

	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	extern_ := readExact(src, KwExtern)
	var type_ parsed.Type
	name := readIdentifier(src, false)

	if nil == name {
		setErrorSource(*src, "expected alias name here")
	}
	typeParams := parseTypeParamNames(src)

	if !extern_ {
		if !readExact(src, SeqEqual) {
			setErrorSource(*src, "expected `=` here")
		}
		type_ = parseType(src)
		if nil == type_ {
			setErrorSource(*src, "expected definedReturn declaration here")
		}
	}

	return &parsed.Alias{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     ast.Identifier(*name),
		Params:   typeParams,
		Type:     type_,
	}
}

func parseDataType(src *Source) *parsed.DataType {
	if !readExact(src, KwData) {
		return nil
	}

	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	name := readIdentifier(src, false)
	if nil == name {
		setErrorSource(*src, "expected data name here")
	}
	typeParams := parseTypeParamNames(src)

	if !readExact(src, SeqEqual) {
		setErrorSource(*src, "expected `=` here")
	}

	var values []parsed.DataTypeValue
	for {
		value := parseDataValue(src)
		values = append(values, value)
		if !readExact(src, SeqBar) {
			break
		}
	}
	return &parsed.DataType{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     ast.Identifier(*name),
		Params:   typeParams,
		Values:   values,
	}
}

func parseDefinition(src *Source) *parsed.Definition {
	cursor := src.cursor

	if !readExact(src, KwDef) {
		return nil
	}
	hidden := readExact(src, KwHidden)
	extern := readExact(src, KwExtern)
	nameCursor := src.cursor
	name := readIdentifier(src, false)
	var type_ parsed.Type
	var expr parsed.Expression

	if nil == name {
		setErrorSource(*src, "expected data name here")
	}

	typeCursor := src.cursor
	params, ret := parseSignature(src)
	if nil == params {
		if readExact(src, SeqColon) {
			type_ = parseType(src)
			if nil == type_ {
				setErrorSource(*src, "expected definedReturn here")
			}
		}
		if !extern {
			if !readExact(src, SeqEqual) {
				setErrorSource(*src, "expected `=` here")
			}
			expr = parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected expression here")
			}
		}
	} else {
		if !extern {
			if !readExact(src, SeqEqual) {
				setErrorSource(*src, "expected `=` here")
			}
			exprCursor := src.cursor
			expr = parseExpression(src)
			if nil == expr {
				setErrorSource(*src, "expected expression here")
			}
			expr = parsed.Lambda{
				Location: loc(src, exprCursor),
				Params:   params,
				Return:   ret,
				Body:     expr,
			}
		}
		type_ = parsed.TFunc{
			Location: loc(src, typeCursor),
			Params:   common.Map(func(x parsed.Pattern) parsed.Type { return x.GetType() }, params),
			Return:   ret,
		}
	}

	return &parsed.Definition{
		Location:   loc(src, cursor),
		Hidden:     hidden,
		Pattern:    parsed.PNamed{Location: loc(src, nameCursor), Name: ast.Identifier(*name)},
		Expression: expr,
		Type:       type_,
	}
}

func parseModule(src *Source) *parsed.Module {
	skipComment(src)

	if !readExact(src, KwModule) {
		return nil
	}

	name := readIdentifier(src, true)

	if nil == name {
		setErrorSource(*src, "expected module name here")
	}
	m := parsed.Module{
		Path: src.filePath,
		Name: *name,
	}

	for {
		imp := parseImport(src)
		if imp == nil {
			break
		}
		m.Imports = append(m.Imports, *imp)
	}

	for {

		if alias := parseAlias(src); alias != nil {
			m.Aliases = append(m.Aliases, *alias)
			continue
		}

		if infixFn := parseInfixFn(src); infixFn != nil {
			m.InfixFns = append(m.InfixFns, *infixFn)
			continue
		}

		if definition := parseDefinition(src); definition != nil {
			m.Definitions = append(m.Definitions, *definition)
			continue
		}

		if dataType := parseDataType(src); dataType != nil {
			m.DataTypes = append(m.DataTypes, *dataType)
			continue
		}

		if isOk(src) {
			setErrorSource(*src, "failed to parse statement")
		}
		break
	}

	return &m
}

func Parse(filePath string, modules map[string]parsed.Module) string {
	var added []string
	var err error
	filePath, err = filepath.Abs(filePath)
	if err != nil {
		panic(common.SystemError{Message: err.Error()})
	}

	added = append(added, filePath)

	for i := 0; i < len(added); i++ {
		absPath, err := filepath.Abs(added[i])
		if _, exists := modules[absPath]; exists {
			continue
		}

		if err != nil {
			panic(common.SystemError{Message: err.Error()})
		}
		data, err := os.ReadFile(absPath)
		if err != nil {
			panic(common.SystemError{Message: err.Error()})
		}
		src := &Source{
			filePath: absPath,
			text:     []rune(string(data)),
		}
		m := parseModule(src)
		if m == nil {
			continue
		}

		for j, imp := range m.Imports {
			p := imp.Path
			if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") {
				p = filepath.Clean(filepath.Join(filepath.Dir(m.Path), p))
			}
			p, err = filepath.Abs(p)
			if err != nil {
				panic(common.SystemError{Message: err.Error()})
			}
			imp.Path = p
			m.Imports[j] = imp
			added = append(added, p)
		}

		modules[absPath] = *m
	}

	return filePath
}