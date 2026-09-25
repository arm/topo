package demo

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/arm/topo/internal/output/term"
	terminal "golang.org/x/term"
)

var ErrCanceled = errors.New("canceled; no updates saved")

func Options() (string, error) {
	layout := flag.String("layout", "sequential", "sequential or form")
	flag.Parse()
	if flag.NArg() != 0 || (*layout != "sequential" && *layout != "form") {
		return "", errors.New("use -layout sequential or -layout form, without positional arguments")
	}
	if !terminal.IsTerminal(int(os.Stdin.Fd())) || !terminal.IsTerminal(int(os.Stderr.Fd())) {
		return "", errors.New("this prototype requires a terminal on stdin and stderr")
	}
	if os.Getenv("TERM") == "dumb" {
		return "", errors.New("this prototype requires cursor addressing; no plain-text fallback is implemented")
	}
	return *layout, nil
}

func Banner() error {
	return term.PrintFirstHeader(os.Stderr, "Configure project parameters")
}

func Transcript(parameter Parameter, answer Answer) string {
	return " " + parameter.Name + ": " + Preview(parameter, answer)
}

func Report(parameters []Parameter, answers []Answer) error {
	type decision struct {
		Name    string   `json:"name"`
		Current *Current `json:"current,omitempty"`
		Action  string   `json:"action"`
	}
	result := struct {
		Decisions []decision        `json:"decisions"`
		Updates   map[string]string `json:"updates"`
	}{Updates: make(map[string]string)}
	for i, parameter := range parameters {
		value, write, err := Resolve(parameter, answers[i])
		if err != nil {
			return fmt.Errorf("%s: %w", parameter.Name, err)
		}
		if write {
			result.Updates[parameter.Name] = value
		}
		result.Decisions = append(result.Decisions, decision{parameter.Name, parameter.Current, Preview(parameter, answers[i])})
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
