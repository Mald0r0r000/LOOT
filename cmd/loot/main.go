package main

import (
	"fmt"
	"os"

	"loot/internal/config"
	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// Version will be set via ldflags during build
// Example: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	cfg, err := config.ParseFlags(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Interactive mode
	if cfg.Interactive {
		p := tea.NewProgram(ui.NewRootModel(cfg))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running loot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// CLI mode
	p := tea.NewProgram(ui.InitialModelWithConfig(cfg))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
