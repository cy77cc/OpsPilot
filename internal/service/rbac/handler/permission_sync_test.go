package handler

import (
	"fmt"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRBACDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Role{}, &model.Permission{}, &model.UserRole{}, &model.RolePermission{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSyncUserRolesTxRejectsUnknownRoleCodes(t *testing.T) {
	db := setupRBACDB(t)
	h := NewHandler(&svc.ServiceContext{DB: db})

	if err := db.Create(&model.User{Username: "testuser", PasswordHash: "x", Email: "u@example.com", Status: 1}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.Role{Name: "Admin", Code: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return h.syncUserRolesTx(tx, 1, []string{"admin", "missing-role"})
	})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	if _, ok := err.(*codeValidationError); !ok {
		t.Fatalf("expected codeValidationError, got %T", err)
	}
}

func TestSyncRolePermissionsTxRejectsUnknownPermissionCodes(t *testing.T) {
	db := setupRBACDB(t)
	h := NewHandler(&svc.ServiceContext{DB: db})

	if err := db.Create(&model.Role{Name: "Admin", Code: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&model.Permission{Name: "Read", Code: "rbac:read", Resource: "rbac", Action: "read", Status: 1}).Error; err != nil {
		t.Fatalf("seed permission: %v", err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		return h.syncRolePermissionsTx(tx, 1, []string{"rbac:read", "rbac:missing"})
	})
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	if _, ok := err.(*codeValidationError); !ok {
		t.Fatalf("expected codeValidationError, got %T", err)
	}
}

// TestAssignRole tests assigning a role to a user.
func TestAssignRole(t *testing.T) {
	db := setupRBACDB(t)
	h := NewHandler(&svc.ServiceContext{DB: db})

	// Create test data
	user := &model.User{Username: "testuser", PasswordHash: "x", Email: "test@example.com", Status: 1}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	role := &model.Role{Name: "Developer", Code: "developer", Status: 1}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	// Assign role
	err := db.Transaction(func(tx *gorm.DB) error {
		return h.syncUserRolesTx(tx, uint64(user.ID), []string{"developer"})
	})
	if err != nil {
		t.Fatalf("assign role: %v", err)
	}

	// Verify assignment
	var userRole model.UserRole
	if err := db.Where("user_id = ? AND role_id = ?", int64(user.ID), int64(role.ID)).First(&userRole).Error; err != nil {
		t.Fatalf("expected user_role to exist: %v", err)
	}
}

// TestPermissionInheritance tests that permissions are correctly linked to roles.
func TestPermissionInheritance(t *testing.T) {
	db := setupRBACDB(t)
	h := NewHandler(&svc.ServiceContext{DB: db})

	// Create role
	role := &model.Role{Name: "Admin", Code: "admin", Status: 1}
	if err := db.Create(role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}

	// Create permissions
	perm1 := &model.Permission{Name: "Read", Code: "resource:read", Resource: "resource", Action: "read", Status: 1}
	perm2 := &model.Permission{Name: "Write", Code: "resource:write", Resource: "resource", Action: "write", Status: 1}
	if err := db.Create(perm1).Error; err != nil {
		t.Fatalf("create perm1: %v", err)
	}
	if err := db.Create(perm2).Error; err != nil {
		t.Fatalf("create perm2: %v", err)
	}

	// Assign permissions to role
	err := db.Transaction(func(tx *gorm.DB) error {
		return h.syncRolePermissionsTx(tx, uint64(role.ID), []string{"resource:read", "resource:write"})
	})
	if err != nil {
		t.Fatalf("assign permissions: %v", err)
	}

	// Verify both permissions are linked
	var count int64
	db.Model(&model.RolePermission{}).Where("role_id = ?", int64(role.ID)).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 permissions, got %d", count)
	}
}

// TestResourcePermissionCheck tests checking permissions for specific resources.
func TestResourcePermissionCheck(t *testing.T) {
	db := setupRBACDB(t)

	// Create user, role, permission
	user := &model.User{Username: "testuser", PasswordHash: "x", Email: "test@example.com", Status: 1}
	db.Create(user)
	role := &model.Role{Name: "Operator", Code: "operator", Status: 1}
	db.Create(role)
	perm := &model.Permission{Name: "Deploy", Code: "deployment:execute", Resource: "deployment", Action: "execute", Status: 1}
	db.Create(perm)

	// Link them
	db.Create(&model.UserRole{UserID: int64(user.ID), RoleID: int64(role.ID)})
	db.Create(&model.RolePermission{RoleID: int64(role.ID), PermissionID: int64(perm.ID)})

	// Check permission exists
	var result int64
	db.Table("user_roles").
		Joins("JOIN role_permissions ON user_roles.role_id = role_permissions.role_id").
		Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
		Where("user_roles.user_id = ? AND permissions.code = ?", int64(user.ID), "deployment:execute").
		Count(&result)

	if result != 1 {
		t.Fatalf("expected user to have deployment:execute permission")
	}
}

// TestPermissionCache tests that role-permission mappings are consistent.
func TestPermissionCache(t *testing.T) {
	db := setupRBACDB(t)

	// Create role with permissions
	role := &model.Role{Name: "Viewer", Code: "viewer", Status: 1}
	db.Create(role)

	for i := 1; i <= 3; i++ {
		perm := &model.Permission{
			Name:   fmt.Sprintf("Perm%d", i),
			Code:   fmt.Sprintf("resource:action%d", i),
			Status: 1,
		}
		db.Create(perm)
		db.Create(&model.RolePermission{RoleID: int64(role.ID), PermissionID: int64(perm.ID)})
	}

	// Verify all permissions are accessible
	var perms []model.Permission
	db.Table("permissions").
		Select("permissions.*").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", int64(role.ID)).
		Find(&perms)

	if len(perms) != 3 {
		t.Fatalf("expected 3 permissions, got %d", len(perms))
	}
}
