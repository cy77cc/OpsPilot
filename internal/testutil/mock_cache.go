package testutil

import (
	"context"
	"sync"
	"time"
)

// MockWhitelistDAO provides a mock implementation of whitelist operations for testing.
// It uses an in-memory map to simulate Redis whitelist behavior.
type MockWhitelistDAO struct {
	mu        sync.RWMutex
	whitelist map[string]time.Time
}

// NewMockWhitelistDAO creates a new MockWhitelistDAO.
func NewMockWhitelistDAO() *MockWhitelistDAO {
	return &MockWhitelistDAO{
		whitelist: make(map[string]time.Time),
	}
}

// AddToWhitelist adds a token to the whitelist with an expiration time.
func (m *MockWhitelistDAO) AddToWhitelist(ctx context.Context, token string, exp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist[token] = exp
	return nil
}

// DeleteToken removes a token from the whitelist.
func (m *MockWhitelistDAO) DeleteToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.whitelist, token)
	return nil
}

// IsWhitelisted checks if a token exists in the whitelist and has not expired.
func (m *MockWhitelistDAO) IsWhitelisted(ctx context.Context, token string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exp, exists := m.whitelist[token]
	if !exists {
		return false, nil
	}
	// Check if token has expired
	if time.Now().After(exp) {
		return false, nil
	}
	return true, nil
}

// Clear removes all tokens from the whitelist (useful for test cleanup).
func (m *MockWhitelistDAO) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist = make(map[string]time.Time)
}

// Count returns the number of tokens in the whitelist (useful for assertions).
func (m *MockWhitelistDAO) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.whitelist)
}

// MockUserDAO provides a mock implementation of user DAO for testing.
// It wraps the IntegrationSuite's database for user operations.
type MockUserDAO struct {
	suite *IntegrationSuite
}

// NewMockUserDAO creates a new MockUserDAO.
func NewMockUserDAO(suite *IntegrationSuite) *MockUserDAO {
	return &MockUserDAO{suite: suite}
}

// The actual UserDAO methods will be called through the IntegrationSuite's DB
// This is a placeholder for future extension if needed.
