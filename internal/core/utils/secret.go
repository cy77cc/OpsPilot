// Package utils 提供通用工具函数。
//
// 本文件实现 AES-GCM 加密和解密功能，用于敏感数据的加密存储。
// 如 SSH 私钥、API 密钥等。
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptText 使用 AES-GCM 加密文本。
//
// 使用 SHA256 对密钥进行规范化，生成随机 nonce，
// 返回 Base64 编码的密文。
func EncryptText(plainText string, key string) (string, error) {
	if key == "" {
		return "", errors.New("encryption key is empty")
	}
	block, err := aes.NewCipher(normalizeKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	cipherData := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherData), nil
}

// DecryptText 使用 AES-GCM 解密文本。
//
// 参数:
//   - cipherTextB64: Base64 编码的密文
//   - key: 加密密钥
//
// 返回解密后的明文。
func DecryptText(cipherTextB64 string, key string) (string, error) {
	if key == "" {
		return "", errors.New("encryption key is empty")
	}
	cipherData, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(normalizeKey(key))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(cipherData) < nonceSize {
		return "", errors.New("cipher data too short")
	}
	nonce, payload := cipherData[:nonceSize], cipherData[nonceSize:]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// normalizeKey 使用 SHA256 将密钥规范化为 32 字节。
func normalizeKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// MaskAccessKey 对 AccessKey ID 进行掩码处理。
//
// 掩码规则:
//   - 长度 <= 8: 显示前2位 + **** + 后2位 (如 "ABCDEF12" -> "AB****12")
//   - 长度 > 8: 显示前4位 + **** + 后4位 (如 "LTAI4xK7mNpQ3wXy" -> "LTAI****3wXy")
//
// 参数:
//   - ak: 原始 AccessKey ID
//
// 返回掩码后的字符串。
func MaskAccessKey(ak string) string {
	if ak == "" {
		return ""
	}
	n := len(ak)
	if n <= 8 {
		if n <= 4 {
			return ak[:2] + "****"
		}
		return ak[:2] + "****" + ak[n-2:]
	}
	return ak[:4] + "****" + ak[n-4:]
}
