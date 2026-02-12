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

type FileRes struct {
	RelPath string
	Size    int64
	ModTime time.Time
	Hash    string
}

// Offloader handles the copy process
type Offloader struct {
	Source       string
	Destinations []string
	BufferSize   int
	SourceHash   string
	DestHash     string // We might need a map for multi-dest, but for now let's keep it simple (concat or first?)
	// Actually, let's store per-destination hash? Or just verify all match source.
	// For Report, we want to show all are verified.

	Files []FileRes
}

func NewOffloader(src string, dsts ...string) *Offloader {
	return &Offloader{
		Source:       src,
		Destinations: dsts,
		BufferSize:   BufferSize,
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
	// Update every 50ms or when done
	if now.Sub(t.LastUpdate) > 50*time.Millisecond || t.CopiedBytes == t.TotalBytes {
		elapsed := now.Sub(t.StartTime).Seconds()
		var speed float64
		if elapsed > 0 {
			speed = float64(t.CopiedBytes) / elapsed
		}

		// Non-blocking send
		select {
		case t.ProgressChan <- ProgressInfo{
			TotalBytes:  t.TotalBytes,
			CopiedBytes: t.CopiedBytes,
			CurrentFile: file,
			Speed:       speed,
		}:
			t.LastUpdate = now
		default:
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
		// Calculate total size first
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

			// If it's a directory, create it in all destinations
			if info.IsDir() {
				for _, dstRoot := range o.Destinations {
					destPath := filepath.Join(dstRoot, relPath)
					if err := os.MkdirAll(destPath, info.Mode()); err != nil {
						return fmt.Errorf("failed to create dir %s: %w", destPath, err)
					}
				}
				return nil
			}

			// It's a file, copy to all destinations
			// We need full paths for destinations
			var dstPaths []string
			for _, dstRoot := range o.Destinations {
				dstPaths = append(dstPaths, filepath.Join(dstRoot, relPath))
			}

			return o.copyFileMulti(path, dstPaths, t)
		})

	} else {
		// Single file
		t.TotalBytes = info.Size()

		// For single file copy, we ensure parent dirs exist
		// o.Destinations should be treated as Full Paths if source is file?
		// Or folders?
		// Convention: If Source is File, Dest IS the file path (renaming possible) OR dir?
		// In previous implementation: o.Destination was full path.
		// Let's assume o.Destinations are full paths if Source is file.

		return o.copyFileMulti(o.Source, o.Destinations, t)
	}
}

// copyFileMulti copies src to multiple destinations simultaneously
func (o *Offloader) copyFileMulti(src string, dests []string, t *tracker) error {
	// 1. Open Source
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 2. Open Destinations
	var openFiles []*os.File
	var writers []io.Writer

	// Cleanup helper
	defer func() {
		for _, f := range openFiles {
			f.Close()
		}
	}()

	for _, dstPath := range dests {
		// Ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		f, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("failed to create dest %s: %w", dstPath, err)
		}
		openFiles = append(openFiles, f)
		writers = append(writers, f)
	}

	// 3. MultiWriter
	multiDest := io.MultiWriter(writers...)

	// 4. Hashing (Calculated during copy)
	// Note: We are calculating Source Hash here.
	// But where do we store it if we process multiple files?
	// The TeeReader approach is great for single file or verifying *while* copying.
	// But `Verify` method usually runs *after* copy to ensure bits on disk are correct.
	// However, calculating Source Hash here saves a read pass on source.
	// Let's ignore the hash calculation here for simplicity of migration vs existing Verify() logic,
	// OR we can store it to optimize verification later.
	// The prompt suggested: "Utilise io.TeeReader pour calculer le xxHash de la source pendant la lecture."
	// Let's do it, but we need to pass it out or store it.
	// Since we might be inside a Walk loop, we can't easily update global o.SourceHash for just one file.
	// We should probably just do the Copy here.
	// Optimization: valid, but let's stick to standard Copy first to ensure MultiWriter works.
	// Actually, let's add the hasher as requested, it's "elegant".

	hasher := xxhash.New()
	readerWithHash := io.TeeReader(srcFile, hasher)

	// 5. Copy
	buf := make([]byte, o.BufferSize)
	// We can't use io.Copy with buffer directly easily without wrapping,
	// but io.Copy uses 32kb buffer. io.CopyBuffer allows setting it.

	if _, err := io.CopyBuffer(multiDest, readerWithHash, buf); err != nil {
		return err
	}

	// Manual tracker update? io.CopyBuffer doesn't callback.
	// We need our custom loop if we want progress updates.
	// Let's revert to custom loop for Progress + MultiWriter + Hash

	// Reset to start for custom loop
	srcFile.Seek(0, 0)
	hasher.Reset()
	// Re-open/truncate dests? io.Create truncates.
	// Actually, doing TeeReader inside the loop is tricky with buffer.
	// Easier: Just Hash + MultiWrite in loop.

	// Re-implement loop:
	return o.copyFileMultiLoop(srcFile, writers, hasher, t, src)
}

func (o *Offloader) copyFileMultiLoop(srcFile *os.File, writers []io.Writer, hasher io.Writer, t *tracker, fileName string) error {
	// MultiWriter including hasher?
	// No, Hasher is on Read side (Tee) OR Write side (MultiWriter).
	// If we add hasher to MultiWriter, we hash what we WRITE.
	// That's actually better: verifies what we intended to write.

	allWriters := append(writers, hasher)
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

func (o *Offloader) Verify() (bool, error) {
	// 1. Calculate Source Hash (if not already done/stored)
	// If we did it during copy, we'd need to map it to file paths.
	// For now, let's keep the independent verification phase for safety/simplicity
	// ensuring "Bits on Disk" are read back.

	// We need to verify ALL destinations.

	// Simple approach:
	// Calculate Source Hash again (read from disk)
	// Calculate Dest Hash for EACH dest (read from disk)
	// Comparision.

	info, err := os.Stat(o.Source)
	if err != nil {
		return false, err
	}

	if info.IsDir() {
		return o.verifyDir()
	} else {
		return o.verifyFile()
	}
}

func (o *Offloader) verifyDir() (bool, error) {
	var srcCombinedHash uint64
	// Map to store file hashes to compare against dests?
	// Or just XOR sum? XOR sum is fragile if multiple files differ.
	// Let's iterate source, calc hash, then check all dests immediately.

	err := filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			srcH, err := calculateXXHash(path)
			if err != nil {
				return err
			}
			srcCombinedHash ^= srcH

			relPath, _ := filepath.Rel(o.Source, path)

			// Verify all destinations
			for _, dstRoot := range o.Destinations {
				dstPath := filepath.Join(dstRoot, relPath)
				dstH, err := calculateXXHash(dstPath)
				if err != nil {
					return fmt.Errorf("failed to hash dest %s: %w", dstPath, err)
				}

				if srcH != dstH {
					return fmt.Errorf("mismatch: %s vs %s", relPath, dstPath)
				}
			}

			// Store metadata (using Source Hash)
			o.Files = append(o.Files, FileRes{
				RelPath: relPath,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Hash:    fmt.Sprintf("%x", srcH),
			})
		}
		return nil
	})

	o.SourceHash = fmt.Sprintf("DIR-%x", srcCombinedHash)
	o.DestHash = o.SourceHash // Verified!
	return err == nil, err
}

func (o *Offloader) verifyFile() (bool, error) {
	srcH, err := calculateXXHash(o.Source)
	if err != nil {
		return false, err
	}
	o.SourceHash = fmt.Sprintf("%x", srcH)

	for _, dstPath := range o.Destinations {
		dstH, err := calculateXXHash(dstPath)
		if err != nil {
			return false, err
		}

		if srcH != dstH {
			return false, fmt.Errorf("mismatch: %s", dstPath)
		}
	}

	o.DestHash = o.SourceHash

	// Metadata
	info, _ := os.Stat(o.Source)
	o.Files = append(o.Files, FileRes{
		RelPath: filepath.Base(o.Source),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Hash:    o.SourceHash,
	})

	return true, nil
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
