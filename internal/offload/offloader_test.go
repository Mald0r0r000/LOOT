package offload

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"loot/internal/config"
)

func TestOffloader_Copy(t *testing.T) {
	// 1. Setup Validation
	// Create Temp Source
	srcDir, err := ioutil.TempDir("", "loot_src_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Create dummy file
	testFile := filepath.Join(srcDir, "test.txt")
	content := []byte("Hello LOOT")
	if err := ioutil.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Create Temp Dest
	dstDir, err := ioutil.TempDir("", "loot_dst_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	// 2. Setup Offloader
	cfg := config.DefaultConfig()
	cfg.Algorithm = "md5" // Use simple hash

	o := NewOffloaderWithConfig(cfg, srcDir, dstDir)

	// 3. Run Copy
	progressChan := make(chan ProgressInfo, 10)
	// Drain channel
	go func() {
		for range progressChan {
		}
	}()

	err = o.Copy(context.Background(), progressChan)
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// 4. Verify
	destFileExpected := filepath.Join(dstDir, "test.txt")

	if _, err := os.Stat(destFileExpected); os.IsNotExist(err) {
		t.Fatalf("Destination file not created: %s", destFileExpected)
	}

	destContent, err := ioutil.ReadFile(destFileExpected)
	if err != nil {
		t.Fatal(err)
	}

	if string(destContent) != string(content) {
		t.Errorf("Content mismatch. Got %s, want %s", string(destContent), string(content))
	}
}
