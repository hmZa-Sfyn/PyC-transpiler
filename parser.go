package main

import (
	"strconv"
	"strings"
)

// ─── Parser ───────────────────────────────────────────────────────────────────

type Parser struct {
	tokens []Token
	pos    int
	errors []PycError
	file   string
}

func NewParser(tokens []Token, file string) *Parser {
	return &Parser{tokens: tokens, file: file}
}

// ── Token navigation ──────────────────────────────────────────────────────────

// peek returns the next meaningful token WITHOUT advancing pos.
// Uses a local index so p.pos is never mutated.
func (p *Parser) peek() Token {
	i := p.pos
	for i < len(p.tokens) {
		t := p.tokens[i]
		if t.Type == TOK_NEWLINE || t.Type == TOK_SEMICOL {
			i++
			continue
		}
		return t
	}
	return Token{Type: TOK_EOF}
}

// advance skips leading newlines/semicolons, then consumes and returns the
// next real token, advancing p.pos past it.
func (p *Parser) advance() Token {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type == TOK_NEWLINE || t.Type == TOK_SEMICOL {
			p.pos++
			continue
		}
		p.pos++
		return t
	}
	return Token{Type: TOK_EOF}
}

// skipNewlines advances p.pos past any NEWLINE/SEMICOLON tokens.
func (p *Parser) skipNewlines() {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type == TOK_NEWLINE || t.Type == TOK_SEMICOL {
			p.pos++
		} else {
			break
		}
	}
}

// matchRaw skips newlines then checks if the raw token at p.pos matches tt.
// If it does, consumes it and returns true.
// Used specifically for INDENT/DEDENT which live directly in the raw token stream.
func (p *Parser) matchRaw(tt TokenType) bool {
	p.skipNewlines()
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == tt {
		p.pos++
		return true
	}
	return false
}

// check returns true if the next meaningful token has type tt.
func (p *Parser) check(tt TokenType) bool {
	return p.peek().Type == tt
}

// match consumes the next meaningful token if it matches tt.
func (p *Parser) match(tt TokenType) bool {
	if p.check(tt) {
		p.advance()
		return true
	}
	return false
}

// expect consumes the next token; emits an error if it doesn't match.
func (p *Parser) expect(tt TokenType) Token {
	t := p.peek()
	if t.Type != tt {
		p.addError(ErrExpectedToken, t.Line, t.Col, t.LineStr, string(tt), t.Value)
		return t
	}
	return p.advance()
}

func (p *Parser) addError(code ErrorCode, line, col int, lineStr string, args ...interface{}) {
	p.errors = append(p.errors, newError(code, line, col, lineStr, args...))
}

// ─── Program ──────────────────────────────────────────────────────────────────

func (p *Parser) ParseProgram() *Program {
	prog := &Program{}
	p.skipNewlines()
	for !p.check(TOK_EOF) {
		prevPos := p.pos
		stmt := p.parseTopLevelStmt()
		if stmt != nil {
			prog.Stmts = append(prog.Stmts, stmt)
		}
		p.skipNewlines()
		// Safety: if nothing was consumed, skip one raw token to avoid infinite loop
		if p.pos == prevPos {
			p.pos++
		}
	}
	return prog
}

func (p *Parser) parseTopLevelStmt() Node {
	t := p.peek()
	switch t.Type {
	case TOK_DEF:
		return p.parseFuncDef()
	case TOK_STRUCT:
		return p.parseStructDef()
	case TOK_IMPORT, TOK_FROM:
		return p.parseImport()
	default:
		return p.parseStmt()
	}
}

// ─── Block ────────────────────────────────────────────────────────────────────
// Blocks are delimited by INDENT/DEDENT tokens in the raw stream.

func (p *Parser) parseBlock(context string) []Node {
	// matchRaw skips newlines then looks for INDENT in raw stream
	if !p.matchRaw(TOK_INDENT) {
		t := p.peek()
		p.addError(ErrExpectedIndent, t.Line, t.Col, t.LineStr, context)
		return nil
	}

	var stmts []Node
	for {
		p.skipNewlines()
		// Check raw stream for DEDENT (end of block)
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TOK_DEDENT {
			p.pos++ // consume DEDENT
			break
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TOK_EOF {
			break
		}
		prevPos := p.pos
		stmt := p.parseStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		// Safety guard: if parseStmt consumed nothing, skip one raw token
		if p.pos == prevPos {
			p.pos++
		}
	}

	if len(stmts) == 0 {
		t := p.peek()
		p.addError(ErrEmptyBody, t.Line, t.Col, t.LineStr)
	}
	return stmts
}

// ─── Function Definition ──────────────────────────────────────────────────────

func (p *Parser) parseFuncDef() *FuncDef {
	tok := p.advance() // consume 'def'
	nameTok := p.peek()
	if nameTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, nameTok.Line, nameTok.Col, nameTok.LineStr, nameTok.Value)
	} else {
		p.advance()
	}

	fd := &FuncDef{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Name: nameTok.Value}

	p.expect(TOK_LPAREN)
	fd.Params = p.parseParams()
	p.expect(TOK_RPAREN)

	// optional return type: -> type
	if p.check(TOK_ARROW) {
		p.advance()
		fd.ReturnType = p.parseTypeExpr()
	} else {
		fd.ReturnType = TypVoid
	}

	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "function definition")
	}

	fd.Body = p.parseBlock("function")
	return fd
}

func (p *Parser) parseParams() []Param {
	var params []Param
	if p.check(TOK_RPAREN) {
		return params
	}
	for {
		if p.check(TOK_RPAREN) || p.check(TOK_EOF) {
			break
		}
		param := p.parseOneParam()
		params = append(params, param)
		if !p.match(TOK_COMMA) {
			break
		}
	}
	return params
}

func (p *Parser) parseOneParam() Param {
	t := p.peek()
	if t.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, t.Line, t.Col, t.LineStr, t.Value)
		return Param{Name: "_", Type: TypAny}
	}
	p.advance()
	param := Param{Name: t.Value, Line: t.Line, Col: t.Col}

	if p.match(TOK_COLON) {
		param.Type = p.parseTypeExpr()
	} else {
		param.Type = TypAny
	}

	if p.match(TOK_ASSIGN) {
		param.Default = p.parseExpr()
	}
	return param
}

// ─── Struct Definition ────────────────────────────────────────────────────────

func (p *Parser) parseStructDef() *StructDef {
	tok := p.advance() // consume 'struct'
	nameTok := p.peek()
	name := nameTok.Value
	if nameTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, nameTok.Line, nameTok.Col, nameTok.LineStr, nameTok.Value)
	} else {
		p.advance()
	}
	sd := &StructDef{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Name: name}

	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "struct definition")
	}

	if !p.matchRaw(TOK_INDENT) {
		p.addError(ErrExpectedIndent, p.peek().Line, p.peek().Col, p.peek().LineStr, "struct definition")
		return sd
	}
	for {
		p.skipNewlines()
		if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TOK_DEDENT {
			p.pos++
			break
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TOK_EOF {
			break
		}
		ft := p.peek()
		if ft.Type != TOK_IDENT {
			p.addError(ErrExpectedIdent, ft.Line, ft.Col, ft.LineStr, ft.Value)
			p.advance()
			continue
		}
		p.advance()
		fieldName := ft.Value
		p.expect(TOK_COLON)
		fieldType := p.parseTypeExpr()
		sd.Fields = append(sd.Fields, StructField{Name: fieldName, Type: fieldType, Line: ft.Line, Col: ft.Col})
	}
	return sd
}

// ─── Import ───────────────────────────────────────────────────────────────────

func (p *Parser) parseImport() Node {
	tok := p.advance()
	imp := &ImportStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	if tok.Type == TOK_FROM {
		modTok := p.peek()
		imp.Module = modTok.Value
		p.advance()
		p.expect(TOK_IMPORT)
		// Consume rest of line using raw scan
		for p.pos < len(p.tokens) {
			tt := p.tokens[p.pos].Type
			if tt == TOK_NEWLINE || tt == TOK_EOF || tt == TOK_DEDENT {
				break
			}
			p.pos++
		}
	} else {
		modTok := p.peek()
		imp.Module = modTok.Value
		p.advance()
		if p.match(TOK_AS) {
			imp.Alias = p.peek().Value
			p.advance()
		}
	}
	return imp
}

// ─── Statement ────────────────────────────────────────────────────────────────

func (p *Parser) parseStmt() Node {
	p.skipNewlines()
	t := p.peek()
	switch t.Type {
	case TOK_DEF:
		return p.parseFuncDef()
	case TOK_STRUCT:
		return p.parseStructDef()
	case TOK_RETURN:
		return p.parseReturn()
	case TOK_IF:
		return p.parseIf()
	case TOK_WHILE:
		return p.parseWhile()
	case TOK_FOR:
		return p.parseFor()
	case TOK_BREAK:
		p.advance()
		return &BreakStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
	case TOK_CONT:
		p.advance()
		return &ContinueStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
	case TOK_PASS:
		p.advance()
		return &PassStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
	case TOK_GLOBAL:
		return p.parseGlobal()
	case TOK_DELETE:
		return p.parseDelete()
	case TOK_IMPORT, TOK_FROM:
		return p.parseImport()
	case TOK_EOF, TOK_DEDENT:
		return nil
	// ── Smalltalk keywords ────────────────────────────────────────────────────
	case TOK_IFTRUE:
		return p.parseIfTrue(false)
	case TOK_IFFALSE, TOK_UNLESS:
		return p.parseIfTrue(true)
	case TOK_REPEAT, TOK_TIMES:
		return p.parseRepeat()
	case TOK_EACH:
		return p.parseEach()
	case TOK_LOOP, TOK_FOREVER:
		return p.parseLoop()
	case TOK_UNTIL:
		return p.parseUntil()
	case TOK_SWAP:
		return p.parseSwap()
	case TOK_DEFAULT:
		return p.parseDefault()
	case TOK_CHECK:
		return p.parseCheck()
	case TOK_DIE:
		return p.parseDie()
	case TOK_MAYBE:
		return p.parseMaybe()
	case TOK_PRINTBANG:
		return p.parsePrintBang()
	case TOK_DONE:
		p.advance()
		return &BreakStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
	case TOK_SKIP:
		p.advance()
		return &ContinueStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
	case TOK_RET, TOK_GIVE:
		p.advance()
		rs := &ReturnStmt{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}
		if p.pos < len(p.tokens) {
			rawNext := p.tokens[p.pos].Type
			if rawNext != TOK_NEWLINE && rawNext != TOK_SEMICOL && rawNext != TOK_EOF && rawNext != TOK_DEDENT {
				rs.Value = p.parseExpr()
			}
		}
		return rs
	default:
		return p.parseExprOrAssign()
	}
}

func (p *Parser) parseReturn() *ReturnStmt {
	tok := p.advance()
	rs := &ReturnStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	// Check the raw stream for end-of-statement markers
	if p.pos >= len(p.tokens) {
		return rs
	}
	rawNext := p.tokens[p.pos].Type
	if rawNext == TOK_NEWLINE || rawNext == TOK_SEMICOL ||
		rawNext == TOK_EOF || rawNext == TOK_DEDENT {
		return rs
	}
	rs.Value = p.parseExpr()
	return rs
}

func (p *Parser) parseIf() *IfStmt {
	tok := p.advance() // consume 'if'
	is := &IfStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	is.Cond = p.parseExpr()

	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "if statement")
	}
	is.Then = p.parseBlock("if")

	for p.check(TOK_ELIF) {
		elifTok := p.advance()
		cond := p.parseExpr()
		ct := p.peek()
		if !p.match(TOK_COLON) {
			p.addError(ErrExpectedColon, ct.Line, ct.Col, ct.LineStr, "elif clause")
		}
		body := p.parseBlock("elif")
		is.Elifs = append(is.Elifs, ElifClause{Cond: cond, Body: body, Line: elifTok.Line})
	}

	if p.check(TOK_ELSE) {
		p.advance()
		ct := p.peek()
		if !p.match(TOK_COLON) {
			p.addError(ErrExpectedColon, ct.Line, ct.Col, ct.LineStr, "else clause")
		}
		is.ElseBody = p.parseBlock("else")
	}
	return is
}

func (p *Parser) parseWhile() *WhileStmt {
	tok := p.advance()
	ws := &WhileStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	ws.Cond = p.parseExpr()
	ct := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, ct.Line, ct.Col, ct.LineStr, "while loop")
	}
	ws.Body = p.parseBlock("while")
	return ws
}

func (p *Parser) parseFor() *ForStmt {
	tok := p.advance() // consume 'for'
	fs := &ForStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}

	varTok := p.peek()
	if varTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, varTok.Line, varTok.Col, varTok.LineStr, varTok.Value)
	} else {
		p.advance()
	}
	fs.Var = varTok.Value

	// optional type annotation:  for x: int in ...
	if p.match(TOK_COLON) {
		fs.VarType = p.parseTypeExpr()
	}

	inTok := p.peek()
	if !p.match(TOK_IN) {
		p.addError(ErrExpectedIn, inTok.Line, inTok.Col, inTok.LineStr, inTok.Value)
	}

	fs.Iter = p.parseExpr()

	ct := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, ct.Line, ct.Col, ct.LineStr, "for loop")
	}
	fs.Body = p.parseBlock("for")
	return fs
}

func (p *Parser) parseGlobal() *GlobalStmt {
	tok := p.advance()
	gs := &GlobalStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	nt := p.peek()
	if nt.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, nt.Line, nt.Col, nt.LineStr, nt.Value)
		return gs
	}
	p.advance()
	gs.Names = append(gs.Names, nt.Value)
	for p.match(TOK_COMMA) {
		nt = p.peek()
		if nt.Type != TOK_IDENT {
			break
		}
		p.advance()
		gs.Names = append(gs.Names, nt.Value)
	}
	return gs
}

func (p *Parser) parseDelete() *DeleteStmt {
	tok := p.advance()
	ds := &DeleteStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	ds.Target = p.parseExpr()
	return ds
}

func (p *Parser) parseExprOrAssign() Node {
	expr := p.parseExpr()
	if expr == nil {
		return nil
	}

	t := p.peek()
	switch t.Type {
	case TOK_ASSIGN, TOK_PLUS_EQ, TOK_MINUS_EQ, TOK_STAR_EQ, TOK_SLASH_EQ:
		p.advance()
		val := p.parseExpr()
		return &AssignStmt{
			BaseNode: BaseNode{Line: t.Line, Col: t.Col},
			Target:   expr,
			Op:       string(t.Type),
			Value:    val,
		}
	case TOK_COLON:
		// typed declaration:  name: type = value
		if id, ok := expr.(*Ident); ok {
			p.advance() // consume ':'
			typ := p.parseTypeExpr()
			var val Node
			if p.match(TOK_ASSIGN) {
				val = p.parseExpr()
			}
			return &VarDecl{
				BaseNode: BaseNode{Line: id.Line, Col: id.Col},
				Name:     id.Name,
				Type:     typ,
				Value:    val,
			}
		}
	}

	return &ExprStmt{BaseNode: BaseNode{Line: expr.GetLine(), Col: expr.GetCol()}, Expr: expr}
}

// ─── Type Expression ──────────────────────────────────────────────────────────

func (p *Parser) parseTypeExpr() *Type {
	t := p.peek()
	switch t.Type {
	case TOK_TYPE_INT:
		p.advance()
		return TypInt
	case TOK_TYPE_FLOAT:
		p.advance()
		return TypFloat
	case TOK_TYPE_STR:
		p.advance()
		return TypStr
	case TOK_TYPE_BOOL:
		p.advance()
		return TypBool
	case TOK_TYPE_VOID:
		p.advance()
		return TypVoid
	case TOK_TYPE_ANY:
		p.advance()
		return TypAny
	case TOK_TYPE_LIST:
		p.advance()
		if p.match(TOK_LBRACKET) {
			elem := p.parseTypeExpr()
			p.expect(TOK_RBRACKET)
			return ListType(elem)
		}
		return ListType(TypAny)
	case TOK_IDENT:
		p.advance()
		return StructType(t.Value)
	default:
		p.addError(ErrExpectedType, t.Line, t.Col, t.LineStr, t.Value)
		return TypAny
	}
}

// ─── Expressions (Pratt-style precedence climbing) ────────────────────────────

func (p *Parser) parseExpr() Node {
	return p.parseTernary()
}

func (p *Parser) parseTernary() Node {
	expr := p.parseOr()
	// Python ternary:  value if condition else other
	// Only treat 'if' as ternary if we can find 'else' ahead before a colon/newline.
	// A bare colon means this 'if' is an if-statement, not a ternary expression.
	if p.check(TOK_IF) && p.hasTernaryElse() {
		p.advance() // consume 'if'
		cond := p.parseOr()
		p.expect(TOK_ELSE)
		other := p.parseTernary()
		return &TernaryExpr{
			BaseNode: BaseNode{Line: expr.GetLine(), Col: expr.GetCol()},
			Then:     expr,
			Cond:     cond,
			Else:     other,
		}
	}
	return expr
}

// hasTernaryElse scans ahead to decide if the upcoming 'if' is a ternary.
// Returns true only if 'else' appears before a statement-ending colon or newline.
func (p *Parser) hasTernaryElse() bool {
	depth := 0
	for i := p.pos; i < len(p.tokens); i++ {
		tt := p.tokens[i].Type
		switch tt {
		case TOK_LPAREN, TOK_LBRACKET, TOK_LBRACE:
			depth++
		case TOK_RPAREN, TOK_RBRACKET, TOK_RBRACE:
			depth--
		case TOK_ELSE:
			if depth == 0 {
				return true
			}
		case TOK_COLON:
			if depth == 0 {
				return false
			}
		case TOK_NEWLINE, TOK_DEDENT, TOK_EOF:
			return false
		}
	}
	return false
}

func (p *Parser) parseOr() Node {
	left := p.parseAnd()
	for p.check(TOK_OR) {
		op := p.advance()
		right := p.parseAnd()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "||", Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseAnd() Node {
	left := p.parseNot()
	for p.check(TOK_AND) {
		op := p.advance()
		right := p.parseNot()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "&&", Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseNot() Node {
	if p.check(TOK_NOT) {
		op := p.advance()
		operand := p.parseNot()
		return &UnaryOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "!", Operand: operand}
	}
	return p.parseComparison()
}

func (p *Parser) parseComparison() Node {
	left := p.parseBitOr()
	for {
		t := p.peek()
		var op string
		switch t.Type {
		case TOK_EQ:
			op = "=="
		case TOK_NEQ:
			op = "!="
		case TOK_LT:
			op = "<"
		case TOK_GT:
			op = ">"
		case TOK_LTE:
			op = "<="
		case TOK_GTE:
			op = ">="
		case TOK_IS:
			op = "=="
		case TOK_IN:
			// x in collection  →  collection.contains(x)
			p.advance()
			right := p.parseBitOr()
			left = &CallExpr{
				BaseNode: BaseNode{Line: t.Line, Col: t.Col},
				Func:     &AttrExpr{Obj: right, Attr: "contains"},
				Args:     []Node{left},
			}
			continue
		default:
			return left
		}
		p.advance()
		right := p.parseBitOr()
		left = &BinOp{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Op: op, Left: left, Right: right}
	}
}

func (p *Parser) parseBitOr() Node {
	left := p.parseBitXor()
	for p.check(TOK_PIPE) {
		op := p.advance()
		right := p.parseBitXor()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "|", Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseBitXor() Node {
	left := p.parseBitAnd()
	for p.check(TOK_CARET) {
		op := p.advance()
		right := p.parseBitAnd()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "^", Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseBitAnd() Node {
	left := p.parseShift()
	for p.check(TOK_AMP) {
		op := p.advance()
		right := p.parseShift()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "&", Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseShift() Node {
	left := p.parseAddSub()
	for p.check(TOK_LSHIFT) || p.check(TOK_RSHIFT) {
		op := p.advance()
		right := p.parseAddSub()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: op.Value, Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseAddSub() Node {
	left := p.parseMulDiv()
	for p.check(TOK_PLUS) || p.check(TOK_MINUS) {
		op := p.advance()
		right := p.parseMulDiv()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: op.Value, Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseMulDiv() Node {
	left := p.parseUnary()
	for p.check(TOK_STAR) || p.check(TOK_SLASH) || p.check(TOK_PERCENT) || p.check(TOK_FLOORDIV) {
		op := p.advance()
		opStr := op.Value
		if opStr == "//" {
			opStr = "FLOORDIV"
		}
		right := p.parseUnary()
		left = &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: opStr, Left: left, Right: right}
	}
	return left
}

func (p *Parser) parseUnary() Node {
	t := p.peek()
	switch t.Type {
	case TOK_MINUS:
		p.advance()
		operand := p.parseUnary()
		return &UnaryOp{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Op: "-", Operand: operand}
	case TOK_PLUS:
		p.advance()
		return p.parseUnary()
	case TOK_TILDE:
		p.advance()
		operand := p.parseUnary()
		return &UnaryOp{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Op: "~", Operand: operand}
	}
	return p.parsePower()
}

func (p *Parser) parsePower() Node {
	base := p.parsePostfix()
	if p.check(TOK_POWER) {
		op := p.advance()
		exp := p.parseUnary() // right-associative
		return &BinOp{BaseNode: BaseNode{Line: op.Line, Col: op.Col}, Op: "**", Left: base, Right: exp}
	}
	return base
}

func (p *Parser) parsePostfix() Node {
	expr := p.parsePrimary()
	for {
		t := p.peek()
		switch t.Type {
		case TOK_LPAREN:
			expr = p.parseCall(expr)
		case TOK_LBRACKET:
			expr = p.parseIndex(expr)
		case TOK_DOT:
			p.advance()
			attrTok := p.peek()
			if attrTok.Type != TOK_IDENT {
				p.addError(ErrExpectedIdent, attrTok.Line, attrTok.Col, attrTok.LineStr, attrTok.Value)
				return expr
			}
			p.advance()
			expr = &AttrExpr{
				BaseNode: BaseNode{Line: t.Line, Col: t.Col},
				Obj:      expr,
				Attr:     attrTok.Value,
			}
		default:
			return expr
		}
	}
}

func (p *Parser) parseCall(fn Node) *CallExpr {
	tok := p.advance() // consume '('
	call := &CallExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Func: fn}

	for !p.check(TOK_RPAREN) && !p.check(TOK_EOF) {
		// Keyword argument detection: look for   ident =
		// We need to scan the meaningful token stream: next token is IDENT
		// and the one after is '=' (not '==').
		nextTok := p.peek()
		if nextTok.Type == TOK_IDENT {
			// Find position of this IDENT in the raw stream
			identPos := -1
			for scan := p.pos; scan < len(p.tokens); scan++ {
				tt := p.tokens[scan].Type
				if tt == TOK_NEWLINE || tt == TOK_SEMICOL {
					continue
				}
				identPos = scan
				break
			}
			// Find the token after the IDENT
			if identPos >= 0 && identPos+1 < len(p.tokens) {
				afterPos := identPos + 1
				for afterPos < len(p.tokens) {
					tt := p.tokens[afterPos].Type
					if tt == TOK_NEWLINE || tt == TOK_SEMICOL {
						afterPos++
						continue
					}
					break
				}
				if afterPos < len(p.tokens) && p.tokens[afterPos].Type == TOK_ASSIGN {
					nameTok := p.advance() // consume IDENT
					p.advance()            // consume '='
					val := p.parseExpr()
					call.KwArgs = append(call.KwArgs, KwArg{Name: nameTok.Value, Value: val})
					if !p.match(TOK_COMMA) {
						break
					}
					continue
				}
			}
		}
		arg := p.parseExpr()
		call.Args = append(call.Args, arg)
		if !p.match(TOK_COMMA) {
			break
		}
	}
	t := p.peek()
	if !p.match(TOK_RPAREN) {
		p.addError(ErrExpectedRParen, t.Line, t.Col, t.LineStr, t.Value)
	}
	return call
}

func (p *Parser) parseIndex(obj Node) Node {
	tok := p.advance() // consume '['
	var low, high, step Node
	isSlice := false

	if !p.check(TOK_COLON) {
		low = p.parseExpr()
	}
	if p.match(TOK_COLON) {
		isSlice = true
		if !p.check(TOK_RBRACKET) && !p.check(TOK_COLON) {
			high = p.parseExpr()
		}
		if p.match(TOK_COLON) {
			if !p.check(TOK_RBRACKET) {
				step = p.parseExpr()
			}
		}
	}

	t := p.peek()
	if !p.match(TOK_RBRACKET) {
		p.addError(ErrExpectedRBracket, t.Line, t.Col, t.LineStr, t.Value)
	}

	if isSlice {
		return &SliceExpr{
			BaseNode: BaseNode{Line: tok.Line, Col: tok.Col},
			Obj:      obj,
			Low:      low,
			High:     high,
			Step:     step,
		}
	}
	return &IndexExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Obj: obj, Index: low}
}

func (p *Parser) parsePrimary() Node {
	t := p.peek()
	switch t.Type {
	case TOK_INT_LIT:
		p.advance()
		v, err := strconv.ParseInt(t.Value, 0, 64)
		if err != nil {
			p.addError(ErrInvalidNumber, t.Line, t.Col, t.LineStr, t.Value)
		}
		return &IntLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: v, Raw: t.Value}

	case TOK_FLOAT_LIT:
		p.advance()
		v, err := strconv.ParseFloat(t.Value, 64)
		if err != nil {
			p.addError(ErrInvalidNumber, t.Line, t.Col, t.LineStr, t.Value)
		}
		return &FloatLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: v, Raw: t.Value}

	case TOK_STRING_LIT:
		p.advance()
		return &StringLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: t.Value}

	case TOK_BOOL_LIT:
		p.advance()
		return &BoolLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: t.Value == "True"}

	case TOK_NONE:
		p.advance()
		return &NoneLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}}

	case TOK_IDENT:
		p.advance()
		// f-string: f"..."
		if t.Value == "f" && p.check(TOK_STRING_LIT) {
			strTok := p.advance()
			return p.parseFString(strTok)
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: t.Value}

	case TOK_LPAREN:
		p.advance()
		expr := p.parseExpr()
		p.expect(TOK_RPAREN)
		return expr

	case TOK_LBRACKET:
		return p.parseListLit()

	case TOK_LBRACE:
		return p.parseDictLit()

	// Built-in type keywords used as cast functions
	case TOK_TYPE_INT:
		p.advance()
		if p.check(TOK_LPAREN) {
			return p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "int"})
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "int"}

	case TOK_TYPE_FLOAT:
		p.advance()
		if p.check(TOK_LPAREN) {
			return p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "float"})
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "float"}

	case TOK_TYPE_STR:
		p.advance()
		if p.check(TOK_LPAREN) {
			return p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "str"})
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "str"}

	case TOK_TYPE_BOOL:
		p.advance()
		if p.check(TOK_LPAREN) {
			return p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "bool"})
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "bool"}

	case TOK_NEW:
		return p.parseNew()
	case TOK_CLAMP:
		return p.parseClamp()
	case TOK_BETWEEN:
		return p.parseBetween()
	case TOK_EITHER:
		return p.parseEither()

	default:
		p.addError(ErrExpectedExpr, t.Line, t.Col, t.LineStr, t.Value)
		p.advance()
		return &IntLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: 0}
	}
}

func (p *Parser) parseListLit() *ListLit {
	tok := p.advance() // consume '['
	ll := &ListLit{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	for !p.check(TOK_RBRACKET) && !p.check(TOK_EOF) {
		ll.Elems = append(ll.Elems, p.parseExpr())
		if !p.match(TOK_COMMA) {
			break
		}
	}
	t := p.peek()
	if !p.match(TOK_RBRACKET) {
		p.addError(ErrExpectedRBracket, t.Line, t.Col, t.LineStr, t.Value)
	}
	return ll
}

func (p *Parser) parseDictLit() *DictLit {
	tok := p.advance() // consume '{'
	dl := &DictLit{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	for !p.check(TOK_RBRACE) && !p.check(TOK_EOF) {
		k := p.parseExpr()
		p.expect(TOK_COLON)
		v := p.parseExpr()
		dl.Keys = append(dl.Keys, k)
		dl.Vals = append(dl.Vals, v)
		if !p.match(TOK_COMMA) {
			break
		}
	}
	t := p.peek()
	if !p.match(TOK_RBRACE) {
		p.addError(ErrExpectedRBrace, t.Line, t.Col, t.LineStr, t.Value)
	}
	return dl
}

func (p *Parser) parseNew() *NewExpr {
	tok := p.advance() // consume 'new'
	nameTok := p.peek()
	name := nameTok.Value
	if nameTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, nameTok.Line, nameTok.Col, nameTok.LineStr, nameTok.Value)
	} else {
		p.advance()
	}
	ne := &NewExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, TypeName: name}
	if p.match(TOK_LPAREN) {
		for !p.check(TOK_RPAREN) && !p.check(TOK_EOF) {
			ne.Args = append(ne.Args, p.parseExpr())
			if !p.match(TOK_COMMA) {
				break
			}
		}
		p.expect(TOK_RPAREN)
	}
	return ne
}

// ─── F-String ─────────────────────────────────────────────────────────────────

func (p *Parser) parseFString(strTok Token) *FStringExpr {
	fs := &FStringExpr{BaseNode: BaseNode{Line: strTok.Line, Col: strTok.Col}}
	raw := strTok.Value
	i := 0
	for i < len(raw) {
		if raw[i] == '{' {
			if i+1 < len(raw) && raw[i+1] == '{' {
				fs.Parts = append(fs.Parts, FStringPart{IsExpr: false, Text: "{"})
				i += 2
				continue
			}
			end := strings.Index(raw[i:], "}")
			if end < 0 {
				p.addError(ErrInvalidFString, strTok.Line, strTok.Col, strTok.LineStr)
				break
			}
			exprStr := raw[i+1 : i+end]
			subLex := NewLexer(exprStr)
			toks, _ := subLex.Tokenize()
			subParser := NewParser(toks, p.file)
			expr := subParser.parseExpr()
			fs.Parts = append(fs.Parts, FStringPart{IsExpr: true, Expr: expr})
			i += end + 1
		} else if raw[i] == '}' && i+1 < len(raw) && raw[i+1] == '}' {
			fs.Parts = append(fs.Parts, FStringPart{IsExpr: false, Text: "}"})
			i += 2
		} else {
			start := i
			for i < len(raw) && raw[i] != '{' && raw[i] != '}' {
				i++
			}
			fs.Parts = append(fs.Parts, FStringPart{IsExpr: false, Text: raw[start:i]})
		}
	}
	return fs
}

// Expose parse errors
func (p *Parser) Errors() []PycError { return p.errors }

// ─── Smalltalk / One-liner Statement Parsers ──────────────────────────────────

func (p *Parser) parseInlineBody() Node {
	// For one-liner keywords: parse a single statement on the same line.
	// After the colon we expect a statement (not an indented block).
	p.skipNewlines()
	return p.parseStmt()
}

func (p *Parser) parseIfTrue(negated bool) *IfTrueStmt {
	tok := p.advance() // consume iftrue/iffalse/unless
	node := &IfTrueStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Negated: negated}
	node.Cond = p.parseExpr()
	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "iftrue/iffalse")
	}
	node.Body = p.parseInlineBody()
	return node
}

func (p *Parser) parseRepeat() *RepeatStmt {
	tok := p.advance() // consume repeat/times
	node := &RepeatStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Count = p.parseExpr()
	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "repeat")
	}
	node.Body = p.parseInlineBody()
	return node
}

func (p *Parser) parseEach() *EachStmt {
	tok := p.advance() // consume 'each'
	node := &EachStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	varTok := p.peek()
	if varTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, varTok.Line, varTok.Col, varTok.LineStr, varTok.Value)
	} else {
		p.advance()
	}
	node.Var = varTok.Value
	inTok := p.peek()
	if !p.match(TOK_IN) {
		p.addError(ErrExpectedIn, inTok.Line, inTok.Col, inTok.LineStr, inTok.Value)
	}
	node.Iter = p.parseExpr()
	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "each")
	}
	node.Body = p.parseInlineBody()
	return node
}

func (p *Parser) parseLoop() *LoopStmt {
	tok := p.advance() // consume loop/forever
	node := &LoopStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "loop")
	}
	node.Body = p.parseBlock("loop")
	return node
}

func (p *Parser) parseUntil() *UntilStmt {
	tok := p.advance() // consume 'until'
	node := &UntilStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Cond = p.parseExpr()
	colonTok := p.peek()
	if !p.match(TOK_COLON) {
		p.addError(ErrExpectedColon, colonTok.Line, colonTok.Col, colonTok.LineStr, "until")
	}
	node.Body = p.parseBlock("until")
	return node
}

func (p *Parser) parseSwap() *SwapStmt {
	tok := p.advance() // consume 'swap'
	node := &SwapStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.A = p.parseExpr()
	if !p.match(TOK_COMMA) {
		p.addError(ErrExpectedComma, tok.Line, tok.Col, tok.LineStr, p.peek().Value)
	}
	node.B = p.parseExpr()
	return node
}

func (p *Parser) parseDefault() *DefaultStmt {
	tok := p.advance() // consume 'default'
	node := &DefaultStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Target = p.parseExpr()
	if !p.match(TOK_ASSIGN) {
		p.addError(ErrExpectedToken, tok.Line, tok.Col, tok.LineStr, "=", p.peek().Value)
	}
	node.Value = p.parseExpr()
	return node
}

func (p *Parser) parseCheck() *CheckStmt {
	tok := p.advance()
	node := &CheckStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Expr = p.parseExpr()
	return node
}

func (p *Parser) parseDie() *DieStmt {
	tok := p.advance()
	node := &DieStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Msg = p.parseExpr()
	return node
}

func (p *Parser) parseMaybe() *MaybeStmt {
	tok := p.advance()
	node := &MaybeStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Expr = p.parseExpr()
	return node
}

func (p *Parser) parsePrintBang() *PrintBangStmt {
	tok := p.advance() // consume 'print!'
	node := &PrintBangStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	// collect comma-separated args until end of line
	for {
		if p.pos >= len(p.tokens) { break }
		rawTT := p.tokens[p.pos].Type
		if rawTT == TOK_NEWLINE || rawTT == TOK_DEDENT || rawTT == TOK_EOF { break }
		arg := p.parseExpr()
		node.Args = append(node.Args, arg)
		if !p.match(TOK_COMMA) { break }
	}
	return node
}

func (p *Parser) parseClamp() *ClampExpr {
	tok := p.advance() // consume 'clamp'
	node := &ClampExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Val = p.parseExpr()
	p.expect(TOK_COMMA)
	node.Lo = p.parseExpr()
	p.expect(TOK_COMMA)
	node.Hi = p.parseExpr()
	return node
}

func (p *Parser) parseBetween() *BetweenExpr {
	tok := p.advance() // consume 'between'
	node := &BetweenExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.Val = p.parseExpr()
	p.expect(TOK_COMMA)
	node.Lo = p.parseExpr()
	p.expect(TOK_COMMA)
	node.Hi = p.parseExpr()
	return node
}

func (p *Parser) parseEither() *EitherExpr {
	tok := p.advance() // consume 'either'
	node := &EitherExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	node.A = p.parseExpr()
	// expect 'or'
	if !p.match(TOK_OR) {
		p.addError(ErrExpectedToken, tok.Line, tok.Col, tok.LineStr, "or", p.peek().Value)
	}
	node.B = p.parseExpr()
	return node
}
