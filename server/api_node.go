package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// handleCreateNode handles POST /api/v1/nodes
func (p *Plugin) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")

	var req struct {
		Name        string `json:"name"`
		ParentID    string `json:"parent_id"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		SortOrder   int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Permission: system admin or node manager of the parent
	if req.ParentID != "" && !p.checkNodePermission(userID, req.ParentID, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	} else if req.ParentID == "" && !p.isSystemAdmin(userID) {
		writeError(w, http.StatusForbidden, "only system admins can create root nodes")
		return
	}

	node := &pluginmodel.OrgNode{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Description: req.Description,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		CreatorID:   userID,
		Source:      "local",
	}

	created, err := p.store.CreateNode(node)
	if err != nil {
		p.API.LogError("failed to create node", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create node")
		return
	}

	p.broadcastTreeUpdate("create", created)
	p.logAudit(userID, "create_node", "node", created.ID, `{}`)
	writeJSON(w, http.StatusCreated, created)
}

// handleGetNode handles GET /api/v1/nodes/{id}
func (p *Plugin) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	node, err := p.store.GetNode(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	// Attach child count and member count
	children, _ := p.store.GetChildNodes(id)
	node.HasChildren = len(children) > 0
	node.MemberCount, _ = p.store.GetNodeMemberCount(id, false)

	writeJSON(w, http.StatusOK, node)
}

// handleUpdateNode handles PUT /api/v1/nodes/{id}
func (p *Plugin) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.checkNodePermission(userID, id, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	node, err := p.store.GetNode(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Icon        *string `json:"icon"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		node.Name = *req.Name
	}
	if req.Description != nil {
		node.Description = *req.Description
	}
	if req.Icon != nil {
		node.Icon = *req.Icon
	}
	if req.SortOrder != nil {
		node.SortOrder = *req.SortOrder
	}

	if err := p.store.UpdateNode(node); err != nil {
		p.API.LogError("failed to update node", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update node")
		return
	}

	p.broadcastTreeUpdate("update", node)
	p.logAudit(userID, "update_node", "node", id, `{}`)
	writeJSON(w, http.StatusOK, node)
}

// handleDeleteNode handles DELETE /api/v1/nodes/{id}
func (p *Plugin) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.isSystemAdmin(userID) {
		writeError(w, http.StatusForbidden, "only system admins can delete nodes")
		return
	}

	node, err := p.store.GetNode(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}

	cascadeStrategy := r.URL.Query().Get("cascade_strategy")
	if cascadeStrategy == "" {
		config := p.getConfiguration()
		cascadeStrategy = config.SyncFullDeleteStrategy
		if cascadeStrategy == "" {
			cascadeStrategy = "cascade_delete"
		}
	}

	if err := p.store.DeleteNode(id, cascadeStrategy); err != nil {
		p.API.LogError("failed to delete node", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete node")
		return
	}

	p.broadcastTreeUpdate("delete", node)
	if details, _ := json.Marshal(map[string]string{"cascade_strategy": cascadeStrategy}); details != nil {
		p.logAudit(userID, "delete_node", "node", id, string(details))
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetChildren handles GET /api/v1/nodes/{id}/children
func (p *Plugin) handleGetChildren(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	children, err := p.store.GetChildNodes(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get children")
		return
	}

	// Attach has_children flag and member counts
	for _, child := range children {
		subs, _ := p.store.GetChildNodes(child.ID)
		child.HasChildren = len(subs) > 0
		child.MemberCount, _ = p.store.GetNodeMemberCount(child.ID, false)
	}

	writeJSON(w, http.StatusOK, children)
}

// handleMoveNode handles POST /api/v1/nodes/{id}/move
func (p *Plugin) handleMoveNode(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.isSystemAdmin(userID) {
		writeError(w, http.StatusForbidden, "only system admins can move nodes")
		return
	}

	var req struct {
		NewParentID string `json:"new_parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := p.store.MoveNode(id, req.NewParentID); err != nil {
		p.API.LogError("failed to move node", "err", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	node, _ := p.store.GetNode(id)
	p.broadcastTreeUpdate("move", node)
	p.logAudit(userID, "move_node", "node", id, `{"new_parent_id":"`+req.NewParentID+`"}`)
	writeJSON(w, http.StatusOK, node)
}

// handleReorderNodes handles POST /api/v1/nodes/{id}/reorder
func (p *Plugin) handleReorderNodes(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-Id")
	id := mux.Vars(r)["id"]

	if !p.checkNodePermission(userID, id, RoleManager) {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	var req struct {
		NodeIDs []string `json:"node_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := p.store.ReorderNodes(id, req.NodeIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reorder nodes")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetNodeStats handles GET /api/v1/nodes/{id}/stats
func (p *Plugin) handleGetNodeStats(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	recursive, _ := strconv.ParseBool(r.URL.Query().Get("recursive"))

	count, err := p.store.GetNodeMemberCount(id, recursive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"member_count": count})
}

// logAudit is a helper to record an audit log entry.
func (p *Plugin) logAudit(actorID, action, targetType, targetID, details string) {
	config := p.getConfiguration()
	if !config.EnableAuditLog {
		return
	}
	if err := p.store.LogAction(&pluginmodel.AuditLog{
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Details:    details,
	}); err != nil {
		p.API.LogWarn("failed to write audit log", "err", err)
	}
}
