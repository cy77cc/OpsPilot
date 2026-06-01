package bootstrap

import (
	"encoding/json"
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/core/utils"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
)

// RunSeedData 初始化系统必须的基础数据。
// 包括: 角色、权限、默认管理员、AI 场景配置、AI 工具风险策略。
// 所有操作都是幂等的 —— 已存在的记录会被跳过。
func RunSeedData(db *gorm.DB) error {
	if err := seedRoles(db); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}
	if err := seedPermissions(db); err != nil {
		return fmt.Errorf("seed permissions: %w", err)
	}
	if err := seedRolePermissions(db); err != nil {
		return fmt.Errorf("seed role permissions: %w", err)
	}
	if err := seedAdminUser(db); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}
	if err := seedAISceneConfigs(db); err != nil {
		return fmt.Errorf("seed ai scene configs: %w", err)
	}
	if err := seedAIScenePrompts(db); err != nil {
		return fmt.Errorf("seed ai scene prompts: %w", err)
	}
	if err := seedAIToolRiskPolicies(db); err != nil {
		return fmt.Errorf("seed ai tool risk policies: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

func seedRoles(db *gorm.DB) error {
	roles := []usermodel.Role{
		{Code: "admin", Name: "管理员", Description: "系统管理员，拥有所有权限", Status: 1},
		{Code: "operator", Name: "运维工程师", Description: "运维工程师，拥有操作权限但无系统管理权限", Status: 1},
		{Code: "viewer", Name: "观察者", Description: "只读用户，仅可查看资源", Status: 1},
	}
	for _, r := range roles {
		var count int64
		if err := db.Model(&usermodel.Role{}).Where("code = ?", r.Code).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Create(&r).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Permissions
// ---------------------------------------------------------------------------

// permEntry 统一定义权限条目。
type permEntry struct {
	code        string
	name        string
	resType     int8   // 0=菜单 1=操作 2=数据
	resource    string
	action      string
	description string
}

func seedPermissions(db *gorm.DB) error {
	perms := buildPermissionList()
	for _, p := range perms {
		var count int64
		if err := db.Model(&usermodel.Permission{}).Where("code = ?", p.code).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			rec := usermodel.Permission{
				Code:        p.code,
				Name:        p.name,
				Type:        p.resType,
				Resource:    p.resource,
				Action:      p.action,
				Description: p.description,
				Status:      1,
			}
			if err := db.Create(&rec).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func buildPermissionList() []permEntry {
	var perms []permEntry
	modules := []struct {
		resource    string
		label       string
		actions     []string // 操作型权限
		hasMenu     bool     // 是否需要菜单权限
		dataActions []string // 数据权限
	}{
		{"host", "主机", []string{"create", "update", "delete", "exec", "import"}, true, []string{"list", "detail"}},
		{"cluster", "集群", []string{"create", "update", "delete", "sync"}, true, []string{"list", "detail"}},
		{"deployment", "部署", []string{"create", "update", "delete", "rollback"}, true, []string{"list", "detail"}},
		{"service", "服务", []string{"create", "update", "delete", "deploy"}, true, []string{"list", "detail"}},
		{"project", "项目", []string{"create", "update", "delete"}, true, []string{"list", "detail"}},
		{"monitoring", "监控", []string{"create", "update", "delete"}, true, []string{"list", "detail"}},
		{"alert", "告警", []string{"create", "update", "delete", "silence"}, true, []string{"list", "detail"}},
		{"cicd", "CI/CD", []string{"create", "update", "delete", "trigger"}, true, []string{"list", "detail"}},
		{"job", "任务", []string{"create", "update", "delete", "run"}, true, []string{"list", "detail"}},
		{"automation", "自动化", []string{"create", "update", "delete", "run"}, true, []string{"list", "detail"}},
		{"cmdb", "CMDB", []string{"create", "update", "delete", "sync"}, true, []string{"list", "detail"}},
		{"governance", "治理", []string{"approve", "reject"}, true, []string{"list", "detail"}},
		{"user", "用户", []string{"create", "update", "delete"}, true, []string{"list", "detail"}},
		{"role", "角色", []string{"create", "update", "delete"}, true, []string{"list", "detail"}},
		{"ai", "AI 助手", []string{"chat", "exec"}, true, []string{"list", "detail"}},
		{"notification", "通知", []string{"create", "update", "delete"}, true, []string{"list", "detail"}},
		{"dashboard", "仪表盘", []string{}, true, []string{"view"}},
		{"plugin", "插件", []string{"create", "update", "delete", "install"}, true, []string{"list", "detail"}},
	}

	for _, m := range modules {
		if m.hasMenu {
			perms = append(perms, permEntry{
				code: m.resource + ":menu", name: m.label + "菜单",
				resType: 0, resource: m.resource, action: "menu",
				description: "访问" + m.label + "模块菜单",
			})
		}
		for _, a := range m.actions {
			perms = append(perms, permEntry{
				code: m.resource + ":" + a, name: m.label + ":" + actionLabel(a),
				resType: 1, resource: m.resource, action: a,
				description: actionLabel(a) + m.label,
			})
		}
		for _, a := range m.dataActions {
			perms = append(perms, permEntry{
				code: m.resource + ":" + a, name: m.label + ":" + actionLabel(a),
				resType: 2, resource: m.resource, action: a,
				description: actionLabel(a) + m.label + "数据",
			})
		}
	}
	return perms
}

func actionLabel(action string) string {
	labels := map[string]string{
		"menu": "菜单", "list": "列表", "detail": "详情",
		"create": "新建", "update": "编辑", "delete": "删除",
		"exec": "执行", "import": "导入", "sync": "同步",
		"deploy": "部署", "rollback": "回滚", "trigger": "触发",
		"run": "运行", "silence": "静默", "approve": "审批",
		"reject": "拒绝", "install": "安装", "view": "查看",
		"chat": "对话",
	}
	if v, ok := labels[action]; ok {
		return v
	}
	return action
}

// ---------------------------------------------------------------------------
// Role-Permission bindings
// ---------------------------------------------------------------------------

func seedRolePermissions(db *gorm.DB) error {
	// 获取角色 ID
	roleMap := make(map[string]int64)
	var roles []usermodel.Role
	if err := db.Find(&roles).Error; err != nil {
		return err
	}
	for _, r := range roles {
		roleMap[r.Code] = int64(r.ID)
	}

	// 获取权限 ID
	permMap := make(map[string]int64)
	var perms []usermodel.Permission
	if err := db.Find(&perms).Error; err != nil {
		return err
	}
	for _, p := range perms {
		permMap[p.Code] = int64(p.ID)
	}

	// admin: 全部权限
	adminPerms := make([]string, 0, len(permMap))
	for code := range permMap {
		adminPerms = append(adminPerms, code)
	}

	// operator: 除 user/role 管理外的操作权限
	operatorExclude := map[string]bool{
		"user:create": true, "user:delete": true,
		"role:create": true, "role:delete": true, "role:update": true,
	}

	// viewer: 仅菜单和数据权限 (type 0 和 2)
	viewerTypes := map[int8]bool{0: true, 2: true}

	bindings := []struct {
		roleCode string
		codes    []string // 空表示动态计算
		dynamic  string   // "all" | "operator" | "viewer"
	}{
		{"admin", nil, "all"},
		{"operator", nil, "operator"},
		{"viewer", nil, "viewer"},
	}

	for _, b := range bindings {
		roleID, ok := roleMap[b.roleCode]
		if !ok || roleID == 0 {
			continue
		}

		var codes []string
		switch b.dynamic {
		case "all":
			codes = adminPerms
		case "operator":
			for _, p := range perms {
				if !operatorExclude[p.Code] {
					codes = append(codes, p.Code)
				}
			}
		case "viewer":
			for _, p := range perms {
				if viewerTypes[p.Type] {
					codes = append(codes, p.Code)
				}
			}
		default:
			codes = b.codes
		}

		for _, code := range codes {
			permID, ok := permMap[code]
			if !ok || permID == 0 {
				continue
			}
			var count int64
			db.Model(&usermodel.RolePermission{}).
				Where("role_id = ? AND permission_id = ?", roleID, permID).
				Count(&count)
			if count == 0 {
				db.Create(&usermodel.RolePermission{
					RoleID:       roleID,
					PermissionID: permID,
				})
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Default admin user
// ---------------------------------------------------------------------------

func seedAdminUser(db *gorm.DB) error {
	var count int64
	if err := db.Model(&usermodel.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// 默认密码: Admin@2026 (生产部署时应强制修改)
	hashedPwd, err := utils.HashPassword("Admin@2026")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	admin := &usermodel.User{
		Username:     "admin",
		PasswordHash: hashedPwd,
		Email:        "admin@opspilot.local",
		Status:       1,
	}
	if err := db.Create(admin).Error; err != nil {
		return err
	}

	// 绑定 admin 角色
	var role usermodel.Role
	if err := db.Where("code = ?", "admin").First(&role).Error; err != nil {
		return err
	}
	return db.Create(&usermodel.UserRole{
		UserID: int64(admin.ID),
		RoleID: int64(role.ID),
	}).Error
}

// ---------------------------------------------------------------------------
// AI Scene Configs
// ---------------------------------------------------------------------------

func seedAISceneConfigs(db *gorm.DB) error {
	sceneToolMap := map[string]struct {
		desc   string
		tools  []string
		blocked []string
	}{
		"ai": {
			desc:   "通用 AI 助手，跨域诊断与问答",
			tools:  []string{"service_get_detail", "service_status", "service_catalog_list", "host_list_inventory", "monitor_alert", "monitor_metric", "k8s_query", "k8s_list_resources"},
			blocked: []string{"host_exec", "service_deploy_apply"},
		},
		"kubernetes": {
			desc:   "Kubernetes 运维与诊断",
			tools:  []string{"k8s_query", "k8s_list_resources", "service_get_detail", "service_status", "host_list_inventory"},
			blocked: []string{"host_exec"},
		},
		"cluster": {
			desc:   "集群清单与部署可视性",
			tools:  []string{"k8s_query", "k8s_list_resources", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status"},
			blocked: []string{"host_exec"},
		},
		"cicd": {
			desc:   "CI/CD 流水线管理与状态查询",
			tools:  []string{"cicd_pipeline_list", "cicd_pipeline_status", "cicd_pipeline_trigger", "job_list", "job_execution_status", "job_run", "deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status"},
			blocked: []string{"host_exec"},
		},
		"monitoring": {
			desc:   "监控告警调查与分析",
			tools:  []string{"monitor_alert_rule_list", "monitor_alert", "monitor_metric"},
			blocked: []string{"host_exec", "service_deploy_apply"},
		},
		"host": {
			tools:   []string{"host_exec", "host_list_inventory"},
			desc:    "主机运维，带审批的命令执行",
			blocked: []string{},
		},
		"deployment": {
			desc:   "部署规划与发布控制",
			tools:  []string{"deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status", "service_deploy_preview", "service_deploy_apply"},
			blocked: []string{"host_exec"},
		},
		"service": {
			desc:   "服务状态与发布上下文",
			tools:  []string{"service_get_detail", "service_status", "service_status_by_target", "service_catalog_list", "service_category_tree", "service_visibility_check", "service_deploy_preview"},
			blocked: []string{"host_exec", "service_deploy_apply"},
		},
		"infrastructure": {
			desc:   "基础设施清单与凭证健康",
			tools:  []string{"credential_list", "credential_test", "host_list_inventory", "cluster_list_inventory"},
			blocked: []string{"host_exec"},
		},
		"governance": {
			desc:   "治理、审计与拓扑审查",
			tools:  []string{"user_list", "role_list", "permission_check", "topology_get", "audit_log_search"},
			blocked: []string{"host_exec", "service_deploy_apply"},
		},
	}

	for scene, cfg := range sceneToolMap {
		var count int64
		if err := db.Model(&aimodel.AISceneConfig{}).Where("scene = ?", scene).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		rec := aimodel.AISceneConfig{
			Scene:            scene,
			Description:      cfg.desc,
			AllowedToolsJSON: marshalJSON(cfg.tools),
			BlockedToolsJSON: marshalJSON(cfg.blocked),
		}
		if err := db.Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// AI Scene Prompts (快捷提示词)
// ---------------------------------------------------------------------------

func seedAIScenePrompts(db *gorm.DB) error {
	prompts := []struct {
		scene   string
		text    string
		order   int
	}{
		// 通用 AI
		{"ai", "最近有哪些告警？帮我分析根因", 1},
		{"ai", "查看集群整体资源使用情况", 2},
		{"ai", "帮我排查服务异常", 3},

		// Kubernetes
		{"kubernetes", "查看 default 命名空间下所有 Pod 状态", 1},
		{"kubernetes", "排查某个 Pod 为什么 CrashLoopBackOff", 2},
		{"kubernetes", "查看节点资源使用率", 3},

		// Cluster
		{"cluster", "列出所有集群及健康状态", 1},
		{"cluster", "查看某个集群的节点详情", 2},
		{"cluster", "检查集群版本是否需要升级", 3},

		// CI/CD
		{"cicd", "查看最近的流水线运行状态", 1},
		{"cicd", "触发某个服务的构建", 2},
		{"cicd", "查看最近失败的部署", 3},

		// Monitoring
		{"monitoring", "当前有哪些活跃告警？", 1},
		{"monitoring", "查看某个服务的 CPU/内存趋势", 2},
		{"monitoring", "分析告警关联关系", 3},

		// Host
		{"host", "查看主机列表及状态", 1},
		{"host", "检查主机磁盘使用情况", 2},
		{"host", "在主机上执行健康检查脚本", 3},

		// Deployment
		{"deployment", "查看部署目标列表", 1},
		{"deployment", "预览某个服务的部署配置", 2},
		{"deployment", "回滚最近一次发布", 3},

		// Service
		{"service", "查看服务目录列表", 1},
		{"service", "查看某个服务的部署状态", 2},
		{"service", "对比不同环境的服务配置", 3},

		// Infrastructure
		{"infrastructure", "查看所有云账号凭证状态", 1},
		{"infrastructure", "测试凭证连接是否正常", 2},
		{"infrastructure", "查看基础设施清单", 3},

		// Governance
		{"governance", "查看最近的操作审计日志", 1},
		{"governance", "查看待审批的操作", 2},
		{"governance", "检查权限配置是否合理", 3},
	}

	for _, p := range prompts {
		var count int64
		db.Model(&aimodel.AIScenePrompt{}).
			Where("scene = ? AND prompt_text = ?", p.scene, p.text).
			Count(&count)
		if count > 0 {
			continue
		}
		rec := aimodel.AIScenePrompt{
			Scene:        p.scene,
			PromptText:   p.text,
			DisplayOrder: p.order,
			IsActive:     true,
		}
		if err := db.Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// AI Tool Risk Policies
// ---------------------------------------------------------------------------

func seedAIToolRiskPolicies(db *gorm.DB) error {
	type policyEntry struct {
		toolName    string
		scene       *string
		riskLevel   string
		approvalReq bool
		priority    int
	}

	policies := []policyEntry{
		// 只读查询 — 低风险，无需审批
		{"host_list_inventory", nil, "low", false, 10},
		{"cluster_list_inventory", nil, "low", false, 10},
		{"service_get_detail", nil, "low", false, 10},
		{"service_status", nil, "low", false, 10},
		{"service_catalog_list", nil, "low", false, 10},
		{"service_list_inventory", nil, "low", false, 10},
		{"k8s_query", nil, "low", false, 10},
		{"k8s_list_resources", nil, "low", false, 10},
		{"deployment_target_list", nil, "low", false, 10},
		{"deployment_target_detail", nil, "low", false, 10},
		{"deployment_bootstrap_status", nil, "low", false, 10},
		{"monitor_alert_rule_list", nil, "low", false, 10},
		{"monitor_alert", nil, "low", false, 10},
		{"monitor_metric", nil, "low", false, 10},
		{"credential_list", nil, "low", false, 10},
		{"user_list", nil, "low", false, 10},
		{"role_list", nil, "low", false, 10},
		{"audit_log_search", nil, "low", false, 10},
		{"topology_get", nil, "low", false, 10},
		{"permission_check", nil, "low", false, 10},
		{"job_list", nil, "low", false, 10},
		{"cicd_pipeline_list", nil, "low", false, 10},
		{"cicd_pipeline_status", nil, "low", false, 10},
		{"job_execution_status", nil, "low", false, 10},
		{"service_status_by_target", nil, "low", false, 10},
		{"service_category_tree", nil, "low", false, 10},
		{"service_visibility_check", nil, "low", false, 10},

		// 预览类 — 低风险
		{"service_deploy_preview", nil, "low", false, 20},
		{"credential_test", nil, "low", false, 20},

		// 执行类 — 中风险，需要审批
		{"host_exec", nil, "high", true, 50},
		{"service_deploy_apply", nil, "high", true, 50},
		{"cicd_pipeline_trigger", nil, "medium", true, 40},
		{"job_run", nil, "medium", true, 40},
	}

	for _, p := range policies {
		var count int64
		q := db.Model(&aimodel.AIToolRiskPolicy{}).Where("tool_name = ?", p.toolName)
		if p.scene != nil {
			q = q.Where("scene = ?", *p.scene)
		} else {
			q = q.Where("scene IS NULL")
		}
		q.Count(&count)
		if count > 0 {
			continue
		}
		rec := aimodel.AIToolRiskPolicy{
			ToolName:         p.toolName,
			Scene:            p.scene,
			RiskLevel:        p.riskLevel,
			ApprovalRequired: p.approvalReq,
			Priority:         p.priority,
			Enabled:          true,
			PolicyVersion:    "v1",
		}
		if err := db.Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func marshalJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
