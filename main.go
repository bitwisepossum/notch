package main

import (
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/bitwisepossum/notch/ui"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "notch: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stdin io.Reader) error {
	if len(args) == 0 {
		return runTUI()
	}
	switch args[0] {
	case "ls":
		return cmdLs(stdout)
	case "cat":
		return cmdCat(stdout, args[1:])
	case "add":
		return cmdAdd(args[1:])
	case "done":
		return cmdDone(args[1:])
	case "rm":
		return cmdRm(stdin, args[1:])
	case "version":
		return cmdVersion(stdout)
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runTUI() error {
	p := tea.NewProgram(ui.New())
	_, err := p.Run()
	return err
}
