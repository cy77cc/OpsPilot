## Why

当前归档变更“service-management-interaction-optimize”在代码层仍存在关键缺口：创建服务时 `team_id` 未从上下文获取而被写死，导致后续部署目标解析经常失败并触发 `deploy target not configured`。同时，服务列表与创建页仍有若干行为与既有 spec 不一致，且手动部署与 CI/CD 的目标解析规则未形成一致闭环。

## What Changes

- 修复服务创建上下文字段回填规则：`team_id` 和相关作用域信息必须来自真实上下文，而不是硬编码默认值。
- 统一部署目标解析链路：手动部署与 CI/CD 都遵循一致的“显式指定 -> 服务默认目标 -> 作用域内可用目标回退”规则，并输出可定位的失败原因。
- 补齐服务管理交互规格落地差异：
  - 列表视图行操作与 spec 对齐（含停止动作）
  - 列表能力满足可排序要求
  - 创建页中英文混用项完成中文统一
- 明确“未配置部署目标”场景的可恢复路径与提示语，降低误报和排障成本。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `service-configuration-management`: 调整服务创建上下文字段来源与服务列表交互行为，确保与既有要求一致。
- `deployment-release-management`: 修改部署目标解析与回退策略，覆盖手动部署链路并增强错误可诊断性。
- `service-ci-management`: 对齐 CI/CD 触发部署时的目标选择逻辑，与手动部署使用同一解析契约。

## Impact

- Backend:
  - `internal/service/service/*`（部署目标解析、发布/部署入口、错误返回）
  - 可能涉及 `api/service/v1/*` 的请求/响应约束说明更新
- Frontend:
  - `web/src/pages/Services/ServiceProvisionPage.tsx`
  - `web/src/pages/Services/ServiceListPage.tsx`
- OpenSpec:
  - 新增本 change 的 delta specs（上述 3 个 modified capabilities）
  - 后续需补充设计与任务拆解，确保手动部署与 CI/CD 联动一致验证
