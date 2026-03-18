package main

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// handleSearchUsers handles GET /api/v1/search/users?q=keyword&page=0&per_page=20
func (p *Plugin) handleSearchUsers(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	config := p.getConfiguration()

	// Check if user search is allowed for non-admins
	if !config.AllowUserSearch && !p.isSystemAdmin(userID) {
		writeError(w, http.StatusForbidden, "user search is restricted to administrators")
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = 20
	}

	results, err := p.store.SearchMembers(q, page, perPage)
	if err != nil {
		p.API.LogError("search members failed", "err", err)
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// handleSearchNodes handles GET /api/v1/search/nodes?q=keyword
func (p *Plugin) handleSearchNodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	nodes, err := p.store.SearchNodes(q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}

	writeJSON(w, http.StatusOK, nodes)
}

// handleGetUserNodes handles GET /api/v1/users/{userId}/nodes
func (p *Plugin) handleGetUserNodes(w http.ResponseWriter, r *http.Request) {
	targetUserID := mux.Vars(r)["userId"]

	nodes, err := p.store.GetUserNodes(targetUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user nodes")
		return
	}

	writeJSON(w, http.StatusOK, nodes)
}
