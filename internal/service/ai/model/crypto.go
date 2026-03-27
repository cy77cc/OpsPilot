// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现 LLM Provider API Key 的加密解密功能。
package model

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/utils"
)

// encryptLLMProviderAPIKey 加密 API Key。
//
// 使用配置的加密密钥对 API Key 进行加密存储。
//
// 参数:
//   - plainText: 明文 API Key
//
// 返回: 加密后的字符串
func encryptLLMProviderAPIKey(plainText string) (string, error) {
	return utils.EncryptText(strings.TrimSpace(plainText), strings.TrimSpace(config.CFG.Security.EncryptionKey))
}

// decryptLLMProviderAPIKey 解密 API Key。
//
// 用于实际调用 LLM 时解密存储的 API Key。
//
// 参数:
//   - cipherText: 加密后的 API Key
//
// 返回: 明文 API Key
func decryptLLMProviderAPIKey(cipherText string) (string, error) {
	return utils.DecryptText(strings.TrimSpace(cipherText), strings.TrimSpace(config.CFG.Security.EncryptionKey))
}

// maskLLMProviderAPIKey 脱敏 API Key。
//
// 用于 API 响应中隐藏敏感信息，只显示前缀和后缀部分。
//
// 参数:
//   - apiKey: 原始 API Key
//
// 返回: 脱敏后的字符串
func maskLLMProviderAPIKey(apiKey string) string {
	return utils.MaskAccessKey(strings.TrimSpace(apiKey))
}
