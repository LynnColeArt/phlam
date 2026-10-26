package components

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestFrozenRawScoresUnchangedAndPredictionAllocatesNothing(t *testing.T) {
	path := "../../experiments/saturating-accumulator/run-0001/"
	key, err := PublicKey(path + "trusted-key.hex")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"counter", "accumulator"} {
		a, _, err := Load(path+name+".json", key)
		if err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path + name + "-predictions.jsonl")
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(f)
		n := 0
		for scanner.Scan() {
			var row struct {
				Operation int        `json:"operation_index"`
				State     int64      `json:"state"`
				Arg       int64      `json:"argument"`
				Scores    [3]float64 `json:"raw_scores"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
				t.Fatal(err)
			}
			got := a.Model.Predict(len(a.Contract.Operations), row.Operation, row.State, row.Arg)
			for h := range got {
				if math.Float64bits(got[h]) != math.Float64bits(row.Scores[h]) {
					t.Fatal("frozen raw score changed", name, n, h)
				}
			}
			n++
		}
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		f.Close()
		want := 330
		if name == "accumulator" {
			want = 594
		}
		if n != want {
			t.Fatal("raw prediction coverage changed")
		}
		var scores [3]float64
		allocs := testing.AllocsPerRun(100, func() { scores = a.Model.Predict(len(a.Contract.Operations), 0, 3, 2) })
		if allocs != 0 {
			t.Fatal("prediction allocated", name, allocs)
		}
		if _, err := Decode(scores); err != nil {
			t.Fatal(err)
		}
	}
}
