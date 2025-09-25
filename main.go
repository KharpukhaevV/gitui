package main

import (
	"fmt"
	"os"

	"github.com/KharpukhaevV/gitui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	model := ui.NewAppModel()
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
