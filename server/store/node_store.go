package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	mmmodel "github.com/mattermost/mattermost/server/public/model"

	pluginmodel "github.com/your-org/mattermost-plugin-org-directory/server/model"
)

// CreateNode inserts a new organization node.
func (s *SQLStore) CreateNode(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error) {
	node.ID = mmmodel.NewId()
	node.CreateAt = time.Now().UnixMilli()
	node.UpdateAt = node.CreateAt
	node.PreSave()

	// Build path from parent
	if node.ParentID == "" {
		node.Path = "/" + node.ID
		node.Depth = 0
	} else {
		parent, err := s.GetNode(node.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent node not found: %w", err)
		}
		node.Path = parent.Path + "/" + node.ID
		node.Depth = parent.Depth + 1
	}

	_, err := s.master().Exec(`
		INSERT INTO org_directory_nodes
		(id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		 source, external_id, create_at, update_at, delete_at, creator_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		node.ID, node.Name, node.ParentID, node.Path, node.Depth, node.SortOrder,
		node.Description, node.Icon, node.Metadata,
		node.Source, node.ExternalID, node.CreateAt, node.UpdateAt, 0, node.CreatorID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}
	return node, nil
}

// GetNode fetches a node by its internal ID.
func (s *SQLStore) GetNode(id string) (*pluginmodel.OrgNode, error) {
	row := s.db().QueryRow(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE id = $1 AND delete_at = 0`, id)
	return scanNode(row)
}

// UpdateNode updates an existing node's mutable fields.
func (s *SQLStore) UpdateNode(node *pluginmodel.OrgNode) error {
	node.UpdateAt = time.Now().UnixMilli()
	_, err := s.master().Exec(`
		UPDATE org_directory_nodes
		SET name=$1, description=$2, icon=$3, metadata=$4, sort_order=$5, update_at=$6
		WHERE id=$7 AND delete_at=0`,
		node.Name, node.Description, node.Icon, node.Metadata, node.SortOrder, node.UpdateAt, node.ID,
	)
	return err
}

// DeleteNode soft-deletes a node by setting delete_at.
func (s *SQLStore) DeleteNode(id string, cascadeStrategy string) error {
	node, err := s.GetNode(id)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()

	tx, err := s.master().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	switch cascadeStrategy {
	case "move_to_parent":
		// Move direct children to the deleted node's parent
		_, err = tx.Exec(`
			UPDATE org_directory_nodes
			SET parent_id=$1, update_at=$2
			WHERE parent_id=$3 AND delete_at=0`,
			node.ParentID, now, id)
		if err != nil {
			return err
		}
		// Soft-delete the node itself
		_, err = tx.Exec(`UPDATE org_directory_nodes SET delete_at=$1, update_at=$1 WHERE id=$2`,
			now, id)
	default: // cascade_delete
		// Soft-delete all descendants using path prefix
		_, err = tx.Exec(`
			UPDATE org_directory_nodes
			SET delete_at=$1, update_at=$1
			WHERE (path LIKE $2 OR id=$3) AND delete_at=0`,
			now, node.Path+"/%", id)
		if err != nil {
			return err
		}
		// Soft-delete all member associations for the subtree
		_, err = tx.Exec(`
			UPDATE org_directory_members SET delete_at=$1, update_at=$1
			WHERE node_id IN (
				SELECT id FROM org_directory_nodes WHERE (path LIKE $2 OR id=$3) AND delete_at=0
			)`,
			now, node.Path+"/%", id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetChildNodes returns all direct children of a node.
func (s *SQLStore) GetChildNodes(parentID string) ([]*pluginmodel.OrgNode, error) {
	rows, err := s.db().Query(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE parent_id=$1 AND delete_at=0
		ORDER BY sort_order, name`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// GetSubTree returns all descendant nodes of a given node using materialized path.
func (s *SQLStore) GetSubTree(nodeID string) ([]*pluginmodel.OrgNode, error) {
	node, err := s.GetNode(nodeID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db().Query(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE path LIKE $1 AND delete_at=0
		ORDER BY path, sort_order`,
		node.Path+"/%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// MoveNode moves a node and all its descendants to a new parent (transactional).
func (s *SQLStore) MoveNode(nodeID, newParentID string) error {
	tx, err := s.master().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var oldPath string
	var oldDepth int
	err = tx.QueryRow(`SELECT path, depth FROM org_directory_nodes WHERE id=$1`, nodeID).
		Scan(&oldPath, &oldDepth)
	if err != nil {
		return err
	}

	var newParentPath string
	var newParentDepth int
	if newParentID == "" {
		newParentPath = ""
		newParentDepth = -1
	} else {
		err = tx.QueryRow(`SELECT path, depth FROM org_directory_nodes WHERE id=$1`, newParentID).
			Scan(&newParentPath, &newParentDepth)
		if err != nil {
			return err
		}
		// Prevent circular reference
		if strings.HasPrefix(newParentPath, oldPath+"/") || newParentPath == oldPath {
			return fmt.Errorf("cannot move node to its own descendant")
		}
	}

	newPath := newParentPath + "/" + nodeID
	newDepth := newParentDepth + 1
	depthDiff := newDepth - oldDepth
	now := time.Now().UnixMilli()

	// Update the node itself
	_, err = tx.Exec(`
		UPDATE org_directory_nodes
		SET parent_id=$1, path=$2, depth=$3, update_at=$4
		WHERE id=$5`,
		newParentID, newPath, newDepth, now, nodeID)
	if err != nil {
		return err
	}

	// Batch-update all descendant paths and depths
	oldPathLen := len(oldPath) + 1 // +1 for the position after the old path segment
	_, err = tx.Exec(`
		UPDATE org_directory_nodes
		SET path = $1 || SUBSTRING(path FROM $2),
		    depth = depth + $3,
		    update_at = $4
		WHERE path LIKE $5 AND delete_at=0`,
		newPath, oldPathLen, depthDiff, now, oldPath+"/%")
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetRootNodes returns all top-level nodes (nodes without a parent).
func (s *SQLStore) GetRootNodes() ([]*pluginmodel.OrgNode, error) {
	rows, err := s.db().Query(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE parent_id='' AND delete_at=0
		ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// SearchNodes searches nodes by name.
func (s *SQLStore) SearchNodes(query string) ([]*pluginmodel.OrgNode, error) {
	like := "%" + strings.ToLower(query) + "%"
	rows, err := s.db().Query(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE LOWER(name) LIKE $1 AND delete_at=0
		ORDER BY depth, sort_order, name
		LIMIT 50`, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// GetNodePath returns all nodes from the root to the given node (breadcrumb).
func (s *SQLStore) GetNodePath(nodeID string) ([]*pluginmodel.OrgNode, error) {
	node, err := s.GetNode(nodeID)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(strings.TrimPrefix(node.Path, "/"), "/")
	nodes := make([]*pluginmodel.OrgNode, 0, len(parts))
	for _, id := range parts {
		if id == "" {
			continue
		}
		n, err := s.GetNode(id)
		if err != nil {
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// ReorderNodes updates sort_order for a list of sibling nodes.
func (s *SQLStore) ReorderNodes(parentID string, nodeIDs []string) error {
	now := time.Now().UnixMilli()
	for i, id := range nodeIDs {
		_, err := s.master().Exec(`
			UPDATE org_directory_nodes SET sort_order=$1, update_at=$2
			WHERE id=$3 AND parent_id=$4`,
			i, now, id, parentID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetNodeMemberCount returns the number of members in a node (optionally recursive).
func (s *SQLStore) GetNodeMemberCount(nodeID string, recursive bool) (int64, error) {
	var count int64
	var err error
	if recursive {
		node, e := s.GetNode(nodeID)
		if e != nil {
			return 0, e
		}
		err = s.db().QueryRow(`
			SELECT COUNT(DISTINCT m.user_id)
			FROM org_directory_members m
			JOIN org_directory_nodes n ON m.node_id = n.id
			WHERE (n.path LIKE $1 OR n.id=$2)
			  AND m.delete_at=0 AND n.delete_at=0`,
			node.Path+"/%", nodeID).Scan(&count)
	} else {
		err = s.db().QueryRow(`
			SELECT COUNT(*) FROM org_directory_members
			WHERE node_id=$1 AND delete_at=0`, nodeID).Scan(&count)
	}
	return count, err
}

// GetNodeByExternalID finds a node by its external system ID.
func (s *SQLStore) GetNodeByExternalID(source, externalID string) (*pluginmodel.OrgNode, error) {
	row := s.db().QueryRow(`
		SELECT id, name, parent_id, path, depth, sort_order, description, icon, metadata,
		       source, external_id, create_at, update_at, delete_at, creator_id
		FROM org_directory_nodes
		WHERE source=$1 AND external_id=$2 AND delete_at=0`, source, externalID)
	return scanNode(row)
}

// UpsertNodeByExternalID creates or updates a node identified by source+external_id.
func (s *SQLStore) UpsertNodeByExternalID(node *pluginmodel.OrgNode) (*pluginmodel.OrgNode, error) {
	existing, err := s.GetNodeByExternalID(node.Source, node.ExternalID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if existing != nil {
		existing.Name = node.Name
		existing.ParentID = node.ParentID
		existing.Description = node.Description
		existing.Icon = node.Icon
		existing.Metadata = node.Metadata
		existing.SortOrder = node.SortOrder
		if err := s.UpdateNode(existing); err != nil {
			return nil, err
		}
		// If parent changed, move the node (updates path/depth)
		if existing.ParentID != node.ParentID {
			if err := s.MoveNode(existing.ID, node.ParentID); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}
	return s.CreateNode(node)
}

// SoftDeleteNodesBySource soft-deletes nodes from a source that are not in the exclude list.
func (s *SQLStore) SoftDeleteNodesBySource(source string, excludeExternalIDs []string) (int, error) {
	if len(excludeExternalIDs) == 0 {
		res, err := s.master().Exec(`
			UPDATE org_directory_nodes SET delete_at=$1, update_at=$1
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
		UPDATE org_directory_nodes SET delete_at=$1, update_at=$1
		WHERE source=$2 AND delete_at=0 AND external_id NOT IN (%s)`,
		strings.Join(placeholders, ","))
	res, err := s.master().Exec(query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// SoftDeleteNodes soft-deletes a list of nodes by ID.
func (s *SQLStore) SoftDeleteNodes(nodeIDs []string, now int64) error {
	if len(nodeIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(nodeIDs))
	args := make([]interface{}, len(nodeIDs)+1)
	args[0] = now
	for i, id := range nodeIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}
	query := fmt.Sprintf(`
		UPDATE org_directory_nodes SET delete_at=$1, update_at=$1
		WHERE id IN (%s)`, strings.Join(placeholders, ","))
	_, err := s.master().Exec(query, args...)
	return err
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanNode(row rowScanner) (*pluginmodel.OrgNode, error) {
	n := &pluginmodel.OrgNode{}
	err := row.Scan(
		&n.ID, &n.Name, &n.ParentID, &n.Path, &n.Depth, &n.SortOrder,
		&n.Description, &n.Icon, &n.Metadata,
		&n.Source, &n.ExternalID,
		&n.CreateAt, &n.UpdateAt, &n.DeleteAt, &n.CreatorID,
	)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

func scanNodes(rows *sql.Rows) ([]*pluginmodel.OrgNode, error) {
	var nodes []*pluginmodel.OrgNode
	for rows.Next() {
		n := &pluginmodel.OrgNode{}
		err := rows.Scan(
			&n.ID, &n.Name, &n.ParentID, &n.Path, &n.Depth, &n.SortOrder,
			&n.Description, &n.Icon, &n.Metadata,
			&n.Source, &n.ExternalID,
			&n.CreateAt, &n.UpdateAt, &n.DeleteAt, &n.CreatorID,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}
