package dao

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIDiagnosisReportDAO handles report persistence.
type AIDiagnosisReportDAO struct {
	db *gorm.DB
}

// NewAIDiagnosisReportDAO creates an AIDiagnosisReportDAO.
func NewAIDiagnosisReportDAO(db *gorm.DB) *AIDiagnosisReportDAO {
	return &AIDiagnosisReportDAO{db: db}
}

// CreateReport persists a diagnosis report.
func (d *AIDiagnosisReportDAO) CreateReport(ctx context.Context, report *model.AIDiagnosisReport) error {
	return d.db.WithContext(ctx).Create(report).Error
}

// GetReport retrieves a report by ID.
func (d *AIDiagnosisReportDAO) GetReport(ctx context.Context, reportID string) (*model.AIDiagnosisReport, error) {
	var report model.AIDiagnosisReport
	if err := d.db.WithContext(ctx).First(&report, "id = ?", reportID).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

// GetReportByRunID retrieves a report linked to a run.
func (d *AIDiagnosisReportDAO) GetReportByRunID(ctx context.Context, runID string) (*model.AIDiagnosisReport, error) {
	var report model.AIDiagnosisReport
	if err := d.db.WithContext(ctx).First(&report, "run_id = ?", runID).Error; err != nil {
		return nil, err
	}
	return &report, nil
}
