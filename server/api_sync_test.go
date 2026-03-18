package main

import (
	"testing"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

func TestTopologicalSortNodes(t *testing.T) {
	t.Run("nil input returns empty", func(t *testing.T) {
		got := topologicalSortNodes(nil)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d nodes", len(got))
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		got := topologicalSortNodes([]pluginmodel.SyncNodePayload{})
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d nodes", len(got))
		}
	})

	t.Run("single root node preserved", func(t *testing.T) {
		nodes := []pluginmodel.SyncNodePayload{
			{ExternalID: "root", Name: "Root"},
		}
		got := topologicalSortNodes(nodes)
		if len(got) != 1 || got[0].ExternalID != "root" {
			t.Errorf("expected single root node, got %v", got)
		}
	})

	t.Run("parent listed after child — parent comes first in output", func(t *testing.T) {
		nodes := []pluginmodel.SyncNodePayload{
			{ExternalID: "child", Name: "Child", ParentExternalID: "parent"},
			{ExternalID: "parent", Name: "Parent"},
		}
		got := topologicalSortNodes(nodes)
		if len(got) != 2 {
			t.Fatalf("expected 2 nodes, got %d", len(got))
		}
		if got[0].ExternalID != "parent" {
			t.Errorf("expected parent first, got %q", got[0].ExternalID)
		}
		if got[1].ExternalID != "child" {
			t.Errorf("expected child second, got %q", got[1].ExternalID)
		}
	})

	t.Run("three-level hierarchy sorted correctly", func(t *testing.T) {
		// Input: grandchild, child, root — all reversed
		nodes := []pluginmodel.SyncNodePayload{
			{ExternalID: "gc", Name: "GrandChild", ParentExternalID: "child"},
			{ExternalID: "child", Name: "Child", ParentExternalID: "root"},
			{ExternalID: "root", Name: "Root"},
		}
		got := topologicalSortNodes(nodes)
		if len(got) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(got))
		}
		order := make(map[string]int, 3)
		for i, n := range got {
			order[n.ExternalID] = i
		}
		if order["root"] >= order["child"] {
			t.Errorf("root (pos %d) should come before child (pos %d)", order["root"], order["child"])
		}
		if order["child"] >= order["gc"] {
			t.Errorf("child (pos %d) should come before grandchild (pos %d)", order["child"], order["gc"])
		}
	})

	t.Run("multiple root nodes with children", func(t *testing.T) {
		nodes := []pluginmodel.SyncNodePayload{
			{ExternalID: "child_a", Name: "ChildA", ParentExternalID: "a"},
			{ExternalID: "b", Name: "B"},
			{ExternalID: "a", Name: "A"},
		}
		got := topologicalSortNodes(nodes)
		if len(got) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(got))
		}
		order := make(map[string]int, 3)
		for i, n := range got {
			order[n.ExternalID] = i
		}
		if order["a"] >= order["child_a"] {
			t.Errorf("a (pos %d) should come before child_a (pos %d)", order["a"], order["child_a"])
		}
	})

	t.Run("all nodes returned even if count matches input", func(t *testing.T) {
		nodes := []pluginmodel.SyncNodePayload{
			{ExternalID: "x", Name: "X"},
			{ExternalID: "y", Name: "Y"},
			{ExternalID: "z", Name: "Z"},
		}
		got := topologicalSortNodes(nodes)
		if len(got) != 3 {
			t.Errorf("expected 3 nodes, got %d", len(got))
		}
	})
}
