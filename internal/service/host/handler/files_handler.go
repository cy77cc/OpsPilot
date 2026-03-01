package handler

import (
	"bytes"
	"errors"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	sshclient "github.com/cy77cc/k8s-manage/internal/client/ssh"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/pkg/sftp"
)

const maxInlineReadBytes = 1024 * 1024 // 1MB

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
