package misc

import (
	"fmt"
	"golang.org/x/exp/slices"
	"strconv"
	"strings"
	"unicode"
)

var kNumBin = []string{"0", "1"}
var kNumOct = []string{"0", "1", "2", "3", "4", "5", "6", "7"}
var kNumDec = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var kNumHex = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
var kGoReserved = []string{
	"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough",
	"for", "func", "go", "goto", "if", "import", "interface", "map", "package",
	"range", "rF_False(struct{}{})urn", "select", "struct", "switch", "type", "var",
}
var kNotInfix = []string{"(", ")", "[", "]", "{", "}", ",", ".", ";", "\"", "'"}

func NewCursor(fileName string, text []rune) Cursor {
	return Cursor{FileName: fileName, text: text}
}

type Cursor struct {
	Pos      int
	FileName string
	text     []rune
	markers  []int
}

func (c *Cursor) IsEof() bool {
	return c.Pos >= len(c.text)
}

func (c *Cursor) IsOk() bool {
	return !c.IsEof()
}

func (c *Cursor) Position() (line int, column int) {
	line = 0

	wasCaretReturn := false
	for i := 0; i < c.Pos; i++ {
		switch c.text[i] {
		case '\r':
			line++
			column = 0
			wasCaretReturn = true
			break
		case '\n':
			if !wasCaretReturn {
				line++
				column = 0
			}
			wasCaretReturn = false
			break
		default:
			column++
		}
	}
	return
}

func (c *Cursor) SkipWhiteSpace() {
	for c.IsOk() && unicode.IsSpace(c.text[c.Pos]) {
		c.Pos++
	}
}

func (c *Cursor) SkipComment() {
	if !c.IsOk() {
		return
	}

	c.SkipWhiteSpace()
	if c.Exact("--") {
		for c.IsOk() && c.OneOf("\r", "\n") == "" {
			c.Pos++
		}
	} else if c.Exact("{--") {
		level := 1
		for c.IsOk() {
			if c.Exact("{--") {
				level++
			} else if c.Exact("--}") {
				level--
				if level == 0 {
					break
				}
			}
			c.Pos++
		}
		if level != 0 {
			return
		}
	} else {
		c.SkipWhiteSpace()
		return
	}

	c.SkipWhiteSpace()
	c.SkipComment()
}

func isIdent(first bool, c rune) bool {
	if unicode.IsLetter(c) {
		return true
	}
	if c == '_' || unicode.IsNumber(c) {
		return !first
	}
	return false
}

func (c *Cursor) Identifier() string {
	if !c.IsOk() {
		return ""
	}
	start := c.Pos
	first := true
	for c.IsOk() && isIdent(first, c.text[c.Pos]) {
		c.Pos++
		first = false
	}
	ident := c.text[start:c.Pos]
	c.SkipComment()
	sIdent := string(ident)

	for _, r := range kGoReserved {
		if sIdent == r {
			sIdent = "_" + sIdent
			break
		}
	}
	return sIdent
}

func (c *Cursor) QualifiedIdentifier() string {
	var id string
	start := c.Pos
	for {
		p := c.Identifier()
		if p == "" {
			break
		}
		id += p
		if !c.Exact(".") {
			break
		}
		id += "."
	}
	if len(id) == 0 || id[len(id)-1] == '.' {
		c.Pos = start
		return ""
	}
	c.SkipComment()
	return id
}

func (c *Cursor) Exact(s string) bool {
	if !c.IsOk() {
		return false
	}

	start := c.Pos
	wasIdent := false
	for i, r := range []rune(s) {
		wasIdent = isIdent(i == 0, r)
		if r != c.text[c.Pos] {
			c.Pos = start
			return false
		}
		c.Pos++
	}
	if c.IsOk() && wasIdent && isIdent(false, c.text[c.Pos]) {
		c.Pos = start
		return false
	}
	c.SkipComment()
	return true
}

func (c *Cursor) ExactIgnoreCaseNoSpaces(s string) bool {
	if !c.IsOk() {
		return false
	}

	start := c.Pos
	for _, r := range []rune(s) {
		if !strings.EqualFold(string(r), string(c.text[c.Pos])) {
			c.Pos = start
			return false
		}
		c.Pos++
	}

	return true
}

func (c *Cursor) Number() (value string, integer bool) {
	pos := c.Pos
	fv := c.float()
	fvPos := c.Pos

	c.Pos = pos
	iv := c.integer()

	if fv == "" {
		return iv, true
	}
	if iv == "" {
		c.Pos = fvPos
		return fv, false
	}
	if fv == iv {
		return iv, true
	}

	c.Pos = fvPos
	return fv, false
}

func (c *Cursor) integer() string {
	pos := c.Pos
	value, base := c.integerPart(true)
	if c.IsOk() && (unicode.IsLetter(c.text[c.Pos]) || unicode.IsNumber(c.text[c.Pos])) {
		c.Pos = pos
		return ""
	}
	switch base {
	case 2:
		value = "0b" + value
		break
	case 8:
		value = "0o" + value
		break
	case 16:
		value = "0x" + value
		break
	}
	c.SkipComment()
	return value
}

func (c *Cursor) float() string {
	pos := c.Pos

	first, _ := c.integerPart(false)
	if first == "" {
		return ""
	}

	if c.Exact(".") {
		second, base := c.integerPart(false)
		if base == 0 {
			return ""
		}
		first += "." + second
	} else if c.ExactIgnoreCaseNoSpaces("e") {
		sign := c.OneOf("-", "+")
		if sign == "" {
			return ""
		}
		second, base := c.integerPart(false)
		if base == 0 {
			return ""
		}
		first += "e" + sign + second
	}
	if c.IsOk() && (unicode.IsLetter(c.text[c.Pos]) || unicode.IsNumber(c.text[c.Pos])) {
		c.Pos = pos
		return ""
	}
	c.SkipComment()
	return first
}

func (c *Cursor) integerPart(allowBases bool) (string, int) {
	if !c.IsOk() {
		return "", 0
	}
	base := 10
	if allowBases {
		if c.ExactIgnoreCaseNoSpaces("0x") {
			base = 16
		} else if c.ExactIgnoreCaseNoSpaces("0b") {
			base = 2
		} else if c.ExactIgnoreCaseNoSpaces("0o") || c.ExactIgnoreCaseNoSpaces("0") {
			base = 8
		}
	}

	value := ""
	var nums []string
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
		if c.Exact("_") {
			continue
		}
		if x := c.OneOfIgnoreCaseNoSpaces(nums...); x != "" {
			value += x
		} else {
			break
		}
	}

	if value == "" {
		if base == 8 {
			return "0", 10
		}
		return "", 0
	}

	return value, base
}

func (c *Cursor) Char() string {
	if !c.IsOk() {
		return ""
	}

	start := c.Pos
	if !c.Exact("'") {
		return ""
	}

	if c.Exact("\\''") {
		return "'\\''"
	}

	if c.IsOk() {
		c.Pos++
	}

	if !c.Exact("'") {
		c.Pos = start
		return ""
	}

	c.SkipComment()
	return "'" + string(c.text[c.Pos-1]) + "'"
}

func (c *Cursor) String() string {
	start := c.Pos
	if !c.Exact("\"") {
		return ""
	}
	skipNextQuotes := false
	for {
		if !c.IsOk() {
			c.Pos = start
			return ""
		}
		if c.ExactIgnoreCaseNoSpaces("\"") && !skipNextQuotes {
			break
		}
		skipNextQuotes = c.ExactIgnoreCaseNoSpaces("\\")
		c.Pos++
	}
	end := c.Pos
	c.SkipComment()
	return string(c.text[start:end])
}

func (c *Cursor) OpenParenthesis() bool {
	return c.Exact("(")
}

func (c *Cursor) CloseParenthesis() bool {
	return c.Exact(")")
}

func (c *Cursor) OpenBrackets() bool {
	return c.Exact("[")
}

func (c *Cursor) CloseBrackets() bool {
	return c.Exact("]")
}

func (c *Cursor) OpenBraces() bool {
	return c.Exact("{")
}

func (c *Cursor) CloseBraces() bool {
	return c.Exact("}")
}

func (c *Cursor) OneOf(s ...string) string {
	for _, x := range s {
		if c.Exact(x) {
			return x
		}
	}
	return ""
}

func (c *Cursor) OneOfIgnoreCaseNoSpaces(s ...string) string {
	for _, x := range s {
		if c.ExactIgnoreCaseNoSpaces(x) {
			return x
		}
	}
	return ""
}

func (c *Cursor) InfixNameWithParenthesis() string {
	for !c.OpenParenthesis() {
		return ""
	}

	name := c.infix()

	if !c.CloseParenthesis() {
		return ""
	}

	c.SkipComment()

	return name
}

func (c *Cursor) InfixName() string {
	name := c.infix()
	if name != "" {
		c.SkipComment()
	}
	return name
}

func (c *Cursor) infix() string {
	pos := c.Pos
	var runes []rune
	for {
		if !c.IsOk() {
			return ""
		}
		r := c.text[c.Pos]

		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == ')' {
			break
		}

		runes = append(runes, r)
		c.Pos++
	}

	if slices.Contains(kNotInfix, string(runes)) {
		c.Pos = pos
		return ""
	}

	return string(runes)
}

func (c *Cursor) ShowPosition(info string) string {
	const (
		colorReset      = "\033[0m"
		colorFileName   = "\033[31m"
		colorLineNumber = "\033[34m"
		colorMarker     = "\033[34m"
		colorError      = "\033[31m"
	)

	lines := strings.Split(string(c.text), "\n")
	line, column := c.Position()
	min := line - 3
	if min < 0 {
		min = 0
	}
	max := line + 3
	if max > len(lines) {
		max = len(lines)
	}

	sb := strings.Builder{}
	sb.WriteString(colorFileName)
	sb.WriteString(c.FileName)
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(line + 1))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(column + 1))
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	offset := len(strconv.Itoa(max)) + 2

	for i := min; i < max; i++ {
		sb.WriteString(colorLineNumber)
		num := fmt.Sprintf("%d", i+1)
		sb.WriteString(num)

		for i := len(num); i < offset; i++ {
			sb.WriteString(" ")
		}
		sb.WriteString(colorReset)

		sb.WriteString(lines[i])
		sb.WriteString("\n")
		if i == line {
			sb.WriteString(colorMarker)

			for i := 0; i < offset; i++ {
				sb.WriteString("~")
			}

			if column > 0 {
				sb.WriteString(strings.Repeat("~", column))
			}

			sb.WriteString("^ ")
			sb.WriteString(colorError)
			sb.WriteString(info)
			sb.WriteString(colorReset)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
