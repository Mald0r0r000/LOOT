package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"loot/internal/mhl"
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
	stateConfirmAddDest
	stateCopying
	stateVerifying
	stateDone
)

// ... Styles ... (kept same)

// fileItem implements list.Item
type fileItem struct {
	title, desc, path string
	isDir             bool
}

func (i fileItem) Title() string       { return i.title }
func (i fileItem) Description() string { return i.desc }
func (i fileItem) FilterValue() string { return i.title }

type Model struct {
	state   state
	srcList list.Model
	dstList list.Model

	currentPath string // Current browsing path
	srcPath     string
	dstPaths    []string // Multiple destinations

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

	width  int
	height int
}

func InitialModel(src, dst string) Model {
	// Default size, will be updated by WindowSizeMsg, but prevents empty render
	defaultWidth := 80
	defaultHeight := 14

	// Source List
	srcList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	srcList.Title = "Select Source"
	srcList.SetShowHelp(false)

	// Dest List
	dstList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	dstList.Title = "Select Destination"
	dstList.SetShowHelp(false)

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
		state:     initialState,
		srcList:   srcList,
		dstList:   dstList,
		srcPath:   src,
		dstPaths:  []string{}, // Empty initially
		progress:  p,
		offloader: offload.NewOffloader(src, dst),
		status:    "Initializing...",
		width:     defaultWidth,
		height:    defaultHeight,
	}
}

func (m Model) Init() tea.Cmd {
	if m.state == stateCopying {
		m.startTime = time.Now()
		return startCopyWrapper(m.offloader)
	}
	// Init by loading volumes for browsing
	return loadRootsCmd
}

func loadRootsCmd() tea.Msg {
	items := []list.Item{}

	// Quick access to Volumes
	volumes, _ := offload.GetVolumes()
	for _, v := range volumes {
		items = append(items, fileItem{title: "DISK: " + v.Name, desc: v.Path, path: v.Path, isDir: true})
	}

	// Home directory
	home, _ := os.UserHomeDir()
	items = append(items, fileItem{title: "HOME", desc: home, path: home, isDir: true})

	return directoryLoadedMsg{items: items, path: "/"}
}

func loadDirCmd(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
			return errMsg{err} // Need to handle
		}

		items := []list.Item{}
		// Parent directory logic handled by "Left" key, but could add ".." here too

		for _, e := range entries {
			// Skip hidden?
			if len(e.Name()) > 0 && e.Name()[0] == '.' {
				continue
			}

			info, _ := e.Info()
			desc := "File"
			if e.IsDir() {
				desc = "Directory"
			} else {
				desc = fmt.Sprintf("%d bytes", info.Size())
			}

			items = append(items, fileItem{
				title: e.Name(),
				desc:  desc,
				path:  filepath.Join(path, e.Name()),
				isDir: e.IsDir(),
			})
		}

		// Sort: Directories first, then files
		sort.Slice(items, func(i, j int) bool {
			iDir := items[i].(fileItem).isDir
			jDir := items[j].(fileItem).isDir
			if iDir != jDir {
				return iDir
			}
			return items[i].(fileItem).title < items[j].(fileItem).title
		})

		return directoryLoadedMsg{items: items, path: path}
	}
}

type directoryLoadedMsg struct {
	items []list.Item
	path  string
}

type errMsg struct{ err error }

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

		// Common Navigation
		if m.state == stateSelectingSource || m.state == stateSelectingDest {
			var activeList *list.Model
			if m.state == stateSelectingSource {
				activeList = &m.srcList
			} else {
				activeList = &m.dstList
			}

			// Right Key or Enter: Enter Directory
			if msg.String() == "right" || msg.Type == tea.KeyEnter {
				if i, ok := activeList.SelectedItem().(fileItem); ok {
					if i.isDir {
						return m, loadDirCmd(i.path)
					}
				}
				return m, nil
			}

			// Left Key: Go Up
			if msg.String() == "left" {
				if m.currentPath != "/" && m.currentPath != "" {
					parent := filepath.Dir(m.currentPath)
					// If we are at root volumes list (simulated root), don't go further up easily
					// But for now, let's just go up
					return m, loadDirCmd(parent)
				}
				// If at top level specific logic?
				return m, loadRootsCmd
			}

			// Space: Select Folder/File
			if msg.String() == " " {
				if i, ok := activeList.SelectedItem().(fileItem); ok {
					if m.state == stateSelectingSource {
						m.srcPath = i.path
						m.state = stateSelectingDest
						m.currentPath = "" // Reset for Dest Browsing to force loadRootsCmd logic if needed, or better yet, trigger loadRootsCmd
						// We need to trigger loadRootsCmd explicitly for Dest
						return m, loadRootsCmd
					} else {
						// Destination Selected
						selectedDst := i.path
						// If selected a file as dest (unlikely but possible), take dir
						if !i.isDir {
							selectedDst = filepath.Dir(i.path)
						}
						// Logic: Copy src to dst/srcBase
						fileName := filepath.Base(m.srcPath)
						finalDst := filepath.Join(selectedDst, fileName)

						m.dstPaths = append(m.dstPaths, finalDst)

						// Transition to Confirmation
						m.state = stateConfirmAddDest
						return m, nil
					}
				}
			}

			// Refresh
			if msg.String() == "r" {
				if m.currentPath == "" || m.currentPath == "/" {
					return m, loadRootsCmd
				}
				return m, loadDirCmd(m.currentPath)
			}
		}

		// Handle Confirmation State
		if m.state == stateConfirmAddDest {
			switch msg.String() {
			case "y", "Y":
				m.state = stateSelectingDest
				m.currentPath = "" // Reset browsing
				return m, loadRootsCmd
			case "n", "N", "enter":
				m.state = stateCopying
				m.offloader = offload.NewOffloader(m.srcPath, m.dstPaths...)
				m.startTime = time.Now()
				return m, startCopyWrapper(m.offloader)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.srcList.SetWidth(msg.Width)
		m.srcList.SetHeight(14)
		m.dstList.SetWidth(msg.Width)
		m.dstList.SetHeight(14)

	case directoryLoadedMsg:
		m.currentPath = msg.path
		if m.state == stateSelectingSource {
			cmd = m.srcList.SetItems(msg.items)
			m.srcList.Title = "Source Browser: " + msg.path
		} else {
			cmd = m.dstList.SetItems(msg.items)
			m.dstList.Title = "Dest Browser: " + msg.path
		}
		return m, cmd
	}

	// Update active list
	if m.state == stateSelectingSource {
		m.srcList, cmd = m.srcList.Update(msg)
		return m, cmd
	}
	if m.state == stateSelectingDest {
		m.dstList, cmd = m.dstList.Update(msg)
		return m, cmd
	}

	// Copying state update logic (keep existing)
	if m.state == stateCopying || m.state == stateVerifying || m.state == stateDone {
		// ... existing copy logic ...
		// We need to bring back the copy logic which got truncated in previous steps/edits if I'm not careful.
		// Since I am replacing the whole block, I must duplicate the copy logic here.

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

		case verifyFinishedMsg:
			// ... verification logic ...
			m.done = true
			m.verifying = false
			m.state = stateDone
			m.endTime = time.Now()

			if msg.err != nil {
				m.status = fmt.Sprintf("Verification error: %v", msg.err)
				m.err = msg.err
			} else if msg.success {
				m.status = "✅ Verification successful!"

				// Reports
				for _, dst := range m.dstPaths {
					// PDF
					reportPath := dst + ".pdf"
					if err := report.GeneratePDF(reportPath, m.offloader, m.startTime, m.endTime); err != nil {
						m.status += fmt.Sprintf("\n⚠️ Report generation failed for %s: %v", dst, err)
					} else {
						m.status += fmt.Sprintf("\n📄 Report saved to: %s", reportPath)
					}

					// MHL
					mhlPath := dst + ".mhl"
					if err := mhl.GenerateMHL(mhlPath, m.offloader.Files); err != nil {
						m.status += fmt.Sprintf("\n⚠️ MHL generation failed for %s: %v", dst, err)
					} else {
						m.status += fmt.Sprintf("\n📄 MHL saved to: %s", mhlPath)
					}
				}
			} else {
				m.status = "❌ Checksum mismatch!"
				m.err = fmt.Errorf("checksum mismatch")
			}
			return m, tea.Quit
		}

		if _, ok := msg.(progress.FrameMsg); ok {
			progressModel, cmd := m.progress.Update(msg)
			m.progress = progressModel.(progress.Model)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	s := titleStyle.Render(logoASCII) + "\n"

	if m.state == stateSelectingSource {
		s += "BROWSE SOURCE:\n"
		s += "(Right/Enter: Open, Left: Back, Space: Select)\n"
		s += fmt.Sprintf("Path: %s\n\n", m.currentPath)
		s += m.srcList.View()
		return s
	}

	// Handle Confirmation State
	if m.state == stateConfirmAddDest {
		s += fmt.Sprintf("Source: %s\n", m.srcPath)
		s += "Destinations:\n"
		for i, d := range m.dstPaths {
			s += fmt.Sprintf("  %d. %s\n", i+1, d)
		}
		s += "\nAdd another destination? (y/N)"
		return s
	}

	if m.state == stateSelectingDest {
		s += fmt.Sprintf("Source Selected: %s\n", m.srcPath)
		if len(m.dstPaths) > 0 {
			s += "Destinations Selected:\n"
			for _, d := range m.dstPaths {
				s += fmt.Sprintf("  - %s\n", d)
			}
		}
		s += "\nBROWSE DESTINATION:\n"
		s += "(Right/Enter: Open, Left: Back, Space: Select)\n"
		s += fmt.Sprintf("Path: %s\n\n", m.currentPath)
		s += m.dstList.View()
		return s
	}

	// Copying/Verifying/Done View
	s += fmt.Sprintf("Source: %s\n", m.srcPath)
	s += "Destinations:\n"
	for _, d := range m.dstPaths {
		s += fmt.Sprintf("  - %s\n", d)
	}
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
