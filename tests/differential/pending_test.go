// Dynamic language semantics remain pending; bounded learned Counter execution
// now has an actual frozen-artifact differential test below.
package differential

import (
	"encoding/json"
	"github.com/LynnColeArt/phlam/runtime/learned"
	"github.com/LynnColeArt/phlam/runtime/reference"
	"github.com/LynnColeArt/phlam/semantics"
	"os"
	"testing"
)

func TestDynamicVariablesAndAliases(t *testing.T) {
	t.Skip("Phase 2: integer Counter oracle exists; variable-cell aliases and dynamic variables remain unimplemented")
}
func TestEvalCallerScopeAndFrozenArtifact(t *testing.T) {
	t.Skip("Phase 2/3: eval overlays and learned executor remain unimplemented; see scenarios.json")
}
func TestReferenceVersusLearnedBoundedCounter(t *testing.T) {
	const folder = "../../experiments/counter/run-0001/"
	key, err := learned.PublicKey(folder + "trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	executor, err := learned.Load(folder+"artifact.json", key)
	if err != nil {
		t.Fatal(err)
	}
	artifactBytes, err := os.ReadFile(folder + "artifact.json")
	if err != nil {
		t.Fatal(err)
	}
	var envelope learned.Envelope
	if err := json.Unmarshal(artifactBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	var artifact learned.Artifact
	if err := json.Unmarshal(envelope.Payload, &artifact); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(folder + "splits.json")
	if err != nil {
		t.Fatal(err)
	}
	if learned.Hash(data) != artifact.SplitsHash {
		t.Fatal("split lineage mismatch")
	}
	training, err := os.ReadFile(folder + "training.json")
	if err != nil || learned.Hash(training) != artifact.TrainingHash {
		t.Fatal("training lineage mismatch", err)
	}
	var cases []learned.Case
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../examples/counter/counter.phlam")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, c := range cases {
		if c.Split == "train" {
			continue
		}
		oracle, err := reference.New(string(source), reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if oracle.SourceHash() != artifact.SourceHash {
			t.Fatal("oracle source lineage mismatch")
		}
		handle, err := oracle.Construct("Counter", semantics.Integer(c.State))
		if err != nil {
			t.Fatal(err)
		}
		method := "read"
		args := []semantics.Value{}
		if c.Operation == "Counter.increment" {
			method = "increment"
			args = append(args, semantics.Integer(c.Delta))
		}
		expected, err := oracle.Call(handle, method, args...)
		if err != nil {
			t.Fatal(err)
		}
		for _, engine := range []string{"learned", "lookup"} {
			got := executor.Run(c, engine)
			if !got.Accepted || got.Result != expected.Int || got.State != oracle.Objects()[0].Fields["value"].Int {
				t.Fatalf("%s mismatch %s: %+v", engine, c.ID, got)
			}
		}
		rejected := executor.Run(c, "zero")
		if rejected.Accepted || rejected.State != c.State || rejected.Version != 1 {
			t.Fatal("zero ablation bypassed exact gate")
		}
		checked++
	}
	if checked != 4216 {
		t.Fatal("held-out coverage changed", checked)
	}
}
