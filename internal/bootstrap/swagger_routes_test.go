package bootstrap

import (
	"os"
	"strings"
	"testing"
)

func TestAlertHealRoutesPresentInSwagger(t *testing.T) {
	raw, err := os.ReadFile("../../docs/swagger/swagger.yaml")
	if err != nil {
		t.Fatalf("read swagger: %v", err)
	}

	required := []string{
		"/ai/alerts/webhook:",
		"/ai/alert-heal/jobs:",
		"/ai/alert-heal/jobs/{id}:",
		"/ai/alert-heal/jobs/{id}/retry:",
		"/ai/approvals/pending/global:",
	}
	for _, route := range required {
		if !strings.Contains(string(raw), route) {
			t.Fatalf("missing %s route in swagger", route)
		}
	}
}
