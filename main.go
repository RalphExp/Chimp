package main

import (
	_ "chimp/repl/compiler"
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

	s := `let m = 1;`
	in := strings.NewReader(s)
	_ = in

	// let fib = func(x){if(x==0){return 0}else{if(x==1){return 1;}else{return fib(x-1) + fib(x-2);}}};
	// compiler.Start(os.Stdin, os.Stdout)
	// interpretor.Start(in, os.Stdout)
	interpretor.Start(os.Stdin, os.Stdout)
}
