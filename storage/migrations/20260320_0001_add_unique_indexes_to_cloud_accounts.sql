-- 添加云账号唯一索引
-- 账号名称在相同云厂商下唯一
-- AccessKey ID 在相同云厂商下唯一

-- 先删除可能存在的旧索引（如果有）
DROP INDEX IF EXISTS idx_provider_account ON host_cloud_accounts;
DROP INDEX IF EXISTS idx_provider_ak ON host_cloud_accounts;

-- 添加唯一索引
CREATE UNIQUE INDEX idx_provider_account ON host_cloud_accounts(provider, account_name);
CREATE UNIQUE INDEX idx_provider_ak ON host_cloud_accounts(provider, access_key_id);
