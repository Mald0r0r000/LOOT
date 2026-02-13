package main

import (
	"fmt"
	"os"

	"loot/internal/config"
	"loot/internal/offload"
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

	// Dry Run
	if cfg.DryRun {
		o := offload.NewOffloaderWithConfig(cfg, cfg.Source, cfg.Destination)
		res, err := o.DryRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during dry run: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("=== DRY RUN SUMMARY ===")
		fmt.Printf("Source: %s\n", res.Source)
		fmt.Printf("Files found: %d\n", len(res.Files))
		fmt.Printf("Total Size: %s\n", offload.FormatBytes(uint64(res.TotalSize)))
		fmt.Println("\nDestinations:")
		for _, dest := range res.Destinations {
			status := "✅ OK"
			if !dest.CanFit {
				status = "❌ INSUFFICIENT SPACE"
			}
			fmt.Printf("  - %s\n", dest.Path)
			fmt.Printf("    Free Space: %s\n", offload.FormatBytes(dest.FreeSpace))
			fmt.Printf("    Status: %s\n", status)
		}
		return
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
