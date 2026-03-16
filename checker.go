package main

import (
	"fmt"
	"strings"
)

// ─── Built-in Function Signatures ────────────────────────────────────────────

type BuiltinSig struct {
	MinArgs int
	MaxArgs int // -1 = variadic
	RetType *Type
	// per-arg type (nil = any)
	ArgTypes []*Type
}

var builtins = map[string]BuiltinSig{
	"print":      {0, -1, TypVoid, nil},
	"println":    {0, -1, TypVoid, nil},
	"input":      {0, 1, TypStr, []*Type{TypStr}},
	"len":        {1, 1, TypInt, nil},
	"int":        {1, 1, TypInt, nil},
	"float":      {1, 1, TypFloat, nil},
	"str":        {1, 1, TypStr, nil},
	"bool":       {1, 1, TypBool, nil},
	"abs":        {1, 1, TypFloat, nil},
	"max":        {1, -1, TypFloat, nil},
	"min":        {1, -1, TypFloat, nil},
	"range":      {1, 3, TypAny, nil}, // special
	"append":     {2, 2, TypAny, nil}, // special
	"pop":        {1, 2, TypAny, nil},
	"type_of":    {1, 1, TypStr, nil},
	"exit":       {0, 1, TypVoid, []*Type{TypInt}},
	"assert":     {1, 2, TypVoid, nil},
	"ord":        {1, 1, TypInt, []*Type{TypStr}},
	"chr":        {1, 1, TypStr, []*Type{TypInt}},
	"hex":        {1, 1, TypStr, []*Type{TypInt}},
	"oct":        {1, 1, TypStr, []*Type{TypInt}},
	"bin":        {1, 1, TypStr, []*Type{TypInt}},
	"pow":        {2, 3, TypFloat, nil},
	"sqrt":       {1, 1, TypFloat, []*Type{TypFloat}},
	"floor":      {1, 1, TypInt, []*Type{TypFloat}},
	"ceil":       {1, 1, TypInt, []*Type{TypFloat}},
	"round":      {1, 2, TypFloat, nil},
	"upper":      {1, 1, TypStr, []*Type{TypStr}},
	"lower":      {1, 1, TypStr, []*Type{TypStr}},
	"strip":      {1, 1, TypStr, []*Type{TypStr}},
	"split":      {1, 2, TypAny, nil},
	"join":       {2, 2, TypStr, nil},
	"replace":    {3, 3, TypStr, nil},
	"format":     {1, -1, TypStr, nil},
	"contains":   {2, 2, TypBool, nil},
	"startswith": {2, 2, TypBool, []*Type{TypStr, TypStr}},
	"endswith":   {2, 2, TypBool, []*Type{TypStr, TypStr}},
	"index_of":   {2, 2, TypInt, nil},
	"count":      {2, 2, TypInt, nil},
	"sort":       {1, 1, TypVoid, nil},
	"reverse":    {1, 1, TypVoid, nil},
	"copy":       {1, 1, TypAny, nil},
	"keys":       {1, 1, TypAny, nil},
	"values":     {1, 1, TypAny, nil},
	"sleep":      {1, 1, TypVoid, []*Type{TypFloat}},
	"rand_int":   {2, 2, TypInt, []*Type{TypInt, TypInt}},
	"rand_float": {0, 0, TypFloat, nil},
	"open":       {1, 2, TypAny, []*Type{TypStr}},
	"close":      {1, 1, TypVoid, nil},
	"read":       {1, 1, TypStr, nil},
	"write":      {2, 2, TypVoid, nil},
	"printf":     {1, -1, TypVoid, nil},
	"scanf":      {1, -1, TypVoid, nil},
}

// String method signatures
var strMethods = map[string]BuiltinSig{
	"upper":      {0, 0, TypStr, nil},
	"lower":      {0, 0, TypStr, nil},
	"strip":      {0, 0, TypStr, nil},
	"lstrip":     {0, 0, TypStr, nil},
	"rstrip":     {0, 0, TypStr, nil},
	"split":      {0, 1, ListType(TypStr), nil},
	"replace":    {2, 2, TypStr, nil},
	"startswith": {1, 1, TypBool, nil},
	"endswith":   {1, 1, TypBool, nil},
	"contains":   {1, 1, TypBool, nil},
	"index_of":   {1, 1, TypInt, nil},
	"find":       {1, 1, TypInt, nil},
	"count":      {1, 1, TypInt, nil},
	"format":     {0, -1, TypStr, nil},
	"join":       {1, 1, TypStr, nil},
	"isdigit":    {0, 0, TypBool, nil},
	"isalpha":    {0, 0, TypBool, nil},
	"isspace":    {0, 0, TypBool, nil},
	"title":      {0, 0, TypStr, nil},
	"zfill":      {1, 1, TypStr, nil},
}

// List method signatures
var listMethods = map[string]BuiltinSig{
	"append":   {1, 1, TypVoid, nil},
	"pop":      {0, 1, TypAny, nil},
	"len":      {0, 0, TypInt, nil},
	"sort":     {0, 0, TypVoid, nil},
	"reverse":  {0, 0, TypVoid, nil},
	"copy":     {0, 0, TypAny, nil},
	"clear":    {0, 0, TypVoid, nil},
	"contains": {1, 1, TypBool, nil},
	"index_of": {1, 1, TypInt, nil},
	"count":    {1, 1, TypInt, nil},
	"extend":   {1, 1, TypVoid, nil},
	"insert":   {2, 2, TypVoid, nil},
	"remove":   {1, 1, TypVoid, nil},
}

// ─── Scope ────────────────────────────────────────────────────────────────────

type Symbol struct {
	Name     string
	Type     *Type
	IsFunc   bool
	FuncDef  *FuncDef
	IsGlobal bool
	Used     bool
	Line     int
	Col      int
	LineStr  string
}

type Scope struct {
	vars   map[string]*Symbol
	parent *Scope
}

func newScope(parent *Scope) *Scope {
	return &Scope{vars: make(map[string]*Symbol), parent: parent}
}

func (s *Scope) define(sym *Symbol) {
	s.vars[sym.Name] = sym
}

func (s *Scope) lookup(name string) (*Symbol, bool) {
	if sym, ok := s.vars[name]; ok {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.lookup(name)
	}
	return nil, false
}

func (s *Scope) lookupLocal(name string) (*Symbol, bool) {
	sym, ok := s.vars[name]
	return sym, ok
}

// ─── TypeChecker ──────────────────────────────────────────────────────────────

type TypeChecker struct {
	scope       *Scope
	structs     map[string]*StructDef
	functions   map[string]*FuncDef
	errors      []PycError
	currentFunc *FuncDef
	inLoop      int
	file        string
	lines       []string
	globals     map[string]bool
}

func NewTypeChecker(file string, source string) *TypeChecker {
	tc := &TypeChecker{
		scope:     newScope(nil),
		structs:   make(map[string]*StructDef),
		functions: make(map[string]*FuncDef),
		file:      file,
		lines:     strings.Split(source, "\n"),
		globals:   make(map[string]bool),
	}
	return tc
}

func (tc *TypeChecker) lineStr(line int) string {
	if line-1 < len(tc.lines) {
		return tc.lines[line-1]
	}
	return ""
}

func (tc *TypeChecker) addError(code ErrorCode, line, col int, args ...interface{}) {
	ls := tc.lineStr(line)
	tc.errors = append(tc.errors, newError(code, line, col, ls, args...))
}

func (tc *TypeChecker) addErrorLen(code ErrorCode, line, col, length int, args ...interface{}) {
	ls := tc.lineStr(line)
	tc.errors = append(tc.errors, newErrorLen(code, line, col, length, ls, args...))
}

func (tc *TypeChecker) pushScope() { tc.scope = newScope(tc.scope) }
func (tc *TypeChecker) popScope()  {
	// warn unused vars
	for _, sym := range tc.scope.vars {
		if !sym.Used && !sym.IsFunc && sym.Name != "_" {
			tc.addError(ErrUnusedVar, sym.Line, sym.Col, sym.Name)
		}
	}
	tc.scope = tc.scope.parent
}

// ─── First Pass: collect all top-level functions and structs ─────────────────

func (tc *TypeChecker) firstPass(prog *Program) {
	for _, node := range prog.Stmts {
		switch n := node.(type) {
		case *FuncDef:
			if _, exists := tc.functions[n.Name]; exists {
				tc.addError(ErrDuplicateFunc, n.Line, n.Col, n.Name)
			}
			tc.functions[n.Name] = n
			tc.scope.define(&Symbol{
				Name: n.Name, IsFunc: true, FuncDef: n,
				Type: n.ReturnType, Line: n.Line, Col: n.Col,
				LineStr: tc.lineStr(n.Line),
			})
		case *StructDef:
			if _, exists := tc.structs[n.Name]; exists {
				tc.addError(ErrDuplicateStruct, n.Line, n.Col, n.Name)
			}
			tc.structs[n.Name] = n
		}
	}
}

// ─── Check Program ────────────────────────────────────────────────────────────

func (tc *TypeChecker) Check(prog *Program) []PycError {
	tc.firstPass(prog)
	for _, stmt := range prog.Stmts {
		tc.checkStmt(stmt)
	}
	return tc.errors
}

// ─── Check Statement ──────────────────────────────────────────────────────────

func (tc *TypeChecker) checkStmt(node Node) {
	switch n := node.(type) {
	case *FuncDef:
		tc.checkFuncDef(n)
	case *StructDef:
		tc.checkStructDef(n)
	case *VarDecl:
		tc.checkVarDecl(n)
	case *AssignStmt:
		tc.checkAssign(n)
	case *ReturnStmt:
		tc.checkReturn(n)
	case *IfStmt:
		tc.checkIf(n)
	case *WhileStmt:
		tc.checkWhile(n)
	case *ForStmt:
		tc.checkFor(n)
	case *BreakStmt:
		if tc.inLoop == 0 {
			tc.addError(ErrBreakOutsideLoop, n.Line, n.Col)
		}
	case *ContinueStmt:
		if tc.inLoop == 0 {
			tc.addError(ErrContinueOutsideLoop, n.Line, n.Col)
		}
	case *GlobalStmt:
		if tc.currentFunc == nil {
			tc.addError(ErrGlobalOutsideFunc, n.Line, n.Col)
		}
		for _, name := range n.Names {
			tc.globals[name] = true
		}
	case *DeleteStmt:
		tc.checkExpr(n.Target)
	case *ExprStmt:
		tc.checkExpr(n.Expr)
	case *PassStmt, *ImportStmt:
		// ok
	// Smalltalk nodes
	case *IfTrueStmt, *RepeatStmt, *EachStmt, *LoopStmt, *UntilStmt,
		*SwapStmt, *DefaultStmt, *CheckStmt, *DieStmt, *MaybeStmt, *PrintBangStmt:
		tc.checkSmallTalk(node)
	}
}

func (tc *TypeChecker) checkFuncDef(n *FuncDef) {
	// Only warn if the name shadows a top-level builtin function, not method names
	if _, ok := builtins[n.Name]; ok {
		if isTopLevelBuiltin(n.Name) {
			tc.addError(ErrShadowBuiltin, n.Line, n.Col, n.Name)
		}
	}
	// Check duplicate params
	seen := map[string]bool{}
	for _, p := range n.Params {
		if seen[p.Name] {
			tc.addError(ErrDuplicateParam, p.Line, p.Col, p.Name, n.Name)
		}
		seen[p.Name] = true
	}

	prev := tc.currentFunc
	tc.currentFunc = n
	tc.pushScope()

	for _, p := range n.Params {
		tc.scope.define(&Symbol{Name: p.Name, Type: p.Type, Used: true, Line: p.Line, Col: p.Col})
	}

	for _, stmt := range n.Body {
		tc.checkStmt(stmt)
	}

	// Check that non-void functions always return on all paths
	if !typesEqual(n.ReturnType, TypVoid) && n.ReturnType.Kind != TyAny {
		if !bodyAlwaysReturns(n.Body) {
			tc.addError(ErrNonVoidNoReturn, n.Line, n.Col, n.Name, n.ReturnType.String())
		}
	}

	tc.popScope()
	tc.currentFunc = prev
}

// isTopLevelBuiltin returns true only for functions that should be called as
// standalone builtins (not method names like "count", "split", etc.)
func isTopLevelBuiltin(name string) bool {
	topLevel := map[string]bool{
		"print": true, "println": true, "input": true, "len": true,
		"int": true, "float": true, "str": true, "bool": true,
		"abs": true, "max": true, "min": true, "range": true,
		"append": true, "exit": true, "assert": true,
		"ord": true, "chr": true, "hex": true, "oct": true, "bin": true,
		"pow": true, "sqrt": true, "floor": true, "ceil": true, "round": true,
		"type_of": true, "sleep": true, "rand_int": true, "rand_float": true,
		"open": true, "close": true, "read": true, "write": true,
		"printf": true, "scanf": true,
	}
	return topLevel[name]
}

// bodyAlwaysReturns returns true if every execution path through stmts
// ends in a return statement.
func bodyAlwaysReturns(stmts []Node) bool {
	for i := len(stmts) - 1; i >= 0; i-- {
		stmt := stmts[i]
		switch s := stmt.(type) {
		case *ReturnStmt:
			return true
		case *IfStmt:
			if stmtAlwaysReturns(s) {
				return true
			}
		}
	}
	return false
}

// stmtAlwaysReturns returns true if a single statement always returns.
func stmtAlwaysReturns(stmt Node) bool {
	switch s := stmt.(type) {
	case *ReturnStmt:
		return true
	case *IfStmt:
		// Must have an else clause, and every branch must return
		if s.ElseBody == nil {
			return false
		}
		if !bodyAlwaysReturns(s.Then) {
			return false
		}
		for _, elif := range s.Elifs {
			if !bodyAlwaysReturns(elif.Body) {
				return false
			}
		}
		if !bodyAlwaysReturns(s.ElseBody) {
			return false
		}
		return true
	}
	return false
}

func (tc *TypeChecker) checkStructDef(n *StructDef) {
	seen := map[string]bool{}
	for _, f := range n.Fields {
		if seen[f.Name] {
			tc.addError(ErrDuplicateVar, f.Line, f.Col, f.Name)
		}
		seen[f.Name] = true
		if f.Type.Kind == TyStruct && f.Type.Name == n.Name {
			tc.addError(ErrRecursiveStruct, f.Line, f.Col, n.Name)
		}
	}
}

func (tc *TypeChecker) checkVarDecl(n *VarDecl) {
	if isTopLevelBuiltin(n.Name) {
		tc.addError(ErrShadowBuiltin, n.Line, n.Col, n.Name)
	}
	var valType *Type
	if n.Value != nil {
		valType = tc.checkExpr(n.Value)
	}
	declType := n.Type
	if declType == nil || declType.Kind == TyUnknown {
		// infer
		if valType != nil {
			declType = valType
		} else {
			declType = TypAny
		}
	} else if valType != nil && !typesEqual(declType, valType) && valType.Kind != TyAny && declType.Kind != TyAny {
		tc.addErrorLen(ErrTypeMismatch, n.Line, n.Col, len(n.Name), declType.String(), valType.String())
	}

	if existing, ok := tc.scope.lookupLocal(n.Name); ok && existing != nil {
		tc.addError(ErrDuplicateVar, n.Line, n.Col, n.Name)
		return
	}
	tc.scope.define(&Symbol{
		Name: n.Name, Type: declType,
		Line: n.Line, Col: n.Col, LineStr: tc.lineStr(n.Line),
	})
}

func (tc *TypeChecker) checkAssign(n *AssignStmt) {
	valType := tc.checkExpr(n.Value)

	switch target := n.Target.(type) {
	case *Ident:
		sym, ok := tc.scope.lookup(target.Name)
		if !ok {
			// auto-declare
			if n.Op != "=" {
				tc.addError(ErrUndefinedVar, n.Line, n.Col, target.Name)
				return
			}
			tc.scope.define(&Symbol{
				Name: target.Name, Type: valType,
				Used: false, Line: n.Line, Col: n.Col,
				LineStr: tc.lineStr(n.Line),
			})
			return
		}
		sym.Used = true
		if sym.Type != nil && valType != nil && !typesEqual(sym.Type, valType) &&
			sym.Type.Kind != TyAny && valType.Kind != TyAny {
			if n.Op == "=" {
				tc.addErrorLen(ErrReassignType, n.Line, n.Col, len(target.Name), target.Name, valType.String(), sym.Type.String())
			}
		}
	case *IndexExpr:
		tc.checkExpr(target)
	case *AttrExpr:
		tc.checkExpr(target)
	default:
		tc.addError(ErrInvalidAssignTarget, n.Line, n.Col, fmt.Sprintf("%T", n.Target))
	}
}

func (tc *TypeChecker) checkReturn(n *ReturnStmt) {
	if tc.currentFunc == nil {
		tc.addError(ErrReturnOutsideFunc, n.Line, n.Col)
		return
	}
	retType := tc.currentFunc.ReturnType
	if n.Value == nil {
		if !typesEqual(retType, TypVoid) && retType.Kind != TyAny {
			tc.addError(ErrNoReturnValue, n.Line, n.Col, tc.currentFunc.Name, retType.String())
		}
		return
	}
	if typesEqual(retType, TypVoid) {
		tc.addError(ErrVoidReturn, n.Line, n.Col, tc.currentFunc.Name)
		return
	}
	valType := tc.checkExpr(n.Value)
	if valType != nil && !typesEqual(retType, valType) && retType.Kind != TyAny && valType.Kind != TyAny {
		tc.addError(ErrReturnTypeMismatch, n.Line, n.Col, retType.String(), valType.String())
	}
}

func (tc *TypeChecker) checkIf(n *IfStmt) {
	tc.checkExpr(n.Cond)
	tc.checkBranchBody(n.Then)
	for _, elif := range n.Elifs {
		tc.checkExpr(elif.Cond)
		tc.checkBranchBody(elif.Body)
	}
	if n.ElseBody != nil {
		tc.checkBranchBody(n.ElseBody)
	}
}

// checkBranchBody checks a branch body in a sub-scope, then promotes any
// newly defined variables up to the parent scope (Python has flat scoping —
// variables assigned inside if/elif/else branches are visible after the block).
func (tc *TypeChecker) checkBranchBody(stmts []Node) {
	tc.pushScope()
	for _, s := range stmts {
		tc.checkStmt(s)
	}
	// Promote vars defined in this branch to parent scope.
	// Mark them as Used=true so they don't trigger unused-var warnings here;
	// the parent scope will track actual usage.
	promoted := tc.scope.vars
	tc.scope = tc.scope.parent // manual pop without unused-var warnings
	for name, sym := range promoted {
		if sym.IsFunc {
			continue
		}
		if _, exists := tc.scope.vars[name]; !exists {
			// Copy to parent scope, mark used so sub-scope doesn't re-warn
			parentSym := &Symbol{
				Name: sym.Name, Type: sym.Type,
				Line: sym.Line, Col: sym.Col, LineStr: sym.LineStr,
				Used: false, // parent scope tracks real usage
			}
			tc.scope.define(parentSym)
		} else {
			// Already in parent (e.g. declared before the if); mark as used
			// if it was used in this branch
			if sym.Used {
				tc.scope.vars[name].Used = true
			}
		}
	}
}

func (tc *TypeChecker) checkWhile(n *WhileStmt) {
	tc.checkExpr(n.Cond)
	tc.inLoop++
	tc.pushScope()
	for _, s := range n.Body { tc.checkStmt(s) }
	tc.popScope()
	tc.inLoop--
}

func (tc *TypeChecker) checkFor(n *ForStmt) {
	iterType := tc.checkExpr(n.Iter)
	var elemType *Type = TypAny

	switch it := n.Iter.(type) {
	case *CallExpr:
		if id, ok := it.Func.(*Ident); ok && id.Name == "range" {
			elemType = TypInt
			// check range args
			if len(it.Args) < 1 || len(it.Args) > 3 {
				tc.addError(ErrRangeArgCount, n.Line, n.Col, len(it.Args))
			}
			for _, arg := range it.Args {
				at := tc.checkExpr(arg)
				if at != nil && at.Kind != TyInt && at.Kind != TyAny {
					tc.addError(ErrRangeArgType, n.Line, n.Col, at.String())
				}
			}
		}
	default:
		if iterType != nil {
			if iterType.Kind == TyList {
				if iterType.ElemType != nil {
					elemType = iterType.ElemType
				}
			} else if iterType.Kind == TyStr {
				elemType = TypStr
			} else if iterType.Kind != TyAny {
				tc.addError(ErrForIterType, n.Line, n.Col, iterType.String())
			}
		}
	}

	varType := elemType
	if n.VarType != nil {
		varType = n.VarType
	}

	tc.inLoop++
	tc.pushScope()
	tc.scope.define(&Symbol{
		Name: n.Var, Type: varType,
		Used: true, Line: n.Line, Col: n.Col,
	})
	for _, s := range n.Body { tc.checkStmt(s) }
	tc.popScope()
	tc.inLoop--
}

// ─── Check Expression → returns type ─────────────────────────────────────────

func (tc *TypeChecker) checkExpr(node Node) *Type {
	if node == nil { return TypAny }

	switch n := node.(type) {
	case *IntLit:
		return TypInt
	case *FloatLit:
		return TypFloat
	case *StringLit:
		return TypStr
	case *BoolLit:
		return TypBool
	case *NoneLit:
		return TypNone
	case *ListLit:
		return tc.checkListLit(n)
	case *DictLit:
		return TypAny // dicts not fully typed yet
	case *Ident:
		return tc.checkIdent(n)
	case *BinOp:
		return tc.checkBinOp(n)
	case *UnaryOp:
		return tc.checkUnaryOp(n)
	case *CallExpr:
		return tc.checkCall(n)
	case *IndexExpr:
		return tc.checkIndex(n)
	case *SliceExpr:
		return tc.checkSlice(n)
	case *AttrExpr:
		return tc.checkAttr(n)
	case *TernaryExpr:
		tc.checkExpr(n.Cond)
		t1 := tc.checkExpr(n.Then)
		t2 := tc.checkExpr(n.Else)
		if typesEqual(t1, t2) { return t1 }
		return TypAny
	case *FStringExpr:
		for _, part := range n.Parts {
			if part.IsExpr {
				tc.checkExpr(part.Expr)
			}
		}
		return TypStr
	case *NewExpr:
		return tc.checkNew(n)
	case *LambdaExpr:
		return TypAny
	case *RangeExpr:
		return ListType(TypInt)
	case *ClampExpr:
		tc.checkSmallTalk(n)
		return tc.checkExpr(n.Val)
	case *BetweenExpr:
		tc.checkSmallTalk(n)
		return TypBool
	case *EitherExpr:
		tc.checkSmallTalk(n)
		t1 := tc.checkExpr(n.A)
		if typesEqual(t1, TypAny) { return tc.checkExpr(n.B) }
		return t1
	}
	return TypAny
}

func (tc *TypeChecker) checkListLit(n *ListLit) *Type {
	if len(n.Elems) == 0 {
		return ListType(TypAny)
	}
	firstType := tc.checkExpr(n.Elems[0])
	for i, elem := range n.Elems[1:] {
		et := tc.checkExpr(elem)
		if et != nil && firstType != nil && !typesEqual(firstType, et) && firstType.Kind != TyAny && et.Kind != TyAny {
			tc.addError(ErrListElemType, elem.GetLine(), elem.GetCol(), i+1, et.String(), firstType.String())
		}
	}
	return ListType(firstType)
}

func (tc *TypeChecker) checkIdent(n *Ident) *Type {
	// builtins
	if sig, ok := builtins[n.Name]; ok {
		n.ResolvedType = sig.RetType
		return sig.RetType
	}
	sym, ok := tc.scope.lookup(n.Name)
	if !ok {
		tc.addErrorLen(ErrUndefinedVar, n.Line, n.Col, len(n.Name), n.Name)
		return TypAny
	}
	sym.Used = true
	n.ResolvedType = sym.Type
	return sym.Type
}

func (tc *TypeChecker) checkBinOp(n *BinOp) *Type {
	lt := tc.checkExpr(n.Left)
	rt := tc.checkExpr(n.Right)

	// Bitwise ops
	bitwiseOps := map[string]bool{"&": true, "|": true, "^": true, "~": true, "<<": true, ">>": true}
	if bitwiseOps[n.Op] {
		if lt != nil && lt.Kind == TyFloat {
			tc.addError(ErrBitwiseOnFloat, n.Line, n.Col, n.Op)
		}
		if lt != nil && lt.Kind == TyStr {
			tc.addError(ErrBitwiseOnStr, n.Line, n.Col, n.Op)
		}
		return TypInt
	}

	// String concatenation
	if n.Op == "+" && lt != nil && lt.Kind == TyStr {
		if rt != nil && rt.Kind != TyStr && rt.Kind != TyAny {
			tc.addError(ErrTypeMismatch, n.Line, n.Col, "str", rt.String())
		}
		return TypStr
	}

	// Comparison ops return bool
	cmpOps := map[string]bool{"==": true, "!=": true, "<": true, ">": true, "<=": true, ">=": true}
	if cmpOps[n.Op] {
		if lt != nil && rt != nil && lt.Kind != TyAny && rt.Kind != TyAny {
			// incompatible comparisons
			if (lt.Kind == TyStr && (rt.Kind == TyInt || rt.Kind == TyFloat)) ||
				(rt.Kind == TyStr && (lt.Kind == TyInt || lt.Kind == TyFloat)) {
				tc.addError(ErrCompareIncompat, n.Line, n.Col, lt.String(), rt.String(), n.Op)
			}
		}
		return TypBool
	}

	// Power always returns float
	if n.Op == "**" {
		return TypFloat
	}

	// Arithmetic
	if lt != nil && rt != nil {
		if lt.Kind == TyStr || rt.Kind == TyStr {
			if n.Op != "+" {
				tc.addError(ErrInvalidOp, n.Line, n.Col, n.Op, lt.String(), rt.String())
			}
		}
		if lt.Kind == TyFloat || rt.Kind == TyFloat {
			return TypFloat
		}
		if lt.Kind == TyInt && rt.Kind == TyInt {
			if n.Op == "FLOORDIV" || n.Op == "/" {
				return TypFloat
			}
			return TypInt
		}
	}
	return TypAny
}

func (tc *TypeChecker) checkUnaryOp(n *UnaryOp) *Type {
	t := tc.checkExpr(n.Operand)
	if n.Op == "!" {
		return TypBool
	}
	return t
}

func (tc *TypeChecker) checkCall(n *CallExpr) *Type {
	// Special handling for range()
	if id, ok := n.Func.(*Ident); ok && id.Name == "range" {
		if len(n.Args) < 1 || len(n.Args) > 3 {
			tc.addError(ErrRangeArgCount, n.Line, n.Col, len(n.Args))
		}
		for _, arg := range n.Args {
			at := tc.checkExpr(arg)
			if at != nil && at.Kind != TyInt && at.Kind != TyAny {
				tc.addError(ErrRangeArgType, n.Line, n.Col, at.String())
			}
		}
		return ListType(TypInt)
	}

	// Attr call  e.g. x.startswith("a")
	if attr, ok := n.Func.(*AttrExpr); ok {
		objType := tc.checkExpr(attr.Obj)
		for _, arg := range n.Args {
			tc.checkExpr(arg)
		}
		return tc.resolveMethodCall(objType, attr.Attr, n)
	}

	// Regular function call
	if id, ok := n.Func.(*Ident); ok {
		// builtin?
		if sig, ok := builtins[id.Name]; ok {
			tc.checkBuiltinCall(n, id.Name, sig)
			return sig.RetType
		}
		// user function?
		if fn, ok := tc.functions[id.Name]; ok {
			tc.checkUserCall(n, fn)
			return fn.ReturnType
		}
		// struct constructor?
		if sd, ok := tc.structs[id.Name]; ok {
			_ = sd
			for _, arg := range n.Args {
				tc.checkExpr(arg)
			}
			return StructType(id.Name)
		}
		// scope lookup
		sym, found := tc.scope.lookup(id.Name)
		if !found {
			tc.addErrorLen(ErrUndefinedFunc, n.Line, n.Col, len(id.Name), id.Name)
			return TypAny
		}
		sym.Used = true
		if !sym.IsFunc {
			tc.addError(ErrNotCallable, n.Line, n.Col, id.Name)
			return TypAny
		}
		for _, arg := range n.Args { tc.checkExpr(arg) }
		return sym.Type
	}

	// callable expression
	for _, arg := range n.Args { tc.checkExpr(arg) }
	return TypAny
}

func (tc *TypeChecker) checkBuiltinCall(n *CallExpr, name string, sig BuiltinSig) {
	totalArgs := len(n.Args) + len(n.KwArgs)
	if sig.MaxArgs >= 0 && totalArgs > sig.MaxArgs {
		tc.addError(ErrWrongArgCount, n.Line, n.Col, name, sig.MaxArgs, totalArgs)
	}
	if totalArgs < sig.MinArgs {
		tc.addError(ErrWrongArgCount, n.Line, n.Col, name, sig.MinArgs, totalArgs)
	}
	for i, arg := range n.Args {
		at := tc.checkExpr(arg)
		if sig.ArgTypes != nil && i < len(sig.ArgTypes) && sig.ArgTypes[i] != nil {
			expected := sig.ArgTypes[i]
			if at != nil && !typesEqual(expected, at) && at.Kind != TyAny && expected.Kind != TyAny {
				tc.addError(ErrWrongArgType, arg.GetLine(), arg.GetCol(), i+1, name, expected.String(), at.String())
			}
		}
	}
}

func (tc *TypeChecker) checkUserCall(n *CallExpr, fn *FuncDef) {
	minArgs := 0
	for _, p := range fn.Params {
		if p.Default == nil {
			minArgs++
		}
	}
	totalArgs := len(n.Args) + len(n.KwArgs)
	if totalArgs < minArgs || totalArgs > len(fn.Params) {
		tc.addError(ErrWrongArgCount, n.Line, n.Col, fn.Name, len(fn.Params), totalArgs)
	}
	for i, arg := range n.Args {
		at := tc.checkExpr(arg)
		if i < len(fn.Params) {
			expected := fn.Params[i].Type
			if at != nil && expected != nil && !typesEqual(expected, at) &&
				at.Kind != TyAny && expected.Kind != TyAny {
				tc.addError(ErrWrongArgType, arg.GetLine(), arg.GetCol(), i+1, fn.Name, expected.String(), at.String())
			}
		}
	}
}

func (tc *TypeChecker) resolveMethodCall(objType *Type, method string, n *CallExpr) *Type {
	if objType == nil { return TypAny }
	switch objType.Kind {
	case TyStr:
		if sig, ok := strMethods[method]; ok {
			return sig.RetType
		}
		tc.addError(ErrAttrNotExist, n.Line, n.Col, "str", method)
		return TypAny
	case TyList:
		if sig, ok := listMethods[method]; ok {
			return sig.RetType
		}
		tc.addError(ErrAttrNotExist, n.Line, n.Col, "list", method)
		return TypAny
	case TyStruct:
		sd, ok := tc.structs[objType.Name]
		if !ok { return TypAny }
		for _, f := range sd.Fields {
			if f.Name == method {
				return f.Type
			}
		}
		tc.addError(ErrStructFieldNotExist, n.Line, n.Col, objType.Name, method)
		return TypAny
	}
	return TypAny
}

func (tc *TypeChecker) checkIndex(n *IndexExpr) *Type {
	objType := tc.checkExpr(n.Obj)
	idxType := tc.checkExpr(n.Index)
	if objType != nil && objType.Kind != TyList && objType.Kind != TyStr && objType.Kind != TyAny {
		tc.addError(ErrIndexNonList, n.Line, n.Col, objType.String())
		return TypAny
	}
	if idxType != nil && idxType.Kind != TyInt && idxType.Kind != TyAny {
		tc.addError(ErrIndexType, n.Line, n.Col, idxType.String())
	}
	if objType != nil && objType.Kind == TyList && objType.ElemType != nil {
		return objType.ElemType
	}
	if objType != nil && objType.Kind == TyStr {
		return TypStr
	}
	return TypAny
}

func (tc *TypeChecker) checkSlice(n *SliceExpr) *Type {
	objType := tc.checkExpr(n.Obj)
	if objType != nil && objType.Kind != TyList && objType.Kind != TyStr && objType.Kind != TyAny {
		tc.addError(ErrSliceOnNonList, n.Line, n.Col, objType.String())
	}
	if n.Low != nil { tc.checkExpr(n.Low) }
	if n.High != nil { tc.checkExpr(n.High) }
	if n.Step != nil { tc.checkExpr(n.Step) }
	return objType
}

func (tc *TypeChecker) checkAttr(n *AttrExpr) *Type {
	objType := tc.checkExpr(n.Obj)
	if objType == nil { return TypAny }
	switch objType.Kind {
	case TyStr:
		if sig, ok := strMethods[n.Attr]; ok {
			return sig.RetType
		}
		tc.addError(ErrAttrNotExist, n.Line, n.Col, "str", n.Attr)
		return TypAny
	case TyList:
		if sig, ok := listMethods[n.Attr]; ok {
			return sig.RetType
		}
		tc.addError(ErrAttrNotExist, n.Line, n.Col, "list", n.Attr)
		return TypAny
	case TyStruct:
		sd, ok := tc.structs[objType.Name]
		if !ok { return TypAny }
		for _, f := range sd.Fields {
			if f.Name == n.Attr {
				return f.Type
			}
		}
		tc.addError(ErrStructFieldNotExist, n.Line, n.Col, objType.Name, n.Attr)
		return TypAny
	case TyInt, TyFloat, TyBool:
		tc.addError(ErrAttrOnPrimitive, n.Line, n.Col, n.Attr, objType.String())
		return TypAny
	}
	return TypAny
}

func (tc *TypeChecker) checkNew(n *NewExpr) *Type {
	if _, ok := tc.structs[n.TypeName]; !ok {
		tc.addError(ErrUndefinedType, n.Line, n.Col, n.TypeName)
		return TypAny
	}
	for _, arg := range n.Args { tc.checkExpr(arg) }
	return StructType(n.TypeName)
}

// ─── Smalltalk node checking ──────────────────────────────────────────────────

func (tc *TypeChecker) checkSmallTalk(node Node) {
	switch n := node.(type) {
	case *IfTrueStmt:
		tc.checkExpr(n.Cond)
		tc.checkStmt(n.Body)
	case *RepeatStmt:
		tc.checkExpr(n.Count)
		tc.inLoop++
		tc.checkStmt(n.Body)
		tc.inLoop--
	case *EachStmt:
		iterType := tc.checkExpr(n.Iter)
		elemType := TypAny
		if iterType != nil && iterType.Kind == TyList && iterType.ElemType != nil {
			elemType = iterType.ElemType
		}
		tc.pushScope()
		tc.scope.define(&Symbol{Name: n.Var, Type: elemType, Used: true, Line: n.Line, Col: n.Col})
		tc.inLoop++
		tc.checkStmt(n.Body)
		tc.inLoop--
		tc.popScope()
	case *LoopStmt:
		tc.inLoop++
		tc.pushScope()
		for _, s := range n.Body { tc.checkStmt(s) }
		tc.popScope()
		tc.inLoop--
	case *UntilStmt:
		tc.checkExpr(n.Cond)
		tc.inLoop++
		tc.pushScope()
		for _, s := range n.Body { tc.checkStmt(s) }
		tc.popScope()
		tc.inLoop--
	case *SwapStmt:
		tc.checkExpr(n.A)
		tc.checkExpr(n.B)
	case *DefaultStmt:
		tc.checkExpr(n.Target)
		tc.checkExpr(n.Value)
	case *CheckStmt:
		tc.checkExpr(n.Expr)
	case *DieStmt:
		tc.checkExpr(n.Msg)
	case *MaybeStmt:
		tc.checkExpr(n.Expr)
	case *PrintBangStmt:
		for _, a := range n.Args { tc.checkExpr(a) }
	case *ClampExpr:
		tc.checkExpr(n.Val); tc.checkExpr(n.Lo); tc.checkExpr(n.Hi)
	case *BetweenExpr:
		tc.checkExpr(n.Val); tc.checkExpr(n.Lo); tc.checkExpr(n.Hi)
	case *EitherExpr:
		tc.checkExpr(n.A); tc.checkExpr(n.B)
	}
}
