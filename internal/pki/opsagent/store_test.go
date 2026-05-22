package opsagent

import (
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&hostpluginmodel.OpsAgentCA{},
		&hostpluginmodel.OpsAgentHostCert{},
		&hostpluginmodel.HostPluginPackage{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestDBStore_LoadCA_Empty(t *testing.T) {
	db := openTestDB(t)
	store := NewDBStore(db)
	_, _, err := store.LoadCA()
	if err == nil {
		t.Fatal("LoadCA() should fail on empty table")
	}
}

func TestDBStore_SaveCA_And_LoadCA(t *testing.T) {
	db := openTestDB(t)
	store := NewDBStore(db)

	certPEM := []byte("-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----")
	keyPEM := []byte("-----BEGIN RSA PRIVATE KEY-----\ntest-key\n-----END RSA PRIVATE KEY-----")

	if err := store.SaveCA(certPEM, keyPEM); err != nil {
		t.Fatalf("SaveCA() error: %v", err)
	}

	loadedCert, loadedKey, err := store.LoadCA()
	if err != nil {
		t.Fatalf("LoadCA() error: %v", err)
	}
	if string(loadedCert) != string(certPEM) {
		t.Fatalf("cert PEM mismatch")
	}
	if string(loadedKey) != string(keyPEM) {
		t.Fatalf("key PEM mismatch")
	}
}

func TestCertStore_SaveAndGetCert(t *testing.T) {
	db := openTestDB(t)
	cs := NewCertStore(db)

	cert := &hostpluginmodel.OpsAgentHostCert{
		HostID:       1,
		InstanceID:   100,
		SerialNumber: "abc123",
		CertPEM:      "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		KeyPEM:       "-----BEGIN RSA PRIVATE KEY-----\ntest-key\n-----END RSA PRIVATE KEY-----",
	}

	if err := cs.SaveCert(cert); err != nil {
		t.Fatalf("SaveCert() error: %v", err)
	}

	got, err := cs.GetCertByInstance(100)
	if err != nil {
		t.Fatalf("GetCertByInstance() error: %v", err)
	}
	if got.SerialNumber != "abc123" {
		t.Fatalf("serial = %q, want %q", got.SerialNumber, "abc123")
	}
	if got.InstanceID != 100 {
		t.Fatalf("instanceID = %d, want 100", got.InstanceID)
	}
}

func TestCertStore_GetCertByInstance_NotFound(t *testing.T) {
	db := openTestDB(t)
	cs := NewCertStore(db)

	_, err := cs.GetCertByInstance(999)
	if err == nil {
		t.Fatal("GetCertByInstance() should fail for nonexistent instance")
	}
}

func TestCertStore_RevokeCert(t *testing.T) {
	db := openTestDB(t)
	cs := NewCertStore(db)

	cert := &hostpluginmodel.OpsAgentHostCert{
		HostID:       1,
		InstanceID:   200,
		SerialNumber: "rev001",
		CertPEM:      "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		KeyPEM:       "-----BEGIN RSA PRIVATE KEY-----\ntest-key\n-----END RSA PRIVATE KEY-----",
	}

	if err := cs.SaveCert(cert); err != nil {
		t.Fatalf("SaveCert() error: %v", err)
	}

	if err := cs.RevokeCert(200); err != nil {
		t.Fatalf("RevokeCert() error: %v", err)
	}

	// After revocation, GetCertByInstance should not find it.
	_, err := cs.GetCertByInstance(200)
	if err == nil {
		t.Fatal("GetCertByInstance() should fail for revoked cert")
	}
}

func TestCertStore_RevokeCert_NotFound(t *testing.T) {
	db := openTestDB(t)
	cs := NewCertStore(db)

	err := cs.RevokeCert(999)
	if err == nil {
		t.Fatal("RevokeCert() should fail for nonexistent instance")
	}
}
