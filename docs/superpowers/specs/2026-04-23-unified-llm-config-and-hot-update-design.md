# 设计文档：统一 LLM 配置与热更新支持

- **日期:** 2026-04-23
- **状态:** 待评审
- **主题:** 将 LLM 配置从 `config.yaml` 迁移至数据库，支持 API Key 加密存储、热更新及多供应商扩展。

## 1. 背景与目标

当前系统的 LLM 配置散落在 `config.yaml` 和数据库中，缺乏统一的管理机制，且 API Key 以明文存储在配置文件中，存在安全风险。此外，修改配置后需要重启系统，不支持热更新。

**目标:**
1. **统一配置源:** 数据库作为 LLM 配置的唯一事实来源。
2. **自动迁移:** 系统首次启动时自动将 `config.yaml` 中的配置迁移至数据库。
3. **安全存储:** 所有 API Key 必须加密存储。
4. **热更新:** 后台修改默认模型后，后续请求立即生效。
5. **扩展供应商:** 增加 DeepSeek、Claude、Gemini、Azure OpenAI 等供应商支持。

## 2. 详细设计

### 2.1 引导迁移逻辑 (Bootstrap Migration)

在系统启动阶段（`internal/svc/app_context.go` 初始化过程中），增加 LLM 迁移检查：

- **检查条件:** `ai_llm_providers` 表为空且 `config.yaml` 中 `llm.enable` 为 `true`。
- **执行动作:**
  - 从 `config.yaml` 读取供应商、BaseURL、APIKey、Model、Temperature 等。
  - 使用 `Security.EncryptionKey` 调用 `utils.EncryptText` 加密 API Key。
  - 在数据库中创建记录，并设置 `is_default = true`, `is_enabled = true`。
- **后置动作:** 迁移完成后，系统运行时将完全忽略 `config.yaml` 中的 `llm` 部分。

### 2.2 安全性设计 (Security)

- **加密算法:** 使用现有的 AES-GCM 工具 (`internal/core/utils/secret.go`)。
- **密钥管理:** 使用配置文件中的 `security.encryption_key`。
- **处理流程:**
  - **写入:** 在 `llmprovider` 的业务逻辑层 (Logic/DAO) 拦截，对 API Key 进行加密。
  - **读取:** 在 `llmprovider/client` 初始化驱动前，调用 `DecryptText` 解密。

### 2.3 热更新与运行时查找 (Hot Update)

重构 `internal/modules/llmprovider/client/model.go` 中的 `GetDefaultChatModel`:

1. **DB 优先:** 始终先查询数据库中 `is_default = true` 的启用记录。
2. **零缓存方案:** 考虑到配置修改频率极低，初期采用直接查询数据库的方式实现热更新。后续若有性能需求，可引入基于时间戳的内存缓存。
3. **移除 YAML 回退:** 迁移完成后，移除对 `config.CFG.LLM` 的回退逻辑，确保配置的一致性。

### 2.4 供应商扩展 (Provider Expansion)

在 `internal/modules/llmprovider/client` 中新增以下驱动：

- **DeepSeek:** 适配其 OpenAI 兼容接口。
- **Claude:** 引入 `claude` 官方或 Eino 适配器。
- **Gemini:** 引入 Google AI SDK 支持。
- **Azure OpenAI:** 增加对部署名 (Deployment Name) 和 API 版本号的支持。

## 3. 数据变更

`AILLMProvider` 模型保持不变，但需确保迁移脚本正确处理 `api_key` 字段。

## 4. 验证计划

1. **迁移验证:** 修改 `config.yaml` 后启动系统，检查数据库是否生成加密后的记录。
2. **加密验证:** 直接查看数据库，确认 `api_key` 为加密后的 Base64 字符串。
3. **热更新验证:** 在后台修改默认供应商，发起新的对话请求，观察系统是否立即切换至新供应商。
4. **多供应商验证:** 分别测试 DeepSeek、Claude 等新供应商的连通性。
