package main

import (
	"fmt"
	"os"

	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: loot <source> <destination>")
		fmt.Println("Example: loot ./large_file.mov /Volumes/Backup/large_file.mov")
		os.Exit(1)
	}

	src := os.Args[1]
	dst := os.Args[2]

	// Verify source exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		fmt.Printf("Error: Source file '%s' does not exist.\n", src)
		os.Exit(1)
	}

	// Check if source is a directory (not supported yet)
	info, err := os.Stat(src)
	if err == nil && info.IsDir() {
		fmt.Printf("Error: Source '%s' is a directory. Directory recursion is not yet implemented.\n", src)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(src, dst))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
