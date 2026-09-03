package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type command struct {
	name        string
	description string
	callback    func(context.Context, []string) error
}

func commands() []command {
	commands := []command{
		{"run", "Run the thing", run},
		{"migrate", "Apply or roll back database migrations (up|down|status|version)", migrate},
	}
	return commands
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := do(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v", err)
		os.Exit(1)
	}
}

func do(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("no subcommand given")
	}

	for _, c := range commands() {
		if c.name != args[0] {
			continue
		}

		return c.callback(ctx, args[1:])
	}
	usage()
	return fmt.Errorf("unknown subcommand: %s", args[0])
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: yaa <command> [args]")
	fmt.Fprintln(os.Stderr, "\ncommands:")

	for _, c := range commands() {
		fmt.Fprintf(os.Stderr, "\t%s: %s\n", c.name, c.description)
	}
}
