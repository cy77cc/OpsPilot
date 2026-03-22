package ai

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestRiskPolicyListEnabledByToolName_UsesToolNameAndEnabledOnly(t *testing.T) {
	capture := &sqlCaptureLogger{}
	db, err := gorm.Open(sqlite.Open("file:risk-policy-dao?mode=memory&cache=shared"), &gorm.Config{
		Logger: capture,
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	dao := NewAIToolRiskPolicyDAO(db.Session(&gorm.Session{DryRun: true, Logger: capture}))
	if _, err := dao.ListEnabledByToolName(context.Background(), "kubectl"); err != nil {
		t.Fatalf("list enabled by tool name: %v", err)
	}

	sql := strings.ToLower(capture.latestSQL())
	for _, fragment := range []string{
		"from `ai_tool_risk_policies`",
		`where tool_name = "kubectl" and enabled = true`,
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected sql to contain %q, got %q", fragment, sql)
		}
	}
	for _, fragment := range []string{"scene", "command_class", "argument_rules"} {
		if strings.Contains(sql, fragment) {
			t.Fatalf("did not expect sql to contain %q, got %q", fragment, sql)
		}
	}
}

type sqlCaptureLogger struct {
	mu  sync.Mutex
	sql []string
}

func (l *sqlCaptureLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *sqlCaptureLogger) Info(context.Context, string, ...interface{})     {}
func (l *sqlCaptureLogger) Warn(context.Context, string, ...interface{})     {}
func (l *sqlCaptureLogger) Error(context.Context, string, ...interface{})    {}

func (l *sqlCaptureLogger) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	l.mu.Lock()
	l.sql = append(l.sql, sql)
	l.mu.Unlock()
}

func (l *sqlCaptureLogger) latestSQL() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.sql) == 0 {
		return ""
	}
	return l.sql[len(l.sql)-1]
}
