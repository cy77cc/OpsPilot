package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstruction_IsStableAcrossRuntimeContexts(t *testing.T) {
	first := BuildInstruction(RuntimeContext{Scene: "deployment:hosts", ProjectID: "1"})
	second := BuildInstruction(RuntimeContext{Scene: "service:list", ProjectID: "9"})

	assert.Equal(t, first, second)
}

func TestInstruction_DescribesSceneBiasWithoutHardRestriction(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})

	assert.Contains(t, result, "scene")
	assert.Contains(t, result, "优先")
	assert.Contains(t, result, "不是硬性限制")
}

func TestInstruction_CoversToolDomainsAndCanonicalSceneRules(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})

	assert.Contains(t, result, "host")
	assert.Contains(t, result, "deployment")
	assert.Contains(t, result, "service")
	assert.Contains(t, result, "kubernetes")
	assert.Contains(t, result, "monitor")
	assert.Contains(t, result, "governance")
	assert.Contains(t, result, "deployment:*")
	assert.Contains(t, result, "service:*")
	assert.Contains(t, result, "host:*")
	assert.Contains(t, result, "k8s:*")
}

func TestInstruction_RequiresReadonlyFirstAndApprovalForMutations(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})

	assert.Contains(t, result, "只读")
	assert.Contains(t, result, "审批")
}
