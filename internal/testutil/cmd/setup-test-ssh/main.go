package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const testSSHConfig = `Host topo-test-*
    HostName localhost
    UserKnownHostsFile ~/.ssh/topo-test-known-hosts/%n

Host *
`

func main() {
	if err := setupSSH(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupSSH() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(filepath.Join(sshDir, "topo-test-known-hosts"), 0o700); err != nil {
		return err
	}
	path := filepath.Join(sshDir, "config")
	if _, err := os.Lstat(path); err == nil {
		path, err = filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if bytes.Contains(content, []byte(testSSHConfig)) {
		return nil
	}
	return replaceConfig(path, append([]byte(testSSHConfig), content...))
}

func replaceConfig(path string, content []byte) (err error) {
	mode := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".topo-test-config-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(file.Name()); !os.IsNotExist(removeErr) { // #nosec G703 -- removes our temporary file.
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(content)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	if err := os.Chmod(file.Name(), mode); err != nil { // #nosec G703 -- preserves permissions on our temporary file.
		return err
	}
	return os.Rename(file.Name(), path) // #nosec G703 -- installs our temporary file at the resolved config path.
}
