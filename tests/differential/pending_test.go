// These are explicit holes in runtime coverage, not passing semantics tests.
package differential

import "testing"

func TestDynamicVariablesAndAliases(t *testing.T) {
	t.Skip("Phase 2: reference evaluator not implemented; see scenarios.json")
}
func TestEvalCallerScopeAndFrozenArtifact(t *testing.T) {
	t.Skip("Phase 2/3: eval overlays and learned executor not implemented; see scenarios.json")
}
func TestReferenceVersusLearned(t *testing.T) {
	t.Skip("Phase 3: no trained artifact; differential execution unavailable")
}
