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
║                    v` + version + `                          ║
╚═══════════════════════════════════════════════════════╝

USAGE:
  pyc <command> [options] <file.pyc>

COMMANDS:
  build   <file.pyc>            Transpile to C and compile with gcc
  check   <file.pyc>            Type-check only (no output)
  emit    <file.pyc>            Emit C code to stdout
  run     <file.pyc> [args...]  Build and run immediately
  version                       Print version

OPTIONS:
  -o <output>    Output binary name (default: file without extension)
  -c <output>    Write C output to file instead of temp file
  -O <level>     GCC optimization level 0-3 (default: 0)
  -g             Include debug info
  -nocolor       Disable colored error output
  --             Pass remaining args to the compiled program

LANGUAGE FEATURES:
  • Python-like syntax (def, if/elif/else, while, for..in)
  • Static typing with type inference: x = 42  →  long long x = 42;
  • Explicit types:   x: int = 42
  • Function return types:  def add(a: int, b: int) -> int:
  • Structs:  struct Point: ...
  • Lists:    x: list[int] = [1, 2, 3]
  • F-strings: f"Hello {name}!"
  • Built-ins: print, len, int, str, float, range, append, ...
  • Methods:  s.upper(), s.startswith(), lst.append(), ...
  • new/delete for heap allocation

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
	var files []string
	var passArgs []string

	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "build" || arg == "check" || arg == "emit" || arg == "run" || arg == "version":
			command = arg
		case arg == "-o" && i+1 < len(args):
			i++; outputBin = args[i]
		case arg == "-c" && i+1 < len(args):
			i++; outputC = args[i]
		case arg == "-g":
			debugInfo = true
		case arg == "--nocolor" || arg == "-nocolor":
			noColor = true
		case strings.HasPrefix(arg, "-O") && len(arg) == 3:
			optLevel = string(arg[2])
		case arg == "--":
			passArgs = args[i+1:]
			i = len(args)
		case strings.HasSuffix(arg, ".pyc"):
			files = append(files, arg)
		default:
			// might be positional after command
			if command == "" {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\n", arg)
				os.Exit(1)
			}
			files = append(files, arg)
		}
		i++
	}

	if command == "version" {
		fmt.Println("PyC version", version)
		os.Exit(0)
	}

	if command == "" {
		// Default: build
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
		if !noColor {
			PrintErrors(lexErrs, sourceFile)
		} else {
			for _, e := range lexErrs {
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", sourceFile, e.Line, e.Col, e.Sev, e.Msg)
			}
		}
		if HasErrors(lexErrs) { os.Exit(1) }
	}

	// 2. Parse
	parser := NewParser(tokens, sourceFile)
	prog := parser.ParseProgram()
	parseErrs := parser.Errors()

	if len(parseErrs) > 0 {
		if !noColor {
			PrintErrors(parseErrs, sourceFile)
		} else {
			for _, e := range parseErrs {
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", sourceFile, e.Line, e.Col, e.Sev, e.Msg)
			}
		}
		if HasErrors(parseErrs) { os.Exit(1) }
	}

	// 3. Type Check
	checker := NewTypeChecker(sourceFile, string(source))
	typeErrs := checker.Check(prog)

	allErrs := append(lexErrs, append(parseErrs, typeErrs...)...)
	allErrs = dedupeErrors(allErrs)

	if len(typeErrs) > 0 {
		if !noColor {
			PrintErrors(typeErrs, sourceFile)
		} else {
			for _, e := range typeErrs {
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", sourceFile, e.Line, e.Col, e.Sev, e.Msg)
			}
		}
		if HasErrors(typeErrs) { os.Exit(1) }
	}

	if command == "check" {
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

	// 5. Write C file
	var cFile string
	if outputC != "" {
		cFile = outputC
	} else {
		tmp, err := os.CreateTemp("", "pyc_*.c")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating temp file: %v\n", err)
			os.Exit(1)
		}
		cFile = tmp.Name()
		tmp.Close()
		defer os.Remove(cFile)
	}

	if err := os.WriteFile(cFile, []byte(cCode), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error writing C file: %v\n", err)
		os.Exit(1)
	}

	// 6. Compile with GCC
	if outputBin == "" {
		base := filepath.Base(sourceFile)
		outputBin = strings.TrimSuffix(base, filepath.Ext(base))
	}

	gccArgs := []string{
		"-o", outputBin,
		cFile,
		"-O" + optLevel,
		"-lm",
		"-Wall",
		"-Wno-unused-value",
		"-Wno-int-conversion",
		"-Wno-incompatible-pointer-types",
	}
	if debugInfo {
		gccArgs = append(gccArgs, "-g")
	}

	gccCmd := exec.Command("gcc", gccArgs...)
	gccOut, gccErr := gccCmd.CombinedOutput()
	if gccErr != nil {
		fmt.Fprintf(os.Stderr, "%sGCC compilation failed:%s\n%s\n", colorRed+colorBold, colorReset, string(gccOut))
		fmt.Fprintf(os.Stderr, "Generated C file kept at: %s\n", cFile)
		if outputC == "" {
			// don't delete — user might want to inspect
			fmt.Fprintf(os.Stderr, "(copy of C source written to %s.c for inspection)\n", outputBin)
			os.WriteFile(outputBin+".c", []byte(cCode), 0644)
		}
		os.Exit(1)
	}

	if len(gccOut) > 0 {
		fmt.Fprintf(os.Stderr, "%sGCC warnings:%s\n%s\n", colorYellow, colorReset, string(gccOut))
	}

	if command == "build" {
		fmt.Printf("%s✓ Built %s%s\n", colorGreen, outputBin, colorReset)
		os.Exit(0)
	}

	if command == "run" {
		runArgs := append([]string{"./" + outputBin}, passArgs...)
		cmd := exec.Command(runArgs[0], runArgs[1:]...)
		cmd.Stdin  = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		// cleanup binary after run
		os.Remove(outputBin)
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "run error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
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
