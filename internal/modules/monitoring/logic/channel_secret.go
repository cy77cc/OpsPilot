package logic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/utils"
)

func encryptChannelConfig(plain, key string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("security.encryption_key is required")
	}
	return utils.EncryptText(plain, key)
}

func decryptAndMaskChannelConfig(cipherText, key string) (string, error) {
	if strings.TrimSpace(cipherText) == "" {
		return "{}", nil
	}
	plain, err := utils.DecryptText(cipherText, key)
	if err != nil {
		return "", err
	}
	return maskJSONSecrets(plain), nil
}

func maskJSONSecrets(raw string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return `{"masked":"***"}`
	}
	for k, v := range payload {
		lower := strings.ToLower(strings.TrimSpace(k))
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "webhook") {
			payload[k] = "***"
			continue
		}
		if s, ok := v.(string); ok && strings.Contains(s, "@") {
			parts := strings.SplitN(s, "@", 2)
			payload[k] = "***@" + parts[1]
		}
	}
	b, _ := json.Marshal(payload)
	return string(b)
}
