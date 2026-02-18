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

func TestShouldSkip(t *testing.T) {
	// System files/dirs that MUST be skipped
	skip := []string{
		".DS_Store",
		".Spotlight-V100",
		".fseventsd",
		".Trashes",
		".TemporaryItems",
		".DocumentRevisions-V100",
	}
	for _, name := range skip {
		if !shouldSkip(name) {
			t.Errorf("shouldSkip(%q) = false, want true", name)
		}
	}

	// Normal files that must NOT be skipped
	keep := []string{
		"A001C001_220101_R4K5.R3D",
		"clip.mov",
		".hidden_but_ok",
		"CLIPINFO.TXT",
	}
	for _, name := range keep {
		if shouldSkip(name) {
			t.Errorf("shouldSkip(%q) = true, want false", name)
		}
	}
}

func TestCopySkipsSystemFiles(t *testing.T) {
	// Create source with a fake .Spotlight-V100 dir
	srcDir, err := ioutil.TempDir("", "loot_src_skip_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(srcDir)

	// Real file
	if err := ioutil.WriteFile(filepath.Join(srcDir, "clip.mov"), []byte("video"), 0644); err != nil {
		t.Fatal(err)
	}

	// System dir that should be skipped
	spotlightDir := filepath.Join(srcDir, ".Spotlight-V100", "Store-V2")
	if err := os.MkdirAll(spotlightDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(spotlightDir, "0.directoryStoreFile"), []byte("index"), 0644); err != nil {
		t.Fatal(err)
	}

	// Dest
	dstDir, err := ioutil.TempDir("", "loot_dst_skip_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dstDir)

	cfg := config.DefaultConfig()
	cfg.Algorithm = "md5"
	o := NewOffloaderWithConfig(cfg, srcDir, dstDir)

	progressChan := make(chan ProgressInfo, 10)
	go func() {
		for range progressChan {
		}
	}()

	if err := o.Copy(context.Background(), progressChan); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Real file should exist
	if _, err := os.Stat(filepath.Join(dstDir, "clip.mov")); os.IsNotExist(err) {
		t.Error("clip.mov should have been copied")
	}

	// System dir should NOT exist
	if _, err := os.Stat(filepath.Join(dstDir, ".Spotlight-V100")); !os.IsNotExist(err) {
		t.Error(".Spotlight-V100 should NOT have been copied")
	}

	// Verify should pass (no mismatch on system files)
	ok, err := o.Verify()
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !ok {
		t.Error("Verify should return true")
	}
}
