// Package parser provides lexical analysis and parsing for Blueprint build definitions.
// Lexer subpackage - Tokenization of Blueprint source files.
//
// This package implements the first stage of the Blueprint build system:
// it reads raw source text and produces a stream of tokens.
// The lexer wraps Go's text/scanner package to provide Blueprint-specific tokenization.
//
// Token types:
//   - Special tokens: EOF (end of file), ILLEGAL (invalid character)
//   - Literals: IDENT (identifiers), STRING (string literals), INT (integers), BOOL (true/false)
//   - Symbols: LPAREN, RPAREN, LBRACE, RBRACE, LBRACKET, RBRACKET
//   - Operators: COLON, COMMA, PLUS, ASSIGN, PLUSEQ, UNSET, AT
//
// String support:
//   - Double-quoted strings: "hello world"
//   - Single-quoted strings: 'hello world'
//   - Raw strings: `hello world`
//   - Escape sequences: \n, \t, \\, \", etc.
//
// Error handling:
//   - Lexer errors are collected and returned separately
//   - Invalid characters are reported but scanning continues
//   - Position information is included for all errors
package parser

import (
	"fmt"
	"io"
	"strconv"
	"text/scanner"
)

// TokenType represents the type of a lexical token.
// Each token type corresponds to a specific kind of syntax element in Blueprint.
//
// Token categories:
//   - Special: EOF (end of file), ILLEGAL (unrecognized character)
//   - Literals: IDENT (variable/module names), STRING (quoted text), INT (numbers), BOOL (true/false)
//   - Grouping: LPAREN/RPAREN (function calls, grouping), LBRACE/RBRACE (modules, maps),
//             LBRACKET/RBRACKET (lists)
//   - Operators: COLON (property separator), COMMA (separator), PLUS (concatenation),
//             ASSIGN simple assignment (=), PLUSEQ (+=), UNSET (unset keyword), AT (@ binding)
//
// These token types are the fundamental building blocks that the parser uses to
// understand the syntactic structure of Blueprint source files. The lexer converts
// raw character input into a stream of these typed tokens for consumption by the parser.
type TokenType int

const (
	// Special tokens (internal markers)
	EOF     TokenType = iota // End of file marker - returned when no more input
	ILLEGAL                  // Unknown/invalid character - recorded as error but scanning continues

	// Literals - values that appear directly in source code
	IDENT  // Identifiers: variable names (my_var), module types (cc_binary), property names (srcs)
	STRING // String literals: "hello", 'hello', `raw string`
	INT    // Integer literals: 42, 100, -10
	BOOL   // Boolean literals: true, false

	// Grouping symbols - structure markers
	LPAREN   // (  Left parenthesis - for function calls and grouping
	RPAREN   // )  Right parenthesis
	LBRACE   // {  Left brace - for module blocks and maps
	RBRACE   // }  Right brace
	LBRACKET // [  Left bracket - for lists
	RBRACKET // ]  Right bracket

	// Operators and separators
	COLON    // :  Colon - property separator in maps
	COMMA    // ,  Comma - list/element separator
	PLUS     // +  Plus - concatenation operator
	ASSIGN   // =  Equals - simple assignment operator
	PLUSEQ   // += Plus-equals - concatenation assignment operator
	UNSET    // unset keyword - for removing property values in select
	AT       // @  At sign - for any @ var binding in select patterns
)

// Token represents a lexical token with its type, literal value, and source position.
// This is the fundamental unit of output from the lexer.
//
// Token structure:
//   - Type: The token type (TokenType enum)
//   - Literal: The actual text from the source (for identifiers, strings, numbers, symbols)
//   - Pos: Source position (filename, line, column) for error reporting
//
// Example tokens:
//   - Token{Type: IDENT, Literal: "cc_binary", Pos: file.bp:1:1}
//   - Token{Type: STRING, Literal: "\"hello\"", Pos: file.bp:2:5}
//   - Token{Type: ASSIGN, Literal: "=", Pos: file.bp:3:10}
//
// The Token struct is the primary data structure that flows between the lexer
// and parser. Each call to NextToken() returns one Token with complete information
// needed for parsing and accurate error reporting.
type Token struct {
	Type    TokenType        // The type of this token
	Literal string           // The actual text of the token (for identifiers, strings, etc.)
	Pos     scanner.Position // Source position (file, line, column) for error reporting
}

// Lexer wraps text/scanner to provide Blueprint-specific tokenization.
// It converts raw source text into a stream of Token values that the parser can consume.
//
// The lexer handles:
//   - Go-compatible string literals (double-quoted, single-quoted, raw)
//   - Integer literals (decimal integers)
//   - Identifiers (variable names, module types, property names)
//   - Keywords (true, false, unset)
//   - Symbols (parentheses, braces, brackets)
//   - Operators (=, +=, :, ,, +)
//   - Comments (skipped entirely)
//
// Token production:
//   - NextToken() returns the next token in the input stream
//   - Tokenization is incremental - tokens are produced on demand
//   - The scanner scans ahead to find token boundaries
//
// Error handling:
//   - Invalid characters are recorded in the errors slice
//   - Scanning continues after errors for incremental reporting
//
// This lexer is the first stage in the Blueprint build system pipeline.
// It sits between the raw input stream and the parser, converting character
// sequences into meaningful tokens that represent the syntactic structure
// of the Blueprint source code.
type Lexer struct {
	scanner scanner.Scanner // The underlying Go scanner - provides character scanning
	ch      rune            // Current character being processed (cached for peeking)
	errors  []error         // List of lexer errors encountered during scanning
}

// NewLexer creates a new lexer from an io.Reader.
// It initializes the Go scanner with appropriate mode settings for Blueprint.
//
// The scanner is configured with the following modes:
//   - ScanIdents: Recognize identifiers (variable names, module types)
//   - ScanInts: Recognize integer literals (42, 100, -10)
//   - ScanStrings: Recognize quoted strings ("hello", 'hello')
//   - ScanRawStrings: Recognize raw strings (`hello`)
//   - ScanComments: Skip comments entirely from the token stream
//
// The lexer also sets up:
//   - Whitespace handling: Space, tab, newline, carriage return are skipped
//   - Error callback: Lexer errors are collected in the errors slice
//   - Filename tracking: The filename is stored for error reporting
//
// Parameters:
//   - r: The input reader containing Blueprint source code
//   - fileName: The name of the file being lexed (used for error messages)
//
// Returns:
//   - A new Lexer instance ready to produce tokens
//
// Example usage:
//   lexer := NewLexer(strings.NewReader("cc_library { srcs: [\"*.c\"] }"), "Android.bp")
//   for tok := lexer.NextToken(); tok.Type != EOF; tok = lexer.NextToken() {
//       // Process token...
//   }
func NewLexer(r io.Reader, fileName string) *Lexer {
	l := &Lexer{}
	l.scanner.Init(r)
	l.scanner.Filename = fileName
	l.scanner.Error = func(s *scanner.Scanner, msg string) {
		l.errors = append(l.errors, fmt.Errorf("%s: %s", s.Position, msg))
	}
	// Allow scanning strings (quoted and raw) and comments.
	l.scanner.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments
	l.scanner.Whitespace = 1<<' ' | 1<<'\t' | 1<<'\n' | 1<<'\r'
	l.next()
	return l
}

// next advances the lexer to the next character in the input stream.
// It calls the underlying Go scanner's Scan() method to retrieve the next rune.
// This is the fundamental operation for traversing the source text character by character.
// After calling next(), the ch field contains the next character to be processed.
func (l *Lexer) next() {
	l.ch = l.scanner.Scan()
}

// peek returns the next character without advancing the scanner.
// This allows the lexer to look ahead at the upcoming character
// to determine how to tokenize it (e.g., to distinguish += from +).
// Returns:
//   - The next rune in the input, or EOF if at end of input
func (l *Lexer) peek() rune {
	return l.scanner.Peek()
}

// NextToken returns the next token from the input.
// This is the main entry point for the parser to consume tokens.
// It handles all token types: special tokens (EOF, ILLEGAL), literals (IDENT, STRING, INT, BOOL),
// and symbols (parentheses, braces, brackets, colon, comma, operators).
// The lexer automatically skips comments and whitespace.
//
// Token processing flow:
//   1. Record the current source position for the token
//   2. Switch on the current character to determine token type
//   3. For identifiers, further classify as keyword or regular identifier
//   4. For multi-character tokens (=, +=), peek at the next character
//   5. Advance to the next character
//   6. Return the complete token
//
// Special handling:
//   - '+' followed by '=': Returns PLUSEQ token, advances past both
//   - 'true'/'false' identifiers: Returns BOOL token
//   - 'unset' identifier: Returns UNSET token
//   - Unknown characters: Returns ILLEGAL token, records error
//
// Returns:
//   - Token: The next lexical token with type, literal value, and position
//
// Error cases:
//   - Invalid characters are recorded but scanning continues
//   - Negative character codes (end of input) return EOF token
func (l *Lexer) NextToken() Token {
	var tok Token
	tok.Pos = l.scanner.Position

	switch l.ch {
	case scanner.EOF:
		// End of file - return special EOF token
		tok.Type = EOF
		tok.Literal = ""
	case '(':
		// Left parenthesis
		tok.Type = LPAREN
		tok.Literal = "("
		l.next()
	case ')':
		// Right parenthesis
		tok.Type = RPAREN
		tok.Literal = ")"
		l.next()
	case '{':
		// Left brace (opening block)
		tok.Type = LBRACE
		tok.Literal = "{"
		l.next()
	case '}':
		// Right brace (closing block)
		tok.Type = RBRACE
		tok.Literal = "}"
		l.next()
	case '[':
		// Left bracket (opening list)
		tok.Type = LBRACKET
		tok.Literal = "["
		l.next()
	case ']':
		// Right bracket (closing list)
		tok.Type = RBRACKET
		tok.Literal = "]"
		l.next()
	case ':':
		// Colon (property separator in maps)
		tok.Type = COLON
		tok.Literal = ":"
		l.next()
	case ',':
		// Comma (list/element separator)
		tok.Type = COMMA
		tok.Literal = ","
		l.next()
	case '+':
		// Plus operator - check for += compound assignment
		l.next()
		if l.ch == '=' {
			tok.Type = PLUSEQ
			tok.Literal = "+="
			l.next()
		} else {
			tok.Type = PLUS
			tok.Literal = "+"
		}
	case '=':
		// Simple assignment operator
		tok.Type = ASSIGN
		tok.Literal = "="
		l.next()
	case '@':
		// At sign for variable binding in select patterns
		tok.Type = AT
		tok.Literal = "@"
		l.next()
	case scanner.Comment:
		// Skip comments and get next token
		// Comments are filtered out entirely from the token stream
		l.next()
		return l.NextToken()
	case scanner.Int:
		// Integer literal (base-10 number)
		tok.Type = INT
		tok.Literal = l.scanner.TokenText()
		l.next()
	case scanner.String, scanner.RawString:
		// Quoted string literal (single or double quotes, raw with backticks)
		tok.Type = STRING
		tok.Literal = l.scanner.TokenText()
		l.next()
	case scanner.Ident:
		// Identifier - could be keyword, variable name, or module type
		tok.Literal = l.scanner.TokenText()
		switch tok.Literal {
		case "true", "false":
			// Boolean literals
			tok.Type = BOOL
		case "unset":
			// Unset keyword for removing property values
			tok.Type = UNSET
		default:
			// Regular identifier (variable name, module type, property name)
			tok.Type = IDENT
		}
		l.next()
	case '\n', '\t', ' ', '\r':
		// Skip whitespace and get next token
		// All whitespace is treated as token separators
		l.next()
		return l.NextToken()
	default:
		// Unknown character - record error but continue processing
		if l.ch < 0 {
			// Negative character means EOF was reached
			tok.Type = EOF
		} else {
			// Illegal character - not recognized by the scanner
			tok.Type = ILLEGAL
			tok.Literal = string(l.ch)
			// Record error for illegal characters so parser can report them
			l.errors = append(l.errors, fmt.Errorf("%s: illegal character '%c'", l.scanner.Position, l.ch))
			l.next()
		}
	}

	return tok
}

// Position returns the current scanner position.
// This is used for error reporting to show exactly where in the source file an error occurred.
// The position includes filename, line number, and column number.
func (l *Lexer) Position() scanner.Position {
	return l.scanner.Position
}

// Error creates an error with position information.
// This is a helper for generating lexer errors with the current source position.
// It includes the file location in the error message for accurate error reporting.
// Parameters:
//   - format: Printf-style format string
//   - args: Arguments for the format string
//
// Returns:
//   - An error with position information formatted as "filename:line:column: message"
func (l *Lexer) Error(format string, args ...interface{}) error {
	return fmt.Errorf("%s: %s", l.scanner.Position, fmt.Sprintf(format, args...))
}

// Errors returns lexer diagnostics collected from text/scanner.
// These are errors encountered during scanning, such as invalid characters or malformed tokens.
// Lexer errors are collected and returned separately so the parser can decide
// whether to continue or abort processing.
// Returns:
//   - []error: List of lexer errors, empty if no errors encountered
func (l *Lexer) Errors() []error {
	return l.errors
}

// Unquote removes quotes from a string literal.
// This is a wrapper around strconv.Unquote that handles Go string syntax,
// including escape sequences like \n, \t, \", etc.
//
// Supported string formats:
//   - Double-quoted strings: "hello\nworld" - supports escape sequences
//   - Single-quoted strings: 'hello' - supports escape sequences
//   - Raw strings: `hello\nworld` - no escape processing
//
// Parameters:
//   - s: A string literal (including quotes)
//
// Returns:
//   - string: The unquoted string content
//   - error: nil if successful, otherwise an error (e.g., invalid escape sequence,
//     unterminated string, invalid unicode surrogate)
//
// Example:
//   Unquote(`"hello\nworld"`) -> "hello\nworld", nil
//   Unquote("'unterminated") -> "", error
func Unquote(s string) (string, error) {
	return strconv.Unquote(s)
}

// String returns a human-readable representation of a TokenType.
// This is useful for debugging and error messages.
// It converts the internal token type constant to a descriptive string.
func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case IDENT:
		return "IDENT"
	case STRING:
		return "STRING"
	case INT:
		return "INT"
	case BOOL:
		return "BOOL"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case LBRACKET:
		return "LBRACKET"
	case RBRACKET:
		return "RBRACKET"
	case COLON:
		return "COLON"
	case COMMA:
		return "COMMA"
	case PLUS:
		return "PLUS"
	case ASSIGN:
		return "ASSIGN"
	case PLUSEQ:
		return "PLUSEQ"
	case UNSET:
		return "UNSET"
	case AT:
		return "AT"
	default:
		return fmt.Sprintf("Token(%d)", t)
	}
}

// TokenError represents an error that occurred during tokenization.
// It includes the position in the source file and a descriptive message.
// This allows errors to be formatted with location information for
// easy identification in the source file.
type TokenError struct {
	Pos     scanner.Position // Position where the error occurred
	Message string           // Description of the error
}

// Error returns a formatted error string including the position and message.
// The format is "filename:line:column: message" for easy parsing by editors.
func (e *TokenError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// NewTokenError creates a new token error with the given position and message.
// This is a convenience constructor for TokenError that wraps the message
// in the error interface.
// Parameters:
//   - pos: The source position where the error occurred
//   - msg: The error message
//
// Returns:
//   - error: A TokenError with the specified position and message
func NewTokenError(pos scanner.Position, msg string) error {
	return &TokenError{Pos: pos, Message: msg}
}
