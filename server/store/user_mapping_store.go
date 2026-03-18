package store

import (
	"database/sql"
	"time"

	mmmodel "github.com/mattermost/mattermost/server/public/model"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// UpsertUserMapping creates or updates a user mapping record.
func (s *SQLStore) UpsertUserMapping(mapping *pluginmodel.UserMapping) (*pluginmodel.UserMapping, error) {
	existing, err := s.GetUserMappingByExternalID(mapping.Source, mapping.ExternalUserID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	now := time.Now().UnixMilli()
	if existing != nil {
		_, err = s.master().Exec(`
			UPDATE org_directory_user_mappings
			SET mm_user_id=$1, external_username=$2, external_email=$3, update_at=$4
			WHERE id=$5`,
			mapping.MmUserID, mapping.ExternalUsername, mapping.ExternalEmail, now, existing.ID)
		if err != nil {
			return nil, err
		}
		existing.MmUserID = mapping.MmUserID
		return existing, nil
	}

	mapping.ID = mmmodel.NewId()
	mapping.CreateAt = now
	mapping.UpdateAt = now
	_, err = s.master().Exec(`
		INSERT INTO org_directory_user_mappings
		(id, source, external_user_id, mm_user_id, external_username, external_email, create_at, update_at, delete_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		mapping.ID, mapping.Source, mapping.ExternalUserID, mapping.MmUserID,
		mapping.ExternalUsername, mapping.ExternalEmail, mapping.CreateAt, mapping.UpdateAt, 0,
	)
	return mapping, err
}

// GetUserMappingByExternalID retrieves a mapping by source and external user ID.
func (s *SQLStore) GetUserMappingByExternalID(source, externalUserID string) (*pluginmodel.UserMapping, error) {
	m := &pluginmodel.UserMapping{}
	err := s.db().QueryRow(`
		SELECT id, source, external_user_id, mm_user_id, external_username, external_email,
		       create_at, update_at, delete_at
		FROM org_directory_user_mappings
		WHERE source=$1 AND external_user_id=$2 AND delete_at=0`,
		source, externalUserID).Scan(
		&m.ID, &m.Source, &m.ExternalUserID, &m.MmUserID,
		&m.ExternalUsername, &m.ExternalEmail,
		&m.CreateAt, &m.UpdateAt, &m.DeleteAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return m, err
}

// GetUserMappingsBySource returns all mappings for a given source, paginated.
func (s *SQLStore) GetUserMappingsBySource(source string, page, perPage int) ([]*pluginmodel.UserMapping, error) {
	offset := page * perPage
	rows, err := s.db().Query(`
		SELECT id, source, external_user_id, mm_user_id, external_username, external_email,
		       create_at, update_at, delete_at
		FROM org_directory_user_mappings
		WHERE source=$1 AND delete_at=0
		ORDER BY create_at DESC
		LIMIT $2 OFFSET $3`, source, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserMappings(rows)
}

// GetUserMappingsByMmUserID returns all mappings for a given Mattermost user ID.
func (s *SQLStore) GetUserMappingsByMmUserID(mmUserID string) ([]*pluginmodel.UserMapping, error) {
	rows, err := s.db().Query(`
		SELECT id, source, external_user_id, mm_user_id, external_username, external_email,
		       create_at, update_at, delete_at
		FROM org_directory_user_mappings
		WHERE mm_user_id=$1 AND delete_at=0`, mmUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserMappings(rows)
}

// DeleteUserMapping soft-deletes a mapping.
func (s *SQLStore) DeleteUserMapping(source, externalUserID string) error {
	now := time.Now().UnixMilli()
	_, err := s.master().Exec(`
		UPDATE org_directory_user_mappings SET delete_at=$1, update_at=$1
		WHERE source=$2 AND external_user_id=$3 AND delete_at=0`,
		now, source, externalUserID)
	return err
}

// DeleteUserMappingsBySource soft-deletes all mappings for a source.
func (s *SQLStore) DeleteUserMappingsBySource(source string) error {
	now := time.Now().UnixMilli()
	_, err := s.master().Exec(`
		UPDATE org_directory_user_mappings SET delete_at=$1, update_at=$1
		WHERE source=$2 AND delete_at=0`, now, source)
	return err
}

func scanUserMappings(rows *sql.Rows) ([]*pluginmodel.UserMapping, error) {
	var mappings []*pluginmodel.UserMapping
	for rows.Next() {
		m := &pluginmodel.UserMapping{}
		err := rows.Scan(
			&m.ID, &m.Source, &m.ExternalUserID, &m.MmUserID,
			&m.ExternalUsername, &m.ExternalEmail,
			&m.CreateAt, &m.UpdateAt, &m.DeleteAt,
		)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, m)
	}
	return mappings, rows.Err()
}
