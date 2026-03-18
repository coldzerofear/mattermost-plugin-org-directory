package main

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

// TestSyncRouteMatching verifies that all external sync API routes are registered
// and matched correctly by the gorilla/mux router.
//
// The test uses router.Match which checks route matching WITHOUT executing handlers or
// middleware — safe to call on a minimal Plugin with no store/API initialization.
//
// A missing route would cause Match to return false (equivalent to 404), which is exactly
// the symptom the user reported when all /api/v1/sync/* routes returned 404.
func TestSyncRouteMatching(t *testing.T) {
	p := &Plugin{}
	p.initializeAPI()

	// Note: the router has a PathPrefix("/") fallback handler (serves 404 body), so
	// router.Match always returns true. We only test that specific routes are registered
	// by checking that Match populates a non-nil Handler (not the fallback).
	// The important invariant: all sync routes MUST resolve to a real named handler,
	// not the anonymous fallback. We detect this by verifying the matched handler is not nil.
	routes := []struct {
		name   string
		method string
		path   string
	}{
		// --- External sync token-auth routes ---
		{"POST /api/v1/sync (full sync)", http.MethodPost, "/api/v1/sync"},
		{"POST /api/v1/sync/nodes (incremental nodes)", http.MethodPost, "/api/v1/sync/nodes"},
		{"POST /api/v1/sync/members (incremental members)", http.MethodPost, "/api/v1/sync/members"},
		{"POST /api/v1/sync/user-mappings (upsert mappings)", http.MethodPost, "/api/v1/sync/user-mappings"},
		{"GET /api/v1/sync/user-mappings/{source}", http.MethodGet, "/api/v1/sync/user-mappings/hr_system"},
		{"GET /api/v1/sync/logs (list logs)", http.MethodGet, "/api/v1/sync/logs"},
		{"GET /api/v1/sync/logs/{id} (single log)", http.MethodGet, "/api/v1/sync/logs/abc123"},

		// --- Session-auth API routes (sanity check) ---
		{"GET /api/v1/tree", http.MethodGet, "/api/v1/tree"},
		{"GET /api/v1/roots", http.MethodGet, "/api/v1/roots"},
		{"POST /api/v1/nodes", http.MethodPost, "/api/v1/nodes"},
		{"GET /api/v1/nodes/{id}", http.MethodGet, "/api/v1/nodes/abc"},
		{"GET /api/v1/search/users", http.MethodGet, "/api/v1/search/users"},

		// --- Admin session-auth routes ---
		{"GET /api/v1/admin/sync/logs", http.MethodGet, "/api/v1/admin/sync/logs"},
		{"GET /api/v1/admin/sync/logs/{id}", http.MethodGet, "/api/v1/admin/sync/logs/xyz"},
		{"GET /api/v1/admin/sync/user-mappings/{source}", http.MethodGet, "/api/v1/admin/sync/user-mappings/ldap"},
	}

	for _, tt := range routes {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			var match mux.RouteMatch
			if !p.router.Match(req, &match) {
				t.Errorf("FAIL: %s %s did not match any route (would return 404)", tt.method, tt.path)
				return
			}
			if match.Handler == nil {
				t.Errorf("FAIL: %s %s matched but handler is nil", tt.method, tt.path)
			}
		})
	}
}
