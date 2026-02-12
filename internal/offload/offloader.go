package offload

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cespare/xxhash/v2"
)

const BufferSize = 4 * 1024 * 1024 // 4MB

// ProgressInfo contains the current status of the copy operation
type ProgressInfo struct {
	TotalBytes  int64
	CopiedBytes int64
	CurrentFile string
	Speed       float64 // bytes per second
}

// Offloader handles the file copy and verification process
type Offloader struct {
	Source      string
	Destination string
	BufferSize  int
}

// NewOffloader creates a new Offloader instance
func NewOffloader(src, dst string) *Offloader {
	return &Offloader{
		Source:      src,
		Destination: dst,
		BufferSize:  BufferSize,
	}
}

// Copy performs the copy operation and reports progress via the provided channel
// It returns an error if the copy fails
func (o *Offloader) Copy(progressChan chan<- ProgressInfo) error {
	sourceFile, err := os.Open(o.Source)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(o.Destination)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer destFile.Close()

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}
	totalSize := fileInfo.Size()

	// Create a proxy reader to track progress
	reader := &progressReader{
		Reader:       sourceFile,
		Total:        totalSize,
		ProgressChan: progressChan,
		StartTime:    time.Now(),
		LastUpdate:   time.Now(),
	}

	buf := make([]byte, o.BufferSize)
	_, err = io.CopyBuffer(destFile, reader, buf)
	if err != nil {
		return fmt.Errorf("error during copy: %w", err)
	}

	// Ensure all data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination: %w", err)
	}

	return nil
}

// Verify performs a xxHash verification of the source and destination files
func (o *Offloader) Verify() (bool, error) {
	srcHash, err := calculateXXHash(o.Source)
	if err != nil {
		return false, fmt.Errorf("failed to hash source: %w", err)
	}

	dstHash, err := calculateXXHash(o.Destination)
	if err != nil {
		return false, fmt.Errorf("failed to hash destination: %w", err)
	}

	return srcHash == dstHash, nil
}

// calculateXXHash calculates the xxHash of a file
func calculateXXHash(path string) (uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hasher := xxhash.New()
	buf := make([]byte, BufferSize)
	if _, err := io.CopyBuffer(hasher, f, buf); err != nil {
		return 0, err
	}

	return hasher.Sum64(), nil
}

// progressReader wraps io.Reader to track progress
type progressReader struct {
	io.Reader
	Total        int64
	Copied       int64
	ProgressChan chan<- ProgressInfo
	StartTime    time.Time
	LastUpdate   time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Copied += int64(n)

		// Update progress periodically or on completion to avoid flooding the channel
		// However, for smooth UI, we want frequent updates. Bubbletea can handle it.
		// Let's limit updates to every 50ms to be safe if buffer is small,
		// but with 4MB buffer, sends will be infrequent enough naturally (4MB at 1GB/s is 4ms, ok maybe too fast).
		// Actually 4MB chunks are quite large.

		now := time.Now()
		elapsed := now.Sub(pr.StartTime).Seconds()
		speed := float64(pr.Copied) / elapsed

		pr.ProgressChan <- ProgressInfo{
			TotalBytes:  pr.Total,
			CopiedBytes: pr.Copied,
			Speed:       speed,
		}
	}
	return n, err
}
