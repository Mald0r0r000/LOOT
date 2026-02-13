package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"loot/internal/config"
	"loot/internal/job"
	"loot/internal/offload"
)

type currentView int

const (
	viewMenu currentView = iota
	viewCopy
	viewVolumeInfo
	viewReadme
	viewSettings
	viewCredits
)

type RootModel struct {
	view     currentView
	menu     MenuModel
	copy     Model // Allows re-using existing logic
	settings SettingsModel

	config *config.Config
	// We might need to reset copy model when entering viewCopy

	width    int
	height   int
	quitting bool
}

func NewRootModel(cfg *config.Config) RootModel {
	return RootModel{
		view:     viewMenu,
		menu:     InitialMenuModel(cfg),
		copy:     InitialModelWithConfig(cfg),
		settings: InitialSettingsModel(cfg),
		config:   cfg,
	}
}

func (m RootModel) Init() tea.Cmd {
	return tea.Batch(m.menu.Init(), m.copy.Init())
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Global Back handling for sub-views
		if m.view != viewMenu && (msg.String() == "esc" || msg.String() == "q") {
			m.view = viewMenu
			// Reset copy state, keep config.
			// IMPORTANT: Use Reset() instead of recreating model to preserve the running Queue!
			m.copy.Reset(m.config)
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate resize
		newMenu, _ := m.menu.Update(msg)
		m.menu = newMenu

		newCopy, _ := m.copy.Update(msg)
		m.copy = newCopy.(Model) // Type assertion safe?
		return m, nil
	}

	// Always pass background updates to copy model for state persistence
	switch msg.(type) {
	case job.QueueState, job.Msg, spinner.TickMsg:
		newCopyModel, copyCmd := m.copy.Update(msg)
		m.copy = newCopyModel.(Model)
		return m, copyCmd
	}

	switch m.view {
	case viewMenu:
		newMenu, menuCmd := m.menu.Update(msg)
		m.menu = newMenu
		cmd = menuCmd

		if m.menu.choice != "" {
			choice := m.menu.choice
			m.menu.choice = "" // Reset choice

			switch choice {
			case "OFFLOAD":
				// Debug
				if m.config.JobName != "" {
					// We need to print this visibly or log it.
					// Since TUI is running, stdout is hijacked.
					// Let's rely on the user seeing it in Settings view for now.
					// Or write to a log file.
				}

				m.view = viewCopy
				// Create a clean copy of config to avoid state persistence
				// Create a clean copy of config to avoid state persistence
				cleanCfg := *m.config
				cleanCfg.Source = ""
				cleanCfg.Destination = ""

				// Reset existing model state instead of recreating
				m.copy.Reset(&cleanCfg)

				// Return Init command to reload roots if needed
				// Reset() doesn't return Cmd, so we need to construct it manually or call Init()
				// But copy.Init() starts the Queue again! We don't want that if it's already running.
				// We just want to load roots.
				return m, loadRootsCmd
			case "Volume Info":
				m.view = viewVolumeInfo
			case "ReadMe/Cmd":
				m.view = viewReadme
			case "Settings":
				m.view = viewSettings
			case "Credits":
				m.view = viewCredits
			case "Exit":
				m.quitting = true
				return m, tea.Quit
			}
		}

	case viewCopy:
		newCopyModel, copyCmd := m.copy.Update(msg)
		m.copy = newCopyModel.(Model)
		cmd = copyCmd
		// Check if copy finished?

	case viewSettings:
		newSettings, settingsCmd := m.settings.Update(msg)
		m.settings = newSettings
		cmd = settingsCmd

	case viewVolumeInfo:
		// Static view, handle keys?
		// Handled by global back

	case viewReadme:
		// Handled by global back

	case viewCredits:
		// Handled by global back
	}

	return m, cmd
}

func (m RootModel) View() string {
	if m.quitting {
		return ""
	}

	// For CLI mode (non-interactive), we don't want to render anything from RootModel either
	if m.config != nil && !m.config.Interactive {
		return ""
		// Actually, RootModel delegates to m.copy.View which we just silenced.
		// But RootModel itself prints header.
	}

	header := titleStyle.Render(logoASCII) + "\n"

	switch m.view {
	case viewMenu:
		return header + m.menu.View()
	case viewCopy:
		return m.copy.View()
	case viewSettings:
		return header + m.settings.View()
	case viewVolumeInfo:
		return header + "\n" + renderVolumeInfo()
	case viewReadme:
		return header + "\n" + renderReadme()
	case viewCredits:
		return header + "\n" + renderCredits()
	}

	return "Unknown View"
}

func renderCredits() string {
	return `
    LOOT v1.0
    
    Developed by Mald0r0r000
    
    A high-performance offload tool for media professionals.
    `
}

func renderReadme() string {
	return `
    KEYBOARD SHORTCUTS
    
    [Up/Down]   Navigate Menu / Lists
    [Left/Right] Navigate Directory
    [Enter]     Select Option / Enter Directory
    [Space]     Select Source/Dest
    [Esc/q]     Back / Cancel / Return to Menu
    [Tab]       Toggle Job Manager (in Offload view)
    [x/X]       Cancel Active Job
    [r/R]       Retry Failed/Cancelled Job
    [Ctrl+C]    Quit Application

    CLI COMMANDS
    
    loot                 Interactive Mode (Default)
    loot --version       Show Version
    loot --source <src> --dest <dst> [flags]
    
    Flags:
      --xxhash           Use xxHash (Default)
      --md5              Use MD5
      --sha1             Use SHA1
      --sha256           Use SHA256
      --no-verify        Skip verification
      --resume           Resume interrupted transfer
      --concurrency <N>  Set number of concurrent workers (Default: 4)
      --json             Output JSON for automation
    `
}

func renderVolumeInfo() string {
	// Implement actual volume info gathering here
	volumes, err := offload.GetVolumes()
	if err != nil {
		return fmt.Sprintf("Error listing volumes: %v", err)
	}

	s := "MOUNTED VOLUMES\n\n"
	s += fmt.Sprintf("%-25s %-10s %-10s %-10s %s\n", "NAME", "TOTAL", "USED", "FREE", "PATH")
	s += "--------------------------------------------------------------------------------\n"

	for _, v := range volumes {
		s += fmt.Sprintf("%-25s %-10s %-10s %-10s %s\n",
			truncate(v.Name, 24),
			offload.FormatBytes(v.Total),
			offload.FormatBytes(v.Used),
			offload.FormatBytes(v.Free),
			v.Path,
		)
	}
	return s
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}
