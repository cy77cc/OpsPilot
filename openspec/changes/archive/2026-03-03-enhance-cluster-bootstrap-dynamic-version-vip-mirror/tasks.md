## 1. API 与数据模型扩展

- [x] 1.1 扩展 bootstrap 请求/响应契约，新增 `k8s.version_channel`、`packages.repo_mode/repo_url`、`images.image_repository`、`endpoint.mode/control_plane_endpoint`、`vip.provider`、`etcd.mode/external` 字段并保持旧参数兼容
- [x] 1.2 扩展 bootstrap task 持久化模型，记录解析后的有效参数与关键预检诊断信息
- [x] 1.3 为 bootstrap profile 新增数据结构与 CRUD API（含 RBAC 校验）
- [x] 1.4 新增版本目录 API，返回官方通道版本与平台支持状态（supported/preview/blocked）

## 2. 参数解析与校验

- [x] 2.1 实现 bootstrap 参数解析优先级：请求覆盖 > profile > 默认值
- [x] 2.2 实现跨字段校验规则（mirror 必填项、vip 模式必填 endpoint、external etcd 证书与 endpoint 必填）
- [x] 2.3 实现 blocked 版本拦截与候选版本建议
- [x] 2.4 统一错误输出结构，返回字段级错误码、失败域分类与修复建议

## 3. kubeadm 配置驱动与执行编排

- [x] 3.1 引入 kubeadm config 生成器，支持 `controlPlaneEndpoint`、`imageRepository`、`etcd.external` 等配置生成
- [x] 3.2 改造 bootstrap 执行链路为配置驱动 init，替换当前写死 `v1.28` 路径依赖
- [x] 3.3 增加版本支持矩阵校验，确保可选版本与可执行脚本/物料一致
- [x] 3.4 保留旧执行路径开关，支持灰度与快速回滚

## 4. 弱离线（mirror）能力

- [x] 4.1 改造安装脚本支持 `online|mirror` 模式切换与内网 repo 地址注入
- [x] 4.2 注入 `imageRepository` 到 kubeadm 配置与镜像拉取流程
- [x] 4.3 增加 mirror 预检（仓库可达、关键包可解析、镜像仓库可达）
- [x] 4.4 对 mirror 场景失败输出结构化诊断（repo/registry 分类）

## 5. VIP 与 external etcd 支持

- [x] 5.1 实现 VIP provider 抽象层（provider 接口与参数模型）
- [x] 5.2 实现 kube-vip 自动化路径（首批默认 provider）
- [x] 5.3 实现 keepalived provider 路径（可选启用）
- [x] 5.4 增加 endpoint 可用性校验，确保通过 `control_plane_endpoint` 可访问 apiserver
- [x] 5.5 实现 external etcd 连接预检与 TLS 失败分类诊断

## 6. 前端向导与交互

- [x] 6.1 扩展 `ClusterBootstrapWizard` 高级配置区，新增版本通道、mirror、image repository、endpoint 模式、VIP provider、etcd 模式输入
- [x] 6.2 对接版本目录 API，替换写死版本下拉并展示支持状态标签
- [x] 6.3 实现前端联动校验与 warning 提示（非阻断）
- [x] 6.4 在预览与执行进度页展示有效参数摘要与关键诊断信息

## 7. 测试与发布保障

- [x] 7.1 新增后端单元测试：参数解析优先级、跨字段校验、版本拦截、错误分类
- [x] 7.2 新增脚本/执行集成测试：mirror + kube-vip 主路径、external etcd 参数化路径
- [x] 7.3 新增前端测试：高级配置联动、版本目录渲染、校验与提示行为
- [x] 7.4 形成灰度发布与回滚 runbook，先内网环境验证后逐步放量
- [x] 7.5 执行 `openspec validate --json` 并修复本变更所有 artifact 校验问题
