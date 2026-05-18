package upgrade

import (
	"path/filepath"
	"strings"
)

func IsTopoBinaryManagedExternally(binPath string) bool {
	return isHomebrewManagedBinary(binPath)
}

func isHomebrewManagedBinary(binPath string) bool {
	path := filepath.ToSlash(binPath)
	return strings.Contains(path, "Cellar/topo/")
}
