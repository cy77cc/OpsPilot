# Contributing

## OpenSpec-First Rule

For any PR that introduces or changes product/platform capability behavior, you MUST update OpenSpec in the same PR.

Required actions:

1. Create or update a change under `openspec/changes/<change-name>/`.
2. Keep artifacts aligned (`proposal.md`, `design.md` when needed, `specs/*/spec.md`, `tasks.md`).
3. Mark completed tasks in `tasks.md` (`- [x]`) and keep pending tasks as `- [ ]`.
4. Run validation before merge:
   - `openspec validate --json`

Allowed exception:

- Pure refactor/chore/docs formatting with no behavior/capability impact may skip OpenSpec updates, but PR must explicitly state the reason.

## Git Safety Operations

### 1. 禁用高危 Git 操作

禁止在未获得明确授权的情况下执行以下具有破坏性的 Git 命令：

- `git reset --hard` (严禁回滚丢失未提交代码)
- `git clean -f/fd` (禁止清理未追踪文件)
- `git restore .` (禁止批量覆盖工作区)
- `rm -rf` (禁止递归删除目录)

### 2. "动刀"前的快照强制要求

在执行任何涉及文件删除、大规模重构或 Git 状态回滚的操作前，必须：

- 询问用户是否已对当前更改进行 `git add` 或 `git commit`
- 主动建议用户执行 `git stash` 备份当前工作区
- 禁止一次性删除超过 3 个文件而不说明具体原因

### 3. 恢复逻辑校验

当要求"恢复文件"时，禁止盲目执行 `git restore`。请先：

- 列出待恢复的文件清单
- 询问用户："这些是你之前主动删除的，确定要找回吗？"

### 4. 报错优先原则

如果遇到 Git 冲突或权限报错，禁止尝试通过"强行重置"来解决。请：

- 停止操作并输出当前的 `git status`
- 由用户决定处理方式
