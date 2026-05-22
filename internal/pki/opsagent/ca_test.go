package opsagent

import (
	"net"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/pki"
)

func TestManager_EnsureCA_GeneratesWhenMissing(t *testing.T) {
	m := NewMemoryManager()
	certPEM, _, err := m.EnsureCA()
	if err != nil {
		t.Fatalf("EnsureCA() error: %v", err)
	}
	if len(certPEM) == 0 {
		t.Fatal("EnsureCA() returned empty cert PEM")
	}

	cert, err := pki.ParseCert(certPEM)
	if err != nil {
		t.Fatalf("ParseCert() error: %v", err)
	}
	if !cert.IsCA {
		t.Fatal("generated cert is not a CA")
	}
	if cert.Subject.CommonName != "OpsAgent CA" {
		t.Fatalf("CN = %q, want %q", cert.Subject.CommonName, "OpsAgent CA")
	}
}

func TestManager_EnsureCA_ReturnsExisting(t *testing.T) {
	m := NewMemoryManager()
	cert1, _, err := m.EnsureCA()
	if err != nil {
		t.Fatalf("first EnsureCA() error: %v", err)
	}
	cert2, _, err := m.EnsureCA()
	if err != nil {
		t.Fatalf("second EnsureCA() error: %v", err)
	}
	if string(cert1) != string(cert2) {
		t.Fatal("EnsureCA() returned different certs on second call")
	}
}

func TestManager_IssueClientCert(t *testing.T) {
	m := NewMemoryManager()
	_, _, err := m.EnsureCA()
	if err != nil {
		t.Fatalf("EnsureCA() error: %v", err)
	}

	certPEM, keyPEM, err := m.IssueClientCert("opsagent-host-1-instance-5", net.ParseIP("10.0.0.21"))
	if err != nil {
		t.Fatalf("IssueClientCert() error: %v", err)
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		t.Fatal("IssueClientCert() returned empty PEM")
	}

	cert, err := pki.ParseCert(certPEM)
	if err != nil {
		t.Fatalf("ParseCert() error: %v", err)
	}
	if cert.Subject.CommonName != "opsagent-host-1-instance-5" {
		t.Fatalf("CN = %q, want %q", cert.Subject.CommonName, "opsagent-host-1-instance-5")
	}
	if len(cert.IPAddresses) == 0 || !cert.IPAddresses[0].Equal(net.ParseIP("10.0.0.21")) {
		t.Fatalf("SAN IP = %v, want 10.0.0.21", cert.IPAddresses)
	}
	if cert.SerialNumber == nil {
		t.Fatal("serial number is nil")
	}
}

func TestManager_IssueClientCert_RequiresCA(t *testing.T) {
	m := NewMemoryManager()
	_, _, err := m.IssueClientCert("test", net.ParseIP("10.0.0.1"))
	if err == nil {
		t.Fatal("IssueClientCert() should fail without CA")
	}
}

func TestManager_GetCACert(t *testing.T) {
	m := NewMemoryManager()
	_, _, _ = m.EnsureCA()
	certPEM, err := m.GetCACert()
	if err != nil {
		t.Fatalf("GetCACert() error: %v", err)
	}
	if len(certPEM) == 0 {
		t.Fatal("GetCACert() returned empty")
	}
}

func TestManager_GetCACert_NotInitialized(t *testing.T) {
	m := NewMemoryManager()
	_, err := m.GetCACert()
	if err == nil {
		t.Fatal("GetCACert() should fail when CA not initialized")
	}
}
