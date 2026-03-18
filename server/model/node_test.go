package model

import (
	"strings"
	"testing"
)

func TestOrgNodeIsValid(t *testing.T) {
	tests := []struct {
		name  string
		node  OrgNode
		valid bool
	}{
		{"empty name", OrgNode{Name: ""}, false},
		{"valid short name", OrgNode{Name: "Engineering"}, true},
		{"single char name", OrgNode{Name: "A"}, true},
		{"name at max length", OrgNode{Name: strings.Repeat("x", 256)}, true},
		{"name over max length", OrgNode{Name: strings.Repeat("x", 257)}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestOrgNodePreSave(t *testing.T) {
	t.Run("sets default metadata when empty", func(t *testing.T) {
		n := &OrgNode{}
		n.PreSave()
		if n.Metadata != "{}" {
			t.Errorf("Metadata = %q, want %q", n.Metadata, "{}")
		}
	})

	t.Run("sets default source when empty", func(t *testing.T) {
		n := &OrgNode{}
		n.PreSave()
		if n.Source != "local" {
			t.Errorf("Source = %q, want %q", n.Source, "local")
		}
	})

	t.Run("preserves existing metadata", func(t *testing.T) {
		n := &OrgNode{Metadata: `{"department":"eng"}`}
		n.PreSave()
		if n.Metadata != `{"department":"eng"}` {
			t.Errorf("Metadata = %q, want original value", n.Metadata)
		}
	})

	t.Run("preserves existing source", func(t *testing.T) {
		n := &OrgNode{Source: "hr_system"}
		n.PreSave()
		if n.Source != "hr_system" {
			t.Errorf("Source = %q, want hr_system", n.Source)
		}
	})

	t.Run("idempotent on repeated calls", func(t *testing.T) {
		n := &OrgNode{}
		n.PreSave()
		n.PreSave()
		if n.Metadata != "{}" || n.Source != "local" {
			t.Error("repeated PreSave should produce same result")
		}
	})
}
