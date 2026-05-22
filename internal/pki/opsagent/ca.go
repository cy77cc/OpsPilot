// Package opsagent implements the OpsAgent CA manager for mTLS certificate lifecycle.
package opsagent

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/cy77cc/OpsPilot/internal/pki"
)

// Store abstracts CA persistence (DB-backed or in-memory for tests).
type Store interface {
	LoadCA() (certPEM, keyPEM []byte, err error)
	SaveCA(certPEM, keyPEM []byte) error
}

// Manager handles OpsAgent CA lifecycle and client certificate issuance.
type Manager struct {
	store   Store
	mu      sync.RWMutex
	caCert  *x509.Certificate
	caKey   *rsa.PrivateKey
	certPEM []byte
}

// NewManager creates a Manager with the given store.
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// EnsureCA loads or generates the OpsAgent CA. Returns the CA cert PEM.
func (m *Manager) EnsureCA() (certPEM, keyPEM []byte, err error) {
	// Fast path: check if already loaded (read lock).
	m.mu.RLock()
	if m.caCert != nil {
		defer m.mu.RUnlock()
		return m.certPEM, nil, nil
	}
	m.mu.RUnlock()

	// Slow path: acquire write lock for initialization.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if m.caCert != nil {
		return m.certPEM, nil, nil
	}

	// Try loading from store.
	storedCert, storedKey, loadErr := m.store.LoadCA()
	if loadErr == nil && len(storedCert) > 0 {
		caCert, err := pki.ParseCert(storedCert)
		if err != nil {
			return nil, nil, fmt.Errorf("opsagent ca: parse stored cert: %w", err)
		}
		caKey, err := pki.ParseRSAKey(storedKey)
		if err != nil {
			return nil, nil, fmt.Errorf("opsagent ca: parse stored key: %w", err)
		}
		m.caCert = caCert
		m.caKey = caKey
		m.certPEM = storedCert
		return storedCert, storedKey, nil
	}

	// Generate new CA.
	spec := pki.CertSpec{
		CommonName: "OpsAgent CA",
		Orgs:       []string{"OpsPilot"},
		IsCA:       true,
		ValidYears: 10,
		KeyUsage:   x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caCert, caKey, newCertPEM, newKeyPEM, err := pki.GenerateCA(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("opsagent ca: generate CA: %w", err)
	}

	if err := m.store.SaveCA(newCertPEM, newKeyPEM); err != nil {
		return nil, nil, fmt.Errorf("opsagent ca: persist CA: %w", err)
	}

	m.caCert = caCert
	m.caKey = caKey
	m.certPEM = newCertPEM
	return newCertPEM, newKeyPEM, nil
}

// IssueClientCert signs a client certificate for an agent instance.
// CN = agentID, SAN includes the host IP. Validity = 1 year.
func (m *Manager) IssueClientCert(agentID string, hostIP net.IP) (certPEM, keyPEM []byte, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.caCert == nil {
		return nil, nil, errors.New("opsagent ca: CA not initialized, call EnsureCA first")
	}

	spec := pki.CertSpec{
		CommonName:  agentID,
		Orgs:        []string{"OpsPilot"},
		IPs:         []net.IP{hostIP},
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certPEM, keyPEM, err = pki.IssueCert(m.caCert, m.caKey, spec)
	if err != nil {
		return nil, nil, fmt.Errorf("opsagent ca: issue cert: %w", err)
	}
	return certPEM, keyPEM, nil
}

// GetCACert returns the CA certificate PEM. Fails if CA not initialized.
func (m *Manager) GetCACert() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.caCert == nil {
		return nil, errors.New("opsagent ca: CA not initialized")
	}
	return m.certPEM, nil
}

// IssueServerCert signs a server certificate for the platform gRPC server.
func (m *Manager) IssueServerCert(dnsNames []string, ips []net.IP) (certPEM, keyPEM []byte, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.caCert == nil {
		return nil, nil, errors.New("opsagent ca: CA not initialized, call EnsureCA first")
	}

	spec := pki.CertSpec{
		CommonName:  "opsagent-grpc-server",
		Orgs:        []string{"OpsPilot"},
		DNSNames:    dnsNames,
		IPs:         ips,
		ValidYears:  1,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certPEM, keyPEM, err = pki.IssueCert(m.caCert, m.caKey, spec)
	if err != nil {
		return nil, nil, fmt.Errorf("opsagent ca: issue server cert: %w", err)
	}
	return certPEM, keyPEM, nil
}

// memoryStore is an in-memory Store for testing.
type memoryStore struct {
	certPEM []byte
	keyPEM  []byte
}

func (s *memoryStore) LoadCA() ([]byte, []byte, error) {
	if len(s.certPEM) == 0 {
		return nil, nil, errors.New("not found")
	}
	return s.certPEM, s.keyPEM, nil
}

func (s *memoryStore) SaveCA(certPEM, keyPEM []byte) error {
	s.certPEM = certPEM
	s.keyPEM = keyPEM
	return nil
}

// NewMemoryManager creates a Manager with an in-memory store for testing.
func NewMemoryManager() *Manager {
	return NewManager(&memoryStore{})
}
