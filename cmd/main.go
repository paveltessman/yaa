package main

import (
	"context"
	"fmt"
	"os"
)

type command struct {
	name        string
	description string
	callback    func(context.Context) error
}

func commands() []command {
	commands := []command{
		{"run", "Run the thing", run},
	}
	return commands
}

func main() {
	if len(os.Args[1:]) != 1 {
		usage()
	}

	for _, c := range commands() {
		if c.name != os.Args[1] {
			continue
		}

		err := c.callback(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	usage()
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: yaa <command>")
	fmt.Fprintln(os.Stderr, "\ncommands:")

	for _, c := range commands() {
		fmt.Fprintf(os.Stderr, "\t%s: %s\n", c.name, c.description)
	}
	os.Exit(1)
}
