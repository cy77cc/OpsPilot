package v1

import "time"

type CI struct {
	ID           uint       `json:"id"`
	CIUID        string     `json:"ci_uid"`
	CIType       string     `json:"ci_type"`
	Name         string     `json:"name"`
	Source       string     `json:"source"`
	ExternalID   string     `json:"external_id"`
	ProjectID    uint       `json:"project_id"`
	TeamID       uint       `json:"team_id"`
	Owner        string     `json:"owner"`
	Status       string     `json:"status"`
	TagsJSON     string     `json:"tags_json"`
	AttrsJSON    string     `json:"attrs_json"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type CIRelation struct {
	ID           uint      `json:"id"`
	FromCIID     uint      `json:"from_ci_id"`
	ToCIID       uint      `json:"to_ci_id"`
	RelationType string    `json:"relation_type"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateCIReq struct {
	CIType     string `json:"ci_type" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Source     string `json:"source"`
	ExternalID string `json:"external_id"`
	ProjectID  uint   `json:"project_id"`
	TeamID     uint   `json:"team_id"`
	Owner      string `json:"owner"`
	Status     string `json:"status"`
	TagsJSON   string `json:"tags_json"`
	AttrsJSON  string `json:"attrs_json"`
}

type UpdateCIReq struct {
	Name      *string `json:"name"`
	Owner     *string `json:"owner"`
	Status    *string `json:"status"`
	TagsJSON  *string `json:"tags_json"`
	AttrsJSON *string `json:"attrs_json"`
}

type CreateRelationReq struct {
	FromCIID     uint   `json:"from_ci_id" binding:"required"`
	ToCIID       uint   `json:"to_ci_id" binding:"required"`
	RelationType string `json:"relation_type" binding:"required"`
}

type TriggerSyncReq struct {
	Source string `json:"source"`
}

type SyncJob struct {
	ID           string    `json:"id"`
	Source       string    `json:"source"`
	Status       string    `json:"status"`
	SummaryJSON  string    `json:"summary_json"`
	ErrorMessage string    `json:"error_message"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	OperatorID   uint      `json:"operator_id"`
	CreatedAt    time.Time `json:"created_at"`
}
