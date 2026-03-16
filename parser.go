package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ─── Parser ───────────────────────────────────────────────────────────────────

type Parser struct {
	tokens  []Token
	pos     int
	errors  []PycError
	source  string
	file    string
}

func NewParser(tokens []Token, file string) *Parser {
	return &Parser{tokens: tokens, file: file}
}

func (p *Parser) peek() Token {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type == TOK_NEWLINE || t.Type == TOK_SEMICOL {
			p.pos++
			continue
		}
		return t
	}
	return Token{Type: TOK_EOF}
}

func (p *Parser) peekRaw() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: TOK_EOF}
}

func (p *Parser) advance() Token {
	t := p.tokens[p.pos]
	p.pos++
	return t
}

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

func (p *Parser) expect(tt TokenType) Token {
	t := p.peek()
	if t.Type != tt {
		p.addError(ErrExpectedToken, t.Line, t.Col, t.LineStr, string(tt), t.Value)
		return t
	}
	return p.advance()
}

func (p *Parser) check(tt TokenType) bool {
	return p.peek().Type == tt
}

func (p *Parser) match(tt TokenType) bool {
	if p.check(tt) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) addError(code ErrorCode, line, col int, lineStr string, args ...interface{}) {
	p.errors = append(p.errors, newError(code, line, col, lineStr, args...))
}

func (p *Parser) addErrorLen(code ErrorCode, line, col, length int, lineStr string, args ...interface{}) {
	p.errors = append(p.errors, newErrorLen(code, line, col, length, lineStr, args...))
}

// ─── Program ──────────────────────────────────────────────────────────────────

func (p *Parser) ParseProgram() *Program {
	prog := &Program{}
	p.skipNewlines()
	for !p.check(TOK_EOF) {
		stmt := p.parseTopLevelStmt()
		if stmt != nil {
			prog.Stmts = append(prog.Stmts, stmt)
		}
		p.skipNewlines()
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

// ─── Function Def ─────────────────────────────────────────────────────────────

func (p *Parser) parseFuncDef() *FuncDef {
	tok := p.advance() // 'def'
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

	// optional return type
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

// ─── Struct Def ───────────────────────────────────────────────────────────────

func (p *Parser) parseStructDef() *StructDef {
	tok := p.advance() // 'struct'
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

	// parse indented fields
	p.skipNewlines()
	if !p.match(TOK_INDENT) {
		p.addError(ErrExpectedIndent, p.peek().Line, p.peek().Col, p.peek().LineStr, "struct definition")
		return sd
	}
	for !p.check(TOK_DEDENT) && !p.check(TOK_EOF) {
		p.skipNewlines()
		if p.check(TOK_DEDENT) || p.check(TOK_EOF) { break }
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
		p.skipNewlines()
	}
	p.match(TOK_DEDENT)
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
		// ignore what's imported — we handle it at a module level
		for !p.check(TOK_NEWLINE) && !p.check(TOK_EOF) {
			p.advance()
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

// ─── Block ────────────────────────────────────────────────────────────────────

func (p *Parser) parseBlock(context string) []Node {
	p.skipNewlines()
	if !p.match(TOK_INDENT) {
		t := p.peek()
		p.addError(ErrExpectedIndent, t.Line, t.Col, t.LineStr, context)
		return nil
	}

	var stmts []Node
	p.skipNewlines()
	for !p.check(TOK_DEDENT) && !p.check(TOK_EOF) {
		stmt := p.parseStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
		p.skipNewlines()
	}
	p.match(TOK_DEDENT)

	if len(stmts) == 0 {
		t := p.peek()
		p.addError(ErrEmptyBody, t.Line, t.Col, t.LineStr)
	}
	return stmts
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
	default:
		return p.parseExprOrAssign()
	}
}

func (p *Parser) parseReturn() *ReturnStmt {
	tok := p.advance()
	rs := &ReturnStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	if p.check(TOK_NEWLINE) || p.check(TOK_SEMICOL) || p.check(TOK_EOF) || p.check(TOK_DEDENT) {
		return rs
	}
	rs.Value = p.parseExpr()
	return rs
}

func (p *Parser) parseIf() *IfStmt {
	tok := p.advance() // 'if'
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
	tok := p.advance() // 'for'
	fs := &ForStmt{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}

	varTok := p.peek()
	if varTok.Type != TOK_IDENT {
		p.addError(ErrExpectedIdent, varTok.Line, varTok.Col, varTok.LineStr, varTok.Value)
	} else {
		p.advance()
	}
	fs.Var = varTok.Value

	// Optional type annotation: for x: int in ...
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
		if nt.Type != TOK_IDENT { break }
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

	// Check for assignment operators
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
		// typed declaration: name: type = expr
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

// ─── Expression Parsing (Pratt-style) ────────────────────────────────────────

func (p *Parser) parseExpr() Node {
	return p.parseTernary()
}

func (p *Parser) parseTernary() Node {
	expr := p.parseOr()
	// Python ternary: value if cond else other
	if p.check(TOK_IF) {
		p.advance()
		cond := p.parseOr()
		p.expect(TOK_ELSE)
		other := p.parseTernary()
		return &TernaryExpr{
			BaseNode: BaseNode{Line: expr.GetLine(), Col: expr.GetCol()},
			Then: expr, Cond: cond, Else: other,
		}
	}
	return expr
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
			// "x in list" — translate to contains
			p.advance()
			right := p.parseBitOr()
			left = &CallExpr{
				BaseNode: BaseNode{Line: t.Line, Col: t.Col},
				Func: &AttrExpr{Obj: right, Attr: "contains"},
				Args: []Node{left},
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
		right := p.parseUnary()
		opStr := op.Value
		if opStr == "//" {
			opStr = "FLOORDIV"
		}
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
		exp := p.parseUnary() // right assoc
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
			attr := &AttrExpr{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Obj: expr, Attr: attrTok.Value}
			expr = attr
		default:
			return expr
		}
	}
}

func (p *Parser) parseCall(fn Node) *CallExpr {
	tok := p.advance() // '('
	call := &CallExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Func: fn}

	for !p.check(TOK_RPAREN) && !p.check(TOK_EOF) {
		// keyword arg: name=value
		if p.peek().Type == TOK_IDENT && p.pos+1 < len(p.tokens) {
			next := p.tokens[p.pos+1]
			if next.Type == TOK_ASSIGN {
				nameTok := p.advance()
				p.advance() // '='
				val := p.parseExpr()
				call.KwArgs = append(call.KwArgs, KwArg{Name: nameTok.Value, Value: val})
				if !p.match(TOK_COMMA) { break }
				continue
			}
		}
		arg := p.parseExpr()
		call.Args = append(call.Args, arg)
		if !p.match(TOK_COMMA) { break }
	}
	t := p.peek()
	if !p.match(TOK_RPAREN) {
		p.addError(ErrExpectedRParen, t.Line, t.Col, t.LineStr, t.Value)
	}
	return call
}

func (p *Parser) parseIndex(obj Node) Node {
	tok := p.advance() // '['
	// slice?
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
		return &SliceExpr{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}, Obj: obj, Low: low, High: high, Step: step}
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
		// f-string  (f"...")
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

	// Built-in type names used as casts: int(...) float(...) str(...) bool(...)
	case TOK_TYPE_INT:
		p.advance()
		if p.check(TOK_LPAREN) {
			call := p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "int"})
			return call
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "int"}

	case TOK_TYPE_FLOAT:
		p.advance()
		if p.check(TOK_LPAREN) {
			call := p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "float"})
			return call
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "float"}

	case TOK_TYPE_STR:
		p.advance()
		if p.check(TOK_LPAREN) {
			call := p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "str"})
			return call
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "str"}

	case TOK_TYPE_BOOL:
		p.advance()
		if p.check(TOK_LPAREN) {
			call := p.parseCall(&Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "bool"})
			return call
		}
		return &Ident{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Name: "bool"}

	case TOK_NEW:
		return p.parseNew()

	default:
		p.addError(ErrExpectedExpr, t.Line, t.Col, t.LineStr, t.Value)
		p.advance()
		return &IntLit{BaseNode: BaseNode{Line: t.Line, Col: t.Col}, Value: 0}
	}
}

func (p *Parser) parseListLit() *ListLit {
	tok := p.advance() // '['
	ll := &ListLit{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	for !p.check(TOK_RBRACKET) && !p.check(TOK_EOF) {
		ll.Elems = append(ll.Elems, p.parseExpr())
		if !p.match(TOK_COMMA) { break }
	}
	t := p.peek()
	if !p.match(TOK_RBRACKET) {
		p.addError(ErrExpectedRBracket, t.Line, t.Col, t.LineStr, t.Value)
	}
	return ll
}

func (p *Parser) parseDictLit() *DictLit {
	tok := p.advance() // '{'
	dl := &DictLit{BaseNode: BaseNode{Line: tok.Line, Col: tok.Col}}
	for !p.check(TOK_RBRACE) && !p.check(TOK_EOF) {
		k := p.parseExpr()
		p.expect(TOK_COLON)
		v := p.parseExpr()
		dl.Keys = append(dl.Keys, k)
		dl.Vals = append(dl.Vals, v)
		if !p.match(TOK_COMMA) { break }
	}
	t := p.peek()
	if !p.match(TOK_RBRACE) {
		p.addError(ErrExpectedRBrace, t.Line, t.Col, t.LineStr, t.Value)
	}
	return dl
}

func (p *Parser) parseNew() *NewExpr {
	tok := p.advance() // 'new'
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
			if !p.match(TOK_COMMA) { break }
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

// Helper to format an int for error messages
func fmtInt(n int) string { return fmt.Sprintf("%d", n) }
