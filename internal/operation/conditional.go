package operation

import "io"

type Predicate interface {
	Eval() bool
}

type Conditional struct {
	condition Predicate
	ifTrue    Operation
	ifFalse   Operation
}

func NewConditional(condition Predicate, ifTrue Operation, ifFalse Operation) Operation {
	return &Conditional{
		condition: condition,
		ifTrue:    ifTrue,
		ifFalse:   ifFalse,
	}
}

func (c *Conditional) Run(cmdOutput io.Writer) error {
	if c.condition.Eval() {
		return c.ifTrue.Run(cmdOutput)
	}
	return c.ifFalse.Run(cmdOutput)
}

func (c *Conditional) DryRun(output io.Writer) error {
	if c.condition.Eval() {
		return c.ifTrue.DryRun(output)
	}
	return c.ifFalse.DryRun(output)
}

func (c *Conditional) Description() string {
	if c.condition.Eval() {
		return c.ifTrue.Description()
	}
	return c.ifFalse.Description()
}
