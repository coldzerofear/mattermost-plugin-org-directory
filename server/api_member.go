package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// handleGetMembers handles GET /api/v1/nodes/{id}/members
func (p *Plugin) handleGetMembers(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = p.getConfiguration().getDefaultPageSize()
	}

	members, err := p.store.GetMembers(id, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get members")
		return
	}
	writeJSON(w, http.StatusOK, members)
}

// handleAddMember handles POST /api/v1/nodes/{id}/members
func (p *Plugin) handleAddMember(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.checkNodePermission(userID, id, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		Role     string `json:"role"`
		Position string `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Verify the target user exists
	if _, appErr := p.API.GetUser(req.UserID); appErr != nil {
		writeError(w, http.StatusBadRequest, "user not found")
		return
	}

	// Check for duplicate membership
	exists, _ := p.store.IsMember(id, req.UserID)
	if exists {
		writeError(w, http.StatusConflict, "user is already a member of this node")
		return
	}

	if req.Role == "" {
		req.Role = RoleMember
	}

	member := &pluginmodel.OrgMember{
		NodeID:   id,
		UserID:   req.UserID,
		Role:     req.Role,
		Position: req.Position,
		Source:   "local",
	}

	created, err := p.store.AddMember(member)
	if err != nil {
		p.API.LogError("failed to add member", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}

	p.broadcastMemberUpdate("add", id, req.UserID)
	p.logAudit(userID, "add_member", "member", created.ID, `{}`)
	writeJSON(w, http.StatusCreated, created)
}

// handleRemoveMember handles DELETE /api/v1/nodes/{id}/members/{userId}
func (p *Plugin) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	vars := mux.Vars(r)
	nodeID := vars["id"]
	targetUserID := vars["userId"]

	if !p.checkNodePermission(userID, nodeID, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := p.store.RemoveMember(nodeID, targetUserID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove member")
		return
	}

	p.broadcastMemberUpdate("remove", nodeID, targetUserID)
	p.logAudit(userID, "remove_member", "member", nodeID+":"+targetUserID, `{}`)
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateMemberRole handles PUT /api/v1/nodes/{id}/members/{userId}/role
func (p *Plugin) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	vars := mux.Vars(r)
	nodeID := vars["id"]
	targetUserID := vars["userId"]

	if !p.checkNodePermission(userID, nodeID, RoleAdmin) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := p.store.UpdateMemberRole(nodeID, targetUserID, req.Role); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update role")
		return
	}

	p.broadcastMemberUpdate("update_role", nodeID, targetUserID)
	if details, _ := json.Marshal(map[string]string{"role": req.Role}); details != nil {
		p.logAudit(userID, "update_member_role", "member", nodeID+":"+targetUserID, string(details))
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateMemberPosition handles PUT /api/v1/nodes/{id}/members/{userId}/position
func (p *Plugin) handleUpdateMemberPosition(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	vars := mux.Vars(r)
	nodeID := vars["id"]
	targetUserID := vars["userId"]

	if !p.checkNodePermission(userID, nodeID, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req struct {
		Position string `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := p.store.UpdateMemberPosition(nodeID, targetUserID, req.Position); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update position")
		return
	}

	if details, _ := json.Marshal(map[string]string{"position": req.Position}); details != nil {
		p.logAudit(userID, "update_member_position", "member", nodeID+":"+targetUserID, string(details))
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleReorderMembers handles POST /api/v1/nodes/{id}/members/reorder
func (p *Plugin) handleReorderMembers(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.checkNodePermission(userID, id, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req struct {
		UserIDs []string `json:"user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := p.store.ReorderMembers(id, req.UserIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder members")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
