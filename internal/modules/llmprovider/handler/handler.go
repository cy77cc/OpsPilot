// Package handler 提供 LLM Provider 的 HTTP 处理器。
package handler

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LLMProviderRecord = model.AILLMProvider

type HTTPHandler struct {
	db *gorm.DB
}

func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	if svcCtx == nil {
		return &HTTPHandler{}
	}
	return &HTTPHandler{db: svcCtx.DB}
}

func NewHTTPHandlerWithDB(db *gorm.DB) *HTTPHandler {
	return &HTTPHandler{db: db}
}

type providerPayload struct {
	Name        string  `json:"name"`
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key"`
	Temperature float64 `json:"temperature"`
	Thinking    bool    `json:"thinking"`
	IsDefault   bool    `json:"is_default"`
	IsEnabled   bool    `json:"is_enabled"`
	SortOrder   int     `json:"sort_order"`
}

type importPayload struct {
	ReplaceAll bool              `json:"replace_all"`
	Providers  []providerPayload `json:"providers"`
}

func (h *HTTPHandler) ListModels(c *gin.Context) {
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	rows, err := dao.ListAll(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	list := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		list = append(list, serializeProvider(item))
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *HTTPHandler) GetModel(c *gin.Context) {
	row, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}
	httpx.OK(c, serializeProvider(*row))
}

func (h *HTTPHandler) CreateModel(c *gin.Context) {
	var req providerPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	record, err := buildProviderRecord(req, nil)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if record.IsDefault {
		if err := dao.ClearDefault(c.Request.Context()); err != nil {
			httpx.ServerErr(c, err)
			return
		}
	}
	if err := dao.Create(c.Request.Context(), record); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) UpdateModel(c *gin.Context) {
	existing, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}
	var req providerPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	record, err := buildProviderRecord(req, existing)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if record.IsDefault {
		if err := dao.ClearDefault(c.Request.Context()); err != nil {
			httpx.ServerErr(c, err)
			return
		}
	}
	if err := dao.Update(c.Request.Context(), record); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) SetDefaultModel(c *gin.Context) {
	record, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if err := dao.ClearDefault(c.Request.Context()); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	record.IsDefault = true
	if err := dao.Update(c.Request.Context(), record); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) DeleteModel(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if err := dao.SoftDelete(c.Request.Context(), id); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"id": id, "deleted": true})
}

func (h *HTTPHandler) PreviewImport(c *gin.Context) {
	payload, ok := h.decodeImportPayload(c)
	if !ok {
		return
	}
	providers := make([]map[string]any, 0, len(payload.Providers))
	for _, item := range payload.Providers {
		record, err := buildProviderRecord(item, nil)
		if err != nil {
			httpx.Fail(c, xcode.LLMImportValidationFail, err.Error())
			return
		}
		providers = append(providers, serializeProvider(*record))
	}
	httpx.OK(c, gin.H{
		"replace_all": payload.ReplaceAll,
		"total":       len(providers),
		"providers":   providers,
		// Backward compatibility for callers still reading old keys.
		"count": len(providers),
		"list":  providers,
	})
}

func (h *HTTPHandler) ImportModels(c *gin.Context) {
	payload, ok := h.decodeImportPayload(c)
	if !ok {
		return
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	created := make([]map[string]any, 0, len(payload.Providers))
	for _, item := range payload.Providers {
		record, err := buildProviderRecord(item, nil)
		if err != nil {
			httpx.Fail(c, xcode.LLMImportValidationFail, err.Error())
			return
		}
		if record.IsDefault {
			if err := dao.ClearDefault(c.Request.Context()); err != nil {
				httpx.ServerErr(c, err)
				return
			}
		}
		if err := dao.Create(c.Request.Context(), record); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		created = append(created, serializeProvider(*record))
	}
	httpx.OK(c, gin.H{
		"replace_all": payload.ReplaceAll,
		"created":     len(created),
		"updated":     0,
		"providers":   created,
		// Backward compatibility for callers still reading old keys.
		"count": len(created),
		"list":  created,
	})
}

func (h *HTTPHandler) dao() *dao.LLMProviderDAO {
	if h == nil || h.db == nil {
		return nil
	}
	return dao.NewLLMProviderDAO(h.db)
}

func (h *HTTPHandler) loadProviderByParam(c *gin.Context) (*LLMProviderRecord, bool) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return nil, false
	}
	dao := h.dao()
	if dao == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return nil, false
	}
	row, err := dao.GetByID(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return nil, false
	}
	if row == nil {
		httpx.Fail(c, xcode.LLMProviderNotFound, "")
		return nil, false
	}
	return row, true
}

func (h *HTTPHandler) decodeImportPayload(c *gin.Context) (*importPayload, bool) {
	var payload importPayload
	if err := json.NewDecoder(c.Request.Body).Decode(&payload); err != nil {
		httpx.Fail(c, xcode.LLMImportInvalidJSON, err.Error())
		return nil, false
	}
	if len(payload.Providers) == 0 {
		httpx.Fail(c, xcode.LLMImportValidationFail, "providers is required")
		return nil, false
	}
	for _, item := range payload.Providers {
		if _, err := buildProviderRecord(item, nil); err != nil {
			httpx.Fail(c, xcode.LLMImportValidationFail, err.Error())
			return nil, false
		}
	}
	return &payload, true
}

func parseUintParam(c *gin.Context, key string) (uint64, bool) {
	value := strings.TrimSpace(c.Param(key))
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		httpx.Fail(c, xcode.ParamError, key+" is invalid")
		return 0, false
	}
	return id, true
}

func buildProviderRecord(input providerPayload, existing *LLMProviderRecord) (*LLMProviderRecord, error) {
	record := &LLMProviderRecord{}
	if existing != nil {
		*record = *existing
	}
	record.Name = strings.TrimSpace(input.Name)
	record.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	record.Model = strings.TrimSpace(input.Model)
	record.BaseURL = strings.TrimSpace(input.BaseURL)
	record.Temperature = input.Temperature
	record.Thinking = input.Thinking
	record.IsDefault = input.IsDefault
	record.IsEnabled = input.IsEnabled
	record.SortOrder = input.SortOrder
	if existing == nil && !input.IsEnabled {
		record.IsEnabled = true
	}
	if record.Name == "" || record.Provider == "" || record.Model == "" || record.BaseURL == "" {
		return nil, xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "name, provider, model, base_url are required")
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" && existing != nil {
		return record, nil
	}
	if apiKey == "" {
		return nil, xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "api_key is required")
	}
	encrypted, err := utils.EncryptText(apiKey, strings.TrimSpace(config.CFG.Security.EncryptionKey))
	if err != nil {
		return nil, err
	}
	record.APIKey = encrypted
	return record, nil
}

func serializeProvider(record LLMProviderRecord) map[string]any {
	return map[string]any{
		"id":              record.ID,
		"name":            record.Name,
		"provider":        record.Provider,
		"model":           record.Model,
		"base_url":        record.BaseURL,
		"temperature":     record.Temperature,
		"thinking":        record.Thinking,
		"is_default":      record.IsDefault,
		"is_enabled":      record.IsEnabled,
		"sort_order":      record.SortOrder,
		"api_key_masked":  maskedSecret(record.APIKey),
		"config_version":  record.ConfigVersion,
		"api_key_version": record.APIKeyVersion,
	}
}

func maskedSecret(cipherText string) string {
	if strings.TrimSpace(cipherText) == "" {
		return ""
	}
	plain, err := utils.DecryptText(cipherText, strings.TrimSpace(config.CFG.Security.EncryptionKey))
	if err != nil || plain == "" {
		return "****"
	}
	if len(plain) <= 8 {
		return plain[:2] + "****"
	}
	return plain[:4] + "****" + plain[len(plain)-4:]
}

func NewDAO(db *gorm.DB) *dao.LLMProviderDAO {
	return dao.NewLLMProviderDAO(db)
}

func WithDB(ctx context.Context, db *gorm.DB) context.Context {
	if db == nil {
		return ctx
	}
	return ctx
}
