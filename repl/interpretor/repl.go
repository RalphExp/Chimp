package interpretor

import (
	"chimp/evaluator"
	"chimp/lexer"
	"chimp/object"
	"chimp/parser"
	"chimp/token"
	"fmt"
	"io"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	env := object.NewEnvironment()

	fmt.Printf(MONKEY_FACE)
	fmt.Fprintf(out, PROMPT)

	l := lexer.New(in)
	p := parser.New(l)

	for {
		statement := p.ParseStatement()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			break
		}

		if statement == nil && p.GetToken().Type == token.EOF {
			fmt.Printf("Bye!!\n")
			break
		}

		evaluated := evaluator.Eval(statement, env)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
		p.NextToken()
	}
}

const MONKEY_FACE = `            __,__
   .--.  .-"     "-.  .--.
  / .. \/  .-. .-.  \/ .. \
 | |  '|  /   Y   \  |'  | |
 | \   \  \ 0 | 0 /  /   / |
  \ '- ,\.-"""""""-./, -' /
   ''-' /_   ^ ^   _\ '-''
       |  \._   _./  |
       \   \ '~' /   /
        '._ '-=-' _.'
           '-----'
`

func printParserErrors(out io.Writer, errors []string) {
	// io.WriteString(out, MONKEY_FACE)
	io.WriteString(out, "Woops! We ran into some monkey business here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
