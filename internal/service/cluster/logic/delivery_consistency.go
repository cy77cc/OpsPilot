package logic

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
)

type DriftResult struct {
	Drifted              bool   `json:"drifted"`
	AppName              string `json:"app_name"`
	DesiredRevision      string `json:"desired_revision"`
	ActualRevision       string `json:"actual_revision"`
	RemediationScheduled bool   `json:"remediation_scheduled"`
	Reason               string `json:"reason,omitempty"`
}

type gitopsReleaseRow struct {
	ID          uint   `gorm:"column:id" json:"id"`
	ClusterID   uint   `gorm:"column:cluster_id" json:"cluster_id"`
	AppName     string `gorm:"column:app_name" json:"app_name"`
	Environment string `gorm:"column:environment" json:"environment"`
	GitRevision string `gorm:"column:git_revision" json:"git_revision"`
	SyncResult  string `gorm:"column:sync_result" json:"sync_result"`
	RollbackRef string `gorm:"column:rollback_ref" json:"rollback_ref"`
	AuditID     uint   `gorm:"column:audit_id" json:"audit_id"`
}

func (gitopsReleaseRow) TableName() string { return "gitops_app_releases" }

func EvaluateDrift(ctx context.Context, db *gorm.DB, clusterID uint, appName string, desiredRevision string) (DriftResult, error) {
	app := strings.TrimSpace(appName)
	desired := strings.TrimSpace(desiredRevision)
	if db == nil {
		return DriftResult{}, errors.New("db is required")
	}
	if clusterID == 0 || app == "" {
		return DriftResult{}, errors.New("cluster_id and app_name are required")
	}

	var latest gitopsReleaseRow
	err := db.WithContext(ctx).
		Where("cluster_id = ? AND app_name = ?", clusterID, app).
		Order("id DESC").
		First(&latest).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return DriftResult{
				Drifted:              false,
				AppName:              app,
				DesiredRevision:      desired,
				ActualRevision:       "",
				RemediationScheduled: false,
				Reason:               "no_release_record",
			}, nil
		}
		return DriftResult{}, err
	}

	result := DriftResult{
		AppName:         app,
		DesiredRevision: desired,
		ActualRevision:  strings.TrimSpace(latest.GitRevision),
	}
	if desired == "" || result.ActualRevision == desired {
		result.Drifted = false
		return result, nil
	}

	result.Drifted = true
	result.RemediationScheduled = true
	result.Reason = "revision_mismatch"
	return result, nil
}

func ShouldTripGitOpsCircuitBreaker(ctx context.Context, db *gorm.DB, clusterID uint, appName string, threshold int) (bool, int, error) {
	if db == nil {
		return false, 0, errors.New("db is required")
	}
	if clusterID == 0 || strings.TrimSpace(appName) == "" {
		return false, 0, errors.New("cluster_id and app_name are required")
	}
	if threshold <= 0 {
		threshold = 3
	}

	var rows []gitopsReleaseRow
	if err := db.WithContext(ctx).
		Where("cluster_id = ? AND app_name = ?", clusterID, strings.TrimSpace(appName)).
		Order("id DESC").
		Limit(threshold).
		Find(&rows).Error; err != nil {
		return false, 0, err
	}
	if len(rows) < threshold {
		return false, 0, nil
	}

	consecutive := 0
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.SyncResult), "failed") {
			consecutive++
			continue
		}
		break
	}
	return consecutive >= threshold, consecutive, nil
}
