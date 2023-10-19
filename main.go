package main

import (
	_ "chimp/repl/compiler"
	"chimp/repl/interpretor"
	"fmt"
	"os"
	"os/user"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hello %s! This is the Chimp programming language!\n",
		user.Username)
	fmt.Printf("Feel free to type in commands\n")

	// s := `let m = 1;`
	// in := strings.NewReader(s)
	// _ = in

	// compiler.Start(os.Stdin, os.Stdout)
	// interpretor.Start(in, os.Stdout)
	interpretor.Start(os.Stdin, os.Stdout)
}
