package main

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	build "github.com/LynnColeArt/phlam/compiler/build"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
)

func runBuild(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: phlam build <phlam.mod> --out <new-dir> --evidence <new-dir> --signing-key <seed.hex> [--trainer path --runtime path] | phlam build-report <report.json>")
		return 2
	}
	if args[0] == "build-report" {
		if len(args) != 2 {
			return 2
		}
		raw, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var r build.Report
		if err := json.Unmarshal(raw, &r); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		json.NewEncoder(stdout).Encode(r)
		return 0
	}
	exe, _ := os.Executable()
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(stderr)
	out := flags.String("out", "", "new deployment directory")
	evidence := flags.String("evidence", "", "new development evidence directory")
	keyPath := flags.String("signing-key", "", "development signing seed hex file")
	trainer := flags.String("trainer", filepath.Join(filepath.Dir(exe), "phlam-train"), "optimizer sidecar")
	runner := flags.String("runtime", filepath.Join(filepath.Dir(exe), "phlam-runtime"), "source-free runtime sidecar")
	if err := flags.Parse(args[2:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return 2
	}
	raw, err := os.ReadFile(*keyPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	seed, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		fmt.Fprintln(stderr, "signing key must be a 32-byte hex seed")
		return 1
	}
	trainerPath, err := filepath.Abs(*trainer)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	runnerPath, err := filepath.Abs(*runner)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fingerprint, err := build.BinaryHash(exe, trainerPath, runnerPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	report, err := build.Build(args[1], build.Options{Output: *out, Evidence: *evidence, Trainer: trainerPath, Runtime: runnerPath, SigningKey: ed25519.NewKeyFromSeed(seed), ImplementationHash: fingerprint, Context: ctx})
	if e := json.NewEncoder(stdout).Encode(report); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
