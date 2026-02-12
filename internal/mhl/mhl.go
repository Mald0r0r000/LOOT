package mhl

import (
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"loot/internal/offload"
)

// MHL represents the Media Hash List structure
type MHL struct {
	XMLName xml.Name `xml:"hashlist"`
	Version string   `xml:"version,attr"`
	Hashes  []Hash   `xml:"hash"`
}

// Hash represents a single file hash entry
type Hash struct {
	File         string `xml:"filename"`
	Size         int64  `xml:"size"`
	LastModified string `xml:"lastmodificationdate"`
	XXHash64     string `xml:"xxhash64,omitempty"`
	MD5          string `xml:"md5,omitempty"`
	SHA1         string `xml:"sha1,omitempty"`
}

// GenerateMHL creates an MHL file for the offload operation
// Note: efficient MHL generation usually requires hashing *during* copy.
// Since we already calculated xxHash in Offloader (for verification),
// we can use that here if we track individual file hashes.
// Current Offloader only exposes a combined hash for directories.
// To support MHL properly, we need detailed file hash logs.
// For now, we will implement a basic generator that might re-scan if needed,
// but ideally we should retain the file list from Offloader.
//
// TODO: Refactor Offloader to return a list of copied files with their hashes.
// For this step, we'll assume we receive a list of file info.
func GenerateMHL(path string, files []offload.FileRes) error {
	mhl := MHL{
		Version: "1.0",
		Hashes:  make([]Hash, len(files)),
	}

	for i, f := range files {
		mhl.Hashes[i] = Hash{
			File:         f.RelPath,
			Size:         f.Size,
			LastModified: f.ModTime.Format(time.RFC3339),
			XXHash64:     f.Hash,
		}
	}

	output, err := xml.MarshalIndent(mhl, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MHL: %w", err)
	}

	header := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return os.WriteFile(path, append(header, output...), 0644)
}
