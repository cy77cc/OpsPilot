// Package planner 实现 AI 编排的规划阶段。
//
// 本文件提供资源解析和辅助函数。
package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// ResolveStatus 表示资源解析状态。
type ResolveStatus string

const (
	ResolveStatusExact     ResolveStatus = "exact"     // 精确匹配
	ResolveStatusAmbiguous ResolveStatus = "ambiguous" // 存在歧义
	ResolveStatusMissing   ResolveStatus = "missing"   // 未找到
)

// ResolveCandidate 表示资源解析的候选结果。
type ResolveCandidate struct {
	ID     string `json:"id"`              // 资源 ID
	Name   string `json:"name"`            // 资源名称
	Score  int    `json:"score"`           // 匹配分数
	Reason string `json:"reason,omitempty"` // 匹配原因
}

// ResolveResult 表示资源解析结果。
type ResolveResult struct {
	Status     ResolveStatus      `json:"status"`           // 解析状态
	Query      string             `json:"query"`            // 查询字符串
	Selected   *ResolveCandidate  `json:"selected,omitempty"` // 选中的候选
	Candidates []ResolveCandidate `json:"candidates,omitempty"` // 候选列表
}

// UserContext 表示用户上下文信息。
type UserContext struct {
	UserID      uint64   `json:"user_id"`          // 用户 ID
	Username    string   `json:"username,omitempty"` // 用户名
	IsAdmin     bool     `json:"is_admin"`          // 是否管理员
	RoleCodes   []string `json:"role_codes,omitempty"` // 角色代码
	ProjectID   string   `json:"project_id,omitempty"` // 项目 ID
	Scene       string   `json:"scene,omitempty"`
	CurrentPage string   `json:"current_page,omitempty"`
}

type InventoryItem struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Data map[string]any `json:"data,omitempty"`
}

type InventoryList struct {
	Kind  string          `json:"kind"`
	Total int             `json:"total"`
	Items []InventoryItem `json:"items"`
}

type SupportTools struct {
	DB *gorm.DB
}

func NewSupportTools(db *gorm.DB) *SupportTools {
	return &SupportTools{DB: db}
}

func (s *SupportTools) ResolveService(ctx context.Context, query string) (*ResolveResult, error) {
	return resolveByQuery(ctx, s.DB, query, &[]model.Service{}, func(db *gorm.DB, pattern string) *gorm.DB {
		return db.Model(&model.Service{}).Where("name LIKE ? OR owner LIKE ?", pattern, pattern).Order("id desc")
	}, func(item model.Service) ResolveCandidate {
		return ResolveCandidate{
			ID:     fmt.Sprintf("%d", item.ID),
			Name:   item.Name,
			Score:  scoreCandidate(item.Name, query),
			Reason: "service inventory match",
		}
	})
}

func (s *SupportTools) ResolveCluster(ctx context.Context, query string) (*ResolveResult, error) {
	return resolveByQuery(ctx, s.DB, query, &[]model.Cluster{}, func(db *gorm.DB, pattern string) *gorm.DB {
		return db.Model(&model.Cluster{}).Where("name LIKE ? OR endpoint LIKE ?", pattern, pattern).Order("id desc")
	}, func(item model.Cluster) ResolveCandidate {
		return ResolveCandidate{
			ID:     fmt.Sprintf("%d", item.ID),
			Name:   item.Name,
			Score:  scoreCandidate(item.Name, query),
			Reason: "cluster inventory match",
		}
	})
}

func (s *SupportTools) ResolveHost(ctx context.Context, query string) (*ResolveResult, error) {
	return resolveByQuery(ctx, s.DB, query, &[]model.Node{}, func(db *gorm.DB, pattern string) *gorm.DB {
		return db.Model(&model.Node{}).Where("name LIKE ? OR ip LIKE ? OR hostname LIKE ?", pattern, pattern, pattern).Order("id desc")
	}, func(item model.Node) ResolveCandidate {
		name := firstNonEmpty(item.Name, item.Hostname, item.IP)
		return ResolveCandidate{
			ID:     fmt.Sprintf("%d", item.ID),
			Name:   name,
			Score:  scoreCandidate(name, query),
			Reason: "host inventory match",
		}
	})
}

func (s *SupportTools) CheckPermission(ctx context.Context, userID uint64, resource, action string) (bool, error) {
	if s == nil || s.DB == nil {
		return false, fmt.Errorf("db unavailable")
	}
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if userID == 0 || resource == "" || action == "" {
		return false, nil
	}
	return httpx.HasAnyPermission(s.DB, userID, resource+":"+action), nil
}

func (s *SupportTools) GetUserContext(ctx context.Context, userID uint64, runtimeCtx map[string]any) (*UserContext, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	out := &UserContext{
		UserID:      userID,
		IsAdmin:     httpx.IsAdmin(s.DB, userID),
		ProjectID:   stringValue(runtimeCtx["project_id"]),
		Scene:       stringValue(runtimeCtx["scene"]),
		CurrentPage: stringValue(runtimeCtx["current_page"]),
	}
	if userID == 0 {
		return out, nil
	}

	var user model.User
	if err := s.DB.WithContext(ctx).Select("id", "username").Where("id = ?", userID).First(&user).Error; err == nil {
		out.Username = user.Username
	}

	var codes []string
	if err := s.DB.WithContext(ctx).Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Pluck("roles.code", &codes).Error; err == nil {
		out.RoleCodes = dedupeStrings(codes)
	}
	return out, nil
}

func (s *SupportTools) HostListInventory(ctx context.Context, keyword, status string, limit int) (*InventoryList, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := s.DB.WithContext(ctx).Model(&model.Node{})
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR ip LIKE ? OR hostname LIKE ?", pattern, pattern, pattern)
	}
	var nodes []model.Node
	if err := query.Order("id desc").Limit(limit).Find(&nodes).Error; err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, InventoryItem{
			ID:   fmt.Sprintf("%d", node.ID),
			Name: firstNonEmpty(node.Name, node.Hostname, node.IP),
			Data: map[string]any{
				"ip":         node.IP,
				"hostname":   node.Hostname,
				"status":     node.Status,
				"ssh_user":   node.SSHUser,
				"port":       node.Port,
				"cpu_cores":  node.CpuCores,
				"memory_mb":  node.MemoryMB,
				"disk_gb":    node.DiskGB,
				"updated_at": node.UpdatedAt,
			},
		})
	}
	return &InventoryList{Kind: "host", Total: len(items), Items: items}, nil
}

func (s *SupportTools) ServiceListInventory(ctx context.Context, keyword, env, runtimeType, status string, limit int) (*InventoryList, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := s.DB.WithContext(ctx).Model(&model.Service{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR owner LIKE ?", pattern, pattern)
	}
	if env = strings.TrimSpace(env); env != "" {
		query = query.Where("env = ?", env)
	}
	if runtimeType = strings.TrimSpace(runtimeType); runtimeType != "" {
		query = query.Where("runtime_type = ?", runtimeType)
	}
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	var services []model.Service
	if err := query.Order("id desc").Limit(limit).Find(&services).Error; err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(services))
	for _, svc := range services {
		items = append(items, InventoryItem{
			ID:   fmt.Sprintf("%d", svc.ID),
			Name: svc.Name,
			Data: map[string]any{
				"owner":        svc.Owner,
				"env":          svc.Env,
				"status":       svc.Status,
				"runtime_type": svc.RuntimeType,
				"image":        svc.Image,
				"replicas":     svc.Replicas,
				"updated_at":   svc.UpdatedAt,
			},
		})
	}
	return &InventoryList{Kind: "service", Total: len(items), Items: items}, nil
}

func (s *SupportTools) ClusterListInventory(ctx context.Context, keyword, status string, limit int) (*InventoryList, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := s.DB.WithContext(ctx).Model(&model.Cluster{})
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR endpoint LIKE ?", pattern, pattern)
	}
	var clusters []model.Cluster
	if err := query.Order("id desc").Limit(limit).Find(&clusters).Error; err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(clusters))
	for _, cluster := range clusters {
		items = append(items, InventoryItem{
			ID:   fmt.Sprintf("%d", cluster.ID),
			Name: cluster.Name,
			Data: map[string]any{
				"endpoint":    cluster.Endpoint,
				"description": cluster.Description,
				"updated_at":  cluster.UpdatedAt,
			},
		})
	}
	return &InventoryList{Kind: "cluster", Total: len(items), Items: items}, nil
}

func (s *SupportTools) UserList(ctx context.Context, keyword string, status, limit int) (*InventoryList, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := s.DB.WithContext(ctx).Model(&model.User{})
	if status != 0 {
		query = query.Where("status = ?", status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ?", pattern, pattern)
	}
	var users []model.User
	if err := query.Order("id desc").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(users))
	for _, user := range users {
		items = append(items, InventoryItem{
			ID:   fmt.Sprintf("%d", user.ID),
			Name: user.Username,
			Data: map[string]any{
				"email":      user.Email,
				"status":     user.Status,
				"updated_at": user.UpdateTime,
			},
		})
	}
	return &InventoryList{Kind: "user", Total: len(items), Items: items}, nil
}

func (s *SupportTools) RoleList(ctx context.Context, keyword string, limit int) (*InventoryList, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	query := s.DB.WithContext(ctx).Model(&model.Role{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		pattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", pattern, pattern)
	}
	var roles []model.Role
	if err := query.Order("id desc").Limit(limit).Find(&roles).Error; err != nil {
		return nil, err
	}
	items := make([]InventoryItem, 0, len(roles))
	for _, role := range roles {
		items = append(items, InventoryItem{
			ID:   fmt.Sprintf("%d", role.ID),
			Name: role.Name,
			Data: map[string]any{
				"code":        role.Code,
				"description": role.Description,
				"updated_at":  role.UpdateTime,
			},
		})
	}
	return &InventoryList{Kind: "role", Total: len(items), Items: items}, nil
}

func resolveByQuery[T any](ctx context.Context, db *gorm.DB, query string, target *[]T, scope func(*gorm.DB, string) *gorm.DB, mapFn func(T) ResolveCandidate) (*ResolveResult, error) {
	if db == nil {
		return nil, fmt.Errorf("db unavailable")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return &ResolveResult{Status: ResolveStatusMissing, Query: query}, nil
	}
	pattern := "%" + query + "%"
	if err := scope(db.WithContext(ctx), pattern).Limit(5).Find(target).Error; err != nil {
		return nil, err
	}
	rows := *target
	candidates := make([]ResolveCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, mapFn(row))
	}
	switch len(candidates) {
	case 0:
		return &ResolveResult{Status: ResolveStatusMissing, Query: query}, nil
	case 1:
		return &ResolveResult{
			Status:     ResolveStatusExact,
			Query:      query,
			Selected:   &candidates[0],
			Candidates: candidates,
		}, nil
	default:
		return &ResolveResult{
			Status:     ResolveStatusAmbiguous,
			Query:      query,
			Candidates: candidates,
		}, nil
	}
}

func scoreCandidate(name, query string) int {
	name = strings.ToLower(strings.TrimSpace(name))
	query = strings.ToLower(strings.TrimSpace(query))
	switch {
	case name == query:
		return 100
	case strings.Contains(name, query):
		return 80
	default:
		return 60
	}
}

func stringValue(value any) string {
	if raw, ok := value.(string); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
