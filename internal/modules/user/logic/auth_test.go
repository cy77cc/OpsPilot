package logic

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, *svc.ServiceContext) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Auto Migrate
	err = db.AutoMigrate(
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.Permission{},
		&model.RolePermission{},
		&model.Department{},
		&model.DepartmentMember{},
		&model.DepartmentRole{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	svcCtx := &svc.ServiceContext{
		DB: db,
	}
	return db, svcCtx
}

func TestLoadRolesAndPermissions(t *testing.T) {
	db, svcCtx := setupTestDB(t)
	l := NewUserLogic(svcCtx)
	ctx := context.Background()

	// 1. Setup Test Data
	// Roles
	adminRole := model.Role{Name: "Admin", Code: "admin"}
	userRole := model.Role{Name: "User", Code: "user"}
	deptRole := model.Role{Name: "DeptRole", Code: "dept_role"}
	db.Create(&adminRole)
	db.Create(&userRole)
	db.Create(&deptRole)

	// Permissions
	p1 := model.Permission{Name: "P1", Code: "p1"}
	p2 := model.Permission{Name: "P2", Code: "p2"}
	db.Create(&p1)
	db.Create(&p2)

	// Role Permissions
	db.Create(&model.RolePermission{RoleID: int64(userRole.ID), PermissionID: int64(p1.ID)})
	db.Create(&model.RolePermission{RoleID: int64(deptRole.ID), PermissionID: int64(p2.ID)})

	// User
	user := model.User{Username: "testuser", PasswordHash: "xxx"}
	db.Create(&user)

	// User Direct Role
	db.Create(&model.UserRole{UserID: int64(user.ID), RoleID: int64(userRole.ID)})

	// Department
	dept := model.Department{Name: "TestDept"}
	db.Create(&dept)

	// Department Member
	db.Create(&model.DepartmentMember{UserID: int64(user.ID), DeptID: int64(dept.ID)})

	// Department Role
	db.Create(&model.DepartmentRole{DeptID: int64(dept.ID), RoleID: int64(deptRole.ID)})

	t.Run("Hybrid Loading", func(t *testing.T) {
		roles, permissions, err := l.loadRolesAndPermissions(ctx, uint64(user.ID))
		assert.NoError(t, err)

		// Should have both userRole and deptRole
		assert.Contains(t, roles, "user")
		assert.Contains(t, roles, "dept_role")
		assert.Len(t, roles, 2)

		// Should have both p1 and p2
		assert.Contains(t, permissions, "p1")
		assert.Contains(t, permissions, "p2")
		assert.Len(t, permissions, 2)
	})

	t.Run("Admin Inheritance", func(t *testing.T) {
		// Add admin role to dept
		db.Create(&model.DepartmentRole{DeptID: int64(dept.ID), RoleID: int64(adminRole.ID)})
		
		roles, permissions, err := l.loadRolesAndPermissions(ctx, uint64(user.ID))
		assert.NoError(t, err)
		assert.Contains(t, roles, "admin")
		assert.Contains(t, permissions, "*:*")
	})
}
