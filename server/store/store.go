package store

import (
	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// Store aggregates all sub-store interfaces.
type Store interface {
	NodeStore
	MemberStore
	UserMappingStore
	SyncStore
	AuditStore
	Close() error
}

// NodeStore defines operations on organization nodes.
type NodeStore interface {
	CreateNode(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error)
	GetNode(id string) (*pluginmodel.OrgNode, error)
	UpdateNode(node *pluginmodel.OrgNode) error
	DeleteNode(id string, cascadeStrategy string) error
	GetChildNodes(parentID string) ([]*pluginmodel.OrgNode, error)
	GetSubTree(nodeID string) ([]*pluginmodel.OrgNode, error)
	MoveNode(nodeID, newParentID string) error
	GetRootNodes() ([]*pluginmodel.OrgNode, error)
	SearchNodes(query string) ([]*pluginmodel.OrgNode, error)
	GetNodePath(nodeID string) ([]*pluginmodel.OrgNode, error)
	ReorderNodes(parentID string, nodeIDs []string) error
	GetNodeMemberCount(nodeID string, recursive bool) (int64, error)
	// External sync
	GetNodeByExternalID(source, externalID string) (*pluginmodel.OrgNode, error)
	UpsertNodeByExternalID(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error)
	SoftDeleteNodesBySource(source string, excludeExternalIDs []string) (int, error)
	SoftDeleteNodes(nodeIDs []string, now int64) error
}

// MemberStore defines operations on node memberships.
type MemberStore interface {
	AddMember(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error)
	RemoveMember(nodeID, userID string) error
	GetMembers(nodeID string, page, perPage int) ([]*pluginmodel.OrgMemberWithUser, error)
	GetAllMembersByNodeID(nodeID string) ([]*pluginmodel.OrgMember, error)
	GetUserNodes(userID string) ([]*pluginmodel.OrgNode, error)
	UpdateMemberRole(nodeID, userID, role string) error
	UpdateMemberPosition(nodeID, userID, position string) error
	SearchMembers(query string, page, perPage int) ([]*pluginmodel.SearchResult, error)
	IsMember(nodeID, userID string) (bool, error)
	GetMemberRole(nodeID, userID string) (*pluginmodel.OrgMember, error)
	ReorderMembers(nodeID string, userIDs []string) error
	SoftDeleteMembersByNodeID(nodeID string, now int64) (int, error)
	// External sync
	UpsertMemberByExternalID(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error)
	UpsertMemberByNodeAndUser(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error)
	SoftDeleteMembersBySource(source string, excludeExternalIDs []string) (int, error)
}

// UserMappingStore defines operations on external user mappings.
type UserMappingStore interface {
	UpsertUserMapping(mapping *pluginmodel.UserMapping) (*pluginmodel.UserMapping, error)
	GetUserMappingByExternalID(source, externalUserID string) (*pluginmodel.UserMapping, error)
	GetUserMappingsBySource(source string, page, perPage int) ([]*pluginmodel.UserMapping, error)
	GetUserMappingsByMmUserID(mmUserID string) ([]*pluginmodel.UserMapping, error)
	DeleteUserMapping(source, externalUserID string) error
	DeleteUserMappingsBySource(source string) error
}

// SyncStore defines operations on sync task logs.
type SyncStore interface {
	CreateSyncLog(log *pluginmodel.SyncLog) (*pluginmodel.SyncLog, error)
	UpdateSyncLog(log *pluginmodel.SyncLog) error
	GetSyncLog(id string) (*pluginmodel.SyncLog, error)
	GetSyncLogs(source string, page, perPage int) ([]*pluginmodel.SyncLog, error)
	GetLatestSyncLog(source string) (*pluginmodel.SyncLog, error)
}

// AuditStore defines operations on audit logs.
type AuditStore interface {
	LogAction(log *pluginmodel.AuditLog) error
	GetAuditLogs(targetType, targetID string, page, perPage int) ([]*pluginmodel.AuditLog, error)
}
