//go:build !windows

package main

func prepareOutput() (func() error, error) {
	return func() error { return nil }, nil
}
