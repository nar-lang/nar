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

var Keywords = []string{
	KwModule, KwImport, KwAs, KwExposing, KwInfix, KwAlias, KwType, KwDef, KwHidden, KwNative, KwLeft, KwRight, KwNon, KwIf, KwThen, KwElse, KwLet, KwIn, KwSelect, KwCase, KwEnd,
}

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

func loc(src *source, start uint32) ast.Location {
	if src.cursor == 0 {
		return ast.NewLocation(src.filePath, src.text, 0, 0)
	}
	end := src.cursor - 1
	for end > start && unicode.IsSpace(src.text[end]) {
		end--
	}
	end++
	return ast.NewLocation(src.filePath, src.text, start, end)
}

func newError(src source, msg string) error {
	return common.NewErrorAt(loc(&src, src.cursor), msg)
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
			return parsed.NewTUnit(loc(src, cursor)), nil
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
			return parsed.NewTFunc(loc(src, cursor), items, ret), nil
		} else {
			if 1 == len(items) {
				return items[0], nil
			} else {
				return parsed.NewTTuple(loc(src, cursor), items), nil
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

		return parsed.NewTRecord(loc(src, cursor), fields), nil
	}

	if name := readIdentifier(src, true); nil != name {
		if unicode.IsLower([]rune(*name)[0]) {
			return parsed.NewTParameter(loc(src, cursor), ast.Identifier(*name)), nil
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

			return parsed.NewTNamed(loc(src, cursor), *name, typeParams), nil
		}
	}
	return nil, nil
}

func parsePattern(src *source) (parsed.Pattern, error) {
	cursor := src.cursor

	//tuple/unit
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return finishParsePattern(src, parsed.NewPConst(loc(src, cursor), ast.CUnit{}))
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
		return finishParsePattern(src, parsed.NewPTuple(loc(src, cursor), items))
	}

	//record
	if readExact(src, SeqBracesOpen) {
		var fields []*parsed.PRecordField
		for {
			fieldCursor := src.cursor
			name := readIdentifier(src, false)
			if nil == name {
				return nil, newError(*src, "expected record field name here")
			}
			fields = append(fields, parsed.NewPRecordField(loc(src, fieldCursor), ast.Identifier(*name)))

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `}` here")
		}

		return finishParsePattern(src, parsed.NewPRecord(loc(src, cursor), fields))
	}

	//list
	if readExact(src, SeqBracketsOpen) {
		if readExact(src, SeqBracketsClose) {
			return finishParsePattern(src, parsed.NewPList(loc(src, cursor), nil))
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

		return finishParsePattern(src, parsed.NewPList(loc(src, cursor), items))
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
		return finishParsePattern(src, parsed.NewPOption(loc(src, cursor), *name, items))
	} else {
		src.cursor = cursor
	}

	name = readIdentifier(src, false)
	if nil != name && unicode.IsLower([]rune(*name)[0]) {
		return finishParsePattern(src, parsed.NewPNamed(loc(src, cursor), ast.Identifier(*name)))
	} else {
		src.cursor = cursor
	}

	//anything
	if readExact(src, SeqUnderscore) {
		return finishParsePattern(src, parsed.NewPAny(loc(src, cursor)))
	}

	const_, err := parseConst(src)
	if err != nil {
		return nil, err
	}
	if nil != const_ {
		return finishParsePattern(src, parsed.NewPConst(loc(src, cursor), const_))
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
		pattern.SetDeclaredType(type_)
		return finishParsePattern(src, pattern)
	}

	if readExact(src, KwAs) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected pattern alias name here")
		}
		return finishParsePattern(src,
			parsed.NewPAlias(loc(src, cursor), ast.Identifier(*name), pattern))
	}

	if readExact(src, SeqBar) {
		tail, err := parsePattern(src)
		if err != nil {
			return nil, err
		}
		if nil == tail {
			return nil, newError(*src, "expected list tail pattern here")
		}

		return finishParsePattern(src, parsed.NewPCons(loc(src, cursor), pattern, tail))
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
		return finishParseExpression(src, parsed.NewConst(loc(src, cursor), const_), negate)
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
		return finishParseExpression(src, parsed.NewList(loc(src, cursor), items), negate)
	}

	//negate
	if readExact(src, SeqMinus) {
		return parseExpression(src, !negate)
	}

	//infix value
	infix := parseInfixIdentifier(src, true)
	if nil != infix {
		return finishParseExpression(src, parsed.NewInfixVar(loc(src, cursor), *infix), negate)
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
			parsed.NewLambda(loc(src, cursor), patterns, ret, body), negate)
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
		return finishParseExpression(src, parsed.NewIf(loc(src, cursor), condition, positive, negative), negate)
	}

	//let
	if readExact(src, KwLet) {
		defCursor := src.cursor
		name := readIdentifier(src, false)
		nameLoc := loc(src, defCursor)
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
			pattern = parsed.NewPNamed(loc(src, defCursor), ast.Identifier(*name))
			fnType = parsed.NewTFunc(
				loc(src, typeCursor),
				common.Map(func(x parsed.Pattern) parsed.Type { return x.Type() }, params),
				ret)
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
			return finishParseExpression(src,
				parsed.NewFunction(loc(src, cursor), ast.Identifier(*name), nameLoc, params, value, fnType, nested),
				negate)
		} else {
			return finishParseExpression(src,
				parsed.NewLet(loc(src, cursor), pattern, value, nested),
				negate)
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

		var cases []*parsed.SelectCase

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
			cases = append(cases, parsed.NewSelectCase(loc(src, caseCursor), pattern, expr))
		}

		if 0 == len(cases) {
			return nil, newError(*src, "expected case expression here")
		}
		return finishParseExpression(src, parsed.NewSelect(loc(src, cursor), condition, cases), negate)
	}

	//accessor
	if readExact(src, SeqDot) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected accessor name here")
		}
		return finishParseExpression(src, parsed.NewAccessor(loc(src, cursor), ast.Identifier(*name)), negate)
	}

	//record / update
	if readExact(src, SeqBracesOpen) {
		if readExact(src, SeqBracesClose) {
			return finishParseExpression(src, parsed.NewRecord(loc(src, cursor), nil), negate)
		}

		recCursor := src.cursor

		name := readIdentifier(src, true)
		if nil != name && !readExact(src, SeqBar) {
			src.cursor = recCursor
			name = nil
		}

		var fields []*parsed.RecordField
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
			fields = append(fields, parsed.NewRecordField(loc(src, fieldCursor), ast.Identifier(*fieldName), expr))

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqBracesClose) {
				break
			}
			return nil, newError(*src, "expected `,` or `}` here")
		}

		if nil == name {
			return finishParseExpression(src, parsed.NewRecord(loc(src, cursor), fields), negate)
		} else {
			return finishParseExpression(src, parsed.NewUpdate(loc(src, cursor), *name, fields), negate)
		}
	}

	//tuple / void / precedence
	if readExact(src, SeqParenthesisOpen) {
		if readExact(src, SeqParenthesisClose) {
			return finishParseExpression(src, parsed.NewConst(loc(src, cursor), ast.CUnit{}), negate)
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
			if bop, ok := expr.(*parsed.BinOp); ok {
				bop.SetInParentheses(true)
				expr = bop
			}
			return finishParseExpression(src, expr, negate)
		} else {
			return finishParseExpression(src, parsed.NewTuple(loc(src, cursor), items), negate)
		}
	}

	name := readIdentifier(src, true)
	if nil != name {
		return finishParseExpression(src, parsed.NewVar(loc(src, cursor), *name), negate)
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
			expr = parsed.NewNegate(loc(src, cursor), expr)
		}

		items := []*parsed.BinOpItem{
			parsed.NewBinOpOperand(expr),
			parsed.NewBinOpFunc(*infixOp),
		}

		if bop, ok := final.(*parsed.BinOp); ok && !bop.InParentheses() {
			items = append(items, bop.Items()...)
		} else {
			items = append(items, parsed.NewBinOpOperand(final))
		}

		return parsed.NewBinOp(loc(src, expr.Location().Start()), items, false), nil
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
		return finishParseExpression(src, parsed.NewApply(loc(src, expr.Location().Start()), expr, items), negate)
	}

	if readExact(src, SeqDot) {
		name := readIdentifier(src, false)
		if nil == name {
			return nil, newError(*src, "expected field name here")
		}
		return finishParseExpression(src, parsed.NewAccess(loc(src, cursor), expr, ast.Identifier(*name)), negate)
	}
	if negate {
		expr = parsed.NewNegate(loc(src, expr.Location().Start()), expr)
	}
	return expr, nil
}

func parseDataOption(src *source) (parsed.DataTypeOption, error) {
	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	var types []*parsed.DataTypeValue

	name := readIdentifier(src, false)

	if nil == name {
		return nil, newError(*src, "expected option name here")
	}
	if readExact(src, SeqParenthesisOpen) {
		index := 0
		for {
			argCursor := src.cursor
			valueName := readIdentifier(src, false)
			if valueName == nil || !readExact(src, SeqColon) {
				src.cursor = argCursor
				n := ast.QualifiedIdentifier(fmt.Sprintf("p%d", index))
				valueName = &n
				index++
			}

			type_, err := parseType(src)
			if err != nil {
				return nil, err
			}
			if nil == type_ {
				return nil, newError(*src, "expected option value type here")
			}
			types = append(types, parsed.NewDataTypeValue(loc(src, argCursor), ast.Identifier(*valueName), type_))

			if readExact(src, SeqComma) {
				continue
			}
			if readExact(src, SeqParenthesisClose) {
				break
			}
		}
	}

	return parsed.NewDataTypeOption(loc(src, cursor), hidden, ast.Identifier(*name), types), nil
}

func parseImport(src *source) (parsed.Import, error) {
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
	return parsed.NewImport(loc(src, cursor), *ident, (*ast.Identifier)(alias), exposingAll, exposing), nil
}

func parseInfixFn(src *source) (parsed.Infix, error) {
	if !readExact(src, KwInfix) {
		return nil, nil
	}
	var err error
	cursor := src.cursor
	hidden := readExact(src, KwHidden)

	var name ast.InfixIdentifier
	var associativity parsed.Associativity
	var precedence int
	var alias ast.Identifier
	var aliasCursor uint32

	pName := parseInfixIdentifier(src, true)
	if nil == pName {
		err = newError(*src, "expected infix statement name here")
	}
	if err == nil {
		name = *pName

		if !readExact(src, SeqColon) {
			err = newError(*src, "expected `:` here")
		}
	}
	if err == nil {
		if !readExact(src, SeqParenthesisOpen) {
			err = newError(*src, "expected `(` here")
		}
	}
	if err == nil {
		if readExact(src, KwLeft) {
			associativity = parsed.Left
		} else if readExact(src, KwRight) {
			associativity = parsed.Right
		} else if readExact(src, KwNon) {
			associativity = parsed.None
		} else {
			err = newError(*src, "expected `left`, `right` or `non` here")
		}
	}

	var pPrecedence *int64
	if err == nil {
		pPrecedence, err = parseInt(src)
	}
	if err == nil {
		if pPrecedence == nil {
			err = newError(*src, "expected precedence (integer number) here")
		}
	}
	if err == nil {
		precedence = int(*pPrecedence)
	}

	if err == nil {
		if !readExact(src, SeqParenthesisClose) {
			err = newError(*src, "expected `)` here")
		}
	}

	if err == nil {
		if !readExact(src, SeqEqual) {
			err = newError(*src, "expected `=` here")
		}
	}

	var pAlias *ast.QualifiedIdentifier
	if err == nil {
		aliasCursor = src.cursor
		pAlias = readIdentifier(src, false)
	}
	if pAlias == nil {
		err = newError(*src, "expected definition name here")
	}
	if err == nil {
		alias = ast.Identifier(*pAlias)
	}

	return parsed.NewInfix(loc(src, cursor), hidden, name, associativity, precedence, loc(src, aliasCursor), alias), err
}

func parseAlias(src *source) (parsed.Alias, error) {
	if !readExact(src, KwAlias) {
		return nil, nil
	}

	var err error
	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	native := readExact(src, KwNative)
	var params []ast.Identifier
	var type_ parsed.Type
	var name ast.Identifier

	pName := readIdentifier(src, false)
	if pName == nil {
		err = newError(*src, "expected alias name here")
	}
	if err == nil {
		name = ast.Identifier(*pName)
	}

	if err == nil {
		params, err = parseTypeParamNames(src)
	}
	if err == nil {
		if !native {
			if !readExact(src, SeqEqual) {
				err = newError(*src, "expected `=` here")
			}
			if err == nil {
				type_, err = parseType(src)
			}
			if err == nil {
				if type_ == nil {
					err = newError(*src, "expected definedReturn declaration here")
				}
			}
		}
	}

	return parsed.NewAlias(loc(src, cursor), hidden, name, params, type_), err
}

func parseDataType(src *source) (parsed.DataType, error) {
	if !readExact(src, KwType) {
		return nil, nil
	}

	var err error
	cursor := src.cursor
	hidden := readExact(src, KwHidden)
	var name ast.Identifier
	var params []ast.Identifier
	var options []parsed.DataTypeOption

	pName := readIdentifier(src, false)
	if pName == nil {
		err = newError(*src, "expected data name here")
	}
	if err == nil {
		name = ast.Identifier(*pName)
	}

	params, err = parseTypeParamNames(src)
	if err == nil {
		if !readExact(src, SeqEqual) {
			err = newError(*src, "expected `=` here")
		}
	}

	for err == nil {
		var option parsed.DataTypeOption
		option, err = parseDataOption(src)
		if err == nil {
			options = append(options, option)
			if !readExact(src, SeqBar) {
				break
			}
		}
	}

	return parsed.NewDataType(loc(src, cursor), hidden, name, params, options), err
}

func parseDefinition(src *source, modName ast.QualifiedIdentifier) (parsed.Definition, error) {
	cursor := src.cursor

	if !readExact(src, KwDef) {
		return nil, nil
	}
	hidden := readExact(src, KwHidden)
	native := readExact(src, KwNative)
	name := readIdentifier(src, false)
	var type_ parsed.Type
	var body parsed.Expression

	if nil == name {
		return nil, newError(*src, "expected data name here")
	}

	typeCursor := src.cursor
	params, ret, err := parseSignature(src)
	if err == nil {
		if nil == params {
			if readExact(src, SeqColon) {
				type_, err = parseType(src)
				if err == nil && nil == type_ {
					err = newError(*src, "expected definedReturn here")
				}
			}
			if err == nil {
				if native {
					body = parsed.NewCall(
						loc(src, typeCursor),
						common.MakeFullIdentifier(modName, ast.Identifier(*name)),
						nil)
				} else {
					if !readExact(src, SeqEqual) {
						err = newError(*src, "expected `=` here")
					}
					if err == nil {
						body, err = parseExpression(src, false)
					}
					if err == nil && body == nil {
						err = newError(*src, "expected expression here")
					}
				}
			}
		} else {
			if native {
				var args []parsed.Expression
				for _, x := range params {
					if named, ok := x.(*parsed.PNamed); ok {
						args = append(args, parsed.NewVar(x.Location(), ast.QualifiedIdentifier(named.Name())))
					} else {
						err = newError(*src,
							"native function should start with lowercase letter and cannot be a pattern match")
						break
					}
				}
				if err == nil {
					body = parsed.NewCall(
						loc(src, typeCursor),
						common.MakeFullIdentifier(modName, ast.Identifier(*name)),
						args)
				}
			} else {
				if !readExact(src, SeqEqual) {
					err = newError(*src, "expected `=` here")
				}
				if err == nil {
					body, err = parseExpression(src, false)
				}
				if err == nil && body == nil {
					err = newError(*src, "expected expression here")
				}
			}

			if err == nil {
				if ret != nil || common.Any(func(x parsed.Pattern) bool { return x.Type() != nil }, params) {
					type_ = parsed.NewTFunc(
						loc(src, typeCursor),
						common.Map(func(x parsed.Pattern) parsed.Type { return x.Type() }, params),
						ret)

				}
			}
		}
	}
	return parsed.NewDefinition(loc(src, cursor), hidden, ast.Identifier(*name), params, body, type_), err
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

	var imports []parsed.Import
	var aliases []parsed.Alias
	var infixFns []parsed.Infix
	var definitions []parsed.Definition
	var dataTypes []parsed.DataType

	for {
		imp, err := parseImport(src)
		if err != nil {
			errors = append(errors, err)
			skipToNextStatement(src)
		}
		if imp == nil {
			break
		}
		imports = append(imports, imp)
	}

	for {
		alias, err := parseAlias(src)
		if alias != nil {
			aliases = append(aliases, alias)
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
			infixFns = append(infixFns, infixFn)
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
			definitions = append(definitions, definition)
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
			dataTypes = append(dataTypes, dataType)
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

	return parsed.NewModule(*name, loc(src, 0), imports, aliases, infixFns, definitions, dataTypes), errors
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
