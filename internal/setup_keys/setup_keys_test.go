package setup_keys_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/arm/topo/internal/setup_keys"
	"github.com/arm/topo/internal/ssh"
	"github.com/stretchr/testify/require"
)

func TestNewKeyCreationAndPlacementOnTargetDryRun(t *testing.T) {
	tests := []struct {
		name         string
		inputKeyPath string
		wantKeyPath  string
	}{
		{
			name:         "default key path",
			inputKeyPath: "",
			wantKeyPath:  filepath.Join(".ssh", "id_ed25519_topo_user_example.com"),
		},
		{
			name:         "custom key path",
			inputKeyPath: filepath.Join("custom_keys", "id_ed25519_custom"),
			wantKeyPath:  filepath.Join("custom_keys", "id_ed25519_custom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			if runtime.GOOS == "windows" {
				t.Setenv("USERPROFILE", tmp)
				if vol := filepath.VolumeName(tmp); vol != "" {
					t.Setenv("HOMEDRIVE", vol)
					t.Setenv("HOMEPATH", strings.TrimPrefix(tmp, vol))
				}
			}

			inputKeyPath := tt.inputKeyPath
			if inputKeyPath != "" {
				inputKeyPath = filepath.Join(tmp, inputKeyPath)
			}

			var buf bytes.Buffer
			targetFileName := setup_keys.SanitizeTarget("user@example.com")
			keyPath, err := setup_keys.CreateKeyPair("user@example.com", targetFileName, inputKeyPath, &buf, true)
			require.NoError(t, err)

			wantKeyPath := filepath.Join(tmp, tt.wantKeyPath)
			require.Equal(t, wantKeyPath, keyPath, "NewKeyPairCreation should return expected key path")

			wantKeygen := "-t ed25519 -f " + keyPath + " -C user@example.com"
			got := buf.String()
			require.Contains(t, got, "ssh-keygen", "DryRun output should include keygen command")
			require.Contains(t, got, wantKeygen, "DryRun output should include keygen arguments")

			transferErr := setup_keys.TransferPubKey("user@example.com", keyPath, &buf, true)
			require.NoError(t, transferErr)
			got = buf.String()
			require.Contains(t, got, "ssh user@example.com", "DryRun output should include ssh command")
			require.Contains(t, got, keyPath+".pub", "DryRun output should include public key path")
		})
	}
}

func TestKeyCreationAndPlacementOnTarget(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "custom_keys", "id_ed25519_custom_run")
	logFile := filepath.Join(t.TempDir(), "commands.log")

	testLogFile := logFile
	origExec := setup_keys.ExecCommand
	setup_keys.ExecCommand = func(command string, args ...string) *exec.Cmd {
		return mockExecCommand(testLogFile, command, args...)
	}
	t.Cleanup(func() { setup_keys.ExecCommand = origExec })

	var buf bytes.Buffer
	targetFileName := setup_keys.SanitizeTarget("user@example.com")
	keyPath, keyPairErr := setup_keys.CreateKeyPair("user@example.com", targetFileName, keyPath, &buf, false)
	require.NoError(t, keyPairErr)
	require.Contains(t, buf.String(), "ssh-keygen invoked", "Run output should include fake ssh-keygen output")

	logData, err := os.ReadFile(logFile)
	require.NoError(t, err)
	log := string(logData)
	require.Contains(t, log, fmt.Sprintf("ssh-keygen -t ed25519 -f %s -C user@example.com", keyPath))

	origSSHExec := setup_keys.SSHExec
	var gotCommand string
	var gotStdin []byte
	setup_keys.SSHExec = func(_ ssh.Host, command string, stdin []byte, _ ...string) (string, error) {
		gotCommand = command
		gotStdin = stdin
		return "", nil
	}
	t.Cleanup(func() { setup_keys.SSHExec = origSSHExec })

	transferErr := setup_keys.TransferPubKey("user@example.com", keyPath, &buf, false)
	require.NoError(t, transferErr)
	pubKey, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	wantCommand := ssh.ShellCommand("mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys")
	require.Equal(t, wantCommand, gotCommand)
	require.Equal(t, pubKey, gotStdin)
}

func TestSanitizeTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user@example.com", "user_example.com"},
		{"Example-Host", "Example-Host"},
		{"spaces and/tabs", "spaces_and_tabs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := setup_keys.SanitizeTarget(tt.input); got != tt.want {
				t.Fatalf("sanitizeTarget(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func mockExecCommand(logFile, command string, args ...string) *exec.Cmd {
	cs := slices.Concat([]string{"-test.run=TestHelperProcess", "--", command}, args)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"TOPO_TEST_LOG="+logFile,
	)
	return cmd
}

func TestHelperProcess(t *testing.T) {
	// Only run if invoked as a from another test. This is not a standalone test, so exit otherwise!
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	sep := slices.Index(args, "--")
	if sep == -1 || sep+1 >= len(args) {
		os.Exit(2)
	}

	command := args[sep+1]
	cmdArgs := args[sep+2:]
	fmt.Printf("%s invoked\n", command)

	logFile := os.Getenv("TOPO_TEST_LOG")
	if logFile == "" || appendLine(logFile, strings.Join(append([]string{command}, cmdArgs...), " ")) != nil {
		os.Exit(2)
	}

	if command == "ssh-keygen" {
		key := ""
		for i := 0; i < len(cmdArgs); i++ {
			if cmdArgs[i] == "-f" && i+1 < len(cmdArgs) {
				key = cmdArgs[i+1]
				break
			}
		}
		if key != "" {
			if err := os.WriteFile(key, []byte("FAKEKEY"), 0o600); err != nil {
				os.Exit(2)
			}
			if err := os.WriteFile(key+".pub", []byte("FAKEPUB"), 0o600); err != nil {
				os.Exit(2)
			}
		}
	}

	os.Exit(0)
}

func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintln(f, line)
	return err
}
