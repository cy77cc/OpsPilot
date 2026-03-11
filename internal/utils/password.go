// Package utils 提供通用工具函数。
//
// 本文件实现基于 bcrypt 的密码哈希和验证功能。
// 支持向后兼容旧版 scrypt 哈希。
package utils

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 对密码进行哈希。
//
// 使用 DefaultCost（10）作为哈希代价。
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword 验证明文密码是否与存储的哈希匹配。
//
// 支持两种哈希格式：
//   - bcrypt: 以 $2a$、$2b$、$2y$ 开头
//   - scrypt: 其他格式（向后兼容）
func VerifyPassword(plaintext, stored string) bool {
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plaintext)) == nil
	}
	return PasswordVerify(plaintext, stored)
}
