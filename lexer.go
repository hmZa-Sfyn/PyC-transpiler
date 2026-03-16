package main

import (
	"fmt"
	"strings"
	"unicode"
)

// ─── Token Types ──────────────────────────────────────────────────────────────

type TokenType string

const (
	// Literals
	TOK_INT_LIT    TokenType = "INT_LIT"
	TOK_FLOAT_LIT  TokenType = "FLOAT_LIT"
	TOK_STRING_LIT TokenType = "STRING_LIT"
	TOK_BOOL_LIT   TokenType = "BOOL_LIT"
	TOK_NONE       TokenType = "NONE"

	// Identifiers & Keywords
	TOK_IDENT  TokenType = "IDENT"
	TOK_DEF    TokenType = "def"
	TOK_RETURN TokenType = "return"
	TOK_IF     TokenType = "if"
	TOK_ELIF   TokenType = "elif"
	TOK_ELSE   TokenType = "else"
	TOK_WHILE  TokenType = "while"
	TOK_FOR    TokenType = "for"
	TOK_IN     TokenType = "in"
	TOK_BREAK  TokenType = "break"
	TOK_CONT   TokenType = "continue"
	TOK_PASS   TokenType = "pass"
	TOK_IMPORT TokenType = "import"
	TOK_FROM   TokenType = "from"
	TOK_AS     TokenType = "as"
	TOK_CLASS  TokenType = "class"
	TOK_AND    TokenType = "and"
	TOK_OR     TokenType = "or"
	TOK_NOT    TokenType = "not"
	TOK_IS     TokenType = "is"
	TOK_GLOBAL TokenType = "global"
	TOK_STRUCT TokenType = "struct"
	TOK_NEW    TokenType = "new"
	TOK_DELETE TokenType = "delete"

	// Type keywords
	TOK_TYPE_INT   TokenType = "int"
	TOK_TYPE_FLOAT TokenType = "float"
	TOK_TYPE_STR   TokenType = "str"
	TOK_TYPE_BOOL  TokenType = "bool"
	TOK_TYPE_VOID  TokenType = "void"
	TOK_TYPE_LIST  TokenType = "list"
	TOK_TYPE_ANY   TokenType = "any"

	// Operators
	TOK_PLUS      TokenType = "+"
	TOK_MINUS     TokenType = "-"
	TOK_STAR      TokenType = "*"
	TOK_SLASH     TokenType = "/"
	TOK_PERCENT   TokenType = "%"
	TOK_POWER     TokenType = "**"
	TOK_FLOORDIV  TokenType = "//"
	TOK_EQ        TokenType = "=="
	TOK_NEQ       TokenType = "!="
	TOK_LT        TokenType = "<"
	TOK_GT        TokenType = ">"
	TOK_LTE       TokenType = "<="
	TOK_GTE       TokenType = ">="
	TOK_ASSIGN    TokenType = "="
	TOK_PLUS_EQ   TokenType = "+="
	TOK_MINUS_EQ  TokenType = "-="
	TOK_STAR_EQ   TokenType = "*="
	TOK_SLASH_EQ  TokenType = "/="
	TOK_ARROW     TokenType = "->"
	TOK_AMP       TokenType = "&"
	TOK_PIPE      TokenType = "|"
	TOK_CARET     TokenType = "^"
	TOK_TILDE     TokenType = "~"
	TOK_LSHIFT    TokenType = "<<"
	TOK_RSHIFT    TokenType = ">>"

	// Delimiters
	TOK_LPAREN   TokenType = "("
	TOK_RPAREN   TokenType = ")"
	TOK_LBRACKET TokenType = "["
	TOK_RBRACKET TokenType = "]"
	TOK_LBRACE   TokenType = "{"
	TOK_RBRACE   TokenType = "}"
	TOK_COLON    TokenType = ":"
	TOK_COMMA    TokenType = ","
	TOK_DOT      TokenType = "."
	TOK_SEMICOL  TokenType = ";"
	TOK_AT       TokenType = "@"

	// Structure
	TOK_NEWLINE TokenType = "NEWLINE"
	TOK_INDENT  TokenType = "INDENT"
	TOK_DEDENT  TokenType = "DEDENT"
	TOK_EOF     TokenType = "EOF"
)

var keywords = map[string]TokenType{
	"def": TOK_DEF, "return": TOK_RETURN, "if": TOK_IF, "elif": TOK_ELIF,
	"else": TOK_ELSE, "while": TOK_WHILE, "for": TOK_FOR, "in": TOK_IN,
	"break": TOK_BREAK, "continue": TOK_CONT, "pass": TOK_PASS,
	"import": TOK_IMPORT, "from": TOK_FROM, "as": TOK_AS, "class": TOK_CLASS,
	"and": TOK_AND, "or": TOK_OR, "not": TOK_NOT, "is": TOK_IS,
	"True": TOK_BOOL_LIT, "False": TOK_BOOL_LIT, "None": TOK_NONE,
	"int": TOK_TYPE_INT, "float": TOK_TYPE_FLOAT, "str": TOK_TYPE_STR,
	"bool": TOK_TYPE_BOOL, "void": TOK_TYPE_VOID, "list": TOK_TYPE_LIST,
	"any": TOK_TYPE_ANY, "global": TOK_GLOBAL, "struct": TOK_STRUCT,
	"new": TOK_NEW, "delete": TOK_DELETE,
}

// ─── Token ────────────────────────────────────────────────────────────────────

type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Col     int
	LineStr string // full source line for error display
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%s, %q, %d:%d)", t.Type, t.Value, t.Line, t.Col)
}

// ─── Lexer ────────────────────────────────────────────────────────────────────

type Lexer struct {
	source   string
	lines    []string
	pos      int
	line     int
	col      int
	tokens   []Token
	indStack []int
	errors   []PycError
}

func NewLexer(source string) *Lexer {
	lines := strings.Split(source, "\n")
	return &Lexer{
		source:   source,
		lines:    lines,
		pos:      0,
		line:     1,
		col:      1,
		indStack: []int{0},
	}
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *Lexer) peekAt(offset int) byte {
	p := l.pos + offset
	if p >= len(l.source) {
		return 0
	}
	return l.source[p]
}

func (l *Lexer) advance() byte {
	ch := l.source[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) lineStr(ln int) string {
	if ln-1 < len(l.lines) {
		return l.lines[ln-1]
	}
	return ""
}

func (l *Lexer) makeToken(typ TokenType, val string, line, col int) Token {
	return Token{Type: typ, Value: val, Line: line, Col: col, LineStr: l.lineStr(line)}
}

func (l *Lexer) addError(code ErrorCode, line, col int, args ...interface{}) {
	ls := l.lineStr(line)
	l.errors = append(l.errors, newError(code, line, col, ls, args...))
}

func (l *Lexer) Tokenize() ([]Token, []PycError) {
	for l.pos < len(l.source) {
		l.lexLine()
	}
	// emit remaining DEDENTs
	for len(l.indStack) > 1 {
		l.tokens = append(l.tokens, l.makeToken(TOK_DEDENT, "", l.line, l.col))
		l.indStack = l.indStack[:len(l.indStack)-1]
	}
	l.tokens = append(l.tokens, l.makeToken(TOK_EOF, "", l.line, l.col))
	return l.tokens, l.errors
}

func (l *Lexer) lexLine() {
	// Handle indentation at start of logical line
	if l.col == 1 {
		indent := 0
		startPos := l.pos
		for l.pos < len(l.source) && (l.source[l.pos] == ' ' || l.source[l.pos] == '\t') {
			if l.source[l.pos] == '\t' {
				indent += 4
			} else {
				indent++
			}
			l.pos++
			l.col++
		}
		_ = startPos

		// skip blank lines and comment-only lines
		if l.pos < len(l.source) && (l.source[l.pos] == '\n' || l.source[l.pos] == '#') {
			// skip
		} else {
			curIndent := l.indStack[len(l.indStack)-1]
			if indent > curIndent {
				l.indStack = append(l.indStack, indent)
				l.tokens = append(l.tokens, l.makeToken(TOK_INDENT, "", l.line, 1))
			} else {
				for indent < l.indStack[len(l.indStack)-1] {
					l.indStack = l.indStack[:len(l.indStack)-1]
					l.tokens = append(l.tokens, l.makeToken(TOK_DEDENT, "", l.line, 1))
				}
				if indent != l.indStack[len(l.indStack)-1] {
					l.addError(ErrBadIndent, l.line, 1)
				}
			}
		}
	}

	if l.pos >= len(l.source) {
		return
	}

	ch := l.peek()

	switch {
	case ch == '#':
		// comment — skip to end of line
		for l.pos < len(l.source) && l.source[l.pos] != '\n' {
			l.pos++
			l.col++
		}

	case ch == '\n':
		line, col := l.line, l.col
		l.advance()
		l.tokens = append(l.tokens, l.makeToken(TOK_NEWLINE, "\n", line, col))

	case ch == '\r':
		l.advance()
		if l.peek() == '\n' {
			l.advance()
		}
		l.tokens = append(l.tokens, l.makeToken(TOK_NEWLINE, "\n", l.line, l.col))

	case ch == ' ' || ch == '\t':
		// mid-line whitespace — skip
		for l.pos < len(l.source) && (l.source[l.pos] == ' ' || l.source[l.pos] == '\t') {
			l.advance()
		}

	case ch == '"' || ch == '\'':
		l.lexString()

	case unicode.IsDigit(rune(ch)):
		l.lexNumber()

	case unicode.IsLetter(rune(ch)) || ch == '_':
		l.lexIdent()

	default:
		l.lexOp()
	}
}

func (l *Lexer) lexString() {
	line, col := l.line, l.col
	quote := l.advance()
	// triple quote?
	triple := false
	if l.peek() == quote && l.peekAt(1) == quote {
		triple = true
		l.advance()
		l.advance()
	}
	var buf strings.Builder
	for l.pos < len(l.source) {
		if triple {
			if l.peek() == quote && l.peekAt(1) == quote && l.peekAt(2) == quote {
				l.advance(); l.advance(); l.advance()
				break
			}
		} else {
			if l.peek() == quote {
				l.advance()
				break
			}
			if l.peek() == '\n' {
				l.addError(ErrUnterminatedString, line, col)
				break
			}
		}
		if l.peek() == '\\' {
			l.advance()
			esc := l.advance()
			switch esc {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '\\':
				buf.WriteByte('\\')
			case '"':
				buf.WriteByte('"')
			case '\'':
				buf.WriteByte('\'')
			case '0':
				buf.WriteByte(0)
			default:
				buf.WriteByte('\\')
				buf.WriteByte(esc)
			}
		} else {
			buf.WriteByte(l.advance())
		}
	}
	l.tokens = append(l.tokens, l.makeToken(TOK_STRING_LIT, buf.String(), line, col))
}

func (l *Lexer) lexNumber() {
	line, col := l.line, l.col
	var buf strings.Builder
	isFloat := false

	// hex
	if l.peek() == '0' && (l.peekAt(1) == 'x' || l.peekAt(1) == 'X') {
		buf.WriteByte(l.advance())
		buf.WriteByte(l.advance())
		for l.pos < len(l.source) && isHexDigit(l.peek()) {
			buf.WriteByte(l.advance())
		}
		l.tokens = append(l.tokens, l.makeToken(TOK_INT_LIT, buf.String(), line, col))
		return
	}

	for l.pos < len(l.source) && unicode.IsDigit(rune(l.peek())) {
		buf.WriteByte(l.advance())
	}
	if l.pos < len(l.source) && l.peek() == '.' && unicode.IsDigit(rune(l.peekAt(1))) {
		isFloat = true
		buf.WriteByte(l.advance())
		for l.pos < len(l.source) && unicode.IsDigit(rune(l.peek())) {
			buf.WriteByte(l.advance())
		}
	}
	if l.pos < len(l.source) && (l.peek() == 'e' || l.peek() == 'E') {
		isFloat = true
		buf.WriteByte(l.advance())
		if l.peek() == '+' || l.peek() == '-' {
			buf.WriteByte(l.advance())
		}
		for l.pos < len(l.source) && unicode.IsDigit(rune(l.peek())) {
			buf.WriteByte(l.advance())
		}
	}
	typ := TOK_INT_LIT
	if isFloat {
		typ = TOK_FLOAT_LIT
	}
	l.tokens = append(l.tokens, l.makeToken(typ, buf.String(), line, col))
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func (l *Lexer) lexIdent() {
	line, col := l.line, l.col
	var buf strings.Builder
	for l.pos < len(l.source) && (unicode.IsLetter(rune(l.peek())) || unicode.IsDigit(rune(l.peek())) || l.peek() == '_') {
		buf.WriteByte(l.advance())
	}
	word := buf.String()
	if tt, ok := keywords[word]; ok {
		l.tokens = append(l.tokens, l.makeToken(tt, word, line, col))
	} else {
		l.tokens = append(l.tokens, l.makeToken(TOK_IDENT, word, line, col))
	}
}

func (l *Lexer) lexOp() {
	line, col := l.line, l.col
	ch := l.advance()

	twoChar := string(ch)
	if l.pos < len(l.source) {
		twoChar += string(l.peek())
	}

	// Three-char
	threeChar := twoChar
	if l.pos+1 < len(l.source) {
		threeChar += string(l.source[l.pos+1])
	}
	_ = threeChar

	switch string(ch) {
	case "*":
		if l.peek() == '*' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				l.tokens = append(l.tokens, l.makeToken("**=", "**=", line, col))
			} else {
				l.tokens = append(l.tokens, l.makeToken(TOK_POWER, "**", line, col))
			}
		} else if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_STAR_EQ, "*=", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_STAR, "*", line, col))
		}
	case "/":
		if l.peek() == '/' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_FLOORDIV, "//", line, col))
		} else if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_SLASH_EQ, "/=", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_SLASH, "/", line, col))
		}
	case "+":
		if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_PLUS_EQ, "+=", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_PLUS, "+", line, col))
		}
	case "-":
		if l.peek() == '>' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_ARROW, "->", line, col))
		} else if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_MINUS_EQ, "-=", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_MINUS, "-", line, col))
		}
	case "=":
		if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_EQ, "==", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_ASSIGN, "=", line, col))
		}
	case "!":
		if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_NEQ, "!=", line, col))
		} else {
			l.addError(ErrUnexpectedChar, line, col, "!")
		}
	case "<":
		if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_LTE, "<=", line, col))
		} else if l.peek() == '<' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_LSHIFT, "<<", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_LT, "<", line, col))
		}
	case ">":
		if l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_GTE, ">=", line, col))
		} else if l.peek() == '>' {
			l.advance()
			l.tokens = append(l.tokens, l.makeToken(TOK_RSHIFT, ">>", line, col))
		} else {
			l.tokens = append(l.tokens, l.makeToken(TOK_GT, ">", line, col))
		}
	case "(":
		l.tokens = append(l.tokens, l.makeToken(TOK_LPAREN, "(", line, col))
	case ")":
		l.tokens = append(l.tokens, l.makeToken(TOK_RPAREN, ")", line, col))
	case "[":
		l.tokens = append(l.tokens, l.makeToken(TOK_LBRACKET, "[", line, col))
	case "]":
		l.tokens = append(l.tokens, l.makeToken(TOK_RBRACKET, "]", line, col))
	case "{":
		l.tokens = append(l.tokens, l.makeToken(TOK_LBRACE, "{", line, col))
	case "}":
		l.tokens = append(l.tokens, l.makeToken(TOK_RBRACE, "}", line, col))
	case ":":
		l.tokens = append(l.tokens, l.makeToken(TOK_COLON, ":", line, col))
	case ",":
		l.tokens = append(l.tokens, l.makeToken(TOK_COMMA, ",", line, col))
	case ".":
		l.tokens = append(l.tokens, l.makeToken(TOK_DOT, ".", line, col))
	case ";":
		l.tokens = append(l.tokens, l.makeToken(TOK_SEMICOL, ";", line, col))
	case "%":
		l.tokens = append(l.tokens, l.makeToken(TOK_PERCENT, "%", line, col))
	case "&":
		l.tokens = append(l.tokens, l.makeToken(TOK_AMP, "&", line, col))
	case "|":
		l.tokens = append(l.tokens, l.makeToken(TOK_PIPE, "|", line, col))
	case "^":
		l.tokens = append(l.tokens, l.makeToken(TOK_CARET, "^", line, col))
	case "~":
		l.tokens = append(l.tokens, l.makeToken(TOK_TILDE, "~", line, col))
	case "@":
		l.tokens = append(l.tokens, l.makeToken(TOK_AT, "@", line, col))
	default:
		l.addError(ErrUnexpectedChar, line, col, string(ch))
	}
}
