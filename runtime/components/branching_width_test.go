package components

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

func TestFixedWidthSaturationCandidates(t *testing.T) {
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
	if len(samples) != 102 || Hash(raw) != a.TrainingHash {
		t.Fatal("training fixture changed")
	}
	for _, width := range []int{2, 4, 8} {
		model, initial, final, err := TrainBranchingWidth(samples, 2, 404, width)
		if err != nil {
			t.Fatal(err)
		}
		again, _, _, err := TrainBranchingWidth(samples, 2, 404, width)
		if err != nil || !reflect.DeepEqual(model, again) || math.IsNaN(final) || final >= initial {
			t.Fatal("non-reproducible training", width, err)
		}
		parameters := 0
		for _, row := range append(append([][]float64{}, model.Hidden...), model.Weights...) {
			parameters += len(row)
		}
		if parameters != 15+8*width || len(model.Hidden) != width {
			t.Fatal("model size differs from requested width")
		}
		candidate := a
		candidate.Model = model
		registry, proofs, err := NewRegistry(map[string]Artifact{"a": candidate}, "learned", "certified")
		if width == 8 {
			if err != nil || registry == nil || !proofs["a"].Certified || proofs["a"].Cases != 594 {
				t.Fatal("selected candidate did not certify", err)
			}
		} else if err == nil || registry != nil || proofs["a"].Certified {
			t.Fatal("failed candidate obtained certificate", width)
		}
	}
	for _, width := range []int{0, -1, 33} {
		if _, _, _, err := TrainBranchingWidth(samples, 2, 404, width); err == nil {
			t.Fatal("unbounded model width", width)
		}
	}
}
