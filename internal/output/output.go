package output

// Output format for the commands
type Format int

const (
	// PlainFormat renders human-readable plain text
	PlainFormat Format = iota
	// JSONFormat renders machine-readable JSON
	JSONFormat
)
