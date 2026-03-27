// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 诊断报告的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIDiagnosisReportDAO 提供 AI 诊断报告的数据访问功能。
type AIDiagnosisReportDAO struct {
	db *gorm.DB
}

// NewAIDiagnosisReportDAO 创建 AI 诊断报告 DAO 实例。
func NewAIDiagnosisReportDAO(db *gorm.DB) *AIDiagnosisReportDAO {
	return &AIDiagnosisReportDAO{db: db}
}

// CreateReport 创建新的诊断报告。
func (d *AIDiagnosisReportDAO) CreateReport(ctx context.Context, report *model.AIDiagnosisReport) error {
	return d.db.WithContext(ctx).Create(report).Error
}

// GetReport 根据报告 ID 获取诊断报告。
//
// 参数:
//   - ctx: 上下文
//   - reportID: 报告 ID
//
// 返回: 诊断报告或 nil（不存在时）
func (d *AIDiagnosisReportDAO) GetReport(ctx context.Context, reportID string) (*model.AIDiagnosisReport, error) {
	var report model.AIDiagnosisReport
	err := d.db.WithContext(ctx).Where("id = ?", reportID).First(&report).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// GetReportByRunID 根据运行 ID 获取诊断报告。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//
// 返回: 诊断报告或 nil（不存在时）
func (d *AIDiagnosisReportDAO) GetReportByRunID(ctx context.Context, runID string) (*model.AIDiagnosisReport, error) {
	var report model.AIDiagnosisReport
	err := d.db.WithContext(ctx).Where("run_id = ?", runID).First(&report).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}
