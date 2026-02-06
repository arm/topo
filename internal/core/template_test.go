package core

import (
	"encoding/json"
	"testing"
)

func TestListTemplatesOperation(t *testing.T) {
	out := captureOutput(func() {
		if err := ListTemplates(); err != nil {
			t.Fatalf("ListTemplates error: %v", err)
		}
	})
	var arr []Template
	if err := json.Unmarshal([]byte(out), &arr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(arr) == 0 {
		t.Fatal("expected templates")
	}
}
