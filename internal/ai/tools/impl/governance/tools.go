package governance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	"github.com/cy77cc/k8s-manage/internal/model"
)

// Input types

type UserListInput struct {
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional username/email keyword"`
	Status  int    `json:"status,omitempty" jsonschema_description:"optional status filter"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max users,default=50"`
}

type RoleListInput struct {
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional role keyword"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max roles,default=50"`
}

type PermissionCheckInput struct {
	UserID   int    `json:"user_id" jsonschema_description:"required,user id"`
	Resource string `json:"resource" jsonschema_description:"required,resource name"`
	Action   string `json:"action" jsonschema_description:"required,action name"`
}

type TopologyGetInput struct {
	ServiceID int `json:"service_id,omitempty" jsonschema_description:"optional service id"`
	Depth     int `json:"depth,omitempty" jsonschema_description:"max depth,default=2"`
}

type AuditLogSearchInput struct {
	TimeRange    string `json:"time_range,omitempty" jsonschema_description:"time range,default=24h"`
	ResourceType string `json:"resource_type,omitempty" jsonschema_description:"optional resource type"`
	Action       string `json:"action,omitempty" jsonschema_description:"optional action type"`
	UserID       int    `json:"user_id,omitempty" jsonschema_description:"optional actor user id"`
	Limit        int    `json:"limit,omitempty" jsonschema_description:"max logs,default=50"`
}

// NewGovernanceTools returns all governance tools.
func NewGovernanceTools(ctx context.Context, deps core.PlatformDeps) []tool.InvokableTool {
	return []tool.InvokableTool{
		UserList(ctx, deps),
		RoleList(ctx, deps),
		PermissionCheck(ctx, deps),
		TopologyGet(ctx, deps),
		AuditLogSearch(ctx, deps),
	}
}

type UserListOutput struct {
	Total int          `json:"total"`
	List  []model.User `json:"list"`
}

func UserList(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"user_list",
		"Query the list of users in the platform. Optional parameters: keyword searches by username or email, status filters by user status (0=disabled, 1=enabled), limit controls max results (default 50, max 200). Returns users with id, username, email, role information, and status. Use this to find user IDs for permission checks. Example: {\"keyword\":\"admin\",\"status\":1}.",
		func(ctx context.Context, input *UserListInput, opts ...tool.Option) (*UserListOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.User{})
			if input.Status != 0 {
				query = query.Where("status = ?", input.Status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("username LIKE ? OR email LIKE ?", pattern, pattern)
			}
			var users []model.User
			if err := query.Order("id desc").Limit(limit).Find(&users).Error; err != nil {
				return nil, err
			}
			return &UserListOutput{
				Total: len(users),
				List:  users,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type RoleListOutput struct {
	Total int          `json:"total"`
	List  []model.Role `json:"list"`
}

func RoleList(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"role_list",
		"Query the list of roles in the platform. Optional parameters: keyword searches by role name or code, limit controls max results (default 50, max 200). Returns roles with id, name, code, description, and permission count. Use this to understand available roles for user assignment. Example: {\"keyword\":\"admin\"}.",
		func(ctx context.Context, input *RoleListInput, opts ...tool.Option) (*RoleListOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.Role{})
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR code LIKE ?", pattern, pattern)
			}
			var roles []model.Role
			if err := query.Order("id desc").Limit(limit).Find(&roles).Error; err != nil {
				return nil, err
			}
			return &RoleListOutput{
				Total: len(roles),
				List:  roles,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type PermissionCheckOutput struct {
	Allowed            bool               `json:"allowed"`
	Reason             string             `json:"reason,omitempty"`
	MatchedPermissions []model.Permission `json:"matched_permissions,omitempty"`
	Checked            map[string]any     `json:"checked"`
}

func PermissionCheck(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"permission_check",
		"Check if a user has a specific permission. user_id, resource, and action are required. Returns whether the permission is granted, matched permissions if any, and the checked parameters. Use this to verify user access before performing sensitive operations. Example: {\"user_id\":1,\"resource\":\"service\",\"action\":\"delete\"}.",
		func(ctx context.Context, input *PermissionCheckInput, opts ...tool.Option) (*PermissionCheckOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			if input.UserID <= 0 {
				return nil, fmt.Errorf("user_id is required")
			}
			resource := strings.TrimSpace(input.Resource)
			action := strings.TrimSpace(input.Action)
			if resource == "" {
				return nil, fmt.Errorf("resource is required")
			}
			if action == "" {
				return nil, fmt.Errorf("action is required")
			}

			var roleIDs []int64
			if err := deps.DB.Model(&model.UserRole{}).Where("user_id = ?", input.UserID).Pluck("role_id", &roleIDs).Error; err != nil {
				return nil, err
			}
			if len(roleIDs) == 0 {
				return &PermissionCheckOutput{
					Allowed: false,
					Reason:  "user has no roles",
					Checked: map[string]any{"user_id": input.UserID, "resource": resource, "action": action},
				}, nil
			}
			var permIDs []int64
			if err := deps.DB.Model(&model.RolePermission{}).Where("role_id IN ?", roleIDs).Pluck("permission_id", &permIDs).Error; err != nil {
				return nil, err
			}
			if len(permIDs) == 0 {
				return &PermissionCheckOutput{
					Allowed: false,
					Reason:  "roles have no permissions",
					Checked: map[string]any{"user_id": input.UserID, "resource": resource, "action": action},
				}, nil
			}
			var perms []model.Permission
			if err := deps.DB.Where("id IN ?", permIDs).Find(&perms).Error; err != nil {
				return nil, err
			}
			matched := make([]model.Permission, 0)
			for _, perm := range perms {
				if strings.EqualFold(strings.TrimSpace(perm.Resource), resource) && strings.EqualFold(strings.TrimSpace(perm.Action), action) {
					matched = append(matched, perm)
				}
			}
			return &PermissionCheckOutput{
				Allowed:            len(matched) > 0,
				MatchedPermissions: matched,
				Checked:            map[string]any{"user_id": input.UserID, "resource": resource, "action": action},
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type TopologyGetOutput struct {
	Nodes []map[string]any `json:"nodes"`
	Edges []map[string]any `json:"edges"`
	Depth int              `json:"depth"`
}

func TopologyGet(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"topology_get",
		"Query service topology showing relationships between services and deployment targets. Optional parameters: service_id focuses topology on a specific service, depth controls how many levels of relationships to explore (default 2, max 5). Returns nodes (services/targets) and edges (deployment relationships). Use this to understand service dependencies. Example: {\"service_id\":12,\"depth\":3}.",
		func(ctx context.Context, input *TopologyGetInput, opts ...tool.Option) (*TopologyGetOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			depth := input.Depth
			if depth <= 0 {
				depth = 2
			}
			services := make([]model.Service, 0)
			query := deps.DB.Model(&model.Service{})
			if input.ServiceID > 0 {
				query = query.Where("id = ?", input.ServiceID)
			}
			if err := query.Order("id desc").Limit(100).Find(&services).Error; err != nil {
				return nil, err
			}
			nodes := make([]map[string]any, 0, len(services))
			edges := make([]map[string]any, 0)
			for _, svc := range services {
				nodes = append(nodes, map[string]any{"id": fmt.Sprintf("service-%d", svc.ID), "type": "service", "label": svc.Name, "service_id": svc.ID})
				var releases []model.DeploymentRelease
				_ = deps.DB.Where("service_id = ?", svc.ID).Order("id desc").Limit(depth).Find(&releases).Error
				for _, rel := range releases {
					targetNodeID := fmt.Sprintf("target-%d", rel.TargetID)
					edges = append(edges, map[string]any{"from": fmt.Sprintf("service-%d", svc.ID), "to": targetNodeID, "type": "deploy"})
				}
			}
			return &TopologyGetOutput{
				Nodes: nodes,
				Edges: edges,
				Depth: depth,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type AuditLogSearchOutput struct {
	Total int              `json:"total"`
	List  []model.AuditLog `json:"list"`
}

func AuditLogSearch(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"audit_log_search",
		"Search audit logs for platform activities. Optional parameters: time_range filters logs within a duration (default 24h, accepts values like 1h, 6h, 24h, 7d), resource_type filters by resource kind (service/cluster/host), action filters by action type (create/update/delete), user_id filters by actor, limit controls max results (default 50, max 200). Returns audit entries with timestamps and details. Example: {\"time_range\":\"24h\",\"resource_type\":\"service\"}.",
		func(ctx context.Context, input *AuditLogSearchInput, opts ...tool.Option) (*AuditLogSearchOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			since := time.Now().Add(-parseTimeRange(strings.TrimSpace(input.TimeRange), 24*time.Hour))
			query := deps.DB.Model(&model.AuditLog{}).Where("created_at >= ?", since)
			if rt := strings.TrimSpace(input.ResourceType); rt != "" {
				query = query.Where("resource_type = ?", rt)
			}
			if action := strings.TrimSpace(input.Action); action != "" {
				query = query.Where("action_type = ?", action)
			}
			if input.UserID > 0 {
				query = query.Where("actor_id = ?", input.UserID)
			}
			var rows []model.AuditLog
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			return &AuditLogSearchOutput{
				Total: len(rows),
				List:  rows,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func parseTimeRange(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
