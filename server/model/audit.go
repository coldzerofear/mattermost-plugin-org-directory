package model

// AuditLog records administrative operations on the org directory.
type AuditLog struct {
	ID         string `json:"id" db:"id"`
	ActorID    string `json:"actor_id" db:"actor_id"`
	Action     string `json:"action" db:"action"`           // create_node, move_node, add_member, ...
	TargetType string `json:"target_type" db:"target_type"` // node / member
	TargetID   string `json:"target_id" db:"target_id"`
	Details    string `json:"details" db:"details"` // JSON
	CreateAt   int64  `json:"create_at" db:"create_at"`
}
