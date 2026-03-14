// Package v1 定义 AIOps 服务 HTTP 接口的请求数据结构。
package v1

// RunInspectionRequest is the request body for triggering an AI-powered deployment inspection.
type RunInspectionRequest struct {
	ReleaseID uint   `json:"release_id"`
	TargetID  uint   `json:"target_id"`
	ServiceID uint   `json:"service_id"`
	Stage     string `json:"stage"`
}

// ApplyPreviewRequest is the request body for previewing AIOPS recommendation application.
type ApplyPreviewRequest struct {
	InspectionID uint `json:"inspection_id"`
}
