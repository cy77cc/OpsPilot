package testutil

import (
	"testing"

	"gorm.io/gorm"
)

// AssertEqual asserts that two values are equal.
func AssertEqual[T comparable](t *testing.T, expected, actual T, msgAndArgs ...interface{}) {
	t.Helper()
	if expected != actual {
		t.Fatalf("assertion failed: expected %v, got %v. %v", expected, actual, msgAndArgs)
	}
}

// AssertNotEqual asserts that two values are not equal.
func AssertNotEqual[T comparable](t *testing.T, expected, actual T, msgAndArgs ...interface{}) {
	t.Helper()
	if expected == actual {
		t.Fatalf("assertion failed: expected %v to not equal %v. %v", expected, actual, msgAndArgs)
	}
}

// AssertTrue asserts that a condition is true.
func AssertTrue(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if !condition {
		t.Fatalf("assertion failed: expected true, got false. %v", msgAndArgs)
	}
}

// AssertFalse asserts that a condition is false.
func AssertFalse(t *testing.T, condition bool, msgAndArgs ...interface{}) {
	t.Helper()
	if condition {
		t.Fatalf("assertion failed: expected false, got true. %v", msgAndArgs)
	}
}

// AssertNil asserts that a value is nil.
func AssertNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value != nil {
		t.Fatalf("assertion failed: expected nil, got %v. %v", value, msgAndArgs)
	}
}

// AssertNotNil asserts that a value is not nil.
func AssertNotNil(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	if value == nil {
		t.Fatalf("assertion failed: expected non-nil value. %v", msgAndArgs)
	}
}

// AssertError asserts that an error occurred.
func AssertError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("assertion failed: expected an error, got nil. %v", msgAndArgs)
	}
}

// AssertNoError asserts that no error occurred.
func AssertNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("assertion failed: expected no error, got %v. %v", err, msgAndArgs)
	}
}

// AssertErrorContains asserts that an error message contains a substring.
func AssertErrorContains(t *testing.T, err error, substr string, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("assertion failed: expected an error, got nil. %v", msgAndArgs)
	}
	if !containsString(err.Error(), substr) {
		t.Fatalf("assertion failed: expected error to contain %q, got %q. %v", substr, err.Error(), msgAndArgs)
	}
}

// AssertLen asserts the length of a slice, array, map, or string.
func AssertLen[T any](t *testing.T, length int, collection interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	actualLen := 0
	switch v := collection.(type) {
	case []T:
		actualLen = len(v)
	case string:
		actualLen = len(v)
	case map[string]T:
		actualLen = len(v)
	default:
		t.Fatalf("assertion failed: unsupported collection type %T. %v", collection, msgAndArgs)
	}
	if actualLen != length {
		t.Fatalf("assertion failed: expected length %d, got %d. %v", length, actualLen, msgAndArgs)
	}
}

// AssertContains asserts that a slice contains an element.
func AssertContains[T comparable](t *testing.T, slice []T, element T, msgAndArgs ...interface{}) {
	t.Helper()
	for _, item := range slice {
		if item == element {
			return
		}
	}
	t.Fatalf("assertion failed: slice does not contain element %v. %v", element, msgAndArgs)
}

// AssertNotContains asserts that a slice does not contain an element.
func AssertNotContains[T comparable](t *testing.T, slice []T, element T, msgAndArgs ...interface{}) {
	t.Helper()
	for _, item := range slice {
		if item == element {
			t.Fatalf("assertion failed: slice should not contain element %v. %v", element, msgAndArgs)
		}
	}
}

// AssertJSONEqual asserts that two JSON strings are equivalent.
func AssertJSONEqual(t *testing.T, expected, actual string, msgAndArgs ...interface{}) {
	t.Helper()
	// Normalize JSON for comparison
	expectedNormalized := normalizeJSON(expected)
	actualNormalized := normalizeJSON(actual)
	if expectedNormalized != actualNormalized {
		t.Fatalf("assertion failed: JSON not equal.\nExpected: %s\nActual: %s\n%v", expectedNormalized, actualNormalized, msgAndArgs)
	}
}

// AssertDBRecordExists asserts that a record exists in the database.
func AssertDBRecordExists(t *testing.T, db *gorm.DB, model interface{}, query string, args ...interface{}) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count == 0 {
		t.Fatalf("assertion failed: expected record to exist")
	}
}

// AssertDBRecordNotExists asserts that a record does not exist in the database.
func AssertDBRecordNotExists(t *testing.T, db *gorm.DB, model interface{}, query string, args ...interface{}) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count > 0 {
		t.Fatalf("assertion failed: expected record to not exist")
	}
}

// AssertDBRecordCount asserts the number of records matching a query.
func AssertDBRecordCount(t *testing.T, db *gorm.DB, expected int64, model interface{}, query string, args ...interface{}) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("failed to query database: %v", err)
	}
	if count != expected {
		t.Fatalf("assertion failed: expected %d records, got %d", expected, count)
	}
}

// RequireNoError fails the test immediately if an error occurred.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err != nil {
		t.Fatalf("required no error, but got: %v. %v", err, msgAndArgs)
	}
}

// RequireError fails the test immediately if no error occurred.
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	if err == nil {
		t.Fatalf("required an error, but got nil. %v", msgAndArgs)
	}
}

// Helper functions

func normalizeJSON(s string) string {
	// Simple normalization: trim whitespace
	// In a real implementation, you might use encoding/json to parse and re-encode
	result := make([]byte, 0, len(s))
	inString := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
		}
		if inString || (c != ' ' && c != '\n' && c != '\t' && c != '\r') {
			result = append(result, c)
		}
	}
	return string(result)
}
