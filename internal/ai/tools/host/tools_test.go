package host

import (
	"context"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHostExecAllowsNonReadonlyCommandPastWhitelist(t *testing.T) {
	t.Parallel()

	db := newHostToolTestDB(t)
	toolCtx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	hostExec := HostExec(toolCtx)

	_, err := hostExec.InvokableRun(toolCtx, `{"host_id":1,"command":"systemctl status nginx"}`)
	if err == nil {
		t.Fatal("expected host lookup to fail in test fixture")
	}
	if strings.Contains(err.Error(), "only readonly commands are permitted") {
		t.Fatalf("expected non-dangerous command to pass old readonly whitelist, got %v", err)
	}
}

func TestHostExecStillBlocksDangerousCommand(t *testing.T) {
	t.Parallel()

	db := newHostToolTestDB(t)
	toolCtx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	hostExec := HostExec(toolCtx)

	_, err := hostExec.InvokableRun(toolCtx, `{"host_id":1,"command":"rm -rf /"}`)
	if err == nil {
		t.Fatal("expected dangerous command to be blocked")
	}
	if !strings.Contains(err.Error(), "dangerous command is blocked") {
		t.Fatalf("expected dangerous command block error, got %v", err)
	}
}

func TestHostExecByTargetAllowsNonReadonlyLocalCommandPastWhitelist(t *testing.T) {
	t.Parallel()

	db := newHostToolTestDB(t)
	toolCtx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	hostExec := HostExecByTarget(toolCtx)

	out, err := hostExec.InvokableRun(toolCtx, `{"target":"localhost","command":"printf ok"}`)
	if err != nil {
		t.Fatalf("expected localhost command to run, got %v", err)
	}
	if !strings.Contains(out, `"stdout":"ok"`) {
		t.Fatalf("expected stdout to contain command output, got %s", out)
	}
}

func newHostToolTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Node{}); err != nil {
		t.Fatalf("migrate node table: %v", err)
	}
	return db
}
