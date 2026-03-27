package v1

import "time"

type UpsertServiceCIConfigReq struct {
	RepoURL        string   `json:"repo_url" binding:"required"`
	Branch         string   `json:"branch"`
	BuildSteps     []string `json:"build_steps"`
	ArtifactTarget string   `json:"artifact_target" binding:"required"`
	TriggerMode    string   `json:"trigger_mode" binding:"required"` // manual|source-event|both
}

type ServiceCIConfigResp struct {
	ID             uint      `json:"id"`
	ServiceID      uint      `json:"service_id"`
	RepoURL        string    `json:"repo_url"`
	Branch         string    `json:"branch"`
	BuildSteps     []string  `json:"build_steps"`
	ArtifactTarget string    `json:"artifact_target"`
	TriggerMode    string    `json:"trigger_mode"`
	Status         string    `json:"status"`
	UpdatedBy      uint      `json:"updated_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TriggerCIRunReq struct {
	TriggerType string `json:"trigger_type" binding:"required"` // manual|source-event
	Reason      string `json:"reason"`
}

type CIRunResp struct {
	ID          uint      `json:"id"`
	ServiceID   uint      `json:"service_id"`
	CIConfigID  uint      `json:"ci_config_id"`
	TriggerType string    `json:"trigger_type"`
	Status      string    `json:"status"`
	Reason      string    `json:"reason"`
	TriggeredBy uint      `json:"triggered_by"`
	TriggeredAt time.Time `json:"triggered_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type UpsertDeploymentCDConfigReq struct {
	Env              string         `json:"env" binding:"required"`
	RuntimeType      string         `json:"runtime_type"`
	Strategy         string         `json:"strategy" binding:"required"` // rolling|blue-green|canary
	StrategyConfig   map[string]any `json:"strategy_config"`
	ApprovalRequired bool           `json:"approval_required"`
}

type DeploymentCDConfigResp struct {
	ID               uint           `json:"id"`
	DeploymentID     uint           `json:"deployment_id"`
	Env              string         `json:"env"`
	RuntimeType      string         `json:"runtime_type"`
	Strategy         string         `json:"strategy"`
	StrategyConfig   map[string]any `json:"strategy_config"`
	ApprovalRequired bool           `json:"approval_required"`
	UpdatedBy        uint           `json:"updated_by"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type TriggerReleaseReq struct {
	ServiceID     uint   `json:"service_id" binding:"required"`
	DeploymentID  uint   `json:"deployment_id"`
	Env           string `json:"env" binding:"required"`
	RuntimeType   string `json:"runtime_type"`
	Version       string `json:"version" binding:"required"`
	CIRunID       uint   `json:"ci_run_id,omitempty"`
	TriggerSource string `json:"trigger_source,omitempty"` // manual|ci, defaults to ci for cicd endpoint
}

type ReleaseDecisionReq struct {
	Comment string `json:"comment"`
}

type RollbackReleaseReq struct {
	TargetVersion string `json:"target_version" binding:"required"`
	Comment       string `json:"comment"`
}

type ReleaseResp struct {
	ID                    uint       `json:"id"`
	UnifiedReleaseID      uint       `json:"unified_release_id,omitempty"`
	ServiceID             uint       `json:"service_id"`
	DeploymentID          uint       `json:"deployment_id"`
	Env                   string     `json:"env"`
	RuntimeType           string     `json:"runtime_type"`
	Version               string     `json:"version"`
	Strategy              string     `json:"strategy"`
	Status                string     `json:"status"`
	TriggeredBy           uint       `json:"triggered_by"`
	ApprovedBy            uint       `json:"approved_by"`
	ApprovalComment       string     `json:"approval_comment"`
	RollbackFromReleaseID uint       `json:"rollback_from_release_id"`
	Diagnostics           any        `json:"diagnostics"`
	TriggerSource         string     `json:"trigger_source,omitempty"`
	TriggerContext        any        `json:"trigger_context,omitempty"`
	CIRunID               uint       `json:"ci_run_id,omitempty"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type ReleaseApprovalResp struct {
	ID         uint      `json:"id"`
	ReleaseID  uint      `json:"release_id"`
	ApproverID uint      `json:"approver_id"`
	Decision   string    `json:"decision"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type ReleaseTimelineEventResp struct {
	ID           uint      `json:"id"`
	ServiceID    uint      `json:"service_id"`
	DeploymentID uint      `json:"deployment_id"`
	ReleaseID    uint      `json:"release_id"`
	EventType    string    `json:"event_type"`
	ActorID      uint      `json:"actor_id"`
	Payload      any       `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
}
