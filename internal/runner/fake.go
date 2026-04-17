package runner

import (
	"context"
	"fmt"
	"slices"
)

// FakeResult holds the canned response for a command.
type FakeResult struct {
	Output string
	Err    error
}

// Fake is a test double that maps commands to canned results.
// It satisfies the Runner interface without coupling tests to method signatures.
type Fake struct {
	Binaries        []string
	BinaryExistsErr map[string]error
	Commands        map[string]FakeResult
}

func (f *Fake) Run(ctx context.Context, command string) (string, error) {
	res, ok := f.Commands[command]
	if !ok {
		return "", fmt.Errorf("unexpected command: %s", command)
	}
	return res.Output, res.Err
}

func (f *Fake) RunWithStdin(ctx context.Context, command string, stdin []byte) (string, error) {
	return f.Run(ctx, command)
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
