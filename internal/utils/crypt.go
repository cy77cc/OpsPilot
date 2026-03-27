// Package utils 提供通用工具函数。
//
// 本文件实现基于 scrypt 的密码加密功能，用于向后兼容旧密码验证。
package utils

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/config"
	"golang.org/x/crypto/scrypt"
)

// EncryptPassword 使用 scrypt 加密密码。
//
// 使用服务器配置的盐值，参数为 N=32768, r=8, p=1, keyLen=32。
// 返回十六进制格式的加密字符串。
func EncryptPassword(pwd string) (string, error) {
	salt := config.CFG.Server.Salt
	dk, err := scrypt.Key([]byte(pwd), []byte(salt), 32768, 8, 1, 32)
	return fmt.Sprintf("%x", string(dk)), err
}

// PasswordVerify 验证密码是否匹配。
//
// 对明文密码进行加密，并与存储的哈希值比较。
func PasswordVerify(password, hashedPassword string) bool {
	bk, err := EncryptPassword(password)
	if err != nil {
		return false
	}
	return bk == hashedPassword
}
