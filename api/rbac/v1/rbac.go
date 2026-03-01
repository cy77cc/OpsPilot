package v1

// CheckReq is the request body for the permission check endpoint (POST /rbac/check).
// It verifies whether the currently authenticated user holds a specific resource action.
type CheckReq struct {
	Resource string `json:"Resource"`
	Action   string `json:"Action"`
}

// CreateUserReq is the request body for creating a new user (POST /rbac/users).
type CreateUserReq struct {
	Username string   `json:"username" binding:"required"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Password string   `json:"password" binding:"required"`
	Roles    []string `json:"roles"`
	Status   string   `json:"status"` // "active" or "disabled"
}

// UpdateUserReq is the request body for updating an existing user (PUT /rbac/users/:id).
// All fields are optional; only non-nil fields are applied.
type UpdateUserReq struct {
	Name     *string  `json:"name"`
	Email    *string  `json:"email"`
	Password *string  `json:"password"`
	Roles    []string `json:"roles"`
	Status   *string  `json:"status"` // "active" or "disabled"
}

// CreateRoleReq is the request body for creating a new role (POST /rbac/roles).
type CreateRoleReq struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleReq is the request body for updating an existing role (PUT /rbac/roles/:id).
// All fields are optional; only non-nil fields are applied.
type UpdateRoleReq struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"`
}

// MigrationEventReq is the request body for recording a frontend migration event
// (POST /rbac/migration-events).
type MigrationEventReq struct {
	EventType  string `json:"eventType" binding:"required"`
	FromPath   string `json:"fromPath"`
	ToPath     string `json:"toPath"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	DurationMs int64  `json:"durationMs"`
}
