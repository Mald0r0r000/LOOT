🗺️ LOOT - Roadmap Détaillée pour IDE Intelligent
📋 Structure de la Roadmap
Cette roadmap est optimisée pour les agents IA (Windsurf, Cursor, etc.) avec des tâches atomiques, contexte complet et tests de validation.

🎯 PHASE 1: Fondations CLI & Architecture (Semaine 1-2)
TASK 1.1: Ajout du système de versioning
Priority: 🔴 CRITICAL
Estimated time: 30 min
Dependencies: Aucune
Context:

Actuellement, pas de version tracking dans le code
Nécessaire pour Homebrew et releases GitHub
Version sera injectée via ldflags au build

Files to modify:

cmd/loot/main.go

Implementation:
go// cmd/loot/main.go
package main

import (
	"fmt"
	"os"

	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// Version will be set via ldflags during build
// Example: go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// Handle version flag
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("loot version %s\n", version)
		os.Exit(0)
	}

	var src, dst string

	if len(os.Args) >= 3 {
		src = os.Args[1]
		dst = os.Args[2]

		// Verify source exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("Error: Source file '%s' does not exist.\n", src)
			os.Exit(1)
		}
	} else if len(os.Args) == 2 {
		fmt.Println("Usage: loot <source> <destination>")
		fmt.Println("       loot (for interactive mode)")
		fmt.Println("       loot --version")
		os.Exit(1)
	}

	p := tea.NewProgram(ui.InitialModel(src, dst))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running loot: %v\n", err)
		os.Exit(1)
	}
}
Tests to validate:
bash# Build with version
go build -ldflags "-X main.version=1.0.0-test" -o loot cmd/loot/main.go

# Test
./loot --version
# Expected output: loot version 1.0.0-test

./loot -v
# Expected output: loot version 1.0.0-test

# Without ldflags
go build -o loot cmd/loot/main.go
./loot --version
# Expected output: loot version dev
Success criteria:

 loot --version affiche la version
 loot -v affiche la version
 Version "dev" par défaut
 ldflags override fonctionne


TASK 1.2: Créer le package de configuration
Priority: 🔴 CRITICAL
Estimated time: 1h
Dependencies: TASK 1.1
Context:

Besoin d'une structure pour gérer les flags CLI
Prépare pour --json, --quiet, --algorithm, etc.
Centralise la configuration de l'app

Files to create:

internal/config/config.go

Implementation:
go// internal/config/config.go
package config

import (
	"flag"
	"fmt"
	"os"
)

type HashAlgorithm string

const (
	AlgoXXHash64 HashAlgorithm = "xxhash64"
	AlgoMD5      HashAlgorithm = "md5"
	AlgoSHA256   HashAlgorithm = "sha256"
)

// Config holds all configuration for LOOT
type Config struct {
	// Operation mode
	Interactive bool
	Source      string
	Destination string

	// Output options
	JSONOutput bool
	Quiet      bool
	Verbose    bool

	// Hash options
	Algorithm HashAlgorithm
	DualHash  bool // Calculate both xxhash64 and MD5

	// Verification
	NoVerify bool

	// Performance
	BufferSize int // in bytes

	// Dry run
	DryRun bool

	// Version info
	Version string
}

// DefaultConfig returns config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Interactive: true,
		Algorithm:   AlgoXXHash64,
		DualHash:    false,
		BufferSize:  4 * 1024 * 1024, // 4MB
		NoVerify:    false,
		DryRun:      false,
		JSONOutput:  false,
		Quiet:       false,
		Verbose:     false,
	}
}

// ParseFlags parses command line arguments and returns Config
func ParseFlags(version string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.Version = version

	// Define flags
	var algorithmStr string
	flag.StringVar(&algorithmStr, "algorithm", "xxhash64", "Hash algorithm: xxhash64, md5, sha256")
	flag.BoolVar(&cfg.DualHash, "dual-hash", false, "Calculate both xxhash64 and MD5")
	flag.BoolVar(&cfg.JSONOutput, "json", false, "Output results in JSON format")
	flag.BoolVar(&cfg.Quiet, "quiet", false, "Suppress all output except errors")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cfg.NoVerify, "no-verify", false, "Skip verification after copy")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Simulate operation without copying")
	flag.IntVar(&cfg.BufferSize, "buffer-size", 4*1024*1024, "Buffer size in bytes")

	versionFlag := flag.Bool("version", false, "Print version")
	v := flag.Bool("v", false, "Print version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "LOOT - Professional Media Offload Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  loot                          Interactive mode\n")
		fmt.Fprintf(os.Stderr, "  loot <source> <dest>          CLI mode\n")
		fmt.Fprintf(os.Stderr, "  loot --version                Print version\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Handle version
	if *versionFlag || *v {
		fmt.Printf("loot version %s\n", version)
		os.Exit(0)
	}

	// Parse algorithm
	switch algorithmStr {
	case "xxhash64":
		cfg.Algorithm = AlgoXXHash64
	case "md5":
		cfg.Algorithm = AlgoMD5
	case "sha256":
		cfg.Algorithm = AlgoSHA256
	default:
		return nil, fmt.Errorf("invalid algorithm: %s (must be xxhash64, md5, or sha256)", algorithmStr)
	}

	// Get positional arguments
	args := flag.Args()

	if len(args) >= 2 {
		cfg.Interactive = false
		cfg.Source = args[0]
		cfg.Destination = args[1]

		// Verify source exists
		if _, err := os.Stat(cfg.Source); os.IsNotExist(err) {
			return nil, fmt.Errorf("source '%s' does not exist", cfg.Source)
		}
	} else if len(args) == 1 {
		return nil, fmt.Errorf("both source and destination required for CLI mode")
	}
	// else: Interactive mode (len(args) == 0)

	return cfg, nil
}
Tests to validate:
bash# Test help
go run cmd/loot/main.go --help
# Should show usage

# Test invalid algorithm
go run cmd/loot/main.go --algorithm invalid /src /dst
# Should error: "invalid algorithm"

# Test valid flags
go run cmd/loot/main.go --algorithm md5 --json --dry-run /tmp/test /tmp/backup
# Should parse without error
Success criteria:

 Flags parsés correctement
 Valeurs par défaut sensées
 Validation des algorithmes
 Help message clair


TASK 1.3: Intégrer Config dans main.go
Priority: 🔴 CRITICAL
Estimated time: 30 min
Dependencies: TASK 1.2
Context:

Remplacer le parsing manuel par le nouveau système Config
Passer Config à travers l'app

Files to modify:

cmd/loot/main.go
internal/ui/model.go

Implementation:
go// cmd/loot/main.go
package main

import (
	"fmt"
	"os"

	"loot/internal/config"
	"loot/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	cfg, err := config.ParseFlags(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Interactive mode
	if cfg.Interactive {
		p := tea.NewProgram(ui.InitialModel("", ""))
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
go// internal/ui/model.go
// Add this function after InitialModel

func InitialModelWithConfig(cfg *config.Config) Model {
	defaultWidth := 80
	defaultHeight := 14

	srcList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	srcList.Title = "Select Source"
	srcList.SetShowHelp(false)

	dstList := list.New([]list.Item{}, list.NewDefaultDelegate(), defaultWidth, defaultHeight)
	dstList.Title = "Select Destination"
	dstList.SetShowHelp(false)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return Model{
		state:     stateCopying,
		srcList:   srcList,
		dstList:   dstList,
		srcPath:   cfg.Source,
		dstPaths:  []string{cfg.Destination},
		progress:  p,
		offloader: offload.NewOffloader(cfg.Source, cfg.Destination),
		status:    "Initializing...",
		width:     defaultWidth,
		height:    defaultHeight,
		config:    cfg, // Add config field to Model struct
	}
}
Also add to Model struct in model.go:
gotype Model struct {
	// ... existing fields ...
	config *config.Config // Add this line
}
Tests to validate:
bash# Interactive mode (no args)
go run cmd/loot/main.go
# Should show TUI

# CLI mode
go run cmd/loot/main.go /tmp/test /tmp/backup
# Should start copy

# With flags
go run cmd/loot/main.go --algorithm md5 /tmp/test /tmp/backup
# Should work with MD5
Success criteria:

 Mode interactif fonctionne
 Mode CLI fonctionne
 Config passée correctement
 Flags respectés


TASK 1.4: Supporter multi-algorithmes de hash
Priority: 🟠 HIGH
Estimated time: 2h
Dependencies: TASK 1.3
Context:

Actuellement seul xxHash64 est supporté
Besoin d'ajouter MD5 et SHA256 pour compatibilité industrie
Architecture doit supporter dual-hash (xxHash + MD5 simultané)

Files to create:

internal/hash/hasher.go

Files to modify:

internal/offload/offloader.go
go.mod (ajouter crypto/md5, crypto/sha256 - déjà dans stdlib)

Implementation:
go// internal/hash/hasher.go
package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/cespare/xxhash/v2"
	"loot/internal/config"
)

// MultiHasher can calculate multiple hashes simultaneously
type MultiHasher struct {
	xxh     hash.Hash64
	md5Hash hash.Hash
	sha     hash.Hash
	
	enabledAlgos map[config.HashAlgorithm]bool
}

// NewMultiHasher creates a hasher for the given algorithms
func NewMultiHasher(algorithms ...config.HashAlgorithm) *MultiHasher {
	mh := &MultiHasher{
		enabledAlgos: make(map[config.HashAlgorithm]bool),
	}

	for _, algo := range algorithms {
		mh.enabledAlgos[algo] = true

		switch algo {
		case config.AlgoXXHash64:
			mh.xxh = xxhash.New()
		case config.AlgoMD5:
			mh.md5Hash = md5.New()
		case config.AlgoSHA256:
			mh.sha = sha256.New()
		}
	}

	return mh
}

// NewHasher creates a hasher for a single algorithm
func NewHasher(algo config.HashAlgorithm) *MultiHasher {
	return NewMultiHasher(algo)
}

// Write implements io.Writer
func (mh *MultiHasher) Write(p []byte) (n int, err error) {
	if mh.xxh != nil {
		mh.xxh.Write(p)
	}
	if mh.md5Hash != nil {
		mh.md5Hash.Write(p)
	}
	if mh.sha != nil {
		mh.sha.Write(p)
	}
	return len(p), nil
}

// HashResult contains all calculated hashes
type HashResult struct {
	XXHash64 string
	MD5      string
	SHA256   string
}

// Sum returns all calculated hashes
func (mh *MultiHasher) Sum() HashResult {
	result := HashResult{}

	if mh.xxh != nil {
		result.XXHash64 = fmt.Sprintf("%x", mh.xxh.Sum64())
	}
	if mh.md5Hash != nil {
		result.MD5 = hex.EncodeToString(mh.md5Hash.Sum(nil))
	}
	if mh.sha != nil {
		result.SHA256 = hex.EncodeToString(mh.sha.Sum(nil))
	}

	return result
}

// GetPrimary returns the hash for the primary algorithm
func (result HashResult) GetPrimary(algo config.HashAlgorithm) string {
	switch algo {
	case config.AlgoXXHash64:
		return result.XXHash64
	case config.AlgoMD5:
		return result.MD5
	case config.AlgoSHA256:
		return result.SHA256
	default:
		return ""
	}
}

// String returns a formatted string of all hashes
func (result HashResult) String() string {
	s := ""
	if result.XXHash64 != "" {
		s += fmt.Sprintf("xxhash64:%s ", result.XXHash64)
	}
	if result.MD5 != "" {
		s += fmt.Sprintf("md5:%s ", result.MD5)
	}
	if result.SHA256 != "" {
		s += fmt.Sprintf("sha256:%s", result.SHA256)
	}
	return s
}

// CalculateFileHash calculates hash(es) for a file
func CalculateFileHash(path string, algorithms ...config.HashAlgorithm) (HashResult, error) {
	f, err := io.Open(path)
	if err != nil {
		return HashResult{}, err
	}
	defer f.Close()

	hasher := NewMultiHasher(algorithms...)
	
	// Use 4MB buffer for efficient reading
	buf := make([]byte, 4*1024*1024)
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return HashResult{}, err
	}

	return hasher.Sum(), nil
}
Update offloader.go to use new hash system:
go// internal/offload/offloader.go
// Update imports
import (
	// ... existing imports ...
	"loot/internal/config"
	"loot/internal/hash"
)

// Update Offloader struct
type Offloader struct {
	Source       string
	Destinations []string
	BufferSize   int
	
	Config       *config.Config // Add this
	
	SourceHash   hash.HashResult // Change from string
	DestHash     hash.HashResult // Change from string

	Files []FileRes
}

// Update NewOffloader
func NewOffloader(src string, dsts ...string) *Offloader {
	return NewOffloaderWithConfig(config.DefaultConfig(), src, dsts...)
}

func NewOffloaderWithConfig(cfg *config.Config, src string, dsts ...string) *Offloader {
	return &Offloader{
		Source:       src,
		Destinations: dsts,
		BufferSize:   cfg.BufferSize,
		Config:       cfg,
	}
}

// Update copyFileMultiLoop to use new hasher
func (o *Offloader) copyFileMultiLoop(srcFile *os.File, writers []io.Writer, hasher io.Writer, t *tracker, fileName string) error {
	// Create multi-hasher based on config
	var hashWriter io.Writer
	
	if o.Config.DualHash {
		hashWriter = hash.NewMultiHasher(config.AlgoXXHash64, config.AlgoMD5)
	} else {
		hashWriter = hash.NewHasher(o.Config.Algorithm)
	}
	
	allWriters := append(writers, hashWriter)
	multiDest := io.MultiWriter(allWriters...)

	buf := make([]byte, o.BufferSize)
	for {
		n, err := srcFile.Read(buf)
		if n > 0 {
			if _, wErr := multiDest.Write(buf[:n]); wErr != nil {
				return wErr
			}
			t.update(n, filepath.Base(fileName))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Update calculateXXHash to calculateFileHash
func calculateFileHash(path string, cfg *config.Config) (hash.HashResult, error) {
	if cfg.DualHash {
		return hash.CalculateFileHash(path, config.AlgoXXHash64, config.AlgoMD5)
	}
	return hash.CalculateFileHash(path, cfg.Algorithm)
}
Tests to validate:
bash# Create test file
echo "test data" > /tmp/test.txt

# Test xxhash64
go run cmd/loot/main.go --algorithm xxhash64 /tmp/test.txt /tmp/backup.txt

# Test MD5
go run cmd/loot/main.go --algorithm md5 /tmp/test.txt /tmp/backup2.txt

# Test SHA256
go run cmd/loot/main.go --algorithm sha256 /tmp/test.txt /tmp/backup3.txt

# Test dual-hash
go run cmd/loot/main.go --dual-hash /tmp/test.txt /tmp/backup4.txt

# Verify with system tools
md5 /tmp/test.txt
shasum -a 256 /tmp/test.txt
# Compare with LOOT output
Success criteria:

 xxHash64 fonctionne (existing behavior)
 MD5 calcul correct (comparé avec md5 command)
 SHA256 calcul correct (comparé avec shasum)
 Dual-hash calcule les deux
 Performance acceptable


🎯 PHASE 2: JSON Output & Job Management (Semaine 2-3)
TASK 2.1: Créer le système de JSON output
Priority: 🟠 HIGH
Estimated time: 2h
Dependencies: TASK 1.4
Context:

Agents IA ont besoin de JSON structuré pour parser les résultats
Doit inclure toutes les métriques importantes
Compatible avec --quiet pour scripts

Files to create:

internal/output/json.go

Implementation:
go// internal/output/json.go
package output

import (
	"encoding/json"
	"fmt"
	"time"

	"loot/internal/config"
	"loot/internal/hash"
	"loot/internal/offload"
)

// JobResult represents the complete result of an offload job
type JobResult struct {
	JobID        string              `json:"job_id"`
	Status       string              `json:"status"` // success, failed, partial
	Source       string              `json:"source"`
	Destinations []string            `json:"destinations"`
	FilesCount   int                 `json:"files_count"`
	TotalBytes   int64               `json:"total_bytes"`
	Duration     float64             `json:"duration_seconds"`
	Speed        float64             `json:"speed_mbps"`
	Hash         hash.HashResult     `json:"hash"`
	Algorithm    config.HashAlgorithm `json:"algorithm"`
	DualHash     bool                `json:"dual_hash"`
	Verified     bool                `json:"verified"`
	Errors       []string            `json:"errors,omitempty"`
	Timestamp    time.Time           `json:"timestamp"`
	Files        []FileResult        `json:"files,omitempty"`
}

// FileResult represents info about a single file
type FileResult struct {
	Path     string          `json:"path"`
	Size     int64           `json:"size"`
	Hash     hash.HashResult `json:"hash"`
	Verified bool            `json:"verified"`
}

// GenerateJobID creates a unique job ID
func GenerateJobID() string {
	return fmt.Sprintf("loot-%s", time.Now().Format("20060102-150405"))
}

// NewJobResult creates a JobResult from offloader
func NewJobResult(o *offload.Offloader, startTime, endTime time.Time, err error) *JobResult {
	result := &JobResult{
		JobID:        GenerateJobID(),
		Source:       o.Source,
		Destinations: o.Destinations,
		FilesCount:   len(o.Files),
		Timestamp:    time.Now(),
		Algorithm:    o.Config.Algorithm,
		DualHash:     o.Config.DualHash,
	}

	// Calculate total size
	for _, f := range o.Files {
		result.TotalBytes += f.Size
	}

	// Calculate duration and speed
	duration := endTime.Sub(startTime)
	result.Duration = duration.Seconds()
	
	if result.Duration > 0 {
		bytesPerSecond := float64(result.TotalBytes) / result.Duration
		result.Speed = (bytesPerSecond / (1024 * 1024)) * 8 // Convert to Mbps
	}

	// Set status
	if err != nil {
		result.Status = "failed"
		result.Errors = append(result.Errors, err.Error())
		result.Verified = false
	} else {
		// Check if verification passed
		if o.SourceHash.GetPrimary(o.Config.Algorithm) == o.DestHash.GetPrimary(o.Config.Algorithm) {
			result.Status = "success"
			result.Verified = true
		} else {
			result.Status = "failed"
			result.Verified = false
			result.Errors = append(result.Errors, "checksum mismatch")
		}
	}

	result.Hash = o.SourceHash

	// Include file details if requested
	// (we can add a flag for this later)
	for _, f := range o.Files {
		result.Files = append(result.Files, FileResult{
			Path:     f.RelPath,
			Size:     f.Size,
			Hash:     hash.HashResult{}, // TODO: parse from f.Hash string
			Verified: true,
		})
	}

	return result
}

// Print outputs the result in the specified format
func (jr *JobResult) Print(cfg *config.Config) error {
	if cfg.JSONOutput {
		return jr.PrintJSON()
	}
	return jr.PrintHuman()
}

// PrintJSON outputs result as JSON
func (jr *JobResult) PrintJSON() error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jr)
}

// PrintHuman outputs result in human-readable format
func (jr *JobResult) PrintHuman() error {
	fmt.Printf("\n")
	fmt.Printf("Job ID: %s\n", jr.JobID)
	fmt.Printf("Status: %s\n", jr.Status)
	fmt.Printf("Source: %s\n", jr.Source)
	for i, dest := range jr.Destinations {
		fmt.Printf("Dest %d: %s\n", i+1, dest)
	}
	fmt.Printf("Files:  %d\n", jr.FilesCount)
	fmt.Printf("Size:   %s\n", formatBytes(uint64(jr.TotalBytes)))
	fmt.Printf("Time:   %.1fs\n", jr.Duration)
	fmt.Printf("Speed:  %.1f MB/s\n", jr.Speed/8) // Convert Mbps to MB/s
	
	if jr.DualHash {
		fmt.Printf("Hash:   %s\n", jr.Hash.String())
	} else {
		fmt.Printf("Hash:   %s\n", jr.Hash.GetPrimary(jr.Algorithm))
	}
	
	if jr.Verified {
		fmt.Printf("✅ Verification: SUCCESS\n")
	} else {
		fmt.Printf("❌ Verification: FAILED\n")
		for _, err := range jr.Errors {
			fmt.Printf("   Error: %s\n", err)
		}
	}
	
	return nil
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
Add to model.go to use JSON output:
go// In verifyFinishedMsg handler
case verifyFinishedMsg:
	m.done = true
	m.verifying = false
	m.state = stateDone
	m.endTime = time.Now()

	// Generate job result
	jobResult := output.NewJobResult(m.offloader, m.startTime, m.endTime, msg.err)
	
	// Print result
	if m.config.JSONOutput {
		jobResult.PrintJSON()
		return m, tea.Quit
	}
	
	// ... rest of existing code for TUI display
Tests to validate:
bash# Test JSON output
go run cmd/loot/main.go --json /tmp/test.txt /tmp/backup.txt
# Should output valid JSON

# Test JSON + quiet
go run cmd/loot/main.go --json --quiet /tmp/test.txt /tmp/backup.txt
# Should output ONLY JSON

# Pipe to jq
go run cmd/loot/main.go --json /tmp/test.txt /tmp/backup.txt | jq .
# Should be valid JSON

# Extract specific fields
go run cmd/loot/main.go --json /tmp/test.txt /tmp/backup.txt | jq '.status'
# Should output "success" or "failed"
Success criteria:

 JSON output valide
 Tous les champs présents
 Compatible avec jq
 --quiet supprime le TUI


TASK 2.2: Système de persistence des jobs
Priority: 🟡 MEDIUM
Estimated time: 3h
Dependencies: TASK 2.1
Context:

Permet de resume les transferts interrompus
Historique des jobs pour audit
Base pour feature "resume"

Files to create:

internal/job/manager.go
internal/job/state.go

Implementation:
go// internal/job/state.go
package job

import (
	"time"
	
	"loot/internal/config"
	"loot/internal/hash"
)

type JobStatus string

const (
	StatusRunning   JobStatus = "running"
	StatusPaused    JobStatus = "paused"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

// FileProgress tracks progress of a single file
type FileProgress struct {
	Path        string          `json:"path"`
	Size        int64           `json:"size"`
	BytesCopied int64           `json:"bytes_copied"`
	Hash        hash.HashResult `json:"hash,omitempty"`
	Completed   bool            `json:"completed"`
}

// JobState represents the state of a job (for persistence)
type JobState struct {
	ID           string              `json:"id"`
	Source       string              `json:"source"`
	Destinations []string            `json:"destinations"`
	Status       JobStatus           `json:"status"`
	Algorithm    config.HashAlgorithm `json:"algorithm"`
	DualHash     bool                `json:"dual_hash"`
	
	Files        []FileProgress      `json:"files"`
	TotalBytes   int64               `json:"total_bytes"`
	CopiedBytes  int64               `json:"copied_bytes"`
	
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	CompletedAt  *time.Time          `json:"completed_at,omitempty"`
	
	Error        string              `json:"error,omitempty"`
}

// Progress returns current progress percentage
func (js *JobState) Progress() float64 {
	if js.TotalBytes == 0 {
		return 0
	}
	return float64(js.CopiedBytes) / float64(js.TotalBytes)
}

// UpdateProgress updates the copied bytes
func (js *JobState) UpdateProgress(file string, bytes int64) {
	js.CopiedBytes += bytes
	js.UpdatedAt = time.Now()
	
	// Update file progress
	for i := range js.Files {
		if js.Files[i].Path == file {
			js.Files[i].BytesCopied += bytes
			if js.Files[i].BytesCopied >= js.Files[i].Size {
				js.Files[i].Completed = true
			}
			break
		}
	}
}
go// internal/job/manager.go
package job

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager handles job persistence
type Manager struct {
	JobsDir string
}

// NewManager creates a job manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	jobsDir := filepath.Join(home, ".loot", "jobs")
	if err := os.MkdirAll(jobsDir, 0755); err != nil {
		return nil, err
	}
	
	return &Manager{JobsDir: jobsDir}, nil
}

// Save persists job state to disk
func (m *Manager) Save(state *JobState) error {
	state.UpdatedAt = time.Now()
	
	path := filepath.Join(m.JobsDir, state.ID+".json")
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// Load retrieves job state from disk
func (m *Manager) Load(jobID string) (*JobState, error) {
	path := filepath.Join(m.JobsDir, jobID+".json")
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var state JobState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	
	return &state, nil
}

// List returns all jobs
func (m *Manager) List() ([]*JobState, error) {
	entries, err := os.ReadDir(m.JobsDir)
	if err != nil {
		return nil, err
	}
	
	var jobs []*JobState
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		
		jobID := entry.Name()[:len(entry.Name())-5] // Remove .json
		job, err := m.Load(jobID)
		if err != nil {
			continue // Skip corrupted files
		}
		jobs = append(jobs, job)
	}
	
	return jobs, nil
}

// Delete removes a job from disk
func (m *Manager) Delete(jobID string) error {
	path := filepath.Join(m.JobsDir, jobID+".json")
	return os.Remove(path)
}

// CleanCompleted removes all completed jobs
func (m *Manager) CleanCompleted() error {
	jobs, err := m.List()
	if err != nil {
		return err
	}
	
	for _, job := range jobs {
		if job.Status == StatusCompleted {
			m.Delete(job.ID)
		}
	}
	
	return nil
}

// GetResumable returns all jobs that can be resumed
func (m *Manager) GetResumable() ([]*JobState, error) {
	jobs, err := m.List()
	if err != nil {
		return nil, err
	}
	
	var resumable []*JobState
	for _, job := range jobs {
		if job.Status == StatusRunning || job.Status == StatusPaused {
			resumable = append(resumable, job)
		}
	}
	
	return resumable, nil
}
Add job persistence to offloader.go:
go// In Offloader struct
type Offloader struct {
	// ... existing fields ...
	JobManager *job.Manager
	JobState   *job.JobState
}

// Update Copy method to save progress periodically
func (o *Offloader) Copy(progressChan chan<- ProgressInfo) error {
	// ... existing code ...
	
	// Save job state periodically (every 5 seconds)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	go func() {
		for range ticker.C {
			if o.JobManager != nil && o.JobState != nil {
				o.JobManager.Save(o.JobState)
			}
		}
	}()
	
	// ... rest of copy logic ...
}
Tests to validate:
bash# Start a job and interrupt it
go run cmd/loot/main.go /large/folder /backup
# Press Ctrl+C after a few seconds

# Check job was saved
ls ~/.loot/jobs/
# Should see loot-YYYYMMDD-HHMMSS.json

# View job state
cat ~/.loot/jobs/loot-*.json | jq .
# Should show partial progress

# List jobs
go run cmd/loot/main.go jobs list
# Should show saved job

# Resume job (TODO: implement in next task)
Success criteria:

 Jobs sauvegardés dans ~/.loot/jobs/
 État JSON valide
 Progrès persisté correctement
 Liste des jobs fonctionne

 