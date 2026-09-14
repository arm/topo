package term

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type SectionPrinter struct {
	*CommandOutput
	pending string
}

type CommandOutput struct {
	mutex   sync.Mutex
	output  io.Writer
	started bool
}

func NewCommandOutput(output io.Writer) *CommandOutput {
	return &CommandOutput{output: output}
}

func NewSectionPrinter(output *CommandOutput, header string) *SectionPrinter {
	return &SectionPrinter{CommandOutput: output, pending: header}
}

func (p *SectionPrinter) Write(data []byte) (int, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(data) > 0 && p.pending != "" {
		if err := p.printHeader(p.pending); err != nil {
			return 0, err
		}
		p.pending = ""
	}
	return p.output.Write(data)
}

func (p *SectionPrinter) SubprocessOutput() (io.Writer, error) {
	if !IsTTY(p.output) {
		return p, nil
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if err := p.printHeader(p.pending); err != nil {
		return nil, err
	}
	p.pending = ""
	return p.output, nil
}

func (p *CommandOutput) Finish() error {
	_, err := fmt.Fprintln(p.output)
	return err
}

func (p *CommandOutput) printHeader(description string) error {
	if description == "" {
		return nil
	}
	if p.started {
		_, err := fmt.Fprintf(p.output, "\n%s\n", Header(description, IsTTY(p.output)))
		return err
	}
	_, err := fmt.Fprintln(p.output, Header(description, IsTTY(p.output)))
	if err == nil {
		p.started = true
	}
	return err
}

func Header(description string, isTTY bool) string {
	if description == "" {
		return ""
	}

	const totalWidth = 60
	prefix := "┌─ "
	suffix := " "

	descriptionWidth := len(description)
	barWidth := max(totalWidth-len(prefix)-descriptionWidth-len(suffix), 0)
	bar := suffix + strings.Repeat("─", barWidth)
	if !isTTY {
		return prefix + description + bar
	}
	return Color(Dim, prefix) + description + Color(Dim, bar)
}
