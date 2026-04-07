// Package handler 提供 Kubernetes 集群管理的 HTTP Handler 实现。
//
// 本文件实现集群高风险操作的统一审批/审计辅助方法，以及操作历史查询接口。
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	clusterlogic "github.com/cy77cc/OpsPilot/internal/service/cluster/logic"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	governanceaudit "github.com/cy77cc/OpsPilot/internal/service/governance/audit"
	"github.com/gin-gonic/gin"
)

// Import types from logic
type OperationApproval = clusterlogic.OperationApproval
type OperationResponse = clusterlogic.OperationResponse

// Re-export constants from logic
const (
	OperationCodeSuccess          = clusterlogic.OperationCodeSuccess
	OperationCodeApprovalRequired = clusterlogic.OperationCodeApprovalRequired
	OperationCodeApprovalRejected = clusterlogic.OperationCodeApprovalRejected
)

const (
	// 与 operation_response.go 保持同一业务码来源，避免契约漂移。
	clusterOperationCodeSuccess          = OperationCodeSuccess
	clusterOperationCodeApprovalRequired = OperationCodeApprovalRequired
	clusterOperationCodeApprovalRejected = OperationCodeApprovalRejected
)

var sensitiveOperationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9\-._~+/=]+`),
	regexp.MustCompile(`(?i)\b(authorization|token|password|secret|key|cert|kubeconfig)\b\s*[:=]\s*([^\s,;]+)`),
}

// ClusterOperationResponse 是高风险操作的统一响应体。
type ClusterOperationResponse struct {
	State          string             `json:"state"`              // 操作状态
	Approval       *OperationApproval `json:"approval,omitempty"` // 审批信息
	AuditID        uint               `json:"audit_id,omitempty"` // 审计 ID
	Code           string             `json:"code,omitempty"`     // 业务结果码
	Message        string             `json:"message,omitempty"`  // 结果消息
	Data           any                `json:"data,omitempty"`     // 结果数据
	ClusterID      uint               `json:"-"`                  // 兼容旧字段
	Resource       string             `json:"-"`                  // 兼容旧字段
	ResourceID     string             `json:"-"`                  // 兼容旧字段
	ApprovalTicket string             `json:"-"`                  // 兼容旧字段
	Diagnostics    string             `json:"-"`                  // 兼容旧字段
	Details        map[string]any     `json:"-"`                  // 兼容旧字段
}

// OperationHistoryItem 是操作历史列表项。
type OperationHistoryItem struct {
	AuditID      uint      `json:"audit_id"`
	ClusterID    uint      `json:"cluster_id"`
	Namespace    string    `json:"namespace"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type,omitempty"`
	ResourceName string    `json:"resource_name,omitempty"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resource_id"`
	Target       string    `json:"target,omitempty"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	Diagnostics  string    `json:"diagnostics,omitempty"`
	OperatorID   uint      `json:"operator_id"`
	Operator     string    `json:"operator"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}

// OperationHistoryResponse 是操作历史分页响应。
type OperationHistoryResponse struct {
	List       []OperationHistoryItem `json:"list"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// OperationAuditDetail 是集群操作审计详情响应。
type OperationAuditDetail struct {
	OperationHistoryItem
	Approval    *OperationApproval `json:"approval,omitempty"`
	Request     map[string]any     `json:"request,omitempty"`
	Response    map[string]any     `json:"response,omitempty"`
	Diagnostics []any              `json:"diagnostics,omitempty"`
}

// OperationGateResult 表示高风险操作的审批门禁结果。
type OperationGateResult struct {
	Allowed        bool
	Code           string
	Message        string
	ApprovalTicket string
	AuditID        uint
}

// RecordClusterOperationAudit 写入集群操作审计。
func (h *Handler) RecordClusterOperationAudit(ctx context.Context, clusterID uint, namespace, action, resource, resourceID, status, message string, operatorID uint) (model.ClusterOperationAudit, error) {
	rec, err := h.recordClusterOperationAuditWithCode(ctx, clusterID, namespace, action, resource, resourceID, status, "", message, operatorID)
	if err != nil {
		return model.ClusterOperationAudit{}, err
	}
	return *rec, nil
}

func (h *Handler) recordClusterOperationAuditWithCode(ctx context.Context, clusterID uint, namespace, action, resource, resourceID, status, code, message string, operatorID uint) (*model.ClusterOperationAudit, error) {
	msg := truncateOperationText(sanitizeOperationText(message), 255)
	finalCode := strings.TrimSpace(code)
	if finalCode == "" {
		finalCode = clusterStatusToGovernanceCode(status)
	}

	govAudit := governanceaudit.NewService(h.svcCtx.DB, nil)
	id, err := govAudit.Record(ctx, governance.FinalizeInput{
		Intent: governance.OperationIntent{
			OperatorID: operatorID,
			Scope: governance.Scope{
				Domain:     "cluster",
				ClusterID:  clusterID,
				Namespace:  strings.TrimSpace(namespace),
				Resource:   strings.TrimSpace(resource),
				ResourceID: strings.TrimSpace(resourceID),
				Action:     strings.TrimSpace(action),
			},
		},
		Decision: governance.Decision{
			State:   clusterStatusToGovernanceState(status),
			Code:    finalCode,
			Message: msg,
		},
		ExecutionCode: finalCode,
		ExecutionMsg:  msg,
		Diagnostics: map[string]any{
			"message": msg,
		},
	})
	if err != nil {
		return nil, err
	}

	return &model.ClusterOperationAudit{
		ID:         id,
		ClusterID:  clusterID,
		Namespace:  strings.TrimSpace(namespace),
		Action:     strings.TrimSpace(action),
		Resource:   strings.TrimSpace(resource),
		ResourceID: strings.TrimSpace(resourceID),
		Status:     strings.TrimSpace(status),
		Message:    msg,
		OperatorID: operatorID,
	}, nil
}

// requireHighRiskApproval 统一处理高风险操作的审批门禁。
//
// 如果操作者拥有审批权限则直接放行；否则根据 approval_token 验证审批票据。
// 当没有票据时，会创建一条待审批记录并返回 approval_required。
func (h *Handler) requireHighRiskApproval(ctx context.Context, clusterID uint, namespace, action, resource, resourceID, approvalToken string, operatorID uint64) OperationGateResult {
	if httpx.HasAnyPermission(h.svcCtx.DB, operatorID, "k8s:approve", "kubernetes:approve") {
		return OperationGateResult{Allowed: true, Code: clusterOperationCodeSuccess}
	}

	token := strings.TrimSpace(approvalToken)
	scope := NormalizeApprovalScope(ApprovalScope{
		ClusterID:  clusterID,
		Namespace:  namespace,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
	})
	if token == "" {
		rec, err := IssueClusterDeployApproval(ctx, h.svcCtx.DB, scope, uint(operatorID), time.Now().UTC().Add(30*time.Minute))
		if err != nil {
			return OperationGateResult{Allowed: false, Code: clusterOperationCodeApprovalRequired, Message: clusterOperationCodeApprovalRequired}
		}
		audit, err := h.RecordClusterOperationAudit(ctx, clusterID, namespace, action, resource, resourceID, "pending", clusterOperationCodeApprovalRequired, uint(operatorID))
		if err != nil {
			return OperationGateResult{Allowed: false, Code: clusterOperationCodeApprovalRequired, Message: clusterOperationCodeApprovalRequired, ApprovalTicket: rec.Ticket}
		}
		return OperationGateResult{
			Allowed:        false,
			Code:           clusterOperationCodeApprovalRequired,
			Message:        clusterOperationCodeApprovalRequired,
			ApprovalTicket: rec.Ticket,
			AuditID:        audit.ID,
		}
	}

	_, err := ConsumeClusterDeployApproval(ctx, h.svcCtx.DB, token, scope, uint(operatorID), time.Now().UTC())
	if err != nil {
		approvalErr, ok := IsApprovalError(err)
		if !ok {
			audit, auditErr := h.RecordClusterOperationAudit(ctx, clusterID, namespace, action, resource, resourceID, "failed", err.Error(), uint(operatorID))
			if auditErr != nil {
				return OperationGateResult{Allowed: false, Code: OperationCodeFailed, Message: sanitizeOperationText(err.Error())}
			}
			return OperationGateResult{Allowed: false, Code: OperationCodeFailed, Message: sanitizeOperationText(err.Error()), AuditID: audit.ID}
		}
		status := "failed"
		switch approvalErr.Code {
		case clusterOperationCodeApprovalRequired:
			status = "pending"
		case clusterOperationCodeApprovalRejected:
			status = "rejected"
		}
		audit, auditErr := h.recordClusterOperationAuditWithCode(ctx, clusterID, namespace, action, resource, resourceID, status, approvalErr.Code, approvalErr.Error(), uint(operatorID))
		if auditErr != nil || audit == nil {
			return OperationGateResult{Allowed: false, Code: approvalErr.Code, Message: approvalErr.Error()}
		}
		return OperationGateResult{Allowed: false, Code: approvalErr.Code, Message: approvalErr.Error(), AuditID: audit.ID}
	}
	return OperationGateResult{Allowed: true, Code: clusterOperationCodeSuccess}
}

// ListOperationHistory 获取集群操作历史。
func (h *Handler) ListOperationHistory(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, nil)
		return
	}

	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	page := httpx.UintFromQuery(c, "page")
	if page == 0 {
		page = 1
	}
	pageSize := httpx.UintFromQuery(c, "page_size")
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	resource := strings.TrimSpace(c.Query("resource"))
	status := strings.TrimSpace(c.Query("status"))
	operator := strings.TrimSpace(c.Query("operator"))
	from, err := parseOperationHistoryTime(c.Query("from"))
	if err != nil {
		httpx.BindErr(c, err)
		return
	}
	to, err := parseOperationHistoryTime(c.Query("to"))
	if err != nil {
		httpx.BindErr(c, err)
		return
	}

	query := h.svcCtx.DB.WithContext(c.Request.Context()).Model(&model.OperationAudit{}).
		Where("domain = ? AND scope_cluster_id = ?", "cluster", clusterID)
	if resource != "" {
		query = query.Where("operation_audits.resource = ?", resource)
	}
	if status != "" {
		query = query.Where("LOWER(operation_audits.status) IN ?", operationHistoryStatusesForFilter(status))
	}
	if from != nil {
		query = query.Where("operation_audits.created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("operation_audits.created_at <= ?", *to)
	}
	if operator != "" {
		if operatorID, parseErr := strconv.ParseUint(operator, 10, 64); parseErr == nil {
			query = query.Where("operator_id = ?", operatorID)
		} else {
			query = query.Joins("LEFT JOIN users ON users.id = operation_audits.operator_id").Where("users.username = ?", operator)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	currentPage := int(page)
	if totalPages == 0 {
		currentPage = 1
	} else if currentPage > totalPages {
		currentPage = totalPages
	}

	offset := (currentPage - 1) * int(pageSize)
	var rows []model.OperationAudit
	if err := query.Order("operation_audits.id DESC").Offset(offset).Limit(int(pageSize)).Find(&rows).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := h.operationAuditsToHistoryItems(c.Request.Context(), rows)
	httpx.OK(c, OperationHistoryResponse{
		List:       items,
		Total:      total,
		Page:       currentPage,
		PageSize:   int(pageSize),
		TotalPages: totalPages,
	})
}

// GetOperationAudit 获取单条集群操作审计详情。
func (h *Handler) GetOperationAudit(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	auditID := httpx.UintFromParam(c, "audit_id")
	if clusterID == 0 || auditID == 0 {
		httpx.BindErr(c, nil)
		return
	}

	var row model.OperationAudit
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("domain = ? AND scope_cluster_id = ? AND id = ?", "cluster", clusterID, auditID).
		First(&row).Error; err != nil {
		httpx.NotFound(c, "operation audit not found")
		return
	}

	detail := h.operationAuditToDetail(c.Request.Context(), row)
	if detail == nil {
		httpx.ServerErr(c, fmt.Errorf("build operation audit detail"))
		return
	}
	httpx.OK(c, detail)
}

func (h *Handler) operationAuditsToHistoryItems(ctx context.Context, rows []model.OperationAudit) []OperationHistoryItem {
	operatorIDs := make([]uint, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, row := range rows {
		if row.OperatorID == 0 {
			continue
		}
		if _, ok := seen[row.OperatorID]; ok {
			continue
		}
		seen[row.OperatorID] = struct{}{}
		operatorIDs = append(operatorIDs, row.OperatorID)
	}

	operatorNames := map[uint]string{}
	if len(operatorIDs) > 0 {
		var users []model.User
		if err := h.svcCtx.DB.WithContext(ctx).Select("id", "username").Where("id IN ?", operatorIDs).Find(&users).Error; err == nil {
			for _, user := range users {
				operatorNames[uint(user.ID)] = strings.TrimSpace(user.Username)
			}
		}
	}

	items := make([]OperationHistoryItem, 0, len(rows))
	for _, row := range rows {
		operator := operatorNames[row.OperatorID]
		if operator == "" {
			switch row.OperatorID {
			case 0:
				operator = "system"
			default:
				operator = fmt.Sprintf("user#%d", row.OperatorID)
			}
		}
		message := sanitizeOperationText(row.Message)
		item := OperationHistoryItem{
			AuditID:    row.ID,
			ClusterID:  uintValue(row.ScopeClusterID),
			Namespace:  row.Namespace,
			Action:     row.Action,
			Resource:   row.Resource,
			ResourceID: row.ResourceID,
			Status:     normalizeOperationAuditStatus(row.Status),
			Message:    message,
			OperatorID: row.OperatorID,
			Operator:   operator,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		item.Target = row.ResourceID
		item.ResourceName = row.ResourceID
		item.ResourceType = row.Resource
		if row.Resource == PolicyReleaseApprovalResource {
			if resourceName, target := policyReleaseAuditDisplay(row); resourceName != "" || target != "" {
				if resourceName != "" {
					item.ResourceName = resourceName
				}
				if target != "" {
					item.Target = target
				}
			}
		}
		if item.Status != OperationStateCompleted && message != "" {
			item.Diagnostics = message
		}
		items = append(items, item)
	}
	return items
}

func (h *Handler) operationAuditToDetail(ctx context.Context, row model.OperationAudit) *OperationAuditDetail {
	item := h.operationAuditsToHistoryItems(ctx, []model.OperationAudit{row})
	if len(item) == 0 {
		return nil
	}
	detail := &OperationAuditDetail{
		OperationHistoryItem: item[0],
		Request:              decodeJSONStringMap(row.RequestSummaryJSON),
		Response:             decodeJSONStringMap(row.ResultSummaryJSON),
		Diagnostics:          decodeJSONStringSlice(row.DiagnosticsJSON),
	}
	if strings.TrimSpace(row.ApprovalTicket) != "" {
		var approvalRow model.OperationApproval
		if err := h.svcCtx.DB.WithContext(ctx).Where("ticket = ?", row.ApprovalTicket).First(&approvalRow).Error; err == nil {
			detail.Approval = operationApprovalFromGovernanceRecord(&approvalRow)
		} else {
			detail.Approval = &OperationApproval{Ticket: row.ApprovalTicket}
		}
	}
	return detail
}

func operationApprovalFromGovernanceRecord(rec *model.OperationApproval) *OperationApproval {
	if rec == nil {
		return nil
	}
	return &OperationApproval{
		Ticket:        rec.Ticket,
		ClusterID:     uintValue(rec.ScopeClusterID),
		Namespace:     rec.Namespace,
		Action:        rec.Action,
		Resource:      rec.Resource,
		ResourceID:    rec.ResourceID,
		ExpiresAt:     rec.ExpiresAt,
		ConsumedAt:    rec.ConsumedAt,
		ConsumedBy:    rec.ConsumedBy,
		ReplayCount:   rec.ReplayCount,
		ReplayAt:      rec.ReplayAt,
		ReplayBy:      rec.ReplayBy,
		ReplayCode:    rec.ReplayCode,
		ReplayMessage: rec.ReplayMessage,
		Status:        rec.Status,
	}
}

func normalizeOperationAuditStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approval_required", "pending":
		return OperationStateApprovalRequired
	case "rejected":
		return OperationStateRejected
	case "failed":
		return OperationStateFailed
	case "success":
		return OperationStateCompleted
	default:
		return strings.TrimSpace(status)
	}
}

func operationHistoryStatusesForFilter(status string) []string {
	switch normalizeOperationAuditStatus(status) {
	case OperationStateApprovalRequired:
		return []string{"approval_required", "pending"}
	case OperationStateCompleted:
		return []string{"completed", "success"}
	case OperationStateRejected:
		return []string{"rejected"}
	case OperationStateFailed:
		return []string{"failed"}
	default:
		return []string{strings.ToLower(strings.TrimSpace(status))}
	}
}

func decodeJSONStringMap(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{"raw": sanitizeOperationText(raw)}
	}
	return out
}

func decodeJSONStringSlice(raw string) []any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []any
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		return out
	}
	var single any
	if err := json.Unmarshal([]byte(raw), &single); err == nil {
		return []any{single}
	}
	return []any{sanitizeOperationText(raw)}
}

func policyReleaseAuditDisplay(row model.OperationAudit) (string, string) {
	resourceName, version := policyReleaseAuditFields(decodeJSONStringMap(row.ResultSummaryJSON))
	if resourceName != "" || version != "" {
		return resourceName, version
	}
	return policyReleaseAuditFields(decodeJSONStringMap(row.RequestSummaryJSON))
}

func policyReleaseAuditFields(payload map[string]any) (string, string) {
	if len(payload) == 0 {
		return "", ""
	}

	releaseValue, ok := payload["release"]
	if !ok {
		return stringValue(payload["policy_name"]), stringValue(payload["version"])
	}

	releaseMap, ok := releaseValue.(map[string]any)
	if !ok {
		return stringValue(payload["policy_name"]), stringValue(payload["version"])
	}

	policyMap, _ := releaseMap["policy"].(map[string]any)
	return stringValue(policyMap["name"]), stringValue(releaseMap["version"])
}

func stringValue(v any) string {
	value, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func uintValue(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func parseOperationHistoryTime(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		t := ts.UTC()
		return &t, nil
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		t := ts.UTC()
		return &t, nil
	}
	if secs, err := strconv.ParseInt(value, 10, 64); err == nil {
		t := time.Unix(secs, 0).UTC()
		return &t, nil
	}
	return nil, fmt.Errorf("invalid time format: %s", value)
}

func sanitizeOperationText(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return text
	}
	for _, pattern := range sensitiveOperationPatterns {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			lower := strings.ToLower(match)
			if strings.HasPrefix(lower, "bearer ") {
				return "Bearer ***"
			}
			parts := strings.FieldsFunc(match, func(r rune) bool {
				return r == ':' || r == '='
			})
			if len(parts) > 0 {
				return parts[0] + "=***"
			}
			return "***"
		})
	}
	return text
}

func truncateOperationText(input string, max int) string {
	if max <= 0 || len(input) <= max {
		return input
	}
	return input[:max]
}

// operationResponseFromGate 将审批门禁结果转换为响应。
func operationResponseFromGate(clusterID uint, resource, resourceID string, gate OperationGateResult) ClusterOperationResponse {
	state := operationStateFromGateCode(gate.Code)
	return ClusterOperationResponse{
		State:          state,
		Approval:       &OperationApproval{Ticket: gate.ApprovalTicket},
		AuditID:        gate.AuditID,
		Code:           gate.Code,
		Message:        gate.Message,
		ClusterID:      clusterID,
		Resource:       resource,
		ResourceID:     resourceID,
		ApprovalTicket: gate.ApprovalTicket,
	}
}

// operationSuccessResponse 构造成功响应。
func operationSuccessResponse(clusterID uint, resource, resourceID, message string, auditID uint, details map[string]any) ClusterOperationResponse {
	return ClusterOperationResponse{
		State:      OperationStateCompleted,
		AuditID:    auditID,
		Code:       clusterOperationCodeSuccess,
		Message:    message,
		Data:       details,
		ClusterID:  clusterID,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
	}
}

func operationStateFromGateCode(code string) string {
	switch code {
	case clusterOperationCodeApprovalRequired:
		return OperationStateApprovalRequired
	case clusterOperationCodeApprovalRejected:
		return OperationStateRejected
	default:
		return OperationStateFailed
	}
}
