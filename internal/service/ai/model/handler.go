package model

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HTTPHandler handles admin LLM provider endpoints.
type HTTPHandler struct {
	logic  *LLMProviderLogic
	svcCtx *svc.ServiceContext
}

func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	if svcCtx == nil {
		return &HTTPHandler{}
	}
	return &HTTPHandler{
		logic:  NewLLMProviderLogic(svcCtx.DB),
		svcCtx: svcCtx,
	}
}

func NewHTTPHandlerWithDB(db *gorm.DB) *HTTPHandler {
	return &HTTPHandler{
		logic:  NewLLMProviderLogic(db),
		svcCtx: &svc.ServiceContext{DB: db},
	}
}

func (h *HTTPHandler) ListModels(c *gin.Context) {
	rows, err := h.logic.List(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *HTTPHandler) GetModel(c *gin.Context) {
	id, ok := parseProviderID(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	row, err := h.logic.Get(c.Request.Context(), id)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *HTTPHandler) CreateModel(c *gin.Context) {
	var req LLMProviderCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.Create(c.Request.Context(), req)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *HTTPHandler) UpdateModel(c *gin.Context) {
	id, ok := parseProviderID(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req LLMProviderUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.Update(c.Request.Context(), id, req)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *HTTPHandler) SetDefaultModel(c *gin.Context) {
	id, ok := parseProviderID(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.logic.SetDefault(c.Request.Context(), id); err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *HTTPHandler) DeleteModel(c *gin.Context) {
	id, ok := parseProviderID(c.Param("id"))
	if !ok {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.logic.Delete(c.Request.Context(), id); err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *HTTPHandler) PreviewImport(c *gin.Context) {
	req, err := decodeImportRequest(c)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	result, err := h.logic.PreviewImport(c.Request.Context(), req)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, result)
}

func (h *HTTPHandler) ImportModels(c *gin.Context) {
	req, err := decodeImportRequest(c)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	result, err := h.logic.Import(c.Request.Context(), req)
	if err != nil {
		respondProviderError(c, err)
		return
	}
	httpx.OK(c, result)
}

func decodeImportRequest(c *gin.Context) (LLMProviderImportRequest, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return LLMProviderImportRequest{}, err
	}
	body = []byte(strings.TrimSpace(string(body)))
	if len(body) == 0 {
		return LLMProviderImportRequest{}, xcode.NewErrCodeMsg(xcode.LLMImportInvalidJSON, "request body is required")
	}
	var req LLMProviderImportRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return LLMProviderImportRequest{}, xcode.NewErrCodeMsg(xcode.LLMImportInvalidJSON, err.Error())
	}
	return req, nil
}

func respondProviderError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	codeErr := xcode.FromError(err)
	if codeErr != nil && codeErr.Code != xcode.ServerError {
		httpx.Fail(c, codeErr.Code, codeErr.Msg)
		return
	}
	httpx.ServerErr(c, err)
}

func parseProviderID(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	var parsed uint64
	_, err := fmt.Sscan(raw, &parsed)
	return parsed, err == nil && parsed > 0
}
