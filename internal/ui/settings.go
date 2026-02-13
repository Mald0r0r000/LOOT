package ui

import (
	"fmt"

	"loot/internal/config"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type settingsItem struct {
	title, desc string
}

func (i settingsItem) Title() string       { return i.title }
func (i settingsItem) Description() string { return i.desc }
func (i settingsItem) FilterValue() string { return i.title }

type SettingsModel struct {
	list   list.Model
	config *config.Config
	width  int
	height int
}

func InitialSettingsModel(cfg *config.Config) SettingsModel {
	items := []list.Item{
		settingsItem{title: "Hash Algorithm", desc: "Select checksum algorithm (Space/Enter to cycle)"},
	}

	const defaultWidth = 20
	const defaultHeight = 14

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	l.Title = "SETTINGS"
	l.SetShowHelp(false) // We'll render our own help or rely on default
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return SettingsModel{
		list:   l,
		config: cfg,
		width:  defaultWidth,
		height: defaultHeight,
	}
}

func (m SettingsModel) Init() tea.Cmd {
	return nil
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == " " || msg.String() == "enter" {
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}
			// currently only one item, but good to be generic
			if i, ok := selectedItem.(settingsItem); ok {
				if i.title == "Hash Algorithm" {
					m.cycleHashAlgo()
				}
			}
			return m, nil // Don't propagate enter to list if handled
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(14)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *SettingsModel) cycleHashAlgo() {
	current := m.config.Algorithm
	switch current {
	case config.AlgoXXHash64:
		m.config.Algorithm = config.AlgoMD5
	case config.AlgoMD5:
		m.config.Algorithm = config.AlgoSHA256
	case config.AlgoSHA256:
		m.config.Algorithm = config.AlgoXXHash64
	// SHA1 missing in config? Wait, check config.go, only xxhash, md5, sha256 defined.
	default:
		m.config.Algorithm = config.AlgoXXHash64
	}
}

func (m SettingsModel) View() string {
	// We want to show the current value in the description or separately
	// Since list items are static strings in BubbleTea's default list (mostly),
	// we might need to update the item description dynamically or render a status below.

	// Actually, let's just render the list and then the current value below it.

	s := "\n" + m.list.View()

	// Status pane
	status := fmt.Sprintf("\nCurrent Configuration:\n\nHash Algorithm: %s", titleStyle.Render(string(m.config.Algorithm)))
	s += "\n" + status

	s += "\n\n" + instructionStyle.Render("(Press Space/Enter to change, Esc/q to return)")

	return s
}
