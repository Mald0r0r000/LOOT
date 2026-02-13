package offload

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"loot/internal/config"
	"loot/internal/hash"
	"loot/internal/metadata"
)

const BufferSize = 4 * 1024 * 1024 // 4MB

type ProgressInfo struct {
	TotalBytes  int64
	CopiedBytes int64
	CurrentFile string
	Speed       float64 // bytes per second
}

type FileRes struct {
	RelPath  string
	Size     int64
	ModTime  time.Time
	Hash     hash.HashResult
	Metadata *metadata.Metadata
}

// Offloader handles the copy process
type Offloader struct {
	Source       string
	Destinations []string
	BufferSize   int
	SourceHash   hash.HashResult
	DestHash     hash.HashResult // We might need a map for multi-dest, but for now let's keep it simple (concat or first?)
	// Actually, let's store per-destination hash? Or just verify all match source.
	// For Report, we want to show all are verified.

	Files  []FileRes
	Config *config.Config

	// Temporary cache for metadata extracted during Copy
	metadataCache sync.Map
}

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

// tracker maintains state across multiple files
type tracker struct {
	mu           sync.Mutex
	TotalBytes   int64
	CopiedBytes  int64
	StartTime    time.Time
	LastUpdate   time.Time
	ProgressChan chan<- ProgressInfo
}

func (t *tracker) update(n int, file string) {
	t.mu.Lock()
	defer t.mu.Unlock()

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

type copyJob struct {
	path    string
	relPath string
}

func (o *Offloader) Copy(ctx context.Context, progressChan chan<- ProgressInfo) error {
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
			if err := ctx.Err(); err != nil {
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

		// Parallel Copy Logic
		numWorkers := o.Config.Concurrency
		if numWorkers < 1 {
			numWorkers = 1
		}

		jobs := make(chan copyJob, numWorkers)
		results := make(chan error, numWorkers)
		var wg sync.WaitGroup

		// Start Workers
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case j, ok := <-jobs:
						if !ok {
							return
						}
						var dstPaths []string
						for _, dstRoot := range o.Destinations {
							dstPaths = append(dstPaths, filepath.Join(dstRoot, j.relPath))
						}
						// Extract Metadata BEFORE copy (as requested for optimization/streaming)
						// This primes the OS cache for the header at least.
						// We ignore error here, we'll try again in verify or just log it?
						// For now, best effort.
						meta, _ := metadata.Extract(j.path, o.Config.MetadataMode)
						if meta != nil {
							o.metadataCache.Store(j.relPath, meta)
						}

						if err := o.copyFileMulti(ctx, j.path, dstPaths, t); err != nil {
							select {
							case results <- err:
							default:
							}
						}
					}
				}
			}()
		}

		// Feed jobs
		go func() {
			defer close(jobs)
			walkErr := filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if err := ctx.Err(); err != nil {
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
					// Create directories synchronously to ensure they exist for files
					for _, dstRoot := range o.Destinations {
						destPath := filepath.Join(dstRoot, relPath)
						if err := os.MkdirAll(destPath, info.Mode()); err != nil {
							return fmt.Errorf("failed to create dir %s: %w", destPath, err)
						}
					}
					return nil
				}

				// Send file job
				select {
				case jobs <- copyJob{path: path, relPath: relPath}:
				case <-ctx.Done(): // Check context during send block
					return ctx.Err()
				}
				return nil
			})

			if walkErr != nil {
				results <- walkErr
			}
		}()

		// Wait for workers
		go func() {
			wg.Wait()
			close(results)
		}()

		// Collect errors (return first one)
		var firstErr error
		for err := range results {
			if err != nil && firstErr == nil {
				firstErr = err
				// Do not return early. Wait for results channel to close (which happens after wg.Wait())
				// This prevents the Job from closing the progress channel while workers are still running.
			}
		}

		return firstErr

	} else {
		// Single file
		t.TotalBytes = info.Size()
		return o.copyFileMulti(ctx, o.Source, o.Destinations, t)
	}
}

// copyFileMulti copies src to multiple destinations simultaneously
func (o *Offloader) copyFileMulti(ctx context.Context, src string, dests []string, t *tracker) error {
	// ... (Skipping Stat and Prep logic which doesn't need context explicitly, but loop does)

	// 1. Stat Source first for size comparison
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// 2. Prepare Destinations
	var openFiles []*os.File
	var writers []io.Writer

	// Cleanup helper
	defer func() {
		for _, f := range openFiles {
			f.Close()
		}
	}()

	skippedCount := 0
	for _, dstPath := range dests {
		// Check for SkipExisting
		if o.Config.SkipExisting {
			if dstInfo, err := os.Stat(dstPath); err == nil && !dstInfo.IsDir() {
				if dstInfo.Size() == srcInfo.Size() {
					skippedCount++
					continue
				}
			}
		}

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

	// If all skipped
	if len(writers) == 0 {
		t.update(int(srcInfo.Size()), filepath.Base(src)+" (skipped)")
		return nil
	}

	// Check context before opening src
	if err := ctx.Err(); err != nil {
		return err
	}

	// 3. Open Source
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 4. Custom Loop for Copy + Progress + Hash
	var hashWriter io.Writer
	if o.Config.DualHash {
		hashWriter = hash.NewMultiHasher(config.AlgoXXHash64, config.AlgoMD5)
	} else {
		hashWriter = hash.NewHasher(o.Config.Algorithm)
	}

	return o.copyFileMultiLoop(ctx, srcFile, writers, hashWriter, t, src)
}

// BufferPool to reduce GC pressure
var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 1024*1024) // 1MB buffer
		return &b
	},
}

func (o *Offloader) copyFileMultiLoop(ctx context.Context, srcFile *os.File, writers []io.Writer, hasher io.Writer, t *tracker, fileName string) error {
	allWriters := append(writers, hasher)
	multiDest := io.MultiWriter(allWriters...)

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	pw := &progressWriter{
		w:        multiDest,
		tracker:  t,
		fileName: filepath.Base(fileName),
		ctx:      ctx,
	}

	_, err := io.CopyBuffer(pw, srcFile, buf)
	return err
}

type progressWriter struct {
	w        io.Writer
	tracker  *tracker
	fileName string
	ctx      context.Context
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	if err := pw.ctx.Err(); err != nil {
		return 0, err
	}
	n, err = pw.w.Write(p)
	if n > 0 {
		pw.tracker.update(n, pw.fileName)
	}
	return
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
	// For Directory verification, we don't have a single "SourceHash" that is easy to check against.
	// unless we implement a merkle tree or similar.
	// For now, let's verify file by file.

	// We will set SourceHash/DestHash to a dummy value or the last specific error?
	// Or maybe "VERIFIED" string?
	// Since HashResult is structured, we can't just put "VERIFIED".
	// We'll leave them empty for Dir-level, or use a specific indicator in a future field.

	err := filepath.Walk(o.Source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .DS_Store
		if info.Name() == ".DS_Store" {
			return nil
		}

		if !info.IsDir() {
			srcH, err := calculateFileHash(path, o.Config)
			if err != nil {
				return err
			}

			relPath, _ := filepath.Rel(o.Source, path)

			// Verify all destinations
			for _, dstRoot := range o.Destinations {
				dstPath := filepath.Join(dstRoot, relPath)
				dstH, err := calculateFileHash(dstPath, o.Config)
				if err != nil {
					return fmt.Errorf("failed to hash dest %s: %w", dstPath, err)
				}

				// Compare Primary
				algo := o.Config.Algorithm
				if srcH.GetPrimary(algo) != dstH.GetPrimary(algo) {
					return fmt.Errorf("mismatch: %s vs %s", relPath, dstPath)
				}
				// Check dual hash if enabled
				if o.Config.DualHash {
					if srcH.MD5 != dstH.MD5 {
						return fmt.Errorf("MD5 mismatch: %s vs %s", relPath, dstPath)
					}
				}
			}

			// Extract Metadata (best effort)
			var paramMeta *metadata.Metadata
			if cached, ok := o.metadataCache.Load(relPath); ok {
				paramMeta = cached.(*metadata.Metadata)
			} else {
				// Fallback if missed during copy
				paramMeta, _ = metadata.Extract(path, o.Config.MetadataMode)
			}

			// Store metadata (using Source Hash)
			o.Files = append(o.Files, FileRes{
				RelPath:  relPath,
				Size:     info.Size(),
				ModTime:  info.ModTime(),
				Hash:     srcH,
				Metadata: paramMeta,
			})
		}
		return nil
	})

	// How to represent "DIR VERIFIED" in a hash struct?
	// Maybe we just don't set SourceHash/DestHash for root dir
	// OR we set it to XXHash of the list of hashes? Too complex for now.
	return err == nil, err
}

func (o *Offloader) verifyFile() (bool, error) {
	srcH, err := calculateFileHash(o.Source, o.Config)
	if err != nil {
		return false, err
	}
	o.SourceHash = srcH

	for _, dstPath := range o.Destinations {
		dstH, err := calculateFileHash(dstPath, o.Config)
		if err != nil {
			return false, err
		}

		algo := o.Config.Algorithm
		if srcH.GetPrimary(algo) != dstH.GetPrimary(algo) {
			return false, fmt.Errorf("mismatch: %s", dstPath)
		}
		if o.Config.DualHash {
			if srcH.MD5 != dstH.MD5 {
				return false, fmt.Errorf("MD5 mismatch: %s", dstPath)
			}
		}
	}

	o.DestHash = o.SourceHash

	// Metadata
	info, _ := os.Stat(o.Source)
	paramMeta, _ := metadata.Extract(o.Source, o.Config.MetadataMode) // Best effort

	o.Files = append(o.Files, FileRes{
		RelPath:  filepath.Base(o.Source),
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Hash:     o.SourceHash,
		Metadata: paramMeta,
	})

	return true, nil
}

func calculateFileHash(path string, cfg *config.Config) (hash.HashResult, error) {
	if cfg.DualHash {
		return hash.CalculateFileHash(path, config.AlgoXXHash64, config.AlgoMD5)
	}
	return hash.CalculateFileHash(path, cfg.Algorithm)
}

// DryRunResult holds the results of a simulation
type DryRunResult struct {
	Source       string
	Files        []FileRes
	TotalSize    int64
	Destinations []DestInfo
}

// DestInfo holds information about a destination
type DestInfo struct {
	Path      string
	FreeSpace uint64
	CanFit    bool
}

// DryRun simulates the copy operation
func (o *Offloader) DryRun() (*DryRunResult, error) {
	result := &DryRunResult{
		Source: o.Source,
	}

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
				result.Files = append(result.Files, FileRes{
					RelPath:  relPath,
					Size:     info.Size(),
					ModTime:  info.ModTime(),
					Metadata: nil, // No metadata in dry run
				})
				result.TotalSize += info.Size()
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		result.Files = append(result.Files, FileRes{
			RelPath:  filepath.Base(o.Source),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
			Metadata: nil,
		})
		result.TotalSize = info.Size()
	}

	// Check destinations
	for _, dst := range o.Destinations {
		di := DestInfo{
			Path: dst,
		}

		// Get free space (best effort)
		// We use the parent directory if dst doesn't exist yet
		checkPath := dst
		if _, err := os.Stat(checkPath); os.IsNotExist(err) {
			checkPath = filepath.Dir(checkPath)
		}

		var stat syscall.Statfs_t
		if err := syscall.Statfs(checkPath, &stat); err == nil {
			// Available blocks * block size
			di.FreeSpace = uint64(stat.Bavail) * uint64(stat.Bsize)
			di.CanFit = di.FreeSpace > uint64(result.TotalSize)
		} else {
			// If we can't check, assume yes but warn?
			// For now, set FreeSpace to 0 (unknown)
			di.CanFit = true // Optimistic
		}
		result.Destinations = append(result.Destinations, di)
	}

	return result, nil
}
