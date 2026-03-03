package cluster

import (
	"context"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newBootstrapTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:test_bootstrap_logic?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ClusterBootstrapProfile{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewHandler(&svc.ServiceContext{DB: db})
}

func TestResolveAndValidateBootstrapReq_ProfilePrecedence(t *testing.T) {
	h := newBootstrapTestHandler(t)
	profile := model.ClusterBootstrapProfile{
		Name:                 "prod-default",
		VersionChannel:       "stable-1",
		K8sVersion:           "1.28.0",
		RepoMode:             "mirror",
		RepoURL:              "https://mirror.example.com/k8s",
		ImageRepository:      "registry.aliyuncs.com/google_containers",
		EndpointMode:         "vip",
		ControlPlaneEndpoint: "10.0.0.10:6443",
		VIPProvider:          "kube-vip",
		EtcdMode:             "stacked",
	}
	if err := h.svcCtx.DB.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	req := BootstrapPreviewReq{
		Name:                 "cluster-a",
		ProfileID:            &profile.ID,
		ControlPlaneID:       1,
		K8sVersion:           "1.28.0", // request override
		RepoMode:             "online", // request override
		EndpointMode:         "nodeIP", // request override
		ControlPlaneEndpoint: "",
	}

	cfg, issues, _, _ := h.resolveAndValidateBootstrapReq(context.Background(), req, "172.16.1.11")
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %+v", issues)
	}
	if cfg.RepoMode != "online" {
		t.Fatalf("expected request override repo_mode=online, got %s", cfg.RepoMode)
	}
	if cfg.ImageRepository != "registry.aliyuncs.com/google_containers" {
		t.Fatalf("expected profile image_repository, got %s", cfg.ImageRepository)
	}
	if cfg.EndpointMode != "nodeIP" {
		t.Fatalf("expected endpoint_mode=nodeIP, got %s", cfg.EndpointMode)
	}
	if cfg.ControlPlaneEndpoint != "172.16.1.11:6443" {
		t.Fatalf("expected nodeIP endpoint fallback, got %s", cfg.ControlPlaneEndpoint)
	}
}

func TestResolveAndValidateBootstrapReq_CrossFieldValidation(t *testing.T) {
	h := newBootstrapTestHandler(t)
	req := BootstrapPreviewReq{
		Name:           "cluster-b",
		ControlPlaneID: 1,
		K8sVersion:     "1.28.0",
		RepoMode:       "mirror",
		EtcdMode:       "external",
	}

	_, issues, _, _ := h.resolveAndValidateBootstrapReq(context.Background(), req, "172.16.1.11")
	if len(issues) == 0 {
		t.Fatalf("expected validation issues")
	}
	foundRepo := false
	foundEtcd := false
	for _, issue := range issues {
		if issue.Field == "repo_url" && issue.Domain == "repo" {
			foundRepo = true
		}
		if issue.Field == "external_etcd" && issue.Domain == "etcd" {
			foundEtcd = true
		}
	}
	if !foundRepo || !foundEtcd {
		t.Fatalf("expected repo_url and external_etcd issues, got %+v", issues)
	}
}

func TestResolveAndValidateBootstrapReq_BlockedVersion(t *testing.T) {
	h := newBootstrapTestHandler(t)
	req := BootstrapPreviewReq{
		Name:           "cluster-c",
		ControlPlaneID: 1,
		K8sVersion:     "1.99.0",
	}

	_, issues, _, _ := h.resolveAndValidateBootstrapReq(context.Background(), req, "172.16.1.11")
	foundBlocked := false
	for _, issue := range issues {
		if issue.Field == "k8s_version" && issue.Code == "blocked_version" && issue.Domain == "version" {
			foundBlocked = true
			if issue.Remediation == "" {
				t.Fatalf("expected remediation for blocked version")
			}
		}
	}
	if !foundBlocked {
		t.Fatalf("expected blocked version issue, got %+v", issues)
	}
}

func TestResolveAndValidateBootstrapReq_ProfileNotFound(t *testing.T) {
	h := newBootstrapTestHandler(t)
	missing := uint(999)
	req := BootstrapPreviewReq{
		Name:           "cluster-d",
		ProfileID:      &missing,
		ControlPlaneID: 1,
		K8sVersion:     "1.28.0",
	}

	_, issues, _, _ := h.resolveAndValidateBootstrapReq(context.Background(), req, "172.16.1.11")
	if len(issues) != 1 {
		t.Fatalf("expected one issue, got %+v", issues)
	}
	if issues[0].Field != "profile_id" || issues[0].Domain != "profile" {
		t.Fatalf("unexpected issue: %+v", issues[0])
	}
}
