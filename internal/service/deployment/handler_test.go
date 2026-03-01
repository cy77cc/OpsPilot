package deployment

import (
	"testing"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHasPermissionRuntimeAware(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:deployrbac?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	stmts := []string{
		`CREATE TABLE permissions (id INTEGER PRIMARY KEY AUTOINCREMENT, code TEXT);`,
		`CREATE TABLE role_permissions (role_id INTEGER, permission_id INTEGER);`,
		`CREATE TABLE user_roles (user_id INTEGER, role_id INTEGER);`,
		`INSERT INTO permissions (id, code) VALUES (1, 'deploy:k8s:read'), (2, 'deploy:compose:apply');`,
		`INSERT INTO role_permissions (role_id, permission_id) VALUES (10, 1), (10, 2);`,
		`INSERT INTO user_roles (user_id, role_id) VALUES (99, 10);`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
	_ = NewHandler(&svc.ServiceContext{DB: db})
	if !httpx.HasAnyPermission(db, 99, "deploy:k8s:read") {
		t.Fatalf("expected k8s read permission")
	}
	if !httpx.HasAnyPermission(db, 99, "deploy:compose:apply") {
		t.Fatalf("expected compose apply permission")
	}
	if httpx.HasAnyPermission(db, 99, "deploy:k8s:rollback") {
		t.Fatalf("unexpected rollback permission")
	}
}
