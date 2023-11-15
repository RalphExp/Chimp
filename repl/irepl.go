package repl

import (
	"chimp/evaluator"
	"chimp/lexer"
	"chimp/object"
	"chimp/parser"
	"chimp/token"
	"fmt"
	"io"
)

func StartInterpreter(in io.Reader, out io.Writer) {
	env := object.NewEnvironment()

	fmt.Fprintf(out, "%s", MONKEY_FACE)
	fmt.Fprintf(out, "%s", PROMPT)

	l := lexer.New(in)
	p := parser.New(l)

	for {
		if p.GetToken().Type == token.EOF {
			fmt.Printf("Bye!!\n")
			break
		}

		statement := p.ParseStatement()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			p.Clear()
			fmt.Fprintf(out, "%s", PROMPT)
			continue
		}

		if statement == nil {
			fmt.Fprintf(out, "\n%s", PROMPT)
			p.NextToken()
			continue
		}

		evaluated := evaluator.Eval(statement, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
		fmt.Fprintf(out, "%s", PROMPT)
		p.NextToken()
	}
}
