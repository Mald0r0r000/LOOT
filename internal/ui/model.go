package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"loot/internal/offload"
	"loot/internal/report"
)

const logoASCII = `
██╗      ██████╗  ██████╗ ████████╗
██║     ██╔══/██╗██╔══/██╗╚══██╔══╝
██║     ██║ / ██║██║ / ██║   ██║   
██║     ██║/  ██║██║/  ██║   ██║   
███████╗╚██████╔╝╚██████╔╝   ██║   
╚══════╝ ╚═════╝  ╚═════╝    ╚═╝   
`

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("202")). // Orange/Gold for Loot
			MarginTop(1).
			MarginBottom(1)

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			MarginBottom(1)

	percentageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120")).
			MarginLeft(1)

	completedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")).
			MarginTop(1)

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			MarginTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginTop(1)

	instructionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				MarginTop(1).
				Italic(true)
)

type state int

const (
	stateSelectingSource state = iota
	stateSelectingDest
	stateCopying
	stateVerifying
	stateDone
)

// volItem implements list.Item
type volItem struct {
	title, desc, path string
}

func (i volItem) Title() string       { return i.title }
func (i volItem) Description() string { return i.desc }
func (i volItem) FilterValue() string { return i.title }

type Model struct {
	state      state
	filepicker filepicker.Model
	volList    list.Model

	srcPath string
	dstPath string

	progress  progress.Model
	offloader *offload.Offloader

	done      bool
	verifying bool
	status    string
	err       error

	totalBytes  int64
	copiedBytes int64
	speed       float64

	startTime time.Time
	endTime   time.Time
}

func InitialModel(src, dst string) Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{} // All files
	fp.CurrentDirectory, _ = os.Getwd()
	fp.ShowHidden = true

	// Initialize empty list, we'll populate it when needed or now
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Destination Volume"
	l.SetShowHelp(false)

	initialState := stateSelectingSource
	if src != "" && dst != "" {
		initialState = stateCopying
	}

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return Model{
		state:      initialState,
		filepicker: fp,
		volList:    l,
		srcPath:    src,
		dstPath:    dst,
		progress:   p,
		offloader:  offload.NewOffloader(src, dst),
		status:     "Initializing...",
	}
}

func (m Model) Init() tea.Cmd {
	if m.state == stateCopying {
		// If started with CLI args
		m.startTime = time.Now() // Start timing
		return startCopyWrapper(m.offloader)
	}
	// Always init filepicker just in case
	return m.filepicker.Init() // Do we need to init volList via Cmd? Usually not unless it has IO.
}

// ... Messages ...
type progressMsg struct {
	offload.ProgressInfo
	SourceChannel <-chan offload.ProgressInfo
}
type progressFinishedMsg struct{}
type copyFinishedMsg struct{ err error }
type verifyFinishedMsg struct {
	success bool
	err     error
}

// Commands Wrapper (same as before)
func startCopyWrapper(o *offload.Offloader) tea.Cmd {
	progressChan := make(chan offload.ProgressInfo, 100)
	resultChan := make(chan error, 1)

	go func() {
		defer close(progressChan)
		defer close(resultChan)
		err := o.Copy(progressChan)
		resultChan <- err
	}()

	return tea.Batch(waitForProgress(progressChan), waitForResult(resultChan))
}

func waitForProgress(ch <-chan offload.ProgressInfo) tea.Cmd {
	return func() tea.Msg {
		info, ok := <-ch
		if !ok {
			return progressFinishedMsg{}
		}
		return progressMsg{ProgressInfo: info, SourceChannel: ch}
	}
}

func waitForResult(ch <-chan error) tea.Cmd {
	return func() tea.Msg {
		err := <-ch
		return copyFinishedMsg{err: err}
	}
}

func startVerifyCmd(o *offload.Offloader) tea.Cmd {
	return func() tea.Msg {
		success, err := o.Verify()
		return verifyFinishedMsg{success: success, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.volList.SetWidth(msg.Width)
		m.volList.SetHeight(14) // Fixed height for list
	}

	switch m.state {
	case stateSelectingSource:
		m.filepicker, cmd = m.filepicker.Update(msg)
		if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
			m.srcPath = path
			m.state = stateSelectingDest

			// Load Volumes
			volumes, err := offload.GetVolumes()
			var items []list.Item
			if err != nil {
				// Handle error? Just show empty or error
				items = append(items, volItem{title: "Error listing volumes", desc: err.Error()})
			} else {
				for _, v := range volumes {
					items = append(items, volItem{title: v.Name, desc: v.Path, path: v.Path})
				}
			}
			cmd = m.volList.SetItems(items)
			return m, cmd
		}
		return m, cmd

	case stateSelectingDest:
		m.volList, cmd = m.volList.Update(msg)
		if msg, ok := msg.(tea.KeyMsg); ok && msg.Type == tea.KeyEnter {
			if selectedItem, ok := m.volList.SelectedItem().(volItem); ok {
				if selectedItem.path != "" {
					// Construct destination path: Volume/Filename
					// We should probably check if m.srcPath is not empty, but flow guarantees it
					fileName := filepath.Base(m.srcPath)
					m.dstPath = filepath.Join(selectedItem.path, fileName) // Default to root of volume

					// Transition to copy
					m.state = stateCopying
					m.offloader = offload.NewOffloader(m.srcPath, m.dstPath)
					m.startTime = time.Now()
					return m, startCopyWrapper(m.offloader)
				}
			}
		}
		return m, cmd

	case stateCopying:
		switch msg := msg.(type) {
		case progressMsg:
			m.status = "Copying..."
			m.totalBytes = msg.TotalBytes
			m.copiedBytes = msg.CopiedBytes
			m.speed = msg.Speed
			percent := float64(m.copiedBytes) / float64(m.totalBytes)
			if m.totalBytes == 0 {
				percent = 0
			}
			cmd = m.progress.SetPercent(percent)
			return m, tea.Batch(cmd, waitForProgress(msg.SourceChannel))

		case progressFinishedMsg:
			return m, nil

		case copyFinishedMsg:
			if msg.err != nil {
				m.err = msg.err
				m.status = fmt.Sprintf("Copy failed: %v", msg.err)
				m.done = true
				m.state = stateDone
				return m, tea.Quit
			}
			m.status = "Verifying..."
			m.verifying = true
			m.state = stateVerifying
			return m, startVerifyCmd(m.offloader)
		}

		if _, ok := msg.(progress.FrameMsg); ok {
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			return m, cmd
		}

	case stateVerifying:
		if msg, ok := msg.(verifyFinishedMsg); ok {
			m.done = true
			m.verifying = false
			m.state = stateDone
			m.endTime = time.Now()

			if msg.err != nil {
				m.status = fmt.Sprintf("Verification error: %v", msg.err)
				m.err = msg.err
			} else if msg.success {
				m.status = "✅ Verification successful!"

				// Generate Report
				reportPath := m.dstPath + ".pdf"
				if err := report.GeneratePDF(reportPath, m.offloader, m.startTime, m.endTime); err != nil {
					m.status += fmt.Sprintf("\n⚠️ Report generation failed: %v", err)
				} else {
					m.status += fmt.Sprintf("\n📄 Report saved to: %s", reportPath)
				}
			} else {
				m.status = "❌ Checksum mismatch!"
				m.err = fmt.Errorf("checksum mismatch")
			}
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	s := titleStyle.Render(logoASCII) + "\n"

	if m.state == stateSelectingSource {
		s += "Select Source File:\n\n"
		s += m.filepicker.View() + "\n"
		s += instructionStyle.Render("Use arrow keys to navigate, Enter to select.")
		return s
	}

	if m.state == stateSelectingDest {
		s += fmt.Sprintf("Source: %s\n", m.srcPath)
		s += "Select Destination Volume:\n"
		s += m.volList.View()
		return s
	}

	// Copying/Verifying/Done View
	s += fmt.Sprintf("Source: %s\n", m.srcPath)
	s += fmt.Sprintf("Dest:   %s\n", m.dstPath)
	s += "\n"

	if m.err != nil {
		return s + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.done {
		if m.status == "✅ Verification successful!" {
			return s + completedStyle.Render(m.status) + "\n"
		}
		return s + m.status
	}

	s += progressStyle.Render(m.progress.View()) +
		percentageStyle.Render(fmt.Sprintf("%.0f%%", m.progress.Percent()*100)) + "\n"

	if m.verifying {
		s += statsStyle.Render("Verifying Checksums...")
	} else {
		speedMB := m.speed / (1024 * 1024)
		s += statsStyle.Render(fmt.Sprintf("%.2f MB/s • %s", speedMB, m.status))
	}

	return s
}
