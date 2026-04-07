// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件提供敏感信息脱敏工具，用于审计持久化和调试输出。
package logic

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

const redactedPlaceholder = "***"

var sensitiveKeySuffixes = []string{
	"_password",
	"-password",
	".password",
	"_secret",
	"-secret",
	".secret",
	"_token",
	"-token",
	".token",
	"_key",
	"-key",
	".key",
}

var sensitiveKeyNames = map[string]struct{}{
	"password":      {},
	"passwd":        {},
	"secret":        {},
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"api_key":       {},
	"apikey":        {},
	"client_secret": {},
	"authorization": {},
	"auth":          {},
	"kubeconfig":    {},
	"private_key":   {},
	"tls_key":       {},
	"credential":    {},
	"credentials":   {},
}

// RedactSensitive 对 map/slice/string 等输入递归脱敏。
//
// 对于 JSON 字符串会先尝试解析，再按同样规则递归脱敏。
func RedactSensitive(v any) any {
	return redactValue(v)
}

// RedactAuditPayload 将任意值转换为适合审计存储的脱敏字符串。
func RedactAuditPayload(v any) string {
	redacted := RedactSensitive(v)
	switch val := redacted.(type) {
	case nil:
		return ""
	case string:
		return val
	case []byte:
		return string(val)
	default:
		buf, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprint(val)
		}
		return string(buf)
	}
}

func redactValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case string:
		return redactString(val)
	case []string:
		out := make([]any, 0, len(val))
		for _, item := range val {
			out = append(out, redactValue(item))
		}
		return out
	case []any:
		out := make([]any, 0, len(val))
		for _, item := range val {
			out = append(out, redactValue(item))
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(val))
		for key, item := range val {
			if isSensitiveKey(key) {
				out[key] = redactedPlaceholder
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(val))
		for key, item := range val {
			if isSensitiveKey(key) {
				out[key] = redactedPlaceholder
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}

	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return redactValue(rv.Elem().Interface())
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			keyStr := fmt.Sprint(key.Interface())
			if isSensitiveKey(keyStr) {
				out[keyStr] = redactedPlaceholder
				continue
			}
			out[keyStr] = redactValue(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out = append(out, redactValue(rv.Index(i).Interface()))
		}
		return out
	case reflect.String:
		return redactString(rv.String())
	default:
		return v
	}
}

func redactString(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return s
	}

	if looksLikeJSON(trimmed) {
		var decoded any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			redacted := RedactSensitive(decoded)
			buf, err := json.Marshal(redacted)
			if err == nil {
				return string(buf)
			}
		}
	}

	if looksSensitiveString(trimmed) {
		return redactedPlaceholder
	}

	return s
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if _, ok := sensitiveKeyNames[lower]; ok {
		return true
	}
	for _, suffix := range sensitiveKeySuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}

func looksLikeJSON(s string) bool {
	if len(s) < 2 {
		return false
	}
	first := s[0]
	last := s[len(s)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

func looksSensitiveString(s string) bool {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "bearer ") ||
		strings.Contains(lower, "authorization:") ||
		strings.Contains(lower, "password=") ||
		strings.Contains(lower, "passwd=") ||
		strings.Contains(lower, "secret=") ||
		strings.Contains(lower, "token=") ||
		strings.Contains(lower, "client_secret=") ||
		strings.Contains(lower, "api_key=") ||
		strings.Contains(lower, "private key") ||
		strings.Contains(lower, "begin private key") ||
		strings.Contains(lower, "begin rsa private key") {
		return true
	}
	return false
}

// SanitizeOperationText sanitizes sensitive patterns in operation text.
func SanitizeOperationText(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return text
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "bearer ") {
		return "Bearer ***"
	}
	if strings.Contains(lower, "authorization:") {
		return "authorization: ***"
	}
	if strings.Contains(lower, "password=") ||
		strings.Contains(lower, "passwd=") ||
		strings.Contains(lower, "secret=") ||
		strings.Contains(lower, "token=") ||
		strings.Contains(lower, "api_key=") ||
		strings.Contains(lower, "private_key=") {
		parts := strings.SplitN(text, "=", 2)
		if len(parts) > 0 {
			return parts[0] + "=***"
		}
	}
	return text
}
