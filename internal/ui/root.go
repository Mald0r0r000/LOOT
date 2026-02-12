package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"loot/internal/offload"
)

type currentView int

const (
	viewMenu currentView = iota
	viewCopy
	viewVolumeInfo
	viewReadme
	viewCredits
)

type RootModel struct {
	view currentView
	menu MenuModel
	copy Model // Allows re-using existing logic
	// We might need to reset copy model when entering viewCopy

	width    int
	height   int
	quitting bool
}

func NewRootModel() RootModel {
	return RootModel{
		view: viewMenu,
		menu: InitialMenuModel(),
		// Delay Copy Model init until selected? Or init empty
		copy: InitialModel("", ""),
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
			// If copying is in progress, maybe confirm?
			// For now, let's allow backing out which might kill the flow if not handled.
			// Current CopyModel handles KeyCtrlC but not ESC/q specifically for back.
			// Let's implement robust back.
			m.view = viewMenu
			m.copy = InitialModel("", "") // Reset copy state
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

	switch m.view {
	case viewMenu:
		newMenu, menuCmd := m.menu.Update(msg)
		m.menu = newMenu
		cmd = menuCmd

		if m.menu.choice != "" {
			choice := m.menu.choice
			m.menu.choice = "" // Reset choice

			switch choice {
			case "Backup Simple":
				m.view = viewCopy
				m.copy = InitialModel("", "") // Fresh start
				return m, m.copy.Init()
			case "Offload Multi-Target":
				// Placeholder
			case "Volume Info":
				m.view = viewVolumeInfo
			case "Readme":
				m.view = viewReadme
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
		// The copy model has a 'done' state but stays there.
		// User can press ESC to go back.

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
		return "Bye!\n"
	}

	header := titleStyle.Render(logoASCII) + "\n"

	switch m.view {
	case viewMenu:
		return header + m.menu.View()
	case viewCopy:
		return m.copy.View()
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
    SHORTCUTS
    
    [Up/Down]   Navigate Menu
    [Enter]     Select Option
    [Space]     Toggle Selection (if applicable)
    [Esc/q]     Back / Cancel
    [Ctrl+C]    Quit
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
