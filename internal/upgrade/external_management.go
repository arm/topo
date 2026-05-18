package upgrade

import (
	"bytes"
	"path/filepath"
)

func IsTopoBinaryManagedExternally(binPath string) bool {
	return isHomebrewManagedBinary(binPath)
}

func isHomebrewManagedBinary(binPath string) bool {
	parts := splitPath(binPath)
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "Cellar" && parts[i+1] == "topo" {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	volume := filepath.VolumeName(path)
	path = path[len(volume):]
	parts := []string{}
	for part := range bytes.SplitSeq([]byte(filepath.ToSlash(path)), []byte{'/'}) {
		if len(part) > 0 {
			parts = append(parts, string(part))
		}
	}
	return parts
}
