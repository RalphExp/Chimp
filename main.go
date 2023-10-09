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

	// in := strings.NewReader(`
	//     let fibonacci = func(x) {
	// 		if (x == 0) {
	// 		  return 0
	// 		} else {
	// 		  if (x == 1) {
	// 			return 1;
	// 		  } else {
	// 			return fibonacci(x - 1) + fibonacci(x - 2);
	// 		  }
	// 		}
	// 	};
	// 	puts(fibonacci)
	// 	puts("may")
	// 	puts(fibonacci(0))
	// `)

	// let fib = func(x){if(x==0){return 0}else{if(x==1){return 1;}else{return fib(x-1) + fib(x-2);}}};
	// compiler.Start(os.Stdin, os.Stdout)
	interpretor.Start(os.Stdin, os.Stdout)
}
