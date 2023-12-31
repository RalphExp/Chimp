package main

import (
	"chimp/repl"
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

	// test case
	s := `for (a = 0; a < 10; a+=1) {}
`
	in := strings.NewReader(s)
	_ = in

	if len(os.Args) >= 2 {
		if os.Args[1] == "-vm" {
			fmt.Printf("engine [vm]\n")
			repl.StartCompiler(os.Stdin, os.Stdout)
			return
		}
	}

	fmt.Printf("engine [interpreter]\n")
	repl.StartInterpreter(os.Stdin, os.Stdout)
}
