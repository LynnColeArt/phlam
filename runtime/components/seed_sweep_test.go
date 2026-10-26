package components

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestLowWidthSeedsReproduceAndCannotBorrowCertificate(t *testing.T) {
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
	original, _, err := Load(path+"accumulator.json", key)
	if err != nil {
		t.Fatal(err)
	}
	_, oldProof, err := NewRegistry(map[string]Artifact{"a": original}, "learned", "certified")
	if err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{2, 4} {
		var previous Model
		for _, seed := range []int64{0, 1} {
			model, _, _, err := TrainBranchingWidth(samples, 2, seed, width)
			if err != nil {
				t.Fatal(err)
			}
			same, _, _, err := TrainBranchingWidth(samples, 2, seed, width)
			if err != nil || !reflect.DeepEqual(model, same) || seed == 1 && reflect.DeepEqual(previous, model) {
				t.Fatal("seed handling is not deterministic and independent", width, seed)
			}
			candidate := original
			candidate.Model = model
			candidate.Seed = seed
			registry, proof, err := NewRegistry(map[string]Artifact{"a": candidate}, "learned", "certified")
			if err == nil || registry != nil || proof["a"].Certified || proof["a"].Key == oldProof["a"].Key {
				t.Fatal("failed model reused old certificate", width, seed)
			}
			previous = model
		}
	}
}
