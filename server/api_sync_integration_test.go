package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/mock"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// ----------------------------------------------------------------------------
// Minimal in-memory store for sync integration tests.
// Only the methods called by executeSyncRequest are implemented non-trivially.
// All other methods panic with "not implemented" to surface unexpected calls.
// ----------------------------------------------------------------------------

type syncTestStore struct {
	nodes        map[string]*pluginmodel.OrgNode     // (source+":"+externalID) → node
	members      map[string]*pluginmodel.OrgMember   // (nodeID+":"+userID) → member
	userMappings map[string]*pluginmodel.UserMapping // (source+":"+extUserID) → mapping
}

func newSyncTestStore() *syncTestStore {
	return &syncTestStore{
		nodes:        make(map[string]*pluginmodel.OrgNode),
		members:      make(map[string]*pluginmodel.OrgMember),
		userMappings: make(map[string]*pluginmodel.UserMapping),
	}
}

func syncNodeKey(source, externalID string) string   { return source + ":" + externalID }
func syncMemberKey(nodeID, userID string) string     { return nodeID + ":" + userID }
func syncMappingKey(source, extUserID string) string { return source + ":" + extUserID }

// --- NodeStore ---

func (s *syncTestStore) GetNodeByExternalID(source, externalID string) (*pluginmodel.OrgNode, error) {
	node := s.nodes[syncNodeKey(source, externalID)]
	if node == nil {
		return nil, nil
	}
	return node, nil
}

func (s *syncTestStore) GetNodesBySource(source string) ([]*pluginmodel.OrgNode, error) {
	result := make([]*pluginmodel.OrgNode, 0)
	for _, node := range s.nodes {
		if node.Source == source {
			result = append(result, node)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Depth != result[j].Depth {
			return result[i].Depth < result[j].Depth
		}
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *syncTestStore) UpsertNodeByExternalID(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error) {
	if node.ID == "" {
		node.ID = model.NewId()
	}
	s.nodes[syncNodeKey(node.Source, node.ExternalID)] = node
	return node, nil
}

func (s *syncTestStore) SoftDeleteNodesBySource(source string, excludeExternalIDs []string) (int, error) {
	exclude := make(map[string]bool, len(excludeExternalIDs))
	for _, id := range excludeExternalIDs {
		exclude[id] = true
	}
	deleted := 0
	for k, n := range s.nodes {
		if n.Source == source && !exclude[n.ExternalID] {
			delete(s.nodes, k)
			deleted++
		}
	}
	return deleted, nil
}

func (s *syncTestStore) CreateNode(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetNode(id string) (*pluginmodel.OrgNode, error) {
	for _, node := range s.nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return nil, nil
}
func (s *syncTestStore) UpdateNode(node *pluginmodel.OrgNode) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) DeleteNode(id string, strategy string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetChildNodes(parentID string) ([]*pluginmodel.OrgNode, error) {
	children := make([]*pluginmodel.OrgNode, 0)
	for _, node := range s.nodes {
		if node.ParentID == parentID {
			children = append(children, node)
		}
	}
	sort.SliceStable(children, func(i, j int) bool {
		if children[i].SortOrder != children[j].SortOrder {
			return children[i].SortOrder < children[j].SortOrder
		}
		return children[i].Name < children[j].Name
	})
	return children, nil
}
func (s *syncTestStore) GetSubTree(nodeID string) ([]*pluginmodel.OrgNode, error) {
	node, _ := s.GetNode(nodeID)
	if node == nil {
		return nil, nil
	}
	result := make([]*pluginmodel.OrgNode, 0)
	prefix := node.Path + "/"
	for _, candidate := range s.nodes {
		if strings.HasPrefix(candidate.Path, prefix) {
			result = append(result, candidate)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result, nil
}
func (s *syncTestStore) MoveNode(nodeID, newParentID string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetRootNodes() ([]*pluginmodel.OrgNode, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) SearchNodes(query string) ([]*pluginmodel.OrgNode, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetNodePath(nodeID string) ([]*pluginmodel.OrgNode, error) {
	node, _ := s.GetNode(nodeID)
	if node == nil {
		return nil, nil
	}
	parts := strings.Split(strings.TrimPrefix(node.Path, "/"), "/")
	result := make([]*pluginmodel.OrgNode, 0, len(parts))
	for _, id := range parts {
		pathNode, _ := s.GetNode(id)
		if pathNode != nil {
			result = append(result, pathNode)
		}
	}
	return result, nil
}
func (s *syncTestStore) ReorderNodes(parentID string, nodeIDs []string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetNodeMemberCount(nodeID string, recursive bool) (int64, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) SoftDeleteNodes(nodeIDs []string, now int64) error {
	panic("not implemented in sync test store")
}

// --- MemberStore ---

func (s *syncTestStore) UpsertMemberByExternalID(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	if member.ID == "" {
		member.ID = model.NewId()
	}
	s.members[syncMemberKey(member.NodeID, member.UserID)] = member
	return member, nil
}

func (s *syncTestStore) UpsertMemberByNodeAndUser(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	if member.ID == "" {
		member.ID = model.NewId()
	}
	s.members[syncMemberKey(member.NodeID, member.UserID)] = member
	return member, nil
}

func (s *syncTestStore) GetMemberRole(nodeID, userID string) (*pluginmodel.OrgMember, error) {
	return s.members[syncMemberKey(nodeID, userID)], nil
}

func (s *syncTestStore) SoftDeleteMembersBySource(source string, excludeExternalIDs []string) (int, error) {
	exclude := make(map[string]bool, len(excludeExternalIDs))
	for _, id := range excludeExternalIDs {
		exclude[id] = true
	}
	deleted := 0
	for k, m := range s.members {
		if m.Source == source && !exclude[m.ExternalID] {
			delete(s.members, k)
			deleted++
		}
	}
	return deleted, nil
}

func (s *syncTestStore) AddMember(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) RemoveMember(nodeID, userID string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetMembers(nodeID string, page, perPage int) ([]*pluginmodel.OrgMemberWithUser, error) {
	return s.GetMembersForNodes([]string{nodeID}, page, perPage)
}
func (s *syncTestStore) GetMembersForNodes(nodeIDs []string, page, perPage int) ([]*pluginmodel.OrgMemberWithUser, error) {
	nodeSet := make(map[string]bool, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		nodeSet[nodeID] = true
	}
	result := make([]*pluginmodel.OrgMemberWithUser, 0)
	for _, member := range s.members {
		if !nodeSet[member.NodeID] {
			continue
		}
		result = append(result, &pluginmodel.OrgMemberWithUser{
			OrgMember: member,
			Username:  member.UserID,
			Email:     member.UserID + "@example.com",
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].NodeID != result[j].NodeID {
			return result[i].NodeID < result[j].NodeID
		}
		return result[i].UserID < result[j].UserID
	})
	start := page * perPage
	if start >= len(result) {
		return []*pluginmodel.OrgMemberWithUser{}, nil
	}
	end := start + perPage
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], nil
}
func (s *syncTestStore) GetAllMembersByNodeID(nodeID string) ([]*pluginmodel.OrgMember, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetUserNodes(userID string) ([]*pluginmodel.OrgNode, error) {
	result := make([]*pluginmodel.OrgNode, 0)
	seen := map[string]bool{}
	for _, member := range s.members {
		if member.UserID != userID {
			continue
		}
		node, _ := s.GetNode(member.NodeID)
		if node == nil || seen[node.ID] {
			continue
		}
		seen[node.ID] = true
		result = append(result, node)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Depth != result[j].Depth {
			return result[i].Depth < result[j].Depth
		}
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}
func (s *syncTestStore) UpdateMemberRole(nodeID, userID, role string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) UpdateMemberPosition(nodeID, userID, position string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) SearchMembers(query string, page, perPage int) ([]*pluginmodel.SearchResult, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) IsMember(nodeID, userID string) (bool, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) ReorderMembers(nodeID string, userIDs []string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) SoftDeleteMembersByNodeID(nodeID string, now int64) (int, error) {
	panic("not implemented in sync test store")
}

// --- UserMappingStore ---

func (s *syncTestStore) UpsertUserMapping(mapping *pluginmodel.UserMapping) (*pluginmodel.UserMapping, error) {
	s.userMappings[syncMappingKey(mapping.Source, mapping.ExternalUserID)] = mapping
	return mapping, nil
}

func (s *syncTestStore) GetUserMappingByExternalID(source, externalUserID string) (*pluginmodel.UserMapping, error) {
	return s.userMappings[syncMappingKey(source, externalUserID)], nil
}

func (s *syncTestStore) GetUserMappingsBySource(source string, page, perPage int) ([]*pluginmodel.UserMapping, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetUserMappingsByMmUserID(mmUserID string) ([]*pluginmodel.UserMapping, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) DeleteUserMapping(source, externalUserID string) error {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) DeleteUserMappingsBySource(source string) error {
	panic("not implemented in sync test store")
}

// --- SyncStore ---

func (s *syncTestStore) CreateSyncLog(log *pluginmodel.SyncLog) (*pluginmodel.SyncLog, error) {
	log.ID = model.NewId()
	return log, nil
}

func (s *syncTestStore) UpdateSyncLog(log *pluginmodel.SyncLog) error { return nil }

func (s *syncTestStore) GetSyncLog(id string) (*pluginmodel.SyncLog, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetSyncLogs(source string, page, perPage int) ([]*pluginmodel.SyncLog, error) {
	panic("not implemented in sync test store")
}
func (s *syncTestStore) GetLatestSyncLog(source string) (*pluginmodel.SyncLog, error) {
	panic("not implemented in sync test store")
}

// --- AuditStore ---

func (s *syncTestStore) LogAction(log *pluginmodel.AuditLog) error { return nil }
func (s *syncTestStore) GetAuditLogs(targetType, targetID string, page, perPage int) ([]*pluginmodel.AuditLog, error) {
	panic("not implemented in sync test store")
}

// --- Close ---

func (s *syncTestStore) Close() error { return nil }

// ----------------------------------------------------------------------------
// Helper: build a Plugin with plugintest.API mock and in-memory test store.
// ----------------------------------------------------------------------------

// newSyncAPIandStore creates a plugintest.API mock pre-configured with the
// expectations needed for executeSyncRequest (PublishWebSocketEvent no-op).
// The caller can add further On() expectations before calling executeSyncRequest.
func newSyncAPIandStore() (*plugintest.API, *syncTestStore) {
	apiMock := &plugintest.API{}
	// broadcastTreeUpdate always calls PublishWebSocketEvent — stub it out.
	apiMock.On("PublishWebSocketEvent",
		mock.AnythingOfType("string"),
		mock.Anything,
		mock.Anything,
	).Return()
	ts := newSyncTestStore()
	return apiMock, ts
}

func buildSyncPlugin(apiMock *plugintest.API, ts *syncTestStore, strategy string) *Plugin {
	p := &Plugin{
		configuration: &configuration{
			SyncUserMatchStrategy: strategy,
		},
		store: ts,
	}
	p.SetAPI(apiMock)
	return p
}

// ----------------------------------------------------------------------------
// Integration tests — multi-source sync scenarios
// ----------------------------------------------------------------------------

// TestSyncMultiSourceIsolation verifies that a full sync from source A does not
// delete nodes belonging to source B.
func TestSyncMultiSourceIsolation(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")

	// Pre-seed: source "oa" already has a node
	oaNodeID := model.NewId()
	ts.nodes[syncNodeKey("oa", "oa-root")] = &pluginmodel.OrgNode{
		ID:         oaNodeID,
		Name:       "OA Root",
		Source:     "oa",
		ExternalID: "oa-root",
		Metadata:   "{}",
	}

	// Source "hr" does a full sync — its own nodes only
	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "hr-root", Name: "HR Root"},
		},
	})

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got %q; errors: %v", resp.Status, resp.Errors)
	}
	if resp.CreatedNodes != 1 {
		t.Errorf("expected 1 created node, got %d", resp.CreatedNodes)
	}
	if resp.DeletedNodes != 0 {
		t.Errorf("expected 0 deleted nodes (no hr stale nodes), got %d", resp.DeletedNodes)
	}
	// OA node must survive — source isolation
	if _, ok := ts.nodes[syncNodeKey("oa", "oa-root")]; !ok {
		t.Error("oa-root was incorrectly deleted by hr full sync — source isolation broken")
	}
}

// TestSyncFullSyncCleansStaleNodes verifies that a full sync deletes previously-
// synced nodes from the same source that are absent from the new payload.
func TestSyncFullSyncCleansStaleNodes(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")

	// First sync: three nodes from "hr"
	p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "hr-a", Name: "HR A"},
			{ExternalID: "hr-b", Name: "HR B"},
			{ExternalID: "hr-c", Name: "HR C"},
		},
	})
	if len(ts.nodes) != 3 {
		t.Fatalf("expected 3 nodes after first sync, got %d", len(ts.nodes))
	}

	// Second sync: only hr-a — hr-b and hr-c should be cleaned up
	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "hr-a", Name: "HR A"},
		},
	})

	if resp.DeletedNodes != 2 {
		t.Errorf("expected 2 deleted nodes, got %d", resp.DeletedNodes)
	}
	if resp.UpdatedNodes != 1 {
		t.Errorf("expected 1 updated node (hr-a re-synced), got %d", resp.UpdatedNodes)
	}
	if len(ts.nodes) != 1 {
		t.Errorf("expected 1 node remaining, got %d", len(ts.nodes))
	}
}

// TestSyncIncrementalDoesNotDelete verifies that an incremental sync never
// deletes nodes absent from the payload.
func TestSyncIncrementalDoesNotDelete(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")

	// Pre-seed two nodes from "hr"
	ts.nodes[syncNodeKey("hr", "hr-a")] = &pluginmodel.OrgNode{
		ID: model.NewId(), Name: "HR A", Source: "hr", ExternalID: "hr-a", Metadata: "{}",
	}
	ts.nodes[syncNodeKey("hr", "hr-b")] = &pluginmodel.OrgNode{
		ID: model.NewId(), Name: "HR B", Source: "hr", ExternalID: "hr-b", Metadata: "{}",
	}

	// Incremental sync: only hr-a in payload — hr-b must survive
	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "incremental",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "hr-a", Name: "HR A Updated"},
		},
	})

	if resp.Status != "success" {
		t.Errorf("expected success, got %q; errors: %v", resp.Status, resp.Errors)
	}
	if resp.DeletedNodes != 0 {
		t.Errorf("incremental sync should not delete any nodes, got %d deletions", resp.DeletedNodes)
	}
	if len(ts.nodes) != 2 {
		t.Errorf("both nodes should survive incremental sync, got %d nodes", len(ts.nodes))
	}
}

// TestSyncHierarchyReversedOrder verifies that nodes sent in child-before-parent
// order are created successfully due to topological sorting.
func TestSyncHierarchyReversedOrder(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")

	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "ldap",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			// Intentionally reversed: grandchild first, root last
			{ExternalID: "grandchild", Name: "Grandchild", ParentExternalID: "child"},
			{ExternalID: "child", Name: "Child", ParentExternalID: "root"},
			{ExternalID: "root", Name: "Root"},
		},
	})

	if resp.Status != "success" {
		t.Errorf("expected success, got %q; errors: %v", resp.Status, resp.Errors)
	}
	if resp.CreatedNodes != 3 {
		t.Errorf("expected 3 created nodes, got %d", resp.CreatedNodes)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", resp.Errors)
	}
}

// TestSyncUserMappingResolution verifies that pre-existing user mappings are
// used to resolve members without needing the plugin API user lookup.
func TestSyncUserMappingResolution(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_only")

	mmUserID := model.NewId()
	nodeID := model.NewId()
	ts.nodes[syncNodeKey("hr", "hr-root")] = &pluginmodel.OrgNode{
		ID: nodeID, Name: "HR Root", Source: "hr", ExternalID: "hr-root", Metadata: "{}",
	}
	ts.userMappings[syncMappingKey("hr", "ext-user-001")] = &pluginmodel.UserMapping{
		Source:         "hr",
		ExternalUserID: "ext-user-001",
		MmUserID:       mmUserID,
	}

	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "incremental",
		Members: []pluginmodel.SyncMemberPayload{
			{ExternalID: "mem-001", ExternalUserID: "ext-user-001", NodeExternalID: "hr-root", Role: "member"},
		},
	})

	if resp.Status != "success" {
		t.Errorf("expected success, got %q; errors: %v; skipped: %v", resp.Status, resp.Errors, resp.SkippedDetails)
	}
	if resp.CreatedMembers != 1 {
		t.Errorf("expected 1 created member, got %d", resp.CreatedMembers)
	}
	if resp.SkippedUsers != 0 {
		t.Errorf("expected 0 skipped users, got %d: %v", resp.SkippedUsers, resp.SkippedDetails)
	}
	if _, ok := ts.members[syncMemberKey(nodeID, mmUserID)]; !ok {
		t.Error("expected member to be stored, but not found")
	}
}

// TestSyncEmailFallbackResolution verifies that when no mapping exists, member
// resolution falls back to email lookup via the plugin API.
func TestSyncEmailFallbackResolution(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	mmUserID := model.NewId()

	apiMock.On("GetUserByEmail", "alice@example.com").Return(
		&model.User{Id: mmUserID, Email: "alice@example.com", Username: "alice"},
		(*model.AppError)(nil),
	)

	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	nodeID := model.NewId()
	ts.nodes[syncNodeKey("hr", "hr-root")] = &pluginmodel.OrgNode{
		ID: nodeID, Name: "HR Root", Source: "hr", ExternalID: "hr-root", Metadata: "{}",
	}

	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "incremental",
		Members: []pluginmodel.SyncMemberPayload{
			{
				ExternalID:     "mem-alice",
				ExternalUserID: "hr-alice-001",
				ExternalEmail:  "alice@example.com",
				NodeExternalID: "hr-root",
				Role:           "member",
			},
		},
	})

	if resp.Status != "success" {
		t.Errorf("expected success, got %q; errors: %v; skipped: %v", resp.Status, resp.Errors, resp.SkippedDetails)
	}
	if resp.CreatedMembers != 1 {
		t.Errorf("expected 1 created member via email fallback, got %d", resp.CreatedMembers)
	}
	// Auto-mapping should have been created
	if ts.userMappings[syncMappingKey("hr", "hr-alice-001")] == nil {
		t.Error("expected auto-mapping to be created for email-matched user")
	}
}

// TestSyncUnresolvableUserPartialSuccess verifies that when a user cannot be
// resolved, the sync produces "partial_success" with a SkippedUsers entry.
func TestSyncUnresolvableUserPartialSuccess(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	// All user lookups fail
	apiMock.On("GetUserByEmail", mock.AnythingOfType("string")).Return(
		(*model.User)(nil),
		&model.AppError{Message: "not found"},
	)
	apiMock.On("GetUserByUsername", mock.AnythingOfType("string")).Return(
		(*model.User)(nil),
		&model.AppError{Message: "not found"},
	)

	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	nodeID := model.NewId()
	ts.nodes[syncNodeKey("hr", "hr-root")] = &pluginmodel.OrgNode{
		ID: nodeID, Name: "HR Root", Source: "hr", ExternalID: "hr-root", Metadata: "{}",
	}

	resp := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "hr",
		SyncType: "incremental",
		Members: []pluginmodel.SyncMemberPayload{
			{
				ExternalID:       "mem-ghost",
				ExternalUserID:   "ext-nobody",
				ExternalEmail:    "nobody@example.com",
				ExternalUsername: "nobody",
				NodeExternalID:   "hr-root",
				Role:             "member",
			},
		},
	})

	if resp.Status != "partial_success" {
		t.Errorf("expected 'partial_success', got %q", resp.Status)
	}
	if resp.SkippedUsers != 1 {
		t.Errorf("expected 1 skipped user, got %d", resp.SkippedUsers)
	}
}

// TestSyncMultiSourceParallelNodes verifies that two sources can sync their own
// node trees independently with no cross-contamination of deletions or counts.
func TestSyncMultiSourceParallelNodes(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")

	// Source A: 3 nodes
	respA := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "source_a",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "a1", Name: "A1"},
			{ExternalID: "a2", Name: "A2", ParentExternalID: "a1"},
			{ExternalID: "a3", Name: "A3", ParentExternalID: "a1"},
		},
	})
	if respA.CreatedNodes != 3 {
		t.Errorf("source_a: expected 3 created, got %d; errors: %v", respA.CreatedNodes, respA.Errors)
	}

	// Source B: 2 nodes
	respB := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "source_b",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "b1", Name: "B1"},
			{ExternalID: "b2", Name: "B2", ParentExternalID: "b1"},
		},
	})
	if respB.CreatedNodes != 2 {
		t.Errorf("source_b: expected 2 created, got %d; errors: %v", respB.CreatedNodes, respB.Errors)
	}
	if len(ts.nodes) != 5 {
		t.Errorf("expected 5 total nodes (3+2), got %d", len(ts.nodes))
	}

	// Source A drops a3 — only source_a's stale node should be deleted
	respA2 := p.executeSyncRequest(&pluginmodel.SyncRequest{
		Source:   "source_a",
		SyncType: "full",
		Nodes: []pluginmodel.SyncNodePayload{
			{ExternalID: "a1", Name: "A1"},
			{ExternalID: "a2", Name: "A2", ParentExternalID: "a1"},
		},
	})
	if respA2.DeletedNodes != 1 {
		t.Errorf("source_a second sync: expected 1 deleted (a3), got %d", respA2.DeletedNodes)
	}

	// source_b nodes must still be intact
	if _, ok := ts.nodes[syncNodeKey("source_b", "b1")]; !ok {
		t.Error("source_b node b1 was incorrectly deleted by source_a sync")
	}
	if _, ok := ts.nodes[syncNodeKey("source_b", "b2")]; !ok {
		t.Error("source_b node b2 was incorrectly deleted by source_a sync")
	}
	if len(ts.nodes) != 4 {
		t.Errorf("expected 4 nodes after second source_a sync, got %d", len(ts.nodes))
	}
}

func TestHandleListSyncNodes(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	p.initializeAPI()

	ts.nodes[syncNodeKey("hr", "root")] = &pluginmodel.OrgNode{
		ID:         "node-root",
		Name:       "Root",
		Path:       "/node-root",
		Depth:      0,
		Source:     "hr",
		ExternalID: "root",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "child")] = &pluginmodel.OrgNode{
		ID:         "node-child",
		Name:       "Child",
		ParentID:   "node-root",
		Path:       "/node-root/node-child",
		Depth:      1,
		Source:     "hr",
		ExternalID: "child",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "leaf")] = &pluginmodel.OrgNode{
		ID:         "node-leaf",
		Name:       "Leaf",
		ParentID:   "node-child",
		Path:       "/node-root/node-child/node-leaf",
		Depth:      2,
		Source:     "hr",
		ExternalID: "leaf",
		Metadata:   "{}",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/nodes?source=hr", nil)
	req.Header.Set("Authorization", "Bearer sync-token")
	rec := httptest.NewRecorder()

	p.configuration = &configuration{SyncAPIToken: "sync-token"}
	p.ServeHTTP(nil, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp pluginmodel.SyncNodeListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Source != "hr" {
		t.Fatalf("expected source hr, got %q", resp.Source)
	}
	if len(resp.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(resp.Nodes))
	}
	if resp.Nodes[1].ParentExternalID != "root" {
		t.Fatalf("expected child parent external id root, got %q", resp.Nodes[1].ParentExternalID)
	}
}

func TestHandleListSyncNodesWithDepthFilter(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	p.initializeAPI()

	ts.nodes[syncNodeKey("hr", "root")] = &pluginmodel.OrgNode{
		ID:         "node-root",
		Name:       "Root",
		Path:       "/node-root",
		Depth:      0,
		Source:     "hr",
		ExternalID: "root",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "child-a")] = &pluginmodel.OrgNode{
		ID:         "node-child-a",
		Name:       "ChildA",
		ParentID:   "node-root",
		Path:       "/node-root/node-child-a",
		Depth:      1,
		Source:     "hr",
		ExternalID: "child-a",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "child-b")] = &pluginmodel.OrgNode{
		ID:         "node-child-b",
		Name:       "ChildB",
		ParentID:   "node-root",
		Path:       "/node-root/node-child-b",
		Depth:      1,
		Source:     "hr",
		ExternalID: "child-b",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "leaf")] = &pluginmodel.OrgNode{
		ID:         "node-leaf",
		Name:       "Leaf",
		ParentID:   "node-child-a",
		Path:       "/node-root/node-child-a/node-leaf",
		Depth:      2,
		Source:     "hr",
		ExternalID: "leaf",
		Metadata:   "{}",
	}

	testCases := []struct {
		name         string
		path         string
		expectCount  int
		expectDepths []int
		expectExtIDs []string
	}{
		{
			name:         "exact root depth",
			path:         "/api/v1/sync/nodes?source=hr&depth=0",
			expectCount:  1,
			expectDepths: []int{0},
			expectExtIDs: []string{"root"},
		},
		{
			name:         "exact first depth",
			path:         "/api/v1/sync/nodes?source=hr&depth=1",
			expectCount:  2,
			expectDepths: []int{1, 1},
			expectExtIDs: []string{"child-a", "child-b"},
		},
		{
			name:         "max depth cumulative",
			path:         "/api/v1/sync/nodes?source=hr&max_depth=1",
			expectCount:  3,
			expectDepths: []int{0, 1, 1},
			expectExtIDs: []string{"root", "child-a", "child-b"},
		},
		{
			name:         "parent relative exact depth",
			path:         "/api/v1/sync/nodes?source=hr&parent_external_id=root&depth=0",
			expectCount:  2,
			expectDepths: []int{1, 1},
			expectExtIDs: []string{"child-a", "child-b"},
		},
		{
			name:         "parent relative max depth",
			path:         "/api/v1/sync/nodes?source=hr&parent_external_id=root&max_depth=1",
			expectCount:  3,
			expectDepths: []int{1, 1, 2},
			expectExtIDs: []string{"child-a", "child-b", "leaf"},
		},
	}

	p.configuration = &configuration{SyncAPIToken: "sync-token"}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.Header.Set("Authorization", "Bearer sync-token")
			rec := httptest.NewRecorder()
			p.ServeHTTP(nil, rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
			}

			var resp pluginmodel.SyncNodeListResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if len(resp.Nodes) != tc.expectCount {
				t.Fatalf("expected %d nodes, got %d", tc.expectCount, len(resp.Nodes))
			}
			for idx, node := range resp.Nodes {
				if node.Depth != tc.expectDepths[idx] {
					t.Fatalf("expected node depth %d, got %d", tc.expectDepths[idx], node.Depth)
				}
				if node.ExternalID != tc.expectExtIDs[idx] {
					t.Fatalf("expected external id %s, got %s", tc.expectExtIDs[idx], node.ExternalID)
				}
			}
		})
	}
}

func TestHandleGetSyncNodeMembersRecursive(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	p.initializeAPI()

	ts.nodes[syncNodeKey("hr", "root")] = &pluginmodel.OrgNode{
		ID:         "node-root",
		Name:       "Root",
		Path:       "/node-root",
		Depth:      0,
		Source:     "hr",
		ExternalID: "root",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "child")] = &pluginmodel.OrgNode{
		ID:         "node-child",
		Name:       "Child",
		ParentID:   "node-root",
		Path:       "/node-root/node-child",
		Depth:      1,
		Source:     "hr",
		ExternalID: "child",
		Metadata:   "{}",
	}
	ts.members[syncMemberKey("node-root", "user-root")] = &pluginmodel.OrgMember{
		ID:         "mem-root",
		NodeID:     "node-root",
		UserID:     "user-root",
		Source:     "hr",
		ExternalID: "m-root",
	}
	ts.members[syncMemberKey("node-child", "user-child")] = &pluginmodel.OrgMember{
		ID:         "mem-child",
		NodeID:     "node-child",
		UserID:     "user-child",
		Source:     "hr",
		ExternalID: "m-child",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/nodes/root/members?source=hr&recursive=true", nil)
	req.Header.Set("Authorization", "Bearer sync-token")
	rec := httptest.NewRecorder()

	p.configuration = &configuration{SyncAPIToken: "sync-token"}
	p.ServeHTTP(nil, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp pluginmodel.SyncNodeMembersResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Recursive {
		t.Fatal("expected recursive flag true")
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	memberNodes := map[string]bool{}
	for _, member := range resp.Members {
		memberNodes[member.NodeExternalID] = true
	}
	if !memberNodes["root"] || !memberNodes["child"] {
		t.Fatalf("expected members from root and child nodes, got %+v", memberNodes)
	}
}

func TestHandleGetSyncUserNodes(t *testing.T) {
	apiMock, ts := newSyncAPIandStore()
	p := buildSyncPlugin(apiMock, ts, "mapping_email_username")
	p.initializeAPI()

	ts.userMappings[syncMappingKey("hr", "HR-EMP-001")] = &pluginmodel.UserMapping{
		Source:         "hr",
		ExternalUserID: "HR-EMP-001",
		MmUserID:       "mm-user-001",
	}
	ts.nodes[syncNodeKey("hr", "root")] = &pluginmodel.OrgNode{
		ID:         "node-root",
		Name:       "Root",
		Path:       "/node-root",
		Depth:      0,
		Source:     "hr",
		ExternalID: "root",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("hr", "child")] = &pluginmodel.OrgNode{
		ID:         "node-child",
		Name:       "Child",
		ParentID:   "node-root",
		Path:       "/node-root/node-child",
		Depth:      1,
		Source:     "hr",
		ExternalID: "child",
		Metadata:   "{}",
	}
	ts.nodes[syncNodeKey("oa", "other")] = &pluginmodel.OrgNode{
		ID:         "node-other",
		Name:       "Other",
		Path:       "/node-other",
		Depth:      0,
		Source:     "oa",
		ExternalID: "other",
		Metadata:   "{}",
	}
	ts.members[syncMemberKey("node-root", "mm-user-001")] = &pluginmodel.OrgMember{
		ID:     "member-root",
		NodeID: "node-root",
		UserID: "mm-user-001",
		Source: "hr",
	}
	ts.members[syncMemberKey("node-child", "mm-user-001")] = &pluginmodel.OrgMember{
		ID:     "member-child",
		NodeID: "node-child",
		UserID: "mm-user-001",
		Source: "hr",
	}
	ts.members[syncMemberKey("node-other", "mm-user-001")] = &pluginmodel.OrgMember{
		ID:     "member-other",
		NodeID: "node-other",
		UserID: "mm-user-001",
		Source: "oa",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/users/HR-EMP-001/nodes?source=hr", nil)
	req.Header.Set("Authorization", "Bearer sync-token")
	rec := httptest.NewRecorder()

	p.configuration = &configuration{SyncAPIToken: "sync-token"}
	p.ServeHTTP(nil, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var resp pluginmodel.SyncUserNodesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.MmUserID != "mm-user-001" {
		t.Fatalf("expected mapped MM user mm-user-001, got %q", resp.MmUserID)
	}
	if resp.Total != 2 {
		t.Fatalf("expected 2 hr nodes, got %d", resp.Total)
	}
	if resp.Nodes[1].ParentExternalID != "root" {
		t.Fatalf("expected child parent external id root, got %q", resp.Nodes[1].ParentExternalID)
	}
}
