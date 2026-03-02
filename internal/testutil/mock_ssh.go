package testutil

import (
	"sync"
	"testing"
)

// MockSSHClient is a mock implementation of SSH client for testing.
type MockSSHClient struct {
	mu sync.Mutex

	// ExecFunc allows customizing the Exec behavior
	ExecFunc func(cmd string) (stdout, stderr string, err error)

	// UploadFunc allows customizing the Upload behavior
	UploadFunc func(remotePath string, content []byte) error

	// recorded calls
	calls []string
}

// NewMockSSHClient creates a new MockSSHClient.
func NewMockSSHClient() *MockSSHClient {
	return &MockSSHClient{
		calls: make([]string, 0),
	}
}

// Exec executes a command on the remote host (mock implementation).
func (m *MockSSHClient) Exec(cmd string) (stdout, stderr string, err error) {
	m.mu.Lock()
	m.calls = append(m.calls, cmd)
	m.mu.Unlock()

	if m.ExecFunc != nil {
		return m.ExecFunc(cmd)
	}
	// Default: return success with empty output
	return "", "", nil
}

// Upload uploads a file to the remote host (mock implementation).
func (m *MockSSHClient) Upload(remotePath string, content []byte) error {
	m.mu.Lock()
	m.calls = append(m.calls, "UPLOAD:"+remotePath)
	m.mu.Unlock()

	if m.UploadFunc != nil {
		return m.UploadFunc(remotePath, content)
	}
	return nil
}

// GetCalls returns all recorded command calls.
func (m *MockSSHClient) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.calls))
	copy(result, m.calls)
	return result
}

// Reset clears all recorded calls.
func (m *MockSSHClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]string, 0)
}

// AssertCalled asserts that a specific command was called.
func (m *MockSSHClient) AssertCalled(t *testing.T, expectedCmd string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.calls {
		if call == expectedCmd {
			return
		}
	}
	t.Fatalf("expected command %q to be called, but it was not. Calls: %v", expectedCmd, m.calls)
}

// AssertCalledContains asserts that a command containing the expected substring was called.
func (m *MockSSHClient) AssertCalledContains(t *testing.T, substr string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, call := range m.calls {
		if containsString(call, substr) {
			return
		}
	}
	t.Fatalf("expected a command containing %q to be called, but none was. Calls: %v", substr, m.calls)
}

// AssertCallCount asserts the number of calls made.
func (m *MockSSHClient) AssertCallCount(t *testing.T, expected int) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.calls) != expected {
		t.Fatalf("expected %d calls, got %d. Calls: %v", expected, len(m.calls), m.calls)
	}
}

// MockSFTPClient is a mock implementation of SFTP client for testing.
type MockSFTPClient struct {
	mu sync.Mutex

	// ReadFileFunc allows customizing ReadFile behavior
	ReadFileFunc func(path string) ([]byte, error)

	// WriteFileFunc allows customizing WriteFile behavior
	WriteFileFunc func(path string, data []byte) error

	// MkdirFunc allows customizing Mkdir behavior
	MkdirFunc func(path string) error

	// RemoveFunc allows customizing Remove behavior
	RemoveFunc func(path string) error

	// ListFunc allows customizing List behavior
	ListFunc func(path string) ([]string, error)

	// recorded operations
	operations []sFTPOperation
}

type sFTPOperation struct {
	op   string
	path string
	data []byte
}

// NewMockSFTPClient creates a new MockSFTPClient.
func NewMockSFTPClient() *MockSFTPClient {
	return &MockSFTPClient{
		operations: make([]sFTPOperation, 0),
	}
}

// ReadFile reads a file from the remote host (mock implementation).
func (m *MockSFTPClient) ReadFile(path string) ([]byte, error) {
	m.mu.Lock()
	m.operations = append(m.operations, sFTPOperation{op: "READ", path: path})
	m.mu.Unlock()

	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(path)
	}
	return []byte{}, nil
}

// WriteFile writes a file to the remote host (mock implementation).
func (m *MockSFTPClient) WriteFile(path string, data []byte) error {
	m.mu.Lock()
	m.operations = append(m.operations, sFTPOperation{op: "WRITE", path: path, data: data})
	m.mu.Unlock()

	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(path, data)
	}
	return nil
}

// Mkdir creates a directory on the remote host (mock implementation).
func (m *MockSFTPClient) Mkdir(path string) error {
	m.mu.Lock()
	m.operations = append(m.operations, sFTPOperation{op: "MKDIR", path: path})
	m.mu.Unlock()

	if m.MkdirFunc != nil {
		return m.MkdirFunc(path)
	}
	return nil
}

// Remove removes a file on the remote host (mock implementation).
func (m *MockSFTPClient) Remove(path string) error {
	m.mu.Lock()
	m.operations = append(m.operations, sFTPOperation{op: "REMOVE", path: path})
	m.mu.Unlock()

	if m.RemoveFunc != nil {
		return m.RemoveFunc(path)
	}
	return nil
}

// List lists files in a directory on the remote host (mock implementation).
func (m *MockSFTPClient) List(path string) ([]string, error) {
	m.mu.Lock()
	m.operations = append(m.operations, sFTPOperation{op: "LIST", path: path})
	m.mu.Unlock()

	if m.ListFunc != nil {
		return m.ListFunc(path)
	}
	return []string{}, nil
}

// GetOperations returns all recorded operations.
func (m *MockSFTPClient) GetOperations() []sFTPOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]sFTPOperation, len(m.operations))
	copy(result, m.operations)
	return result
}

// Reset clears all recorded operations.
func (m *MockSFTPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations = make([]sFTPOperation, 0)
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
