package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"loot/internal/offload"
)

// JobResult represents the final status of an offload job
type JobResult struct {
	Timestamp    time.Time         `json:"timestamp"`
	Source       string            `json:"source"`
	Destinations []string          `json:"destinations"`
	Status       string            `json:"status"` // "success", "failed"
	TotalFiles   int               `json:"total_files"`
	TotalBytes   int64             `json:"total_bytes"`
	Duration     string            `json:"duration"`
	DurationMs   int64             `json:"duration_ms"`
	SpeedMBps    float64           `json:"speed_mbps"`
	Files        []offload.FileRes `json:"files,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// PrintJSON outputs the result as formatted JSON to stdout
func PrintJSON(result JobResult) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// PrintHuman outputs a human-readable summary (CLI text mode)
func PrintHuman(result JobResult) {
	fmt.Println()
	fmt.Printf("Source: %s\n", result.Source)
	fmt.Println("Destinations:")
	for _, d := range result.Destinations {
		fmt.Printf("  - %s\n", d)
	}
	fmt.Println()

	if result.Status == "success" {
		fmt.Println("✅ Verification successful!")
		fmt.Printf("Processed %d files (%s) in %s\n",
			result.TotalFiles,
			offload.FormatBytes(uint64(result.TotalBytes)),
			result.Duration,
		)
		fmt.Printf("Average Speed: %.2f MB/s\n", result.SpeedMBps)
	} else {
		fmt.Printf("❌ Job Failed: %s\n", result.Error)
	}
}
