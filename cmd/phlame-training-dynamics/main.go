// Offline training diagnostics: input is only the original training samples.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"os"
)

func main() {
	input := flag.String("samples", "experiments/saturating-accumulator/run-0001/accumulator-training.json", "training samples only")
	init := flag.String("initialization", "legacy", "legacy, paired-centered or paired-affine")
	width := flag.Int("width", 2, "hidden width")
	seed := flag.Int64("seed", 404, "seed")
	flag.Parse()
	raw, err := os.ReadFile(*input)
	if err != nil {
		panic(err)
	}
	var samples []abi.Sample
	if err := json.Unmarshal(raw, &samples); err != nil {
		panic(err)
	}
	snapshots := []abi.TrainingSnapshot{}
	model, initial, final, err := abi.TrainBranchingObserved(samples, 2, *seed, *width, *init, func(s abi.TrainingSnapshot) { snapshots = append(snapshots, s) })
	if err != nil {
		panic(err)
	}
	out := struct {
		Model          abi.Model `json:"model"`
		Initial, Final float64
		Snapshots      []abi.TrainingSnapshot `json:"snapshots"`
	}{model, initial, final, snapshots}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
