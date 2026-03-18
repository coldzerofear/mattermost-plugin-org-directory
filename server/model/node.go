package model

// OrgNode represents a node in the organization tree.
type OrgNode struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	ParentID    string `json:"parent_id" db:"parent_id"`
	Path        string `json:"path" db:"path"`
	Depth       int    `json:"depth" db:"depth"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
	Description string `json:"description" db:"description"`
	Icon        string `json:"icon" db:"icon"`
	Metadata    string `json:"metadata" db:"metadata"`
	Source      string `json:"source" db:"source"`           // local, hr_system, ldap, etc.
	ExternalID  string `json:"external_id" db:"external_id"` // ID in external system
	CreateAt    int64  `json:"create_at" db:"create_at"`
	UpdateAt    int64  `json:"update_at" db:"update_at"`
	DeleteAt    int64  `json:"delete_at" db:"delete_at"`
	CreatorID   string `json:"creator_id" db:"creator_id"`

	// Fields populated at query time, not stored in DB
	Children    []*OrgNode `json:"children,omitempty" db:"-"`
	MemberCount int64      `json:"member_count,omitempty" db:"-"`
	HasChildren bool       `json:"has_children" db:"-"`
}

// OrgTreeNode is used for frontend tree rendering.
type OrgTreeNode struct {
	*OrgNode
	Children    []*OrgTreeNode      `json:"children"`
	Members     []*OrgMemberWithUser `json:"members,omitempty"`
	MemberCount int64               `json:"member_count"`
}

// IsValid validates the OrgNode fields.
func (n *OrgNode) IsValid() bool {
	return n.Name != "" && len(n.Name) <= 256
}

// PreSave prepares the node before saving (sets default values).
func (n *OrgNode) PreSave() {
	if n.Metadata == "" {
		n.Metadata = "{}"
	}
	if n.Source == "" {
		n.Source = "local"
	}
}
