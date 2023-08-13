package compiler

import (
	"errors"
	"fmt"
	"oak-compiler/pkg/misc"
	"oak-compiler/pkg/parsed"
	"strconv"
	"unicode"
)

const (
	kwModule   = "module"
	kwImport   = "import"
	kwFrom     = "from"
	kwAs       = "as"
	kwExposing = "exposing"
	kwHidden   = "hidden"
	kwExtern   = "extern"
	kwType     = "type"
	kwDef      = "def"
	kwLeft     = "left"
	kwRight    = "right"
	kwNon      = "non"
	kwIf       = "if"
	kwThen     = "then"
	kwElse     = "else"
	kwSelect   = "select"
	kwCase     = "case"
	kwLet      = "let"
	kwIn       = "in"
)

var expressionStopWords = []string{")", kwHidden, kwType, kwDef, kwThen, kwElse, kwCase, kwIn}

func ParseModule(c *misc.Cursor, packageName parsed.PackageFullName) (parsed.Module, error) {
	moduleDefinition, err := parseModuleStatement(c)
	if err != nil {
		return parsed.Module{}, err
	}

	var imports []parsed.StatementImport
	for {
		imp, ok, err := parseImportStatement(c)
		if err != nil {
			return parsed.Module{}, err
		}
		if !ok {
			break
		}
		imports = append(imports, imp)
	}

	var definitions []parsed.Definition
	var order []string
	for !c.IsEof() {
		definition, err := parseDefinition(
			c, parsed.NewModuleFullName(packageName, moduleDefinition.Name()),
		)
		if err != nil {
			return parsed.Module{}, err
		}
		definitions = append(definitions, definition)
		order = append(order, definition.Name())
	}

	return parsed.NewModule(moduleDefinition, imports, definitions), nil
}

func parseModuleStatement(c *misc.Cursor) (parsed.StatementModule, error) {
	c.SkipComment()

	if !c.Exact(kwModule) {
		return parsed.StatementModule{}, misc.NewError(*c, "expected `module` keyword here")
	}
	name, _ := c.QualifiedIdentifier()
	if name == "" {
		return parsed.StatementModule{}, misc.NewError(*c, "expected identifier here")
	}
	return parsed.NewModuleStatement(name), nil
}

func parseImportStatement(c *misc.Cursor) (parsed.StatementImport, bool, error) {
	startCursor := *c
	if !c.Exact(kwImport) {
		return parsed.StatementImport{}, false, nil
	}
	module, _ := c.QualifiedIdentifier()
	alias := module
	if module == "" {
		return parsed.StatementImport{}, false, misc.NewError(*c, "expected module name here")
	}
	if c.Exact(kwAs) {
		alias, _ = c.Identifier()
		if alias == "" {
			return parsed.StatementImport{}, false, misc.NewError(*c, "expected module alias here")
		}
	}
	var package_ parsed.PackageFullName
	if c.Exact(kwFrom) {
		package_ = parsed.PackageFullName(c.String())
		if package_ == "" {
			return parsed.StatementImport{}, false, misc.NewError(*c, "expected package url here")
		}
		package_ = package_[1 : len(package_)-1]
	}
	exposingAll := false
	var exposing []string
	if c.Exact(kwExposing) {
		if c.Exact("*") {
			exposingAll = true
		} else {
			if !c.OpenParenthesis() {
				return parsed.StatementImport{}, false, misc.NewError(*c, "expected `(` here")
			}

			for {
				id := c.InfixNameWithParenthesis()
				if id == "" {
					id, _ = c.Identifier()
				}
				if id == "" {
					return parsed.StatementImport{}, false, misc.NewError(*c, "expected identifier here")
				}
				exposing = append(exposing, id)
				if c.Exact(",") {
					continue
				}
				if c.CloseParenthesis() {
					break
				}
				return parsed.StatementImport{}, false, misc.NewError(*c, "expected `,` or `)` here")
			}
		}
	}

	return parsed.NewImportStatement(startCursor, package_, module, alias, exposingAll, exposing), true, nil
}

func parseDefinition(c *misc.Cursor, modName parsed.ModuleFullName) (parsed.Definition, error) {
	startCursor := *c
	hidden := c.Exact(kwHidden)

	kw := c.OneOf(kwType, kwDef)
	if kw == "" {
		return nil, misc.NewError(*c, "expected `type` or `def` keyword here")
	}

	name := c.InfixNameWithParenthesis()
	infix := false

	if name != "" {
		infix = true
	} else {
		name, _ = c.Identifier()
	}

	if name == "" {
		return nil, misc.NewError(*c, "expected definition name here")
	}
	if kw == kwType && !unicode.IsUpper([]rune(name)[0]) {
		return nil, misc.NewError(*c, "type name should start with uppercase letter")
	}
	if kw == kwDef && unicode.IsUpper([]rune(name)[0]) {
		return nil, misc.NewError(*c, "definition name should start with lowercase letter")
	}

	gps, err := parseGenericParameters(c, modName)
	if err != nil {
		return nil, err
	}

	switch kw {
	case kwType:
		if !c.Exact("=") {
			return nil, misc.NewError(*c, "expected `=` here")
		}

		var type_ parsed.Type
		extern := c.Exact(kwExtern)
		if extern {
			type_ = parsed.NewAddressedType(
				startCursor, modName, parsed.NewDefinitionAddress(modName, name), nil, false,
			)
		} else {
			type_, err = parseType(c, modName, true, false, true, gps)
			if err != nil {
				return nil, err
			}
		}
		return parsed.NewTypeDefinition(
			startCursor, parsed.NewDefinitionAddress(modName, name), gps, hidden, extern, type_,
		), nil

	case kwDef:
		if !c.Exact(":") {
			return nil, misc.NewError(*c, "expected `:` here")
		}
		if infix {
			if !c.OpenParenthesis() {
				return nil, misc.NewError(*c, "expected `(` here")
			}

			var assoc parsed.InfixAssociativity
			if c.Exact(kwLeft) {
				assoc = parsed.InfixAssociativityLeft
			} else if c.Exact(kwRight) {
				assoc = parsed.InfixAssociativityRight
			} else if c.Exact(kwNon) {
				assoc = parsed.InfixAssociativityNon
			} else {
				return nil, misc.NewError(*c, "expected `left`, `right` or `non` here")
			}

			sPriority, integer := c.Number()
			if sPriority == "" || !integer {
				return nil, misc.NewError(*c, "expected priority value (integer) here")
			}

			priority, _ := strconv.ParseInt(sPriority, 10, 32)

			if !c.CloseParenthesis() {
				return nil, misc.NewError(*c, "expected `)` here")
			}

			if !c.Exact("=") {
				return nil, misc.NewError(*c, "expected `=` here")
			}

			alias, _ := c.Identifier()
			if alias == "" {
				return nil, misc.NewError(*c, "expected infix function alias here")
			}

			return parsed.NewInfixDefinition(
				startCursor, parsed.NewDefinitionAddress(modName, name), hidden, assoc, int(priority), alias,
			), nil
		} else {
			type_, err := parseType(c, modName, false, false, false, gps)
			if err != nil {
				return nil, err
			}
			if !c.Exact("=") {
				return nil, misc.NewError(*c, "expected `=` here")
			}
			var ex parsed.Expression
			extern := c.Exact(kwExtern)
			if !extern {
				ex, err = parseExpression(c, modName, gps)
				if err != nil {
					return nil, err
				}
			}
			return parsed.NewFuncDefinition(
				startCursor, parsed.NewDefinitionAddress(modName, name), gps, hidden, extern, type_, ex,
			), nil
		}
	default:
		return nil, misc.NewError(startCursor, "impossible branch type/def, this is a compiler error")
	}
}

func parseGenericParameters(c *misc.Cursor, modName parsed.ModuleFullName) (parsed.GenericParams, error) {
	if !c.OpenBrackets() {
		return nil, nil
	}

	var parameters parsed.GenericParams

	for {
		eltStart := *c
		name, _ := c.Identifier()
		if unicode.IsLower([]rune(name)[0]) {
			return nil, misc.NewError(eltStart, "generic parameter name should start with uppercase letter")
		}

		constraint := parsed.GenericConstraint(parsed.GenericConstraintAny{})
		if c.Exact(":") {
			csStart := *c
			cs, _ := c.Identifier()
			if cs == "" {
				return nil, misc.NewError(*c, "expected generic parameter constraint here")
			}
			switch cs {
			case "Any":
				constraint = parsed.GenericConstraintAny{}
				break
			case "Comparable":
				constraint = parsed.GenericConstraintComparable{}
				break
			case "Equatable":
				constraint = parsed.GenericConstraintEquatable{}
				break
			case "Number":
				constraint = parsed.GenericConstraintNumber{}
				break
			default:
				return nil, misc.NewError(csStart, "unsupported generic type `%s`, expected `Any`, `Comparable`, `Equatable` of `Number`", cs)
			}
		}

		parameters = append(parameters, parsed.NewGenericParam(eltStart, modName, name, constraint))

		if c.Exact(",") {
			continue
		}
		if c.CloseBrackets() {
			break
		}
		return nil, misc.NewError(*c, "expected `,` or `]` here")
	}

	if len(parameters) == 0 {
		return nil, misc.NewError(*c, "empty generic parameters list")
	}

	return parameters, nil
}

var errNotAType = fmt.Errorf("optional type is not not a type")
var uniqueIndex = 0

func parseType(
	c *misc.Cursor,
	modName parsed.ModuleFullName,
	definition bool,
	optional bool,
	allowSignature bool,
	genericParams parsed.GenericParams,
) (parsed.Type, error) {
	startCursor := *c

	if definition {
		var options []parsed.UnionOption
		for {
			if !c.Exact("|") {
				break
			}

			optionStart := *c
			pos := c.Pos
			name, _ := c.Identifier()
			if name == "" {
				return nil, misc.NewError(*c, "failed tor read union option, expected name here")
			}
			if !unicode.IsUpper([]rune(name)[0]) {
				c.Pos = pos
				return nil, misc.NewError(*c, "option name should start with uppercase letter")
			}
			for _, o := range options {
				if o.Name() == name {
					c.Pos = pos
					return nil, misc.NewError(*c, "union has already declared option `%s`", name)
				}
			}

			type_, err := parseType(c, modName, false, true, false, genericParams)
			if err != nil && err != errNotAType {
				return nil, err
			}
			if type_ == nil {
				type_ = parsed.NewVoidType(*c, modName)
			}
			options = append(options, parsed.NewUnionOption(optionStart, name, type_))
		}
		if len(options) > 0 {
			return parsed.NewUnionType(startCursor, modName, options), nil
		}
	}

	if allowSignature {
		param, err := parseParameter(c, uniqueIndex, true)
		if err != nil {
			return nil, err
		}

		var paramType parsed.Type
		noType := false
		if c.Exact(":") {
			if param == nil {
				return nil, misc.NewError(startCursor, "expected parameter here")
			}
			paramType, err = parseType(c, modName, false, false, false, genericParams)
			if err != nil {
				return nil, err
			}
		} else {
			*c = startCursor
			paramType, err = parseType(c, modName, false, true, false, genericParams)
			noType = errors.Is(err, errNotAType)
		}

		if !noType {
			if c.Exact("->") {
				returnType, err := parseType(c, modName, false, false, true, genericParams)
				if err != nil {
					return nil, err
				}
				return parsed.NewSignatureType(startCursor, modName, paramType, returnType, param), nil
			} else {
				c.Pos = startCursor.Pos
			}
		}
	}

	if c.OpenBraces() {
		start := c.Pos
		c.Identifier()
		if c.Exact(":") {
			var fields []parsed.RecordField
			c.Pos = start
			for {
				fieldStart := *c
				name, _ := c.Identifier()
				if name == "" {
					return nil, misc.NewError(*c, "failed to read record, expected field name here")
				}
				if !unicode.IsLower([]rune(name)[0]) {
					return nil, misc.NewError(*c, "record field name should start with lowercase letter")
				}
				for _, f := range fields {
					if f.Name() == name {
						return nil, misc.NewError(*c, "record has already declared field `%s`", name)
					}
				}
				if !c.Exact(":") {
					return nil, misc.NewError(*c, "failed to read record, expected `:` here")
				}
				type_, err := parseType(c, modName, false, false, false, genericParams)
				if err != nil {
					return nil, err
				}
				fields = append(fields, parsed.NewRecordField(fieldStart, name, type_))
				if c.Exact(",") {
					continue
				}
				if c.CloseBraces() {
					break
				}
				return nil, misc.NewError(*c, "failed to read record, expected `,` or `}` here")
			}
			return parsed.NewRecordType(startCursor, modName, fields), nil
		} else {
			var items []parsed.Type
			c.Pos = start

			for {
				item, err := parseType(c, modName, false, false, true, genericParams)
				if err != nil {
					return nil, misc.NewError(*c, "failed to read tuple type")
				}
				items = append(items, item)

				if c.Exact(",") {
					continue
				}
				if c.CloseBraces() {
					break
				}
				return nil, misc.NewError(*c, "failed to read tuple, expected `,` or `}` here")
			}

			return parsed.NewTupleType(startCursor, modName, items), nil
		}
	}

	if c.OpenParenthesis() {
		type_, err := parseType(c, modName, false, false, true, genericParams)
		if err != nil {
			return nil, err
		}
		if !c.CloseParenthesis() {
			return nil, misc.NewError(*c, "expected `)` here")
		}
		return type_, nil
	}

	for _, g := range genericParams {
		cs := *c
		if c.Exact(g.Name()) {
			return parsed.NewGenericNameType(cs, modName, g.Name()), nil
		}
	}

	typeName, spaceAfter := c.QualifiedIdentifier()
	if typeName != "" {
		if !unicode.IsUpper([]rune(typeName)[0]) {
			if optional {
				c.Pos = startCursor.Pos
				return nil, nil
			}
			c.Pos = startCursor.Pos
			return nil, misc.NewError(*c, "expected type/generic name starting with uppercase letter here")
		}
		var genericArgs parsed.GenericArgs
		if !spaceAfter {
			var err error
			genericArgs, err = parseGenericArgs(c, modName, genericParams)
			if err != nil {
				return nil, err
			}
		}
		return parsed.NewNamedType(startCursor, modName, typeName, genericArgs), nil
	}
	c.Pos = startCursor.Pos

	if !optional {
		return nil, misc.NewError(*c, "expected type declaration here")
	}

	return nil, errNotAType
}

func parseGenericArgs(
	c *misc.Cursor, modName parsed.ModuleFullName, generics parsed.GenericParams,
) (parsed.GenericArgs, error) {
	var genericArgs parsed.GenericArgs
	if c.OpenBrackets() {
		for {
			genericArg, err := parseType(c, modName, false, false, true, generics)
			if err != nil {
				return nil, err
			}
			genericArgs = append(genericArgs, genericArg)
			if c.Exact(",") {
				continue
			}
			if c.CloseBrackets() {
				break
			}
			return nil, misc.NewError(*c, "expected `,` or `]` here")
		}
	}
	return genericArgs, nil
}

func parseExpression(
	c *misc.Cursor, modName parsed.ModuleFullName, generics parsed.GenericParams,
) (parsed.Expression, error) {
	startCursor := *c
	var exs []parsed.Expression

	for {
		eltCursor := *c
		// expression end
		if c.OneOf(expressionStopWords...) != "" {
			c.Pos = eltCursor.Pos
			break
		}

		// nested expression
		if c.OpenParenthesis() {
			nested, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			if !c.CloseParenthesis() {
				return nil, misc.NewError(*c, "failed to read nexted expression, expeted `)` here")
			}
			if inf, ok := nested.(parsed.ExpressionInfix); ok {
				exs = append(exs, inf.AsParameter())
			} else {
				exs = append(exs, nested)
			}
			continue
		}

		// const char
		if constChar := c.Char(); constChar != "" {
			exs = append(exs, parsed.NewConstExpression(eltCursor, parsed.ConstKindChar, constChar))
			continue
		}

		// const int / float
		if constNumber, integer := c.Number(); constNumber != "" {
			if integer {
				exs = append(exs, parsed.NewConstExpression(eltCursor, parsed.ConstKindInt, constNumber))
			} else {
				exs = append(exs, parsed.NewConstExpression(eltCursor, parsed.ConstKindFloat, constNumber))
			}
			continue
		}

		// const string
		if constString := c.String(); constString != "" {
			exs = append(exs, parsed.NewConstExpression(eltCursor, parsed.ConstKindString, constString))
			continue
		}

		// if
		if c.Exact(kwIf) {
			condition, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			if !c.Exact(kwThen) {
				return nil, misc.NewError(*c, "expected `then` here")
			}
			positive, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			if !c.Exact(kwElse) {
				return nil, misc.NewError(*c, "expected `else` here")
			}
			negative, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			exs = append(exs, parsed.NewIfExpression(eltCursor, condition, positive, negative))
			continue
		}

		// select
		if c.Exact(kwSelect) {
			condition, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			var cases []parsed.ExpressionSelectCase
			for {
				if !c.Exact(kwCase) {
					break
				}
				ds, err := parseDecons(c, modName, generics)
				if err != nil {
					return nil, err
				}
				c.Exact("->")
				ex, err := parseExpression(c, modName, generics)
				if err != nil {
					return nil, err
				}
				cases = append(cases, parsed.ExpressionSelectCase{Decons: ds, Expression: ex})
			}
			if len(cases) == 0 {
				return nil, misc.NewError(*c, "expected at least one case here")
			}
			exs = append(exs, parsed.NewSelectExpression(eltCursor, condition, cases))
			continue
		}

		//let
		if c.Exact(kwLet) {
			var defs []parsed.LetDefinition
			index := 0
			for {
				param, err := parseParameter(c, index, false)
				index++

				if !c.Exact(":") {
					return nil, misc.NewError(*c, "expected `:` here")
				}
				protoType, err := parseType(c, modName, false, false, false, generics)
				if err != nil {
					return nil, err
				}
				if !c.Exact("=") {
					return nil, misc.NewError(*c, "expected `=` here")
				}
				expr, err := parseExpression(c, modName, generics)
				if err != nil {
					return nil, err
				}

				defs = append(defs, parsed.NewLetDefinition(param, protoType, expr))

				if c.Exact(";") {
					continue
				}
				if c.Exact(kwIn) {
					break
				}
				return nil, misc.NewError(*c, "expected `;` or `in` here")
			}

			expr, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}

			exs = append(exs, parsed.NewLetExpression(eltCursor, defs, expr))
			continue
		}

		//tuple & list
		isTuple := c.OpenBraces()
		isList := !isTuple && c.OpenBrackets()
		if isTuple || isList {
			var items []parsed.Expression

			if !isList || !c.CloseBrackets() {
				for {
					item, err := parseExpression(c, modName, generics)
					if err != nil {
						return nil, err
					}
					items = append(items, item)

					if c.Exact(",") {
						continue
					}
					if isTuple && c.CloseBraces() {
						break
					}
					if isList && c.CloseBrackets() {
						break
					}
					if isTuple {
						return nil, misc.NewError(*c, "expected `,` or `}` here")
					}
					if isList {
						return nil, misc.NewError(*c, "expected `,` or `]` here")
					}
				}
			}

			if isTuple {
				if len(items) < 2 {
					return nil, misc.NewError(*c, "tuple should contain at least 2 items")
				}
				exs = append(exs, parsed.NewTupleExpression(eltCursor, items))
			}
			if isList {
				exs = append(exs, parsed.NewListExpression(eltCursor, items))
			}
			continue
		}

		//infix
		if wp, infix, wa := c.InfixName(); infix != "" {
			exs = append(exs, parsed.NewInfixExpression(eltCursor, wp, infix, wa))
			continue
		}

		//identifier/call
		if id, spaceAfter := c.QualifiedIdentifier(); id != "" {
			var genericArgs parsed.GenericArgs
			if len(exs) == 0 {
				var err error
				if !spaceAfter {
					genericArgs, err = parseGenericArgs(c, modName, generics)
					if err != nil {
						return nil, err
					}
				}
			}
			exs = append(exs, parsed.NewIdentifierExpression(eltCursor, id, genericArgs))
			continue
		}

		break
	}

	if len(exs) == 1 {
		return exs[0], nil
	}
	if len(exs) > 1 {
		return parsed.NewChainExpression(startCursor, exs), nil
	}

	c.Pos = startCursor.Pos
	return nil, misc.NewError(*c, "failed to read expression here")
}

func parseDecons(
	c *misc.Cursor, modName parsed.ModuleFullName, generics parsed.GenericParams,
) (parsed.Decons, error) {
	startCursor := *c
	// any
	if c.Exact("_") {
		return finishParseDecons(c, modName, generics, parsed.NewAnyDecons(startCursor))
	}

	// const char
	if constChar := c.Char(); constChar != "" {
		return finishParseDecons(
			c, modName, generics, parsed.NewConstDecons(startCursor, parsed.ConstKindChar, constChar),
		)
	}

	if constNumber, integer := c.Number(); constNumber != "" {
		if integer {
			return finishParseDecons(
				c, modName, generics, parsed.NewConstDecons(startCursor, parsed.ConstKindInt, constNumber),
			)
		} else {
			return finishParseDecons(
				c, modName, generics, parsed.NewConstDecons(startCursor, parsed.ConstKindFloat, constNumber),
			)
		}
	}

	// const string
	if constString := c.String(); constString != "" {
		return finishParseDecons(
			c, modName, generics, parsed.NewConstDecons(startCursor, parsed.ConstKindString, constString),
		)
	}

	//tuple
	if c.OpenBraces() {
		var items []parsed.Decons
		for {
			item, err := parseDecons(c, modName, generics)
			if err != nil {
				return nil, err
			}
			items = append(items, item)

			if c.Exact(",") {
				continue
			}
			if c.CloseBraces() {
				break
			}
			return nil, misc.NewError(*c, "expected `,` or `}` here")
		}
		if len(items) < 2 {
			return nil, misc.NewError(*c, "decons typle should contain at least 2 elements")
		}

		return finishParseDecons(c, modName, generics, parsed.NewTupleDecons(startCursor, items))
	}

	//list
	if c.OpenBrackets() {
		var items []parsed.Decons
		if !c.CloseBrackets() {
			for {
				item, err := parseDecons(c, modName, generics)
				if err != nil {
					return nil, err
				}
				items = append(items, item)

				if c.Exact(",") {
					continue
				}
				if c.CloseBrackets() {
					break
				}
				return nil, misc.NewError(*c, "expected `,` or `]` here")
			}
		}

		return finishParseDecons(c, modName, generics, parsed.NewListDecons(startCursor, items))
	}

	// union
	if id, spaceAfter := c.QualifiedIdentifier(); id != "" {
		cs := *c
		if unicode.IsUpper([]rune(id)[0]) {
			var genericArgs parsed.GenericArgs
			if !spaceAfter {
				var err error
				genericArgs, err = parseGenericArgs(c, modName, generics)
				if err != nil {
					return nil, err
				}
			}

			pos := c.Pos
			var arg parsed.Decons
			if c.Exact("->") {
				c.Pos = pos
			} else {
				var err error
				arg, err = parseDecons(c, modName, generics)
				if err != nil {
					return nil, err
				}
			}
			if arg == nil {
				var err error
				arg, err = finishParseDecons(c, modName, generics, parsed.NewAnyDecons(cs))
				if err != nil {
					return nil, err
				}
			}
			return finishParseDecons(c, modName, generics, parsed.NewOptionDecons(cs, id, genericArgs, arg))
		} else {
			return finishParseDecons(c, modName, generics, parsed.NewNamedDecons(cs, id))
		}
	}

	return nil, misc.NewError(*c, "expected case value here")
}

func finishParseDecons(
	c *misc.Cursor, modName parsed.ModuleFullName, generics parsed.GenericParams, first parsed.Decons,
) (parsed.Decons, error) {
	cs := *c
	if c.Exact("::") {
		second, err := parseDecons(c, modName, generics)
		if err != nil {
			return nil, err
		}
		return parsed.NewConsDecons(cs, first, second), nil
	}
	if c.Exact(kwAs) {
		alias, _ := c.Identifier()
		if alias == "" {
			return nil, misc.NewError(*c, "expected alias here")
		}
		return first.SetAlias(alias)
	}
	return first, nil
}

func parseParameter(c *misc.Cursor, index int, optional bool) (parsed.Parameter, error) {
	startCursor := *c
	//tuple
	if c.OpenBraces() {
		var items []parsed.Parameter
		nestedIndex := 0
		for {
			item, err := parseParameter(c, nestedIndex, true)
			nestedIndex++
			if err != nil {
				return nil, err
			}
			if item == nil {
				if optional {
					*c = startCursor
					return nil, nil
				}
				return nil, misc.NewError(startCursor, "expected tuple deconstruction here")
			}
			items = append(items, item)
			if c.Exact(",") {
				continue
			}
			if c.CloseBraces() {
				break
			}
			if optional {
				*c = startCursor
				return nil, nil
			}
			return nil, misc.NewError(*c, "expected `,` or `}` here")
		}
		if len(items) < 2 {
			return nil, misc.NewError(startCursor, "tuple deconstruction should have at least 2 items")
		}
		return parsed.NewTupleParameter(startCursor, index, items), nil
	}

	//union
	if c.OpenParenthesis() {
		id, _ := c.QualifiedIdentifier()
		value, err := parseParameter(c, 0, true)
		if err != nil {
			return nil, err
		}
		if !c.CloseParenthesis() {
			if optional {
				*c = startCursor
				return nil, nil
			}
			return nil, misc.NewError(*c, "expected `)` here")
		}
		return parsed.NewOptionParameter(startCursor, index, id, value), nil
	}

	//omitted
	if c.Exact("_") {
		return parsed.NewOmittedParameter(startCursor), nil
	}

	//named
	id, _ := c.Identifier()
	if id != "" {
		if unicode.IsUpper([]rune(id)[0]) {
			if optional {
				*c = startCursor
				return nil, nil
			}
			return nil, misc.NewError(startCursor, "variable name should start with lowercase")
		}
		return parsed.NewNamedParameter(startCursor, id), nil
	}

	if optional {
		*c = startCursor
		return nil, nil
	}

	return nil, misc.NewError(startCursor, "expected parameter here")
}
