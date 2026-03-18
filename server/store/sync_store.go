package store

import (
	"database/sql"
	"time"

	mmmodel "github.com/mattermost/mattermost/server/public/model"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// CreateSyncLog creates a new sync log entry.
func (s *SQLStore) CreateSyncLog(log *pluginmodel.SyncLog) (*pluginmodel.SyncLog, error) {
	log.ID = mmmodel.NewId()
	_, err := s.master().Exec(`
		INSERT INTO org_directory_sync_logs
		(id, source, sync_type, status, total_nodes, created_nodes, updated_nodes, deleted_nodes,
		 total_members, created_members, updated_members, deleted_members, skipped_users,
		 error_message, details, started_at, finished_at, triggered_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		log.ID, log.Source, log.SyncType, log.Status,
		log.TotalNodes, log.CreatedNodes, log.UpdatedNodes, log.DeletedNodes,
		log.TotalMembers, log.CreatedMembers, log.UpdatedMembers, log.DeletedMembers,
		log.SkippedUsers, log.ErrorMessage, log.Details,
		log.StartedAt, log.FinishedAt, log.TriggeredBy,
	)
	return log, err
}

// UpdateSyncLog updates an existing sync log.
func (s *SQLStore) UpdateSyncLog(log *pluginmodel.SyncLog) error {
	_, err := s.master().Exec(`
		UPDATE org_directory_sync_logs SET
		  status=$1, total_nodes=$2, created_nodes=$3, updated_nodes=$4, deleted_nodes=$5,
		  total_members=$6, created_members=$7, updated_members=$8, deleted_members=$9,
		  skipped_users=$10, error_message=$11, details=$12, finished_at=$13
		WHERE id=$14`,
		log.Status,
		log.TotalNodes, log.CreatedNodes, log.UpdatedNodes, log.DeletedNodes,
		log.TotalMembers, log.CreatedMembers, log.UpdatedMembers, log.DeletedMembers,
		log.SkippedUsers, log.ErrorMessage, log.Details, log.FinishedAt,
		log.ID,
	)
	return err
}

// GetSyncLog retrieves a sync log by ID.
func (s *SQLStore) GetSyncLog(id string) (*pluginmodel.SyncLog, error) {
	log := &pluginmodel.SyncLog{}
	err := s.db().QueryRow(`
		SELECT id, source, sync_type, status, total_nodes, created_nodes, updated_nodes, deleted_nodes,
		       total_members, created_members, updated_members, deleted_members, skipped_users,
		       error_message, details, started_at, finished_at, triggered_by
		FROM org_directory_sync_logs WHERE id=$1`, id).Scan(
		&log.ID, &log.Source, &log.SyncType, &log.Status,
		&log.TotalNodes, &log.CreatedNodes, &log.UpdatedNodes, &log.DeletedNodes,
		&log.TotalMembers, &log.CreatedMembers, &log.UpdatedMembers, &log.DeletedMembers,
		&log.SkippedUsers, &log.ErrorMessage, &log.Details,
		&log.StartedAt, &log.FinishedAt, &log.TriggeredBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return log, err
}

// GetSyncLogs retrieves paginated sync logs, optionally filtered by source.
func (s *SQLStore) GetSyncLogs(source string, page, perPage int) ([]*pluginmodel.SyncLog, error) {
	offset := page * perPage
	var rows *sql.Rows
	var err error
	if source != "" {
		rows, err = s.db().Query(`
			SELECT id, source, sync_type, status, total_nodes, created_nodes, updated_nodes, deleted_nodes,
			       total_members, created_members, updated_members, deleted_members, skipped_users,
			       error_message, details, started_at, finished_at, triggered_by
			FROM org_directory_sync_logs WHERE source=$1
			ORDER BY started_at DESC LIMIT $2 OFFSET $3`, source, perPage, offset)
	} else {
		rows, err = s.db().Query(`
			SELECT id, source, sync_type, status, total_nodes, created_nodes, updated_nodes, deleted_nodes,
			       total_members, created_members, updated_members, deleted_members, skipped_users,
			       error_message, details, started_at, finished_at, triggered_by
			FROM org_directory_sync_logs
			ORDER BY started_at DESC LIMIT $1 OFFSET $2`, perPage, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSyncLogs(rows)
}

// GetLatestSyncLog returns the most recent sync log for a source.
func (s *SQLStore) GetLatestSyncLog(source string) (*pluginmodel.SyncLog, error) {
	log := &pluginmodel.SyncLog{}
	err := s.db().QueryRow(`
		SELECT id, source, sync_type, status, total_nodes, created_nodes, updated_nodes, deleted_nodes,
		       total_members, created_members, updated_members, deleted_members, skipped_users,
		       error_message, details, started_at, finished_at, triggered_by
		FROM org_directory_sync_logs WHERE source=$1
		ORDER BY started_at DESC LIMIT 1`, source).Scan(
		&log.ID, &log.Source, &log.SyncType, &log.Status,
		&log.TotalNodes, &log.CreatedNodes, &log.UpdatedNodes, &log.DeletedNodes,
		&log.TotalMembers, &log.CreatedMembers, &log.UpdatedMembers, &log.DeletedMembers,
		&log.SkippedUsers, &log.ErrorMessage, &log.Details,
		&log.StartedAt, &log.FinishedAt, &log.TriggeredBy,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return log, err
}

func scanSyncLogs(rows *sql.Rows) ([]*pluginmodel.SyncLog, error) {
	var logs []*pluginmodel.SyncLog
	for rows.Next() {
		log := &pluginmodel.SyncLog{}
		err := rows.Scan(
			&log.ID, &log.Source, &log.SyncType, &log.Status,
			&log.TotalNodes, &log.CreatedNodes, &log.UpdatedNodes, &log.DeletedNodes,
			&log.TotalMembers, &log.CreatedMembers, &log.UpdatedMembers, &log.DeletedMembers,
			&log.SkippedUsers, &log.ErrorMessage, &log.Details,
			&log.StartedAt, &log.FinishedAt, &log.TriggeredBy,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// LogAction records an audit log entry.
func (s *SQLStore) LogAction(log *pluginmodel.AuditLog) error {
	log.ID = mmmodel.NewId()
	log.CreateAt = time.Now().UnixMilli()
	_, err := s.master().Exec(`
		INSERT INTO org_directory_audit_log (id, actor_id, action, target_type, target_id, details, create_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		log.ID, log.ActorID, log.Action, log.TargetType, log.TargetID, log.Details, log.CreateAt,
	)
	return err
}

// GetAuditLogs returns paginated audit logs for a target.
func (s *SQLStore) GetAuditLogs(targetType, targetID string, page, perPage int) ([]*pluginmodel.AuditLog, error) {
	offset := page * perPage
	rows, err := s.db().Query(`
		SELECT id, actor_id, action, target_type, target_id, details, create_at
		FROM org_directory_audit_log
		WHERE target_type=$1 AND target_id=$2
		ORDER BY create_at DESC
		LIMIT $3 OFFSET $4`, targetType, targetID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*pluginmodel.AuditLog
	for rows.Next() {
		l := &pluginmodel.AuditLog{}
		err := rows.Scan(&l.ID, &l.ActorID, &l.Action, &l.TargetType, &l.TargetID, &l.Details, &l.CreateAt)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
