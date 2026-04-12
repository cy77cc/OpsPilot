// Package infrastructure 提供基础设施相关的工具实现。
//
// 本文件实现基础设施操作工具集，包括：
//   - 集群凭证列表查询
//   - 凭证连通性测试
package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

func serviceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}

// =============================================================================
// 输入类型定义
// =============================================================================

// CredentialListInput 凭证列表查询输入。
type CredentialListInput struct {
	Type    string `json:"type,omitempty" jsonschema_description:"credential type or runtime type"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/endpoint"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max credentials,default=50"`
}

// CredentialTestInput 凭证测试输入。
type CredentialTestInput struct {
	CredentialID int `json:"credential_id" jsonschema_description:"required,credential id"`
}

// NewInfrastructureTools 创建所有基础设施工具。
//
// 基础设施工具全部为只读工具，不修改任何状态。
func NewInfrastructureTools(ctx context.Context) []tool.InvokableTool {
	return NewInfrastructureReadonlyTools(ctx)
}

// NewInfrastructureReadonlyTools 创建基础设施只读工具子集。
//
// 返回只读工具列表，包括：
//   - 凭证列表查询（credential_list）
//   - 凭证连通性测试（credential_test）
//
// 这些工具不修改任何状态，可安全用于诊断场景。
func NewInfrastructureReadonlyTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		CredentialList(ctx),
		CredentialTest(ctx),
	}
}

type CredentialListOutput struct {
	Total int              `json:"total"`
	List  []map[string]any `json:"list"`
}

func CredentialList(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"credential_list",
		"Query cluster credential list for accessing Kubernetes clusters or other infrastructure. Optional parameters: type filters by runtime type or source (k8s/helm/compose), keyword searches by name or endpoint, limit controls max results (default 50, max 200). Returns credentials with id, name, runtime type, endpoint, status, and last test result. Use credential IDs for deployment target configuration. Example: {\"type\":\"k8s\",\"limit\":20}.",
		func(ctx context.Context, input *CredentialListInput, opts ...tool.Option) (*CredentialListOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.ClusterCredential{})
			if t := strings.TrimSpace(input.Type); t != "" {
				query = query.Where("runtime_type = ? OR source = ?", t, t)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR endpoint LIKE ?", pattern, pattern)
			}
			var rows []model.ClusterCredential
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(rows))
			for _, item := range rows {
				list = append(list, map[string]any{
					"id":                item.ID,
					"name":              item.Name,
					"runtime_type":      item.RuntimeType,
					"source":            item.Source,
					"endpoint":          item.Endpoint,
					"status":            item.Status,
					"last_test_at":      item.LastTestAt,
					"last_test_status":  item.LastTestStatus,
					"last_test_message": item.LastTestMessage,
				})
			}
			return &CredentialListOutput{
				Total: len(list),
				List:  list,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type CredentialTestOutput struct {
	CredentialID    uint   `json:"credential_id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	LastTestAt      string `json:"last_test_at"`
	LastTestStatus  string `json:"last_test_status"`
	LastTestMessage string `json:"last_test_message"`
}

func CredentialTest(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"credential_test",
		"Get credential connectivity test result. credential_id is required. Returns the last test result including test timestamp, status (success/failed), and any error message. Use this to verify if a credential is valid before using it for deployment. Example: {\"credential_id\":5}.",
		func(ctx context.Context, input *CredentialTestInput, opts ...tool.Option) (*CredentialTestOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.CredentialID <= 0 {
				return nil, fmt.Errorf("credential_id is required")
			}
			var cred model.ClusterCredential
			if err := svcCtx.DB.First(&cred, input.CredentialID).Error; err != nil {
				return nil, err
			}
			return &CredentialTestOutput{
				CredentialID:    cred.ID,
				Name:            cred.Name,
				Status:          cred.Status,
				LastTestAt:      cred.LastTestAt.Format("2006-01-02 15:04:05"),
				LastTestStatus:  cred.LastTestStatus,
				LastTestMessage: cred.LastTestMessage,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}
