package store

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const schemaVersionKey = "org_directory_schema_version"

// SQLStore is the SQL-backed implementation of the Store interface.
type SQLStore struct {
	client     *pluginapi.Client
	api        plugin.API
	masterDB   *sql.DB
	replicaDB  *sql.DB
	driverName string
}

// NewSQLStore creates a new SQLStore and runs all pending migrations.
func NewSQLStore(client *pluginapi.Client, api plugin.API) (*SQLStore, error) {
	masterDB, err := client.Store.GetMasterDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get master DB: %w", err)
	}

	replicaDB, err := client.Store.GetReplicaDB()
	if err != nil {
		// Fall back to master if no replica is configured
		replicaDB = masterDB
	}

	s := &SQLStore{
		client:     client,
		api:        api,
		masterDB:   masterDB,
		replicaDB:  replicaDB,
		driverName: client.Store.DriverName(),
	}

	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

// Close is a no-op since the DB connection is managed by the Mattermost server.
func (s *SQLStore) Close() error {
	return nil
}

// migrate runs all pending schema migrations.
func (s *SQLStore) migrate() error {
	migrations := []struct {
		version int
		sql     string
	}{
		{1, s.migrationV1CreateTables()},
		{2, s.migrationV2AddIndexes()},
		{3, s.migrationV3AddSyncQueryIndexes()},
	}

	currentVersion := s.getSchemaVersion()
	for _, m := range migrations {
		if m.version > currentVersion {
			if _, err := s.masterDB.Exec(m.sql); err != nil {
				return fmt.Errorf("migration v%d failed: %w", m.version, err)
			}
			s.setSchemaVersion(m.version)
		}
	}
	return nil
}

func (s *SQLStore) getSchemaVersion() int {
	data, appErr := s.api.KVGet(schemaVersionKey)
	if appErr != nil || data == nil {
		return 0
	}
	v, _ := strconv.Atoi(string(data))
	return v
}

func (s *SQLStore) setSchemaVersion(version int) {
	_ = s.api.KVSet(schemaVersionKey, []byte(strconv.Itoa(version)))
}

// migrationV1CreateTables returns the SQL for creating all plugin tables.
func (s *SQLStore) migrationV1CreateTables() string {
	return `
CREATE TABLE IF NOT EXISTS org_directory_nodes (
    id          VARCHAR(26) PRIMARY KEY,
    name        VARCHAR(256) NOT NULL,
    parent_id   VARCHAR(26) NOT NULL DEFAULT '',
    path        VARCHAR(2048) NOT NULL DEFAULT '',
    depth       INTEGER NOT NULL DEFAULT 0,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    icon        VARCHAR(256) NOT NULL DEFAULT '',
    metadata    TEXT NOT NULL DEFAULT '{}',
    source      VARCHAR(64) NOT NULL DEFAULT 'local',
    external_id VARCHAR(256) NOT NULL DEFAULT '',
    create_at   BIGINT NOT NULL,
    update_at   BIGINT NOT NULL,
    delete_at   BIGINT NOT NULL DEFAULT 0,
    creator_id  VARCHAR(26) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS org_directory_members (
    id          VARCHAR(26) PRIMARY KEY,
    node_id     VARCHAR(26) NOT NULL,
    user_id     VARCHAR(26) NOT NULL,
    role        VARCHAR(64) NOT NULL DEFAULT 'member',
    position    VARCHAR(256) NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    source      VARCHAR(64) NOT NULL DEFAULT 'local',
    external_id VARCHAR(256) NOT NULL DEFAULT '',
    create_at   BIGINT NOT NULL,
    update_at   BIGINT NOT NULL,
    delete_at   BIGINT NOT NULL DEFAULT 0,
    UNIQUE(node_id, user_id)
);

CREATE TABLE IF NOT EXISTS org_directory_user_mappings (
    id                VARCHAR(26) PRIMARY KEY,
    source            VARCHAR(64) NOT NULL,
    external_user_id  VARCHAR(256) NOT NULL,
    mm_user_id        VARCHAR(26) NOT NULL,
    external_username VARCHAR(256) NOT NULL DEFAULT '',
    external_email    VARCHAR(256) NOT NULL DEFAULT '',
    create_at         BIGINT NOT NULL,
    update_at         BIGINT NOT NULL,
    delete_at         BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS org_directory_sync_logs (
    id              VARCHAR(26) PRIMARY KEY,
    source          VARCHAR(64) NOT NULL,
    sync_type       VARCHAR(32) NOT NULL,
    status          VARCHAR(32) NOT NULL,
    total_nodes     INTEGER NOT NULL DEFAULT 0,
    created_nodes   INTEGER NOT NULL DEFAULT 0,
    updated_nodes   INTEGER NOT NULL DEFAULT 0,
    deleted_nodes   INTEGER NOT NULL DEFAULT 0,
    total_members   INTEGER NOT NULL DEFAULT 0,
    created_members INTEGER NOT NULL DEFAULT 0,
    updated_members INTEGER NOT NULL DEFAULT 0,
    deleted_members INTEGER NOT NULL DEFAULT 0,
    skipped_users   INTEGER NOT NULL DEFAULT 0,
    error_message   TEXT NOT NULL DEFAULT '',
    details         TEXT NOT NULL DEFAULT '{}',
    started_at      BIGINT NOT NULL,
    finished_at     BIGINT NOT NULL DEFAULT 0,
    triggered_by    VARCHAR(26) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS org_directory_audit_log (
    id          VARCHAR(26) PRIMARY KEY,
    actor_id    VARCHAR(26) NOT NULL,
    action      VARCHAR(64) NOT NULL,
    target_type VARCHAR(32) NOT NULL,
    target_id   VARCHAR(26) NOT NULL,
    details     TEXT NOT NULL DEFAULT '{}',
    create_at   BIGINT NOT NULL
);
`
}

// migrationV2AddIndexes returns the SQL for creating indexes.
func (s *SQLStore) migrationV2AddIndexes() string {
	return `
CREATE INDEX IF NOT EXISTS idx_org_nodes_parent_id ON org_directory_nodes(parent_id);
CREATE INDEX IF NOT EXISTS idx_org_nodes_path ON org_directory_nodes(path);
CREATE INDEX IF NOT EXISTS idx_org_nodes_delete_at ON org_directory_nodes(delete_at);
CREATE INDEX IF NOT EXISTS idx_org_nodes_sort ON org_directory_nodes(parent_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_org_members_node_id ON org_directory_members(node_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON org_directory_members(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_delete_at ON org_directory_members(delete_at);
CREATE INDEX IF NOT EXISTS idx_org_members_role ON org_directory_members(node_id, role);
CREATE INDEX IF NOT EXISTS idx_org_members_source ON org_directory_members(source);
CREATE INDEX IF NOT EXISTS idx_user_map_mm_user ON org_directory_user_mappings(mm_user_id);
CREATE INDEX IF NOT EXISTS idx_user_map_source ON org_directory_user_mappings(source);
CREATE INDEX IF NOT EXISTS idx_sync_logs_source ON org_directory_sync_logs(source);
CREATE INDEX IF NOT EXISTS idx_sync_logs_status ON org_directory_sync_logs(status);
CREATE INDEX IF NOT EXISTS idx_sync_logs_time ON org_directory_sync_logs(started_at);
CREATE INDEX IF NOT EXISTS idx_org_audit_actor ON org_directory_audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_org_audit_target ON org_directory_audit_log(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_org_audit_time ON org_directory_audit_log(create_at);
`
}

// migrationV3AddSyncQueryIndexes returns additional indexes for sync query APIs.
func (s *SQLStore) migrationV3AddSyncQueryIndexes() string {
	return `
CREATE INDEX IF NOT EXISTS idx_org_nodes_source_delete_depth_sort_name ON org_directory_nodes(source, delete_at, depth, sort_order, name);
CREATE INDEX IF NOT EXISTS idx_org_nodes_source_external_delete ON org_directory_nodes(source, external_id, delete_at);
`
}

// db returns the appropriate database connection for reads.
func (s *SQLStore) db() *sql.DB {
	return s.replicaDB
}

// master returns the master database connection for writes.
func (s *SQLStore) master() *sql.DB {
	return s.masterDB
}
