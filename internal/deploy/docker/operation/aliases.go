package operation

import (
	base "github.com/arm/topo/internal/operation"
)

type (
	Sequence  = base.Sequence
	Operation = base.Operation
)

var (
	NewSequence      = base.NewSequence
	NewConditional   = base.NewConditional
	SetupExitCleanup = base.SetupExitCleanup
)
