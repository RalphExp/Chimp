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

	// test case
	s := `
	let a = 1;
	let f1 = func(b) {
		let f2 = func(c) {
			let f3 = func(d) {
				return a + b + c + d
			};
			return f3;
		};
		return f2;
	};
	let x = f1(2);
	let y = x(3);
	y(4);
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
