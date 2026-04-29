package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	mmmodel "github.com/mattermost/mattermost/server/public/model"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// handleSyncData handles POST /api/v1/sync — full or incremental sync of nodes+members.
func (p *Plugin) handleSyncData(w http.ResponseWriter, r *http.Request) {
	var req pluginmodel.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	resp := p.executeSyncRequest(&req)
	writeJSON(w, http.StatusOK, resp)
}

// handleSyncNodes handles POST /api/v1/sync/nodes — sync only nodes.
func (p *Plugin) handleSyncNodes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source   string                        `json:"source"`
		SyncType string                        `json:"sync_type"`
		Nodes    []pluginmodel.SyncNodePayload `json:"nodes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	syncReq := &pluginmodel.SyncRequest{
		Source:   req.Source,
		SyncType: req.SyncType,
		Nodes:    req.Nodes,
	}
	resp := p.executeSyncRequest(syncReq)
	writeJSON(w, http.StatusOK, resp)
}

// handleSyncMembers handles POST /api/v1/sync/members — sync only members.
func (p *Plugin) handleSyncMembers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source   string                          `json:"source"`
		SyncType string                          `json:"sync_type"`
		Members  []pluginmodel.SyncMemberPayload `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	syncReq := &pluginmodel.SyncRequest{
		Source:   req.Source,
		SyncType: req.SyncType,
		Members:  req.Members,
	}
	resp := p.executeSyncRequest(syncReq)
	writeJSON(w, http.StatusOK, resp)
}

// handleSyncUserMappings handles POST /api/v1/sync/user-mappings.
func (p *Plugin) handleSyncUserMappings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source   string                    `json:"source"`
		Mappings []pluginmodel.UserMapping `json:"mappings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created := 0
	updated := 0
	for _, m := range req.Mappings {
		m.Source = req.Source
		_, err := p.store.UpsertUserMapping(&m)
		if err != nil {
			p.API.LogError("failed to upsert user mapping", "err", err)
			continue
		}
		if m.CreateAt == 0 {
			updated++
		} else {
			created++
		}
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"created": created,
		"updated": updated,
	})
}

// handleGetUserMappings handles GET /api/v1/sync/user-mappings/{source}.
func (p *Plugin) handleGetUserMappings(w http.ResponseWriter, r *http.Request) {
	source := mux.Vars(r)["source"]
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = p.getConfiguration().getDefaultPageSize()
	}

	mappings, err := p.store.GetUserMappingsBySource(source, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get mappings")
		return
	}

	writeJSON(w, http.StatusOK, mappings)
}

// handleGetSyncLogs handles GET /api/v1/sync/logs or /api/v1/admin/sync/logs.
func (p *Plugin) handleGetSyncLogs(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = p.getConfiguration().getDefaultPageSize()
	}

	logs, err := p.store.GetSyncLogs(source, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get sync logs")
		return
	}

	writeJSON(w, http.StatusOK, logs)
}

// handleGetSyncLog handles GET /api/v1/sync/logs/{id} or /api/v1/admin/sync/logs/{id}.
func (p *Plugin) handleGetSyncLog(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	log, err := p.store.GetSyncLog(id)
	if err != nil || log == nil {
		writeError(w, http.StatusNotFound, "sync log not found")
		return
	}

	writeJSON(w, http.StatusOK, log)
}

// handleListSyncNodes handles GET /api/v1/sync/nodes.
func (p *Plugin) handleListSyncNodes(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	parseNonNegativeInt := func(name string) (int, bool, error) {
		value := strings.TrimSpace(r.URL.Query().Get(name))
		if value == "" {
			return -1, false, nil
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return -1, true, err
		}
		return parsed, true, nil
	}

	depth, hasDepth, err := parseNonNegativeInt("depth")
	if err != nil {
		writeError(w, http.StatusBadRequest, "depth must be a non-negative integer")
		return
	}

	maxDepth, hasMaxDepth, err := parseNonNegativeInt("max_depth")
	if err != nil {
		writeError(w, http.StatusBadRequest, "max_depth must be a non-negative integer")
		return
	}

	parentExternalID := strings.TrimSpace(r.URL.Query().Get("parent_external_id"))

	nodes, err := p.store.GetNodesBySource(source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get nodes")
		return
	}

	filteredNodes := make([]*pluginmodel.OrgNode, 0, len(nodes))

	var parentNode *pluginmodel.OrgNode
	if parentExternalID != "" {
		parentNode, err = p.store.GetNodeByExternalID(source, parentExternalID)
		if err != nil || parentNode == nil {
			writeError(w, http.StatusNotFound, "parent node not found")
			return
		}
	}

	for _, node := range nodes {
		if parentNode != nil {
			if node.ID == parentNode.ID {
				continue
			}
			prefix := parentNode.Path + "/"
			if !strings.HasPrefix(node.Path, prefix) {
				continue
			}
			relativeDepth := node.Depth - parentNode.Depth - 1
			if relativeDepth < 0 {
				continue
			}
			if hasDepth && relativeDepth != depth {
				continue
			}
			if hasMaxDepth && relativeDepth > maxDepth {
				continue
			}
			filteredNodes = append(filteredNodes, node)
			continue
		}

		if hasDepth && node.Depth != depth {
			continue
		}
		if hasMaxDepth && node.Depth > maxDepth {
			continue
		}
		filteredNodes = append(filteredNodes, node)
	}

	// Pass the full unfiltered `nodes` set as the parent-resolution pool so that
	// parents excluded by depth/parent_external_id filters are still resolved
	// in memory — avoids per-parent GetNode calls in buildSyncNodeDTOs.
	responseNodes := p.buildSyncNodeDTOsWithPool(filteredNodes, nodes)
	writeJSON(w, http.StatusOK, &pluginmodel.SyncNodeListResponse{
		Source: source,
		Nodes:  responseNodes,
	})
}

// handleGetSyncNode handles GET /api/v1/sync/nodes/{externalID}.
func (p *Plugin) handleGetSyncNode(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	externalID := mux.Vars(r)["externalID"]
	node, err := p.store.GetNodeByExternalID(source, externalID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	pathNodes, err := p.store.GetNodePath(node.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get node path")
		return
	}

	writeJSON(w, http.StatusOK, &pluginmodel.SyncNodeDetailResponse{
		Node: p.buildSyncNodeDTO(node),
		Path: p.buildSyncNodeDTOs(pathNodes),
	})
}

// handleGetSyncNodeChildren handles GET /api/v1/sync/nodes/{externalID}/children.
func (p *Plugin) handleGetSyncNodeChildren(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	externalID := mux.Vars(r)["externalID"]
	node, err := p.store.GetNodeByExternalID(source, externalID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	children, err := p.store.GetChildNodes(node.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get child nodes")
		return
	}

	responseNodes := make([]*pluginmodel.SyncNodeDTO, 0, len(children))
	for _, child := range children {
		if child.Source != source {
			continue
		}
		responseNodes = append(responseNodes, p.buildSyncNodeDTO(child))
	}

	writeJSON(w, http.StatusOK, &pluginmodel.SyncNodeListResponse{
		Source: source,
		Nodes:  responseNodes,
	})
}

// handleGetSyncNodeMembers handles GET /api/v1/sync/nodes/{externalID}/members.
func (p *Plugin) handleGetSyncNodeMembers(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = p.getConfiguration().getDefaultPageSize()
	}
	recursive, _ := strconv.ParseBool(r.URL.Query().Get("recursive"))

	externalID := mux.Vars(r)["externalID"]
	node, err := p.store.GetNodeByExternalID(source, externalID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "node not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get node")
		return
	}

	nodesByID := map[string]*pluginmodel.OrgNode{node.ID: node}
	targetNodeIDs := []string{node.ID}
	if recursive {
		subtree, subTreeErr := p.store.GetSubTree(node.ID)
		if subTreeErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to get subtree")
			return
		}
		for _, subNode := range subtree {
			if subNode.Source != source {
				continue
			}
			nodesByID[subNode.ID] = subNode
			targetNodeIDs = append(targetNodeIDs, subNode.ID)
		}
	}

	members, err := p.store.GetMembersForNodes(targetNodeIDs, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get members")
		return
	}

	responseMembers := make([]*pluginmodel.SyncNodeMemberItem, 0, len(members))
	for _, member := range members {
		memberNode, ok := nodesByID[member.NodeID]
		if !ok {
			continue
		}
		responseMembers = append(responseMembers, &pluginmodel.SyncNodeMemberItem{
			OrgMemberWithUser: member,
			NodeName:          memberNode.Name,
			NodeSource:        memberNode.Source,
			NodeExternalID:    memberNode.ExternalID,
		})
	}

	writeJSON(w, http.StatusOK, &pluginmodel.SyncNodeMembersResponse{
		Node:      p.buildSyncNodeDTO(node),
		Recursive: recursive,
		Total:     len(responseMembers),
		Members:   responseMembers,
	})
}

// handleGetSyncUserNodes handles GET /api/v1/sync/users/{externalUserID}/nodes.
func (p *Plugin) handleGetSyncUserNodes(w http.ResponseWriter, r *http.Request) {
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		writeError(w, http.StatusBadRequest, "source is required")
		return
	}

	externalUserID := mux.Vars(r)["externalUserID"]
	mapping, err := p.store.GetUserMappingByExternalID(source, externalUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user mapping not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user mapping")
		return
	}
	if mapping == nil || mapping.MmUserID == "" {
		writeError(w, http.StatusNotFound, "user mapping not found")
		return
	}

	nodes, err := p.store.GetUserNodes(mapping.MmUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get user nodes")
		return
	}

	filteredNodes := make([]*pluginmodel.OrgNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Source == source {
			filteredNodes = append(filteredNodes, node)
		}
	}

	writeJSON(w, http.StatusOK, &pluginmodel.SyncUserNodesResponse{
		Source:         source,
		ExternalUserID: externalUserID,
		MmUserID:       mapping.MmUserID,
		Total:          len(filteredNodes),
		Nodes:          p.buildSyncNodeDTOs(filteredNodes),
	})
}

func (p *Plugin) buildSyncNodeDTO(node *pluginmodel.OrgNode) *pluginmodel.SyncNodeDTO {
	if node == nil {
		return nil
	}

	dto := &pluginmodel.SyncNodeDTO{OrgNode: node}
	if node.ParentID != "" {
		parent, err := p.store.GetNode(node.ParentID)
		if err == nil {
			dto.ParentExternalID = parent.ExternalID
		}
	}
	return dto
}

func (p *Plugin) buildSyncNodeDTOs(nodes []*pluginmodel.OrgNode) []*pluginmodel.SyncNodeDTO {
	return p.buildSyncNodeDTOsWithPool(nodes, nodes)
}

// buildSyncNodeDTOsWithPool resolves each ParentID -> ParentExternalID using
// `pool` first (no DB hit) and falls back to a single GetNode per unique
// unresolved ParentID. Callers that already have a larger in-memory corpus
// (e.g. the full source-set) should pass it as `pool` to avoid N+1 lookups
// when `nodes` is a filtered subset. Sorts `nodes` in place — callers must
// not retain insertion order after this call.
func (p *Plugin) buildSyncNodeDTOsWithPool(nodes []*pluginmodel.OrgNode, pool []*pluginmodel.OrgNode) []*pluginmodel.SyncNodeDTO {
	if len(nodes) == 0 {
		return []*pluginmodel.SyncNodeDTO{}
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Depth != nodes[j].Depth {
			return nodes[i].Depth < nodes[j].Depth
		}
		if nodes[i].SortOrder != nodes[j].SortOrder {
			return nodes[i].SortOrder < nodes[j].SortOrder
		}
		if nodes[i].Path != nodes[j].Path {
			return nodes[i].Path < nodes[j].Path
		}
		return nodes[i].Name < nodes[j].Name
	})

	poolByID := make(map[string]*pluginmodel.OrgNode, len(pool))
	for _, node := range pool {
		poolByID[node.ID] = node
	}

	parentExternalIDByParentID := map[string]string{}
	var unresolvedParentIDs map[string]struct{}
	for _, node := range nodes {
		if node.ParentID == "" {
			continue
		}

		if _, ok := parentExternalIDByParentID[node.ParentID]; ok {
			continue
		}

		if parentNode, ok := poolByID[node.ParentID]; ok {
			parentExternalIDByParentID[node.ParentID] = parentNode.ExternalID
			continue
		}

		if unresolvedParentIDs == nil {
			unresolvedParentIDs = map[string]struct{}{}
		}
		unresolvedParentIDs[node.ParentID] = struct{}{}
	}

	for parentID := range unresolvedParentIDs {
		parentNode, err := p.store.GetNode(parentID)
		if err != nil || parentNode == nil {
			continue
		}
		parentExternalIDByParentID[parentID] = parentNode.ExternalID
	}

	result := make([]*pluginmodel.SyncNodeDTO, 0, len(nodes))
	for _, node := range nodes {
		dto := &pluginmodel.SyncNodeDTO{OrgNode: node}
		if node.ParentID != "" {
			dto.ParentExternalID = parentExternalIDByParentID[node.ParentID]
		}
		result = append(result, dto)
	}
	return result
}

// executeSyncRequest runs a full or incremental sync and returns the response.
func (p *Plugin) executeSyncRequest(req *pluginmodel.SyncRequest) *pluginmodel.SyncResponse {
	resp := &pluginmodel.SyncResponse{}

	// Create sync log
	syncLog := &pluginmodel.SyncLog{
		Source:    req.Source,
		SyncType:  req.SyncType,
		Status:    "running",
		StartedAt: mmmodel.GetMillis(),
		Details:   "{}",
	}
	if created, err := p.store.CreateSyncLog(syncLog); err == nil {
		syncLog = created
		resp.SyncLogID = syncLog.ID
	}

	// Topologically sort nodes (parents before children)
	sortedNodes := topologicalSortNodes(req.Nodes)

	// Determine cascade strategy once (used for action=delete nodes)
	syncConfig := p.getConfiguration()
	cascadeStrategy := syncConfig.SyncFullDeleteStrategy
	if cascadeStrategy == "" {
		cascadeStrategy = "cascade_delete"
	}

	// Process nodes
	syncedNodeExtIDs := make([]string, 0, len(sortedNodes))
	for i := range sortedNodes {
		payload := &sortedNodes[i]
		resp.TotalNodes++

		// Explicit delete action
		if payload.Action == "delete" {
			existing, err := p.store.GetNodeByExternalID(req.Source, payload.ExternalID)
			if err != nil || existing == nil {
				resp.Errors = append(resp.Errors, "node not found for delete: "+payload.ExternalID)
				continue
			}
			if err := p.store.DeleteNode(existing.ID, cascadeStrategy); err != nil {
				resp.Errors = append(resp.Errors, "failed to delete node "+payload.ExternalID+": "+err.Error())
			} else {
				resp.DeletedNodes++
			}
			continue
		}

		// Resolve parent internal ID
		parentID := ""
		if payload.ParentExternalID != "" {
			parent, err := p.store.GetNodeByExternalID(req.Source, payload.ParentExternalID)
			if err != nil {
				resp.Errors = append(resp.Errors, "parent not found for node: "+payload.ExternalID)
				continue
			}
			parentID = parent.ID
		}

		node := &pluginmodel.OrgNode{
			Name:        payload.Name,
			ParentID:    parentID,
			SortOrder:   payload.SortOrder,
			Description: payload.Description,
			Icon:        payload.Icon,
			Metadata:    payload.Metadata,
			Source:      req.Source,
			ExternalID:  payload.ExternalID,
		}
		if node.Metadata == "" {
			node.Metadata = "{}"
		}

		existing, _ := p.store.GetNodeByExternalID(req.Source, payload.ExternalID)
		_, err := p.store.UpsertNodeByExternalID(node)
		if err != nil {
			resp.Errors = append(resp.Errors, "failed to upsert node "+payload.ExternalID+": "+err.Error())
		} else if existing != nil {
			resp.UpdatedNodes++
		} else {
			resp.CreatedNodes++
		}
		syncedNodeExtIDs = append(syncedNodeExtIDs, payload.ExternalID)
	}

	// Process members
	syncedMemberExtIDs := make([]string, 0, len(req.Members))
	syncedMemberNodeUserKeys := make([]string, 0, len(req.Members))
	for i := range req.Members {
		payload := &req.Members[i]
		resp.TotalMembers++

		// Resolve node (needed for both upsert and delete)
		node, err := p.store.GetNodeByExternalID(req.Source, payload.NodeExternalID)
		if err != nil {
			resp.Errors = append(resp.Errors, "node not found for member: "+payload.ExternalUserID)
			continue
		}

		// Explicit delete action
		if payload.Action == "delete" {
			mmUserID, skipReason := p.resolveMMUser(syncConfig.SyncUserMatchStrategy, req.Source, payload)
			if mmUserID == "" {
				resp.SkippedUsers++
				resp.SkippedDetails = append(resp.SkippedDetails, pluginmodel.SkippedUser{
					ExternalUserID:   payload.ExternalUserID,
					ExternalUsername: payload.ExternalUsername,
					ExternalEmail:    payload.ExternalEmail,
					Reason:           skipReason,
				})
				continue
			}
			if err := p.store.RemoveMember(node.ID, mmUserID); err != nil {
				resp.Errors = append(resp.Errors, "failed to delete member: "+err.Error())
			} else {
				resp.DeletedMembers++
			}
			continue
		}

		// Resolve Mattermost user
		mmUserID, skipReason := p.resolveMMUser(syncConfig.SyncUserMatchStrategy, req.Source, payload)
		if mmUserID == "" {
			resp.SkippedUsers++
			resp.SkippedDetails = append(resp.SkippedDetails, pluginmodel.SkippedUser{
				ExternalUserID:   payload.ExternalUserID,
				ExternalUsername: payload.ExternalUsername,
				ExternalEmail:    payload.ExternalEmail,
				Reason:           skipReason,
			})
			continue
		}

		role := payload.Role
		if role == "" {
			role = "member"
		}
		member := &pluginmodel.OrgMember{
			NodeID:     node.ID,
			UserID:     mmUserID,
			Role:       role,
			Position:   payload.Position,
			SortOrder:  payload.SortOrder,
			Source:     req.Source,
			ExternalID: payload.ExternalID,
		}

		existing, _ := p.store.GetMemberRole(node.ID, mmUserID)
		if strings.TrimSpace(payload.ExternalID) != "" {
			membersAtNode, getErr := p.store.GetAllMembersByNodeID(node.ID)
			if getErr == nil {
				for _, candidate := range membersAtNode {
					if candidate.Source == req.Source && candidate.ExternalID == payload.ExternalID {
						existing = candidate
						break
					}
				}
			}
		}
		_, err = p.store.UpsertMemberByExternalID(member)
		if err != nil {
			resp.Errors = append(resp.Errors, "failed to upsert member: "+err.Error())
		} else if existing != nil {
			resp.UpdatedMembers++
		} else {
			resp.CreatedMembers++
		}
		if payload.ExternalID != "" {
			syncedMemberExtIDs = append(syncedMemberExtIDs, payload.ExternalID)
		}
		syncedMemberNodeUserKeys = append(syncedMemberNodeUserKeys, node.ID+":"+mmUserID)
	}

	// Full sync: soft-delete stale data from this source
	if req.SyncType == "full" {
		if len(req.Nodes) > 0 {
			deleted, _ := p.store.SoftDeleteNodesBySource(req.Source, syncedNodeExtIDs)
			resp.DeletedNodes += deleted
		}
		if len(req.Members) > 0 {
			deleted, _ := p.store.SoftDeleteMembersBySource(req.Source, syncedMemberExtIDs, syncedMemberNodeUserKeys)
			resp.DeletedMembers += deleted
		}
	}

	// Determine final status
	switch {
	case len(resp.Errors) > 0 && resp.CreatedNodes+resp.UpdatedNodes == 0:
		resp.Status = "failed"
	case resp.SkippedUsers > 0 || len(resp.Errors) > 0:
		resp.Status = "partial_success"
	default:
		resp.Status = "success"
	}

	// Update sync log
	syncLog.Status = resp.Status
	syncLog.FinishedAt = mmmodel.GetMillis()
	syncLog.TotalNodes = resp.TotalNodes
	syncLog.CreatedNodes = resp.CreatedNodes
	syncLog.UpdatedNodes = resp.UpdatedNodes
	syncLog.DeletedNodes = resp.DeletedNodes
	syncLog.TotalMembers = resp.TotalMembers
	syncLog.CreatedMembers = resp.CreatedMembers
	syncLog.UpdatedMembers = resp.UpdatedMembers
	syncLog.DeletedMembers = resp.DeletedMembers
	syncLog.SkippedUsers = resp.SkippedUsers
	_ = p.store.UpdateSyncLog(syncLog)

	// Broadcast tree update to frontend
	p.broadcastTreeUpdate("sync_complete", nil)

	return resp
}

// resolveMMUser attempts to find a Mattermost user ID for an external user.
func (p *Plugin) resolveMMUser(strategy, source string, payload *pluginmodel.SyncMemberPayload) (string, string) {
	// Step 1: Check user mapping table
	mapping, err := p.store.GetUserMappingByExternalID(source, payload.ExternalUserID)
	if err == nil && mapping != nil {
		return mapping.MmUserID, ""
	}

	if strategy == "mapping_only" {
		return "", "user_not_found: no mapping for " + payload.ExternalUserID
	}

	// Step 2: Match by email
	if payload.ExternalEmail != "" {
		user, appErr := p.API.GetUserByEmail(payload.ExternalEmail)
		if appErr == nil {
			p.autoCreateUserMapping(source, payload, user.Id)
			return user.Id, ""
		}
		user, appErr = p.API.GetUserByEmail(strings.ToLower(payload.ExternalEmail))
		if appErr == nil {
			p.autoCreateUserMapping(source, payload, user.Id)
			return user.Id, ""
		}
	}

	if strategy == "email_only" {
		return "", "user_not_found: no Mattermost user with email " + payload.ExternalEmail
	}

	// Step 3: Match by username
	if payload.ExternalUsername != "" {
		user, appErr := p.API.GetUserByUsername(payload.ExternalUsername)
		if appErr == nil {
			p.autoCreateUserMapping(source, payload, user.Id)
			return user.Id, ""
		}
	}

	return "", "user_not_found: no match for external_user_id=" + payload.ExternalUserID
}

// autoCreateUserMapping writes a mapping record when auto-matching succeeds.
func (p *Plugin) autoCreateUserMapping(source string, payload *pluginmodel.SyncMemberPayload, mmUserID string) {
	mapping := &pluginmodel.UserMapping{
		Source:           source,
		ExternalUserID:   payload.ExternalUserID,
		MmUserID:         mmUserID,
		ExternalUsername: payload.ExternalUsername,
		ExternalEmail:    payload.ExternalEmail,
	}
	if _, err := p.store.UpsertUserMapping(mapping); err != nil {
		p.API.LogWarn("failed to auto-create user mapping", "err", err)
	}
}

// topologicalSortNodes sorts nodes so parents always come before their children.
func topologicalSortNodes(nodes []pluginmodel.SyncNodePayload) []pluginmodel.SyncNodePayload {
	childrenMap := make(map[string][]int) // parentExtID -> indices of children
	roots := []int{}

	for i, n := range nodes {
		if n.ParentExternalID == "" {
			roots = append(roots, i)
		} else {
			childrenMap[n.ParentExternalID] = append(childrenMap[n.ParentExternalID], i)
		}
	}

	sorted := make([]pluginmodel.SyncNodePayload, 0, len(nodes))
	queue := roots
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nodes[idx])
		extID := nodes[idx].ExternalID
		queue = append(queue, childrenMap[extID]...)
	}

	// Append any remaining nodes (handles cycles or disconnected nodes gracefully)
	if len(sorted) < len(nodes) {
		added := make(map[int]bool)
		for _, n := range sorted {
			for i, orig := range nodes {
				if orig.ExternalID == n.ExternalID {
					added[i] = true
					break
				}
			}
		}
		for i, n := range nodes {
			if !added[i] {
				sorted = append(sorted, n)
			}
		}
	}

	return sorted
}
