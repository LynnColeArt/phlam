// Authoring-side optimizer subprocess. It is never part of the deployed runner.
package main

import (
	"encoding/json"
	"fmt"
	build "github.com/LynnColeArt/phlam/compiler/build"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"io"
	"os"
	"runtime"
	"runtime/debug"
)

func run() error {
	debug.SetMemoryLimit(256 * 1024 * 1024) // GC soft limit, not an OS sandbox/RSS cap.
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, 8*1024*1024+1))
	if err != nil {
		return err
	}
	if len(raw) > 8*1024*1024 {
		return fmt.Errorf("training task limit")
	}
	var task build.TrainTask
	if err := abi.StrictJSON(raw, &task); err != nil {
		return err
	}
	if task.Schema != "phlame.train-task.v0" || len(task.Samples) < 1 || len(task.Samples) > 50000 {
		return fmt.Errorf("invalid task")
	}
	var result build.TrainResult
	switch task.Recipe.Kind {
	case "affine":
		result.Model, result.Initial, result.Final, err = abi.Train(task.Samples, task.Count, task.Seed)
	case "relu":
		result.Model, result.Initial, result.Final, err = abi.TrainBranchingObserved(task.Samples, task.Count, task.Seed, task.Recipe.Width, task.Recipe.Initialization, nil)
	default:
		return fmt.Errorf("unknown recipe")
	}
	if err != nil {
		return err
	}
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	result.ManagedBytes = memory.Sys
	return json.NewEncoder(os.Stdout).Encode(result)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
