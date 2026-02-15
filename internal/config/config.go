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
	BufferSize  int // in bytes
	Concurrency int // Number of parallel file copies

	// Dry run
	DryRun bool

	// Resume
	SkipExisting bool

	// Metadata
	JobName      string
	Camera       string
	Reel         string
	MetadataMode string

	// Version info
	Version string
}

// DefaultConfig returns config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Interactive:  true,
		Algorithm:    AlgoXXHash64,
		DualHash:     false,
		BufferSize:   4 * 1024 * 1024, // 4MB
		Concurrency:  4,
		NoVerify:     false,
		DryRun:       false,
		JSONOutput:   false,
		Quiet:        false,
		Verbose:      false,
		SkipExisting: false,
		MetadataMode: "hybrid",
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
	flag.IntVar(&cfg.Concurrency, "concurrency", 4, "Number of parallel file copies")
	flag.IntVar(&cfg.Concurrency, "c", 4, "Number of parallel file copies (shorthand)")
	flag.BoolVar(&cfg.SkipExisting, "skip-existing", false, "Skip files that exist at destination")
	flag.BoolVar(&cfg.SkipExisting, "resume", false, "Resume interrupted transfer (alias for --skip-existing)")
	flag.StringVar(&cfg.JobName, "job-name", "", "Job name for report metadata")
	flag.StringVar(&cfg.Camera, "camera", "", "Camera identifier (e.g. 'A', 'B')")
	flag.StringVar(&cfg.Reel, "reel", "", "Reel identifier (e.g. '001', 'A002')")
	flag.StringVar(&cfg.MetadataMode, "metadata-mode", "hybrid", "Metadata extraction mode: hybrid (default), header, exiftool, off")

	flag.StringVar(&cfg.Source, "source", "", "Source directory")
	flag.StringVar(&cfg.Source, "s", "", "Source directory (shorthand)")
	flag.StringVar(&cfg.Destination, "dest", "", "Destination directory")
	flag.StringVar(&cfg.Destination, "d", "", "Destination directory (shorthand)")
	flag.StringVar(&cfg.Destination, "destination", "", "Destination directory")

	versionFlag := flag.Bool("version", false, "Print version")
	v := flag.Bool("v", false, "Print version (shorthand)")

	// Algo booleans
	md5Flag := flag.Bool("md5", false, "Use MD5 hash algorithm")
	sha256Flag := flag.Bool("sha256", false, "Use SHA256 hash algorithm")
	xxhashFlag := flag.Bool("xxhash64", false, "Use xxHash64 hash algorithm")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "LOOT - Professional Media Offload Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  loot [flags]                  Interactive mode\n")
		fmt.Fprintf(os.Stderr, "  loot [flags] <source> <dest>  CLI mode (positional)\n")
		fmt.Fprintf(os.Stderr, "  loot --source <src> --dest <dst> [flags]  CLI mode (flags)\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Handle version
	if *versionFlag || *v {
		fmt.Printf("loot version %s\n", version)
		os.Exit(0)
	}

	// Parse algorithm priority: Bool Flags > String Flag > Default
	if *md5Flag {
		cfg.Algorithm = AlgoMD5
	} else if *sha256Flag {
		cfg.Algorithm = AlgoSHA256
	} else if *xxhashFlag {
		cfg.Algorithm = AlgoXXHash64
	} else {
		// Fallback to string flag
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
	}

	// Get positional arguments
	args := flag.Args()

	// 1. Check Flags first
	if cfg.Source != "" && cfg.Destination != "" {
		cfg.Interactive = false
	} else if len(args) >= 2 {
		// 2. Check Positional Args (override flags if provided? or fallback?)
		// Let's use positional if flags are empty
		if cfg.Source == "" {
			cfg.Source = args[0]
		}
		if cfg.Destination == "" {
			cfg.Destination = args[1]
		}
		cfg.Interactive = false
	} else if cfg.Source != "" || cfg.Destination != "" {
		// Partial flags?
		return nil, fmt.Errorf("both --source and --dest are required for CLI mode")
	} else {
		// No flags, no positional -> Interactive
		cfg.Interactive = true
	}

	if !cfg.Interactive {
		// Verify source exists
		if _, err := os.Stat(cfg.Source); os.IsNotExist(err) {
			return nil, fmt.Errorf("source '%s' does not exist", cfg.Source)
		}
	}

	return cfg, nil
}
