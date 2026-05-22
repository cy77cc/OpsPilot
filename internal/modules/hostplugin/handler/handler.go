package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	hostpluginlogic "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *hostpluginlogic.Service
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		service: hostpluginlogic.NewService(svcCtx),
	}
}

func (h *Handler) ListCatalog(c *gin.Context) {
	plugins, err := h.service.ListCatalog(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": plugins, "total": len(plugins)})
}

func (h *Handler) ListHostInstances(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid host id")
		return
	}

	instances, err := h.service.ListInstancesByHost(c.Request.Context(), hostID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": instances, "total": len(instances), "host_id": hostID})
}

func (h *Handler) RunInstanceAction(c *gin.Context) {
	httpx.OK(c, gin.H{
		"instance_id": c.Param("instance_id"),
		"status":      "pending",
		"message":     "host plugin instance actions are not implemented yet",
	})
}

func (h *Handler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("task_id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid task id")
		return
	}

	task, err := h.service.GetTask(c.Request.Context(), taskID)
	if err != nil {
		httpx.NotFound(c, "task not found")
		return
	}
	httpx.OK(c, task)
}

func (h *Handler) ListTaskLogs(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("task_id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid task id")
		return
	}

	logs, err := h.service.ListTaskLogs(c.Request.Context(), taskID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": logs, "total": len(logs), "task_id": taskID})
}

// InstallPlugin installs a plugin on an existing host.
func (h *Handler) InstallPlugin(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid host id")
		return
	}

	var req struct {
		PluginKey string `json:"plugin_key"`
		Version   string `json:"version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	if req.PluginKey == "" {
		req.PluginKey = "opsagent"
	}

	// Get host IP
	host, err := h.service.GetHost(c.Request.Context(), hostID)
	if err != nil {
		httpx.NotFound(c, "host not found")
		return
	}

	taskID, err := h.service.InstallOnHost(c.Request.Context(), hostID, req.PluginKey, req.Version, host.IP)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"task_id": taskID, "message": "install task created"})
}

// UninstallPlugin uninstalls a plugin from a host.
func (h *Handler) UninstallPlugin(c *gin.Context) {
	hostID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid host id")
		return
	}

	var req struct {
		InstanceID uint64 `json:"instance_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}

	taskID, err := h.service.UninstallOnHost(c.Request.Context(), hostID, req.InstanceID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"task_id": taskID, "message": "uninstall task created"})
}

// ListPackages returns all uploaded plugin packages.
func (h *Handler) ListPackages(c *gin.Context) {
	pkgs, err := h.service.ListPackages(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": pkgs, "total": len(pkgs)})
}

// UploadPackage handles plugin package upload.
func (h *Handler) UploadPackage(c *gin.Context) {
	pluginKey := c.PostForm("plugin_key")
	version := c.PostForm("version")
	arch := c.PostForm("arch")

	if pluginKey == "" || version == "" || arch == "" {
		httpx.BadRequest(c, "plugin_key, version, and arch are required")
		return
	}
	if arch != "amd64" && arch != "arm64" {
		httpx.BadRequest(c, "arch must be amd64 or arm64")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		httpx.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	// Create storage directory
	storageDir := filepath.Join("storage", "packages", pluginKey, version, arch)
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		httpx.ServerErr(c, fmt.Errorf("create storage dir: %w", err))
		return
	}

	storagePath := filepath.Join(storageDir, header.Filename)

	// Write file
	dst, err := os.Create(storagePath)
	if err != nil {
		httpx.ServerErr(c, fmt.Errorf("create file: %w", err))
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		httpx.ServerErr(c, fmt.Errorf("write file: %w", err))
		return
	}

	// Compute checksum
	checksum, err := computeFileChecksum(storagePath)
	if err != nil {
		httpx.ServerErr(c, fmt.Errorf("compute checksum: %w", err))
		return
	}

	pkg := &hostpluginlogic.HostPluginPackageInput{
		PluginKey:   pluginKey,
		Version:     version,
		Arch:        arch,
		Filename:    header.Filename,
		StoragePath: storagePath,
		Checksum:    checksum,
		SizeBytes:   size,
	}

	if err := h.service.CreatePackageFromInput(c.Request.Context(), pkg); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "package uploaded", "checksum": checksum, "size_bytes": size})
}

// DeletePackage removes a plugin package.
func (h *Handler) DeletePackage(c *gin.Context) {
	packageID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.BadRequest(c, "invalid package id")
		return
	}

	if err := h.service.DeletePackage(c.Request.Context(), packageID); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"message": "package deleted"})
}

func computeFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
