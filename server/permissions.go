package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const (
	RoleMember  = "member"
	RoleManager = "manager"
	RoleAdmin   = "admin"
)

// checkAuth is a middleware that validates the Mattermost session token.
func (p *Plugin) checkAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-Id")
		if userID == "" {
			http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// checkAdmin is a middleware that requires system administrator role.
func (p *Plugin) checkAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-Id")
		if !p.isSystemAdmin(userID) {
			http.Error(w, `{"error":"requires system admin"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// checkSyncToken validates sync API access.
//
// Rules:
// 1. If SyncAPIToken is configured, all sync routes require the configured Bearer token.
// 2. If SyncAPIToken is not configured:
//   - Any authenticated Mattermost user may call safe sync GET query routes.
//   - Only system administrators may call write routes and sensitive GET routes.
func (p *Plugin) checkSyncToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		config := p.getConfiguration()
		if strings.TrimSpace(config.SyncAPIToken) != "" {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token == "" {
				http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
				return
			}
			if subtle.ConstantTimeCompare([]byte(token), []byte(config.SyncAPIToken)) != 1 {
				http.Error(w, `{"error":"invalid sync token"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		userID := strings.TrimSpace(r.Header.Get("Mattermost-User-Id"))
		if userID == "" {
			http.Error(w, `{"error":"not authenticated"}`, http.StatusUnauthorized)
			return
		}

		if p.canAccessSyncWithoutToken(userID, r) {
			next.ServeHTTP(w, r)
			return
		}

		http.Error(w, `{"error":"requires system admin"}`, http.StatusForbidden)
	})
}

func (p *Plugin) canAccessSyncWithoutToken(userID string, r *http.Request) bool {
	if p.isSystemAdmin(userID) {
		return true
	}

	if r.Method != http.MethodGet {
		return false
	}

	path := r.URL.Path
	switch {
	case path == "/api/v1/sync/nodes":
		return true
	case strings.HasPrefix(path, "/api/v1/sync/nodes/"):
		return true
	case strings.HasPrefix(path, "/api/v1/sync/users/") && strings.HasSuffix(path, "/nodes"):
		return true
	default:
		return false
	}
}

// checkNodePermission verifies that a user has at least the required role for a node
// (or one of its ancestors).
func (p *Plugin) checkNodePermission(userID, nodeID string, requiredRole string) bool {
	if p.isSystemAdmin(userID) {
		return true
	}

	node, err := p.store.GetNode(nodeID)
	if err != nil {
		return false
	}

	// Walk from the current node up to the root, checking ancestry roles
	pathParts := strings.Split(strings.TrimPrefix(node.Path, "/"), "/")
	for i := len(pathParts); i > 0; i-- {
		ancestorID := pathParts[i-1]
		if ancestorID == "" {
			continue
		}
		member, err := p.store.GetMemberRole(ancestorID, userID)
		if err == nil && isRoleSufficient(member.Role, requiredRole) {
			return true
		}
	}
	return false
}

func isRoleSufficient(userRole, requiredRole string) bool {
	roleWeight := map[string]int{
		RoleMember:  1,
		RoleManager: 2,
		RoleAdmin:   3,
	}
	return roleWeight[userRole] >= roleWeight[requiredRole]
}

// withRecovery is a middleware that recovers from panics in handlers.
func (p *Plugin) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				p.API.LogError("recovered from panic in HTTP handler", "err", rec)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
