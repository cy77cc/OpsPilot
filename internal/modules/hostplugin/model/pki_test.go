package model

import (
	"testing"
	"time"
)

func TestOpsAgentCA_TableName(t *testing.T) {
	m := OpsAgentCA{}
	if got := m.TableName(); got != "opsagent_ca" {
		t.Fatalf("TableName() = %q, want %q", got, "opsagent_ca")
	}
}

func TestOpsAgentHostCert_TableName(t *testing.T) {
	m := OpsAgentHostCert{}
	if got := m.TableName(); got != "opsagent_host_certificates" {
		t.Fatalf("TableName() = %q, want %q", got, "opsagent_host_certificates")
	}
}

func TestHostPluginPackage_TableName(t *testing.T) {
	m := HostPluginPackage{}
	if got := m.TableName(); got != "host_plugin_packages" {
		t.Fatalf("TableName() = %q, want %q", got, "host_plugin_packages")
	}
}

func TestOpsAgentHostCert_Fields(t *testing.T) {
	now := time.Now().UTC()
	cert := OpsAgentHostCert{
		HostID:       1,
		InstanceID:   5,
		SerialNumber: "abc123",
		CertPEM:      "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
		KeyPEM:       "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
		NotAfter:     now.AddDate(1, 0, 0),
		Revoked:      false,
	}
	if cert.HostID != 1 {
		t.Fatalf("HostID = %d, want 1", cert.HostID)
	}
	if cert.Revoked {
		t.Fatal("Revoked should be false")
	}
}

func TestHostPluginPackage_Fields(t *testing.T) {
	pkg := HostPluginPackage{
		PluginKey:   "opsagent",
		Version:     "v1.0.0",
		Arch:        "amd64",
		Filename:    "opsagent-v1.0.0-linux-amd64.tar.gz",
		StoragePath: "storage/packages/opsagent/v1.0.0/amd64/opsagent-v1.0.0-linux-amd64.tar.gz",
		Checksum:    "abc123",
		SizeBytes:   1024000,
		UploadedBy:  1,
	}
	if pkg.PluginKey != "opsagent" {
		t.Fatalf("PluginKey = %q, want %q", pkg.PluginKey, "opsagent")
	}
	if pkg.SizeBytes != 1024000 {
		t.Fatalf("SizeBytes = %d, want 1024000", pkg.SizeBytes)
	}
}
