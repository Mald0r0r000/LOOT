package metadata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MediaMetadata holds technical information extracted from parsers
// This struct mirrors Metadata but is used for the parser interface return type
// converting is easy as fields align generally
type MediaMetadata struct {
	Source           string // "r3d_header", "exiftool", etc
	Format           string
	Codec            string
	Resolution       string
	Width            int
	Height           int
	FPS              string
	Duration         string
	DurationFrames   int
	Timecode         string
	CameraModel      string
	CameraID         string
	ReelNumber       string
	ClipName         string
	TakeNumber       string
	SerialNumber     string
	ISO              int
	ColorTemperature int
	VideoFormat      string
	Quality          string
}

// ToMetadata converts MediaMetadata to the main Metadata struct
func (m *MediaMetadata) ToMetadata() *Metadata {
	return &Metadata{
		Format:     m.Format,
		Codec:      m.Codec,
		Resolution: m.Resolution,
		FrameRate:  m.FPS,
		Duration:   m.Duration,
		Timecode:   m.Timecode,
		CameraID:   m.CameraID,
		ReelNumber: m.ReelNumber,
		ClipName:   m.ClipName,
		Take:       m.TakeNumber,
	}
}

// Parser extracts metadata from file headers
type Parser interface {
	// CanHandle returns true if this parser can handle the file extension
	CanHandle(ext string) bool

	// Parse extracts metadata from reader (limited to maxBytes)
	Parse(r io.Reader, maxBytes int) (*MediaMetadata, error)

	// Name returns parser name for debugging
	Name() string
}

// ParserRegistry manages available parsers
type ParserRegistry struct {
	parsers []Parser
}

var defaultRegistry = &ParserRegistry{
	parsers: []Parser{},
}

// RegisterParser adds a parser to the registry
func RegisterParser(parser Parser) {
	defaultRegistry.parsers = append(defaultRegistry.parsers, parser)
}

// GetParser returns appropriate parser for file extension
func GetParser(ext string) Parser {
	ext = strings.ToLower(ext)
	for _, parser := range defaultRegistry.parsers {
		if parser.CanHandle(ext) {
			return parser
		}
	}
	return nil
}

// ParseHeader attempts to extract metadata from file header
func ParseHeader(path string) (*MediaMetadata, error) {
	ext := strings.ToLower(filepath.Ext(path))
	parser := GetParser(ext)

	if parser == nil {
		return nil, fmt.Errorf("no parser for extension: %s", ext)
	}

	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read only header (128KB max for speed, sufficient for most headers)
	// Some formats like R3D have headers at start. MOV/MP4 might have moov atom at end (requires seeking, not just Reader)
	// For R3D, header is at start.
	const headerLimit = 128 * 1024
	lr := io.LimitReader(f, headerLimit)

	return parser.Parse(lr, headerLimit)
}
