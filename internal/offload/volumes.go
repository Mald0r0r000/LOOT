package offload

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Volume represents a mounted storage device
type Volume struct {
	Name  string
	Path  string
	Total uint64
	Free  uint64
	Used  uint64
}

// GetVolumes returns a list of mounted volumes in /Volumes
func GetVolumes() ([]Volume, error) {
	var volumes []Volume

	// Add root volume
	root := Volume{Name: "Macintosh HD", Path: "/"}
	fillDiskUsage(&root)
	volumes = append(volumes, root)

	// Add current directory (useful for testing/mock volumes)
	if cwd, err := os.Getwd(); err == nil {
		cwdVol := Volume{Name: "Current Directory (.)", Path: cwd}
		fillDiskUsage(&cwdVol)
		volumes = append(volumes, cwdVol)
	}

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return volumes, err // Return what we have
	}

	for _, entry := range entries {
		// Skip hidden files/dirs
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		v := Volume{
			Name: entry.Name(),
			Path: filepath.Join("/Volumes", entry.Name()),
		}
		fillDiskUsage(&v)
		volumes = append(volumes, v)
	}

	return volumes, nil
}

func fillDiskUsage(v *Volume) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(v.Path, &stat); err == nil {
		// Blocks * BlockSize
		v.Total = uint64(stat.Blocks) * uint64(stat.Bsize)
		v.Free = uint64(stat.Bavail) * uint64(stat.Bsize) // Bavail is for unprivileged users
		v.Used = v.Total - v.Free
	}
}

// FormatBytes converts bytes to human readable string
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
