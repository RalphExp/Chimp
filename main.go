package main

import (
	"chimp/repl/compiler"
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

	if len(os.Args) >= 2 {
		if os.Args[1] == "-vm" {
			compiler.Start(os.Stdin, os.Stdout)
		} else if os.Args[1] == "-eval" {
			interpretor.Start(os.Stdin, os.Stdout)
		}
	}
	interpretor.Start(os.Stdin, os.Stdout)
}
