package tools

import (
	"context"
	"testing"

	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestADKToolAdapter_AdaptAll(t *testing.T) {
	registry := NewRegistry(common.PlatformDeps{})
	adapter := NewADKToolAdapter(registry, nil)

	tools := adapter.AdaptAll()
	assert.NotEmpty(t, tools, "should have tools")

	for _, current := range tools {
		info, err := current.Info(context.Background())
		require.NoError(t, err)
		assert.NotEmpty(t, info.Name)
		assert.NotEmpty(t, info.Desc)
	}
}

func TestADKToolAdapter_MutatingToolNeedsGate(t *testing.T) {
	registry := NewRegistry(common.PlatformDeps{})
	adapter := NewADKToolAdapter(registry, nil)

	spec, ok := registry.Get("service_deploy_apply")
	require.True(t, ok, "service_deploy_apply should exist")

	adapted := adapter.adaptTool(spec)
	assert.NotNil(t, adapted)

	_, wrapped := adapted.(*approvaltools.Gate)
	assert.True(t, wrapped, "mutating tool should be wrapped by approval gate")

	info, err := adapted.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "service_deploy_apply", info.Name)
}
