# AI Assistant Enhancement

## Quick Links

- [Proposal](./proposal.md) - 变更提案
- [Design](./design.md) - 技术设计
- [Tasks](./tasks.md) - 实现任务
- [Spec](./spec.md) - 功能规格

## Summary

增强 AI Copilot 助手的用户体验，包括：

1. **Think 组件替换** - 使用 `@ant-design/x` 的 Think 组件展示思考过程
2. **历史对话持久化** - 页面刷新后自动恢复上次对话
3. **工具调用详情** - 展示工具调用的参数和结果
4. **下一步提示** - 对话结束后显示 AI 生成的推荐
5. **场景感知快捷指令** - 根据场景动态显示快捷提示词
6. **复制功能修复** - 修复消息操作栏复制按钮无效的问题
7. **重新生成功能** - 允许用户重新生成不满意的 AI 回复

## Status

| Phase | Tasks | Status |
|-------|-------|--------|
| Phase 1 | Think组件、历史恢复、复制修复 | 🔴 Not Started |
| Phase 2 | 工具调用详情、重新生成 | 🔴 Not Started |
| Phase 3 | 下一步提示 | 🔴 Not Started |
| Phase 4 | 场景感知快捷指令 | 🔴 Not Started |
| Phase 5 | 消息操作组件 | 🔴 Not Started |

## Files Changed

### Frontend
- `web/src/components/AI/Copilot.tsx` (修改)
- `web/src/components/AI/types.ts` (修改)
- `web/src/components/AI/components/ToolCard.tsx` (修改)
- `web/src/components/AI/components/RecommendationCard.tsx` (新增)
- `web/src/components/AI/components/MessageActions.tsx` (新增)
- `web/src/components/AI/hooks/useConversationRestore.ts` (新增)
- `web/src/components/AI/hooks/useScenePrompts.ts` (新增)
- `web/src/api/modules/ai.ts` (修改)

### Backend
- `internal/model/ai_scene_prompt.go` (新增)
- `internal/service/ai/handler/scene_prompt_handler.go` (新增)
- `internal/service/ai/routes.go` (修改)
- `configs/scene_mappings.yaml` (修改)

### Database
- `ai_scene_prompts` 表 (新增)

## Quick Start

```bash
# 1. Run database migration
mysql -u root -p k8s_manage < migrations/add_ai_scene_prompts.sql

# 2. Start backend
go run cmd/api/main.go

# 3. Start frontend
cd web && npm run dev
```
