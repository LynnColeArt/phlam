// These are explicit holes in runtime coverage, not passing semantics tests.
package differential

import "testing"

func TestDynamicVariablesAndAliases(t *testing.T) {
	t.Skip("Phase 2: integer Counter oracle exists; variable-cell aliases and dynamic variables remain unimplemented")
}
func TestEvalCallerScopeAndFrozenArtifact(t *testing.T) {
	t.Skip("Phase 2/3: eval overlays and learned executor remain unimplemented; see scenarios.json")
}
func TestReferenceVersusLearned(t *testing.T) {
	t.Skip("Phase 3: no trained artifact; differential execution unavailable")
}
