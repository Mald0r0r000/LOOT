package metadata

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed exiftool_dist/exiftool/exiftool exiftool_dist/exiftool/lib
var exifToolFS embed.FS

// Metadata holds technical information about a media file
type Metadata struct {
	Format     string
	Codec      string
	Resolution string
	FrameRate  string
	Duration   string
	Timecode   string
	Bitrate    string
	CameraID   string
	ReelNumber string
	ClipName   string
	Take       string
}

// FFProbeOutput matches the JSON structure output by ffprobe
type FFProbeOutput struct {
	Streams []struct {
		CodecName    string `json:"codec_name"`
		CodecType    string `json:"codec_type"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		RFrameRate   string `json:"r_frame_rate"`
		AvgFrameRate string `json:"avg_frame_rate"`
		BitRate      string `json:"bit_rate"`
		Tags         struct {
			Timecode string `json:"timecode"`
		} `json:"tags"`
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
		Tags       struct {
			Timecode string `json:"timecode"`
		} `json:"tags"`
	} `json:"format"`
}

// IsAvailable checks if ffprobe is installed and executable
func IsAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// Extract retrieves metadata using specified mode
// modes: "hybrid" (default), "header", "exiftool", "off"
func Extract(path string, mode string) (*Metadata, error) {
	if mode == "off" {
		return nil, nil
	}

	// Only process video/audio extensions to save time/errors
	ext := strings.ToLower(filepath.Ext(path))
	knownExts := map[string]bool{
		".mov": true, ".mp4": true, ".mxf": true, ".mkv": true, ".avi": true,
		".r3d": true, ".braw": true, ".crm": true, ".ari": true,
	}
	if !knownExts[ext] {
		return nil, nil // Not a supported media file
	}

	// 1. FAST: Try Header Parsing (if mode is hybrid or header)
	if mode == "hybrid" || mode == "header" || mode == "" { // default to hybrid if empty
		if mediaMeta, err := ParseHeader(path); err == nil && mediaMeta != nil {
			// Convert to main Metadata struct
			m := mediaMeta.ToMetadata()

			// If mode is header-only, return whatever we found
			if mode == "header" {
				return m, nil
			}

			// Hybrid: If we have enough info, return early (Speed: <5ms)
			if m.Resolution != "" && m.FrameRate != "" {
				return m, nil
			}

			// If partial, keep it and try to enhance with ExifTool?
			// For simplicity, just fallthrough to ExifTool if hybrid and partial
		}
	}

	// 2. SLOW: ExifTool (if hybrid or exiftool)
	if mode == "hybrid" || mode == "exiftool" || mode == "" {
		if meta, err := extractExifTool(path); err == nil && meta != nil {
			return meta, nil
		}
	}

	// 3. FALLBACK: FFprobe (final fallback for hybrid/exiftool)
	if mode == "hybrid" || mode == "exiftool" || mode == "" {
		return extractFFProbe(path)
	}

	return nil, nil
}

func extractFFProbe(path string) (*Metadata, error) {
	// DEBUG LOGGING
	debugLog := func(format string, args ...interface{}) {
		f, err := os.OpenFile("/tmp/loot_metadata.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			timestamp := time.Now().Format("15:04:05.000")
			fmt.Fprintf(f, timestamp+" "+format+"\n", args...)
		}
	}

	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	debugLog("Executing FFprobe: %s", cmd.String())

	output, err := cmd.Output()
	if err != nil {
		debugLog("Error executing ffprobe for %s: %v", path, err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			debugLog("Stderr: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("ffprobe execution failed: %w", err)
	}

	var data FFProbeOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	m := &Metadata{}
	// Format
	m.Format = data.Format.FormatName
	if data.Format.Duration != "" {
		d, _ := time.ParseDuration(data.Format.Duration + "s")
		m.Duration = d.String()
	}

	for _, s := range data.Streams {
		if s.CodecType == "video" {
			m.Codec = s.CodecName
			m.Resolution = fmt.Sprintf("%dx%d", s.Width, s.Height)
			m.FrameRate = s.RFrameRate
			if s.Tags.Timecode != "" {
				m.Timecode = s.Tags.Timecode
			}
			break
		}
	}
	if m.Timecode == "" && data.Format.Tags.Timecode != "" {
		m.Timecode = data.Format.Tags.Timecode
	}
	return m, nil
}

// ExifToolOutput matches JSON output from exiftool
type ExifToolOutput struct {
	SourceFile string `json:"SourceFile"`
	// Common fields
	MIMEType string `json:"MIMEType"`
	Duration string `json:"Duration"`
	// Video fields
	ImageWidth     int         `json:"ImageWidth"`
	ImageHeight    int         `json:"ImageHeight"`
	VideoFrameRate float64     `json:"VideoFrameRate"`
	FrameRate      interface{} `json:"FrameRate"` // Sometimes string, sometimes float
	CompressorID   string      `json:"CompressorID"`
	ProResCodec    string      `json:"CompressorName"` // ProRes often here

	// RED Specific
	CameraModelName string `json:"CameraModelName"`
	ClipName        string `json:"ClipName"`
	CameraID        string `json:"CameraID"`
	ReelNumber      string `json:"ReelNumber"`

	// Timecode
	TimeCode      string `json:"TimeCode"`
	StartTimecode string `json:"StartTimecode"`
}

var exiftoolPath string

func setupExifTool() (string, error) {
	if exiftoolPath != "" {
		return exiftoolPath, nil
	}

	tmpDir := filepath.Join(os.TempDir(), "loot_exiftool")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", err
	}

	// Extract exiftool script
	scriptPath := filepath.Join(tmpDir, "exiftool")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		data, err := exifToolFS.ReadFile("exiftool_dist/exiftool/exiftool")
		if err != nil {
			return "", fmt.Errorf("failed to read embedded exiftool: %w", err)
		}
		if err := os.WriteFile(scriptPath, data, 0755); err != nil {
			return "", fmt.Errorf("failed to write exiftool script: %w", err)
		}
	}

	// Extract lib directory (recursively is hard with just ReadFile, need WalkDir)
	libDir := filepath.Join(tmpDir, "lib")
	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		if err := os.MkdirAll(libDir, 0755); err != nil {
			return "", err
		}

		err := fsWalkDir(exifToolFS, "exiftool_dist/exiftool/lib", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel("exiftool_dist/exiftool/lib", path)
			destPath := filepath.Join(libDir, relPath)

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			data, err := exifToolFS.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(destPath, data, 0644)
		})
		if err != nil {
			return "", fmt.Errorf("failed to extract lib: %w", err)
		}
	}

	exiftoolPath = scriptPath
	return scriptPath, nil
}

func fsWalkDir(fs embed.FS, root string, fn func(path string, d os.DirEntry, err error) error) error {
	dirEntries, err := fs.ReadDir(root)
	if err != nil {
		return fn(root, nil, err)
	}
	for _, entry := range dirEntries {
		path := filepath.Join(root, entry.Name())
		if err := fn(path, entry, nil); err != nil {
			return err
		}
		if entry.IsDir() {
			if err := fsWalkDir(fs, path, fn); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractExifTool(path string) (*Metadata, error) {
	script, err := setupExifTool()
	if err != nil {
		return nil, err
	}

	// perl exiftool -j -n -CameraModelName -ImageWidth -ImageHeight -VideoFrameRate -Duration -TimeCode -CompressorID path
	cmd := exec.Command("perl",
		script,
		"-j", "-n",
		"-API", "LargeFileSupport=1",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var data []ExifToolOutput // Exiftool returns array
	if err := json.Unmarshal(output, &data); err != nil || len(data) == 0 {
		return nil, fmt.Errorf("exiftool parse error")
	}

	item := data[0]
	m := &Metadata{}

	// Format / Codec
	if strings.Contains(strings.ToLower(item.MIMEType), "red") {
		m.Codec = "REDCODE RAW"
	} else if item.ProResCodec != "" {
		m.Codec = item.ProResCodec
	} else if item.CompressorID != "" {
		m.Codec = item.CompressorID
	} else {
		m.Codec = filepath.Ext(path) // Fallback
	}

	// Resolution
	if item.ImageWidth > 0 && item.ImageHeight > 0 {
		m.Resolution = fmt.Sprintf("%dx%d", item.ImageWidth, item.ImageHeight)
	}

	// FrameRate
	if item.VideoFrameRate > 0 {
		m.FrameRate = fmt.Sprintf("%.3f", item.VideoFrameRate)
	} else if val, ok := item.FrameRate.(float64); ok && val > 0 {
		m.FrameRate = fmt.Sprintf("%.3f", val)
	}

	// Duration
	if item.Duration != "" {
		// Try parsing as seconds
		m.Duration = item.Duration + "s"
	}

	// Timecode
	if item.TimeCode != "" {
		m.Timecode = item.TimeCode
	} else if item.StartTimecode != "" {
		m.Timecode = item.StartTimecode
	}

	// Extra fields
	m.CameraID = item.CameraID
	m.ReelNumber = item.ReelNumber
	m.ClipName = item.ClipName

	return m, nil
}
