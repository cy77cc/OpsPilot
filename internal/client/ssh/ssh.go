// Package client 提供 SSH 客户端实现。
//
// 本文件实现 SSH 连接和命令执行功能，
// 支持密码和私钥两种认证方式。
package client

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// NewSSHClient 创建 SSH 客户端连接。
//
// 参数:
//   - user: 用户名
//   - password: 密码（可选）
//   - host: 主机地址
//   - port: SSH 端口
//   - privateKey: 私钥内容（可选）
//   - passphrase: 私钥密码（可选）
//
// 支持密码和私钥两种认证方式，优先使用私钥认证。
func NewSSHClient(user, password, host string, port int, privateKey, passphrase string) (*ssh.Client, error) {
	authMethods, err := buildAuthMethods(password, privateKey, passphrase)
	if err != nil {
		return nil, err
	}
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Auth:            authMethods,
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, config)
}

// buildAuthMethods 构建认证方法列表。
//
// 认证优先级：
//  1. 私钥认证（如果提供且有效）
//  2. 密码认证（如果提供）
//
// 参数:
//   - password: 密码
//   - privateKey: 私钥内容
//   - passphrase: 私钥密码
//
// 返回: 认证方法列表和错误。
func buildAuthMethods(password, privateKey, passphrase string) ([]ssh.AuthMethod, error) {
	authMethods := make([]ssh.AuthMethod, 0, 2)
	var keyParseErr error
	trimmedKey := strings.TrimSpace(privateKey)
	if trimmedKey != "" {
		var (
			signer ssh.Signer
			err    error
		)
		if strings.TrimSpace(passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(trimmedKey), []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(trimmedKey))
		}
		if err != nil {
			keyParseErr = err
		} else {
			// Prefer key auth first when both are configured.
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	if strings.TrimSpace(password) != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}
	if len(authMethods) == 0 {
		if keyParseErr != nil {
			return nil, keyParseErr
		}
		return nil, fmt.Errorf("no ssh auth method provided")
	}
	return authMethods, nil
}

// RunCommand 在 SSH 会话中执行命令。
//
// 返回命令的标准输出和标准错误合并结果。
func RunCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}

	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	return strings.TrimSpace(string(output)), err
}
