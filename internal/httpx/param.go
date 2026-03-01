package httpx

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// UIDFromCtx extracts the user ID from gin context (set by JWT middleware).
// Returns 0 if not present or not convertible.
func UIDFromCtx(c *gin.Context) uint64 {
	v, ok := c.Get("uid")
	if !ok {
		return 0
	}
	return ToUint64(v)
}

// ToUint64 converts common numeric types to uint64.
func ToUint64(v any) uint64 {
	switch x := v.(type) {
	case uint:
		return uint64(x)
	case uint64:
		return x
	case int:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case int64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case float64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	default:
		return 0
	}
}

// UintFromParam parses a uint from a gin route path parameter.
// Returns 0 if the parameter is missing or not a valid number.
func UintFromParam(c *gin.Context, key string) uint {
	v, _ := strconv.ParseUint(c.Param(key), 10, 64)
	return uint(v)
}

// UintFromQuery parses a uint from a gin query string parameter.
// Returns 0 if the parameter is missing or not a valid number.
func UintFromQuery(c *gin.Context, key string) uint {
	v, _ := strconv.ParseUint(strings.TrimSpace(c.Query(key)), 10, 64)
	return uint(v)
}
