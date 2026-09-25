package main

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"
)

type inputEvent struct {
	key  string
	text string
}

type inputRead struct {
	data []byte
	err  error
}

func readInput(reader io.Reader, reads chan<- inputRead, done <-chan struct{}) {
	for {
		buffer := make([]byte, 256)
		n, err := reader.Read(buffer)
		select {
		case reads <- inputRead{buffer[:n], err}:
		case <-done:
			return
		}
		if err != nil {
			return
		}
	}
}

// decodeInput consumes one event. A zero length asks the caller to await more bytes.
func decodeInput(data []byte, expired bool) (inputEvent, int) {
	if len(data) == 0 {
		return inputEvent{}, 0
	}
	if data[0] == 27 {
		return decodeEscape(data, expired)
	}
	keys := map[byte]string{
		3: "cancel", 4: "cancel", 9: "next", 10: "next", 13: "next",
		8: "backspace", 127: "backspace", 1: "home", 5: "end",
		21: "clear",
	}
	if key, ok := keys[data[0]]; ok {
		return inputEvent{key: key}, 1
	}
	if !utf8.FullRune(data) {
		return inputEvent{}, 0
	}
	r, size := utf8.DecodeRune(data)
	if r == utf8.RuneError && size == 1 {
		return inputEvent{key: "invalid"}, size
	}
	if r < 32 {
		return inputEvent{key: "ignored"}, size
	}
	return inputEvent{key: "text", text: string(r)}, size
}

func decodeEscape(data []byte, expired bool) (inputEvent, int) {
	const pasteStart, pasteEnd = "\x1b[200~", "\x1b[201~"
	if bytes.HasPrefix(data, []byte(pasteStart)) {
		end := bytes.Index(data[len(pasteStart):], []byte(pasteEnd))
		if end < 0 {
			return inputEvent{}, 0
		}
		end += len(pasteStart)
		return inputEvent{key: "text", text: string(data[len(pasteStart):end])}, end + len(pasteEnd)
	}
	sequences := map[string]string{
		"\x1b[D": "left", "\x1b[C": "right", "\x1b[Z": "previous",
		"\x1b[H": "home", "\x1b[F": "end", "\x1b[3~": "delete",
		"\x1bOH": "home", "\x1bOF": "end", "\x1b[1~": "home", "\x1b[4~": "end",
	}
	for sequence, key := range sequences {
		if bytes.HasPrefix(data, []byte(sequence)) {
			return inputEvent{key: key}, len(sequence)
		}
	}
	if len(data) > 1 && (data[1] == '[' || data[1] == 'O') {
		for i := 2; i < len(data); i++ {
			if data[i] >= 0x40 && data[i] <= 0x7e {
				return inputEvent{key: "ignored"}, i + 1
			}
		}
	}
	if !expired && (len(data) == 1 || strings.HasPrefix(pasteStart, string(data)) || len(data) < 16) {
		return inputEvent{}, 0
	}
	return inputEvent{key: "ignored"}, 1
}
