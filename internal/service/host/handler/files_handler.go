// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"bytes"
	"errors"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	sshclient "github.com/cy77cc/OpsPilot/internal/client/ssh"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
)

// maxInlineReadBytes 内联读取文件的最大字节数（1MB）。
const maxInlineReadBytes = 1024 * 1024 // 1MB

// normalizeRemotePath 规范化远程路径。
//
// 处理空路径、去除首尾空格、清理路径中的 . 和 ..。
//
// 参数:
//   - p: 原始路径
//
// 返回: 规范化后的路径
func normalizeRemotePath(p string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return "."
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "" {
		return "."
	}
	return cleaned
}

// withSFTP 使用 SFTP 客户端执行操作。
//
// 建立 SSH 连接并创建 SFTP 客户端，执行传入的操作函数后自动关闭连接。
//
// 参数:
//   - c: Gin 上下文
//   - hostID: 主机 ID
//   - fn: 操作函数
//
// 返回: 操作错误
func (h *Handler) withSFTP(c *gin.Context, hostID uint64, fn func(*sftp.Client) error) error {
	node, err := h.hostService.Get(c.Request.Context(), hostID)
	if err != nil {
		return errors.New("host not found")
	}
	privateKey, passphrase, err := h.loadNodePrivateKey(c, node)
	if err != nil {
		return err
	}
	password := strings.TrimSpace(node.SSHPassword)
	if strings.TrimSpace(privateKey) != "" {
		password = ""
	}
	cli, err := sshclient.NewSSHClient(node.SSHUser, password, node.IP, node.Port, privateKey, passphrase)
	if err != nil {
		return err
	}
	defer cli.Close()
	sftpClient, err := sshclient.NewSFTPClient(cli)
	if err != nil {
		return err
	}
	defer sftpClient.Close()
	return fn(sftpClient)
}

// ListFiles 列出远程目录文件。
//
// @Summary 列出远程文件
// @Description 列出主机上指定目录的所有文件和子目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param path query string false "目录路径，默认为当前目录"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files [get]
func (h *Handler) ListFiles(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	target := normalizeRemotePath(c.Query("path"))
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		items, err := cli.ReadDir(target)
		if err != nil {
			return err
		}
		list := make([]gin.H, 0, len(items))
		for _, item := range items {
			list = append(list, gin.H{
				"name":       item.Name(),
				"path":       path.Join(target, item.Name()),
				"is_dir":     item.IsDir(),
				"size":       item.Size(),
				"mode":       item.Mode().String(),
				"updated_at": item.ModTime(),
			})
		}
		httpx.OK(c, gin.H{"path": target, "list": list, "total": len(list)})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// ReadFileContent 读取远程文件内容。
//
// @Summary 读取文件内容
// @Description 读取主机上的文件内容，支持最大 1MB 的文本文件预览
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param path query string true "文件路径"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/content [get]
func (h *Handler) ReadFileContent(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	target := normalizeRemotePath(c.Query("path"))
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		f, err := cli.Open(target)
		if err != nil {
			return err
		}
		defer f.Close()
		buf := bytes.NewBuffer(nil)
		if _, err := io.CopyN(buf, f, maxInlineReadBytes+1); err != nil && err != io.EOF {
			return err
		}
		if buf.Len() > maxInlineReadBytes {
			httpx.Fail(c, xcode.ParamError, "file too large for inline preview")
			return nil
		}
		raw := buf.Bytes()
		if !utf8.Valid(raw) {
			httpx.Fail(c, xcode.ParamError, "binary file is not supported for inline edit")
			return nil
		}
		httpx.OK(c, gin.H{"path": target, "content": string(raw)})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// WriteFileContent 写入远程文件内容。
//
// @Summary 写入文件内容
// @Description 创建或覆盖主机上的文件内容
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "文件内容请求 {path: string, content: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/content [put]
func (h *Handler) WriteFileContent(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	target := normalizeRemotePath(req.Path)
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		f, err := cli.Create(target)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := f.Write([]byte(req.Content)); err != nil {
			return err
		}
		httpx.OK(c, gin.H{"path": target, "size": len(req.Content)})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// UploadFile 上传文件到主机。
//
// @Summary 上传文件
// @Description 通过 multipart/form-data 上传文件到主机指定目录
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param path query string false "目标目录路径"
// @Param file formData file true "上传的文件"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/upload [post]
func (h *Handler) UploadFile(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	dirPath := normalizeRemotePath(c.Query("path"))
	file, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "file is required")
		return
	}
	src, err := file.Open()
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	defer src.Close()
	target := path.Join(dirPath, file.Filename)
	err = h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		dst, err := cli.Create(target)
		if err != nil {
			return err
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		httpx.OK(c, gin.H{"path": target})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// DownloadFile 从主机下载文件。
//
// @Summary 下载文件
// @Description 从主机下载指定文件
// @Tags 文件管理
// @Accept json
// @Produce application/octet-stream
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param path query string true "文件路径"
// @Success 200 {file} binary
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/download [get]
func (h *Handler) DownloadFile(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	target := normalizeRemotePath(c.Query("path"))
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		f, err := cli.Open(target)
		if err != nil {
			return err
		}
		defer f.Close()
		name := path.Base(target)
		c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
		c.Header("Content-Type", "application/octet-stream")
		_, _ = io.Copy(c.Writer, f)
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// MakeDir 创建远程目录。
//
// @Summary 创建目录
// @Description 在主机上创建目录，支持递归创建父目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "目录请求 {path: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/mkdir [post]
func (h *Handler) MakeDir(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	target := normalizeRemotePath(req.Path)
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		if err := cli.MkdirAll(target); err != nil {
			return err
		}
		httpx.OK(c, gin.H{"path": target})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// RenamePath 重命名远程文件或目录。
//
// @Summary 重命名文件或目录
// @Description 重命名主机上的文件或目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "重命名请求 {old_path: string, new_path: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files/rename [post]
func (h *Handler) RenamePath(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	oldPath := normalizeRemotePath(req.OldPath)
	newPath := normalizeRemotePath(req.NewPath)
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		if err := cli.Rename(oldPath, newPath); err != nil {
			return err
		}
		httpx.OK(c, gin.H{"old_path": oldPath, "new_path": newPath})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}

// DeletePath 删除远程文件或目录。
//
// @Summary 删除文件或目录
// @Description 删除主机上的文件或空目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param path query string true "文件或目录路径"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/files [delete]
func (h *Handler) DeletePath(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	target := normalizeRemotePath(c.Query("path"))
	err := h.withSFTP(c, hostID, func(cli *sftp.Client) error {
		info, err := cli.Stat(target)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := cli.RemoveDirectory(target); err != nil {
				return err
			}
		} else {
			if err := cli.Remove(target); err != nil {
				return err
			}
		}
		httpx.OK(c, gin.H{"path": target})
		return nil
	})
	if err != nil {
		httpx.Fail(c, xcode.ExternalAPIFail, err.Error())
	}
}
