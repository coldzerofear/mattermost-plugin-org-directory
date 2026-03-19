package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	mmmodel "github.com/mattermost/mattermost/server/public/model"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// AddMember adds a user to an organization node.
// If a soft-deleted record exists for the same (node_id, user_id), it is restored
// instead of inserting a new row (which would violate the UNIQUE constraint).
func (s *SQLStore) AddMember(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	now := time.Now().UnixMilli()
	member.PreSave()

	// Try to restore a soft-deleted row first.
	var existingID string
	err := s.db().QueryRow(`
		SELECT id FROM org_directory_members
		WHERE node_id=$1 AND user_id=$2 AND delete_at!=0`,
		member.NodeID, member.UserID).Scan(&existingID)
	if err == nil {
		// Soft-deleted row found — restore it with updated fields.
		_, err = s.master().Exec(`
			UPDATE org_directory_members
			SET role=$1, position=$2, sort_order=$3, source=$4, external_id=$5,
			    update_at=$6, delete_at=0
			WHERE id=$7`,
			member.Role, member.Position, member.SortOrder, member.Source, member.ExternalID,
			now, existingID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to restore member: %w", err)
		}
		member.ID = existingID
		member.UpdateAt = now
		member.CreateAt = now
		return member, nil
	}

	// No soft-deleted row — insert new.
	member.ID = mmmodel.NewId()
	member.CreateAt = now
	member.UpdateAt = now

	_, err = s.master().Exec(`
		INSERT INTO org_directory_members
		(id, node_id, user_id, role, position, sort_order, source, external_id, create_at, update_at, delete_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		member.ID, member.NodeID, member.UserID, member.Role, member.Position,
		member.SortOrder, member.Source, member.ExternalID,
		member.CreateAt, member.UpdateAt, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}
	return member, nil
}

// RemoveMember soft-deletes a member association.
func (s *SQLStore) RemoveMember(nodeID, userID string) error {
	now := time.Now().UnixMilli()
	_, err := s.master().Exec(`
		UPDATE org_directory_members SET delete_at=$1, update_at=$1
		WHERE node_id=$2 AND user_id=$3 AND delete_at=0`,
		now, nodeID, userID)
	return err
}

// GetMembers returns paginated members of a node with user details.
func (s *SQLStore) GetMembers(nodeID string, page, perPage int) ([]*pluginmodel.OrgMemberWithUser, error) {
	offset := page * perPage
	rows, err := s.db().Query(`
		SELECT m.id, m.node_id, m.user_id, m.role, m.position, m.sort_order,
		       m.source, m.external_id, m.create_at, m.update_at, m.delete_at,
		       COALESCE(u.username,''), COALESCE(u.firstname,''), COALESCE(u.lastname,''),
		       COALESCE(u.nickname,''), COALESCE(u.email,''), COALESCE(u.position,'')
		FROM org_directory_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.node_id=$1 AND m.delete_at=0 AND u.deleteat=0
		ORDER BY m.sort_order, u.username
		LIMIT $2 OFFSET $3`,
		nodeID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMembersWithUser(rows)
}

// GetMembersForNodes returns paginated members for a set of nodes.
func (s *SQLStore) GetMembersForNodes(nodeIDs []string, page, perPage int) ([]*pluginmodel.OrgMemberWithUser, error) {
	if len(nodeIDs) == 0 {
		return []*pluginmodel.OrgMemberWithUser{}, nil
	}

	sortedNodeIDs := append([]string(nil), nodeIDs...)
	sort.Strings(sortedNodeIDs)

	placeholders := make([]string, len(sortedNodeIDs))
	args := make([]interface{}, 0, len(sortedNodeIDs)+2)
	for i, nodeID := range sortedNodeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args = append(args, nodeID)
	}

	offset := page * perPage
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	args = append(args, perPage, offset)

	query := fmt.Sprintf(`
		SELECT m.id, m.node_id, m.user_id, m.role, m.position, m.sort_order,
		       m.source, m.external_id, m.create_at, m.update_at, m.delete_at,
		       COALESCE(u.username,''), COALESCE(u.firstname,''), COALESCE(u.lastname,''),
		       COALESCE(u.nickname,''), COALESCE(u.email,''), COALESCE(u.position,'')
		FROM org_directory_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.node_id IN (%s) AND m.delete_at=0 AND u.deleteat=0
		ORDER BY m.node_id, m.sort_order, u.username
		LIMIT %s OFFSET %s`, strings.Join(placeholders, ","), limitPlaceholder, offsetPlaceholder)

	rows, err := s.db().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMembersWithUser(rows)
}

// GetAllMembersByNodeID returns all (non-deleted) members of a node.
func (s *SQLStore) GetAllMembersByNodeID(nodeID string) ([]*pluginmodel.OrgMember, error) {
	rows, err := s.db().Query(`
		SELECT id, node_id, user_id, role, position, sort_order,
		       source, external_id, create_at, update_at, delete_at
		FROM org_directory_members
		WHERE node_id=$1 AND delete_at=0`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMembers(rows)
}

// GetUserNodes returns all organization nodes a user belongs to.
func (s *SQLStore) GetUserNodes(userID string) ([]*pluginmodel.OrgNode, error) {
	rows, err := s.db().Query(`
		SELECT n.id, n.name, n.parent_id, n.path, n.depth, n.sort_order,
		       n.description, n.icon, n.metadata, n.source, n.external_id,
		       n.create_at, n.update_at, n.delete_at, n.creator_id
		FROM org_directory_nodes n
		JOIN org_directory_members m ON n.id = m.node_id
		WHERE m.user_id=$1 AND m.delete_at=0 AND n.delete_at=0
		ORDER BY n.depth, n.sort_order`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// UpdateMemberRole updates the role of a member in a node.
func (s *SQLStore) UpdateMemberRole(nodeID, userID, role string) error {
	_, err := s.master().Exec(`
		UPDATE org_directory_members SET role=$1, update_at=$2
		WHERE node_id=$3 AND user_id=$4 AND delete_at=0`,
		role, time.Now().UnixMilli(), nodeID, userID)
	return err
}

// UpdateMemberPosition updates the position of a member in a node.
func (s *SQLStore) UpdateMemberPosition(nodeID, userID, position string) error {
	_, err := s.master().Exec(`
		UPDATE org_directory_members SET position=$1, update_at=$2
		WHERE node_id=$3 AND user_id=$4 AND delete_at=0`,
		position, time.Now().UnixMilli(), nodeID, userID)
	return err
}

// SearchMembers searches for users across all org nodes.
func (s *SQLStore) SearchMembers(query string, page, perPage int) ([]*pluginmodel.SearchResult, error) {
	like := "%" + strings.ToLower(query) + "%"
	offset := page * perPage
	rows, err := s.db().Query(`
		SELECT DISTINCT ON (u.id)
		       m.id, m.node_id, m.user_id, m.role, m.position, m.sort_order,
		       m.source, m.external_id, m.create_at, m.update_at, m.delete_at,
		       COALESCE(u.username,''), COALESCE(u.firstname,''), COALESCE(u.lastname,''),
		       COALESCE(u.nickname,''), COALESCE(u.email,''), COALESCE(u.position,''),
		       n.name AS node_name, n.path AS node_path
		FROM users u
		JOIN org_directory_members m ON u.id = m.user_id
		JOIN org_directory_nodes n ON m.node_id = n.id
		WHERE m.delete_at=0 AND n.delete_at=0 AND u.deleteat=0
		  AND (LOWER(COALESCE(u.username,'')) LIKE $1
		    OR LOWER(COALESCE(u.firstname,'')) LIKE $1
		    OR LOWER(COALESCE(u.lastname,'')) LIKE $1
		    OR LOWER(COALESCE(u.nickname,'')) LIKE $1
		    OR LOWER(COALESCE(u.email,'')) LIKE $1
		    OR LOWER(n.name) LIKE $1)
		ORDER BY u.id, COALESCE(u.username,'')
		LIMIT $2 OFFSET $3`,
		like, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*pluginmodel.SearchResult
	for rows.Next() {
		m := &pluginmodel.OrgMember{}
		wu := &pluginmodel.OrgMemberWithUser{OrgMember: m}
		var nodeName, nodePath string
		err := rows.Scan(
			&m.ID, &m.NodeID, &m.UserID, &m.Role, &m.Position, &m.SortOrder,
			&m.Source, &m.ExternalID, &m.CreateAt, &m.UpdateAt, &m.DeleteAt,
			&wu.Username, &wu.FirstName, &wu.LastName, &wu.Nickname, &wu.Email, &wu.MmPosition,
			&nodeName, &nodePath,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, &pluginmodel.SearchResult{
			User:     wu,
			NodeName: nodeName,
			NodePath: nodePath,
		})
	}
	return results, rows.Err()
}

// IsMember checks if a user is a member of a node.
func (s *SQLStore) IsMember(nodeID, userID string) (bool, error) {
	var count int
	err := s.db().QueryRow(`
		SELECT COUNT(*) FROM org_directory_members
		WHERE node_id=$1 AND user_id=$2 AND delete_at=0`,
		nodeID, userID).Scan(&count)
	return count > 0, err
}

// GetMemberRole returns the membership record for a user in a node.
func (s *SQLStore) GetMemberRole(nodeID, userID string) (*pluginmodel.OrgMember, error) {
	m := &pluginmodel.OrgMember{}
	err := s.db().QueryRow(`
		SELECT id, node_id, user_id, role, position, sort_order,
		       source, external_id, create_at, update_at, delete_at
		FROM org_directory_members
		WHERE node_id=$1 AND user_id=$2 AND delete_at=0`,
		nodeID, userID).Scan(
		&m.ID, &m.NodeID, &m.UserID, &m.Role, &m.Position, &m.SortOrder,
		&m.Source, &m.ExternalID, &m.CreateAt, &m.UpdateAt, &m.DeleteAt,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return m, err
}

// ReorderMembers updates sort_order for a list of members in a node.
func (s *SQLStore) ReorderMembers(nodeID string, userIDs []string) error {
	now := time.Now().UnixMilli()
	for i, uid := range userIDs {
		_, err := s.master().Exec(`
			UPDATE org_directory_members SET sort_order=$1, update_at=$2
			WHERE node_id=$3 AND user_id=$4`,
			i, now, nodeID, uid)
		if err != nil {
			return err
		}
	}
	return nil
}

// SoftDeleteMembersByNodeID soft-deletes all member associations for a node.
func (s *SQLStore) SoftDeleteMembersByNodeID(nodeID string, now int64) (int, error) {
	res, err := s.master().Exec(`
		UPDATE org_directory_members SET delete_at=$1, update_at=$1
		WHERE node_id=$2 AND delete_at=0`, now, nodeID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// UpsertMemberByExternalID creates or updates a member identified by external_id.
func (s *SQLStore) UpsertMemberByExternalID(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	var existing pluginmodel.OrgMember
	err := s.db().QueryRow(`
		SELECT id, node_id, user_id, role, position, sort_order,
		       source, external_id, create_at, update_at, delete_at
		FROM org_directory_members
		WHERE source=$1 AND external_id=$2 AND delete_at=0`,
		member.Source, member.ExternalID).Scan(
		&existing.ID, &existing.NodeID, &existing.UserID, &existing.Role,
		&existing.Position, &existing.SortOrder, &existing.Source, &existing.ExternalID,
		&existing.CreateAt, &existing.UpdateAt, &existing.DeleteAt,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == nil {
		// Update
		now := time.Now().UnixMilli()
		_, err = s.master().Exec(`
			UPDATE org_directory_members SET role=$1, position=$2, sort_order=$3, update_at=$4
			WHERE id=$5`,
			member.Role, member.Position, member.SortOrder, now, existing.ID)
		if err != nil {
			return nil, err
		}
		existing.Role = member.Role
		existing.Position = member.Position
		return &existing, nil
	}
	return s.AddMember(member)
}

// UpsertMemberByNodeAndUser upserts by (node_id, user_id).
func (s *SQLStore) UpsertMemberByNodeAndUser(member *pluginmodel.OrgMember) (*pluginmodel.OrgMember, error) {
	exists, err := s.IsMember(member.NodeID, member.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return member, nil
	}
	return s.AddMember(member)
}

// SoftDeleteMembersBySource soft-deletes members from a source not in the exclude list.
func (s *SQLStore) SoftDeleteMembersBySource(source string, excludeExternalIDs []string) (int, error) {
	if len(excludeExternalIDs) == 0 {
		res, err := s.master().Exec(`
			UPDATE org_directory_members SET delete_at=$1, update_at=$1
			WHERE source=$2 AND delete_at=0`, time.Now().UnixMilli(), source)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}

	placeholders := make([]string, len(excludeExternalIDs))
	args := make([]interface{}, len(excludeExternalIDs)+2)
	args[0] = time.Now().UnixMilli()
	args[1] = source
	for i, id := range excludeExternalIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}
	query := fmt.Sprintf(`
		UPDATE org_directory_members SET delete_at=$1, update_at=$1
		WHERE source=$2 AND delete_at=0 AND external_id NOT IN (%s)`,
		strings.Join(placeholders, ","))
	res, err := s.master().Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// --- helpers ---

func scanMembersWithUser(rows *sql.Rows) ([]*pluginmodel.OrgMemberWithUser, error) {
	var members []*pluginmodel.OrgMemberWithUser
	for rows.Next() {
		m := &pluginmodel.OrgMember{}
		wu := &pluginmodel.OrgMemberWithUser{OrgMember: m}
		err := rows.Scan(
			&m.ID, &m.NodeID, &m.UserID, &m.Role, &m.Position, &m.SortOrder,
			&m.Source, &m.ExternalID, &m.CreateAt, &m.UpdateAt, &m.DeleteAt,
			&wu.Username, &wu.FirstName, &wu.LastName, &wu.Nickname, &wu.Email, &wu.MmPosition,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, wu)
	}
	return members, rows.Err()
}

func scanMembers(rows *sql.Rows) ([]*pluginmodel.OrgMember, error) {
	var members []*pluginmodel.OrgMember
	for rows.Next() {
		m := &pluginmodel.OrgMember{}
		err := rows.Scan(
			&m.ID, &m.NodeID, &m.UserID, &m.Role, &m.Position, &m.SortOrder,
			&m.Source, &m.ExternalID, &m.CreateAt, &m.UpdateAt, &m.DeleteAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
