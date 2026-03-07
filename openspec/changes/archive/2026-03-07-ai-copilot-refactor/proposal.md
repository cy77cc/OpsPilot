# AI Copilot 统一入口与后端重构

## 概述

将前端 AI 助手统一为单一入口，优化用户体验；清理后端冗余代码，重组目录结构。

## 背景

### 当前问题

**前端：**
- 存在两个独立的 AI 助手按钮（全局助手 + 场景助手）
- 用户需要判断使用哪个入口，心智负担高
- 未充分利用 @ant-design/x-sdk 组件能力

**后端：**
- AI 模块经过多次重构，存在冗余代码
- `approval_notifier.go` 和 `permission_checker.go` 功能将移到前端实现
- 空目录 `handler/` 和 `logic/` 是历史遗留
- 目录结构扁平化，缺乏清晰的职责划分

## 目标

1. **前端统一入口**：单一 "✨ AI Copilot" 按钮，自动感知场景，支持切换
2. **后端代码清理**：删除冗余文件，重组目录结构
3. **技术升级**：使用 @ant-design/x-sdk 优化前端实现

## 范围

### 包含

**前端：**
- 合并 `AIAssistantButton.tsx` 中的两个按钮
- 实现场景自动感知与下拉切换
- 集成 @ant-design/x-sdk 组件：useXChat, useXConversations, Bubble, Sender, Conversations
- 实现自定义 ChatProvider 适配现有后端 SSE API

**后端：**
- 删除 `approval_notifier.go`, `permission_checker.go` 及其测试
- 删除空目录 `handler/`, `logic/`
- 合并 9 个工具分类文件到 `category.go`
- 重组 `internal/ai/` 目录结构
- 重组 `internal/service/ai/` 目录结构
- 修改 `svc.go` 使用 HybridAgent 作为主入口

### 不包含

- 后端 API 接口变更
- 数据库 schema 变更
- 现有功能逻辑修改

## 成功标准

1. 前端只有一个 "✨ AI Copilot" 入口
2. 抽屉内可自动感知并切换场景
3. 后端测试全部通过
4. 代码覆盖率不降低

## 风险与依赖

### 风险
- @ant-design/x-sdk 与现有 SSE 格式的兼容性需要验证
- 目录重组可能影响 IDE 的 import 路径

### 依赖
- @ant-design/x-sdk v2.3.0 已安装
- 项目已可正常编译

## 时间线

预计 2-3 个工作日完成：
- Day 1: 后端清理与目录重组
- Day 2: 前端组件重构与 x-sdk 集成
- Day 3: 测试与修复
