package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// initializeAPI sets up all HTTP routes.
//
// IMPORTANT: more-specific PathPrefix routes must be registered BEFORE broader
// ones. gorilla/mux tries routes in registration order; once PathPrefix("/api/v1")
// matches it claims the path, preventing PathPrefix("/api/v1/sync") from ever
// being reached if registered afterwards.
func (p *Plugin) initializeAPI() {
	p.router = mux.NewRouter()
	p.router.Use(p.withRecovery)

	// --- External sync API (Token auth) — registered FIRST (more specific prefix) ---
	//
	// The root path POST /api/v1/sync is registered on the main router directly.
	// gorilla/mux's PathPrefix subrouter has undefined behavior when the remaining
	// path after stripping the prefix is an empty string (HandleFunc("")), so we
	// avoid that edge case by using an exact-path route on the main router instead.
	p.router.Handle("/api/v1/sync", p.checkSyncToken(http.HandlerFunc(p.handleSyncData))).Methods(http.MethodPost)

	syncAPI := p.router.PathPrefix("/api/v1/sync").Subrouter()
	syncAPI.Use(p.checkSyncToken)
	syncAPI.HandleFunc("/nodes", p.handleSyncNodes).Methods(http.MethodPost)
	syncAPI.HandleFunc("/members", p.handleSyncMembers).Methods(http.MethodPost)
	syncAPI.HandleFunc("/user-mappings", p.handleSyncUserMappings).Methods(http.MethodPost)
	syncAPI.HandleFunc("/user-mappings/{source}", p.handleGetUserMappings).Methods(http.MethodGet)
	syncAPI.HandleFunc("/logs", p.handleGetSyncLogs).Methods(http.MethodGet)
	syncAPI.HandleFunc("/logs/{id}", p.handleGetSyncLog).Methods(http.MethodGet)

	// --- Static assets ---
	p.router.HandleFunc("/icon", p.handleIcon).Methods(http.MethodGet)

	// --- Session-auth API — registered AFTER sync (broader prefix) ---
	api := p.router.PathPrefix("/api/v1").Subrouter()

	// Organization node management
	api.HandleFunc("/nodes", p.checkAuth(p.handleCreateNode)).Methods(http.MethodPost)
	api.HandleFunc("/nodes/{id}", p.checkAuth(p.handleGetNode)).Methods(http.MethodGet)
	api.HandleFunc("/nodes/{id}", p.checkAuth(p.handleUpdateNode)).Methods(http.MethodPut)
	api.HandleFunc("/nodes/{id}", p.checkAuth(p.handleDeleteNode)).Methods(http.MethodDelete)
	api.HandleFunc("/nodes/{id}/children", p.checkAuth(p.handleGetChildren)).Methods(http.MethodGet)
	api.HandleFunc("/nodes/{id}/move", p.checkAuth(p.handleMoveNode)).Methods(http.MethodPost)
	api.HandleFunc("/nodes/{id}/reorder", p.checkAuth(p.handleReorderNodes)).Methods(http.MethodPost)
	api.HandleFunc("/nodes/{id}/stats", p.checkAuth(p.handleGetNodeStats)).Methods(http.MethodGet)

	// Tree queries
	api.HandleFunc("/tree", p.checkAuth(p.handleGetFullTree)).Methods(http.MethodGet)
	api.HandleFunc("/tree/{id}", p.checkAuth(p.handleGetSubTree)).Methods(http.MethodGet)
	api.HandleFunc("/roots", p.checkAuth(p.handleGetRoots)).Methods(http.MethodGet)

	// Member management
	api.HandleFunc("/nodes/{id}/members", p.checkAuth(p.handleGetMembers)).Methods(http.MethodGet)
	api.HandleFunc("/nodes/{id}/members", p.checkAuth(p.handleAddMember)).Methods(http.MethodPost)
	api.HandleFunc("/nodes/{id}/members/{userId}", p.checkAuth(p.handleRemoveMember)).Methods(http.MethodDelete)
	api.HandleFunc("/nodes/{id}/members/{userId}/role", p.checkAuth(p.handleUpdateMemberRole)).Methods(http.MethodPut)
	api.HandleFunc("/nodes/{id}/members/{userId}/position", p.checkAuth(p.handleUpdateMemberPosition)).Methods(http.MethodPut)
	api.HandleFunc("/nodes/{id}/members/reorder", p.checkAuth(p.handleReorderMembers)).Methods(http.MethodPost)

	// Search
	api.HandleFunc("/search/users", p.checkAuth(p.handleSearchUsers)).Methods(http.MethodGet)
	api.HandleFunc("/search/nodes", p.checkAuth(p.handleSearchNodes)).Methods(http.MethodGet)

	// User
	api.HandleFunc("/users/{userId}/nodes", p.checkAuth(p.handleGetUserNodes)).Methods(http.MethodGet)

	// Admin sync management (session auth, system admin only)
	api.HandleFunc("/admin/sync/logs", p.checkAuth(p.checkAdmin(p.handleGetSyncLogs))).Methods(http.MethodGet)
	api.HandleFunc("/admin/sync/logs/{id}", p.checkAuth(p.checkAdmin(p.handleGetSyncLog))).Methods(http.MethodGet)
	api.HandleFunc("/admin/sync/user-mappings/{source}", p.checkAuth(p.checkAdmin(p.handleGetUserMappings))).Methods(http.MethodGet)

	// 404 fallback
	p.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}

// handleIcon serves the plugin icon as SVG.
func (p *Plugin) handleIcon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(iconSVG))
}

const iconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <rect width="100" height="100" rx="20" fill="#1c58d9"/>
  <circle cx="50" cy="18" r="8" fill="white"/>
  <circle cx="25" cy="50" r="8" fill="white"/>
  <circle cx="75" cy="50" r="8" fill="white"/>
  <circle cx="15" cy="80" r="7" fill="white" opacity="0.8"/>
  <circle cx="38" cy="80" r="7" fill="white" opacity="0.8"/>
  <circle cx="62" cy="80" r="7" fill="white" opacity="0.8"/>
  <circle cx="85" cy="80" r="7" fill="white" opacity="0.8"/>
  <line x1="50" y1="26" x2="25" y2="42" stroke="white" stroke-width="2.5" opacity="0.7"/>
  <line x1="50" y1="26" x2="75" y2="42" stroke="white" stroke-width="2.5" opacity="0.7"/>
  <line x1="25" y1="58" x2="15" y2="73" stroke="white" stroke-width="2" opacity="0.6"/>
  <line x1="25" y1="58" x2="38" y2="73" stroke="white" stroke-width="2" opacity="0.6"/>
  <line x1="75" y1="58" x2="62" y2="73" stroke="white" stroke-width="2" opacity="0.6"/>
  <line x1="75" y1="58" x2="85" y2="73" stroke="white" stroke-width="2" opacity="0.6"/>
</svg>`

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, statusCode int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
