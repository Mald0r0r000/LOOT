package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"loot/internal/config"
	"loot/internal/job"
	"loot/internal/offload"
	"loot/internal/output"
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
	stateJobManager
	stateErrorDetails
	stateDryRun
)

// fileItem implements list.Item
type fileItem struct {
	title, desc, path string
	isDir             bool
}

type jobItem struct {
	j *job.Job
}

func (i jobItem) Title() string {
	statusIcon := "⏳"
	switch i.j.Status {
	case job.StatusRunning, job.StatusCopying, job.StatusVerifying:
		statusIcon = "🚀"
	case job.StatusCompleted:
		statusIcon = "✅"
	case job.StatusFailed:
		statusIcon = "❌"
	case job.StatusCancelled:
		statusIcon = "🚫"
	}
	return fmt.Sprintf("%s %s", statusIcon, filepath.Base(i.j.Offloader.Source))
}

func (i jobItem) Description() string {
	dest := "Multiple"
	if len(i.j.Offloader.Destinations) == 1 {
		dest = filepath.Base(i.j.Offloader.Destinations[0])
	}
	return fmt.Sprintf("ID: %s | To: %s | Status: %s", i.j.ID, dest, i.j.Status)
}
func (i jobItem) FilterValue() string { return i.j.ID }

func (i fileItem) Title() string       { return i.title }
func (i fileItem) Description() string { return i.desc }
func (i fileItem) FilterValue() string { return i.title }

type Model struct {
	state         state
	previousState state // To return from Job Manager
	srcList       list.Model
	dstList       list.Model

	currentPath string // Current browsing path
	srcPath     string
	dstPaths    []string // Multiple destinations

	progress progress.Model

	// Job Management
	queue      *job.Queue
	msgChan    chan job.Msg // Persistent channel for job updates
	queueState job.QueueState
	jobList    list.Model // List component for Job Manager

	CurrentJob *job.Job // Keep track of active job for display details

	// Display state
	status  string
	err     error
	spinner spinner.Model

	width  int
	height int

	// Dry Run
	dryRunResult *offload.DryRunResult

	config *config.Config
}

func InitialModel(src, dst string) Model {
	return InitialModelWithConfig(config.DefaultConfig())
}

func InitialModelWithConfig(cfg *config.Config) Model {
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

	// Job List
	jobList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	jobList.Title = "Job Queue (Tab: Toggle View, X: Cancel, R: Retry/Resume)"
	jobList.SetShowHelp(false)

	initialState := stateSelectingSource
	if cfg.Source != "" && cfg.Destination != "" {
		initialState = stateCopying
	}

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize dstPaths from config if present
	dstPaths := []string{}
	if cfg.Destination != "" {
		dstPaths = append(dstPaths, cfg.Destination)
	}

	// Initialize Queue
	q := job.NewQueue()
	msgChan := make(chan job.Msg, 100)

	var currentJob *job.Job

	if initialState == stateCopying {
		currentJob = job.NewJob(cfg)
		// Force update destinations in offloader (hack until config refactor)
		currentJob.Offloader.Destinations = dstPaths
		q.Add(currentJob)
	}

	return Model{
		state:      initialState,
		srcList:    srcList,
		dstList:    dstList,
		jobList:    jobList,
		srcPath:    cfg.Source,
		dstPaths:   dstPaths,
		progress:   p,
		spinner:    s,
		queue:      q,
		msgChan:    msgChan,
		CurrentJob: currentJob, // Initial active job (if likely to start immediately/soon)
		status:     "Initializing...",
		width:      defaultWidth,
		height:     defaultHeight,
		config:     cfg,
	}
}

func (m *Model) Reset(cfg *config.Config) {
	m.state = stateSelectingSource
	if cfg.Source != "" && cfg.Destination != "" {
		m.state = stateCopying
	}
	m.srcPath = cfg.Source
	m.dstPaths = []string{}
	if cfg.Destination != "" {
		m.dstPaths = append(m.dstPaths, cfg.Destination)
	}
	m.currentPath = ""
	m.status = "Ready"
	m.err = nil
	m.config = cfg

	// Reset lists titles/content?
	// Maybe keep browsing history?
	// Better to reset browsing to roots?
	// We will trigger loadRootsCmd from Init or manually if needed?
	// Let's rely on the Update loop catching the state change or passing a Cmd.
}

func (m Model) Init() tea.Cmd {
	// Start Queue Processing
	m.queue.Start(m.msgChan)

	cmds := []tea.Cmd{
		waitForJobMsg(m.msgChan),
		waitForQueueState(m.queue.UpdateChan),
		m.spinner.Tick, // Start spinner
	}

	if m.state == stateCopying {
		// Already added in InitialModel
	} else {
		cmds = append(cmds, loadRootsCmd)
	}

	return tea.Batch(cmds...)
}

func waitForQueueState(ch <-chan job.QueueState) tea.Cmd {
	return func() tea.Msg {
		state, ok := <-ch
		if !ok {
			return nil
		}
		return state
	}
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

// Wrapper to run job in goroutine and pipe messages to tea
func startJobCmd(j *job.Job) tea.Cmd {
	ch := make(chan job.Msg, 10)
	go func() {
		defer close(ch)
		j.Run(ch)
	}()
	return waitForJobMsg(ch)
}

func waitForJobMsg(ch <-chan job.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil // Channel closed, job finished
		}
		return msg
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		// Quick Cancel in Copy/Verify view
		if (m.state == stateCopying || m.state == stateVerifying) && (msg.String() == "x" || msg.String() == "X") {
			if m.CurrentJob != nil {
				m.queue.CancelJob(m.CurrentJob.ID)
				m.status = "Cancelling..."
			}
			return m, nil
		}

		// Toggle Job Manager
		if msg.String() == "tab" {
			if m.state == stateJobManager {
				// Restore previous state
				m.state = m.previousState
				// If we returned to a completed state but queue is done, maybe verify?
				// For now just restore.
				if m.state == stateJobManager { // Fallback
					m.state = stateSelectingSource
				}
			} else {
				m.previousState = m.state
				m.state = stateJobManager
			}
			return m, nil
		}

		// Job Manager Logic
		if m.state == stateJobManager {
			if msg.String() == "x" || msg.String() == "X" {
				if i, ok := m.jobList.SelectedItem().(jobItem); ok {
					m.queue.CancelJob(i.j.ID)
					// State update will come via QueueState
				}
			}

			if msg.String() == "r" || msg.String() == "R" {
				if i, ok := m.jobList.SelectedItem().(jobItem); ok {
					// Allow retrying finished/failed/cancelled jobs
					if i.j.Status == job.StatusFailed || i.j.Status == job.StatusCancelled || i.j.Status == job.StatusCompleted {
						// Clone config
						newCfg := *i.j.Config
						newCfg.SkipExisting = true // Enable resume mode

						// Create new Job
						newJob := job.NewJob(&newCfg)
						// Restore destinations (important for multiple dests)
						newJob.Offloader.Destinations = i.j.Offloader.Destinations

						m.queue.Add(newJob)
						m.status = fmt.Sprintf("Retrying job %s...", i.j.ID)
						m.err = nil // Clear error state
					}
				}
			}
			if msg.String() == "enter" {
				if i, ok := m.jobList.SelectedItem().(jobItem); ok {
					if i.j.Status == job.StatusFailed && i.j.Err != nil {
						m.CurrentJob = i.j
						m.state = stateErrorDetails
						return m, nil
					}
				}
			}
			var cmd tea.Cmd
			m.jobList, cmd = m.jobList.Update(msg)
			return m, cmd
		}

		if m.state == stateErrorDetails {
			if msg.String() == "esc" || msg.String() == "q" || msg.String() == "enter" {
				m.state = stateJobManager
				return m, nil
			}
		}

		// Dry Run Interaction
		if m.state == stateDryRun {
			if msg.String() == "c" || msg.String() == "C" {
				// Continue to Actual Copy
				m.state = stateCopying
				j := job.NewJob(m.config)
				j.Offloader.Destinations = m.dstPaths
				m.queue.Add(j)
				return m, nil
			}
			if msg.String() == "q" || msg.String() == "Q" || msg.String() == "esc" {
				// Cancel back to menu
				m.Reset(m.config) // Reset state but keep config
				return m, loadRootsCmd
			}
			return m, nil
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
				// Update config with selections
				m.config.Source = m.srcPath
				m.config.Destination = m.dstPaths[0]

				if m.config.DryRun {
					m.state = stateDryRun
					return m, performDryRunCmd(m.config, m.dstPaths)
				}

				m.state = stateCopying
				// Create and add job
				j := job.NewJob(m.config)
				j.Offloader.Destinations = m.dstPaths
				m.queue.Add(j)

				// Wait for queue to pick it up?
				// Just continue.
				return m, nil
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case job.QueueState:
		m.queueState = msg
		m.CurrentJob = m.queue.Active
		status := fmt.Sprintf("Queue: %d Pending, %d Active, %d Done", msg.Pending, 1, msg.Completed)
		if m.CurrentJob == nil {
			status = fmt.Sprintf("Queue: %d Pending, 0 Active, %d Done", msg.Pending, msg.Completed)
			if msg.Total > 0 && msg.Pending == 0 {
				status = "All jobs completed."
				if m.state != stateJobManager {
					m.state = stateDone
				}
			}
		}
		m.status = status

		// Update Job List
		active, pending, completed, failed := m.queue.Snapshot()
		items := []list.Item{}
		if active != nil {
			items = append(items, jobItem{j: active})
		}
		for _, j := range pending {
			items = append(items, jobItem{j: j})
		}
		for _, j := range completed {
			items = append(items, jobItem{j: j})
		}
		for _, j := range failed {
			items = append(items, jobItem{j: j})
		}
		m.jobList.SetItems(items)

		return m, waitForQueueState(m.queue.UpdateChan)

	case job.Msg:
		jobMsg := msg
		switch jobMsg.Stage {
		case job.StatusCopying:
			m.state = stateCopying
			m.status = jobMsg.Status
			percent := float64(jobMsg.Progress.CopiedBytes) / float64(jobMsg.Progress.TotalBytes)
			if jobMsg.Progress.TotalBytes == 0 {
				percent = 0
			}
			cmd = m.progress.SetPercent(percent)
			return m, tea.Batch(cmd, waitForJobMsg(m.msgChan)) // Use m.msgChan

		case job.StatusVerifying:
			m.state = stateVerifying
			m.status = jobMsg.Status
			return m, waitForJobMsg(m.msgChan)

		case job.StatusCompleted:
			m.status = jobMsg.Status
			// CLI handling logic should be here if we want to quit
			if !m.config.Interactive {
				if m.config.JSONOutput {
					output.PrintJSON(*jobMsg.Job.Result)
				} else {
					output.PrintHuman(*jobMsg.Job.Result)
				}
				return m, tea.Quit
			}
			m.state = stateDone // Transition to Done state
			return m, waitForJobMsg(m.msgChan)

		case job.StatusCancelled:
			m.status = "Cancelled"
			m.state = stateDone // Transition to Done state (or just stick around)
			return m, waitForJobMsg(m.msgChan)

		case job.StatusFailed:
			m.err = jobMsg.Err
			m.status = fmt.Sprintf("Failed: %v", jobMsg.Err)
			if !m.config.Interactive {
				if jobMsg.Job.Result != nil {
					if m.config.JSONOutput {
						output.PrintJSON(*jobMsg.Job.Result)
					} else {
						output.PrintHuman(*jobMsg.Job.Result)
					}
				}
				return m, tea.Quit
			}
			m.state = stateDone
			return m, waitForJobMsg(m.msgChan)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.srcList.SetWidth(msg.Width)
		m.srcList.SetHeight(14)
		m.dstList.SetWidth(msg.Width)
		m.dstList.SetHeight(14)
		// Update progress bar width (minus padding)
		m.progress.Width = msg.Width - 10
		if m.progress.Width < 10 {
			m.progress.Width = 10
		}

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

	case errMsg:
		m.err = msg.err
		m.status = fmt.Sprintf("Error: %v", msg.err)
		return m, nil

	case dryRunResultMsg:
		m.dryRunResult = msg.result
		return m, nil
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

	return m, nil
}

func (m Model) View() string {
	if !m.config.Interactive {
		return ""
	}
	s := titleStyle.Render(logoASCII) + "\n"

	if m.state == stateJobManager {
		s += "JOB MANAGER:\n"
		s += m.jobList.View()
		s += "\n(Enter: View Error Details for Failed Jobs)"
		return s
	}

	if m.state == stateErrorDetails {
		s += titleStyle.Render("ERROR DETAILS") + "\n\n"
		if m.CurrentJob != nil {
			s += fmt.Sprintf("Job ID: %s\n", m.CurrentJob.ID)
			s += fmt.Sprintf("Source: %s\n", m.CurrentJob.Offloader.Source)
			s += fmt.Sprintf("Error:\n%v\n", m.CurrentJob.Err)
		}
		s += "\n\n(Press Esc/Enter to return)"
		return s
	}

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
	if m.state == stateDryRun {
		s += titleStyle.Render("DRY RUN SIMULATION") + "\n\n"
		s += fmt.Sprintf("Source: %s\n", m.config.Source)
		if m.dryRunResult != nil {
			s += fmt.Sprintf("Files to copy: %d\n", len(m.dryRunResult.Files))
			s += fmt.Sprintf("Total Size:    %s\n\n", offload.FormatBytes(uint64(m.dryRunResult.TotalSize)))
			s += "Destinations:\n"
			for _, d := range m.dryRunResult.Destinations {
				status := "✅ OK"
				if !d.CanFit {
					status = "❌ INSUFFICIENT SPACE"
				}
				s += fmt.Sprintf("  - %s\n", d.Path)
				s += fmt.Sprintf("    Free Space: %s\n", offload.FormatBytes(d.FreeSpace))
				s += fmt.Sprintf("    Status:     %s\n", status)
			}
		} else {
			s += "Simulating...\n" + m.spinner.View()
		}
		s += "\n\n" + instructionStyle.Render("(Press 'c' to continue Copy, 'q' to cancel)")
		return s
	}

	s += fmt.Sprintf("Source: %s\n", m.srcPath)
	s += "Destinations:\n"
	for _, d := range m.dstPaths {
		s += fmt.Sprintf("  - %s\n", d)
	}
	s += "\n"

	if m.err != nil {
		return s + errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.state == stateDone {
		if m.status == "✅ Verification successful!" {
			return s + completedStyle.Render(m.status) + "\n\n" + instructionStyle.Render("(Press 'q' to return to menu)")
		}
		// Failure case or other done state
		return s + m.status + "\n\n" + instructionStyle.Render("(Press 'q' to return to menu)")
	}

	s += progressStyle.Render(m.progress.View()) +
		percentageStyle.Render(fmt.Sprintf("%.0f%%", m.progress.Percent()*100)) + "\n"

	if m.state == stateVerifying {
		s += m.spinner.View() + " " + statsStyle.Render("Verifying Checksums...")
	} else if m.CurrentJob != nil {
		speedMB := m.CurrentJob.Speed / (1024 * 1024)
		s += statsStyle.Render(fmt.Sprintf("%.2f MB/s • %s", speedMB, m.status))
	}

	return s
}

type dryRunResultMsg struct {
	result *offload.DryRunResult
}

func performDryRunCmd(cfg *config.Config, dests []string) tea.Cmd {
	return func() tea.Msg {
		o := offload.NewOffloaderWithConfig(cfg, cfg.Source, dests...)
		res, err := o.DryRun()
		if err != nil {
			return errMsg{err}
		}
		return dryRunResultMsg{result: res}
	}
}
