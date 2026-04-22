package user

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeDeptCache struct {
	values map[string]string
}

func newFakeDeptCache() *fakeDeptCache {
	return &fakeDeptCache{values: make(map[string]string)}
}

func (f *fakeDeptCache) SetEx(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	return f.Set(context.Background(), key, value, 0)
}

func (f *fakeDeptCache) Del(_ context.Context, keys ...string) *redis.IntCmd {
	var removed int64
	for _, key := range keys {
		if _, ok := f.values[key]; ok {
			delete(f.values, key)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func (f *fakeDeptCache) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (f *fakeDeptCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	switch typed := value.(type) {
	case string:
		f.values[key] = typed
	default:
		buf, err := json.Marshal(typed)
		if err != nil {
			return redis.NewStatusResult("", err)
		}
		f.values[key] = string(buf)
	}
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeDeptCache) TTL(_ context.Context, key string) *redis.DurationCmd {
	if _, ok := f.values[key]; !ok {
		return redis.NewDurationResult(-1, redis.Nil)
	}
	return redis.NewDurationResult(time.Minute, nil)
}

func (f *fakeDeptCache) Expire(_ context.Context, key string, _ time.Duration) *redis.BoolCmd {
	if _, ok := f.values[key]; !ok {
		return redis.NewBoolResult(false, redis.Nil)
	}
	return redis.NewBoolResult(true, nil)
}

func setupDeptTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Department{}); err != nil {
		t.Fatalf("auto migrate dept: %v", err)
	}
	return db
}

func TestDepartmentDAO_Create(t *testing.T) {
	db := setupDeptTestDB(t)
	cache := newFakeDeptCache()
	dao := &DepartmentDAO{db: db, rdb: cache}
	ctx := context.Background()

	dept := &model.Department{
		Name:     "Test Dept",
		ParentID: 0,
		Status:   1,
	}

	if err := dao.Create(ctx, dept); err != nil {
		t.Fatalf("create dept: %v", err)
	}

	if dept.ID == 0 {
		t.Fatal("expected dept id to be set")
	}

	key := constants.DeptIdKey + "1"
	if _, ok := cache.values[key]; !ok {
		t.Fatalf("expected cache key %s to be set", key)
	}
}

func TestDepartmentDAO_Update(t *testing.T) {
	db := setupDeptTestDB(t)
	cache := newFakeDeptCache()
	dao := &DepartmentDAO{db: db, rdb: cache}
	ctx := context.Background()

	dept := &model.Department{
		Name:   "Old Name",
		Status: 1,
	}
	db.Create(dept)
	key := constants.DeptIdKey + "1"
	cache.values[key] = `{"name":"Old Name"}`

	dept.Name = "New Name"
	if err := dao.Update(ctx, dept); err != nil {
		t.Fatalf("update dept: %v", err)
	}

	if _, ok := cache.values[key]; ok {
		t.Fatalf("expected cache key %s to be invalidated", key)
	}

	var saved model.Department
	db.First(&saved, dept.ID)
	if saved.Name != "New Name" {
		t.Fatalf("expected name to be New Name, got %s", saved.Name)
	}
}

func TestDepartmentDAO_Delete(t *testing.T) {
	db := setupDeptTestDB(t)
	cache := newFakeDeptCache()
	dao := &DepartmentDAO{db: db, rdb: cache}
	ctx := context.Background()

	dept := &model.Department{Name: "To Delete"}
	db.Create(dept)
	key := constants.DeptIdKey + "1"
	cache.values[key] = `{"name":"To Delete"}`

	if err := dao.Delete(ctx, dept.ID); err != nil {
		t.Fatalf("delete dept: %v", err)
	}

	if _, ok := cache.values[key]; ok {
		t.Fatal("expected cache to be invalidated")
	}

	var count int64
	db.Model(&model.Department{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 depts, got %d", count)
	}
}

func TestDepartmentDAO_FindByID(t *testing.T) {
	db := setupDeptTestDB(t)
	cache := newFakeDeptCache()
	dao := &DepartmentDAO{db: db, rdb: cache}
	ctx := context.Background()

	dept := &model.Department{ID: 1, Name: "Found"}
	db.Create(dept)

	// Test from DB
	got, err := dao.FindByID(ctx, 1)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got.Name != "Found" {
		t.Fatalf("expected Found, got %s", got.Name)
	}

	key := constants.DeptIdKey + "1"
	if _, ok := cache.values[key]; !ok {
		t.Fatal("expected cache to be set after find")
	}

	// Test from Cache
	cache.values[key] = `{"id":1, "name":"From Cache"}`
	got, err = dao.FindByID(ctx, 1)
	if err != nil {
		t.Fatalf("find by id (cache): %v", err)
	}
	if got.Name != "From Cache" {
		t.Fatalf("expected From Cache, got %s", got.Name)
	}
}

func TestDepartmentDAO_FindAll(t *testing.T) {
	db := setupDeptTestDB(t)
	dao := &DepartmentDAO{db: db}
	ctx := context.Background()

	db.Create(&model.Department{Name: "Dept 1"})
	db.Create(&model.Department{Name: "Dept 2"})

	depts, err := dao.FindAll(ctx)
	if err != nil {
		t.Fatalf("find all: %v", err)
	}
	if len(depts) != 2 {
		t.Fatalf("expected 2 depts, got %d", len(depts))
	}
}
