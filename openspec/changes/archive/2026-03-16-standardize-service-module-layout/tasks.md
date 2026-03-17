## 1. Spec And Baseline Alignment

- [x] 1.1 对照 `internal/service/user` 梳理统一后的服务模块基线骨架，确认各模块需要保留的例外文件类型
- [x] 1.2 盘点 `internal/service/` 下需要纳入本次整理的模块，标注平铺模块与复杂模块两类迁移范围

## 2. Reorganize Flat Service Modules

- [x] 2.1 将 `automation`、`cmdb`、`dashboard`、`jobs`、`monitoring`、`topology` 的 handler 实现迁移到各自 `handler/` 目录
- [x] 2.2 将 `automation`、`cmdb`、`dashboard`、`jobs`、`monitoring`、`topology` 的 logic 实现迁移到各自 `logic/` 目录，并保持 `routes.go` 为唯一路由入口
- [x] 2.3 清理上述模块根目录中已被迁移的平铺 `handler.go`、`logic.go` 历史残留，保留必要的基础设施文件并记录保留原因

## 3. Normalize Partially Structured Modules

- [x] 3.1 归整 `ai` 与 `cicd` 模块，使其入口、handler、logic、repo 等目录边界与统一规范保持一致
- [ ] 3.2 归整 `cluster`、`deployment`、`service` 等复杂模块的根目录文件，只保留明确的跨职责基础设施文件
- [ ] 3.3 检查 `project`、`host`、`node`、`user` 等已接近规范的模块，补齐命名或目录边界上的不一致点

## 4. Verification And Follow-Through

- [x] 4.1 逐模块运行编译或测试验证，确保目录迁移不改变导出入口、路由注册与鉴权行为
- [ ] 4.2 运行全量 OpenSpec 与代码验证，确认 `standardize-service-module-layout` 变更和实现状态一致
