package main

import (
	"fmt"
	"os"

	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var src, dst string

	if len(os.Args) >= 3 {
		src = os.Args[1]
		dst = os.Args[2]

		// Verify source exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("Error: Source file '%s' does not exist.\n", src)
			os.Exit(1)
		}
	} else if len(os.Args) == 2 {
		fmt.Println("Usage: loot <source> <destination>")
		fmt.Println("       loot (for interactive mode)")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(src, dst))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
