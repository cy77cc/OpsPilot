package service

import (
	"context"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVisibilityLogic(t *testing.T) *Logic {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:service_visibility?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Service{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewLogic(&svc.ServiceContext{DB: db})
}

func TestVisibilityPermissionAndUpdate(t *testing.T) {
	logic := newVisibilityLogic(t)
	ctx := context.Background()

	svcRow := &model.Service{
		ProjectID:    1,
		TeamID:       10,
		OwnerUserID:  99,
		Name:         "svc-a",
		Type:         "stateless",
		Image:        "nginx:latest",
		ServiceKind:  "business",
		Visibility:   "team-granted",
		GrantedTeams: `[20,30]`,
	}
	if err := logic.svcCtx.DB.Create(svcRow).Error; err != nil {
		t.Fatalf("create service: %v", err)
	}

	if !logic.CheckViewPermission(svcRow, 200, 20, false) {
		t.Fatalf("expected granted team member can view")
	}
	if logic.CheckViewPermission(svcRow, 200, 21, false) {
		t.Fatalf("expected unauthorized team cannot view")
	}
	if logic.CheckEditPermission(svcRow, 200, 20, false) {
		t.Fatalf("expected granted team cannot edit")
	}

	updated, err := logic.UpdateVisibility(ctx, svcRow.ID, VisibilityUpdateReq{Visibility: "public"})
	if err != nil {
		t.Fatalf("UpdateVisibility error: %v", err)
	}
	if updated.Visibility != "public" {
		t.Fatalf("expected public visibility, got %s", updated.Visibility)
	}

	updated, err = logic.UpdateGrantedTeams(ctx, svcRow.ID, GrantTeamsReq{GrantedTeams: []uint{40, 41}})
	if err != nil {
		t.Fatalf("UpdateGrantedTeams error: %v", err)
	}
	if len(updated.GrantedTeams) != 2 || updated.GrantedTeams[0] != 40 || updated.GrantedTeams[1] != 41 {
		t.Fatalf("unexpected granted teams: %#v", updated.GrantedTeams)
	}
}
