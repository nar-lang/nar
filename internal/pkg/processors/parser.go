package processors

import (
	"fmt"
	"nar-compiler/internal/pkg/ast"
	"nar-compiler/internal/pkg/ast/parsed"
	"nar-compiler/internal/pkg/common"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

func ParseWithContent(filePath string, fileContent string) (*parsed.Module, []error) {
	src := &source{
		filePath: filePath,
		text:     []rune(fileContent),
	}
	return parseModule(src)
}

func Parse(filePath string) (*parsed.Module, []error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, []error{common.NewSystemError(fmt.Errorf("failed to read module `%s`: %w", filePath, err))}
	}
	return ParseWithContent(filePath, string(data))
}

const (
	KwModule   = "module"
	KwImport   = "import"
	KwAs       = "as"
	KwExposing = "exposing"
	KwInfix    = "infix"
	KwAlias    = "alias"
	KwType     = "type"
	KwDef      = "def"
	KwHidden   = "hidden"
	KwNative   = "native"
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
	KwEnd      = "end"

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

type source struct {
	filePath string
	cursor   uint32
	text     []rune
	log      *common.LogWriter
}

func loc(src *source, cursor uint32) ast.Location {
	return ast.Location{FilePath: src.filePath, FileContent: src.text, Position: cursor}
}

func newError(src source, msg string) error {
	return common.Error{
		Location: ast.Location{
			FilePath:    src.filePath,
			FileContent: src.text,
			Position:    src.cursor,
		},
		Message: msg,
	}
}

func isOk(src *source) bool {
	return src.cursor < uint32(len(src.text))
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

func readSequence(src *source, value string) *string {
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

func skipWhiteSpace(src *source) {
	for isOk(src) && unicode.IsSpace(src.text[src.cursor]) {
		src.cursor++
	}
}

func skipComment(src *source) {
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

func readIdentifier(src *source, qualified bool) *ast.QualifiedIdentifier {
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

func parseInt(src *source) (*int64, error) {
	if !isOk(src) {
		return nil, nil
	}

	pos := src.cursor

	strValue, base := readIntegerPart(src, true)

	if strValue == "" {
		src.cursor = pos
		return nil, nil
	}

	value, err := strconv.ParseInt(strValue, base, 64)
	if err != nil {
		return nil, newError(*src, "failed to parse integer: "+err.Error())
	}

	skipComment(src)
	return &value, nil
}

func parseFloat(src *source) (*float64, error) {
	if !isOk(src) {
		return nil, nil
	}
	pos := src.cursor

	first, _ := readIntegerPart(src, false)
	if first == "" {
		return nil, nil
	}

	if readSequence(src, ".") != nil {
		second, base := readIntegerPart(src, false)
		if base == 0 {
			return nil, nil
		}
		first += "." + second
	}
	if readSequence(src, "e") != nil || readSequence(src, "E") != nil {
		var sign string
		if readSequence(src, "-") != nil {
			sign = "-"
		} else if readSequence(src, "+") != nil {
			sign = "+"
		}
		second, base := readIntegerPart(src, false)
		if base == 0 {
			return nil, nil
		}
		first += "e" + sign + second
	}

	if isOk(src) && (unicode.IsLetter(src.text[src.cursor]) || unicode.IsNumber(src.text[src.cursor])) {
		src.cursor = pos
		return nil, nil
	}
	skipComment(src)

	value, err := strconv.ParseFloat(first, 64)
	if err != nil {
		return nil, newError(*src, "failed to parse float: "+err.Error())
	}
	return &value, nil
}

var kNumBin = []rune{'0', '1'}
var kNumOct = []rune{'0', '1', '2', '3', '4', '5', '6', '7'}
var kNumDec = []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
var kNumHex = []rune{
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'a', 'b', 'c', 'd', 'e', 'f', 'A', 'B', 'C', 'D', 'E', 'F'}

func readIntegerPart(src *source, allowBases bool) (string, int) {
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

func readExact(src *source, value string) bool {
	if nil != readSequence(src, value) {
		skipComment(src)
		return true
	}
	return false
}

func parseChar(src *source) (*rune, error) {
	if !isOk(src) {
		return nil, nil
	}

	if SmbQuoteChar != src.text[src.cursor] {
		return nil, nil
	}
	src.cursor++
	if !isOk(src) {
		return nil, newError(*src, "character is not closed before end of file")
	}
	escaped := src.text[src.cursor] == SmbEscape

	var r rune

	if escaped {
		src.cursor++
		r = unicode.ToLower(src.text[src.cursor])

		switch r {
		case '0':
			r = '\u0000'
			break
		case 'a':
			r = '\a'
			break
		case 'b':
			r = '\b'
			break
		case 'f':
			r = '\f'
			break
		case 'n':
			r = '\n'
			break
		case 'r':
			r = '\r'
			break
		case 't':
			r = '\t'
			break
		case 'v':
			r = '\v'
			break
		case SmbQuoteChar:
			r = SmbQuoteChar
			break
		case SmbEscape:
			r = SmbEscape
			break
		case 'u':
			src.cursor++
			if !isOk(src) {
				return nil, newError(*src, "expected unicode character here")
			}
			var value []rune
			for i := 0; i < 4; i++ {
				if !isOk(src) || !unicode.IsDigit(src.text[src.cursor]) {
					return nil, newError(*src, "expected unicode character here")
				}
				value = append(value, src.text[src.cursor])
				src.cursor++
			}
			valueStr := string(value)
			valueInt, err := strconv.ParseInt(valueStr, 16, 32)
			if err != nil {
				return nil, newError(*src, "failed to parse unicode character: "+err.Error())
			}
			r = rune(valueInt)
			break
		default:
			return nil, newError(*src, "unknown escape sequence")
		}
	} else {
		r = src.text[src.cursor]
	}
	src.cursor++
	if !isOk(src) || SmbQuoteChar != src.text[src.cursor] {
		return nil, newError(*src, "expected "+string(SmbQuoteChar)+"here")
	}
	src.cursor++

	skipComment(src)
	return &r, nil
}

var controlCharsReplacer = strings.NewReplacer(
	"\\0", "\u0000",
	"\\a", "\a",
	"\\b", "\b",
	"\\f", "\f",
	"\\n", "\n",
	"\\r", "\r",
	"\\t", "\t",
	"\\v", "\v",
	"\\\"", "\"",
)

func parseString(src *source) (*string, error) {
	if !isOk(src) {
		return nil, nil
	}

	start := src.cursor

	if SmbQuoteString != src.text[src.cursor] {
		return nil, nil
	}

	src.cursor++
	skipNextQuote := false
	for {
		if !isOk(src) {
			return nil, newError(*src, "string is not closed before the end of file")
		}
		if SmbQuoteString == src.text[src.cursor] && !skipNextQuote {
			break
		}
		skipNextQuote = SmbEscape == src.text[src.cursor]
		src.cursor++
	}
	src.cursor++
	str := string(src.text[start+1 : src.cursor-1])
	skipComment(src)
	str = controlCharsReplacer.Replace(str)
	return &str, nil
}

func parseNumber(src *source) (iValue *int64, fValue *float64, err error) {
	pos := src.cursor
	fv, err := parseFloat(src)
	if err != nil {
		return nil, nil, err
	}
	fvPos := src.cursor

	src.cursor = pos
	iv, err := parseInt(src)
	if err != nil {
		return nil, nil, err
	}

	if fv == nil {
		return iv, nil, nil
	}
	if iv == nil {
		src.cursor = fvPos
		return nil, fv, nil
	}

	if src.cursor != fvPos {
		src.cursor = fvPos
		return nil, fv, nil
	}

	return iv, nil, nil
}

func parseConst(src *source) (ast.ConstValue, error) {
	r, err := parseChar(src)
	if err != nil {
		return nil, err
	}
	if nil != r {
		return ast.CChar{Value: *r}, nil
	}

	s, err := parseString(src)
	if err != nil {
		return nil, err
	}
	if nil != s {
		return ast.CString{Value: *s}, nil
	}

	i, f, err := parseNumber(src)
	if err != nil {
		return nil, err
	}
	if f != nil {
		return ast.CFloat{Value: *f}, nil
	}
	if i != nil {
		return ast.CInt{Value: *i}, nil
	}

	return nil, nil
}

func parseInfixIdentifier(src *source, withParenthesis bool) *ast.InfixIdentifier {
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
		src.cursor = cursor
		return nil
	}

	if 0 == end-start {
		src.cursor = cursor
		return nil
	}
	result := ast.InfixIdentifier(src.text[start:end])

	skipComment(src)

	return &result
}

func parseTypeParamNames(src *source) ([]ast.Identifier, error) {
	if !readExact(src, SeqBracketsOpen) {
		return nil, nil
	}

	var result []ast.Identifier
	for {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected variable type name here")
		} else if !unicode.IsLower([]rune(*name)[0]) {
			return nil, newError(*src, "type parameter name should start with lowercase letter")
		} else {
			result = append(result, ast.Identifier(*name))
		}

		if readExact(src, SeqComma) {
			continue
		}
		if readExact(src, SeqBracketsClose) {
			break
		}
		return nil, newError(*src, "expected `,` or `]` here")
	}

	return result, nil
}

func parseType(src *source) (parsed.Type, error) {
	cursor := src.cursor

	//signature/tuple/unit
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return parsed.TUnit{Location: loc(src, cursor)}, nil
		}

		var items []parsed.Type

		for {

			type_, err := parseType(src)
			if err != nil {
				return nil, err
			}
			if nil == type_ {
				return nil, newError(*src, "expected type here")
			}
			items = append(items, type_)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `)` here")
		}

		if readExact(src, SeqColon) {
			ret, err := parseType(src)
			if err != nil {
				return nil, err
			}
			if nil == ret {
				return nil, newError(*src, "expected return type here")
			}
			return parsed.TFunc{Location: loc(src, cursor), Return: ret, Params: items}, nil
		} else {
			if 1 == len(items) {
				return items[0], nil
			} else {
				return parsed.TTuple{Location: loc(src, cursor), Items: items}, nil
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
				return nil, newError(*src, "expected field name here")
			}
			if !readExact(src, SeqColon) {
				return nil, newError(*src, "expected `:` here")
			}
			type_, err := parseType(src)
			if err != nil {
				return nil, err
			}
			if nil == type_ {
				return nil, newError(*src, "expected field type here")
			}

			if _, ok := fields[ast.Identifier(*name)]; ok {
				return nil, newError(*src, "field with this name has already declared for the record")
			}
			fields[ast.Identifier(*name)] = type_

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `}` here")
		}

		return parsed.TRecord{Location: loc(src, cursor), Fields: fields}, nil
	}

	if name := readIdentifier(src, true); nil != name {
		if unicode.IsLower([]rune(*name)[0]) {
			return parsed.TTypeParameter{Location: loc(src, cursor), Name: ast.Identifier(*name)}, nil
		} else {
			var typeParams []parsed.Type
			if readExact(src, SeqBracketsOpen) {
				for {
					type_, err := parseType(src)
					if err != nil {
						return nil, err
					}
					if nil == type_ {
						return nil, newError(*src, "expected type parameter here")
					}
					typeParams = append(typeParams, type_)

					if readExact(src, SeqComma) {
						continue
					}
					if readExact(src, SeqBracketsClose) {
						break
					}
					return nil, newError(*src, "expected `,` or `]`  here")
				}
			}

			return parsed.TNamed{Location: loc(src, cursor), Name: *name, Args: typeParams}, nil
		}
	}
	return nil, nil
}

func parsePattern(src *source) (parsed.Pattern, error) {
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
			item, err := parsePattern(src)
			if err != nil {
				return nil, err
			}
			if nil == item {
				return nil, newError(*src, "expected tuple item pattern here")
			}
			items = append(items, item)
			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `)` here")
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
				return nil, newError(*src, "expected record field name here")
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
			return nil, newError(*src, "expected `,` or `}` here")
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
			p, err := parsePattern(src)
			if err != nil {
				return nil, err
			}
			if nil == p {
				return nil, newError(*src, "expected list item pattern here")
			}
			items = append(items, p)
			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracketsClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `}` here")
		}

		return finishParsePattern(src, parsed.PList{Location: loc(src, cursor), Items: items})
	}

	//union
	name := readIdentifier(src, true)
	if nil != name && unicode.IsUpper([]rune(*name)[0]) {
		var items []parsed.Pattern
		if readExact(src, SeqParenthesisOpen) {
			for {
				item, err := parsePattern(src)
				if err != nil {
					return nil, err
				}
				if nil == item {
					return nil, newError(*src, "expected option value pattern here")
				}
				items = append(items, item)
				if readExact(src, SeqComma) {
					continue
				}
				if readExact(src, SeqParenthesisClose) {
					break
				}
				return nil, newError(*src, "expected `,` or `)` here")
			}
		}
		return finishParsePattern(src, parsed.PDataOption{Location: loc(src, cursor), Name: *name, Values: items})
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

	const_, err := parseConst(src)
	if err != nil {
		return nil, err
	}
	if nil != const_ {
		return finishParsePattern(src, parsed.PConst{Location: loc(src, cursor), Value: const_})
	}

	return nil, nil
}

func finishParsePattern(src *source, pattern parsed.Pattern) (parsed.Pattern, error) {
	cursor := src.cursor

	if readExact(src, SeqColon) {
		type_, err := parseType(src)
		if err != nil {
			return nil, err
		}
		if nil == type_ {
			return nil, newError(*src, "expected type here")
		}
		return finishParsePattern(src, pattern.WithType(type_))
	}

	if readExact(src, KwAs) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected pattern alias name here")
		}
		return finishParsePattern(src,
			parsed.PAlias{Location: loc(src, cursor), Alias: ast.Identifier(*name), Nested: pattern})
	}

	if readExact(src, SeqBar) {
		tail, err := parsePattern(src)
		if err != nil {
			return nil, err
		}
		if nil == tail {
			return nil, newError(*src, "expected list tail pattern here")
		}

		return finishParsePattern(src, parsed.PCons{Location: loc(src, cursor), Head: pattern, Tail: tail})
	}

	return pattern, nil
}

func parseSignature(src *source) ([]parsed.Pattern, parsed.Type, error) {
	if !readExact(src, SeqParenthesisOpen) {
		return nil, nil, nil
	}

	var patterns []parsed.Pattern
	var ret parsed.Type
	var err error

	for {
		pattern, err := parsePattern(src)
		if err != nil {
			return nil, nil, err
		}
		if nil == pattern {
			return nil, nil, newError(*src, "expected pattern here")
		}
		patterns = append(patterns, pattern)

		if readExact(src, SeqComma) {
			continue
		}
		if readExact(src, SeqParenthesisClose) {
			break
		}
		return nil, nil, newError(*src, "expected `,` or `)` here")
	}
	if readExact(src, SeqColon) {
		ret, err = parseType(src)
		if err != nil {
			return nil, nil, err
		}
		if nil == ret {
			return nil, nil, newError(*src, "expected return type here")
		}
	}

	return patterns, ret, nil
}

func parseExpression(src *source, negate bool) (parsed.Expression, error) {
	cursor := src.cursor

	//const
	const_, err := parseConst(src)
	if err != nil {
		return nil, err
	}
	if nil != const_ {
		return finishParseExpression(src, parsed.Const{Location: loc(src, cursor), Value: const_}, negate)
	}

	//list
	if readExact(src, SeqBracketsOpen) {
		var items []parsed.Expression
		if !readExact(src, SeqBracketsClose) {
			for {
				item, err := parseExpression(src, false)
				if err != nil {
					return nil, err
				}
				if nil == item {
					return nil, newError(*src, "expected list item expression here")
				}
				items = append(items, item)

				if readExact(src, SeqComma) {
					continue
				}
				if readExact(src, SeqBracketsClose) {
					break
				}
				return nil, newError(*src, "expected `,` or `]` here")
			}
		}
		return finishParseExpression(src, parsed.List{Location: loc(src, cursor), Items: items}, negate)
	}

	//negate
	if readExact(src, SeqMinus) {
		return parseExpression(src, !negate)
	}

	//infix value
	infix := parseInfixIdentifier(src, true)
	if nil != infix {
		return finishParseExpression(src, parsed.InfixVar{Location: loc(src, cursor), Infix: *infix}, negate)
	}

	//lambda
	if readExact(src, SeqLambda) {
		src.cursor = cursor + 1

		patterns, ret, err := parseSignature(src)
		if err != nil {
			return nil, err
		}
		if nil == patterns {
			return nil, newError(*src, "expected lambda signature here")
		}

		if !readExact(src, SeqLambdaBind) {
			return nil, newError(*src, "expected `->` here")
		}

		body, err := parseExpression(src, false)
		if err != nil {
			return nil, err
		}
		if nil == body {
			return nil, newError(*src, "expected lambda expression body here")
		}
		return finishParseExpression(src,
			parsed.Lambda{Location: loc(src, cursor), Params: patterns, Body: body, Return: ret}, negate)
	}

	//if
	if readExact(src, KwIf) {
		condition, err := parseExpression(src, false)
		if err != nil {
			return nil, err
		}
		if nil == condition {
			return nil, newError(*src, "expected condition expression here")
		}
		if !readExact(src, KwThen) {
			return nil, newError(*src, "expected `then` here")
		}
		positive, err := parseExpression(src, false)
		if nil == positive {
			return nil, newError(*src, "expected positive branch expression here")
		}
		if !readExact(src, KwElse) {
			return nil, newError(*src, "expected `else` here")
		}
		negative, err := parseExpression(src, false)
		if nil == negative {
			return nil, newError(*src, "expected negative branch expression here")
		}
		return finishParseExpression(src,
			parsed.If{Location: loc(src, cursor), Condition: condition, Positive: positive, Negative: negative},
			negate)
	}

	//let
	if readExact(src, KwLet) {
		defCursor := src.cursor
		name := readIdentifier(src, false)
		typeCursor := src.cursor
		params, ret, err := parseSignature(src)
		if err != nil {
			return nil, err
		}

		var pattern parsed.Pattern
		var value parsed.Expression
		var fnType parsed.Type
		isDef := nil != name && nil != params && len(*name) > 0 && unicode.IsLower([]rune(*name)[0])
		if isDef {
			if !readExact(src, SeqEqual) {
				return nil, newError(*src, "expected `=` here")
			}
			value, err = parseExpression(src, false)
			if err != nil {
				return nil, err
			}
			if nil == value {
				return nil, newError(*src, "expected function body here")
			}
			pattern = parsed.PNamed{
				Location: loc(src, defCursor),
				Name:     ast.Identifier(*name),
			}
			fnType = parsed.TFunc{
				Location: loc(src, typeCursor),
				Params:   common.Map(func(x parsed.Pattern) parsed.Type { return x.GetType() }, params),
				Return:   ret,
			}
		} else {
			src.cursor = defCursor
			pattern, err = parsePattern(src)
			if err != nil {
				return nil, err
			}
			if nil == pattern {
				return nil, newError(*src, "expected pattern here")
			}
			if !readExact(src, SeqEqual) {
				return nil, newError(*src, "expected `=` here")
			}
			value, err = parseExpression(src, false)
			if err != nil {
				return nil, err
			}
			if nil == value {
				return nil, newError(*src, "expected expression here")
			}
		}

		preLet := src.cursor
		if readExact(src, KwLet) {
			src.cursor = preLet
		} else if !readExact(src, KwIn) {
			return nil, newError(*src, "expected `let` or `in` here")
		}

		nested, err := parseExpression(src, false)
		if nil == nested {
			return nil, newError(*src, "expected expression here")
		}
		if isDef {
			return finishParseExpression(src, parsed.LetDef{
				Location: loc(src, cursor),
				Name:     ast.Identifier(*name),
				Params:   params,
				Body:     value,
				FnType:   fnType,
				Nested:   nested,
			}, negate)
		} else {
			return finishParseExpression(src, parsed.LetMatch{
				Location: loc(src, cursor),
				Pattern:  pattern,
				Value:    value,
				Nested:   nested,
			}, negate)
		}
	}

	//select
	if readExact(src, KwSelect) {
		condition, err := parseExpression(src, false)
		if err != nil {
			return nil, err
		}
		if nil == condition {
			return nil, newError(*src, "expected select condition expression here")
		}

		var cases []parsed.SelectCase

		for {
			caseCursor := src.cursor
			if !readExact(src, KwCase) {
				if !readExact(src, KwEnd) {
					return nil, newError(*src, "expected `case` or `end` here")
				}
				break
			}

			pattern, err := parsePattern(src)
			if err != nil {
				return nil, err
			}
			if nil == pattern {
				return nil, newError(*src, "expected pattern here")
			}

			if !readExact(src, SeqCaseBind) {
				return nil, newError(*src, "expected `->` here")
			}

			expr, err := parseExpression(src, false)
			if nil == expr {
				return nil, newError(*src, "expected case expression here")
			}
			cases = append(cases, parsed.SelectCase{Location: loc(src, caseCursor), Pattern: pattern, Expression: expr})
		}

		if 0 == len(cases) {
			return nil, newError(*src, "expected case expression here")
		}
		return finishParseExpression(src, parsed.Select{Location: loc(src, cursor), Condition: condition, Cases: cases}, negate)
	}

	//accessor
	if readExact(src, SeqDot) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected accessor name here")
		}
		return finishParseExpression(src, parsed.Accessor{Location: loc(src, cursor), FieldName: ast.Identifier(*name)}, negate)
	}

	//record / update
	if readExact(src, SeqBracesOpen) {
		if readExact(src, SeqBracesClose) {
			return finishParseExpression(src, parsed.Record{Location: loc(src, cursor)}, negate)
		}

		recCursor := src.cursor

		name := readIdentifier(src, true)
		if nil != name && !readExact(src, SeqBar) {
			src.cursor = recCursor
			name = nil
		}

		var fields []parsed.RecordField
		for {
			fieldCursor := src.cursor

			fieldName := readIdentifier(src, true)
			if nil == fieldName {
				return nil, newError(*src, "expected field name here")
			}
			if !readExact(src, SeqEqual) {
				return nil, newError(*src, "expected `=` here")
			}
			expr, err := parseExpression(src, false)
			if err != nil {
				return nil, err
			}

			if nil == expr {
				return nil, newError(*src, "expected record field value expression here")
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
			return nil, newError(*src, "expected `,` or `}` here")
		}

		if nil == name {
			return finishParseExpression(src, parsed.Record{Location: loc(src, cursor), Fields: fields}, negate)
		} else {
			return finishParseExpression(src,
				parsed.Update{Location: loc(src, cursor), RecordName: *name, Fields: fields}, negate)
		}
	}

	//tuple / void / precedence
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return finishParseExpression(src, parsed.Const{Location: loc(src, cursor), Value: ast.CUnit{}}, negate)
		}

		var items []parsed.Expression
		for {
			expr, err := parseExpression(src, false)
			if err != nil {
				return nil, err
			}
			if nil == expr {
				return nil, newError(*src, "expected expression here")
			}
			items = append(items, expr)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `)` here")
		}

		if 1 == len(items) {
			expr := items[0]
			if bop, ok := expr.(parsed.BinOp); ok {
				bop.InParentheses = true
				expr = bop
			}
			return finishParseExpression(src, expr, negate)
		} else {
			return finishParseExpression(src, parsed.Tuple{Location: loc(src, cursor), Items: items}, negate)
		}
	}

	name := readIdentifier(src, true)
	if nil != name {
		return finishParseExpression(src, parsed.Var{Location: loc(src, cursor), Name: *name}, negate)
	}

	return nil, nil
}

func finishParseExpression(src *source, expr parsed.Expression, negate bool) (parsed.Expression, error) {
	cursor := src.cursor

	infixOp := parseInfixIdentifier(src, false)
	if nil != infixOp {
		final, err := parseExpression(src, false)
		if err != nil {
			return nil, err
		}
		if nil == final {
			return nil, newError(*src, "expected second operand expression of binary expression here")
		}

		if negate {
			expr = parsed.Negate{Location: loc(src, cursor), Nested: expr}
		}

		items := []parsed.BinOpItem{{Expression: expr}, {Infix: *infixOp}}

		if bop, ok := final.(parsed.BinOp); ok && !bop.InParentheses {
			items = append(items, bop.Items...)
		} else {
			items = append(items, parsed.BinOpItem{Expression: final})
		}

		return parsed.BinOp{Location: loc(src, cursor), Items: items}, nil
	}

	if readExact(src, SeqParenthesisOpen) {
		var items []parsed.Expression
		for {
			item, err := parseExpression(src, false)
			if err != nil {
				return nil, err
			}
			if nil == item {
				return nil, newError(*src, "expected function argument expression here")
			}
			items = append(items, item)

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `)` here")
		}
		return finishParseExpression(src, parsed.Apply{Location: loc(src, cursor), Func: expr, Args: items}, negate)
	}

	if readExact(src, SeqDot) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected field name here")
		}
		return finishParseExpression(src, parsed.Access{
			Location:  loc(src, cursor),
			Record:    expr,
			FieldName: ast.Identifier(*name),
		}, negate)
	}
	if negate {
		expr = parsed.Negate{Location: loc(src, cursor), Nested: expr}
	}
	return expr, nil
}

func parseDataOption(src *source) (*parsed.DataTypeOption, error) {
	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	var types []parsed.Type

	name := readIdentifier(src, false)

	if nil == name {
		return nil, newError(*src, "expected option name here")
	}
	if readExact(src, SeqParenthesisOpen) {
		for {
			argCursor := src.cursor
			if readIdentifier(src, false) == nil || !readExact(src, SeqColon) {
				src.cursor = argCursor
			}

			type_, err := parseType(src)
			if err != nil {
				return nil, err
			}
			if nil == type_ {
				return nil, newError(*src, "expected option value type here")
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

	return &parsed.DataTypeOption{
		Location: loc(src, cursor),
		Name:     ast.Identifier(*name),
		Values:   types,
		Hidden:   hidden,
	}, nil
}

func parseImport(src *source) (*parsed.Import, error) {
	if !readExact(src, KwImport) {
		return nil, nil
	}

	cursor := src.cursor
	exposingAll := false
	var alias *ast.QualifiedIdentifier
	var exposing []string
	ident := readIdentifier(src, true)

	if nil == ident {
		return nil, newError(*src, "expected module path string here")
	}

	if readExact(src, KwAs) {
		alias = readIdentifier(src, false)
		if nil == alias {
			return nil, newError(*src, "expected alias name here")
		}
	}

	if readExact(src, KwExposing) {
		exposingAll = readExact(src, SeqExposingAll)
		if !exposingAll {
			if !readExact(src, SeqParenthesisOpen) {
				return nil, newError(*src, "expected `(`")
			}

			for {
				id := readIdentifier(src, false)
				if nil == id {
					inf := parseInfixIdentifier(src, true)
					if nil == inf {
						return nil, newError(*src, "expected definition/infix name here")
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
				return nil, newError(*src, "expected `,` or `)`")
			}
		}
	}
	return &parsed.Import{
		Location:         loc(src, cursor),
		ModuleIdentifier: *ident,
		Alias:            (*ast.Identifier)(alias),
		ExposingAll:      exposingAll,
		Exposing:         exposing,
	}, nil
}

func parseInfixFn(src *source) (infix *parsed.Infix, err error) {
	if !readExact(src, KwInfix) {
		return
	}
	cursor := src.cursor
	hidden := readExact(src, KwHidden)

	name := parseInfixIdentifier(src, true)
	if nil == name {
		err = newError(*src, "expected infix statement name here")
		return
	}

	infix = &parsed.Infix{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     *name,
	}

	if !readExact(src, SeqColon) {
		err = newError(*src, "expected `:` here")
		return
	}
	if !readExact(src, SeqParenthesisOpen) {
		err = newError(*src, "expected `(` here")
		return
	}

	if readExact(src, KwLeft) {
		infix.Associativity = parsed.Left
	} else if readExact(src, KwRight) {
		infix.Associativity = parsed.Right
	} else if readExact(src, KwNon) {
		infix.Associativity = parsed.None
	} else {
		err = newError(*src, "expected `left`, `right` or `non` here")
		return
	}

	var precedence *int64
	precedence, err = parseInt(src)
	if err != nil {
		return
	}
	if precedence == nil {
		err = newError(*src, "expected precedence (integer number) here")
		return
	}
	infix.Precedence = int(*precedence)

	if !readExact(src, SeqParenthesisClose) {
		err = newError(*src, "expected `)` here")
		return
	}
	if !readExact(src, SeqEqual) {
		err = newError(*src, "expected `=` here")
		return
	}

	aliasCursor := src.cursor
	alias := readIdentifier(src, false)
	if nil == alias {
		err = newError(*src, "expected definition name here")
		return
	}
	infix.Alias = ast.Identifier(*alias)
	infix.AliasLocation = loc(src, aliasCursor)
	return
}

func parseAlias(src *source) (alias *parsed.Alias, err error) {
	if !readExact(src, KwAlias) {
		return
	}

	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	native := readExact(src, KwNative)
	name := readIdentifier(src, false)

	if nil == name {
		err = newError(*src, "expected alias name here")
	}

	alias = &parsed.Alias{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     ast.Identifier(*name),
	}

	alias.Params, err = parseTypeParamNames(src)
	if err != nil {
		return
	}

	if !native {
		if !readExact(src, SeqEqual) {
			err = newError(*src, "expected `=` here")
			return
		}
		alias.Type, err = parseType(src)
		if err != nil {
			return
		}
		if nil == alias.Type {
			err = newError(*src, "expected definedReturn declaration here")
			return
		}
	}

	return
}

func parseDataType(src *source) (dataType *parsed.DataType, err error) {
	if !readExact(src, KwType) {
		return nil, nil
	}

	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	name := readIdentifier(src, false)
	if nil == name {
		err = newError(*src, "expected data name here")
		return
	}

	dataType = &parsed.DataType{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     ast.Identifier(*name),
	}

	dataType.Params, err = parseTypeParamNames(src)
	if err != nil {
		return
	}

	if !readExact(src, SeqEqual) {
		err = newError(*src, "expected `=` here")
		return
	}

	for {
		var option *parsed.DataTypeOption
		option, err = parseDataOption(src)
		if err != nil {
			return
		}
		dataType.Options = append(dataType.Options, *option)
		if !readExact(src, SeqBar) {
			break
		}
	}
	return
}

func parseDefinition(src *source, modName ast.QualifiedIdentifier) (def *parsed.Definition, err error) {
	cursor := src.cursor

	if !readExact(src, KwDef) {
		return
	}
	hidden := readExact(src, KwHidden)
	native := readExact(src, KwNative)
	name := readIdentifier(src, false)

	if nil == name {
		err = newError(*src, "expected data name here")
		return
	}

	def = &parsed.Definition{
		Location: loc(src, cursor),
		Hidden:   hidden,
		Name:     ast.Identifier(*name),
	}

	typeCursor := src.cursor
	params, ret, typeErr := parseSignature(src)
	if err != nil {
		err = typeErr
		return
	}
	if nil == params {
		if readExact(src, SeqColon) {
			def.Type, err = parseType(src)
			if err != nil {
				return
			}
			if nil == def.Type {
				err = newError(*src, "expected definedReturn here")
				return
			}
		}
		if native {
			def.Expression = parsed.NativeCall{
				Location: loc(src, typeCursor),
				Name:     common.MakeFullIdentifier(modName, ast.Identifier(*name)),
			}
		} else {
			if !readExact(src, SeqEqual) {
				err = newError(*src, "expected `=` here")
				return
			}
			def.Expression, err = parseExpression(src, false)
			if err != nil {
				return
			}
			if nil == def.Expression {
				err = newError(*src, "expected expression here")
				return
			}
		}
	} else {
		if native {
			var args []parsed.Expression
			for _, x := range params {
				if named, ok := x.(parsed.PNamed); ok {
					args = append(args, parsed.Var{
						Location: x.GetLocation(),
						Name:     ast.QualifiedIdentifier(named.Name),
					})
				} else {
					err = newError(*src,
						"native function should start with lowercase letter and cannot be a pattern match")
					return
				}
			}
			def.Expression = parsed.NativeCall{
				Location: loc(src, typeCursor),
				Name:     common.MakeFullIdentifier(modName, ast.Identifier(*name)),
				Args:     args,
			}
		} else {
			if !readExact(src, SeqEqual) {
				err = newError(*src, "expected `=` here")
				return
			}
			def.Expression, err = parseExpression(src, false)
			if err != nil {
				return
			}
			if nil == def.Expression {
				err = newError(*src, "expected expression here")
				return
			}
		}

		def.Type = parsed.TFunc{
			Location: loc(src, typeCursor),
			Params:   common.Map(func(x parsed.Pattern) parsed.Type { return x.GetType() }, params),
			Return:   ret,
		}
		def.Params = params
	}
	return
}

func parseModule(src *source) (module *parsed.Module, errors []error) {
	skipComment(src)

	if !readExact(src, KwModule) {
		errors = append(errors, newError(*src, "expected `module` keyword here"))
		return
	}

	name := readIdentifier(src, true)

	if nil == name {
		errors = append(errors, newError(*src, "expected module name here"))
		return
	}

	m := parsed.Module{
		Name:     *name,
		Location: loc(src, 0),
	}

	for {
		imp, err := parseImport(src)
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
		}
		if imp == nil {
			break
		}
		m.Imports = append(m.Imports, *imp)
	}

	for {
		alias, err := parseAlias(src)
		if alias != nil {
			m.Aliases = append(m.Aliases, *alias)
			if err == nil {
				continue
			}
		}
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
			continue
		}

		infixFn, err := parseInfixFn(src)
		if infixFn != nil {
			m.InfixFns = append(m.InfixFns, *infixFn)
			if err == nil {
				continue
			}
		}
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
			continue
		}

		definition, err := parseDefinition(src, *name)
		if definition != nil {
			m.Definitions = append(m.Definitions, *definition)
			if err == nil {
				continue
			}
		}
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
			continue
		}

		dataType, err := parseDataType(src)

		if dataType != nil {
			m.DataTypes = append(m.DataTypes, *dataType)
			if err == nil {
				continue
			}
		}
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
			continue
		}

		if isOk(src) {
			errors = append(errors, newError(*src, "failed to parse statement"))
			if skipToNextStatement(src) {
				continue
			}
		}
		break
	}

	return &m, errors
}

func skipToNextStatement(src *source) bool {
	for isOk(src) {
		src.cursor++
		start := src.cursor

		if readExact(src, KwAlias) ||
			readExact(src, KwDef) ||
			readExact(src, KwType) ||
			readExact(src, KwInfix) ||
			readExact(src, KwModule) {
			src.cursor = start
			return true
		}
	}
	return false
}
