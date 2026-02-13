package ui

import (
	"fmt"

	"loot/internal/config"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type settingsItem struct {
	title, desc string
}

func (i settingsItem) Title() string       { return i.title }
func (i settingsItem) Description() string { return i.desc }
func (i settingsItem) FilterValue() string { return i.title }

type SettingsModel struct {
	list      list.Model
	config    *config.Config
	width     int
	height    int
	textInput textinput.Model
	editing   bool
}

func InitialSettingsModel(cfg *config.Config) SettingsModel {
	items := []list.Item{
		settingsItem{title: "Job Name", desc: "Set job name for report metadata (Enter to edit)"},
		settingsItem{title: "Camera", desc: "Set camera identifier (Enter to edit)"},
		settingsItem{title: "Reel", desc: "Set reel identifier (Enter to edit)"},
		settingsItem{title: "Hash Algorithm", desc: "Select checksum algorithm (Space/Enter to cycle)"},
		settingsItem{title: "Metadata Mode", desc: "Select extraction strategy (Space/Enter to cycle)"},
		settingsItem{title: "Dry Run Mode", desc: "Simulate transfer without copying (Space/Enter to toggle)"},
	}

	const defaultWidth = 20
	const defaultHeight = 14

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	l.Title = "SETTINGS"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	ti := textinput.New()
	ti.Placeholder = "Enter value..."
	ti.CharLimit = 50
	ti.Width = 30

	return SettingsModel{
		list:      l,
		config:    cfg,
		width:     defaultWidth,
		height:    defaultHeight,
		textInput: ti,
		editing:   false,
	}
}

func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmd tea.Cmd

	// Handle text input when editing
	if m.editing {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				// Determine which field we are editing based on selected item
				if i, ok := m.list.SelectedItem().(settingsItem); ok {
					switch i.title {
					case "Job Name":
						m.config.JobName = m.textInput.Value()
					case "Camera":
						m.config.Camera = m.textInput.Value()
					case "Reel":
						m.config.Reel = m.textInput.Value()
					}
				}
				m.editing = false
				m.textInput.Blur()
				return m, nil
			}
			if msg.Type == tea.KeyEsc {
				m.editing = false
				m.textInput.Blur()
				return m, nil
			}
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == " " || msg.String() == "enter" {
			selectedItem := m.list.SelectedItem()
			if selectedItem == nil {
				return m, nil
			}
			if i, ok := selectedItem.(settingsItem); ok {
				switch i.title {
				case "Hash Algorithm":
					m.cycleHashAlgo()
				case "Metadata Mode":
					m.cycleMetadataMode()
				case "Dry Run Mode":
					m.config.DryRun = !m.config.DryRun
				case "Job Name":
					m.editing = true
					m.textInput.SetValue(m.config.JobName)
					m.textInput.Placeholder = "Enter job name..."
					m.textInput.Focus()
					return m, textinput.Blink
				case "Camera":
					m.editing = true
					m.textInput.SetValue(m.config.Camera)
					m.textInput.Placeholder = "Enter camera ID..."
					m.textInput.Focus()
					return m, textinput.Blink
				case "Reel":
					m.editing = true
					m.textInput.SetValue(m.config.Reel)
					m.textInput.Placeholder = "Enter reel ID..."
					m.textInput.Focus()
					return m, textinput.Blink
				}
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(14)
	}

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
	default:
		m.config.Algorithm = config.AlgoXXHash64
	}
}

func (m *SettingsModel) cycleMetadataMode() {
	current := m.config.MetadataMode
	switch current {
	case "hybrid":
		m.config.MetadataMode = "header"
	case "header":
		m.config.MetadataMode = "exiftool"
	case "exiftool":
		m.config.MetadataMode = "off"
	case "off":
		m.config.MetadataMode = "hybrid"
	default:
		m.config.MetadataMode = "hybrid"
	}
}

func (m SettingsModel) View() string {
	if m.editing {
		return fmt.Sprintf(
			"\n%s\n\n%s\n\n%s",
			titleStyle.Render("EDIT JOB NAME"),
			m.textInput.View(),
			instructionStyle.Render("(Enter and Save, Esc to Cancel)"),
		)
	}

	s := "\n" + m.list.View()

	// Status pane
	dryRunStatus := "OFF"
	if m.config.DryRun {
		dryRunStatus = "ON"
	}

	jobName := m.config.JobName
	if jobName == "" {
		jobName = "(none)"
	}

	camera := m.config.Camera
	if camera == "" {
		camera = "(none)"
	}

	reel := m.config.Reel
	if reel == "" {
		reel = "(none)"
	}

	status := fmt.Sprintf("\nCurrent Configuration:\n\nJob Name:       %s\nCamera:         %s\nReel:           %s\nHash Algorithm: %s\nMetadata Mode:  %s\nDry Run Mode:   %s",
		titleStyle.Render(jobName),
		titleStyle.Render(camera),
		titleStyle.Render(reel),
		titleStyle.Render(string(m.config.Algorithm)),
		titleStyle.Render(m.config.MetadataMode),
		titleStyle.Render(dryRunStatus))
	s += "\n" + status

	s += "\n\n" + instructionStyle.Render("(Press Space/Enter to change, Esc/q to return)")

	return s
}
