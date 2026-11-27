package runner

import (
	"context"
	"fmt"
	"slices"
)

// FakeResult holds the canned response for a command.
type FakeResult struct {
	Output string
	Stderr string
	Err    error
}

// Fake maps quoted command lines and raw shell scripts to canned results.
type Fake struct {
	Binaries        []string
	BinaryExistsErr map[string]error
	Commands        map[string]FakeResult
	Scripts         map[string]FakeResult
}

func (f *Fake) Run(_ context.Context, program string, args ...string) (string, string, error) {
	return fakeResult(f.Commands, commandLine(program, args...))
}

func (f *Fake) RunShell(_ context.Context, script string) (string, string, error) {
	return fakeResult(f.Scripts, script)
}

func (f *Fake) RunWithStdin(ctx context.Context, _ []byte, program string, args ...string) (string, string, error) {
	return f.Run(ctx, program, args...)
}

func (f *Fake) BinaryExists(_ context.Context, bin string) error {
	if f.BinaryExistsErr != nil {
		if err, ok := f.BinaryExistsErr[bin]; ok {
			return err
		}
	}
	if slices.Contains(f.Binaries, bin) {
		return nil
	}
	return fmt.Errorf("%q not found in $PATH", bin)
}

func fakeResult(results map[string]FakeResult, command string) (string, string, error) {
	res, ok := results[command]
	if !ok {
		return "", "", fmt.Errorf("unexpected command: %s", command)
	}
	return res.Output, res.Stderr, res.Err
}
