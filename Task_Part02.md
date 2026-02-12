🗺️ LOOT - Roadmap Complète (Suite)

TASK 2.3: Implémentation du Resume
Priority: 🟠 HIGH
Estimated time: 4h
Dependencies: TASK 2.2
Context:

Permet de reprendre les jobs interrompus
Feature critique pour gros transferts (500GB+)
Compare les fichiers déjà copiés et skip si hash match

Files to modify:

cmd/loot/main.go
internal/offload/offloader.go
internal/ui/model.go

Files to create:

internal/commands/resume.go

Implementation:
go// internal/commands/resume.go
package commands

import (
	"fmt"
	"os"
	
	"loot/internal/job"
	"loot/internal/offload"
	"loot/internal/ui"
	
	tea "github.com/charmbracelet/bubbletea"
)

// Resume resumes a paused or interrupted job
func Resume(jobID string) error {
	mgr, err := job.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create job manager: %w", err)
	}
	
	state, err := mgr.Load(jobID)
	if err != nil {
		return fmt.Errorf("failed to load job %s: %w", jobID, err)
	}
	
	if state.Status == job.StatusCompleted {
		return fmt.Errorf("job %s is already completed", jobID)
	}
	
	if state.Status == job.StatusFailed {
		fmt.Printf("Warning: Job %s previously failed with error: %s\n", jobID, state.Error)
		fmt.Printf("Continue anyway? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("resume cancelled")
		}
	}
	
	fmt.Printf("Resuming job %s...\n", jobID)
	fmt.Printf("Source: %s\n", state.Source)
	fmt.Printf("Progress: %.1f%% (%d/%d files)\n", 
		state.Progress()*100,
		countCompletedFiles(state),
		len(state.Files))
	
	// Create offloader from saved state
	o := offload.NewOffloaderFromJobState(state)
	o.JobManager = mgr
	o.JobState = state
	
	// Run with TUI
	model := ui.InitialModelFromOffloader(o)
	p := tea.NewProgram(model)
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running resume: %w", err)
	}
	
	return nil
}

// ListJobs lists all saved jobs
func ListJobs() error {
	mgr, err := job.NewManager()
	if err != nil {
		return err
	}
	
	jobs, err := mgr.List()
	if err != nil {
		return err
	}
	
	if len(jobs) == 0 {
		fmt.Println("No jobs found")
		return nil
	}
	
	fmt.Printf("%-25s %-12s %-10s %s\n", "JOB ID", "STATUS", "PROGRESS", "SOURCE")
	fmt.Println("--------------------------------------------------------------------------------")
	
	for _, j := range jobs {
		progress := "N/A"
		if j.Status == job.StatusRunning || j.Status == job.StatusPaused {
			progress = fmt.Sprintf("%.1f%%", j.Progress()*100)
		}
		
		fmt.Printf("%-25s %-12s %-10s %s\n", 
			j.ID, 
			j.Status, 
			progress,
			truncate(j.Source, 40))
	}
	
	return nil
}

// CleanJobs removes completed jobs
func CleanJobs() error {
	mgr, err := job.NewManager()
	if err != nil {
		return err
	}
	
	jobs, err := mgr.List()
	if err != nil {
		return err
	}
	
	completed := 0
	for _, j := range jobs {
		if j.Status == job.StatusCompleted {
			if err := mgr.Delete(j.ID); err != nil {
				fmt.Printf("Warning: failed to delete %s: %v\n", j.ID, err)
			} else {
				completed++
			}
		}
	}
	
	fmt.Printf("Cleaned %d completed jobs\n", completed)
	return nil
}

func countCompletedFiles(state *job.JobState) int {
	count := 0
	for _, f := range state.Files {
		if f.Completed {
			count++
		}
	}
	return count
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
go// internal/offload/offloader.go
// Add method to create offloader from job state

func NewOffloaderFromJobState(state *job.JobState) *Offloader {
	cfg := config.DefaultConfig()
	cfg.Algorithm = state.Algorithm
	cfg.DualHash = state.DualHash
	
	o := &Offloader{
		Source:       state.Source,
		Destinations: state.Destinations,
		BufferSize:   cfg.BufferSize,
		Config:       cfg,
		JobState:     state,
	}
	
	return o
}

// Update Copy to skip already completed files
func (o *Offloader) Copy(progressChan chan<- ProgressInfo) error {
	// ... existing setup code ...
	
	if info.IsDir() {
		return filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.Name() == ".DS_Store" {
				return nil
			}

			relPath, err := filepath.Rel(o.Source, path)
			if err != nil {
				return err
			}

			if info.IsDir() {
				// Create directories
				for _, dstRoot := range o.Destinations {
					destPath := filepath.Join(dstRoot, relPath)
					if err := os.MkdirAll(destPath, info.Mode()); err != nil {
						return fmt.Errorf("failed to create dir %s: %w", destPath, err)
					}
				}
				return nil
			}

			// Check if file already completed (for resume)
			if o.JobState != nil {
				for _, fp := range o.JobState.Files {
					if fp.Path == relPath && fp.Completed {
						// Skip already completed file
						t.CopiedBytes += info.Size() // Update progress tracker
						t.update(0, filepath.Base(path)) // Update UI
						return nil
					}
				}
			}

			// Copy the file
			var dstPaths []string
			for _, dstRoot := range o.Destinations {
				dstPaths = append(dstPaths, filepath.Join(dstRoot, relPath))
			}

			return o.copyFileMulti(path, dstPaths, t)
		})
	} else {
		// Single file copy
		// Check if already completed
		if o.JobState != nil && len(o.JobState.Files) > 0 {
			if o.JobState.Files[0].Completed {
				fmt.Println("File already copied and verified")
				return nil
			}
		}
		
		t.TotalBytes = info.Size()
		return o.copyFileMulti(o.Source, o.Destinations, t)
	}
}
go// internal/ui/model.go
// Add function to create model from offloader

func InitialModelFromOffloader(o *offload.Offloader) Model {
	defaultWidth := 80
	defaultHeight := 14

	srcList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	dstList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return Model{
		state:     stateCopying,
		srcList:   srcList,
		dstList:   dstList,
		srcPath:   o.Source,
		dstPaths:  o.Destinations,
		progress:  p,
		offloader: o,
		status:    "Resuming...",
		width:     defaultWidth,
		height:    defaultHeight,
		config:    o.Config,
	}
}
go// cmd/loot/main.go
// Add subcommands support

package main

import (
	"fmt"
	"os"

	"loot/internal/commands"
	"loot/internal/config"
	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	// Check for subcommands
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "resume":
			if len(os.Args) < 3 {
				fmt.Println("Usage: loot resume <job-id>")
				os.Exit(1)
			}
			if err := commands.Resume(os.Args[2]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
			
		case "jobs":
			if len(os.Args) < 3 {
				fmt.Println("Usage: loot jobs <list|clean>")
				os.Exit(1)
			}
			
			switch os.Args[2] {
			case "list":
				if err := commands.ListJobs(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
			case "clean":
				if err := commands.CleanJobs(); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
			default:
				fmt.Println("Unknown jobs command. Use: list, clean")
				os.Exit(1)
			}
			return
		}
	}

	// Regular flow
	cfg, err := config.ParseFlags(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.Interactive {
		p := tea.NewProgram(ui.InitialModel("", ""))
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running loot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	p := tea.NewProgram(ui.InitialModelWithConfig(cfg))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
Tests to validate:
bash# Create large test directory
mkdir -p /tmp/bigtest
for i in {1..100}; do
  dd if=/dev/urandom of=/tmp/bigtest/file$i.dat bs=1M count=10
done

# Start copy and interrupt
go run cmd/loot/main.go /tmp/bigtest /tmp/backup
# Wait 5 seconds, press Ctrl+C

# List jobs
go run cmd/loot/main.go jobs list
# Should show paused job

# Resume
go run cmd/loot/main.go resume loot-YYYYMMDD-HHMMSS
# Should continue from where it stopped

# Verify no duplicate copying
# Check that already copied files are skipped

# Clean completed jobs
go run cmd/loot/main.go jobs clean
Success criteria:

 Job interrompu est sauvegardé
 Resume reprend exactement où c'était
 Fichiers déjà copiés sont skippés
 Progress bar continue correctement
 jobs list/clean fonctionnent


TASK 2.4: Dry-run mode
Priority: 🟡 MEDIUM
Estimated time: 1h
Dependencies: TASK 1.3
Context:

Permet de simuler l'opération sans copier
Utile pour vérifier l'espace disque
Debug des patterns d'exclusion

Files to modify:

internal/offload/offloader.go
internal/ui/model.go

Implementation:
go// internal/offload/offloader.go
// Add DryRun method

// DryRun simulates the copy operation without actually copying
func (o *Offloader) DryRun() ([]FileInfo, error) {
	var files []FileInfo
	
	info, err := os.Stat(o.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source: %w", err)
	}

	if info.IsDir() {
		err := filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.Name() == ".DS_Store" {
				return nil
			}

			if !info.IsDir() {
				relPath, _ := filepath.Rel(o.Source, path)
				files = append(files, FileInfo{
					Path: relPath,
					Size: info.Size(),
				})
			}
			
			return nil
		})
		
		if err != nil {
			return nil, err
		}
	} else {
		files = append(files, FileInfo{
			Path: filepath.Base(o.Source),
			Size: info.Size(),
		})
	}
	
	return files, nil
}

type FileInfo struct {
	Path string
	Size int64
}

// PrintDryRunSummary prints what would be copied
func PrintDryRunSummary(files []FileInfo, destinations []string) {
	var totalSize int64
	
	fmt.Println("\n=== DRY RUN ===")
	fmt.Printf("Files to copy: %d\n", len(files))
	
	for _, f := range files {
		totalSize += f.Size
		fmt.Printf("  - %s (%s)\n", f.Path, formatBytes(uint64(f.Size)))
	}
	
	fmt.Printf("\nTotal size: %s\n", formatBytes(uint64(totalSize)))
	fmt.Printf("Destinations: %d\n", len(destinations))
	
	for i, dest := range destinations {
		fmt.Printf("  %d. %s\n", i+1, dest)
		
		// Check available space
		var stat syscall.Statfs_t
		if err := syscall.Statfs(dest, &stat); err == nil {
			available := uint64(stat.Bavail) * uint64(stat.Bsize)
			fmt.Printf("     Available: %s", formatBytes(available))
			
			if available < uint64(totalSize) {
				fmt.Printf(" ⚠️  INSUFFICIENT SPACE (need %s)\n", formatBytes(uint64(totalSize)))
			} else {
				fmt.Printf(" ✅\n")
			}
		}
	}
	
	fmt.Println("\nNo files were copied (dry-run mode)")
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
go// internal/ui/model.go
// Add dry-run handling in Init

func (m Model) Init() tea.Cmd {
	if m.state == stateCopying {
		if m.config != nil && m.config.DryRun {
			return dryRunCmd(m.offloader)
		}
		m.startTime = time.Now()
		return startCopyWrapper(m.offloader)
	}
	return loadRootsCmd
}

func dryRunCmd(o *offload.Offloader) tea.Cmd {
	return func() tea.Msg {
		files, err := o.DryRun()
		if err != nil {
			return errMsg{err}
		}
		return dryRunResultMsg{files: files}
	}
}

type dryRunResultMsg struct {
	files []offload.FileInfo
}

// In Update method, handle dryRunResultMsg
case dryRunResultMsg:
	offload.PrintDryRunSummary(msg.files, m.dstPaths)
	return m, tea.Quit
Tests to validate:
bash# Dry run
go run cmd/loot/main.go --dry-run /tmp/test /tmp/backup
# Should show summary without copying

# Check nothing was copied
ls /tmp/backup
# Should be empty or not exist

# Dry run with insufficient space
# Create a small partition or use a nearly full disk
go run cmd/loot/main.go --dry-run /large/dataset /small/disk
# Should warn about insufficient space

# Dry run with JSON
go run cmd/loot/main.go --dry-run --json /tmp/test /tmp/backup
# Should output JSON summary
Success criteria:

 Rien n'est copié en mode dry-run
 Summary complet affiché
 Espace disque vérifié
 Warnings si espace insuffisant


🎯 PHASE 3: Testing & Quality (Semaine 3-4)
TASK 3.1: Setup testing infrastructure
Priority: 🔴 CRITICAL
Estimated time: 2h
Dependencies: Toutes les tâches précédentes
Context:

Nécessaire avant release
Tests automatisés pour éviter regressions
Coverage minimum 60%

Files to create:

internal/hash/hasher_test.go
internal/offload/offloader_test.go
internal/job/manager_test.go
Makefile

Implementation:
go// internal/hash/hasher_test.go
package hash

import (
	"bytes"
	"testing"
	
	"loot/internal/config"
)

func TestMultiHasher(t *testing.T) {
	tests := []struct {
		name  string
		data  string
		algos []config.HashAlgorithm
	}{
		{
			name:  "xxhash64 only",
			data:  "test data",
			algos: []config.HashAlgorithm{config.AlgoXXHash64},
		},
		{
			name:  "md5 only",
			data:  "test data",
			algos: []config.HashAlgorithm{config.AlgoMD5},
		},
		{
			name:  "sha256 only",
			data:  "test data",
			algos: []config.HashAlgorithm{config.AlgoSHA256},
		},
		{
			name:  "dual hash",
			data:  "test data",
			algos: []config.HashAlgorithm{config.AlgoXXHash64, config.AlgoMD5},
		},
		{
			name:  "all algorithms",
			data:  "test data",
			algos: []config.HashAlgorithm{config.AlgoXXHash64, config.AlgoMD5, config.AlgoSHA256},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewMultiHasher(tt.algos...)
			
			_, err := hasher.Write([]byte(tt.data))
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			
			result := hasher.Sum()
			
			// Verify requested algorithms produced hashes
			for _, algo := range tt.algos {
				hash := result.GetPrimary(algo)
				if hash == "" {
					t.Errorf("Expected hash for %s, got empty string", algo)
				}
			}
		})
	}
}

func TestHashConsistency(t *testing.T) {
	data := []byte("consistent test data")
	
	// Hash twice with same algorithm
	hasher1 := NewHasher(config.AlgoMD5)
	hasher1.Write(data)
	result1 := hasher1.Sum()
	
	hasher2 := NewHasher(config.AlgoMD5)
	hasher2.Write(data)
	result2 := hasher2.Sum()
	
	if result1.MD5 != result2.MD5 {
		t.Errorf("MD5 hashes not consistent: %s != %s", result1.MD5, result2.MD5)
	}
}

func TestKnownMD5(t *testing.T) {
	// Known MD5 for "test"
	data := []byte("test")
	expected := "098f6bcd4621d373cade4e832627b4f6"
	
	hasher := NewHasher(config.AlgoMD5)
	hasher.Write(data)
	result := hasher.Sum()
	
	if result.MD5 != expected {
		t.Errorf("MD5 mismatch: got %s, want %s", result.MD5, expected)
	}
}
go// internal/offload/offloader_test.go
package offload

import (
	"os"
	"path/filepath"
	"testing"
	
	"loot/internal/config"
)

func TestNewOffloader(t *testing.T) {
	o := NewOffloader("/src", "/dst1", "/dst2")
	
	if o.Source != "/src" {
		t.Errorf("Expected source /src, got %s", o.Source)
	}
	
	if len(o.Destinations) != 2 {
		t.Errorf("Expected 2 destinations, got %d", len(o.Destinations))
	}
}

func TestCopySingleFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	
	testData := []byte("test file content")
	if err := os.WriteFile(srcFile, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test
	cfg := config.DefaultConfig()
	o := NewOffloaderWithConfig(cfg, srcFile, dstFile)
	
	progressChan := make(chan ProgressInfo, 100)
	done := make(chan error, 1)
	
	go func() {
		done <- o.Copy(progressChan)
	}()
	
	// Consume progress updates
	go func() {
		for range progressChan {
			// Just consume
		}
	}()
	
	err := <-done
	close(progressChan)
	
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Errorf("Destination file was not created")
	}
	
	// Verify content
	dstData, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	
	if string(dstData) != string(testData) {
		t.Errorf("Content mismatch: got %s, want %s", dstData, testData)
	}
}

func TestVerifySingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	
	testData := []byte("verify test")
	os.WriteFile(srcFile, testData, 0644)
	os.WriteFile(dstFile, testData, 0644)
	
	cfg := config.DefaultConfig()
	o := NewOffloaderWithConfig(cfg, srcFile, dstFile)
	
	// Copy first
	progressChan := make(chan ProgressInfo, 100)
	go func() {
		for range progressChan {
		}
	}()
	
	if err := o.Copy(progressChan); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	close(progressChan)
	
	// Verify
	success, err := o.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	
	if !success {
		t.Errorf("Verification should succeed for identical files")
	}
}

func TestVerifyMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	
	os.WriteFile(srcFile, []byte("original"), 0644)
	os.WriteFile(dstFile, []byte("modified"), 0644)
	
	cfg := config.DefaultConfig()
	o := NewOffloaderWithConfig(cfg, srcFile, dstFile)
	
	success, err := o.Verify()
	
	if err == nil {
		t.Errorf("Expected error for mismatched files")
	}
	
	if success {
		t.Errorf("Verification should fail for different files")
	}
}
go// internal/job/manager_test.go
package job

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJobManager(t *testing.T) {
	// Use temp directory for testing
	tmpDir := t.TempDir()
	
	mgr := &Manager{JobsDir: tmpDir}
	
	// Create test job
	state := &JobState{
		ID:           "test-job-123",
		Source:       "/test/source",
		Destinations: []string{"/test/dest"},
		Status:       StatusRunning,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	// Test Save
	if err := mgr.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Verify file exists
	jobPath := filepath.Join(tmpDir, "test-job-123.json")
	if _, err := os.Stat(jobPath); os.IsNotExist(err) {
		t.Errorf("Job file was not created")
	}
	
	// Test Load
	loaded, err := mgr.Load("test-job-123")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if loaded.ID != state.ID {
		t.Errorf("Loaded job ID mismatch: got %s, want %s", loaded.ID, state.ID)
	}
	
	if loaded.Source != state.Source {
		t.Errorf("Loaded job source mismatch")
	}
	
	// Test List
	jobs, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	
	if len(jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs))
	}
	
	// Test Delete
	if err := mgr.Delete("test-job-123"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	
	if _, err := os.Stat(jobPath); !os.IsNotExist(err) {
		t.Errorf("Job file was not deleted")
	}
}

func TestJobProgress(t *testing.T) {
	state := &JobState{
		ID:          "progress-test",
		TotalBytes:  1000,
		CopiedBytes: 250,
	}
	
	progress := state.Progress()
	expected := 0.25
	
	if progress != expected {
		t.Errorf("Progress mismatch: got %.2f, want %.2f", progress, expected)
	}
	
	// Update progress
	state.UpdateProgress("test.txt", 250)
	
	if state.CopiedBytes != 500 {
		t.Errorf("CopiedBytes not updated: got %d, want 500", state.CopiedBytes)
	}
}
makefile# Makefile
.PHONY: all build test clean install

VERSION ?= dev
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

all: test build

build:
	go build $(LDFLAGS) -o loot cmd/loot/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

bench:
	go test -bench=. -benchmem ./...

clean:
	rm -f loot
	rm -f coverage.out coverage.html

install: build
	mv loot /usr/local/bin/

lint:
	golangci-lint run

# Development helpers
run:
	go run cmd/loot/main.go

run-version:
	go run $(LDFLAGS) cmd/loot/main.go --version

# Release builds
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/loot-darwin-amd64 cmd/loot/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/loot-darwin-arm64 cmd/loot/main.go

.DEFAULT_GOAL := all
Tests to validate:
bash# Run all tests
make test

# Check coverage
make coverage
open coverage.html

# Run benchmarks
make bench

# Lint code
make lint

# Build with version
make build VERSION=1.0.0
./loot --version
Success criteria:

 Tous les tests passent
 Coverage >= 60%
 Pas de race conditions
 Lint passe sans erreurs


TASK 3.2: Integration tests
Priority: 🟠 HIGH
Estimated time: 3h
Dependencies: TASK 3.1
Context:

Tests end-to-end complets
Scénarios réels d'utilisation
Validates toute la chaîne

Files to create:

test/integration_test.go
test/helpers.go

Implementation:
go// test/helpers.go
package test

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

// CreateTestFiles creates a test directory structure
func CreateTestFiles(t *testing.T, dir string, fileCount int, fileSize int64) {
	t.Helper()
	
	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(dir, fmt.Sprintf("file%03d.dat", i))
		CreateRandomFile(t, filename, fileSize)
	}
}

// CreateRandomFile creates a file with random data
func CreateRandomFile(t *testing.T, path string, size int64) {
	t.Helper()
	
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer f.Close()
	
	// Write random data
	written := int64(0)
	buf := make([]byte, 1024*1024) // 1MB chunks
	
	for written < size {
		toWrite := size - written
		if toWrite > int64(len(buf)) {
			toWrite = int64(len(buf))
		}
		
		rand.Read(buf[:toWrite])
		n, err := f.Write(buf[:toWrite])
		if err != nil {
			t.Fatalf("Failed to write: %v", err)
		}
		written += int64(n)
	}
}

// CompareDirectories verifies two directories are identical
func CompareDirectories(t *testing.T, dir1, dir2 string) {
	t.Helper()
	
	// Get all files from both dirs
	files1 := getAllFiles(t, dir1)
	files2 := getAllFiles(t, dir2)
	
	if len(files1) != len(files2) {
		t.Errorf("File count mismatch: %d vs %d", len(files1), len(files2))
	}
	
	for relPath, info1 := range files1 {
		info2, exists := files2[relPath]
		if !exists {
			t.Errorf("File missing in dir2: %s", relPath)
			continue
		}
		
		if info1.Size != info2.Size {
			t.Errorf("Size mismatch for %s: %d vs %d", relPath, info1.Size, info2.Size)
		}
	}
}

func getAllFiles(t *testing.T, dir string) map[string]os.FileInfo {
	t.Helper()
	
	files := make(map[string]os.FileInfo)
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			files[relPath] = info
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	
	return files
}
go// test/integration_test.go
package test

import (
	"os"
	"path/filepath"
	"testing"
	
	"loot/internal/config"
	"loot/internal/offload"
)

func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Setup
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dstDir := filepath.Join(tmpDir, "destination")
	
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(dstDir, 0755)
	
	// Create test data: 10 files of 1MB each
	CreateTestFiles(t, srcDir, 10, 1024*1024)
	
	// Configure
	cfg := config.DefaultConfig()
	cfg.Algorithm = config.AlgoMD5
	
	o := offload.NewOffloaderWithConfig(cfg, srcDir, dstDir)
	
	// Copy
	progressChan := make(chan offload.ProgressInfo, 100)
	done := make(chan error, 1)
	
	go func() {
		done <- o.Copy(progressChan)
	}()
	
	// Monitor progress
	var lastProgress float64
	go func() {
		for p := range progressChan {
			progress := float64(p.CopiedBytes) / float64(p.TotalBytes)
			if progress < lastProgress {
				t.Errorf("Progress went backwards: %.2f -> %.2f", lastProgress, progress)
			}
			lastProgress = progress
		}
	}()
	
	err := <-done
	close(progressChan)
	
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	
	// Verify
	success, err := o.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	
	if !success {
		t.Errorf("Verification failed")
	}
	
	// Compare directories
	CompareDirectories(t, srcDir, dstDir)
}

func TestMultiDestination(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	dst1Dir := filepath.Join(tmpDir, "dest1")
	dst2Dir := filepath.Join(tmpDir, "dest2")
	
	os.MkdirAll(srcDir, 0755)
	
	CreateTestFiles(t, srcDir, 5, 512*1024)
	
	cfg := config.DefaultConfig()
	o := offload.NewOffloaderWithConfig(cfg, srcDir, dst1Dir, dst2Dir)
	
	progressChan := make(chan offload.ProgressInfo, 100)
	go func() {
		for range progressChan {
		}
	}()
	
	if err := o.Copy(progressChan); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	close(progressChan)
	
	// Verify both destinations
	success, err := o.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	
	if !success {
		t.Errorf("Verification failed")
	}
	
	// Compare both destinations with source
	CompareDirectories(t, srcDir, dst1Dir)
	CompareDirectories(t, srcDir, dst2Dir)
}

func TestResumeWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// TODO: This needs job manager integration
	// Simulate interruption and resume
	t.Skip("Resume workflow test not yet implemented")
}

func TestDifferentAlgorithms(t *testing.T) {
	algorithms := []config.HashAlgorithm{
		config.AlgoXXHash64,
		config.AlgoMD5,
		config.AlgoSHA256,
	}
	
	for _, algo := range algorithms {
		t.Run(string(algo), func(t *testing.T) {
			tmpDir := t.TempDir()
			srcFile := filepath.Join(tmpDir, "source.txt")
			dstFile := filepath.Join(tmpDir, "dest.txt")
			
			CreateRandomFile(t, srcFile, 10*1024) // 10KB
			
			cfg := config.DefaultConfig()
			cfg.Algorithm = algo
			
			o := offload.NewOffloaderWithConfig(cfg, srcFile, dstFile)
			
			progressChan := make(chan offload.ProgressInfo, 100)
			go func() {
				for range progressChan {
				}
			}()
			
			if err := o.Copy(progressChan); err != nil {
				t.Fatalf("Copy failed: %v", err)
			}
			close(progressChan)
			
			success, err := o.Verify()
			if err != nil {
				t.Fatalf("Verify with %s failed: %v", algo, err)
			}
			
			if !success {
				t.Errorf("Verification with %s failed", algo)
			}
		})
	}
}
Tests to validate:
bash# Run integration tests
go test -v ./test/...

# Run with coverage
go test -v -coverprofile=integration.out ./test/...

# Skip in short mode
go test -short ./test/...

# Run specific test
go test -v -run TestFullWorkflow ./test/...
Success criteria:

 Full workflow test passe
 Multi-destination test passe
 Tous les algorithmes fonctionnent
 Pas de memory leaks


TASK 3.3: CI/CD avec GitHub Actions
Priority: 🔴 CRITICAL
Estimated time: 2h
Dependencies: TASK 3.1, 3.2
Context:

Automation des tests sur chaque commit
Build automatique des releases
Validation avant merge

Files to create:

.github/workflows/test.yml
.github/workflows/release.yml
.github/workflows/lint.yml

Implementation:
yaml# .github/workflows/test.yml
name: Test

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    name: Test on ${{ matrix.os }}
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [macos-latest, ubuntu-latest]
        go-version: ['1.25']
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v4
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Download dependencies
      run: go mod download

    - name: Run tests
      run: go test -v -race -coverprofile=coverage.out ./...

    - name: Upload coverage
      uses: codecov/codecov-action@v4
      with:
        files: ./coverage.out
        flags: unittests

    - name: Run integration tests
      run: go test -v ./test/...

  build:
    name: Build
    runs-on: macos-latest
    needs: test
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.25'

    - name: Build for macOS AMD64
      run: GOOS=darwin GOARCH=amd64 go build -o loot-darwin-amd64 cmd/loot/main.go

    - name: Build for macOS ARM64
      run: GOOS=darwin GOARCH=arm64 go build -o loot-darwin-arm64 cmd/loot/main.go

    - name: Upload artifacts
      uses: actions/upload-artifact@v4
      with:
        name: binaries
        path: loot-*
yaml# .github/workflows/lint.yml
name: Lint

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  golangci:
    name: lint
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
yaml# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    name: Create Release
    runs-on: macos-latest
    
    steps:
    - name: Checkout code
      uses: actions/checkout@v4
      with:
        fetch-depth: 0

    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.25'

    - name: Get version
      id: get_version
      run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

    - name: Build binaries
      run: |
        VERSION=${{ steps.get_version.outputs.VERSION }}
        
        # macOS AMD64
        GOOS=darwin GOARCH=amd64 go build \
          -ldflags "-X main.version=$VERSION" \
          -o loot-darwin-amd64 \
          cmd/loot/main.go
        
        # macOS ARM64
        GOOS=darwin GOARCH=arm64 go build \
          -ldflags "-X main.version=$VERSION" \
          -o loot-darwin-arm64 \
          cmd/loot/main.go
        
        # Create tarballs
        tar czf loot-$VERSION-darwin-amd64.tar.gz loot-darwin-amd64
        tar czf loot-$VERSION-darwin-arm64.tar.gz loot-darwin-arm64
        
        # Generate checksums
        shasum -a 256 loot-$VERSION-*.tar.gz > checksums.txt

    - name: Create Release
      uses: softprops/action-gh-release@v1
      with:
        files: |
          loot-${{ steps.get_version.outputs.VERSION }}-darwin-amd64.tar.gz
          loot-${{ steps.get_version.outputs.VERSION }}-darwin-arm64.tar.gz
          checksums.txt
        generate_release_notes: true
        draft: false
        prerelease: false
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
Also create:
yaml# .golangci.yml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign

linters-settings:
  gofmt:
    simplify: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0

run:
  timeout: 5m
Tests to validate:
bash# Locally test lint
golangci-lint run

# Create a test tag
git tag v0.0.1-test
git push origin v0.0.1-test

# Check GitHub Actions
# Should trigger release workflow

# Download artifact
# Verify binary works
Success criteria:

 Tests run sur chaque push
 Lint s'exécute automatiquement
 Release builds fonctionnent
 Artifacts uploadés correctement


🎯 PHASE 4: Documentation & Distribution (Semaine 4-5)
TASK 4.1: Documentation complète
Priority: 🟠 HIGH
Estimated time: 4h
Dependencies: Toutes les features complétées
Context:

Documentation critique pour adoption
Guide utilisateur complet
Examples concrets

Files to create:

docs/INSTALLATION.md
docs/USAGE.md
docs/WORKFLOWS.md
docs/AUTOMATION.md
docs/TROUBLESHOOTING.md
docs/API.md

Implementation:
markdown<!-- docs/INSTALLATION.md -->
# Installation Guide

## Homebrew (Recommended)
```bash
brew tap Mald0r0r000/loot
brew install loot
```

## Manual Installation

### Download Binary

1. Go to [Releases](https://github.com/Mald0r0r000/LOOT/releases)
2. Download the appropriate version:
   - `loot-vX.X.X-darwin-amd64.tar.gz` (Intel Mac)
   - `loot-vX.X.X-darwin-arm64.tar.gz` (Apple Silicon)

3. Extract and install:
```bash
tar xzf loot-v1.0.0-darwin-arm64.tar.gz
sudo mv loot-darwin-arm64 /usr/local/bin/loot
chmod +x /usr/local/bin/loot
```

### Build from Source

**Requirements:**
- Go 1.25 or higher
- Git

**Steps:**
```bash
git clone https://github.com/Mald0r0r000/LOOT.git
cd LOOT
make build
sudo make install
```

**Verify Installation:**
```bash
loot --version
# Should output: loot version X.X.X
```

## Shell Completion (Optional)

### Bash
```bash
loot completion bash > /usr/local/etc/bash_completion.d/loot
```

### Zsh
```bash
loot completion zsh > /usr/local/share/zsh/site-functions/_loot
```

## Uninstall

### Homebrew
```bash
brew uninstall loot
brew untap Mald0r0r000/loot
```

### Manual
```bash
sudo rm /usr/local/bin/loot
rm -rf ~/.loot
```
markdown<!-- docs/USAGE.md -->
# Usage Guide

## Quick Start

### Interactive Mode

Launch LOOT without arguments for the TUI:
```bash
loot
```

Navigate with:
- `↑/↓` - Move up/down
- `→/Enter` - Enter directory
- `←` - Go back
- `Space` - Select source/destination
- `q/Esc` - Quit

### CLI Mode

Basic syntax:
```bash
loot  
```

Example:
```bash
loot /Volumes/SD_CARD /Volumes/BACKUP
```

## Options

### Hash Algorithms
```bash
# Use MD5 (industry standard)
loot --algorithm md5 /Volumes/CARD /Volumes/BACKUP

# Use SHA256 (forensic-grade)
loot --algorithm sha256 /Volumes/CARD /Volumes/BACKUP

# Use xxHash64 (fastest, default)
loot --algorithm xxhash64 /Volumes/CARD /Volumes/BACKUP

# Dual-hash (xxHash64 + MD5)
loot --dual-hash /Volumes/CARD /Volumes/BACKUP
```

### Output Formats
```bash
# JSON output (for scripts/agents)
loot --json /Volumes/CARD /Volumes/BACKUP

# Quiet mode (no TUI)
loot --quiet /Volumes/CARD /Volumes/BACKUP

# Verbose logging
loot --verbose /Volumes/CARD /Volumes/BACKUP
```

### Dry Run
```bash
# Simulate without copying
loot --dry-run /Volumes/CARD /Volumes/BACKUP
```

Shows:
- Files to be copied
- Total size
- Available space on destination
- Warnings if insufficient space

### Job Management
```bash
# List all jobs
loot jobs list

# Resume interrupted job
loot resume loot-20260212-143022

# Clean completed jobs
loot jobs clean
```

## Multi-Destination

Copy to multiple destinations simultaneously:
```bash
loot /Volumes/CARD /Volumes/BACKUP1 /Volumes/BACKUP2
```

All destinations are verified independently.

## Examples

### Basic DIT Workflow
```bash
# 1. Offload with MD5 verification
loot --algorithm md5 /Volumes/CFAST_A /Volumes/RAID/Dailies/2026-02-12

# 2. Review generated reports
open /Volumes/RAID/Dailies/2026-02-12.pdf
open /Volumes/RAID/Dailies/2026-02-12.mhl
```

### Production Safety Workflow
```bash
# 1. Dry run to check space
loot --dry-run /Volumes/CARD /Volumes/BACKUP

# 2. Dual-destination with dual-hash
loot --dual-hash \
  /Volumes/CARD \
  /Volumes/BACKUP_A \
  /Volumes/BACKUP_B

# 3. Keep JSON log for records
loot --json --algorithm sha256 \
  /Volumes/CARD \
  /Volumes/ARCHIVE \
  > offload-$(date +%Y%m%d).json
```

### Resume Interrupted Transfer
```bash
# If transfer was interrupted (Ctrl+C, power loss)
# List jobs to find ID
loot jobs list

# Resume
loot resume loot-20260212-143022
```

## Performance Tips

### Buffer Size

Adjust for your hardware:
```bash
# Default: 4MB
loot /Volumes/CARD /Volumes/BACKUP

# Larger buffer for SSDs (8MB)
loot --buffer-size 8388608 /Volumes/CARD /Volumes/SSD

# Smaller buffer for slower media (1MB)
loot --buffer-size 1048576 /Volumes/CARD /Volumes/HDD
```

### Algorithm Performance

On Apple Silicon Mac:

- **xxHash64**: ~3.5 GB/s (fastest)
- **MD5**: ~350 MB/s (10x slower)
- **SHA256**: ~180 MB/s (20x slower)

Choose based on requirements:
- Speed → xxHash64
- Industry compatibility → MD5
- Forensic/archival → SHA256
markdown<!-- docs/WORKFLOWS.md -->
# DIT Workflows

## Daily Rushes Workflow

### Morning Card Offload
```bash
#!/bin/bash
# daily-offload.sh

DATE=$(date +%Y-%m-%d)
CARD="/Volumes/CFAST_A"
DEST="/Volumes/RAID/Rushes/$DATE"

# Check card is mounted
if [ ! -d "$CARD" ]; then
  echo "Error: Card not mounted"
  exit 1
fi

# Offload with MD5
loot --algorithm md5 \
     --json \
     "$CARD" \
     "$DEST" \
     > "offload-$DATE.json"

# Parse result
STATUS=$(jq -r '.status' "offload-$DATE.json")

if [ "$STATUS" == "success" ]; then
  echo "✅ Offload complete: $DEST"
  
  # Safe to format card
  echo "Card ready to format"
else
  echo "❌ Offload failed!"
  exit 1
fi
```

## Multi-Cam Production

### Sync Multiple Cards
```bash
#!/bin/bash
# multi-cam-offload.sh

DATE=$(date +%Y-%m-%d)
CARDS=(
  "/Volumes/A_CAM"
  "/Volumes/B_CAM"
  "/Volumes/C_CAM"
)

for CARD in "${CARDS[@]}"; do
  CAM_NAME=$(basename "$CARD")
  DEST="/Volumes/RAID/Rushes/$DATE/$CAM_NAME"
  
  echo "Offloading $CAM_NAME..."
  
  loot --algorithm md5 \
       --dual-hash \
       "$CARD" \
       "$DEST" \
       "/Volumes/BACKUP/$DATE/$CAM_NAME"
       
  if [ $? -eq 0 ]; then
    echo "✅ $CAM_NAME complete"
  else
    echo "❌ $CAM_NAME failed"
  fi
done
```

## Archive Workflow

### Long-Term Archive
```bash
# Use SHA256 for forensic-grade verification
loot --algorithm sha256 \
     --json \
     /Volumes/PROJECT/Finals \
     /Volumes/LTO_STAGING \
     > archive-manifest.json

# MHL file will be created for tape systems
# archive.mhl contains SHA256 hashes
```

## Cloud Backup Integration

### Upload After Verification
```bash
#!/bin/bash
# offload-and-upload.sh

SOURCE="/Volumes/SD_CARD"
LOCAL="/Volumes/BACKUP/$(date +%Y-%m-%d)"
BUCKET="s3://my-production-bucket/rushes"

# 1. Local offload
RESULT=$(loot --json --algorithm md5 "$SOURCE" "$LOCAL")
STATUS=$(echo "$RESULT" | jq -r '.status')

if [ "$STATUS" != "success" ]; then
  echo "Offload failed!"
  exit 1
fi

# 2. Upload to cloud (after local verification)
aws s3 sync "$LOCAL" "$BUCKET/$(date +%Y-%m-%d)" \
  --storage-class DEEP_ARCHIVE

echo "✅ Local backup verified and uploaded"
```
markdown<!-- docs/AUTOMATION.md -->
# Automation with AI Agents

## Claude Agent Examples

### Simple Offload

**User:** "Offload my SD card to my backup drive"

**Claude generates:**
```bash
#!/bin/bash
# Auto-detect SD card
CARD=$(ls /Volumes | grep -i "SD" | head -1)

if [ -z "$CARD" ]; then
  echo "No SD card found"
  exit 1
fi

# Create dated backup folder
BACKUP="/Volumes/BACKUP/$(date +%Y-%m-%d)"

# Execute offload
loot --json "/Volumes/$CARD" "$BACKUP"
```

### Advanced Workflow

**User:** "Offload all my camera cards, verify with MD5, email me when done"

**Claude generates:**
```bash
#!/bin/bash

DATE=$(date +%Y-%m-%d)
RESULTS=()

# Find all mounted cards
CARDS=$(ls /Volumes | grep -E "(CFAST|SD|XQD)")

for CARD in $CARDS; do
  echo "Processing $CARD..."
  
  DEST="/Volumes/RAID/$DATE/$CARD"
  
  RESULT=$(loot --json --algorithm md5 "/Volumes/$CARD" "$DEST")
  STATUS=$(echo "$RESULT" | jq -r '.status')
  FILES=$(echo "$RESULT" | jq -r '.files_count')
  SIZE=$(echo "$RESULT" | jq -r '.total_bytes')
  
  RESULTS+=("$CARD: $STATUS - $FILES files ($SIZE bytes)")
done

# Email summary
{
  echo "Offload Summary - $DATE"
  echo "===================="
  printf '%s\n' "${RESULTS[@]}"
} | mail -s "Offload Complete" you@example.com
```

## JSON API Integration

### Parse Results Programmatically
```python
#!/usr/bin/env python3
import subprocess
import json

result = subprocess.run(
    ['loot', '--json', '/Volumes/CARD', '/Volumes/BACKUP'],
    capture_output=True,
    text=True
)

data = json.loads(result.stdout)

if data['status'] == 'success':
    print(f"✅ Success!")
    print(f"Files: {data['files_count']}")
    print(f"Speed: {data['speed_mbps']:.1f} Mbps")
    print(f"Hash: {data['hash']}")
else:
    print(f"❌ Failed: {data['errors']}")
```

### Node.js Integration
```javascript
const { exec } = require('child_process');

function offload(source, dest) {
  return new Promise((resolve, reject) => {
    exec(`loot --json ${source} ${dest}`, (error, stdout, stderr) => {
      if (error) {
        reject(error);
        return;
      }
      
      const result = JSON.parse(stdout);
      resolve(result);
    });
  });
}

// Usage
offload('/Volumes/CARD', '/Volumes/BACKUP')
  .then(result => {
    console.log(`Status: ${result.status}`);
    console.log(`Files: ${result.files_count}`);
  })
  .catch(err => console.error(err));
```

## Watch Folder Automation

### Auto-Offload on Mount
```bash
# ~/.loot/auto-offload.sh

#!/bin/bash
# Watches for card mount and auto-offloads

fswatch -0 /Volumes | while read -d "" event; do
  # Check if it's a camera card
  if echo "$event" | grep -q "SD\|CFAST\|XQD"; then
    CARD=$(basename "$event")
    DEST="/Volumes/AUTO_BACKUP/$(date +%Y-%m-%d-%H%M%S)-$CARD"
    
    echo "Detected card: $CARD"
    echo "Auto-offloading to: $DEST"
    
    loot --algorithm md5 "$event" "$DEST"
    
    if [ $? -eq 0 ]; then
      osascript -e 'display notification "Card offload complete" with title "LOOT"'
    fi
  fi
done
```

### Run on Login (macOS)

Create `~/Library/LaunchAgents/com.loot.auto-offload.plist`:
```xml

<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">


    Label
    com.loot.auto-offload
    ProgramArguments
    
        /Users/YOUR_USERNAME/.loot/auto-offload.sh
    
    RunAtLoad
    


```

Load:
```bash
launchctl load ~/Library/LaunchAgents/com.loot.auto-offload.plist
```