package model

// OrgMember represents the relationship between a user and an organization node.
type OrgMember struct {
	ID         string `json:"id" db:"id"`
	NodeID     string `json:"node_id" db:"node_id"`
	UserID     string `json:"user_id" db:"user_id"`
	Role       string `json:"role" db:"role"`         // member, admin, manager
	Position   string `json:"position" db:"position"` // job title within the org
	SortOrder  int    `json:"sort_order" db:"sort_order"`
	Source     string `json:"source" db:"source"`           // data origin
	ExternalID string `json:"external_id" db:"external_id"` // ID in external system
	CreateAt   int64  `json:"create_at" db:"create_at"`
	UpdateAt   int64  `json:"update_at" db:"update_at"`
	DeleteAt   int64  `json:"delete_at" db:"delete_at"`
}

// OrgMemberWithUser combines member data with Mattermost user info.
type OrgMemberWithUser struct {
	*OrgMember
	Username         string `json:"username"`
	FirstName        string `json:"first_name"`
	LastName         string `json:"last_name"`
	Nickname         string `json:"nickname"`
	Email            string `json:"email"`
	MmPosition       string `json:"mm_position"`        // Mattermost native position field
	Status           string `json:"status"`             // online status
	LastPictureUpdate int64 `json:"last_picture_update"` // user avatar update time
}

// SearchResult represents a user search result with node context.
type SearchResult struct {
	User     *OrgMemberWithUser `json:"user"`
	NodeName string             `json:"node_name"`
	NodePath string             `json:"node_path"`
}

// PreSave prepares the member before saving.
func (m *OrgMember) PreSave() {
	if m.Role == "" {
		m.Role = "member"
	}
	if m.Source == "" {
		m.Source = "local"
	}
}
