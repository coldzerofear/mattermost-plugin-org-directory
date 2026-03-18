package main

import "testing"

func TestIsRoleSufficient(t *testing.T) {
	tests := []struct {
		userRole     string
		requiredRole string
		want         bool
	}{
		// Same role
		{RoleMember, RoleMember, true},
		{RoleManager, RoleManager, true},
		{RoleAdmin, RoleAdmin, true},

		// Higher role satisfies lower requirement
		{RoleManager, RoleMember, true},
		{RoleAdmin, RoleMember, true},
		{RoleAdmin, RoleManager, true},

		// Lower role does not satisfy higher requirement
		{RoleMember, RoleManager, false},
		{RoleMember, RoleAdmin, false},
		{RoleManager, RoleAdmin, false},

		// Unknown roles have weight 0
		{"unknown", RoleMember, false},
		{"", RoleMember, false},

		// Empty required role (weight 0) is always satisfied
		{RoleMember, "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got := isRoleSufficient(tt.userRole, tt.requiredRole)
		if got != tt.want {
			t.Errorf("isRoleSufficient(%q, %q) = %v, want %v",
				tt.userRole, tt.requiredRole, got, tt.want)
		}
	}
}
