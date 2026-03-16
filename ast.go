package main

// ─── Types ────────────────────────────────────────────────────────────────────

type TypeKind string

const (
	TyInt    TypeKind = "int"
	TyFloat  TypeKind = "float"
	TyStr    TypeKind = "str"
	TyBool   TypeKind = "bool"
	TyVoid   TypeKind = "void"
	TyAny    TypeKind = "any"
	TyList   TypeKind = "list"
	TyStruct TypeKind = "struct"
	TyNone   TypeKind = "none"
	TyUnknown TypeKind = "unknown"
)

type Type struct {
	Kind     TypeKind
	ElemType *Type  // for list
	Name     string // for struct
}

var (
	TypInt   = &Type{Kind: TyInt}
	TypFloat = &Type{Kind: TyFloat}
	TypStr   = &Type{Kind: TyStr}
	TypBool  = &Type{Kind: TyBool}
	TypVoid  = &Type{Kind: TyVoid}
	TypAny   = &Type{Kind: TyAny}
	TypNone  = &Type{Kind: TyNone}
	TypUnknown = &Type{Kind: TyUnknown}
)

func ListType(elem *Type) *Type { return &Type{Kind: TyList, ElemType: elem} }
func StructType(name string) *Type { return &Type{Kind: TyStruct, Name: name} }

func (t *Type) String() string {
	if t == nil { return "unknown" }
	switch t.Kind {
	case TyList:
		if t.ElemType != nil {
			return "list[" + t.ElemType.String() + "]"
		}
		return "list"
	case TyStruct:
		return t.Name
	default:
		return string(t.Kind)
	}
}

func typesEqual(a, b *Type) bool {
	if a == nil || b == nil { return false }
	if a.Kind == TyAny || b.Kind == TyAny { return true }
	if a.Kind != b.Kind { return false }
	if a.Kind == TyStruct { return a.Name == b.Name }
	if a.Kind == TyList {
		if a.ElemType == nil || b.ElemType == nil { return true }
		return typesEqual(a.ElemType, b.ElemType)
	}
	return true
}

// ─── AST Nodes ────────────────────────────────────────────────────────────────

type Node interface {
	nodeTag()
	GetLine() int
	GetCol() int
}

type BaseNode struct {
	Line int
	Col  int
}
func (b BaseNode) nodeTag() {}
func (b BaseNode) GetLine() int { return b.Line }
func (b BaseNode) GetCol() int  { return b.Col }

// ── Program ──
type Program struct {
	BaseNode
	Stmts []Node
}

// ── Statements ──

type FuncDef struct {
	BaseNode
	Name       string
	Params     []Param
	ReturnType *Type
	Body       []Node
	IsVarArg   bool
}

type Param struct {
	Name    string
	Type    *Type
	Default Node // optional
	Line    int
	Col     int
}

type StructDef struct {
	BaseNode
	Name   string
	Fields []StructField
}

type StructField struct {
	Name string
	Type *Type
	Line int
	Col  int
}

type VarDecl struct {
	BaseNode
	Name  string
	Type  *Type // nil = auto-infer
	Value Node
}

type AssignStmt struct {
	BaseNode
	Target Node
	Op     string // = += -= *= /=
	Value  Node
}

type ReturnStmt struct {
	BaseNode
	Value Node // nil = return void
}

type IfStmt struct {
	BaseNode
	Cond     Node
	Then     []Node
	Elifs    []ElifClause
	ElseBody []Node
}

type ElifClause struct {
	Cond Node
	Body []Node
	Line int
}

type WhileStmt struct {
	BaseNode
	Cond Node
	Body []Node
}

type ForStmt struct {
	BaseNode
	Var    string
	VarType *Type
	Iter   Node // range(...) or list
	Body   []Node
}

type BreakStmt    struct{ BaseNode }
type ContinueStmt struct{ BaseNode }
type PassStmt     struct{ BaseNode }

type ExprStmt struct {
	BaseNode
	Expr Node
}

type ImportStmt struct {
	BaseNode
	Module string
	Alias  string
}

type GlobalStmt struct {
	BaseNode
	Names []string
}

type DeleteStmt struct {
	BaseNode
	Target Node
}

// ── Expressions ──

type IntLit struct {
	BaseNode
	Value int64
	Raw   string
}

type FloatLit struct {
	BaseNode
	Value float64
	Raw   string
}

type StringLit struct {
	BaseNode
	Value string
}

type BoolLit struct {
	BaseNode
	Value bool
}

type NoneLit struct{ BaseNode }

type Ident struct {
	BaseNode
	Name string
	ResolvedType *Type
}

type BinOp struct {
	BaseNode
	Op    string
	Left  Node
	Right Node
}

type UnaryOp struct {
	BaseNode
	Op      string
	Operand Node
}

type CallExpr struct {
	BaseNode
	Func Node
	Args []Node
	KwArgs []KwArg
}

type KwArg struct {
	Name  string
	Value Node
}

type IndexExpr struct {
	BaseNode
	Obj   Node
	Index Node
}

type SliceExpr struct {
	BaseNode
	Obj   Node
	Low   Node // nil = 0
	High  Node // nil = len
	Step  Node // nil = 1
}

type AttrExpr struct {
	BaseNode
	Obj  Node
	Attr string
}

type ListLit struct {
	BaseNode
	Elems []Node
}

type DictLit struct {
	BaseNode
	Keys []Node
	Vals []Node
}

type TernaryExpr struct {
	BaseNode
	Cond  Node
	Then  Node
	Else  Node
}

type LambdaExpr struct {
	BaseNode
	Params []Param
	Body   Node
}

type TypeCastExpr struct {
	BaseNode
	TargetType *Type
	Expr       Node
}

type NewExpr struct {
	BaseNode
	TypeName string
	Args     []Node
}

type RangeExpr struct {
	BaseNode
	Start Node
	Stop  Node
	Step  Node
}

type FStringExpr struct {
	BaseNode
	Parts []FStringPart
}

type FStringPart struct {
	IsExpr bool
	Text   string
	Expr   Node
}
