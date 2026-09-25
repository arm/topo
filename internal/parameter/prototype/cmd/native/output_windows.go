package main

import (
	"os"

	"golang.org/x/sys/windows"
)

func prepareOutput() (func() error, error) {
	handle := windows.Handle(os.Stderr.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return nil, err
	}
	return func() error { return windows.SetConsoleMode(handle, mode) }, nil
}
