package main

import (
	"fmt"
	"strings"
)

// ─── ANSI Colors ──────────────────────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorFaint  = "\033[2m"
	colorGreen  = "\033[32m"
	colorMagenta = "\033[35m"
	colorWhite  = "\033[37m"
)

// ─── Error Codes ─────────────────────────────────────────────────────────────

type ErrorCode int

const (
	// Lexer errors (1-19)
	ErrUnexpectedChar    ErrorCode = 1
	ErrUnterminatedString ErrorCode = 2
	ErrBadIndent         ErrorCode = 3
	ErrInvalidNumber     ErrorCode = 4
	ErrInvalidEscape     ErrorCode = 5

	// Syntax / Parser errors (20-59)
	ErrExpectedToken       ErrorCode = 20
	ErrExpectedExpr        ErrorCode = 21
	ErrExpectedColon       ErrorCode = 22
	ErrExpectedIndent      ErrorCode = 23
	ErrUnexpectedToken     ErrorCode = 24
	ErrExpectedIdent       ErrorCode = 25
	ErrExpectedType        ErrorCode = 26
	ErrExpectedRParen      ErrorCode = 27
	ErrExpectedRBracket    ErrorCode = 28
	ErrExpectedComma       ErrorCode = 29
	ErrExpectedArrow       ErrorCode = 30
	ErrMissingFuncBody     ErrorCode = 31
	ErrInvalidAssignTarget ErrorCode = 32
	ErrTooManyStars        ErrorCode = 33
	ErrExpectedIn          ErrorCode = 34
	ErrInvalidDecorator    ErrorCode = 35
	ErrExpectedRBrace      ErrorCode = 36
	ErrEmptyBody           ErrorCode = 37
	ErrInvalidSlice        ErrorCode = 38
	ErrDuplicateParam      ErrorCode = 39

	// Type errors (60-99)
	ErrTypeMismatch       ErrorCode = 60
	ErrUndefinedVar       ErrorCode = 61
	ErrUndefinedFunc      ErrorCode = 62
	ErrUndefinedType      ErrorCode = 63
	ErrWrongArgCount      ErrorCode = 64
	ErrWrongArgType       ErrorCode = 65
	ErrNotCallable        ErrorCode = 66
	ErrReturnTypeMismatch ErrorCode = 67
	ErrNoReturnValue      ErrorCode = 68
	ErrVoidReturn         ErrorCode = 69
	ErrInvalidOp         ErrorCode = 70
	ErrIndexNonList       ErrorCode = 71
	ErrIndexType          ErrorCode = 72
	ErrAttrNotExist       ErrorCode = 73
	ErrAttrOnPrimitive    ErrorCode = 74
	ErrReassignType       ErrorCode = 75
	ErrDuplicateVar       ErrorCode = 76
	ErrDuplicateFunc      ErrorCode = 77
	ErrDuplicateStruct    ErrorCode = 78
	ErrStructFieldNotExist ErrorCode = 79
	ErrStructFieldType    ErrorCode = 80
	ErrDivideByZero       ErrorCode = 81
	ErrUninitializedVar   ErrorCode = 82
	ErrBreakOutsideLoop   ErrorCode = 83
	ErrContinueOutsideLoop ErrorCode = 84
	ErrReturnOutsideFunc  ErrorCode = 85
	ErrGlobalOutsideFunc  ErrorCode = 86
	ErrNonVoidNoReturn    ErrorCode = 87
	ErrInvalidCast        ErrorCode = 88
	ErrListElemType       ErrorCode = 89
	ErrNegativeIndex      ErrorCode = 90
	ErrSliceOnNonList     ErrorCode = 91
	ErrBitwiseOnFloat     ErrorCode = 92
	ErrBitwiseOnStr       ErrorCode = 93
	ErrCompareIncompat    ErrorCode = 94
	ErrNullDeref          ErrorCode = 95
	ErrRecursiveStruct    ErrorCode = 96
	ErrMainNotDefined     ErrorCode = 97  // warning only
	ErrUnusedVar          ErrorCode = 98  // warning
	ErrShadowBuiltin      ErrorCode = 99

	// Semantic (100+)
	ErrForIterType         ErrorCode = 100
	ErrRangeArgCount       ErrorCode = 101
	ErrRangeArgType        ErrorCode = 102
	ErrDeleteNonPointer    ErrorCode = 103
	ErrImportUnknown       ErrorCode = 104
	ErrCircularImport      ErrorCode = 105
	ErrStructInitFields    ErrorCode = 106
	ErrMissingReturnPath   ErrorCode = 107
	ErrInvalidFString      ErrorCode = 108
	ErrNegativeSlice       ErrorCode = 109
	ErrLambdaInStmt        ErrorCode = 110
	ErrVoidInExpr          ErrorCode = 111
)

type Severity string
const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
	SevNote    Severity = "note"
)

var errorMeta = map[ErrorCode]struct {
	sev  Severity
	tmpl string
}{
	ErrUnexpectedChar:     {SevError, "unexpected character %q"},
	ErrUnterminatedString: {SevError, "unterminated string literal"},
	ErrBadIndent:          {SevError, "indentation does not match any outer indentation level"},
	ErrInvalidNumber:      {SevError, "invalid numeric literal %q"},
	ErrInvalidEscape:      {SevError, "invalid escape sequence '\\%s'"},

	ErrExpectedToken:       {SevError, "expected %q but got %q"},
	ErrExpectedExpr:        {SevError, "expected expression but got %q"},
	ErrExpectedColon:       {SevError, "expected ':' after %s"},
	ErrExpectedIndent:      {SevError, "expected indented block after %s"},
	ErrUnexpectedToken:     {SevError, "unexpected token %q"},
	ErrExpectedIdent:       {SevError, "expected identifier but got %q"},
	ErrExpectedType:        {SevError, "expected type annotation but got %q"},
	ErrExpectedRParen:      {SevError, "expected closing ')' but got %q"},
	ErrExpectedRBracket:    {SevError, "expected closing ']' but got %q"},
	ErrExpectedComma:       {SevError, "expected ',' or closing delimiter but got %q"},
	ErrExpectedArrow:       {SevError, "expected '->' for return type but got %q"},
	ErrMissingFuncBody:     {SevError, "function %q has no body"},
	ErrInvalidAssignTarget: {SevError, "cannot assign to %q — not a valid target"},
	ErrTooManyStars:        {SevError, "invalid syntax: too many '*' in expression"},
	ErrExpectedIn:          {SevError, "expected 'in' in for loop but got %q"},
	ErrInvalidDecorator:    {SevError, "invalid decorator expression"},
	ErrExpectedRBrace:      {SevError, "expected closing '}' but got %q"},
	ErrEmptyBody:           {SevError, "block body cannot be empty — use 'pass' for empty blocks"},
	ErrInvalidSlice:        {SevError, "invalid slice expression"},
	ErrDuplicateParam:      {SevError, "duplicate parameter name %q in function %q"},

	ErrTypeMismatch:        {SevError, "type mismatch: expected %s but got %s"},
	ErrUndefinedVar:        {SevError, "undefined variable %q — did you forget to declare it?"},
	ErrUndefinedFunc:       {SevError, "call to undefined function %q"},
	ErrUndefinedType:       {SevError, "unknown type %q"},
	ErrWrongArgCount:       {SevError, "function %q expects %d argument(s) but got %d"},
	ErrWrongArgType:        {SevError, "argument %d of %q: expected %s but got %s"},
	ErrNotCallable:         {SevError, "%q is not callable"},
	ErrReturnTypeMismatch:  {SevError, "return type mismatch: function expects %s but returning %s"},
	ErrNoReturnValue:       {SevError, "function %q declared return type %s but has a bare return"},
	ErrVoidReturn:          {SevError, "cannot return a value from void function %q"},
	ErrInvalidOp:           {SevError, "operator %q cannot be applied to types %s and %s"},
	ErrIndexNonList:        {SevError, "cannot index into type %s — only lists and strings are indexable"},
	ErrIndexType:           {SevError, "list index must be int, got %s"},
	ErrAttrNotExist:        {SevError, "type %s has no attribute %q"},
	ErrAttrOnPrimitive:     {SevError, "cannot access attribute %q on primitive type %s"},
	ErrReassignType:        {SevError, "cannot assign %s to variable %q which has type %s"},
	ErrDuplicateVar:        {SevError, "variable %q already declared in this scope"},
	ErrDuplicateFunc:       {SevError, "function %q already defined"},
	ErrDuplicateStruct:     {SevError, "struct %q already defined"},
	ErrStructFieldNotExist: {SevError, "struct %q has no field %q"},
	ErrStructFieldType:     {SevError, "field %q of struct %q: expected %s but got %s"},
	ErrDivideByZero:        {SevError, "division by zero detected"},
	ErrUninitializedVar:    {SevError, "variable %q used before initialization"},
	ErrBreakOutsideLoop:    {SevError, "'break' used outside of a loop"},
	ErrContinueOutsideLoop: {SevError, "'continue' used outside of a loop"},
	ErrReturnOutsideFunc:   {SevError, "'return' used outside of a function"},
	ErrGlobalOutsideFunc:   {SevError, "'global' used outside of a function"},
	ErrNonVoidNoReturn:     {SevError, "function %q has no return statement — return type is %s"},
	ErrInvalidCast:         {SevError, "cannot cast %s to %s"},
	ErrListElemType:        {SevError, "list element at index %d has type %s, expected %s"},
	ErrNegativeIndex:       {SevError, "negative index %d in list literal"},
	ErrSliceOnNonList:      {SevError, "slice operation on non-list type %s"},
	ErrBitwiseOnFloat:      {SevError, "bitwise operator %q cannot be used on float values"},
	ErrBitwiseOnStr:        {SevError, "bitwise operator %q cannot be used on string values"},
	ErrCompareIncompat:     {SevError, "cannot compare %s with %s using %q"},
	ErrNullDeref:           {SevError, "possible null dereference on %q"},
	ErrRecursiveStruct:     {SevError, "struct %q cannot contain itself by value — use a pointer"},
	ErrMainNotDefined:      {SevWarning, "no 'main' function defined — program may not have an entry point"},
	ErrUnusedVar:           {SevWarning, "variable %q is declared but never used"},
	ErrShadowBuiltin:       {SevWarning, "%q shadows a built-in name"},

	ErrForIterType:         {SevError, "'for' loop expects a list or range, got %s"},
	ErrRangeArgCount:       {SevError, "range() takes 1-3 arguments, got %d"},
	ErrRangeArgType:        {SevError, "range() arguments must be int, got %s"},
	ErrDeleteNonPointer:    {SevError, "delete can only be used on heap-allocated objects"},
	ErrImportUnknown:       {SevError, "unknown module %q — only 'stdio', 'math', 'string' are supported"},
	ErrCircularImport:      {SevError, "circular import detected for module %q"},
	ErrStructInitFields:    {SevError, "struct %q initializer: expected %d field(s) but got %d"},
	ErrMissingReturnPath:   {SevError, "function %q may not return a value on all code paths"},
	ErrInvalidFString:      {SevError, "invalid f-string expression"},
	ErrNegativeSlice:       {SevError, "slice indices cannot be negative"},
	ErrLambdaInStmt:        {SevError, "lambda cannot be used as a statement"},
	ErrVoidInExpr:          {SevError, "void expression used in context requiring a value"},
}

// ─── PycError ─────────────────────────────────────────────────────────────────

type PycError struct {
	Code    ErrorCode
	Sev     Severity
	Msg     string
	Line    int
	Col     int
	LineStr string
	Len     int // underline length (default 1)
}

func newError(code ErrorCode, line, col int, lineStr string, args ...interface{}) PycError {
	meta, ok := errorMeta[code]
	if !ok {
		meta.sev = SevError
		meta.tmpl = "unknown error"
	}
	msg := fmt.Sprintf(meta.tmpl, args...)
	return PycError{Code: code, Sev: meta.sev, Msg: msg, Line: line, Col: col, LineStr: lineStr, Len: 1}
}

func newErrorLen(code ErrorCode, line, col, length int, lineStr string, args ...interface{}) PycError {
	e := newError(code, line, col, lineStr, args...)
	e.Len = length
	return e
}

// ─── Pretty Printer ──────────────────────────────────────────────────────────

func (e PycError) Pretty(filename string) string {
	var sb strings.Builder

	// Severity color & label
	var sevColor, sevLabel string
	switch e.Sev {
	case SevError:
		sevColor = colorRed
		sevLabel = "error"
	case SevWarning:
		sevColor = colorYellow
		sevLabel = "warning"
	default:
		sevColor = colorCyan
		sevLabel = "note"
	}

	// Header line
	sb.WriteString(fmt.Sprintf("%s%s%s[E%03d]%s %s%s%s\n",
		colorBold, sevColor, sevLabel, int(e.Code), colorReset,
		colorBold, e.Msg, colorReset))

	// Location
	sb.WriteString(fmt.Sprintf("  %s--> %s%s:%d:%d%s\n",
		colorFaint, colorCyan, filename, e.Line, e.Col, colorReset))

	if e.LineStr == "" {
		return sb.String()
	}

	// Gutter width
	lineNumStr := fmt.Sprintf("%d", e.Line)
	gutter := len(lineNumStr) + 1

	// Empty gutter line
	sb.WriteString(fmt.Sprintf("%s%s |%s\n", colorFaint, strings.Repeat(" ", gutter), colorReset))

	// Source line
	sb.WriteString(fmt.Sprintf("%s%s%s %s|%s %s\n",
		colorFaint, lineNumStr, colorReset,
		colorFaint, colorReset,
		e.LineStr))

	// Underline
	col := e.Col - 1
	if col < 0 {
		col = 0
	}
	if col > len(e.LineStr) {
		col = len(e.LineStr)
	}
	underLen := e.Len
	if underLen < 1 {
		underLen = 1
	}
	if col+underLen > len(e.LineStr)+1 {
		underLen = len(e.LineStr) - col + 1
	}
	if underLen < 1 {
		underLen = 1
	}

	under := strings.Repeat(" ", col) + sevColor + colorBold + strings.Repeat("^", underLen) + colorReset
	sb.WriteString(fmt.Sprintf("%s%s |%s %s\n",
		colorFaint, strings.Repeat(" ", gutter), colorReset, under))

	// Empty gutter line
	sb.WriteString(fmt.Sprintf("%s%s |%s\n", colorFaint, strings.Repeat(" ", gutter), colorReset))

	return sb.String()
}

func PrintErrors(errs []PycError, filename string) {
	errCount := 0
	warnCount := 0
	for _, e := range errs {
		fmt.Print(e.Pretty(filename))
		if e.Sev == SevError {
			errCount++
		} else if e.Sev == SevWarning {
			warnCount++
		}
	}
	if errCount > 0 || warnCount > 0 {
		fmt.Printf("\n%s%sSummary:%s %s%d error(s)%s, %s%d warning(s)%s\n\n",
			colorBold, colorWhite, colorReset,
			colorRed, errCount, colorReset,
			colorYellow, warnCount, colorReset)
	}
}

func HasErrors(errs []PycError) bool {
	for _, e := range errs {
		if e.Sev == SevError {
			return true
		}
	}
	return false
}
