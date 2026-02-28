package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/bitwisepossum/notch/ui"
)

func main() {
	m := ui.New()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
