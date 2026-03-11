// Package client 提供 SSH 客户端实现。
//
// 本文件实现 SFTP 文件传输功能，基于 SSH 连接。
package client

import (
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// NewSFTPClient 创建 SFTP 客户端。
//
// 基于已有的 SSH 连接创建 SFTP 会话。
func NewSFTPClient(sshClient *ssh.Client) (*sftp.Client, error) {
	return sftp.NewClient(sshClient)
}

// UploadFile 上传本地文件到远程服务器。
//
// 参数:
//   - sftpClient: SFTP 客户端
//   - local: 本地文件路径
//   - remote: 远程文件路径
func UploadFile(sftpClient *sftp.Client, local, remote string) error {
	src, err := os.Open(local)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := sftpClient.Create(remote)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = dst.ReadFrom(src)
	return err
}