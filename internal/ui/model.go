package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"loot/internal/offload"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("147")).
			MarginTop(1).
			MarginBottom(2)

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
)

// Model defines the application state
type Model struct {
	progress  progress.Model
	offloader *offload.Offloader
	done      bool
	verifying bool
	status    string
	err       error

	totalBytes  int64
	copiedBytes int64
	speed       float64
}

// InitialModel creates the initial model state
func InitialModel(src, dst string) Model {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return Model{
		progress:  p,
		offloader: offload.NewOffloader(src, dst),
		status:    "Initializing...",
	}
}

// Init callback
func (m Model) Init() tea.Cmd {
	return startCopyWrapper(m.offloader)
}

// Messages

type progressMsg struct {
	offload.ProgressInfo
	SourceChannel <-chan offload.ProgressInfo
}

type progressFinishedMsg struct{} // Internal msg when progress channel closes

type copyFinishedMsg struct {
	err error
}

type verifyFinishedMsg struct {
	success bool
	err     error
}

// Commands Wrapper

func startCopyWrapper(o *offload.Offloader) tea.Cmd {
	// Buffered channel to prevent blocking the reader too much
	progressChan := make(chan offload.ProgressInfo, 100)
	resultChan := make(chan error, 1)

	// Start worker
	go func() {
		defer close(progressChan)
		defer close(resultChan)
		err := o.Copy(progressChan)
		resultChan <- err
	}()

	// Return batch of listeners
	return tea.Batch(
		waitForProgress(progressChan),
		waitForResult(resultChan),
	)
}

func waitForProgress(ch <-chan offload.ProgressInfo) tea.Cmd {
	return func() tea.Msg {
		info, ok := <-ch
		if !ok {
			return progressFinishedMsg{}
		}
		return progressMsg{
			ProgressInfo:  info,
			SourceChannel: ch,
		}
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

// Update loop
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

	case progressMsg:
		// Update stats
		m.status = "Copying..."
		m.totalBytes = msg.TotalBytes
		m.copiedBytes = msg.CopiedBytes
		m.speed = msg.Speed

		percent := float64(m.copiedBytes) / float64(m.totalBytes)
		if m.totalBytes == 0 {
			percent = 0
		}

		cmd := m.progress.SetPercent(percent)

		// Continue listening to the same channel
		return m, tea.Batch(cmd, waitForProgress(msg.SourceChannel))

	case progressFinishedMsg:
		return m, nil

	case copyFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = fmt.Sprintf("Copy failed: %v", msg.err)
			m.done = true
			return m, tea.Quit
		}
		m.status = "Verifying..."
		m.verifying = true
		return m, startVerifyCmd(m.offloader)

	case verifyFinishedMsg:
		m.done = true
		m.verifying = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Verification error: %v", msg.err)
			m.err = msg.err
		} else if msg.success {
			m.status = "✅ Verification successful!"
		} else {
			m.status = "❌ Checksum mismatch!"
			m.err = fmt.Errorf("checksum mismatch")
		}
		return m, tea.Quit

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.done {
		if m.err != nil {
			return errorStyle.Render(fmt.Sprintf("\n%s\n", m.status))
		}
		if m.status == "✅ Verification successful!" {
			return completedStyle.Render("\n✨ OFFLOAD COMPLETED") + "\n" +
				completedStyle.Render(m.status) + "\n"
		}
		return m.status
	}

	// Header
	s := titleStyle.Render("💰 LOOT - FAST OFFLOAD") + "\n"

	// File info
	s += fmt.Sprintf("Source: %s\n", m.offloader.Source)
	s += fmt.Sprintf("Dest:   %s\n", m.offloader.Destination)
	s += "\n"

	if m.err != nil {
		return s + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	// Progress
	s += progressStyle.Render(m.progress.View()) +
		percentageStyle.Render(fmt.Sprintf("%.0f%%", m.progress.Percent()*100)) + "\n"

	// Stats
	if m.verifying {
		s += statsStyle.Render("Verifying Checksums...")
	} else {
		speedMB := m.speed / (1024 * 1024)
		s += statsStyle.Render(fmt.Sprintf("%.2f MB/s • %s", speedMB, m.status))
	}

	return s
}
