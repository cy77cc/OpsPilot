package v1

// RolloutApplyReq is the request body for RolloutPreview and RolloutApply handlers.
type RolloutApplyReq struct {
	Namespace     string            `json:"namespace" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	Image         string            `json:"image" binding:"required"`
	Replicas      int32             `json:"replicas"`
	Strategy      string            `json:"strategy"`
	Labels        map[string]string `json:"labels"`
	CanarySteps   []map[string]any  `json:"canary_steps"`
	ActiveService string            `json:"active_service"`
	PreviewSvc    string            `json:"preview_service"`
	ApprovalToken string            `json:"approval_token"`
}
