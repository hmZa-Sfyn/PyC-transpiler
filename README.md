# PyC — Python-like → C Transpiler
**Version 0.1.0** — Written entirely in Go (`package main`, no GOPATH required)

PyC is a statically-typed, Python-syntax language that compiles to C via GCC.
It gives you Python's readable syntax with C's performance.

---

## Quick Start

```bash
# 1. Build the transpiler (requires Go 1.21+)
cd pyc/
go build -o pyc .

# 2. Run a program
./pyc run demo.pyc

# 3. Build to binary
./pyc build demo.pyc -o demo
./demo

# 4. Just type-check
./pyc check myfile.pyc

# 5. See emitted C code
./pyc emit myfile.pyc
```

---

## Language Tour

### Variables — Auto Type Inference
```python
x = 42          # → long long x = 42;
pi = 3.14       # → double pi = 3.14;
name = "hello"  # → char* name = "hello";
flag = True     # → int flag = 1;
```

### Explicit Types
```python
count: int = 0
price: float = 9.99
label: str = "PyC"
items: list[int] = [1, 2, 3]
```

### Functions
```python
def add(a: int, b: int) -> int:
    return a + b

def greet(name: str) -> str:
    return f"Hello, {name}!"

def say_hi():       # → void function
    print("Hi!")
```

### If / Elif / Else
```python
if x > 10:
    print("big")
elif x > 5:
    print("medium")
else:
    print("small")
```

### While Loop
```python
i = 0
while i < 10:
    print(i)
    i += 1
```

### For Loop — range()
```python
for i in range(10):      # 0..9
    print(i)

for i in range(1, 11):   # 1..10
    print(i)

for i in range(0, 100, 5):  # 0,5,10,...
    print(i)
```

### For Loop — list iteration
```python
names: list[str] = ["Alice", "Bob", "Charlie"]
for name in names:
    print(name)
```

### Structs
```python
struct Point:
    x: float
    y: float

p = new Point(3.0, 4.0)
print(p.x)     # → p->x
print(p.y)
```

### F-Strings
```python
name = "World"
age = 42
msg = f"Hello {name}, you are {age} years old!"
print(msg)
```

### Lists
```python
nums: list[int] = [1, 2, 3]
nums.append(4)
first = nums[0]
n = len(nums)
popped = nums.pop()
sub = nums[1:3]     # slice
```

### String Methods
```python
s = "  Hello, World!  "
s.strip()           # "Hello, World!"
s.upper()           # "  HELLO, WORLD!  "
s.lower()           # "  hello, world!  "
s.replace("o", "0") # "  Hell0, W0rld!  "
s.startswith("  ")  # True
s.endswith("  ")    # True
s.contains("World") # True
s.split(" ")        # list of words
s.find("World")     # index
s.count("l")        # 3
s.isdigit()         # False
s.isalpha()         # False
```

### List Methods
```python
lst = [3, 1, 4, 1, 5]
lst.append(9)
lst.pop()
lst.pop(0)
lst.contains(4)     # True
lst.len()           # current size
lst.clear()
```

### Ternary Expression
```python
result = "even" if n % 2 == 0 else "odd"
```

### Type Conversions
```python
int("42")       # string → int
float("3.14")   # string → float
str(42)         # int → string
str(3.14)       # float → string
bool(0)         # → False
```

### Heap Allocation (new / delete)
```python
p = new Point(1.0, 2.0)
delete p
```

### Global Variables
```python
counter: int = 0

def increment():
    global counter
    counter += 1
```

### Import (maps to C headers)
```python
import stdio    # stdio.h already included by default
import math     # math.h already included
```

---

## Built-in Functions

| Function | Description |
|---|---|
| `print(...)` | Print values (space separated) |
| `println(...)` | Print + newline |
| `len(x)` | Length of string or list |
| `int(x)` | Convert to int |
| `float(x)` | Convert to float |
| `str(x)` | Convert to string |
| `bool(x)` | Convert to bool |
| `range(n)` | `0..n-1` |
| `range(a,b)` | `a..b-1` |
| `range(a,b,s)` | `a..b-1` step `s` |
| `abs(x)` | Absolute value |
| `max(a,b)` | Maximum |
| `min(a,b)` | Minimum |
| `sqrt(x)` | Square root |
| `floor(x)` | Floor |
| `ceil(x)` | Ceiling |
| `round(x)` | Round |
| `pow(x,y)` | Power |
| `ord(c)` | Char to ASCII int |
| `chr(n)` | ASCII int to char |
| `hex(n)` | Int to hex string |
| `input(prompt)` | Read line from stdin |
| `append(lst, val)` | Append to list |
| `contains(s, sub)` | String/list contains |
| `startswith(s, p)` | String starts with |
| `endswith(s, p)` | String ends with |
| `upper(s)` | Uppercase string |
| `lower(s)` | Lowercase string |
| `strip(s)` | Trim whitespace |
| `replace(s,a,b)` | Replace in string |
| `split(s, delim)` | Split string |
| `join(lst, sep)` | Join list to string |
| `assert(cond)` | Assert condition |
| `exit(code)` | Exit program |
| `rand_int(a,b)` | Random int in [a,b] |
| `rand_float()` | Random float 0..1 |
| `type_of(x)` | Returns type name as string |
| `printf(fmt,...)` | Direct C printf |
| `scanf(fmt,...)` | Direct C scanf |

---

## Error System

PyC has **111+ error codes** organized as:

- **E001–E019** — Lexer errors (unexpected chars, unterminated strings, bad indent)
- **E020–E059** — Syntax errors (missing colon, unexpected token, bad function def)
- **E060–E099** — Type errors (type mismatch, undefined variable, wrong arg count)
- **E100–E111** — Semantic errors (invalid for-iter, bad range args, recursive struct)

### Error Display Example:
```
error[E061] undefined variable "myVar" — did you forget to declare it?
  --> demo.pyc:15:5
   |
15 | result = myVar + 1
   |          ^^^^^ 
   |
```

Errors include:
- Full colored output (red errors, yellow warnings)
- Source line display
- `^^^^^` underline pointing to the exact token
- Error code for easy lookup
- Severity: `error`, `warning`, `note`

---

## CLI Reference

```
pyc build  <file.pyc>           # Compile to binary
pyc run    <file.pyc> [args]    # Compile and run
pyc check  <file.pyc>           # Type-check only
pyc emit   <file.pyc>           # Print generated C code
pyc emit   <file.pyc> -c out.c  # Save C code to file
pyc version                     # Print version

Options:
  -o <name>    Output binary name
  -c <file>    Write C source to file
  -O0..3       Optimization level (passed to gcc)
  -g           Debug symbols
  -nocolor     Disable ANSI colors in error output
  --           Pass remaining args to the program (for run)
```

---

## Type System

| PyC Type | C Type | Notes |
|---|---|---|
| `int` | `long long` | 64-bit signed |
| `float` | `double` | 64-bit float |
| `str` | `char*` | Heap string |
| `bool` | `int` | 1=True, 0=False |
| `void` | `void` | No return value |
| `list[T]` | `PycList*` | Dynamic array |
| `StructName` | `StructName*` | Heap pointer |
| `any` | `void*` | Escape hatch |

---

## File Structure

All files are `package main` — just drop them in one directory:

```
pyc/
├── main.go      # CLI entry point, pipeline orchestration
├── lexer.go     # Tokenizer (handles indent/dedent, all operators)
├── ast.go       # AST node types + Type system
├── errors.go    # 111 error codes + pretty-printer with underlines
├── parser.go    # Recursive-descent parser (Pratt expressions)
├── checker.go   # Type checker + scope analysis
├── codegen.go   # C code generator + PyC runtime
├── go.mod       # Module file (no GOPATH needed)
└── demo.pyc     # Example program
```

Build:
```bash
go build -o pyc .
```

---

## Known Limitations

- No closures / first-class functions (lambdas work in expressions only)
- No exceptions (use `assert` + `exit`)
- Dicts are loosely typed (`void*`)
- No class methods (use structs + standalone functions)
- Nested function definitions emit a warning (not supported in C)
- String indexing returns a temporary char array (not a persistent string)

---

## License
MIT
