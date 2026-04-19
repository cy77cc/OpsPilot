package logic

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateRuleChannelBinding_ConcurrentScopedCallsCreateSingleRow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create-binding-concurrent?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db handle: %v", err)
	}
	// Keep sqlite test deterministic under concurrent goroutines.
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&model.AlertRule{}, &model.AlertNotificationChannel{}, &model.AlertRuleChannelBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{
		ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	projectID := uint(42)
	const workers = 24

	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := l.CreateRuleChannelBinding(context.Background(), projectID, 7, 1001, nil, nil)
			if err != nil {
				errCh <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	errs := make([]error, 0, workers)
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		t.Fatalf("expected no concurrent create errors, got %d first=%v", len(errs), errs[0])
	}

	var scopedCount int64
	if err := db.Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ? AND channel_id = ? AND project_id = ?", 7, 1001, projectID).
		Count(&scopedCount).Error; err != nil {
		t.Fatalf("count scoped bindings: %v", err)
	}
	if scopedCount != 1 {
		t.Fatalf("expected exactly one scoped binding, got %d", scopedCount)
	}
}

func TestResolveChannels_BindingWinsSeverityFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:routing-precedence?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertRuleChannelBinding{}, &model.AlertSeverityRoute{}, &model.AlertNotificationChannel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "bound", Type: "log", Provider: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed bound channel: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 2001, Name: "fallback", Type: "log", Provider: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed fallback channel: %v", err)
	}
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{Scope: "global", Severity: "critical", ChannelIDsJSON: `[2001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	logic := NewLogic(&svc.ServiceContext{DB: db})
	channels, err := logic.ResolveChannelsForAlert(context.Background(), 0, 7, "critical")
	if err != nil {
		t.Fatalf("resolve channels: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != 1001 {
		t.Fatalf("expected bound channel 1001, got %#v", channels)
	}
}

func TestUpdateSeverityRoute_ScopedSuccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-scoped-success?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 101, Scope: "project", ProjectID: &projectID, Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed project route: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 102, Scope: "global", Severity: "critical", ChannelIDsJSON: `[2002]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed global route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	updated, err := l.UpdateSeverityRoute(context.Background(), 101, 42, SeverityRouteInput{
		Scope:      "project",
		Severity:   "critical",
		ChannelIDs: []uint{3003},
		Enabled:    false,
	})
	if err != nil {
		t.Fatalf("update severity route: %v", err)
	}
	if updated.ID != 101 {
		t.Fatalf("expected updated id=101, got %d", updated.ID)
	}
	if updated.ProjectID == nil || *updated.ProjectID != 42 {
		t.Fatalf("expected updated route in project 42, got %#v", updated.ProjectID)
	}
	if updated.Severity != "critical" {
		t.Fatalf("expected severity critical, got %q", updated.Severity)
	}
	if updated.ChannelIDsJSON != `[3003]` {
		t.Fatalf("expected channels [3003], got %s", updated.ChannelIDsJSON)
	}
	if updated.Enabled {
		t.Fatalf("expected enabled=false after update")
	}
}

func TestUpdateSeverityRoute_ReturnsNotFoundOnScopeMismatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-scope-mismatch?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 201, Scope: "project", ProjectID: &projectID, Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed project route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	updated, err := l.UpdateSeverityRoute(context.Background(), 201, 7, SeverityRouteInput{
		Scope:      "project",
		Severity:   "critical",
		ChannelIDs: []uint{9009},
		Enabled:    false,
	})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got row=%#v err=%v", updated, err)
	}

	var row model.AlertSeverityRoute
	if err := db.Where("id = ?", 201).Take(&row).Error; err != nil {
		t.Fatalf("refetch seeded route: %v", err)
	}
	if row.Severity != "warning" || row.ChannelIDsJSON != `[1001]` || !row.Enabled {
		t.Fatalf("expected route unchanged on scope mismatch, got %#v", row)
	}
}

func TestCreateSeverityRoute_DefaultScopeAndNormalizedChannels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create-route-normalized?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	globalRow, err := l.CreateSeverityRoute(context.Background(), 0, SeverityRouteInput{
		Scope:      "   ",
		Severity:   " WARNING ",
		ChannelIDs: []uint{0, 2, 2, 3, 0},
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create global route: %v", err)
	}
	if globalRow.Scope != "global" || globalRow.Severity != "warning" || globalRow.ChannelIDsJSON != `[2,3]` {
		t.Fatalf("unexpected normalized global route: %#v", globalRow)
	}

	projectRow, err := l.CreateSeverityRoute(context.Background(), 9, SeverityRouteInput{
		Scope:      "",
		Severity:   "critical",
		ChannelIDs: []uint{0, 5, 5, 6},
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create project route: %v", err)
	}
	if projectRow.Scope != "project" || projectRow.ProjectID == nil || *projectRow.ProjectID != 9 || projectRow.ChannelIDsJSON != `[5,6]` {
		t.Fatalf("unexpected normalized project route: %#v", projectRow)
	}
}

func TestUpdateSeverityRoute_RejectsBlankSeverityAndInvalidScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-validation?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 301, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	if _, err := l.UpdateSeverityRoute(context.Background(), 301, 0, SeverityRouteInput{
		Scope:      "global",
		Severity:   "   ",
		ChannelIDs: []uint{1},
		Enabled:    true,
	}); err == nil {
		t.Fatalf("expected blank severity to be rejected")
	}

	if _, err := l.UpdateSeverityRoute(context.Background(), 301, 0, SeverityRouteInput{
		Scope:      "cluster",
		Severity:   "warning",
		ChannelIDs: []uint{1},
		Enabled:    true,
	}); err == nil {
		t.Fatalf("expected invalid scope to be rejected")
	}

	if _, err := l.UpdateSeverityRoute(context.Background(), 301, 0, SeverityRouteInput{
		Scope:      "global",
		Severity:   "urgent",
		ChannelIDs: []uint{1},
		Enabled:    true,
	}); err == nil {
		t.Fatalf("expected invalid severity to be rejected")
	}
}

func TestCreateSeverityRoute_RejectsProjectScopeWithoutProjectID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:create-route-project-scope-without-project-id?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	if _, err := l.CreateSeverityRoute(context.Background(), 0, SeverityRouteInput{
		Scope:      "project",
		Severity:   "warning",
		ChannelIDs: []uint{1},
		Enabled:    true,
	}); err == nil {
		t.Fatalf("expected project scope without projectID to be rejected")
	}
}

func TestUpdateSeverityRoute_NormalizesChannelIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-normalize-channels?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 401, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	updated, err := l.UpdateSeverityRoute(context.Background(), 401, 0, SeverityRouteInput{
		Scope:      " GLOBAL ",
		Severity:   " CRITICAL ",
		ChannelIDs: []uint{0, 8, 8, 7, 0},
		Enabled:    false,
	})
	if err != nil {
		t.Fatalf("update route normalize channels: %v", err)
	}
	if updated.Scope != "global" || updated.Severity != "critical" || updated.ChannelIDsJSON != `[8,7]` || updated.Enabled {
		t.Fatalf("unexpected normalized update result: %#v", updated)
	}
}
