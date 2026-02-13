package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"loot/internal/config"
	_ "loot/internal/metadata/parsers" // Register parsers
	"loot/internal/offload"
)

func main() {
	src := "/Users/antoinebedos/ownapps/ANTIGRAVITY/LOOT/CARD_VIRTUEL/A005/A006_1123HI.RDM/A006_A001_1123S9.RDC"
	dest := "/tmp/loot_bench_hybrid"

	// Cleanup
	os.RemoveAll(dest)

	cfg := config.DefaultConfig()
	cfg.Concurrency = 4
	cfg.Algorithm = "xxhash"
	cfg.MetadataMode = "hybrid"

	fmt.Printf("Starting Hybrid Benchmark...\n")
	fmt.Printf("Source: %s\n", src)
	fmt.Printf("Dest: %s\n", dest)

	offloader := offload.NewOffloaderWithConfig(cfg, src, dest)

	start := time.Now()

	// Copy
	progressChan := make(chan offload.ProgressInfo, 100)
	go func() {
		for range progressChan {
			// drain
		}
	}()

	fmt.Println("Copying...")
	if err := offloader.Copy(context.Background(), progressChan); err != nil {
		panic(err)
	}
	close(progressChan)

	copyTime := time.Since(start)
	fmt.Printf("Copy finished in %v\n", copyTime)

	// Verify
	fmt.Println("Verifying...")
	verifyStart := time.Now()
	success, err := offloader.Verify()
	verifyTime := time.Since(verifyStart)

	if err != nil || !success {
		panic(fmt.Errorf("verification failed: %v", err))
	}

	totalTime := time.Since(start)
	fmt.Printf("\n=== Benchmark Results ===\n")
	fmt.Printf("Total Time: %v\n", totalTime)
	fmt.Printf("Copy Time: %v\n", copyTime)
	fmt.Printf("Verify Time: %v\n", verifyTime)

	fmt.Printf("\n=== Metadata Extracted ===\n")
	for _, f := range offloader.Files {
		if f.Metadata != nil {
			fmt.Printf("[%s] Res: %s, FPS: %s, Cam: %s, Reel: %s\n",
				f.RelPath, f.Metadata.Resolution, f.Metadata.FrameRate,
				f.Metadata.CameraID, f.Metadata.ReelNumber)
		} else {
			fmt.Printf("[%s] NO METADATA\n", f.RelPath)
		}
	}
}
