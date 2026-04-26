// Package handler 提供 LLM Provider 的 HTTP 处理器。
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/core/utils"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type LLMProviderRecord = model.AILLMProvider

const defaultTemperature = 0.7

type llmProviderDBCtxKey struct{}

type HTTPHandler struct {
	db  *gorm.DB
	rdb redis.UniversalClient
}

func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	if svcCtx == nil {
		return &HTTPHandler{}
	}
	return &HTTPHandler{
		db:  svcCtx.DB,
		rdb: svcCtx.Rdb,
	}
}

func NewHTTPHandlerWithDB(db *gorm.DB) *HTTPHandler {
	return &HTTPHandler{db: db}
}

type providerPayload struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	BaseURL     string   `json:"base_url"`
	APIKey      string   `json:"api_key"`
	Temperature *float64 `json:"temperature"`
	Thinking    *bool    `json:"thinking"`
	IsDefault   *bool    `json:"is_default"`
	IsEnabled   *bool    `json:"is_enabled"`
	SortOrder   *int     `json:"sort_order"`
}

type providerUpdatePayload struct {
	Name        *string  `json:"name"`
	Provider    *string  `json:"provider"`
	Model       *string  `json:"model"`
	BaseURL     *string  `json:"base_url"`
	APIKey      *string  `json:"api_key"`
	Temperature *float64 `json:"temperature"`
	Thinking    *bool    `json:"thinking"`
	IsDefault   *bool    `json:"is_default"`
	IsEnabled   *bool    `json:"is_enabled"`
	SortOrder   *int     `json:"sort_order"`
}

type importPayload struct {
	ReplaceAll bool              `json:"replace_all"`
	Providers  []providerPayload `json:"providers"`
}

type normalizedImportPayload struct {
	ReplaceAll bool
	Providers  []LLMProviderRecord
}

func (h *HTTPHandler) ListModels(c *gin.Context) {
	dao := h.dao(c.Request.Context())
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
	record, err := buildProviderRecord(req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}

	db := h.resolveDB(c.Request.Context())
	if db == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if record.IsDefault {
			if err := clearDefaultInTx(tx); err != nil {
				return err
			}
		}
		return tx.Create(record).Error
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.publishConfigChange(c.Request.Context())
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) UpdateModel(c *gin.Context) {
	existing, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}

	var req providerUpdatePayload
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	record, err := patchProviderRecord(existing, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}

	db := h.resolveDB(c.Request.Context())
	if db == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if record.IsDefault {
			if err := clearDefaultInTx(tx); err != nil {
				return err
			}
		}
		return tx.Model(&LLMProviderRecord{}).Where("id = ?", record.ID).Updates(map[string]any{
			"name":            record.Name,
			"provider":        record.Provider,
			"model":           record.Model,
			"base_url":        record.BaseURL,
			"api_key":         record.APIKey,
			"temperature":     record.Temperature,
			"thinking":        record.Thinking,
			"is_default":      record.IsDefault,
			"is_enabled":      record.IsEnabled,
			"sort_order":      record.SortOrder,
			"api_key_version": record.APIKeyVersion,
			"config_version":  record.ConfigVersion,
		}).Error
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.publishConfigChange(c.Request.Context())
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) SetDefaultModel(c *gin.Context) {
	record, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}
	if !record.IsEnabled {
		httpx.Fail(c, xcode.LLMProviderDisabled, "")
		return
	}

	db := h.resolveDB(c.Request.Context())
	if db == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}
	if err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := clearDefaultInTx(tx); err != nil {
			return err
		}
		return tx.Model(&LLMProviderRecord{}).
			Where("id = ? AND deleted_at IS NULL", record.ID).
			Update("is_default", true).Error
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	record.IsDefault = true
	h.publishConfigChange(c.Request.Context())
	httpx.OK(c, serializeProvider(*record))
}

func (h *HTTPHandler) DeleteModel(c *gin.Context) {
	record, ok := h.loadProviderByParam(c)
	if !ok {
		return
	}
	if record.IsDefault {
		httpx.Fail(c, xcode.LLMProviderInUse, "默认模型不能删除")
		return
	}

	db := h.resolveDB(c.Request.Context())
	if db == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}

	result := db.WithContext(c.Request.Context()).
		Where("id = ? AND deleted_at IS NULL", record.ID).
		Delete(&LLMProviderRecord{})
	if result.Error != nil {
		httpx.ServerErr(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, xcode.LLMProviderNotFound, "")
		return
	}
	h.publishConfigChange(c.Request.Context())
	httpx.OK(c, gin.H{"id": record.ID, "deleted": true})
}

func (h *HTTPHandler) PreviewImport(c *gin.Context) {
	payload, ok := h.decodeImportPayload(c)
	if !ok {
		return
	}
	providers := make([]map[string]any, 0, len(payload.Providers))
	for _, item := range payload.Providers {
		providers = append(providers, serializeProvider(item))
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

	db := h.resolveDB(c.Request.Context())
	if db == nil {
		httpx.ServerErr(c, gorm.ErrInvalidDB)
		return
	}

	if err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if payload.ReplaceAll {
			if err := tx.Where("deleted_at IS NULL").Delete(&LLMProviderRecord{}).Error; err != nil {
				return err
			}
		}

		hasDefault := false
		for _, row := range payload.Providers {
			if row.IsDefault {
				hasDefault = true
				break
			}
		}
		if hasDefault {
			if err := clearDefaultInTx(tx); err != nil {
				return err
			}
		}

		for i := range payload.Providers {
			row := payload.Providers[i]
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			payload.Providers[i] = row
		}
		return nil
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	h.publishConfigChange(c.Request.Context())

	created := make([]map[string]any, 0, len(payload.Providers))
	for _, row := range payload.Providers {
		created = append(created, serializeProvider(row))
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

func (h *HTTPHandler) publishConfigChange(ctx context.Context) {
	if h.rdb != nil {
		_ = h.rdb.Publish(ctx, constants.LLMConfigUpdateChannel, "updated").Err()
	}
}

func (h *HTTPHandler) resolveDB(ctx context.Context) *gorm.DB {
	if h != nil && h.db != nil {
		return h.db
	}
	return DBFromContext(ctx)
}

func (h *HTTPHandler) dao(ctx context.Context) *dao.LLMProviderDAO {
	db := h.resolveDB(ctx)
	if db == nil {
		return nil
	}
	return dao.NewLLMProviderDAO(db)
}

func (h *HTTPHandler) loadProviderByParam(c *gin.Context) (*LLMProviderRecord, bool) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		return nil, false
	}
	dao := h.dao(c.Request.Context())
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

func (h *HTTPHandler) decodeImportPayload(c *gin.Context) (*normalizedImportPayload, bool) {
	var payload importPayload
	if err := json.NewDecoder(c.Request.Body).Decode(&payload); err != nil {
		httpx.Fail(c, xcode.LLMImportInvalidJSON, err.Error())
		return nil, false
	}
	if len(payload.Providers) == 0 {
		httpx.Fail(c, xcode.LLMImportValidationFail, "providers is required")
		return nil, false
	}

	rows := make([]LLMProviderRecord, 0, len(payload.Providers))
	defaultCount := 0
	for _, item := range payload.Providers {
		record, err := buildProviderRecord(item)
		if err != nil {
			httpx.Fail(c, xcode.LLMImportValidationFail, err.Error())
			return nil, false
		}
		if record.IsDefault {
			defaultCount++
		}
		rows = append(rows, *record)
	}
	if defaultCount > 1 {
		httpx.Fail(c, xcode.LLMImportValidationFail, "at most one default model is allowed in import payload")
		return nil, false
	}

	return &normalizedImportPayload{
		ReplaceAll: payload.ReplaceAll,
		Providers:  rows,
	}, true
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

func buildProviderRecord(input providerPayload) (*LLMProviderRecord, error) {
	record := &LLMProviderRecord{
		Name:        strings.TrimSpace(input.Name),
		Provider:    strings.TrimSpace(strings.ToLower(input.Provider)),
		Model:       strings.TrimSpace(input.Model),
		BaseURL:     strings.TrimSpace(input.BaseURL),
		Temperature: floatValue(input.Temperature, defaultTemperature),
		Thinking:    boolValue(input.Thinking, false),
		IsDefault:   boolValue(input.IsDefault, false),
		IsEnabled:   boolValue(input.IsEnabled, true),
		SortOrder:   intValue(input.SortOrder, 0),
	}
	if err := validateProviderRecord(record); err != nil {
		return nil, err
	}
	if err := validateBaseURL(record.BaseURL); err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(input.APIKey)
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

func patchProviderRecord(existing *LLMProviderRecord, input providerUpdatePayload) (*LLMProviderRecord, error) {
	record := &LLMProviderRecord{}
	if existing != nil {
		*record = *existing
	}
	if input.Name != nil {
		record.Name = strings.TrimSpace(*input.Name)
	}
	if input.Provider != nil {
		record.Provider = strings.TrimSpace(strings.ToLower(*input.Provider))
	}
	if input.Model != nil {
		record.Model = strings.TrimSpace(*input.Model)
	}
	if input.BaseURL != nil {
		record.BaseURL = strings.TrimSpace(*input.BaseURL)
		if err := validateBaseURL(record.BaseURL); err != nil {
			return nil, err
		}
	}
	if input.Temperature != nil {
		record.Temperature = *input.Temperature
	}
	if input.Thinking != nil {
		record.Thinking = *input.Thinking
	}
	if input.IsDefault != nil {
		record.IsDefault = *input.IsDefault
	}
	if input.IsEnabled != nil {
		record.IsEnabled = *input.IsEnabled
	}
	if input.SortOrder != nil {
		record.SortOrder = *input.SortOrder
	}
	if err := validateProviderRecord(record); err != nil {
		return nil, err
	}

	if input.APIKey == nil {
		return record, nil
	}

	apiKey := strings.TrimSpace(*input.APIKey)
	if apiKey == "" {
		return nil, xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "api_key cannot be empty")
	}
	encrypted, err := utils.EncryptText(apiKey, strings.TrimSpace(config.CFG.Security.EncryptionKey))
	if err != nil {
		return nil, err
	}
	record.APIKey = encrypted
	return record, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url is not a valid URL: %w", err)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "0.0.0.0")) {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "base_url must use https or http for localhost")
	}
	return nil
}

func validateProviderRecord(record *LLMProviderRecord) error {
	if record == nil {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "record is required")
	}
	if record.Name == "" || record.Provider == "" || record.Model == "" || record.BaseURL == "" {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "name, provider, model, base_url are required")
	}
	if record.Temperature < 0 || record.Temperature > 2 {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "temperature must be between 0 and 2")
	}
	if record.IsDefault && !record.IsEnabled {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "default model must be enabled")
	}
	return nil
}

func clearDefaultInTx(tx *gorm.DB) error {
	return tx.Model(&LLMProviderRecord{}).
		Where("is_default = ? AND deleted_at IS NULL", true).
		Update("is_default", false).Error
}

func floatValue(v *float64, fallback float64) float64 {
	if v == nil {
		return fallback
	}
	return *v
}

func boolValue(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func intValue(v *int, fallback int) int {
	if v == nil {
		return fallback
	}
	return *v
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
	switch n := len(plain); {
	case n <= 1:
		return "****"
	case n <= 4:
		return plain[:1] + "****"
	case n <= 8:
		return plain[:2] + "****" + plain[n-1:]
	default:
		return plain[:4] + "****" + plain[n-4:]
	}
}

func NewDAO(db *gorm.DB) *dao.LLMProviderDAO {
	return dao.NewLLMProviderDAO(db)
}

func WithDB(ctx context.Context, db *gorm.DB) context.Context {
	if db == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, llmProviderDBCtxKey{}, db)
}

func DBFromContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return nil
	}
	db, _ := ctx.Value(llmProviderDBCtxKey{}).(*gorm.DB)
	return db
}
