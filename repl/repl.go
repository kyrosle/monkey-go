package repl

import (
	"bufio"
	"fmt"
	"io"
	"lexer-parser/evaluator"
	"lexer-parser/lexer"
	"lexer-parser/object"
	"lexer-parser/parser"
	"strings"
)

const PROMPT = ">> "

const builtinFns = `
let map = fn(arr, f) { let iter = fn(arr, accumulated) { if (len(arr) == 0) { accumulated } else { iter(rest(arr), push(accumulated, f(first(arr)))); } }; iter(arr, []); };
let reduce = fn(arr, initial, f) { let iter = fn(arr, result) { if (len(arr) == 0) { result } else { iter(rest(arr), f(result, first(arr))); } }; iter(arr, initial); };
let sum = fn(arr) { reduce(arr, 0, fn(initial, el) { initial + el }); };
	`

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment()

	l := lexer.New(builtinFns)
	p := parser.New(l)
	program := p.ParseProgram()
	evaluator.Eval(program, env)

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l = lexer.New(line)
		p = parser.New(l)

		program = p.ParseProgram()

		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		evaluator := evaluator.Eval(program, env)
		if evaluator != nil {
			io.WriteString(out, evaluator.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

func StartHandle(raw string) (string, bool) {
	l := lexer.New(builtinFns)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()
	evaluator.Eval(program, env)

	raw = strings.ReplaceAll(raw, "\n", "")
	sr := strings.NewReader(raw)
	scanner := bufio.NewScanner(sr)
	buf := new(strings.Builder)
	buf.Grow(len(raw))

	check := true

	for {
		scanned := scanner.Scan()
		if !scanned {
			return buf.String(), check
		}
		line := scanner.Text()
		l = lexer.New(line)
		p = parser.New(l)

		program = p.ParseProgram()

		if len(p.Errors()) != 0 {
			printParserErrors(buf, p.Errors())
			check = false
			continue
		}

		evaluator := evaluator.Eval(program, env)
		if evaluator != nil {
			io.WriteString(buf, evaluator.Inspect())
			io.WriteString(buf, "\n")
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
