package model

import "testing"

func TestOrgMemberPreSave(t *testing.T) {
	t.Run("sets default role when empty", func(t *testing.T) {
		m := &OrgMember{}
		m.PreSave()
		if m.Role != "member" {
			t.Errorf("Role = %q, want %q", m.Role, "member")
		}
	})

	t.Run("sets default source when empty", func(t *testing.T) {
		m := &OrgMember{}
		m.PreSave()
		if m.Source != "local" {
			t.Errorf("Source = %q, want %q", m.Source, "local")
		}
	})

	t.Run("preserves existing role", func(t *testing.T) {
		m := &OrgMember{Role: "admin"}
		m.PreSave()
		if m.Role != "admin" {
			t.Errorf("Role = %q, want admin", m.Role)
		}
	})

	t.Run("preserves manager role", func(t *testing.T) {
		m := &OrgMember{Role: "manager"}
		m.PreSave()
		if m.Role != "manager" {
			t.Errorf("Role = %q, want manager", m.Role)
		}
	})

	t.Run("preserves existing source", func(t *testing.T) {
		m := &OrgMember{Source: "ldap"}
		m.PreSave()
		if m.Source != "ldap" {
			t.Errorf("Source = %q, want ldap", m.Source)
		}
	})

	t.Run("idempotent on repeated calls", func(t *testing.T) {
		m := &OrgMember{}
		m.PreSave()
		m.PreSave()
		if m.Role != "member" || m.Source != "local" {
			t.Error("repeated PreSave should produce same result")
		}
	})
}
