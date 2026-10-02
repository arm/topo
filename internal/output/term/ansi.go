package term

import "fmt"

const (
	MoveToRowStart = "\r"
	ClearLine      = "\x1b[J"
	NewLine        = "\r\n"
)

func MoveUp(rows int) string {
	return fmt.Sprintf("\x1b[%dA", rows)
}
