package offload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"
)

const BufferSize = 4 * 1024 * 1024 // 4MB

type ProgressInfo struct {
	TotalBytes  int64
	CopiedBytes int64
	CurrentFile string
	Speed       float64 // bytes per second
}

type Offloader struct {
	Source      string
	Destination string
	BufferSize  int
	SourceHash  string
	DestHash    string
}

func NewOffloader(src, dst string) *Offloader {
	return &Offloader{
		Source:      src,
		Destination: dst,
		BufferSize:  BufferSize,
	}
}

// tracker maintains state across multiple files
type tracker struct {
	TotalBytes   int64
	CopiedBytes  int64
	StartTime    time.Time
	LastUpdate   time.Time
	ProgressChan chan<- ProgressInfo
}

func (t *tracker) update(n int, file string) {
	t.CopiedBytes += int64(n)
	now := time.Now()
	if now.Sub(t.LastUpdate) > 50*time.Millisecond || t.CopiedBytes == t.TotalBytes {
		elapsed := now.Sub(t.StartTime).Seconds()
		var speed float64
		if elapsed > 0 {
			speed = float64(t.CopiedBytes) / elapsed
		}

		// Non-blocking send to avoid stalling copy
		select {
		case t.ProgressChan <- ProgressInfo{
			TotalBytes:  t.TotalBytes,
			CopiedBytes: t.CopiedBytes,
			CurrentFile: file,
			Speed:       speed,
		}:
			t.LastUpdate = now
		default:
			// Skip update if channel full
		}
	}
}

func (o *Offloader) Copy(progressChan chan<- ProgressInfo) error {
	info, err := os.Stat(o.Source)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	t := &tracker{
		StartTime:    time.Now(),
		LastUpdate:   time.Now(),
		ProgressChan: progressChan,
	}

	if info.IsDir() {
		// Calculate total size
		err := filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				t.TotalBytes += info.Size()
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to calculate total size: %w", err)
		}

		// Walk and copy
		return filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(o.Source, path)
			if err != nil {
				return err
			}
			destPath := filepath.Join(o.Destination, relPath)

			if info.IsDir() {
				return os.MkdirAll(destPath, info.Mode())
			}

			return o.copyFile(path, destPath, t)
		})
	} else {
		// Single file
		t.TotalBytes = info.Size()
		// Ensure destination directory exists (if user gave a path including filename)
		// But usually Destination IS the file path.
		// If Source is /a/b.txt and Dest is /Volumes/X/b.txt.
		// Construct parent dir.
		destDir := filepath.Dir(o.Destination)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		return o.copyFile(o.Source, o.Destination, t)
	}
}

func (o *Offloader) copyFile(src, dst string, t *tracker) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, o.BufferSize)

	// Custom loop to update tracker
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, wErr := destFile.Write(buf[:n]); wErr != nil {
				return wErr
			}
			t.update(n, filepath.Base(src))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return destFile.Sync()
}

func (o *Offloader) Verify() (bool, error) {
	info, err := os.Stat(o.Source)
	if err != nil {
		return false, err
	}

	if info.IsDir() {
		// Recursive verification
		// Simple approach: Walk and hash everything, combine hashes.
		// We can use a Merkle-like approach or just sum of hashes.
		// Combining xxHashes via XOR or addition is acceptable for simple integrity check.

		var srcCombinedHash uint64
		var dstCombinedHash uint64

		err = filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				h, err := calculateXXHash(path)
				if err != nil {
					return err
				}
				srcCombinedHash ^= h // XOR for combination

				// Verify counterpart immediately?
				relPath, _ := filepath.Rel(o.Source, path)
				dstPath := filepath.Join(o.Destination, relPath)

				dstH, err := calculateXXHash(dstPath)
				if err != nil {
					return err
				}
				dstCombinedHash ^= dstH

				if h != dstH { // Fail fast
					return fmt.Errorf("hash mismatch at %s", relPath)
				}
			}
			return nil
		})

		o.SourceHash = fmt.Sprintf("DIR-%x", srcCombinedHash)
		o.DestHash = fmt.Sprintf("DIR-%x", dstCombinedHash)

		if err != nil {
			return false, err
		}
		return srcCombinedHash == dstCombinedHash, nil
	} else {
		srcHashInt, err := calculateXXHash(o.Source)
		if err != nil {
			return false, err
		}
		o.SourceHash = fmt.Sprintf("%x", srcHashInt)

		dstHashInt, err := calculateXXHash(o.Destination)
		if err != nil {
			return false, err
		}
		o.DestHash = fmt.Sprintf("%x", dstHashInt)

		return srcHashInt == dstHashInt, nil
	}
}

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
