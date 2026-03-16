package dao

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type AIDiagnosisReportDAO struct {
	db *gorm.DB
}

func NewAIDiagnosisReportDAO(db *gorm.DB) *AIDiagnosisReportDAO {
	return &AIDiagnosisReportDAO{db: db}
}

func (d *AIDiagnosisReportDAO) CreateReport(ctx context.Context, report *model.AIDiagnosisReport) error {
	return d.db.WithContext(ctx).Create(report).Error
}

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
