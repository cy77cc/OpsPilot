## Context

当前系统在“服务创建 -> 部署目标解析 -> 发布/部署执行”链路上存在上下文不一致问题：前端创建服务时将 `team_id` 固定写入 `1`，后端在解析默认部署目标时又按 `project_id + team_id + env + target_type` 过滤，导致大量服务无法匹配到可用目标并抛出 `deploy target not configured`。此外，服务管理 UI 与既有规范仍有偏差（列表行操作缺失、列表列不可排序、创建页中英文混用），与“交互优化已归档”状态不一致。

涉及路径：
- 前端：`web/src/pages/Services/ServiceProvisionPage.tsx`、`web/src/pages/Services/ServiceListPage.tsx`
- 后端：`internal/service/service/logic_deploy.go` 及相关发布/CI 调用入口
- 规格：`service-configuration-management`、`deployment-release-management`、`service-ci-management`

## Goals / Non-Goals

**Goals:**
- 让服务创建阶段的作用域字段（尤其 `team_id`）来源于真实上下文，禁止硬编码默认值污染数据。
- 统一手动部署与 CI/CD 的部署目标解析顺序与回退语义，确保两条链路行为一致、错误可诊断。
- 补齐服务管理页面与现有 spec 的行为差异，避免“文档已完成但产品行为不一致”。
- 在不引入破坏性接口变更前提下，提升部署失败问题的可恢复性（可引导用户配置 target）。

**Non-Goals:**
- 不在本变更中新增全新的部署编排能力或多云策略引擎。
- 不重构整个权限系统（仅复用现有 project/team 上下文与 RBAC 检查）。
- 不替换当前 Service/Release 数据模型的大结构（仅做必要字段来源与解析规则修正）。

## Decisions

### Decision 1: 服务作用域字段以上下文为准，前端不再写死 team_id
- 方案：在服务创建请求中使用已登录上下文/当前项目上下文提供的 team 信息；若前端不可可靠获得，后端在 create 逻辑中补齐并校验，拒绝硬编码兜底。
- 原因：部署目标匹配与 project/team 强相关，错误 team 会直接导致 fallback 失效。
- 备选方案：继续前端固定 `team_id=1` 并在部署时“忽略 team 过滤”。
  - 放弃原因：会造成跨团队目标误匹配，属于安全与隔离风险。

### Decision 2: 部署目标解析契约统一为三段式
- 解析顺序：
  1) 请求显式指定（cluster/target/namespace）
  2) 服务默认 deploy target（service_deploy_targets）
  3) 作用域 fallback（project/team/env/target_type 下的 active target）
- 手动部署与 CI/CD 统一调用同一解析逻辑（同一逻辑单元或共享 helper），禁止两套分叉规则。
- 原因：减少“手动能发、CI 失败”或反向情况，降低维护复杂度。
- 备选方案：CI 使用独立“更宽松”回退规则。
  - 放弃原因：行为不可预期，排障成本高，且难以在 spec 层表达一致性。

### Decision 3: 错误语义升级为“可操作”信息
- 方案：保留统一错误主语义（未配置 deploy target），同时附带上下文缺失点（project/team/env/target_type）和建议动作（配置默认目标或创建作用域目标）。
- 原因：当前 `deploy target not configured` 过于抽象，难以快速定位。
- 备选方案：保持当前错误文本不变。
  - 放弃原因：无法满足“联动场景可诊断”目标。

### Decision 4: UI 差异修复按 spec 最小补齐
- 列表行操作补齐“停止”；表格关键列支持排序；创建页剩余英文标签中文化。
- 原因：这些是已存在要求，属于一致性修复而非新增产品定义。
- 备选方案：在新 spec 中放宽现有要求。
  - 放弃原因：会把实现缺口转移为“需求回退”，不符合本次变更目标。

## Risks / Trade-offs

- [Risk] 上下文来源在不同登录态/入口不一致，可能导致创建服务校验失败增多  
  → Mitigation: 在后端统一兜底校验并返回明确缺失字段；前端提交前做轻量校验。

- [Risk] 统一解析后，部分历史“侥幸成功”的部署路径会变成显式失败  
  → Mitigation: 提供兼容期日志与告警，记录旧行为命中比例，必要时提供一次性迁移脚本回填默认目标。

- [Risk] CI 调用链改造触及多个入口，回归范围扩大  
  → Mitigation: 为手动部署与 CI 增加同构用例，至少覆盖“显式指定/默认目标/fallback/失败提示”四类。

- [Risk] 前端中文化与排序改动可能影响既有自动化 UI 测试快照  
  → Mitigation: 同步更新快照与断言文案，避免不必要视觉回归。

## Migration Plan

1. 先改后端：收敛部署目标解析逻辑与错误结构，保证 API 行为稳定且可回滚。  
2. 再改前端：移除 `team_id` 硬编码，补齐列表/创建页 spec 差异。  
3. 数据修复：对历史受影响服务执行检查，识别 team/project 作用域异常记录并回填默认 target（仅对确认可推断记录）。  
4. 验证发布：联调手动部署与 CI/CD 双链路，确认解析行为一致。  
5. 回滚策略：如出现大面积部署阻塞，可回滚到旧解析逻辑，同时保留新增日志，继续离线修复历史数据后再灰度。

## Open Questions

- 当前用户上下文中 team 信息的权威来源是 JWT claims、用户资料接口，还是项目详情接口？需要在实现前固定单一来源。  
- CI 触发接口是否已完整透传 env/target 参数，还是需要补充 API 合约字段？  
- “可操作错误信息”是否需要前端结构化展示（如 action hint），还是先用文本提示过渡？
