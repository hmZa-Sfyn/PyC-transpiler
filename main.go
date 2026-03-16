package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const version = "0.1.0"

const helpText = `
╔═══════════════════════════════════════════════════════╗
║          PyC — Python-like → C Transpiler             ║
║                    v0.1.0                             ║
╚═══════════════════════════════════════════════════════╝

USAGE:
  pyc <command> [options] <file.pyc>

COMMANDS:
  build   <file.pyc>            Transpile to C and compile with gcc
  check   <file.pyc>            Type-check only (no output)
  emit    <file.pyc>            Emit C code to stdout
  run     <file.pyc> [args...]  Build and run immediately
  version                       Print version
  help                          Show this help

OPTIONS:
  -o <name>      Output binary name (default: file without extension)
  -c <file>      Write C output to this file instead of a temp file
  -O<level>      GCC optimization level 0-3 (default: 0)
  -g             Include debug info (passed to gcc)
  -nocolor       Disable colored error output
  --             Pass remaining args to the compiled program (for 'run')

LANGUAGE FEATURES:
  • Python-like syntax (def, if/elif/else, while, for..in)
  • Type inference:   x = 42  →  long long x = 42;
  • Explicit types:   x: int = 42
  • Return types:     def add(a: int, b: int) -> int:
  • Structs:          struct Point:
  • Lists:            x: list[int] = [1, 2, 3]
  • F-strings:        f"Hello {name}!"
  • Built-ins:        print, len, int, str, float, range, append, ...
  • Methods:          s.upper()  s.startswith()  lst.append()  ...
  • Heap alloc:       new / delete

EXAMPLE:
  # hello.pyc
  def greet(name: str) -> str:
      return f"Hello, {name}!"

  msg = greet("world")
  print(msg)
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Print(helpText)
		os.Exit(0)
	}

	// Flags
	outputBin := ""
	outputC   := ""
	optLevel  := "0"
	debugInfo := false
	noColor   := false
	command   := ""
	var files    []string
	var passArgs []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "build" || arg == "check" || arg == "emit" ||
			arg == "run" || arg == "version" || arg == "help":
			command = arg
		case arg == "-o" && i+1 < len(args):
			i++
			outputBin = args[i]
		case arg == "-c" && i+1 < len(args):
			i++
			outputC = args[i]
		case arg == "-g":
			debugInfo = true
		case arg == "--nocolor" || arg == "-nocolor":
			noColor = true
		case len(arg) == 3 && strings.HasPrefix(arg, "-O"):
			optLevel = string(arg[2])
		case arg == "--":
			passArgs = args[i+1:]
			i = len(args)
		case strings.HasSuffix(arg, ".pyc"):
			files = append(files, arg)
		default:
			// If a command is already set, treat unknown args as files.
			// Otherwise print error.
			if command != "" {
				files = append(files, arg)
			} else {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\nRun 'pyc help' for usage.\n", arg)
				os.Exit(1)
			}
		}
		i++
	}

	// Handle help / version early
	if command == "help" {
		fmt.Print(helpText)
		os.Exit(0)
	}
	if command == "version" {
		fmt.Println("PyC version", version)
		os.Exit(0)
	}

	// Default command when a .pyc file is given with no command
	if command == "" {
		if len(files) > 0 {
			command = "build"
		} else {
			fmt.Print(helpText)
			os.Exit(0)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "error: no input file specified")
		os.Exit(1)
	}

	sourceFile := files[0]
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read %s: %v\n", sourceFile, err)
		os.Exit(1)
	}

	// ── Pipeline ──────────────────────────────────────────────────────────────

	// 1. Lex
	lexer := NewLexer(string(source))
	tokens, lexErrs := lexer.Tokenize()

	if len(lexErrs) > 0 {
		printErrs(lexErrs, sourceFile, noColor)
		if HasErrors(lexErrs) {
			os.Exit(1)
		}
	}

	// 2. Parse
	parser := NewParser(tokens, sourceFile)
	prog := parser.ParseProgram()
	parseErrs := parser.Errors()

	if len(parseErrs) > 0 {
		printErrs(parseErrs, sourceFile, noColor)
		if HasErrors(parseErrs) {
			os.Exit(1)
		}
	}

	// 3. Type Check
	checker := NewTypeChecker(sourceFile, string(source))
	typeErrs := checker.Check(prog)

	if len(typeErrs) > 0 {
		printErrs(typeErrs, sourceFile, noColor)
		if HasErrors(typeErrs) {
			os.Exit(1)
		}
	}

	if command == "check" {
		allErrs := dedupeErrors(append(lexErrs, append(parseErrs, typeErrs...)...))
		if !HasErrors(allErrs) {
			fmt.Printf("%s✓ %s — no errors%s\n", colorGreen, sourceFile, colorReset)
		}
		os.Exit(0)
	}

	// 4. Code Generate
	cgen := NewCGen(checker.structs, checker.functions, sourceFile)
	cCode := cgen.Generate(prog)

	if command == "emit" {
		if outputC != "" {
			if err := os.WriteFile(outputC, []byte(cCode), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "error writing C output: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "C output written to %s\n", outputC)
		} else {
			fmt.Print(cCode)
		}
		os.Exit(0)
	}

	// 5. Write C file (temp or named)
	var cFile string
	if outputC != "" {
		cFile = outputC
		if err := os.WriteFile(cFile, []byte(cCode), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing C file: %v\n", err)
			os.Exit(1)
		}
	} else {
		tmp, err := os.CreateTemp("", "pyc_*.c")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating temp file: %v\n", err)
			os.Exit(1)
		}
		cFile = tmp.Name()
		if _, err := tmp.WriteString(cCode); err != nil {
			fmt.Fprintf(os.Stderr, "error writing temp C file: %v\n", err)
			os.Exit(1)
		}
		tmp.Close()
		defer os.Remove(cFile)
	}

	// 6. Derive output binary name
	if outputBin == "" {
		base := filepath.Base(sourceFile)
		outputBin = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// 7. Compile with GCC
	gccArgs := []string{
		"-o", outputBin,
		cFile,
		"-O" + optLevel,
		"-lm",
		"-Wall",
		"-Wno-unused-value",
		"-Wno-int-conversion",
		"-Wno-incompatible-pointer-types",
		"-Wno-implicit-function-declaration",
	}
	if debugInfo {
		gccArgs = append(gccArgs, "-g")
	}

	gccCmd := exec.Command("gcc", gccArgs...)
	gccOut, gccErr := gccCmd.CombinedOutput()
	if gccErr != nil {
		fmt.Fprintf(os.Stderr, "%s%sGCC compilation failed:%s\n%s\n",
			colorRed, colorBold, colorReset, string(gccOut))
		// Save C source for inspection
		inspectFile := outputBin + "_debug.c"
		_ = os.WriteFile(inspectFile, []byte(cCode), 0644)
		fmt.Fprintf(os.Stderr, "Generated C saved to: %s\n", inspectFile)
		os.Exit(1)
	}

	if len(gccOut) > 0 {
		fmt.Fprintf(os.Stderr, "%sGCC warnings:%s\n%s\n", colorYellow, colorReset, string(gccOut))
	}

	if command == "build" {
		fmt.Printf("%s✓ Built '%s'%s\n", colorGreen, outputBin, colorReset)
		os.Exit(0)
	}

	if command == "run" {
		runArgs := append([]string{"./" + outputBin}, passArgs...)
		cmd := exec.Command(runArgs[0], runArgs[1:]...)
		cmd.Stdin  = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		runErr := cmd.Run()
		os.Remove(outputBin) // clean up binary after run
		if runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "run error: %v\n", runErr)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printErrs(errs []PycError, filename string, noColor bool) {
	if noColor {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s:%d:%d: %s[E%03d]: %s\n",
				filename, e.Line, e.Col, e.Sev, int(e.Code), e.Msg)
		}
	} else {
		PrintErrors(errs, filename)
	}
}

func dedupeErrors(errs []PycError) []PycError {
	seen := map[string]bool{}
	var out []PycError
	for _, e := range errs {
		key := fmt.Sprintf("%d:%d:%d:%s", e.Line, e.Col, e.Code, e.Msg)
		if !seen[key] {
			seen[key] = true
			out = append(out, e)
		}
	}
	return out
}
