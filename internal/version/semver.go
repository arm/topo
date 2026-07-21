package version

import (
	"regexp"
	"strconv"
)

var semverRe = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)

func parseSemver(version string) (major, minor, patch int) {
	matches := semverRe.FindStringSubmatch(version)
	if matches == nil {
		return 0, 0, 0
	}

	major, _ = strconv.Atoi(matches[1])
	minor, _ = strconv.Atoi(matches[2])
	patch, _ = strconv.Atoi(matches[3])
	return major, minor, patch
}

func compareSemver(a, b string) int {
	majorA, minorA, patchA := parseSemver(a)
	majorB, minorB, patchB := parseSemver(b)

	if majorA != majorB {
		return majorA - majorB
	}
	if minorA != minorB {
		return minorA - minorB
	}
	return patchA - patchB
}

func IsAtLeastVersion(current, minimum string) bool {
	return compareSemver(current, minimum) >= 0
}
