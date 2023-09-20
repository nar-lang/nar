package parser

import (
	"oak-compiler/pkg/a"
	"oak-compiler/pkg/ast"
	"oak-compiler/pkg/parsed"
	"slices"
	"strconv"
	"unicode"
)

const (
	KW_MODULE   = "module"
	kwImport   = "import"
	kwFrom     = "from"
	kwAs       = "as"
	kwExposing = "exposing"
	kwHidden   = "hidden"
	kwExtern   = "extern"
	kwAlias    = "alias"
	kwUnion    = "union"
	kwDef      = "def"
	kwConst    = "const"
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

func parseModule(fileName ModuleFileName, src ModuleSource, packageName ast.PackageFullName) (parsed.Module, error) {
	c := a.NewCursor(string(fileName), []rune(src))
	return parseModuleWithCursor(&c, packageName)
}

func parseModuleWithCursor(c *a.Cursor, packageName ast.PackageFullName) (parsed.Module, error) {
	name, err := parseModuleName(c)
	if err != nil {
		return parsed.Module{}, err
	}

	moduleName := parsed.NewModuleFullName(packageName, name)

	var imports []parsed.Import
	for {
		imp, ok, err := parseImport(c)
		if err != nil {
			return parsed.Module{}, err
		}
		if !ok {
			break
		}
		imports = append(imports, imp)
	}

	var definitions []parsed.Definition
	for !c.IsEof() {
		definition, err := parseDefinition(c, moduleName)
		if err != nil {
			return parsed.Module{}, err
		}
		definitions = append(definitions, definition)
	}

	return parsed.NewModule(name, imports, definitions), nil
}

func parseModuleName(c *a.Cursor) (string, error) {
	c.SkipComment()

	if !c.Exact(KW_MODULE) {
		return "", a.NewError(*c, "expected `module` keyword here")
	}
	name, _ := c.QualifiedIdentifier()
	if name == "" {
		return "", a.NewError(*c, "expected identifier here")
	}
	return name, nil
}

func parseImport(c *a.Cursor) (parsed.Import, bool, error) {
	startCursor := *c
	if !c.Exact(kwImport) {
		return parsed.Import{}, false, nil
	}
	module, _ := c.QualifiedIdentifier()
	alias := module
	if module == "" {
		return parsed.Import{}, false, a.NewError(*c, "expected module name here")
	}
	if c.Exact(kwAs) {
		alias, _ = c.Identifier()
		if alias == "" {
			return parsed.Import{}, false, a.NewError(*c, "expected module alias here")
		}
	}
	var package_ ast.PackageFullName
	if c.Exact(kwFrom) {
		package_ = ast.PackageFullName(c.String())
		if package_ == "" {
			return parsed.Import{}, false, a.NewError(*c, "expected package url here")
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
				return parsed.Import{}, false, a.NewError(*c, "expected `(` here")
			}

			for {
				id := c.InfixNameWithParenthesis()
				if id == "" {
					id, _ = c.Identifier()
				}
				if id == "" {
					return parsed.Import{}, false, a.NewError(*c, "expected identifier here")
				}
				exposing = append(exposing, id)
				if c.Exact(",") {
					continue
				}
				if c.CloseParenthesis() {
					break
				}
				return parsed.Import{}, false, a.NewError(*c, "expected `,` or `)` here")
			}
		}
	}

	return parsed.NewImport(startCursor, package_, module, alias, exposingAll, exposing), true, nil
}

func parseDefinition(c *a.Cursor, moduleName parsed.ModuleFullName) (parsed.Definition, error) {
	startCursor := *c
	kw := c.OneOf(kwUnion, kwDef, kwAlias, kwConst)
	if kw == "" {
		return nil, a.NewError(*c, "expected `union`, `alias`, `const` or `def` keyword here")
	}
	hidden := c.Exact(kwHidden)

	name := c.InfixNameWithParenthesis()
	infix := false

	if name != "" {
		infix = true
	} else {
		name, _ = c.Identifier()
	}

	if name == "" {
		return nil, a.NewError(*c, "expected definition name here")
	}
	if kw == kwUnion && !unicode.IsUpper([]rune(name)[0]) {
		return nil, a.NewError(*c, "definedType name should start with uppercase letter")
	}
	if kw == kwAlias && !unicode.IsUpper([]rune(name)[0]) {
		return nil, a.NewError(*c, "alias name should start with uppercase letter")
	}
	if kw == kwDef && unicode.IsUpper([]rune(name)[0]) {
		return nil, a.NewError(*c, "function name should start with lowercase letter")
	}
	if kw == kwConst && unicode.IsUpper([]rune(name)[0]) {
		return nil, a.NewError(*c, "const name should start with lowercase letter")
	}

	switch kw {
	case kwAlias:
		typeParams, err := parseTypeParameters(c)
		if err != nil {
			return nil, err
		}

		if !c.Exact("=") {
			return nil, a.NewError(*c, "expected `=` here")
		}

		var mbType a.Maybe[parsed.Type]
		extern := c.Exact(kwExtern)
		if !extern {
			tp, err := parseType(c, moduleName)
			if err != nil {
				return nil, err
			}
			mbType = a.Just(tp)
		}
		return parsed.NewAliasDefinition(startCursor, name, moduleName, typeParams, hidden, mbType), nil
	case kwUnion:
		typeParams, err := parseTypeParameters(c)
		if err != nil {
			return nil, err
		}

		if !c.Exact("=") {
			return nil, a.NewError(*c, "expected `=` here")
		}

		options, err := parseUnionOptions(c, moduleName)
		if err != nil {
			return nil, err
		}
		return parsed.NewUnionDefinition(startCursor, name, moduleName, typeParams, hidden, options), nil
	case kwDef:
		if infix {
			if !c.Exact(":") {
				return nil, a.NewError(*c, "expected `:` here")
			}

			if !c.OpenParenthesis() {
				return nil, a.NewError(*c, "expected `(` here")
			}

			var assoc ast.InfixAssociativity
			if c.Exact(kwLeft) {
				assoc = ast.InfixAssociativityLeft
			} else if c.Exact(kwRight) {
				assoc = ast.InfixAssociativityRight
			} else if c.Exact(kwNon) {
				assoc = ast.InfixAssociativityNon
			} else {
				return nil, a.NewError(*c, "expected `left`, `right` or `non` here")
			}

			sPriority, integer := c.Number()
			if sPriority == "" || !integer {
				return nil, a.NewError(*c, "expected priority value (integer) here")
			}

			priority, _ := strconv.ParseInt(sPriority, 10, 32)

			if !c.CloseParenthesis() {
				return nil, a.NewError(*c, "expected `)` here")
			}

			if !c.Exact("=") {
				return nil, a.NewError(*c, "expected `=` here")
			}

			alias, _ := c.Identifier()
			if alias == "" {
				return nil, a.NewError(*c, "expected infix function alias here")
			}

			return parsed.NewInfixDefinition(startCursor, name, moduleName, hidden, assoc, int(priority), alias), nil
		} else {
			mbType, params, err := parseFuncSignature(c, moduleName)
			if err != nil {
				return nil, err
			}
			if !c.Exact("=") {
				return nil, a.NewError(*c, "expected `=` here")
			}
			var expr parsed.Expression
			extern := c.Exact(kwExtern)
			if !extern {
				expr, err = parseExpression(c, moduleName)
				if err != nil {
					return nil, err
				}
			}
			return parsed.NewFuncDefinition(startCursor, name, hidden, extern, mbType, params, expr), nil
		}
	case kwConst:
		var mbType a.Maybe[parsed.Type]
		if c.Exact(":") {
			tp, err := parseType(c, moduleName)
			if err != nil {
				return nil, err
			}
			mbType = a.Just(tp)
		}
		if !c.Exact("=") {
			return nil, a.NewError(*c, "expected `=` here")
		}
		expr, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}

		return parsed.NewConstDefinition(startCursor, name, hidden, mbType, expr), nil
	default:
		return nil, a.NewError(startCursor, "impossible branch alias/union/const/def, this is a parser error")
	}
}

func parseTypeParameters(c *a.Cursor) ([]string, error) {
	if !c.OpenBrackets() {
		return nil, nil
	}

	var parameters []string

	for {
		eltStart := *c

		tp := parseTypeParameter(c)
		if tp == "" {
			return nil, a.NewError(eltStart, "expected definedType parameter here")
		}

		parameters = append(parameters, tp)

		if c.Exact(",") {
			continue
		}
		if c.CloseBrackets() {
			break
		}
		return nil, a.NewError(*c, "expected `,` or `]` here")
	}

	if len(parameters) == 0 {
		return nil, a.NewError(*c, "empty generic parameters list")
	}

	return parameters, nil
}

func parseUnionOptions(c *a.Cursor, moduleName parsed.ModuleFullName) ([]parsed.UnionOption, error) {
	var options []parsed.UnionOption
	for {
		hidden := c.Exact(kwHidden)

		optionStart := *c
		pos := c.Pos
		name, _ := c.Identifier()
		if name == "" {
			return nil, a.NewError(*c, "failed tor read union option, expected name here")
		}
		if !unicode.IsUpper([]rune(name)[0]) {
			c.Pos = pos
			return nil, a.NewError(*c, "option name should start with uppercase letter")
		}

		var types []parsed.Type

		if c.OpenParenthesis() {
			for {
				//optional name
				optStart := *c
				if id, _ := c.Identifier(); id != "" && !c.Exact(":") {
					c.Pos = optStart.Pos
				}

				type_, err := parseType(c, moduleName)
				if err != nil {
					return nil, err
				}
				types = append(types, type_)

				if c.Exact(",") {
					continue
				}
				if c.CloseParenthesis() {
					break
				}
				return nil, a.NewError(*c, "expected `,` or `)` here")
			}
		}

		options = append(options, parsed.NewUnionOption(
			optionStart, name, types, hidden,
		))

		if !c.Exact("|") {
			break
		}
	}
	return options, nil
}

func parseFuncSignature(
	c *a.Cursor, moduleName parsed.ModuleFullName,
) (a.Maybe[parsed.TypeSignature], []parsed.Pattern, error) {
	startCursor := *c
	if !c.OpenParenthesis() {
		return a.Maybe[parsed.TypeSignature]{}, nil, nil
	}

	var patterns []parsed.Pattern
	for {
		definedType, err := parsePattern(c, moduleName)
		if err != nil {
			return a.Maybe[parsed.TypeSignature]{}, nil, err
		}
		patterns = append(patterns, definedType)

		if c.Exact(",") {
			continue
		}
		if c.CloseParenthesis() {
			break
		}
		return a.Maybe[parsed.TypeSignature]{}, nil, a.NewError(*c, "expected `,` or `)` here")
	}

	hasType := slices.ContainsFunc(patterns, func(definedType parsed.Pattern) bool {
		_, hasType := definedType.GetType().Unwrap()
		return hasType
	})
	for _, t := range patterns {
		if _, ht := t.GetType().Unwrap(); ht != hasType {
			return a.Maybe[parsed.TypeSignature]{}, nil,
				a.NewError(*c, "expected all parameters are either typed or not")
		}
	}

	mbType := a.Nothing[parsed.TypeSignature]()
	if hasType {
		if !c.Exact(":") {
			return a.Maybe[parsed.TypeSignature]{}, nil, a.NewError(*c, "expected `:` here")
		}
		ret, err := parseType(c, moduleName)
		if err != nil {
			return a.Maybe[parsed.TypeSignature]{}, nil, err
		}

		var types []parsed.Type
		for _, definedType := range patterns {
			t, _ := definedType.GetType().Unwrap()
			types = append(types, t)
		}

		mbType = a.Just(parsed.NewSignatureType(startCursor, types, ret))
	} else if c.Exact(":") {
		return a.Maybe[parsed.TypeSignature]{}, nil,
			a.NewError(*c, "expected all parameters are typed if return definedType is provided")
	}

	return mbType, patterns, nil
}

func parsePattern(c *a.Cursor, moduleName parsed.ModuleFullName) (parsed.Pattern, error) {
	startCursor := *c

	//tuple/void
	if c.OpenParenthesis() {
		if c.CloseParenthesis() {
			return finishParsePattern(c, parsed.NewVoidPattern(startCursor), moduleName)
		}
		var items []parsed.Pattern
		for {
			item, err := parsePattern(c, moduleName)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			if c.Exact(",") {
				continue
			}
			if c.CloseParenthesis() {
				break
			}
			return nil, a.NewError(*c, "expected `,` or `)` here")
		}
		if len(items) < 2 {
			return nil, a.NewError(*c, "tuple definedType should have at least 2 items")
		}
		return finishParsePattern(c, parsed.NewTuplePattern(startCursor, items), moduleName)
	}

	//record
	if c.OpenBraces() {
		var names []string
		for {
			name, _ := c.Identifier()
			if name == "" {
				return nil, a.NewError(*c, "expected field name here")
			}
			names = append(names, name)

			if c.Exact(",") {
				continue
			}
			if c.CloseBrackets() {
				break
			}
			return nil, a.NewError(*c, "expected `,` or `)` here")
		}

		return finishParsePattern(c, parsed.NewRecordPattern(startCursor, names), moduleName)
	}

	//list
	if c.OpenBrackets() {
		if c.CloseBrackets() {
			return finishParsePattern(c, parsed.NewListPattern(startCursor, nil), moduleName)
		}

		var items []parsed.Pattern
		for {
			item, err := parsePattern(c, moduleName)
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
			return nil, a.NewError(*c, "expected `,` or `]` here")
		}

		return finishParsePattern(c, parsed.NewListPattern(startCursor, items), moduleName)
	}

	//var/union
	module, name, _ := c.QualifiedIdentifierSeparated()
	if name != "" {
		if unicode.IsUpper([]rune(name)[0]) {
			var items []parsed.Pattern
			if c.OpenParenthesis() {
				for {
					item, err := parsePattern(c, moduleName)
					if err != nil {
						return nil, err
					}
					items = append(items, item)
					if c.Exact(",") {
						continue
					}
					if c.CloseParenthesis() {
						break
					}
					return nil, a.NewError(*c, "expected `,` or `)` here")
				}
			}
			fullName := name
			if module != "" {
				fullName = module + "." + name
			}
			return finishParsePattern(c, parsed.NewConstructorPattern(startCursor, fullName, items), moduleName)
		} else {
			if module != "" {
				return nil, a.NewError(*c, "unexpected qualified identifier")
			}
			return finishParsePattern(c, parsed.NewNamedPattern(startCursor, name), moduleName)
		}
	}

	//anything
	if c.Exact("_") {
		return finishParsePattern(c, parsed.NewOmittedPattern(startCursor), moduleName)
	}

	if ch := c.Char(); ch != "" {
		return finishParsePattern(c, parsed.NewConstPattern(startCursor, parsed.ConstKindChar, ch), moduleName)
	}

	if str := c.String(); str != "" {
		return finishParsePattern(c, parsed.NewConstPattern(startCursor, parsed.ConstKindString, str), moduleName)
	}

	if i, isInt := c.Number(); i != "" && isInt {
		return finishParsePattern(c, parsed.NewConstPattern(startCursor, parsed.ConstKindInt, i), moduleName)
	}

	return nil, a.NewError(*c, "expected definedType here")
}

func finishParsePattern(c *a.Cursor, definedType parsed.Pattern, moduleName parsed.ModuleFullName) (parsed.Pattern, error) {
	startCursor := *c
	if c.Exact(kwAs) {
		id, _ := c.Identifier()
		if id == "" {
			return nil, a.NewError(*c, "expected identifier here")
		}
		return parsed.NewAliasPattern(startCursor, id, definedType), nil
	}

	if c.Exact("|") {
		tail, err := parsePattern(c, moduleName)
		if err != nil {
			return nil, err
		}
		return finishParsePattern(c, parsed.NewConsPattern(startCursor, definedType, tail), moduleName)
	}

	if c.Exact(":") {
		if _, hasType := definedType.GetType().Unwrap(); hasType {
			return nil, a.NewError(*c, "definedType has already got a definedType")
		}
		t, err := parseType(c, moduleName)
		if err != nil {
			return nil, err
		}
		return definedType.SetType(startCursor, t)
	}

	return definedType, nil
}

func parseType(c *a.Cursor, moduleName parsed.ModuleFullName) (parsed.Type, error) {
	maybeType, err := parseTypeOptional(c, moduleName)
	if err != nil {
		return nil, err
	}
	if t, ok := maybeType.Unwrap(); ok {
		return t, nil
	}
	return nil, a.NewError(*c, "expected definedType definition here")
}

func parseTypeOptional(c *a.Cursor, moduleName parsed.ModuleFullName) (a.Maybe[parsed.Type], error) {
	startCursor := *c

	//signature/tuple/void
	if c.OpenParenthesis() {
		if c.CloseParenthesis() {
			return a.Just(parsed.NewVoidType(startCursor)), nil
		}

		var items []parsed.Type
		for {
			type_, err := parseType(c, moduleName)
			if err != nil {
				return a.Maybe[parsed.Type]{}, err
			}
			items = append(items, type_)

			if c.Exact(",") {
				continue
			}
			if c.CloseParenthesis() {
				break
			}
			return a.Maybe[parsed.Type]{}, a.NewError(*c, "expected `,` or `)` here")
		}
		if !c.Exact("=>") {
			if len(items) < 2 {
				return a.Maybe[parsed.Type]{},
					a.NewError(*c, "tuple definedType should have at least 2 items")
			}
			return a.Just(parsed.NewTupleType(startCursor, items)), nil
		}
		retType, err := parseType(c, moduleName)
		if err != nil {
			return a.Maybe[parsed.Type]{}, err
		}
		return a.Just[parsed.Type](parsed.NewSignatureType(startCursor, items, retType)), nil
	}

	//record
	if c.OpenBraces() {
		recStart := *c
		ext := a.Nothing[string]()
		name, _ := c.QualifiedIdentifier()
		if name != "" && c.Exact("|") {
			ext = a.Just(name)
		} else {
			c.Pos = recStart.Pos
		}

		var fields []parsed.RecordField
		for {
			at := *c
			name, _ := c.Identifier()
			if name == "" {
				return a.Maybe[parsed.Type]{}, a.NewError(*c, "expected field name here")
			}
			if !c.Exact(":") {
				return a.Maybe[parsed.Type]{}, a.NewError(*c, "expected `:` here")
			}
			type_, err := parseType(c, moduleName)
			if err != nil {
				return a.Maybe[parsed.Type]{}, err
			}
			fields = append(fields, parsed.NewRecordField(at, name, type_))

			if c.Exact(",") {
				continue
			}
			if c.CloseBraces() {
				break
			}
			return a.Maybe[parsed.Type]{}, a.NewError(*c, "expected `,` or `}` here")
		}
		return a.Just(parsed.NewRecordType(startCursor, fields, ext)), nil
	}

	//definedType parameter
	if tp := parseTypeParameter(c); tp != "" {
		return a.Just(parsed.NewVariableType(startCursor, tp)), nil
	}

	//named
	module, name, _ := c.QualifiedIdentifierSeparated()
	if name != "" {
		if unicode.IsUpper([]rune(name)[0]) {
			var args []parsed.Type

			if c.OpenBrackets() {
				for {
					arg, err := parseType(c, moduleName)
					if err != nil {
						return a.Maybe[parsed.Type]{}, err
					}
					args = append(args, arg)

					if c.Exact(",") {
						continue
					}
					if c.CloseBrackets() {
						break
					}
					return a.Maybe[parsed.Type]{}, a.NewError(*c, "expected `,` or `)` here")
				}
			}

			fullName := name
			if module != "" {
				fullName = module + "." + name
			}
			return a.Just(parsed.NewNamedType(startCursor, fullName, args, moduleName)), nil
		}
		return a.Maybe[parsed.Type]{}, a.NewError(*c, "definedType name should be uppercase")
	}

	c.Pos = startCursor.Pos
	return a.Nothing[parsed.Type](), nil
}

func parseTypeParameter(c *a.Cursor) string {
	startCursor := *c
	if c.Exact("?") {
		name, _ := c.Identifier()
		if name != "" {
			c.Pos = startCursor.Pos
			if c.Exact("?" + name) {
				return "?" + name
			}
		}
		c.Pos = startCursor.Pos
	}

	return ""
}

func parseExpression(c *a.Cursor, moduleName parsed.ModuleFullName) (parsed.Expression, error) {
	startCursor := *c

	// const char
	if constChar := c.Char(); constChar != "" {
		return finishParseExpression(
			c, parsed.NewConstExpression(startCursor, parsed.ConstKindChar, constChar), moduleName,
		)
	}

	// const int / float
	if constNumber, integer := c.Number(); constNumber != "" {
		if integer {
			return finishParseExpression(
				c, parsed.NewConstExpression(startCursor, parsed.ConstKindInt, constNumber), moduleName,
			)
		} else {
			return finishParseExpression(
				c, parsed.NewConstExpression(startCursor, parsed.ConstKindFloat, constNumber), moduleName,
			)
		}
	}

	// const string
	if constString := c.String(); constString != "" {
		return finishParseExpression(
			c, parsed.NewConstExpression(startCursor, parsed.ConstKindString, constString), moduleName,
		)
	}

	if c.OpenBrackets() {
		var items []parsed.Expression
		if !c.CloseBrackets() {
			for {
				item, err := parseExpression(c, moduleName)
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
				return nil, a.NewError(*c, "expected `,` or `]` here")
			}
		}
		return finishParseExpression(c, parsed.NewListExpression(startCursor, items), moduleName)
	}

	//infix value
	if op := c.InfixNameWithParenthesis(); op != "" {
		return finishParseExpression(c, parsed.NewInfixExpression(startCursor, op, moduleName), moduleName)
	}

	//negate
	if c.Exact("-") {
		nested, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		return finishParseExpression(c, parsed.NewNegateExpression(startCursor, nested), moduleName)
	}

	//lambda
	if c.Exact("\\(") {
		c.Pos = startCursor.Pos
		c.Exact("\\")
		_, params, err := parseFuncSignature(c, moduleName)
		if err != nil {
			return nil, err
		}
		if !c.Exact("->") {
			return nil, a.NewError(*c, "expected -> here")
		}
		body, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		return finishParseExpression(c, parsed.NewLambdaExpression(startCursor, params, body), moduleName)
	}

	// if
	if c.Exact(kwIf) {
		condition, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		if !c.Exact(kwThen) {
			return nil, a.NewError(*c, "expected `then` here")
		}
		positive, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		if !c.Exact(kwElse) {
			return nil, a.NewError(*c, "expected `else` here")
		}
		negative, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		return finishParseExpression(c, parsed.NewIfExpression(startCursor, condition, positive, negative), moduleName)
	}

	//let
	if c.Exact(kwLet) {
		var defs []parsed.LetDefinition
		for {
			defStart := *c
			id, _ := c.Identifier()
			typeSignature, patterns, err := parseFuncSignature(c, moduleName)
			if len(patterns) > 0 && id != "" {
				var type_ a.Maybe[parsed.Type]
				if t, ok := typeSignature.Unwrap(); ok {
					type_ = a.Just[parsed.Type](t)
				}
				if err != nil {
					return nil, err
				}
				if _, ok := type_.Unwrap(); !ok && c.Exact(":") {
					type_, err = parseTypeOptional(c, moduleName)
					if _, ok := type_.Unwrap(); !ok {
						return nil, a.NewError(*c, "expected definedType here")
					}
				}
				if !c.Exact("=") {
					return nil, a.NewError(*c, "expected `=` here")
				}
				expr, err := parseExpression(c, moduleName)
				if err != nil {
					return nil, err
				}
				defs = append(defs, parsed.NewLetDefine(id, patterns, typeSignature, expr))
			} else {
				c.Pos = defStart.Pos
				definedType, err := parsePattern(c, moduleName)
				if err != nil {
					return nil, err
				}
				if !c.Exact("=") {
					return nil, a.NewError(*c, "expected `=` here")
				}
				expr, err := parseExpression(c, moduleName)
				if err != nil {
					return nil, err
				}
				defs = append(defs, parsed.NewLetDestruct(definedType, expr))
			}

			if c.Exact(kwLet) {
				continue
			}
			if c.Exact(kwIn) {
				break
			}
			return nil, a.NewError(*c, "expected `let` or `in` here")
		}

		expr, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}

		return finishParseExpression(c, parsed.NewLetExpression(startCursor, defs, expr), moduleName)
	}

	//select
	if c.Exact(kwSelect) {
		condition, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		var cases []parsed.ExpressionSelectCase
		for {
			at := *c
			if !c.Exact(kwCase) {
				break
			}
			definedType, err := parsePattern(c, moduleName)
			if err != nil {
				return nil, err
			}

			if !c.Exact("->") {
				return nil, a.NewError(*c, "expected `->` here")
			}
			expr, err := parseExpression(c, moduleName)
			if err != nil {
				return nil, err
			}
			cases = append(cases, parsed.NewSelectExpressionCase(at, definedType, expr))
		}
		if len(cases) == 0 {
			return nil, a.NewError(*c, "expected at least one case here")
		}
		return finishParseExpression(c, parsed.NewSelectExpression(startCursor, condition, cases), moduleName)
	}

	//accessor
	if c.Exact(".") {
		id, _ := c.Identifier()
		if id == "" {
			return nil, a.NewError(*c, "expected accessor name here")
		}
		return parsed.NewAccessorExpression(startCursor, id), nil
	}

	//record / update
	if c.OpenBraces() {
		recStart := *c
		update := a.Nothing[string]()
		name, _ := c.QualifiedIdentifier()
		if name != "" && c.Exact("|") {
			update = a.Just(name)
		} else {
			c.Pos = recStart.Pos
		}

		var fields []parsed.ExpressionRecordField
		for {
			at := *c
			name, _ := c.Identifier()
			if name == "" {
				return nil, a.NewError(*c, "expected field name here")
			}
			if !c.Exact("=") {
				return nil, a.NewError(*c, "expected `:` here")
			}
			expr, err := parseExpression(c, moduleName)
			if err != nil {
				return nil, err
			}
			fields = append(fields, parsed.NewRecordExpressionField(at, name, expr))

			if c.Exact(",") {
				continue
			}
			if c.CloseBraces() {
				break
			}
			return nil, a.NewError(*c, "expected `,` or `}` here")
		}
		if upd, ok := update.Unwrap(); ok {
			return finishParseExpression(
				c, parsed.NewUpdateExpression(startCursor, upd, moduleName, fields), moduleName,
			)
		} else {
			return finishParseExpression(c, parsed.NewRecordExpression(startCursor, fields), moduleName)
		}
	}

	//tuple / void / precedence
	if c.OpenParenthesis() {
		if c.CloseParenthesis() {
			return finishParseExpression(c, parsed.NewVoidExpression(startCursor), moduleName)
		}

		var items []parsed.Expression
		for {
			item, err := parseExpression(c, moduleName)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			if c.Exact(",") {
				continue
			}
			if c.CloseParenthesis() {
				break
			}
			return nil, a.NewError(*c, "expected `,` or `)` here")
		}

		if len(items) == 1 { //wrapped in brackets to increase precedence
			return finishParseExpression(c, items[0], moduleName)
		}

		return finishParseExpression(c, parsed.NewTupleExpression(startCursor, items), moduleName)
	}

	//var
	name, _ := c.QualifiedIdentifier()
	if name != "" {
		return finishParseExpression(c, parsed.NewVarExpression(startCursor, name, moduleName), moduleName)
	}

	return nil, a.NewError(*c, "expected expression here")
}

func finishParseExpression(
	c *a.Cursor, expr parsed.Expression, moduleName parsed.ModuleFullName,
) (parsed.Expression, error) {
	startCursor := *c
	if _, infix, _ := c.InfixName(); infix != "" {
		final, err := parseExpression(c, moduleName)
		if err != nil {
			return nil, err
		}
		return parsed.NewBinOpExpression(startCursor, infix, moduleName, expr, final), nil
	}

	if c.OpenParenthesis() {
		var args []parsed.Expression
		for {
			arg, err := parseExpression(c, moduleName)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if c.Exact(",") {
				continue
			}
			if c.CloseParenthesis() {
				break
			}
			return nil, a.NewError(*c, "expected `,` or `)` here")
		}
		return finishParseExpression(c, parsed.NewCallExpression(startCursor, expr, args), moduleName)
	}

	if c.Exact(".") {
		id, _ := c.Identifier()
		if id == "" {
			return nil, a.NewError(*c, "expected field name here")
		}
		return parsed.NewAccessExpression(startCursor, expr, id), nil
	}

	return expr, nil
}
