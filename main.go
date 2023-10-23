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
			fmt.Printf("engine [vm]\n")
			compiler.Start(os.Stdin, os.Stdout)
			return
		}
	}

	fmt.Printf("engine [interpreter]\n")
	interpretor.Start(os.Stdin, os.Stdout)
}
