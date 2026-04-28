// Package parser provides lexical analysis and parsing for Blueprint build definitions.
// Parser subpackage - Blueprint recursive descent parser.
//
// This package implements the second stage of the Blueprint build system:
// it takes a stream of tokens from the lexer and produces an Abstract Syntax Tree (AST).
// The parser uses a recursive descent approach, building parse trees top-down
// by following the grammar rules defined for Blueprint source files.
//
// The parser handles:
//   - Module definitions: cc_binary { ... }, cc_library { ... }, etc.
//   - Variable assignments: my_var = "value", my_list += ["item"]
//   - Expressions: strings, integers, booleans, lists, maps
//   - select() conditional expressions for architecture-specific values
//   - Property overrides: arch: {...}, host: {...}, target: {...}, multilib: {...}
//
// Grammar overview:
//
//	File        -> Definition*
//	Definition  -> Module | Assignment
//	Module      -> IDENT LBRACE PropertyList RBRACE
//	Assignment  -> IDENT (ASSIGN | PLUSEQ) Expression
//	Expression  -> Primary (PLUS Primary)*
//	Primary     -> STRING | INT | BOOL | LIST | MAP | IDENT | select()
//
// Error handling:
//
//	Parse errors are collected and aggregated rather than failing immediately.
//	This allows users to fix multiple issues in a single pass.
//	The parser uses error recovery to skip to the next definition after an error.
//
// The parser is the second stage in the Blueprint pipeline, consuming tokens
// from the lexer and producing AST nodes that represent the syntactic structure
// of the Blueprint source code.
package parser

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/scanner"

	"minibp/lib/errors"
)

// Parser parses Blueprint files.
// It uses a recursive descent parsing approach, consuming tokens from the lexer
// and building an AST (Abstract Syntax Tree) representation of the Blueprint code.
//
// The parser handles:
//   - Modules: type { property_list }
//   - Assignments: name = value or name += value
//   - Expressions: literals, variables, operators, select()
//   - Property overrides: arch:, host:, target:, multilib:, override:
//
// The parser maintains lookahead tokens (curToken and peekToken) to enable
// grammar decisions that require looking ahead more than one token.
// It also provides error recovery by skipping to the next definition when
// a parse error occurs, allowing multiple errors to be reported in a single pass.
//
// Token management:
//   - curToken: The current token being processed
//   - peekToken: The next token (lookahead) for grammar look-ahead
//   - nextToken(): Advances both curToken and peekToken forward
//
// Error handling:
//   - Errors are collected in the errors slice rather than failing immediately
//   - skipToNextDefinition() provides error recovery
//   - All errors are returned with source position information
type Parser struct {
	lexer     *Lexer  // The lexer used to tokenize the input
	curToken  Token   // The current token being processed
	peekToken Token   // The next token (lookahead) for grammar look-ahead
	fileName  string  // Name of the file being parsed (for error reporting)
	source    string  // Source content for error line display
	errors    []error // List of parsing errors encountered
}

// NewParser creates a new Parser from an io.Reader.
// It initializes the parser with a new lexer for the given input source
// and advances past the first two tokens to set up curToken and peekToken.
//
// This two-token initialization is required because the recursive descent
// parser often needs to look ahead one token to make grammar decisions.
// For example, when parsing an identifier, the parser needs to know
// whether the next token is LBRACE (module) or ASSIGN (assignment).
//
// Setup process:
//  1. Create lexer with the input reader and filename
//  2. Call nextToken() twice to fill curToken and peekToken
//  3. Parser is now ready to parse
//
// Parameters:
//   - r: The input io.Reader containing Blueprint source code
//   - fileName: The name of the file being parsed (for error messages)
//
// Returns:
//   - A new Parser instance ready to parse the input
func NewParser(r io.Reader, fileName string, source ...string) *Parser {
	src := ""
	if len(source) > 0 {
		src = source[0]
	}
	p := &Parser{
		lexer:    NewLexer(r, fileName),
		fileName: fileName,
		source:   src,
		errors:   []error{},
	}
	// Initialize curToken and peekToken by advancing twice
	// This sets up the initial state for the recursive descent parser
	p.nextToken()
	p.nextToken()
	return p
}

// nextToken advances the parser to the next token in the input stream.
// It performs a "shift" operation: curToken becomes the previous token,
// peekToken becomes the current token, and a new peekToken is fetched
// from the lexer.
//
// This is the fundamental token advancement mechanism for the recursive
// descent parser. Each call to nextToken() consumes one token and makes
// the next token available for inspection via peekToken.
//
// Token flow:
//
//	Before: curToken=A, peekToken=B, lexer.position=C
//	After:  curToken=B, peekToken=C, lexer.position=D
func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.lexer.NextToken()
}

// expect checks if the current token matches the expected type.
// If it matches, it consumes the token (via nextToken()) and returns the matched token.
// If it doesn't match, it returns an error with the position and expected vs actual token types.
//
// This method is used when the grammar requires a specific token type.
// For example, after parsing a property name, the parser expects a COLON token.
//
// Error message includes:
//   - Source position of the current token
//   - Expected token type
//   - Actual token type found
//
// Parameters:
//   - t: The expected TokenType (e.g., LBRACE, ASSIGN, COLON)
//
// Returns:
//   - Token: The matched token if successful
//   - error: nil if successful, otherwise an error describing the mismatch
func (p *Parser) expect(t TokenType) (Token, error) {
	if p.curToken.Type == t {
		tok := p.curToken
		p.nextToken()
		return tok, nil
	}
	err := errors.Syntax(fmt.Sprintf("expected %s, got %s", t, p.curToken.Type)).
		WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column)
	return Token{}, err
}

// expectPeek checks if the peek token (lookahead) matches the expected type.
// Unlike expect(), this checks the lookahead token without consuming it.
// The token is consumed only if it matches.
//
// This is used when we need to look ahead to make a parsing decision,
// but don't want to commit to consuming the token yet.
// For example, to decide if something is a module vs assignment,
// we need to look at peekToken without consuming it.
//
// Parameters:
//   - t: The expected TokenType for the peek token
//
// Returns:
//   - bool: true if the peek token matched and was consumed, false otherwise
func (p *Parser) expectPeek(t TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	return false
}

// Parse parses the entire input and returns a File AST node.
// It repeatedly parses definitions until the end of file is reached.
// After parsing all definitions, it collects any lexer errors to ensure
// all issues are reported to the caller.
//
// Parse flow:
//  1. Create an empty File node with the filename
//  2. Loop: parseDefinition() until EOF token
//  3. Collect errors from parser and lexer
//  4. Return File and errors
//
// Error handling:
//   - Parse errors are collected rather than failing immediately
//   - Error recovery via skipToNextDefinition() continues after errors
//   - Lexer errors are included in the final error list
//
// Returns:
//   - *File: The parsed AST representation of the Blueprint file
//   - []error: A list of errors encountered during parsing (empty if successful)
func (p *Parser) Parse() (*File, []error) {
	file := &File{Name: p.fileName}

	// Parse definitions until EOF
	// Each definition is either a module or an assignment
	for p.curToken.Type != EOF {
		def, err := p.parseDefinition()
		if err != nil {
			// Collect error but continue parsing to find more issues
			p.errors = append(p.errors, err)
			p.skipToNextDefinition()
		} else if def != nil {
			// Add successfully parsed definition to file
			file.Defs = append(file.Defs, def)
		}
	}

	// Include lexer errors in the final error list
	// This captures issues like invalid characters detected during scanning
	if len(p.errors) == 0 {
		p.errors = append(p.errors, p.lexer.Errors()...)
	}

	return file, p.errors
}

// skipToNextDefinition skips tokens until we reach a potential start of a definition.
// This is used for error recovery - when a parse error occurs, we skip forward
// to try to continue parsing subsequent definitions rather than stopping entirely.
//
// It skips tokens until it finds an IDENT token (which could be a module type
// or variable name) or reaches EOF. This allows the parser to
// recover from syntax errors and continue processing the rest of the file.
//
// Example error recovery:
//
//	my_module { srcs: ["file.c", }
//	         ^ parse error here
//	another_module { }  <- skipToNextDefinition skips to here
func (p *Parser) skipToNextDefinition() {
	for p.curToken.Type != EOF && p.curToken.Type != IDENT {
		p.nextToken()
	}
}

// parseDefinition parses either a module or an assignment.
// A definition starts with an identifier (module type or variable name).
// After the identifier:
//   - If followed by LBRACE ({), it's a module definition
//   - If followed by ASSIGN (=) or PLUSEQ (+=), it's an assignment
//
// Grammar:
//
//	Definition -> IDENT (LBRACE Module | (ASSIGN | PLUSEQ) Assignment)
//
// Token flow:
//  1. Verify current token is IDENT
//  2. Record name and position
//  3. Advance to next token
//  4. Check token to decide definition type
//
// Returns:
//   - Definition: A Module or Assignment AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseDefinition() (Definition, error) {
	if p.curToken.Type != IDENT {
		return nil, errors.Syntax(fmt.Sprintf("expected identifier, got %s", p.curToken.Type)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(len(p.curToken.Literal)).
			WithSuggestion("Module or variable name must be an unquoted identifier")
	}

	// Record the name and its position for error reporting
	name := p.curToken.Literal
	namePos := p.curToken.Pos

	p.nextToken()

	// Decide what kind of definition based on the token after the name
	switch p.curToken.Type {
	case LBRACE:
		// Module definition: name { ... }
		// Examples: cc_binary { ... }, cc_library { ... }
		return p.parseModule(name, namePos)
	case ASSIGN, PLUSEQ:
		// Assignment: name = value or name += value
		// Examples: my_var = "value", my_list += ["item"]
		return p.parseAssignment(name, namePos)
	default:
		return nil, errors.Syntax(fmt.Sprintf("unexpected token %s after identifier '%s'", p.curToken.Type, name)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column)
	}
}

// parseModule parses a module definition: type { property_list }
// A module consists of a type name (like "cc_binary", "cc_library") followed by
// a block of properties enclosed in braces. Special properties "arch", "host",
// "target", and "multilib" are extracted as architecture/target-specific overrides
// that apply to different build configurations.
//
// Parameters:
//   - typeName: The module type name (e.g., "cc_binary", "cc_library")
//   - typePos: The source position of the type name
//
// Returns:
//   - *Module: The parsed module AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseModule(typeName string, typePos scanner.Position) (*Module, error) {
	// Current token is LBRACE - consume the opening brace
	lbracePos := p.curToken.Pos
	p.nextToken()

	// Parse the property list inside the braces
	propertyList, rbracePos, err := p.parsePropertyList()
	if err != nil {
		return nil, err
	}

	// Extract arch, host, target, and multilib overrides from properties.
	// These special properties are removed from the main property list
	// and stored separately for variant matching during build.
	archProps := make(map[string]*Map)
	var hostProps *Map
	var targetProps *Map
	multilibProps := make(map[string]*Map)
	var overrideFound bool
	var filteredProps []*Property

	// Process each property to extract special override properties
	for _, prop := range propertyList {
		switch prop.Name {
		case "arch":
			// Architecture-specific overrides: arch: { arm: {...}, arm64: {...} }
			archMap, ok := prop.Value.(*Map)
			if !ok {
				return nil, errors.Syntax("expected map value for 'arch' override").
					WithLocation(p.fileName, prop.ColonPos.Line, prop.ColonPos.Column).
					WithSuggestion("arch: requires map value like arch: { arm: {...} }")
			}
			for _, ap := range archMap.Properties {
				archInner, ok := ap.Value.(*Map)
				if !ok {
					return nil, errors.Syntax(fmt.Sprintf("expected map value for arch override '%s'", ap.Name)).
						WithLocation(p.fileName, ap.ColonPos.Line, ap.ColonPos.Column).
						WithSuggestion("Architecture variant requires map value")
				}
				archProps[ap.Name] = archInner
			}
		case "host":
			// Host-specific overrides: host: { ... }
			m, ok := prop.Value.(*Map)
			if !ok {
				return nil, errors.Syntax("expected map value for 'host' override").
					WithLocation(p.fileName, prop.ColonPos.Line, prop.ColonPos.Column).
					WithSuggestion("host: requires map value like host: { ... }")
			}
			hostProps = m
		case "target":
			// Target-specific overrides: target: { ... }
			m, ok := prop.Value.(*Map)
			if !ok {
				return nil, errors.Syntax("expected map value for 'target' override").
					WithLocation(p.fileName, prop.ColonPos.Line, prop.ColonPos.Column).
					WithSuggestion("target: requires map value like target: { ... }")
			}
			targetProps = m
		case "multilib":
			// Multilib overrides: multilib: { lib32: {...}, lib64: {...} }
			mlMap, ok := prop.Value.(*Map)
			if !ok {
				return nil, errors.Syntax("expected map value for 'multilib' override").
					WithLocation(p.fileName, prop.ColonPos.Line, prop.ColonPos.Column).
					WithSuggestion("multilib: requires map value like multilib: { lib32: {...} }")
			}
			for _, mp := range mlMap.Properties {
				mlInner, ok := mp.Value.(*Map)
				if !ok {
					return nil, errors.Syntax(fmt.Sprintf("expected map value for multilib override '%s'", mp.Name)).
						WithLocation(p.fileName, mp.ColonPos.Line, mp.ColonPos.Column).
						WithSuggestion("Multilib variant requires map value")
				}
				multilibProps[mp.Name] = mlInner
			}
		case "override":
			// Override flag: override: true
			if b, ok := prop.Value.(*Bool); ok {
				overrideFound = b.Value
			}
		default:
			// Regular property - keep in main property list
			filteredProps = append(filteredProps, prop)
		}
	}

	// Create the module with extracted properties
	mod := &Module{
		Type:     typeName,
		TypePos:  typePos,
		Map:      &Map{Properties: filteredProps, LBracePos: lbracePos, RBracePos: rbracePos},
		Arch:     archProps,
		Host:     hostProps,
		Target:   targetProps,
		Multilib: multilibProps,
		Override: overrideFound,
	}

	return mod, nil
}

// parsePropertyList parses a list of properties: { property [, property] }
// Properties are key-value pairs separated by commas. Trailing commas are allowed.
// The parser reads properties until it encounters a closing brace (}).
//
// Returns:
//   - []*Property: List of parsed properties
//   - scanner.Position: Position of the closing right brace
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parsePropertyList() ([]*Property, scanner.Position, error) {
	properties := []*Property{}
	var rbracePos scanner.Position
	var lastProp *Property

	// Parse properties until we hit the closing brace or EOF
	for p.curToken.Type != EOF && p.curToken.Type != RBRACE {
		prop, err := p.parseProperty()
		if err != nil {
			return nil, rbracePos, err
		}
		if prop != nil {
			properties = append(properties, prop)
			lastProp = prop
		}

		// Check if we've reached the closing brace
		if p.curToken.Type == RBRACE {
			break
		}

		// Comma separates adjacent properties; trailing commas are still allowed.
		if p.curToken.Type == COMMA {
			p.nextToken()
			continue
		}

		// Error if neither comma nor closing brace - point to last property
		var errPos scanner.Position
		errContent := p.lineContent(p.curToken.Pos.Line)
		caretLen := 0
		if lastProp != nil {
			errPos = lastProp.NamePos
			errContent = p.lineContent(lastProp.NamePos.Line)
			caretLen = len(lastProp.Name)
		} else {
			errPos = p.curToken.Pos
		}
		return nil, rbracePos, errors.Syntax("expected ',' or '}' after property").
			WithLocation(p.fileName, errPos.Line, errPos.Column).
			WithContent(errContent).
			WithContentCaret(caretLen).
			WithSuggestion("Properties must be separated by commas")
	}

	// Verify we found the closing brace
	if p.curToken.Type != RBRACE {
		return nil, rbracePos, errors.Syntax("expected }").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(1).
			WithSuggestion("Module block should end with '}'")
	}
	rbracePos = p.curToken.Pos
	p.nextToken()

	return properties, rbracePos, nil
}

// parseProperty parses a single property: name : expression
// A property consists of an identifier, followed by a colon, followed by an expression.
// The expression can be a string, integer, boolean, list, map, variable, or select statement.
//
// Returns:
//   - *Property: The parsed property AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseProperty() (*Property, error) {
	if p.curToken.Type != IDENT {
		return nil, errors.Syntax(fmt.Sprintf("expected property name (identifier), got %s", p.curToken.Type)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(len(p.curToken.Literal)).
			WithSuggestion("Property names must be identifiers (unquoted names like name, srcs)")
	}

	name := p.curToken.Literal
	namePos := p.curToken.Pos

	p.nextToken()

	// Verify colon separator
	if p.curToken.Type != COLON {
		return nil, errors.Syntax(fmt.Sprintf("expected ':' after property name '%s'", name)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(1).
			WithSuggestion("Property name must be followed by ':'")
	}
	colonPos := p.curToken.Pos
	p.nextToken()

	// Parse the property value expression
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &Property{
		Name:     name,
		NamePos:  namePos,
		Value:    expr,
		ColonPos: colonPos,
	}, nil
}

// parseAssignment parses an assignment statement: name (= | +=) expression
// Assignments can be simple (=) or concatenative (+=).
// For +=, the parser handles string and list concatenation differently during evaluation:
// - String += appends to existing string
// - List += appends to existing list (or creates new list)
//
// Parameters:
//   - name: The variable name being assigned to
//   - namePos: The source position of the variable name
//
// Returns:
//   - *Assignment: The parsed assignment AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseAssignment(name string, namePos scanner.Position) (*Assignment, error) {
	assigner := "="
	equalsPos := p.curToken.Pos

	if p.curToken.Type == PLUSEQ {
		assigner = "+="
	} else if p.curToken.Type != ASSIGN {
		return nil, errors.Syntax(fmt.Sprintf("expected '=' or '+=', got %s", p.curToken.Type)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(len(p.curToken.Literal)).
			WithSuggestion("Assignment operator should be '=' or '+='")
	}
	p.nextToken()

	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &Assignment{
		Name:      name,
		NamePos:   namePos,
		EqualsPos: equalsPos,
		Assigner:  assigner,
		Value:     expr,
	}, nil
}

// parseExpression parses any expression, including + operators.
// This handles left-to-right associativity for the + operator.
// For example, "a + b + c" is parsed as "(a + b) + c".
//
// The + operator can perform:
// - Integer addition (int64 + int64)
// - String concatenation (string + string)
// - List concatenation (list + list)
//
// Returns:
//   - Expression: The parsed expression AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseExpression() (Expression, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Handle + operator for concatenation/addition
	// Uses left-to-right associativity
	for p.curToken.Type == PLUS {
		opPos := p.curToken.Pos
		p.nextToken()

		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}

		// Create binary operator node
		left = &Operator{
			Args:        [2]Expression{left, right},
			Operator:    '+',
			OperatorPos: opPos,
		}
	}

	return left, nil
}

// parsePrimary parses a single primary expression (no operators).
// Primary expressions are the base units that cannot be broken down further:
//   - STRING: Quoted string literals
//   - INT: Integer literals
//   - BOOL: Boolean literals (true/false)
//   - LBRACKET: List expressions [expr, ...]
//   - LBRACE: Map expressions { prop: value, ... }
//   - IDENT: Either the "select" keyword or a variable reference
//
// Returns:
//   - Expression: The parsed primary expression
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parsePrimary() (Expression, error) {
	switch p.curToken.Type {
	case STRING:
		return p.parseString()
	case INT:
		return p.parseInt()
	case BOOL:
		return p.parseBool()
	case LBRACKET:
		return p.parseList()
	case LBRACE:
		return p.parseMap()
	case IDENT:
		// Check for select() keyword vs variable reference
		if p.curToken.Literal == "select" {
			return p.parseSelect()
		}
		return p.parseVariable()
	case UNSET:
		// Unset keyword for removing property values
		pos := p.curToken.Pos
		p.nextToken()
		return &Unset{KeywordPos: pos}, nil
	default:
		return nil, errors.Syntax(fmt.Sprintf("unexpected token %s in expression", p.curToken.Type)).
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(len(p.curToken.Literal)).
			WithSuggestion("Expression value expected (string, list, or map)")
	}
}

// parseString parses a string literal.
// String literals are surrounded by quotes and may contain escape sequences.
// The parser removes the quotes and processes escape sequences using strconv.Unquote.
// Both single-quoted and double-quoted strings are supported, as well as raw strings.
//
// Returns:
//   - *String: The parsed string AST node
//   - error: nil if successful, otherwise a parse error (e.g., unterminated string)
func (p *Parser) parseString() (*String, error) {
	pos := p.curToken.Pos
	literal := p.curToken.Literal
	p.nextToken()

	// Remove quotes from literal and process escape sequences
	value, err := strconv.Unquote(literal)
	if err != nil {
		return nil, errors.Syntax(fmt.Sprintf("invalid string literal: %v", err)).
			WithLocation(p.fileName, pos.Line, pos.Column).
			WithContent(p.lineContent(pos.Line)).
			WithContentCaret(len(literal)).
			WithSuggestion("String literal must be properly quoted")
	}

	return &String{
		Value:      value,
		LiteralPos: pos,
	}, nil
}

// parseInt parses an integer literal.
// Integer literals are base-10 numbers that are parsed into int64 values.
// They can be positive or negative.
//
// Returns:
//   - *Int64: The parsed integer AST node
//   - error: nil if successful, otherwise a parse error (e.g., overflow)
func (p *Parser) parseInt() (*Int64, error) {
	pos := p.curToken.Pos
	literal := p.curToken.Literal
	p.nextToken()

	value, err := strconv.ParseInt(literal, 10, 64)
	if err != nil {
		return nil, errors.Syntax(fmt.Sprintf("invalid integer literal: %v", err)).
			WithLocation(p.fileName, pos.Line, pos.Column).
			WithContent(p.lineContent(pos.Line)).
			WithContentCaret(len(literal)).
			WithSuggestion("Integer must be a valid number")
	}

	return &Int64{
		Value:      value,
		LiteralPos: pos,
	}, nil
}

// parseBool parses a boolean literal.
// Boolean literals are the keywords "true" and "false".
//
// Returns:
//   - *Bool: The parsed boolean AST node
func (p *Parser) parseBool() (*Bool, error) {
	pos := p.curToken.Pos
	literal := p.curToken.Literal
	p.nextToken()

	return &Bool{
		Value:      literal == "true",
		LiteralPos: pos,
	}, nil
}

// parseVariable parses a variable reference.
// A variable reference is an identifier that refers to a previously defined variable
// or assignment. During evaluation, the variable's value will be substituted
// for the reference.
//
// Returns:
//   - *Variable: The parsed variable reference AST node
func (p *Parser) parseVariable() (*Variable, error) {
	pos := p.curToken.Pos
	name := p.curToken.Literal
	p.nextToken()

	return &Variable{
		Name:    name,
		NamePos: pos,
	}, nil
}

// parseList parses a list: [ expression [, expression] ]
// Lists are ordered collections of expressions, separated by commas.
// Trailing commas are allowed.
//
// Returns:
//   - *List: The parsed list AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseList() (*List, error) {
	lbracePos := p.curToken.Pos
	p.nextToken()

	values := []Expression{}
	var rbracePos scanner.Position

	// Parse list elements until closing bracket
	for p.curToken.Type != EOF && p.curToken.Type != RBRACKET {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		values = append(values, expr)

		// Check for closing bracket
		if p.curToken.Type == RBRACKET {
			break
		}

		// Comma separates adjacent elements; trailing commas are still allowed.
		if p.curToken.Type == COMMA {
			p.nextToken()
			continue
		}

		return nil, errors.Syntax("expected ',' or ']' after list element").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(len(p.curToken.Literal)).
			WithSuggestion("List elements must be separated by commas")
	}

	// Verify closing bracket
	if p.curToken.Type != RBRACKET {
		return nil, errors.Syntax("expected ]").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithContent(p.lineContent(p.curToken.Pos.Line)).
			WithContentCaret(1).
			WithSuggestion("List should end with ']'")
	}
	rbracePos = p.curToken.Pos
	p.nextToken()

	return &List{
		Values:    values,
		LBracePos: lbracePos,
		RBracePos: rbracePos,
	}, nil
}

// parseMap parses a map: { property_list }
// Maps are collections of key-value pairs enclosed in braces.
// They share the same syntax as property lists, so parsePropertyList is reused.
//
// Returns:
//   - *Map: The parsed map AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseMap() (*Map, error) {
	lbracePos := p.curToken.Pos
	p.nextToken()

	propertyList, rbracePos, err := p.parsePropertyList()
	if err != nil {
		return nil, err
	}

	return &Map{
		Properties: propertyList,
		LBracePos:  lbracePos,
		RBracePos:  rbracePos,
	}, nil
}

// parseSelect parses a select expression: select(conditions, { cases })
// Select is a conditional expression that chooses values based on configuration.
// The syntax is: select(condition, { pattern1: value1, pattern2: value2, ... })
// The first argument is a condition (like "arch", "os", "host") or a variable.
// The second argument is a map of patterns to values. The "default" pattern is used
// when no other pattern matches.
//
// Select also supports:
// - Tuple conditions: select((arch(), os()), { ... }) for multi-condition matching
// - Unset patterns: select(arch(), { unset: value })
// - Any patterns: select(arch(), { any: value }) or select(arch(), { any @var: value })
// - Any @var binding: Binds the matched value to a variable for use in the result
//
// Example usage:
//
//	srcs: select(arch(), {
//	    arm: ["arm.c"],
//	    arm64: ["arm64.c"],
//	    default: ["common.c"],
//	})
//
// Parameters:
//   - None (uses parser state)
//
// Returns:
//   - *Select: The parsed select AST node
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseSelect() (*Select, error) {
	keywordPos := p.curToken.Pos
	p.nextToken()

	// Expect opening parenthesis after select keyword
	if p.curToken.Type != LPAREN {
		return nil, errors.Syntax("expected '(' after 'select'").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("select() requires parentheses")
	}
	p.nextToken()

	conditions := []ConfigurableCondition{}

	// Check for tuple condition: select((arch(), os()), {...})
	// When conditions are enclosed in extra parens, multiple conditions are evaluated together
	if p.curToken.Type == LPAREN {
		p.nextToken()
		for p.curToken.Type != EOF && p.curToken.Type != RPAREN {
			cond, err := p.parseConfigurableCondition()
			if err != nil {
				return nil, err
			}
			conditions = append(conditions, cond)
			if p.curToken.Type == COMMA {
				p.nextToken()
			}
		}
		if p.curToken.Type != RPAREN {
			return nil, errors.Syntax("expected ')' after tuple conditions").
				WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
				WithSuggestion("Tuple conditions must be closed with ')'")
		}
		p.nextToken()
	} else {
		// Single condition
		cond, err := p.parseConfigurableCondition()
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, cond)
	}

	// Expect comma between conditions and cases
	if p.curToken.Type == COMMA {
		p.nextToken()
	}

	// Parse cases: { case_pattern: value, ... }
	if p.curToken.Type != LBRACE {
		return nil, errors.Syntax("expected '{' for select cases").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("select() needs cases like { arch: value }")
	}
	lbracePos := p.curToken.Pos
	p.nextToken()

	// Parse each case in the select
	cases := []SelectCase{}
	for p.curToken.Type != EOF && p.curToken.Type != RBRACE {
		caseItem, err := p.parseSelectCase(len(conditions) > 1)
		if err != nil {
			return nil, err
		}
		cases = append(cases, caseItem)

		if p.curToken.Type == COMMA {
			p.nextToken()
		}
	}

	// Verify closing braces and parenthesis
	if p.curToken.Type != RBRACE {
		return nil, errors.Syntax("expected '}' after select cases").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("select() cases block should end with '}'")
	}
	rbracePos := p.curToken.Pos
	p.nextToken()

	if p.curToken.Type != RPAREN {
		return nil, errors.Syntax("expected ')' after select cases").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("select() should end with ')'")
	}
	p.nextToken()

	return &Select{
		KeywordPos: keywordPos,
		Conditions: conditions,
		LBracePos:  lbracePos,
		RBracePos:  rbracePos,
		Cases:      cases,
	}, nil
}

// parseConfigurableCondition parses a condition for select.
// Conditions can be simple identifiers (like "arch", "os") or function calls
// with arguments (like "target(android)").
//
// Built-in condition functions:
// - arch(): Current architecture (arm, arm64, x86, x86_64)
// - os(): Current operating system (linux, android, darwin)
// - host(): Whether building for host
// - target(): Target platform
// - variant(): Build variant (debug, release)
// - product_variable(): Product-specific variable
// - soong_config_variable(): Configuration variable from namespace
// - release_flag(): Release flag check
//
// Returns:
//   - ConfigurableCondition: The parsed condition
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseConfigurableCondition() (ConfigurableCondition, error) {
	if p.curToken.Type != IDENT {
		return ConfigurableCondition{}, errors.Syntax("expected identifier for condition").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("Use condition function like arch(), os()")
	}

	funcName := p.curToken.Literal
	pos := p.curToken.Pos
	p.nextToken()

	// Parse arguments if parentheses follow the function name
	args := []Expression{}
	if p.curToken.Type == LPAREN {
		p.nextToken()
		for p.curToken.Type != EOF && p.curToken.Type != RPAREN {
			arg, err := p.parseExpression()
			if err != nil {
				return ConfigurableCondition{}, err
			}
			args = append(args, arg)
			if p.curToken.Type == COMMA {
				p.nextToken()
			}
		}
		if p.curToken.Type == RPAREN {
			p.nextToken()
		}
	}

	return ConfigurableCondition{
		Position:     pos,
		FunctionName: funcName,
		Args:         args,
	}, nil
}

// parseSelectCase parses a single case in a select statement.
// A case consists of one or more patterns separated by commas, followed by a colon
// and a value expression. Multiple patterns can map to the same value.
// Example: "linux", "android": ["unix.c"]
//
// Parameters:
//   - isTuple: True if this is a tuple select (multiple conditions)
//
// Returns:
//   - SelectCase: The parsed case
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseSelectCase(isTuple bool) (SelectCase, error) {
	// Handle tuple patterns (multiple values in parentheses)
	if isTuple && p.curToken.Type == LPAREN {
		return p.parseTupleSelectCase()
	}
	return p.parseSimpleSelectCase()
}

// parseTupleSelectCase parses a tuple case in a select statement.
// A tuple case has multiple patterns enclosed in parentheses, e.g., (arm, linux): value.
// This is used when select() has multiple conditions.
//
// Returns:
//   - SelectCase: The parsed case with tuple patterns and a value
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseTupleSelectCase() (SelectCase, error) {
	if p.curToken.Type != LPAREN {
		return SelectCase{}, errors.Syntax("expected '(' for tuple pattern in select case").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("Tuple pattern needs parentheses like (arm, linux)")
	}
	p.nextToken()

	// Parse each pattern in the tuple
	var patterns []SelectPattern
	for p.curToken.Type != EOF && p.curToken.Type != RPAREN {
		pattern, err := p.parseSelectPattern()
		if err != nil {
			return SelectCase{}, err
		}
		patterns = append(patterns, pattern)
		if p.curToken.Type == COMMA {
			p.nextToken()
		}
	}

	if p.curToken.Type != RPAREN {
		return SelectCase{}, errors.Syntax("expected ')' after tuple pattern").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("Tuple pattern must be closed with ')'")
	}
	p.nextToken()

	// Expect colon before value
	if p.curToken.Type != COLON {
		return SelectCase{}, errors.Syntax("expected ':' after select pattern").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("Pattern must be followed by ':' and value")
	}
	colonPos := p.curToken.Pos
	p.nextToken()

	// Parse the value expression
	value, err := p.parseExpression()
	if err != nil {
		return SelectCase{}, err
	}

	return SelectCase{
		Patterns: patterns,
		ColonPos: colonPos,
		Value:    value,
	}, nil
}

// parseSimpleSelectCase parses a simple (non-tuple) case in a select statement.
// A simple case has one or more patterns separated by commas, then a colon, then a value.
// Multiple patterns can map to the same value (e.g., "linux", "android": ["unix.c"]).
//
// Returns:
//   - SelectCase: The parsed case with one or more patterns and a value
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseSimpleSelectCase() (SelectCase, error) {
	// Parse first pattern
	pattern, err := p.parseSelectPattern()
	if err != nil {
		return SelectCase{}, err
	}
	patterns := []SelectPattern{pattern}

	// Parse additional patterns separated by commas
	for p.curToken.Type == COMMA {
		p.nextToken()
		pattern, err := p.parseSelectPattern()
		if err != nil {
			return SelectCase{}, err
		}
		patterns = append(patterns, pattern)
	}

	// Expect colon before value
	if p.curToken.Type != COLON {
		return SelectCase{}, errors.Syntax("expected ':' after select pattern").
			WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
			WithSuggestion("Pattern must be followed by ':' and value")
	}
	colonPos := p.curToken.Pos
	p.nextToken()

	// Parse the value expression
	value, err := p.parseExpression()
	if err != nil {
		return SelectCase{}, err
	}

	return SelectCase{
		Patterns: patterns,
		ColonPos: colonPos,
		Value:    value,
	}, nil
}

// parseSelectPattern parses a single pattern in a select case.
// A pattern is an expression that is compared against the condition value.
// Common patterns include string literals (e.g., "linux"), integer literals,
// boolean literals, variable references like "default", or special keywords:
//
// - unset: Matches when configuration is not set or empty
// - any: Matches any value (wildcard)
// - any @ var: Matches any value and binds it to a variable
//
// Returns:
//   - SelectPattern: The parsed pattern
//   - error: nil if successful, otherwise a parse error
func (p *Parser) parseSelectPattern() (SelectPattern, error) {
	switch p.curToken.Type {
	case UNSET:
		// Unset pattern - matches nil or empty configuration value
		pos := p.curToken.Pos
		p.nextToken()
		return SelectPattern{Value: &Unset{KeywordPos: pos}}, nil
	case AT:
		// @ prefix for binding: @variable
		p.nextToken()
		if p.curToken.Type != IDENT {
			return SelectPattern{}, errors.Syntax("expected variable name after '@'").
				WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
				WithSuggestion("Use @variable to bind matched value")
		}
		binding := p.curToken.Literal
		p.nextToken()
		return SelectPattern{Value: &Variable{Name: "any", NamePos: p.curToken.Pos}, IsAny: true, Binding: binding}, nil
	case IDENT:
		// Check for "any @ var" pattern
		if p.curToken.Literal == "any" && p.peekToken.Type == AT {
			p.nextToken() // consume "any"
			p.nextToken() // consume "@"
			if p.curToken.Type != IDENT {
				return SelectPattern{}, errors.Syntax("expected variable name after '@'").
					WithLocation(p.fileName, p.curToken.Pos.Line, p.curToken.Pos.Column).
					WithSuggestion("Use @variable to bind matched value")
			}
			binding := p.curToken.Literal
			p.nextToken()
			return SelectPattern{Value: &Variable{Name: "any", NamePos: p.curToken.Pos}, IsAny: true, Binding: binding}, nil
		}
		fallthrough
	default:
		// Regular expression as pattern
		expr, err := p.parseExpression()
		if err != nil {
			return SelectPattern{}, err
		}
		return SelectPattern{Value: expr}, nil
	}
}

// ParseFile parses a Blueprint file from an io.Reader.
// This is a convenience function that creates a parser and parses the entire file.
// It handles all the setup工作和错误处理 so callers don't need to deal with the parser directly.
//
// Parameters:
//   - r: The input reader containing Blueprint source code
//   - fileName: The name of the file (used for error messages)
//
// Returns:
//   - *File: The parsed AST
//   - error: nil if successful, otherwise the first error encountered
func ParseFile(r io.Reader, fileName string, source ...string) (*File, error) {
	if r == nil {
		if len(source) > 0 {
			r = strings.NewReader(source[0])
		} else {
			return nil, fmt.Errorf("ParseFile: reader is nil and no source provided for %s", fileName)
		}
	}
	parser := NewParser(r, fileName, source...)
	file, errors := parser.Parse()
	if len(errors) > 0 {
		return file, errors[0]
	}
	return file, nil
}

// lineContent returns the content of the specified line (1-indexed).
// Returns empty string if line number is out of range or source is not available.
func (p *Parser) lineContent(line int) string {
	if p.source == "" || line <= 0 {
		return ""
	}
	lines := strings.Split(p.source, "\n")
	if line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// init is called to initialize the parser package.
// Currently a no-op but may be used for future setup.
func init() {
	// Reserved for package initialization
}
