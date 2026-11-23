package term

import (
	"context"
	"sync"

	"github.com/chzyer/readline"
)

type promptActions struct {
	editor  *readline.Instance
	actions []PromptAction
	ctx     context.Context
	cancel  context.CancelFunc
	workers sync.WaitGroup
	mu      sync.Mutex
	running map[rune]bool
	message string
	inputMu *sync.Mutex
}

func newPromptActions(
	editor *readline.Instance, actions []PromptAction, inputMu *sync.Mutex,
) *promptActions {
	ctx, cancel := context.WithCancel(context.Background())
	return &promptActions{
		editor: editor, actions: actions, ctx: ctx, cancel: cancel,
		running: make(map[rune]bool), inputMu: inputMu,
	}
}

func (a *promptActions) invoke(key rune, input string) bool {
	for _, action := range a.actions {
		if action.Key != key {
			continue
		}
		a.mu.Lock()
		if a.ctx.Err() != nil || a.running[key] {
			a.mu.Unlock()
			return true
		}
		a.running[key] = true
		a.message = "Action running; you can still enter a value"
		a.mu.Unlock()
		a.workers.Add(1)
		go a.run(action, input)
		return true
	}
	return false
}

func (a *promptActions) run(action PromptAction, input string) {
	defer a.workers.Done()
	value, err := action.Run(a.ctx, input)
	a.mu.Lock()
	a.running[action.Key] = false
	a.message = ""
	if err != nil {
		a.message = "Action failed: " + err.Error()
	}
	a.mu.Unlock()
	a.inputMu.Lock()
	defer a.inputMu.Unlock()
	if a.ctx.Err() != nil {
		return
	}
	if err == nil {
		a.editor.Operation.SetBuffer(value)
	} else {
		a.editor.Refresh()
	}
}

func (a *promptActions) status() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.message
}

func (a *promptActions) close() {
	a.inputMu.Lock()
	a.cancel()
	a.inputMu.Unlock()
	a.workers.Wait()
}
