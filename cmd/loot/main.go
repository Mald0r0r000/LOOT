package main

import (
	"fmt"
	"os"

	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// For now, we launch directly into the Dashboard.
	// CLI args support can be re-added later by passing them to NewRootModel
	// or implementing a specific "Headless" mode.

	// Check if terminal is interactive?
	// For this version: Always Dashboard.

	p := tea.NewProgram(ui.NewRootModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
