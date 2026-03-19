-- +migrate Up
-- 添加 runtime_json 字段用于持久化 AI 对话运行时状态
ALTER TABLE ai_chat_messages ADD COLUMN runtime_json LONGTEXT NULL AFTER status;

-- +migrate Down
ALTER TABLE ai_chat_messages DROP COLUMN runtime_json;
