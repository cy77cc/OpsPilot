# OpsPilot Agent 文档

本目录包含项目可用的 Agent 说明文档。

## 目录

### 规划与架构

| Agent | 文件 | 用途 |
|-------|------|------|
| Planner | [planner.md](./planner.md) | 实现计划、任务分解 |
| Architect | [architect.md](./architect.md) | 架构设计、技术选型 |

### 开发流程

| Agent | 文件 | 用途 |
|-------|------|------|
| TDD Guide | [tdd-guide.md](./tdd-guide.md) | 测试驱动开发 |
| Code Reviewer | [code-reviewer.md](./code-reviewer.md) | 代码审查 |
| Security Reviewer | [security-reviewer.md](./security-reviewer.md) | 安全审查 |

### 问题解决

| Agent | 文件 | 用途 |
|-------|------|------|
| Build Error Resolver | [build-error-resolver.md](./build-error-resolver.md) | 构建错误修复 |
| Refactor Cleaner | [refactor-cleaner.md](./refactor-cleaner.md) | 代码清理、重构 |

### 测试

| Agent | 文件 | 用途 |
|-------|------|------|
| E2E Runner | [e2e-runner.md](./e2e-runner.md) | 端到端测试 |

### 文档

| Agent | 文件 | 用途 |
|-------|------|------|
| Doc Updater | [doc-updater.md](./doc-updater.md) | 文档更新、代码映射 |

### 语言专项

| Agent | 文件 | 用途 |
|-------|------|------|
| Go Reviewer | [go-reviewer.md](./go-reviewer.md) | Go 代码审查 |
| Database Reviewer | [database-reviewer.md](./database-reviewer.md) | 数据库审查 |

## 使用方式

### 通过 Agent 工具

```bash
Agent(subagent_type="<agent-name>", prompt="<任务描述>")
```

### 通过 Team 分配

```bash
TaskUpdate(taskId="1", owner="<agent-name>")
```

## 触发规则

根据 `~/.claude/rules/agents.md` 配置：

| 场景 | 自动触发的 Agent |
|------|------------------|
| 复杂功能请求 | planner |
| 代码刚编写/修改 | code-reviewer |
| Bug 修复或新功能 | tdd-guide |
| 架构决策 | architect |

## Agent 权限级别

```
┌─────────────────────────────────────────────────────┐
│              Agent Permission Levels                 │
├─────────────────────────────────────────────────────┤
│                                                      │
│  Level 1: Read Only (只读)                          │
│  • planner, architect, code-reviewer, security-     │
│    reviewer, go-reviewer, python-reviewer,          │
│    database-reviewer                                │
│                                                      │
│  Level 2: Read + Write (读写)                       │
│  • tdd-guide, e2e-runner, doc-updater,              │
│    refactor-cleaner                                 │
│                                                      │
│  Level 3: Full Access (完全访问)                     │
│  • build-error-resolver (需要运行命令)              │
│                                                      │
└─────────────────────────────────────────────────────┘
```

## 并行执行

当多个独立任务可以并行执行时，使用 Task 工具启动多个 Agent：

```bash
# 并行执行安全和代码审查
Agent(subagent_type="security-reviewer", prompt="审查 auth 模块")
Agent(subagent_type="code-reviewer", prompt="审查 service 模块")
```

## 自定义 Agent

如需创建项目特定的 Agent，在本目录添加 `<name>.md` 文件，包含：

1. 触发时机
2. 能力范围
3. 工具权限
4. 输出格式
5. 约束条件
