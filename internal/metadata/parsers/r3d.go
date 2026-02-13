package parsers

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"loot/internal/metadata"
)

type R3DParser struct{}

func init() {
	metadata.RegisterParser(&R3DParser{})
}

func (p *R3DParser) Name() string {
	return "R3D"
}

func (p *R3DParser) CanHandle(ext string) bool {
	return strings.ToLower(ext) == ".r3d"
}

func (p *R3DParser) Parse(r io.Reader, maxBytes int) (*metadata.MediaMetadata, error) {
	// Read first 4KB (R3D header is small)
	header := make([]byte, 4096)
	n, err := io.ReadFull(r, header)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read R3D header: %w", err)
	}

	if n < 512 {
		return nil, fmt.Errorf("R3D header too small: %d bytes", n)
	}

	header = header[:n]

	// Validate magic bytes (Offset 4)
	if n < 8 {
		return nil, fmt.Errorf("R3D header too small")
	}
	magic := string(header[4:8])
	if magic != "RED2" && magic != "RED1" {
		return nil, fmt.Errorf("invalid R3D magic: %s (expected RED1/RED2 at offset 4)", magic)
	}

	meta := &metadata.MediaMetadata{
		Source: "r3d_header",
		Codec:  "R3D", // Can be refined to REDCODE RAW if needed
	}

	// Parse R3D header based on observed structure (Big Endian mostly)

	// Image dimensions
	// Observed: Width at 0x4C (4 bytes), Height at 0x50 (4 bytes)
	if len(header) >= 0x54 {
		meta.Width = int(binary.BigEndian.Uint32(header[0x4C:0x50]))
		meta.Height = int(binary.BigEndian.Uint32(header[0x50:0x54]))
		if meta.Width > 0 && meta.Height > 0 {
			meta.Resolution = fmt.Sprintf("%dx%d", meta.Width, meta.Height)
		}
	}

	// FPS observed at 0x58 as scaled integer (e.g. 25000 for 25.0 fps)
	if len(header) >= 0x5C {
		fpsScaled := binary.BigEndian.Uint32(header[0x58:0x5C])
		if fpsScaled > 0 {
			// Check if it looks like scaled FPS (e.g. > 1000)
			if fpsScaled >= 1000 {
				meta.FPS = fmt.Sprintf("%.3f", float64(fpsScaled)/1000.0)
			} else {
				// Maybe straight float or int?
				// Fallback to simple int if small
				meta.FPS = fmt.Sprintf("%d", fpsScaled)
			}
		}
	}

	// Serial Number (KMDBK...)
	// Observed at 0x96
	if len(header) > 0xA0 {
		// Reads until null or space
		start := 0x96
		end := start + 16 // reasonable max length
		if end > len(header) {
			end = len(header)
		}

		serialBytes := header[start:end]
		// Find end of string (null or non-printable)
		validLen := 0
		for i, b := range serialBytes {
			if b < 32 || b > 126 { // Non-printable
				break
			}
			validLen = i + 1
		}
		if validLen > 0 {
			meta.SerialNumber = string(serialBytes[:validLen])
		}
	}

	// Reel (e.g. 006) observed at 0xC2
	// Preceded by 0x1A (type?) and length?
	if len(header) > 0xC8 {
		// 0xC2 seems to start "006"
		meta.ReelNumber = string(header[0xC2:0xC5])
	}

	// Take (e.g. 001) observed at 0xCA
	if len(header) > 0xCD {
		meta.TakeNumber = string(header[0xCA:0xCD])
	}

	return meta, nil
}

func formatDuration(seconds float64) string {
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	secs := int(seconds) % 60
	frames := int((seconds - float64(int(seconds))) * 100) // Rough frame approximation

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d.%02d", hours, minutes, secs, frames)
	}
	return fmt.Sprintf("%02d:%02d.%02d", minutes, secs, frames)
}

func findNull(data []byte) int {
	for i, b := range data {
		if b == 0 {
			return i
		}
	}
	return -1
}
