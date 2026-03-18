package model

// UserMapping stores the relationship between an external user ID and a Mattermost user ID.
type UserMapping struct {
	ID               string `json:"id" db:"id"`
	Source           string `json:"source" db:"source"`
	ExternalUserID   string `json:"external_user_id" db:"external_user_id"`
	MmUserID         string `json:"mm_user_id" db:"mm_user_id"`
	ExternalUsername string `json:"external_username" db:"external_username"`
	ExternalEmail    string `json:"external_email" db:"external_email"`
	CreateAt         int64  `json:"create_at" db:"create_at"`
	UpdateAt         int64  `json:"update_at" db:"update_at"`
	DeleteAt         int64  `json:"delete_at" db:"delete_at"`
}
