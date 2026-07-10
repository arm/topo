package upgrade

import (
	"path/filepath"
	"strings"
)

func GetUpgradeCommand(binPath string) (isManagedByUs bool, cmd string) {
	switch {
	case IsBinaryManagedByHomebrew(binPath):
		return false, "brew upgrade topo"
	default:
		return true, "topo upgrade"
	}
}

func IsBinaryManagedByUs(binPath string) bool {
	isManagedByUs, _ := GetUpgradeCommand(binPath)
	return isManagedByUs
}

func IsBinaryManagedByHomebrew(binPath string) bool {
	path := filepath.ToSlash(binPath)
	return strings.Contains(path, "Cellar/topo/")
}
