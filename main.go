package main

import (
	"chimp/repl/compiler"
	"chimp/repl/interpretor"
	"fmt"
	"os"
	"os/user"
	"strings"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hello %s! This is the Chimp programming language!\n",
		user.Username)
	fmt.Printf("Feel free to type in commands\n")

	s := `
	let newClosure = func(a) { return func() { return a; }; }; let closure = newClosure(99);
	closure();
`
	in := strings.NewReader(s)
	_ = in

	if len(os.Args) >= 2 {
		if os.Args[1] == "-vm" {
			fmt.Printf("engine [vm]\n")
			compiler.Start(os.Stdin, os.Stdout)
			return
		}
	}

	fmt.Printf("engine [interpreter]\n")
	interpretor.Start(os.Stdin, os.Stdout)
}
