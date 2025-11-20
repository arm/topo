package source

import (
	"fmt"
	"os"
	"os/exec"
)

type Git struct {
	URL string
	Ref string
}

func (g Git) CopyTo(destDir string) error {
	if _, err := os.Stat(destDir); err == nil {
		return DestDirExistsError{Dir: destDir}
	}

	args := []string{"clone", "--depth", "1"}
	if g.Ref != "" {
		args = append(args, "--branch", g.Ref)
	}
	args = append(args, g.URL, destDir)
	cmd := exec.Command("git", args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g Git) String() string {
	if g.Ref != "" {
		return fmt.Sprintf("git:%s#%s", g.URL, g.Ref)
	}
	return fmt.Sprintf("git:%s", g.URL)
}
