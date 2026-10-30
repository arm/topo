package fact

import (
	"context"
	"fmt"
	"strconv"

	"github.com/arm/topo/internal/runner"
)

var Providers = map[string]Provider{
	"topo.memory":     MemoryProvider{},
	"topo.remoteproc": RemoteprocProvider{},
}

type Provider interface {
	Facts(ctx context.Context, targetRunner runner.CommandRunner) (Report, error)
}

type Report struct {
	Groups []Group
}

func (r Report) Find(groupID, key string) (Fact, bool) {
	for _, group := range r.Groups {
		if group.ID != groupID {
			continue
		}
		return group.Find(key)
	}
	return Fact{}, false
}

func (r Report) Validate() error {
	groups := make(map[string]bool, len(r.Groups))
	for _, group := range r.Groups {
		if group.ID == "" || groups[group.ID] {
			return fmt.Errorf("empty or duplicate fact group: %q", group.ID)
		}
		groups[group.ID] = true
		keys := make(map[string]bool, len(group.Facts))
		for _, fact := range group.Facts {
			if fact.Key == "" || keys[fact.Key] {
				return fmt.Errorf("invalid or duplicate fact: %q in group %q", fact.Key, group.ID)
			}
			keys[fact.Key] = true
			if err := fact.Value.Validate(); err != nil {
				return fmt.Errorf("invalid fact %q in group %q: %w", fact.Key, group.ID, err)
			}
		}
	}
	return nil
}

type Group struct {
	ID    string
	Facts []Fact
}

func (g Group) Find(key string) (Fact, bool) {
	for _, fact := range g.Facts {
		if fact.Key == key {
			return fact, true
		}
	}
	return Fact{}, false
}

type Fact struct {
	Key         string
	Value       Value
	Label       string
	Description string
}

type Value struct {
	Type    string
	Boolean bool
	Integer int64
	Text    string
}

func (v Value) String() string {
	switch v.Type {
	case "boolean":
		return strconv.FormatBool(v.Boolean)
	case "integer":
		return strconv.FormatInt(v.Integer, 10)
	case "string":
		return v.Text
	}

	panic(fmt.Sprintf("unsupported value type: %q", v.Type))
}

func (v Value) Validate() error {
	switch v.Type {
	case "boolean":
		if v.Integer == 0 && v.Text == "" {
			return nil
		}
	case "integer":
		if !v.Boolean && v.Text == "" {
			return nil
		}
	case "string":
		if !v.Boolean && v.Integer == 0 {
			return nil
		}
	}
	return fmt.Errorf("invalid fact value type or fields: %q", v.Type)
}
