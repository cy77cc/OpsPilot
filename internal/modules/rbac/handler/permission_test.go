package handler

import "testing"

func TestAdminPermissionSet_IncludesAIPermissions(t *testing.T) {
	perms := adminPermissionSet()
	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[p] = struct{}{}
	}

	required := []string{
		"ai:approval:read",
		"ai:approval:write",
		"ai:alert:read",
		"ai:alert:write",
	}
	for _, permission := range required {
		if _, ok := set[permission]; !ok {
			t.Fatalf("expected admin permission set to include %q", permission)
		}
	}
}
