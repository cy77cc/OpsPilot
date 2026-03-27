// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现 LLM Provider 管理的业务逻辑:
//   - 模型的 CRUD 操作
//   - API Key 加密存储
//   - 模型导入导出
//   - 默认模型管理
package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"gorm.io/gorm"
)

// LLMProviderRecord LLM Provider 数据库记录。
//
// 存储在 ai_llm_providers 表中，包含模型配置和加密后的 API Key。
type LLMProviderRecord struct {
	ID            uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Provider      string         `gorm:"column:provider;type:varchar(32);not null;index:idx_provider_model,priority:1" json:"provider"`
	Model         string         `gorm:"column:model;type:varchar(128);not null;index:idx_provider_model,priority:2" json:"model"`
	BaseURL       string         `gorm:"column:base_url;type:varchar(512);not null" json:"base_url"`
	APIKey        string         `gorm:"column:api_key;type:varchar(512);not null" json:"-"`
	APIKeyVersion int            `gorm:"column:api_key_version;default:1" json:"api_key_version"`
	Temperature   float64        `gorm:"column:temperature;type:decimal(3,2);default:0.70" json:"temperature"`
	Thinking      bool           `gorm:"column:thinking;default:false" json:"thinking"`
	IsDefault     bool           `gorm:"column:is_default;default:false;index" json:"is_default"`
	IsEnabled     bool           `gorm:"column:is_enabled;default:true;index" json:"is_enabled"`
	SortOrder     int            `gorm:"column:sort_order;default:0;index" json:"sort_order"`
	ConfigVersion int            `gorm:"column:config_version;default:1" json:"config_version"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (LLMProviderRecord) TableName() string {
	return "ai_llm_providers"
}

// LLMProviderView LLM Provider 响应视图。
//
// 用于 API 响应，API Key 已脱敏处理。
type LLMProviderView struct {
	ID            uint64    `json:"id"`
	Name          string    `json:"name"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	BaseURL       string    `json:"base_url"`
	APIKeyMasked  string    `json:"api_key_masked,omitempty"`
	APIKeyVersion int       `json:"api_key_version"`
	Temperature   float64   `json:"temperature"`
	Thinking      bool      `json:"thinking"`
	IsDefault     bool      `json:"is_default"`
	IsEnabled     bool      `json:"is_enabled"`
	SortOrder     int       `json:"sort_order"`
	ConfigVersion int       `json:"config_version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// LLMProviderCreateRequest 创建 LLM Provider 请求。
type LLMProviderCreateRequest struct {
	Name        string   `json:"name" binding:"required"`
	Provider    string   `json:"provider" binding:"required"`
	Model       string   `json:"model" binding:"required"`
	BaseURL     string   `json:"base_url" binding:"required"`
	APIKey      string   `json:"api_key" binding:"required"`
	Temperature *float64 `json:"temperature"`
	Thinking    *bool    `json:"thinking"`
	IsDefault   *bool    `json:"is_default"`
	IsEnabled   *bool    `json:"is_enabled"`
	SortOrder   *int     `json:"sort_order"`
}

// LLMProviderUpdateRequest 更新 LLM Provider 请求。
type LLMProviderUpdateRequest struct {
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

// LLMProviderImportRequest 批量导入 LLM Provider 请求。
type LLMProviderImportRequest struct {
	ReplaceAll bool                       `json:"replace_all"`
	Providers  []LLMProviderCreateRequest `json:"providers"`
}

// LLMProviderImportPreview 导入预览结果。
type LLMProviderImportPreview struct {
	ReplaceAll bool              `json:"replace_all"`
	Total      int               `json:"total"`
	Providers  []LLMProviderView `json:"providers"`
}

// LLMProviderImportResult 导入执行结果。
type LLMProviderImportResult struct {
	ReplaceAll bool              `json:"replace_all"`
	Created    int               `json:"created"`
	Updated    int               `json:"updated"`
	Providers  []LLMProviderView `json:"providers"`
}

// LLMProviderLogic 封装 LLM Provider 业务逻辑。
type LLMProviderLogic struct {
	db *gorm.DB
}

// NewLLMProviderLogic 创建 LLM Provider 业务逻辑实例。
func NewLLMProviderLogic(db *gorm.DB) *LLMProviderLogic {
	return &LLMProviderLogic{db: db}
}

// List 列出所有 LLM Provider。
func (l *LLMProviderLogic) List(ctx context.Context) ([]LLMProviderView, error) {
	rows, err := l.listRecords(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LLMProviderView, 0, len(rows))
	for i := range rows {
		out = append(out, toLLMProviderView(&rows[i]))
	}
	return out, nil
}

// Get 获取单个 LLM Provider。
func (l *LLMProviderLogic) Get(ctx context.Context, id uint64) (*LLMProviderView, error) {
	rec, err := l.getRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, xcode.NewErrCode(xcode.LLMProviderNotFound)
	}
	view := toLLMProviderView(rec)
	return &view, nil
}

// Create 创建 LLM Provider。
func (l *LLMProviderLogic) Create(ctx context.Context, req LLMProviderCreateRequest) (*LLMProviderView, error) {
	if err := validateLLMProviderCreateRequest(req); err != nil {
		return nil, err
	}
	encryptedKey, err := encryptLLMProviderAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}
	rec := &LLMProviderRecord{
		Name:          strings.TrimSpace(req.Name),
		Provider:      strings.TrimSpace(req.Provider),
		Model:         strings.TrimSpace(req.Model),
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKey:        encryptedKey,
		APIKeyVersion: 1,
		Temperature:   defaultFloatPtr(req.Temperature, 0.70),
		Thinking:      defaultBoolPtr(req.Thinking, false),
		IsDefault:     defaultBoolPtr(req.IsDefault, false),
		IsEnabled:     defaultBoolPtr(req.IsEnabled, true),
		SortOrder:     defaultIntPtr(req.SortOrder, 0),
		ConfigVersion: 1,
	}

	if err := l.withTransaction(ctx, func(tx *gorm.DB) error {
		if rec.IsDefault {
			if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("deleted_at IS NULL").Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).Create(rec).Error
	}); err != nil {
		return nil, err
	}
	view := toLLMProviderView(rec)
	return &view, nil
}

// Update 更新 LLM Provider。
func (l *LLMProviderLogic) Update(ctx context.Context, id uint64, req LLMProviderUpdateRequest) (*LLMProviderView, error) {
	rec, err := l.getRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, xcode.NewErrCode(xcode.LLMProviderNotFound)
	}

	updates := map[string]any{}
	setString := func(column string, value *string) {
		if value != nil {
			updates[column] = strings.TrimSpace(*value)
		}
	}
	setBool := func(column string, value *bool) {
		if value != nil {
			updates[column] = *value
		}
	}
	setFloat := func(column string, value *float64) {
		if value != nil {
			updates[column] = *value
		}
	}
	setInt := func(column string, value *int) {
		if value != nil {
			updates[column] = *value
		}
	}

	setString("name", req.Name)
	setString("provider", req.Provider)
	setString("model", req.Model)
	setString("base_url", req.BaseURL)
	setBool("thinking", req.Thinking)
	setBool("is_default", req.IsDefault)
	setBool("is_enabled", req.IsEnabled)
	setInt("sort_order", req.SortOrder)
	setFloat("temperature", req.Temperature)

	if req.APIKey != nil {
		encryptedKey, encErr := encryptLLMProviderAPIKey(*req.APIKey)
		if encErr != nil {
			return nil, encErr
		}
		updates["api_key"] = encryptedKey
		updates["api_key_version"] = rec.APIKeyVersion
	}

	if len(updates) == 0 {
		view := toLLMProviderView(rec)
		return &view, nil
	}

	if err := l.withTransaction(ctx, func(tx *gorm.DB) error {
		if isDefault, ok := updates["is_default"].(bool); ok && isDefault {
			if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("deleted_at IS NULL AND id <> ?", rec.ID).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("id = ? AND deleted_at IS NULL", rec.ID).Updates(updates).Error
	}); err != nil {
		return nil, err
	}

	return l.Get(ctx, id)
}

// Delete 删除 LLM Provider。
func (l *LLMProviderLogic) Delete(ctx context.Context, id uint64) error {
	rec, err := l.getRecord(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil {
		return xcode.NewErrCode(xcode.LLMProviderNotFound)
	}
	if rec.IsDefault {
		return xcode.NewErrCode(xcode.LLMProviderInUse)
	}
	return l.db.WithContext(ctx).Delete(&LLMProviderRecord{}, rec.ID).Error
}

// SetDefault 设置默认 LLM Provider。
func (l *LLMProviderLogic) SetDefault(ctx context.Context, id uint64) error {
	rec, err := l.getRecord(ctx, id)
	if err != nil {
		return err
	}
	if rec == nil {
		return xcode.NewErrCode(xcode.LLMProviderNotFound)
	}
	if !rec.IsEnabled {
		return xcode.NewErrCode(xcode.LLMProviderDisabled)
	}
	return l.withTransaction(ctx, func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("deleted_at IS NULL").Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("id = ? AND deleted_at IS NULL", id).Update("is_default", true).Error
	})
}

// PreviewImport 预览导入结果（不实际执行）。
func (l *LLMProviderLogic) PreviewImport(ctx context.Context, req LLMProviderImportRequest) (*LLMProviderImportPreview, error) {
	if err := validateLLMProviderImportRequest(req); err != nil {
		return nil, err
	}
	items := make([]LLMProviderView, 0, len(req.Providers))
	for i := range req.Providers {
		view, err := createRequestToView(req.Providers[i])
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return &LLMProviderImportPreview{
		ReplaceAll: req.ReplaceAll,
		Total:      len(items),
		Providers:  items,
	}, nil
}

// Import 执行批量导入。
func (l *LLMProviderLogic) Import(ctx context.Context, req LLMProviderImportRequest) (*LLMProviderImportResult, error) {
	if err := validateLLMProviderImportRequest(req); err != nil {
		return nil, err
	}
	result := &LLMProviderImportResult{ReplaceAll: req.ReplaceAll}
	if err := l.withTransaction(ctx, func(tx *gorm.DB) error {
		if req.ReplaceAll {
			if err := tx.WithContext(ctx).Unscoped().Where("deleted_at IS NULL").Delete(&LLMProviderRecord{}).Error; err != nil {
				return err
			}
		}

		for i := range req.Providers {
			reqItem := req.Providers[i]
			rec, err := l.findByProviderModel(ctx, reqItem.Provider, reqItem.Model)
			if err != nil {
				return err
			}
			if rec == nil {
				created, createErr := l.createRecordFromRequest(ctx, reqItem)
				if createErr != nil {
					return createErr
				}
				if err := tx.WithContext(ctx).Create(created).Error; err != nil {
					return err
				}
				result.Created++
				result.Providers = append(result.Providers, toLLMProviderView(created))
				continue
			}

			updates, err := buildUpdatesFromCreateRequest(reqItem, rec)
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("id = ? AND deleted_at IS NULL", rec.ID).Updates(updates).Error; err != nil {
				return err
			}
			updated, loadErr := l.getRecord(ctx, rec.ID)
			if loadErr != nil {
				return loadErr
			}
			if updated != nil {
				result.Updated++
				result.Providers = append(result.Providers, toLLMProviderView(updated))
			}
		}

		if hasAnyDefault(req.Providers) {
			for i := len(req.Providers) - 1; i >= 0; i-- {
				if req.Providers[i].IsDefault != nil && *req.Providers[i].IsDefault {
					rec, findErr := l.findByProviderModel(ctx, req.Providers[i].Provider, req.Providers[i].Model)
					if findErr != nil {
						return findErr
					}
					if rec != nil {
						if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("deleted_at IS NULL").Update("is_default", false).Error; err != nil {
							return err
						}
						if err := tx.WithContext(ctx).Model(&LLMProviderRecord{}).Where("id = ? AND deleted_at IS NULL", rec.ID).Update("is_default", true).Error; err != nil {
							return err
						}
					}
					break
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

// GetProviderForUse 获取 Provider 用于实际调用（解密 API Key）。
func (l *LLMProviderLogic) GetProviderForUse(ctx context.Context, id uint64) (*LLMProviderRecord, error) {
	rec, err := l.getRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, xcode.NewErrCode(xcode.LLMProviderNotFound)
	}
	if !rec.IsEnabled {
		return nil, xcode.NewErrCode(xcode.LLMProviderDisabled)
	}
	return decryptRecord(rec)
}

// GetDefaultForUse 获取默认 Provider 用于实际调用。
func (l *LLMProviderLogic) GetDefaultForUse(ctx context.Context) (*LLMProviderRecord, error) {
	rec, err := l.getDefaultRecord(ctx)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		rec, err = l.getFirstEnabledRecord(ctx)
		if err != nil {
			return nil, err
		}
	}
	if rec == nil {
		fallback, fbErr := configLLMProviderFallback()
		if fbErr != nil {
			return nil, xcode.NewErrCode(xcode.LLMProviderNotFound)
		}
		return fallback, nil
	}
	if !rec.IsEnabled {
		return nil, xcode.NewErrCode(xcode.LLMProviderDisabled)
	}
	return decryptRecord(rec)
}

func (l *LLMProviderLogic) listRecords(ctx context.Context) ([]LLMProviderRecord, error) {
	var rows []LLMProviderRecord
	if err := l.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("is_default DESC, sort_order DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (l *LLMProviderLogic) getRecord(ctx context.Context, id uint64) (*LLMProviderRecord, error) {
	var rec LLMProviderRecord
	if err := l.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (l *LLMProviderLogic) getDefaultRecord(ctx context.Context) (*LLMProviderRecord, error) {
	var rec LLMProviderRecord
	if err := l.db.WithContext(ctx).
		Where("is_default = ? AND is_enabled = ? AND deleted_at IS NULL", true, true).
		Order("sort_order DESC, id DESC").
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (l *LLMProviderLogic) getFirstEnabledRecord(ctx context.Context) (*LLMProviderRecord, error) {
	var rec LLMProviderRecord
	if err := l.db.WithContext(ctx).
		Where("is_enabled = ? AND deleted_at IS NULL", true).
		Order("is_default DESC, sort_order DESC, id DESC").
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (l *LLMProviderLogic) findByProviderModel(ctx context.Context, providerName, modelName string) (*LLMProviderRecord, error) {
	var rec LLMProviderRecord
	if err := l.db.WithContext(ctx).
		Where("provider = ? AND model = ? AND deleted_at IS NULL", strings.TrimSpace(providerName), strings.TrimSpace(modelName)).
		First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (l *LLMProviderLogic) withTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if l.db == nil {
		return fmt.Errorf("llm provider logic not initialized")
	}
	return l.db.WithContext(ctx).Transaction(fn)
}

func (l *LLMProviderLogic) createRecordFromRequest(ctx context.Context, req LLMProviderCreateRequest) (*LLMProviderRecord, error) {
	if err := validateLLMProviderCreateRequest(req); err != nil {
		return nil, err
	}
	encryptedKey, err := encryptLLMProviderAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}
	return &LLMProviderRecord{
		Name:          strings.TrimSpace(req.Name),
		Provider:      strings.TrimSpace(req.Provider),
		Model:         strings.TrimSpace(req.Model),
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKey:        encryptedKey,
		APIKeyVersion: 1,
		Temperature:   defaultFloatPtr(req.Temperature, 0.70),
		Thinking:      defaultBoolPtr(req.Thinking, false),
		IsDefault:     defaultBoolPtr(req.IsDefault, false),
		IsEnabled:     defaultBoolPtr(req.IsEnabled, true),
		SortOrder:     defaultIntPtr(req.SortOrder, 0),
		ConfigVersion: 1,
	}, nil
}

func buildUpdatesFromCreateRequest(req LLMProviderCreateRequest, rec *LLMProviderRecord) (map[string]any, error) {
	updates := map[string]any{
		"name":     strings.TrimSpace(req.Name),
		"provider": strings.TrimSpace(req.Provider),
		"model":    strings.TrimSpace(req.Model),
		"base_url": strings.TrimSpace(req.BaseURL),
	}
	encryptedKey, err := encryptLLMProviderAPIKey(req.APIKey)
	if err != nil {
		return nil, err
	}
	updates["api_key"] = encryptedKey
	updates["temperature"] = defaultFloatPtr(req.Temperature, rec.Temperature)
	updates["thinking"] = defaultBoolPtr(req.Thinking, rec.Thinking)
	updates["is_default"] = defaultBoolPtr(req.IsDefault, rec.IsDefault)
	updates["is_enabled"] = defaultBoolPtr(req.IsEnabled, rec.IsEnabled)
	updates["sort_order"] = defaultIntPtr(req.SortOrder, rec.SortOrder)
	updates["api_key_version"] = rec.APIKeyVersion
	return updates, nil
}

func validateLLMProviderCreateRequest(req LLMProviderCreateRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return xcode.NewErrCodeMsg(xcode.ParamError, "name is required")
	}
	if strings.TrimSpace(req.Provider) == "" {
		return xcode.NewErrCodeMsg(xcode.ParamError, "provider is required")
	}
	if strings.TrimSpace(req.Model) == "" {
		return xcode.NewErrCodeMsg(xcode.ParamError, "model is required")
	}
	if strings.TrimSpace(req.BaseURL) == "" {
		return xcode.NewErrCodeMsg(xcode.ParamError, "base_url is required")
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return xcode.NewErrCodeMsg(xcode.ParamError, "api_key is required")
	}
	return nil
}

func validateLLMProviderImportRequest(req LLMProviderImportRequest) error {
	if len(req.Providers) == 0 {
		return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "providers is required")
	}
	seen := make(map[string]struct{}, len(req.Providers))
	for i := range req.Providers {
		if err := validateLLMProviderCreateRequest(req.Providers[i]); err != nil {
			return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, err.Error())
		}
		key := strings.ToLower(strings.TrimSpace(req.Providers[i].Provider)) + "\x00" + strings.ToLower(strings.TrimSpace(req.Providers[i].Model))
		if _, ok := seen[key]; ok {
			return xcode.NewErrCodeMsg(xcode.LLMImportValidationFail, "duplicate provider/model entry")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func createRequestToView(req LLMProviderCreateRequest) (LLMProviderView, error) {
	if err := validateLLMProviderCreateRequest(req); err != nil {
		return LLMProviderView{}, err
	}
	return LLMProviderView{
		Name:          strings.TrimSpace(req.Name),
		Provider:      strings.TrimSpace(req.Provider),
		Model:         strings.TrimSpace(req.Model),
		BaseURL:       strings.TrimSpace(req.BaseURL),
		APIKeyMasked:  maskLLMProviderAPIKey(req.APIKey),
		Temperature:   defaultFloatPtr(req.Temperature, 0.70),
		Thinking:      defaultBoolPtr(req.Thinking, false),
		IsDefault:     defaultBoolPtr(req.IsDefault, false),
		IsEnabled:     defaultBoolPtr(req.IsEnabled, true),
		SortOrder:     defaultIntPtr(req.SortOrder, 0),
		ConfigVersion: 1,
	}, nil
}

func toLLMProviderView(rec *LLMProviderRecord) LLMProviderView {
	if rec == nil {
		return LLMProviderView{}
	}
	return LLMProviderView{
		ID:            rec.ID,
		Name:          rec.Name,
		Provider:      rec.Provider,
		Model:         rec.Model,
		BaseURL:       rec.BaseURL,
		APIKeyMasked:  maskLLMProviderAPIKey(rec.APIKey),
		APIKeyVersion: rec.APIKeyVersion,
		Temperature:   rec.Temperature,
		Thinking:      rec.Thinking,
		IsDefault:     rec.IsDefault,
		IsEnabled:     rec.IsEnabled,
		SortOrder:     rec.SortOrder,
		ConfigVersion: rec.ConfigVersion,
		CreatedAt:     rec.CreatedAt,
		UpdatedAt:     rec.UpdatedAt,
	}
}

func decryptRecord(rec *LLMProviderRecord) (*LLMProviderRecord, error) {
	if rec == nil {
		return nil, nil
	}
	out := *rec
	if strings.TrimSpace(out.APIKey) != "" && out.ID != 0 {
		plain, err := decryptLLMProviderAPIKey(out.APIKey)
		if err != nil {
			return nil, err
		}
		out.APIKey = plain
	}
	return &out, nil
}

func configLLMProviderFallback() (*LLMProviderRecord, error) {
	if !config.CFG.LLM.Enable {
		return nil, errors.New("llm disabled")
	}
	if strings.TrimSpace(config.CFG.LLM.Provider) == "" || strings.TrimSpace(config.CFG.LLM.Model) == "" || strings.TrimSpace(config.CFG.LLM.BaseURL) == "" {
		return nil, errors.New("llm config incomplete")
	}
	return &LLMProviderRecord{
		ID:            0,
		Name:          strings.TrimSpace(config.CFG.LLM.Model),
		Provider:      strings.TrimSpace(config.CFG.LLM.Provider),
		Model:         strings.TrimSpace(config.CFG.LLM.Model),
		BaseURL:       strings.TrimSpace(config.CFG.LLM.BaseURL),
		APIKey:        strings.TrimSpace(config.CFG.LLM.APIKey),
		APIKeyVersion: 1,
		Temperature:   config.CFG.LLM.Temperature,
		Thinking:      false,
		IsDefault:     true,
		IsEnabled:     true,
		SortOrder:     0,
		ConfigVersion: 1,
		CreatedAt:     time.Time{},
		UpdatedAt:     time.Time{},
	}, nil
}

func defaultBoolPtr(value *bool, def bool) bool {
	if value == nil {
		return def
	}
	return *value
}

func defaultFloatPtr(value *float64, def float64) float64 {
	if value == nil {
		return def
	}
	return *value
}

func defaultIntPtr(value *int, def int) int {
	if value == nil {
		return def
	}
	return *value
}

func hasAnyDefault(items []LLMProviderCreateRequest) bool {
	for i := range items {
		if items[i].IsDefault != nil && *items[i].IsDefault {
			return true
		}
	}
	return false
}
