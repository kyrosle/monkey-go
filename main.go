package main

import (
	"fmt"
	"lexer-parser/repl"
	"os"
	"os/user"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Hello %s! This is the Monkey programming language!\n", user.Username)
	fmt.Printf("Typing Commands\n")
	repl.Start(os.Stdin, os.Stdout)
}
