package offload

import (
	"os"
	"path/filepath"
	"strings"
)

// Volume represents a mounted storage device
type Volume struct {
	Name string
	Path string
}

// GetVolumes returns a list of mounted volumes in /Volumes
func GetVolumes() ([]Volume, error) {
	var volumes []Volume

	// Add root volume
	volumes = append(volumes, Volume{Name: "Macintosh HD", Path: "/"})

	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return volumes, err
	}

	for _, entry := range entries {
		// Skip hidden files/dirs
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		volumes = append(volumes, Volume{
			Name: entry.Name(),
			Path: filepath.Join("/Volumes", entry.Name()),
		})
	}

	return volumes, nil
}
