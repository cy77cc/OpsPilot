package planner

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveClusterReturnsExactAndAmbiguousResults(t *testing.T) {
	db := openSupportTestDB(t, "planner_resolve_cluster")
	seedSupportTestData(t, db)

	tools := NewSupportTools(db)

	exact, err := tools.ResolveCluster(context.Background(), "local")
	if err != nil {
		t.Fatalf("ResolveCluster() error = %v", err)
	}
	if exact.Status != ResolveStatusExact {
		t.Fatalf("exact status = %s, want %s", exact.Status, ResolveStatusExact)
	}
	if exact.Selected == nil || exact.Selected.Name != "local" || exact.Selected.ID != "1" {
		t.Fatalf("exact selected = %#v", exact.Selected)
	}

	ambiguous, err := tools.ResolveCluster(context.Background(), "prod")
	if err != nil {
		t.Fatalf("ResolveCluster() error = %v", err)
	}
	if ambiguous.Status != ResolveStatusAmbiguous {
		t.Fatalf("ambiguous status = %s, want %s", ambiguous.Status, ResolveStatusAmbiguous)
	}
	if len(ambiguous.Candidates) != 2 {
		t.Fatalf("ambiguous candidates = %#v", ambiguous.Candidates)
	}
}

func TestResolveServiceAndHostHandleMissingQueries(t *testing.T) {
	db := openSupportTestDB(t, "planner_resolve_misc")
	seedSupportTestData(t, db)

	tools := NewSupportTools(db)

	service, err := tools.ResolveService(context.Background(), "billing-api")
	if err != nil {
		t.Fatalf("ResolveService() error = %v", err)
	}
	if service.Status != ResolveStatusExact {
		t.Fatalf("service status = %s, want %s", service.Status, ResolveStatusExact)
	}
	if service.Selected == nil || service.Selected.ID != "11" {
		t.Fatalf("service selected = %#v", service.Selected)
	}

	host, err := tools.ResolveHost(context.Background(), "10.0.0.9")
	if err != nil {
		t.Fatalf("ResolveHost() error = %v", err)
	}
	if host.Status != ResolveStatusExact {
		t.Fatalf("host status = %s, want %s", host.Status, ResolveStatusExact)
	}
	if host.Selected == nil || host.Selected.Name != "ops-bastion" {
		t.Fatalf("host selected = %#v", host.Selected)
	}

	missing, err := tools.ResolveCluster(context.Background(), "")
	if err != nil {
		t.Fatalf("ResolveCluster() error = %v", err)
	}
	if missing.Status != ResolveStatusMissing {
		t.Fatalf("missing status = %s, want %s", missing.Status, ResolveStatusMissing)
	}
}

func openSupportTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Cluster{}, &model.Service{}, &model.Node{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedSupportTestData(t *testing.T, db *gorm.DB) {
	t.Helper()

	clusters := []model.Cluster{
		{ID: 1, Name: "local", Status: "ready", Type: "kubernetes", EnvType: "development"},
		{ID: 2, Name: "prod-a", Status: "ready", Type: "kubernetes", EnvType: "production"},
		{ID: 3, Name: "prod-b", Status: "ready", Type: "kubernetes", EnvType: "production"},
	}
	services := []model.Service{
		{ID: 11, Name: "billing-api", Type: "stateless", Image: "billing:v1"},
	}
	nodes := []model.Node{
		{ID: 21, Name: "ops-bastion", Hostname: "ops-bastion", IP: "10.0.0.9", Status: "online"},
	}

	if err := db.Create(&clusters).Error; err != nil {
		t.Fatalf("seed clusters: %v", err)
	}
	if err := db.Create(&services).Error; err != nil {
		t.Fatalf("seed services: %v", err)
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}
}
