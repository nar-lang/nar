package parsed

import (
	"fmt"
	"oak-compiler/pkg/misc"
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

func ParseModule(c *misc.Cursor, packageName PackageFullName) (Module, error) {
	moduleDefinition, err := parseModuleStatement(c)
	if err != nil {
		return Module{}, err
	}

	var imports []StatementImport
	for {
		imp, ok, err := parseImportStatement(c)
		if err != nil {
			return Module{}, err
		}
		if !ok {
			break
		}
		imports = append(imports, imp)
	}

	var definitions []Definition
	var order []string
	for !c.IsEof() {
		definition, err := parseDefinition(
			c, ModuleFullName{packageName: packageName, moduleName: moduleDefinition.Name()},
		)
		if err != nil {
			return Module{}, err
		}
		definitions = append(definitions, definition)
		order = append(order, definition.Name())
	}

	return NewModule(moduleDefinition, imports, definitions), nil
}

func parseModuleStatement(c *misc.Cursor) (StatementModule, error) {
	c.SkipComment()

	if !c.Exact(kwModule) {
		return StatementModule{}, misc.NewError(*c, "expected `module` keyword here")
	}
	name := c.QualifiedIdentifier()
	if name == "" {
		return StatementModule{}, misc.NewError(*c, "expected identifier here")
	}
	return NewModuleStatement(name), nil
}

func parseImportStatement(c *misc.Cursor) (StatementImport, bool, error) {
	startCursor := *c
	if !c.Exact(kwImport) {
		return StatementImport{}, false, nil
	}
	module := c.QualifiedIdentifier()
	alias := module
	if module == "" {
		return StatementImport{}, false, misc.NewError(*c, "expected module name here")
	}
	if c.Exact(kwAs) {
		alias = c.Identifier()
		if alias == "" {
			return StatementImport{}, false, misc.NewError(*c, "expected module alias here")
		}
	}
	var package_ PackageFullName
	if c.Exact(kwFrom) {
		package_ = PackageFullName(c.String())
		if package_ == "" {
			return StatementImport{}, false, misc.NewError(*c, "expected package url here")
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
				return StatementImport{}, false, misc.NewError(*c, "expected `(` here")
			}

			for {
				id := c.InfixNameWithParenthesis()
				if id == "" {
					id = c.Identifier()
				}
				if id == "" {
					return StatementImport{}, false, misc.NewError(*c, "expected identifier here")
				}
				exposing = append(exposing, id)
				if c.Exact(",") {
					continue
				}
				if c.CloseParenthesis() {
					break
				}
				return StatementImport{}, false, misc.NewError(*c, "expected `,` or `)` here")
			}
		}
	}

	return NewImportStatement(startCursor, package_, module, alias, exposingAll, exposing), true, nil
}

func parseDefinition(c *misc.Cursor, modName ModuleFullName) (Definition, error) {
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
		name = c.Identifier()
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

		var type_ Type
		extern := c.Exact(kwExtern)
		if extern {
			type_ = NewAddressedType(startCursor, modName, NewDefinitionAddress(modName, name), nil, false)
		} else {
			type_, err = parseType(c, modName, true, false, true, gps)
			if err != nil {
				return nil, err
			}
		}
		return NewTypeDefinition(
			startCursor, NewDefinitionAddress(modName, name), gps, hidden, extern, type_,
		), nil

	case kwDef:
		if !c.Exact(":") {
			return nil, misc.NewError(*c, "expected `:` here")
		}
		if infix {
			if !c.OpenParenthesis() {
				return nil, misc.NewError(*c, "expected `(` here")
			}

			var assoc InfixAssociativity
			if c.Exact(kwLeft) {
				assoc = InfixAssociativityLeft
			} else if c.Exact(kwRight) {
				assoc = InfixAssociativityRight
			} else if c.Exact(kwNon) {
				assoc = InfixAssociativityNon
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

			alias := c.Identifier()
			if alias == "" {
				return nil, misc.NewError(*c, "expected infix function alias here")
			}

			return NewInfixDefinition(
				startCursor, NewDefinitionAddress(modName, name), hidden, assoc, int(priority), alias,
			), nil
		} else {
			type_, err := parseType(c, modName, false, false, false, gps)
			if err != nil {
				return nil, err
			}
			if !c.Exact("=") {
				return nil, misc.NewError(*c, "expected `=` here")
			}
			var ex Expression
			extern := c.Exact(kwExtern)
			if !extern {
				ex, err = parseExpression(c, modName, gps)
				if err != nil {
					return nil, err
				}
			}
			return NewFuncDefinition(
				startCursor, NewDefinitionAddress(modName, name), gps, hidden, extern, type_, ex,
			), nil
		}
	default:
		return nil, misc.NewError(startCursor, "impossible branch type/def, this is a compiler error")
	}
}

func parseGenericParameters(c *misc.Cursor, modName ModuleFullName) (GenericParams, error) {
	if !c.OpenBrackets() {
		return nil, nil
	}

	var parameters GenericParams

	for {
		eltStart := *c
		name := c.Identifier()
		if !unicode.IsLower([]rune(name)[0]) {
			return nil, misc.NewError(*c, "generic parameter name shold start with lowercase letter")
		}

		constraint := GenericConstraint(GenericConstraintAny{})
		if c.Exact(":") {
			var gcs []GenericConstraint
			for {
				cs := c.Identifier()
				if cs == "" {
					return nil, misc.NewError(*c, "expected generic parameter constraint here")
				}
				switch cs {
				case "any":
					constraint = GenericConstraintAny{}
					break
				case "comparable":
					constraint = GenericConstraintComparable{}
					break
				case "equatable":
					constraint = GenericConstraintEquatable{}
					break
				default:
					constraint = GenericConstraintType{Name: cs} //TODO: generic args
				}
				gcs = append(gcs, constraint)
				if c.Exact("+") {
					continue
				}
				break
			}
			if len(gcs) > 1 {
				constraint = GenericConstraintCombined{Constraints: gcs}
			}
		}

		parameters = append(parameters, NewGenericParam(eltStart, modName, name, constraint))

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

func parseType(
	c *misc.Cursor,
	modName ModuleFullName,
	definition bool,
	optional bool,
	allowSignature bool,
	genericParams GenericParams,
) (Type, error) {
	startCursor := *c

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

	if c.OpenBraces() {
		start := c.Pos
		c.Identifier()
		if c.Exact(":") {
			var fields []RecordField
			c.Pos = start
			for {
				fieldStart := *c
				name := c.Identifier()
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
				fields = append(fields, NewRecordField(fieldStart, name, type_))
				if c.Exact(",") {
					continue
				}
				if c.CloseBraces() {
					break
				}
				return nil, misc.NewError(*c, "failed to read record, expected `,` or `}` here")
			}
			return NewRecordType(startCursor, modName, fields), nil
		} else {
			var items []Type
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

			return NewTupleType(startCursor, modName, items), nil
		}
	}

	if definition {
		var options []UnionOption
		for {
			if !c.Exact("|") {
				break
			}

			optionStart := *c
			pos := c.Pos
			name := c.Identifier()
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
				type_ = NewVoidType(*c, modName)
			}
			options = append(options, NewUnionOption(optionStart, name, type_))
		}
		if len(options) > 0 {
			return NewUnionType(startCursor, modName, options), nil
		}
	}

	if allowSignature {
		paramName := c.Identifier()
		var paramType Type
		noType := false
		if c.Exact(":") {
			var err error
			paramType, err = parseType(c, modName, false, false, false, genericParams)
			if err != nil {
				return nil, err
			}
		} else {
			c.Pos = startCursor.Pos
			var err error
			paramType, err = parseType(c, modName, false, true, false, genericParams)
			noType = err == errNotAType
		}

		if !noType {
			if c.Exact("->") {
				returnType, err := parseType(c, modName, false, false, true, genericParams)
				if err != nil {
					return nil, err
				}
				return NewSignatureType(startCursor, modName, paramType, returnType, paramName, nil), nil
			} else {
				c.Pos = startCursor.Pos
			}
		}
	}

	for _, g := range genericParams {
		cs := *c
		if c.Exact(g.name) {
			return NewGenericNameType(cs, modName, g.name), nil
		}
	}

	typeName := c.QualifiedIdentifier()
	if typeName != "" {
		if !unicode.IsUpper([]rune(typeName)[0]) {
			if optional {
				c.Pos = startCursor.Pos
				return nil, nil
			}
			c.Pos = startCursor.Pos
			return nil, misc.NewError(*c, "expected type name starting with uppercase letter here,this one looks like not declared generic parameter")
		}
		genericArgs, err := parseGenericArgs(c, modName, genericParams)
		if err != nil {
			return nil, err
		}
		return NewNamedType(startCursor, modName, typeName, genericArgs), nil
	}
	c.Pos = startCursor.Pos

	if !optional {
		return nil, misc.NewError(*c, "expected type declaration here")
	}

	return nil, errNotAType
}

func parseGenericArgs(c *misc.Cursor, modName ModuleFullName, generics GenericParams) (GenericArgs, error) {
	var genericArgs GenericArgs
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

func parseExpression(c *misc.Cursor, modName ModuleFullName, generics GenericParams) (Expression, error) {
	startCursor := *c
	var exs []Expression

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
			exs = append(exs, nested)
			continue
		}

		// const char
		if constChar := c.Char(); constChar != "" {
			exs = append(exs, NewConstExpression(eltCursor, ConstKindChar, constChar))
			continue
		}

		// const int / float
		if constNumber, integer := c.Number(); constNumber != "" {
			if integer {
				exs = append(exs, NewConstExpression(eltCursor, ConstKindInt, constNumber))
			} else {
				exs = append(exs, NewConstExpression(eltCursor, ConstKindFloat, constNumber))
			}
			continue
		}

		// const string
		if constString := c.String(); constString != "" {
			exs = append(exs, NewConstExpression(eltCursor, ConstKindString, constString))
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
			exs = append(exs, NewIfExpression(eltCursor, condition, positive, negative))
			continue
		}

		// select
		if c.Exact(kwSelect) {
			condition, err := parseExpression(c, modName, generics)
			if err != nil {
				return nil, err
			}
			var cases []ExpressionSelectCase
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
				cases = append(cases, ExpressionSelectCase{Decons: ds, Expression: ex})
			}
			if len(cases) == 0 {
				return nil, misc.NewError(*c, "expected at least one case here")
			}
			exs = append(exs, NewSelectExpression(eltCursor, condition, cases))
			continue
		}

		//let
		if c.Exact(kwLet) {
			var defs []LetDefinition
			for {
				var name string
				if c.Exact("_") {
					name = "_"
				} else {
					name = c.Identifier()
				}
				if name == "" {
					return nil, misc.NewError(*c, "expected name here")
				}
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

				defs = append(defs, NewLetDefinition(name, protoType, expr))

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

			exs = append(exs, NewLetExpression(eltCursor, defs, expr))
			continue
		}

		//tuple
		if c.OpenBraces() {
			var items []Expression
			for {
				item, err := parseExpression(c, modName, generics)
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
				return nil, misc.NewError(*c, "tuple should contain at least 2 items")
			}
			exs = append(exs, NewTupleExpression(eltCursor, items))
			continue
		}

		//infix
		if infix := c.InfixName(); infix != "" {
			exs = append(exs, NewInfixExpression(eltCursor, infix))
			continue
		}

		//identifier/call
		if id := c.QualifiedIdentifier(); id != "" {
			var genericArgs GenericArgs
			if len(exs) == 0 {
				var err error
				genericArgs, err = parseGenericArgs(c, modName, generics)
				if err != nil {
					return nil, err
				}
			}
			exs = append(exs, NewIdentifierExpression(eltCursor, id, genericArgs))
			continue
		}

		break
	}

	if len(exs) == 1 {
		return exs[0], nil
	}
	if len(exs) > 1 {
		return NewChainExpression(startCursor, exs), nil
	}

	c.Pos = startCursor.Pos
	return nil, misc.NewError(*c, "failed to read expression here")
}

func parseDecons(c *misc.Cursor, modName ModuleFullName, generics GenericParams) (Decons, error) {
	startCursor := *c
	// any
	if c.Exact("_") {
		return NewAnyDecons(), nil
	}

	// const char
	if constChar := c.Char(); constChar != "" {
		return NewConstDecons(ConstKindChar, constChar), nil
	}

	if constNumber, integer := c.Number(); constNumber != "" {
		if integer {
			return NewConstDecons(ConstKindInt, constNumber), nil
		} else {
			return NewConstDecons(ConstKindFloat, constNumber), nil
		}
	}

	// const string
	if constString := c.String(); constString != "" {
		return NewConstDecons(ConstKindString, constString), nil
	}

	//tuple
	if c.OpenBraces() {
		var items []Decons
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

		return NewTupleDecons(startCursor, items, ""), nil //TODO: alias
	}

	// union
	if id := c.QualifiedIdentifier(); id != "" {
		cs := *c
		if unicode.IsUpper([]rune(id)[0]) {
			genericArgs, err := parseGenericArgs(c, modName, generics)
			if err != nil {
				return nil, err
			}

			pos := c.Pos
			var arg Decons
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
				arg = NewAnyDecons()
			}
			return NewOptionDecons(cs, id, genericArgs, arg, ""), nil //TODO: alias
		} else {
			return NewNamedDecons(cs, id), nil
		}
	}

	return nil, misc.NewError(*c, "expected case value here")
}
