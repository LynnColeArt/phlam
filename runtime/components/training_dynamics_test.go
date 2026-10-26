package components

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestTrainingObserverIsPassiveAndInitializationLineageIsEnforced(t *testing.T) {
	path := "../../experiments/saturating-accumulator/run-0001/"
	raw, err := os.ReadFile(path + "accumulator-training.json")
	if err != nil {
		t.Fatal(err)
	}
	var samples []Sample
	if err := json.Unmarshal(raw, &samples); err != nil {
		t.Fatal(err)
	}
	key, err := PublicKey(path + "trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	a, _, err := Load(path+"accumulator.json", key)
	if err != nil {
		t.Fatal(err)
	}
	baseline, _, _, err := TrainBranchingWidth(samples, 2, 404, 2)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	observed, _, _, err := TrainBranchingObserved(samples, 2, 404, 2, "legacy", func(s TrainingSnapshot) {
		if len(s.Units) != 2 || len(s.Model.Hidden) != 2 {
			t.Fatal("bad snapshot shape")
		}
		s.Model.Weights[0][0] = 99
		s.Model.Hidden[0][0] = 99
		count++
	})
	if err != nil || count != 11 || !reflect.DeepEqual(baseline, observed) {
		t.Fatal("observer affects optimization", count, err)
	}
	for _, init := range []string{"paired-centered", "paired-affine"} {
		model, _, _, err := TrainBranchingObserved(samples, 2, 404, 4, init, nil)
		if err != nil {
			t.Fatal(err)
		}
		candidate := a
		candidate.Model = model
		candidate.Initialization = init
		if init == "paired-affine" {
			candidate.PrefitEpochs = Epochs
		}
		_, proofs, err := NewRegistry(map[string]Artifact{"a": candidate}, "learned", "certified")
		if err != nil || !proofs["a"].Certified {
			t.Fatal("treatment failed finite domain", init, err)
		}
		candidate.PrefitEpochs++
		if ValidateArtifact(candidate) == nil {
			t.Fatal("misstated initialization lineage accepted")
		}
	}
	if _, _, _, err := TrainBranchingObserved(samples, 2, 404, 2, "unknown", nil); err == nil {
		t.Fatal("unknown initializer accepted")
	}
}
