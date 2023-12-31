package repl

import (
	"chimp/ast"
	"chimp/compiler"
	"chimp/lexer"
	"chimp/object"
	"chimp/parser"
	"chimp/token"
	"chimp/vm"
	"fmt"
	"io"
)

func StartCompiler(in io.Reader, out io.Writer) {
	constants := []object.Object{}
	globals := make([]object.Object, vm.GlobalsSize)

	symbolTable := compiler.NewSymbolTable()
	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

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

		comp := compiler.NewWithState(symbolTable, constants)
		err := comp.Compile(statement)
		if err != nil {
			fmt.Fprintf(out, "Woops! Compilation failed:\n %s\n", err)
			// if compile failed, we should remove all the contents
			// left in the previous buffer
			p.Clear()
			fmt.Fprintf(out, "%s", PROMPT)
			continue
		}

		code := comp.Bytecode()
		fmt.Fprintf(out, "%s\n", code.Instructions.String())

		constants = code.Constants

		machine := vm.NewWithGlobalsStore(code, globals)
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(out, "Woops! Executing bytecode failed:\n %s\n", err)
			fmt.Fprintf(out, "%s", PROMPT)
			continue
		}

		lastPopped := machine.LastPoppedStackElem()
		io.WriteString(out, fmt.Sprintf("stack size: %d\n", machine.GetStackSize()))

		if _, ok := statement.(*ast.ExpressionStatement); ok {
			io.WriteString(out, lastPopped.Inspect())
		} else {
			io.WriteString(out, "nil")
		}
		io.WriteString(out, "\n")

		fmt.Fprintf(out, "%s", PROMPT)
		p.NextToken()
	}
}
