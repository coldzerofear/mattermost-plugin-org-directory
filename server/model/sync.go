package model

// SyncLog records the result of a synchronization task.
type SyncLog struct {
	ID             string `json:"id" db:"id"`
	Source         string `json:"source" db:"source"`
	SyncType       string `json:"sync_type" db:"sync_type"`       // full / incremental
	Status         string `json:"status" db:"status"`             // running / success / partial_success / failed
	TotalNodes     int    `json:"total_nodes" db:"total_nodes"`
	CreatedNodes   int    `json:"created_nodes" db:"created_nodes"`
	UpdatedNodes   int    `json:"updated_nodes" db:"updated_nodes"`
	DeletedNodes   int    `json:"deleted_nodes" db:"deleted_nodes"`
	TotalMembers   int    `json:"total_members" db:"total_members"`
	CreatedMembers int    `json:"created_members" db:"created_members"`
	UpdatedMembers int    `json:"updated_members" db:"updated_members"`
	DeletedMembers int    `json:"deleted_members" db:"deleted_members"`
	SkippedUsers   int    `json:"skipped_users" db:"skipped_users"`
	ErrorMessage   string `json:"error_message" db:"error_message"`
	Details        string `json:"details" db:"details"` // JSON
	StartedAt      int64  `json:"started_at" db:"started_at"`
	FinishedAt     int64  `json:"finished_at" db:"finished_at"`
	TriggeredBy    string `json:"triggered_by" db:"triggered_by"`
}

// SyncNodePayload is a single node payload from an external sync request.
type SyncNodePayload struct {
	ExternalID       string `json:"external_id"`
	Name             string `json:"name"`
	ParentExternalID string `json:"parent_external_id"`
	SortOrder        int    `json:"sort_order"`
	Description      string `json:"description"`
	Icon             string `json:"icon"`
	Metadata         string `json:"metadata"`
	// Action controls how this payload is processed.
	// "" or "upsert" — create or update (default).
	// "delete"        — delete the node identified by (source, external_id).
	Action string `json:"action"`
}

// SyncMemberPayload is a single member payload from an external sync request.
type SyncMemberPayload struct {
	ExternalID       string `json:"external_id"`
	NodeExternalID   string `json:"node_external_id"`
	ExternalUserID   string `json:"external_user_id"`
	ExternalUsername string `json:"external_username"`
	ExternalEmail    string `json:"external_email"`
	Role             string `json:"role"`
	Position         string `json:"position"`
	SortOrder        int    `json:"sort_order"`
	// Action controls how this payload is processed.
	// "" or "upsert" — create or update (default).
	// "delete"        — remove the member identified by (node_external_id + external_user_id).
	Action string `json:"action"`
}

// SyncRequest is the request body for the sync API.
type SyncRequest struct {
	Source   string              `json:"source"`
	SyncType string              `json:"sync_type"` // full / incremental
	Nodes    []SyncNodePayload   `json:"nodes"`
	Members  []SyncMemberPayload `json:"members"`
}

// SyncResponse is the response body for the sync API.
type SyncResponse struct {
	SyncLogID      string        `json:"sync_log_id"`
	Status         string        `json:"status"`
	TotalNodes     int           `json:"total_nodes"`
	CreatedNodes   int           `json:"created_nodes"`
	UpdatedNodes   int           `json:"updated_nodes"`
	DeletedNodes   int           `json:"deleted_nodes"`
	TotalMembers   int           `json:"total_members"`
	CreatedMembers int           `json:"created_members"`
	UpdatedMembers int           `json:"updated_members"`
	DeletedMembers int           `json:"deleted_members"`
	SkippedUsers   int           `json:"skipped_users"`
	Errors         []string      `json:"errors,omitempty"`
	SkippedDetails []SkippedUser `json:"skipped_details,omitempty"`
}

// SkippedUser records info about a user that could not be matched during sync.
type SkippedUser struct {
	ExternalUserID   string `json:"external_user_id"`
	ExternalUsername string `json:"external_username"`
	ExternalEmail    string `json:"external_email"`
	Reason           string `json:"reason"`
}
