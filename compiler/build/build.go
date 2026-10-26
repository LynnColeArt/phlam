package build

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/LynnColeArt/phlam/runtime/bundle"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func Save(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func BinaryHash(paths ...string) (string, error) {
	hashes := []string{}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		hashes = append(hashes, abi.Hash(raw))
	}
	raw, _ := json.Marshal(hashes)
	return abi.Hash(raw), nil
}
func Build(path string, opts Options) (report Report, err error) {
	report = Report{Schema: "phlame.build-report.v0", Status: "failed", Toolchain: runtime.Version(), ArgumentPolicy: bundle.ArgumentPolicy, ImplementationHash: opts.ImplementationHash, Components: map[string]*ComponentReport{}, Limitations: []string{"Finite scalar domains; no arrays/loops/eval/gateway or production authority.", "Explicit int64 arguments: source method defaults are not inserted by the scalar ABI.", "Training subprocess wall timeout is enforced; Go managed-memory metric/soft limit is not an OS RSS sandbox.", "Finite certification repeats at runtime; a signed candidate/report is not an accepted proof bit."}}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Output == "" || opts.Evidence == "" || len(opts.SigningKey) != ed25519.PrivateKeySize || opts.Trainer == "" || opts.Runtime == "" {
		return report, fmt.Errorf("output, separate evidence, trainer/runtime paths and explicit signing key required")
	}
	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return report, err
	}
	evidence, err := filepath.Abs(opts.Evidence)
	if err != nil {
		return report, err
	}
	if output == evidence || within(output, evidence) || within(evidence, output) {
		return report, fmt.Errorf("deployment and evidence directories must be disjoint")
	}
	if _, e := os.Lstat(output); !os.IsNotExist(e) {
		return report, fmt.Errorf("output must not exist")
	}
	if _, e := os.Lstat(evidence); !os.IsNotExist(e) {
		return report, fmt.Errorf("evidence must not exist")
	}
	if err := os.MkdirAll(evidence, 0755); err != nil {
		return report, err
	}
	defer func() {
		if err != nil {
			report.Error = err.Error()
			report.Status = "failed"
		}
		if report.Status == "built" {
			return
		}
		if e := Save(filepath.Join(evidence, "report.json"), report); err == nil && e != nil {
			err = e
		}
	}()
	manifest, plan, data, err := inputs(path)
	if err != nil {
		return report, err
	}
	manifestBytes, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	report.ManifestHash = abi.Hash(manifestBytes)
	report.PlanHash = abi.Hash(data[manifest.TrainingPlan])
	if len(report.ImplementationHash) != 64 {
		return report, fmt.Errorf("implementation fingerprint required")
	}
	ctx, cancel := context.WithTimeout(opts.Context, time.Duration(plan.BuildTimeoutMS)*time.Millisecond)
	defer cancel()
	keyPath := filepath.Join(evidence, "development-public-key.hex")
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(opts.SigningKey.Public().(ed25519.PublicKey))), 0644); err != nil {
		return report, err
	}
	// Evidence preserves locked authoring inputs; none enter the deployment staging directory.
	if err := Save(filepath.Join(evidence, "manifest.json"), manifest); err != nil {
		return report, err
	}
	if err := Save(filepath.Join(evidence, "training-plan.json"), plan); err != nil {
		return report, err
	}
	if err := os.Mkdir(filepath.Join(evidence, "inputs"), 0755); err != nil {
		return report, err
	}
	names := []string{}
	for name := range data {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		if err := os.WriteFile(filepath.Join(evidence, "inputs", fmt.Sprintf("%02d-%s", i, filepath.Base(name))), data[name], 0644); err != nil {
			return report, err
		}
	}
	parent := filepath.Dir(output)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return report, err
	}
	stage, err := os.MkdirTemp(parent, ".phlam-build-")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(stage)
	deployment := bundle.Manifest{Schema: bundle.Schema, ABI: abi.ABI, ArgumentPolicy: bundle.ArgumentPolicy, Module: manifest.Module, Modules: map[string]bundle.Binding{}}
	for _, component := range manifest.Components {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		info := &ComponentReport{Source: component.Source, Class: component.Class, Selected: -1}
		report.Components[component.Alias] = info
		contract, samples, observations, oracleNS, err := observe(ctx, component, plan, string(data[component.Source]))
		if err != nil {
			return report, err
		}
		info.SourceHash = contract.SourceHash
		info.OracleNS = oracleNS
		info.OracleCases = len(observations)
		info.TrainingCases = len(samples)
		dir := filepath.Join(evidence, component.Alias)
		if err := os.Mkdir(dir, 0755); err != nil {
			return report, err
		}
		if err := Save(filepath.Join(dir, "training.json"), samples); err != nil {
			return report, err
		}
		trainingBytes, _ := os.ReadFile(filepath.Join(dir, "training.json"))
		info.TrainingHash = abi.Hash(trainingBytes)
		if err := Save(filepath.Join(dir, "observations.json"), observations); err != nil {
			return report, err
		}
		observationBytes, _ := os.ReadFile(filepath.Join(dir, "observations.json"))
		info.ObservationHash = abi.Hash(observationBytes)
		acceptedTotal := 0
		for _, o := range observations {
			if o.Accepted {
				acceptedTotal++
			}
		}
		selected := false
	candidateLoop:
		for _, recipe := range plan.Candidates {
			for _, seed := range recipe.Seeds {
				if err := ctx.Err(); err != nil {
					return report, err
				}
				index := len(info.Attempts)
				attempt := Attempt{Recipe: recipe, Seed: seed}
				attemptDir := filepath.Join(dir, fmt.Sprintf("candidate-%02d", index))
				if err := os.Mkdir(attemptDir, 0755); err != nil {
					return report, err
				}
				task := TrainTask{Schema: "phlame.train-task.v0", Count: len(contract.Operations), Seed: seed, Recipe: recipe, Samples: samples}
				raw, _ := json.Marshal(task)
				taskContext, stop := context.WithTimeout(ctx, time.Duration(plan.CandidateTimeoutMS)*time.Millisecond)
				command := exec.CommandContext(taskContext, opts.Trainer)
				command.Stdin = bytes.NewReader(raw)
				command.Env = []string{"GOMAXPROCS=1"}
				var stderr bytes.Buffer
				command.Stderr = &stderr
				start := time.Now()
				resultBytes, trainErr := command.Output()
				attempt.TrainingNS = time.Since(start).Nanoseconds()
				if command.ProcessState != nil {
					attempt.TrainingCPUNS = (command.ProcessState.UserTime() + command.ProcessState.SystemTime()).Nanoseconds()
				}
				timedOut := taskContext.Err()
				stop()
				if trainErr != nil || timedOut != nil {
					attempt.Error = fmt.Sprintf("training failed: %v %v %s", trainErr, timedOut, &stderr)
					info.Attempts = append(info.Attempts, attempt)
					if err := Save(filepath.Join(attemptDir, "attempt.json"), attempt); err != nil {
						return report, err
					}
					continue
				}
				var trained TrainResult
				if err := abi.StrictJSON(resultBytes, &trained); err != nil {
					return report, err
				}
				attempt.Initial = trained.Initial
				attempt.Final = trained.Final
				attempt.ManagedBytes = trained.ManagedBytes
				artifact := abi.Artifact{Schema: "phlame.scalar-module.v0", ABI: abi.ABI, Contract: contract, Domain: component.Domain, Model: trained.Model, Seed: seed, Epochs: abi.Epochs, TrainingHash: info.TrainingHash, VerificationHash: info.ObservationHash, ImplementationHash: opts.ImplementationHash, Toolchain: runtime.Version()}
				if recipe.Kind == "relu" {
					artifact.Epochs = abi.BranchEpochs
					if recipe.Initialization != "legacy" {
						artifact.Initialization = recipe.Initialization
					}
					if recipe.Initialization == "paired-affine" {
						artifact.PrefitEpochs = abi.Epochs
					}
				}
				candidateDir := filepath.Join(attemptDir, "deployment")
				if err := os.Mkdir(candidateDir, 0755); err != nil {
					return report, err
				}
				moduleFile := component.Alias + ".json"
				if err := abi.Freeze(filepath.Join(candidateDir, moduleFile), artifact, opts.SigningKey); err != nil {
					return report, err
				}
				attempt.Root = abi.Root(artifact)
				attempt.Artifact = filepath.ToSlash(filepath.Join(component.Alias, fmt.Sprintf("candidate-%02d", index), "deployment", moduleFile))
				candidateManifest := bundle.Manifest{Schema: bundle.Schema, ABI: abi.ABI, ArgumentPolicy: bundle.ArgumentPolicy, Module: manifest.Module, Modules: map[string]bundle.Binding{component.Alias: {File: moduleFile, Root: attempt.Root, Entries: component.Entries}}}
				if err := bundle.SignManifest(filepath.Join(candidateDir, "bundle.json"), candidateManifest, opts.SigningKey); err != nil {
					return report, err
				}
				predictions := []map[string]any{}
				for _, o := range observations {
					scores := artifact.Model.Predict(len(contract.Operations), o.Operation, o.State, o.Arg)
					prediction, predictErr := abi.Decode(scores)
					correct := o.Accepted && predictErr == nil && prediction == o.Prediction
					if correct {
						if o.Training {
							attempt.RawTrainingCorrect++
						} else {
							attempt.RawOtherCorrect++
						}
					}
					predictions = append(predictions, map[string]any{"operation_index": o.Operation, "state": o.State, "argument": o.Arg, "source_accepted": o.Accepted, "training": o.Training, "raw_scores": scores, "decoded_prediction": prediction, "correct_accepted_prediction": correct})
				}
				if err := Save(filepath.Join(attemptDir, "predictions.json"), predictions); err != nil {
					return report, err
				}
				response, checkErr := certify(ctx, opts.Runtime, candidateDir, keyPath, component, contract, plan.CandidateTimeoutMS)
				attempt.LoadNS = response.LoadNS
				attempt.VerificationNS = response.ProcessWallNS
				attempt.VerificationCPUNS = response.ProcessCPUNS
				if checkErr != nil {
					attempt.Error = checkErr.Error()
				} else {
					attempt.Certificate = response.Certificates[component.Alias]
					if response.PreflightError != "" {
						attempt.Error = response.PreflightError
					} else if !attempt.Certificate.Certified || attempt.RawTrainingCorrect+attempt.RawOtherCorrect != acceptedTotal {
						attempt.Error = "certificate or source-observation coverage incomplete"
					}
				}
				info.Attempts = append(info.Attempts, attempt)
				if err := Save(filepath.Join(attemptDir, "attempt.json"), attempt); err != nil {
					return report, err
				}
				if attempt.Error == "" {
					raw, err := os.ReadFile(filepath.Join(candidateDir, moduleFile))
					if err != nil {
						return report, err
					}
					if err := os.WriteFile(filepath.Join(stage, moduleFile), raw, 0444); err != nil {
						return report, err
					}
					deployment.Modules[component.Alias] = candidateManifest.Modules[component.Alias]
					info.Selected = index
					selected = true
					break candidateLoop
				}
			}
		}
		if !selected {
			return report, fmt.Errorf("%s: all bounded candidates failed certification; see %s", component.Source, component.Alias)
		}
	}
	if err := bundle.SignManifest(filepath.Join(stage, "bundle.json"), deployment, opts.SigningKey); err != nil {
		return report, err
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if _, e := os.Lstat(output); !os.IsNotExist(e) {
		return report, fmt.Errorf("output appeared during build")
	}
	// Same-filesystem rename is the sole publication point. No source, key trust
	// configuration, report, training data or verification bit is deployed.
	report.Status = "built"
	if err := Save(filepath.Join(evidence, "report.json"), report); err != nil {
		return report, err
	}
	if err := os.Rename(stage, output); err != nil {
		return report, err
	}
	report.Status = "built"
	return report, nil
}
func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && rel != "." && len(rel) > 0 && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type runtimeResponse struct {
	LoadNS         int64                      `json:"load_ns"`
	ProcessWallNS  int64                      `json:"-"`
	ProcessCPUNS   int64                      `json:"-"`
	Certificates   map[string]abi.Certificate `json:"certificates"`
	PreflightError string                     `json:"preflight_error"`
}

func certify(ctx context.Context, runner, dir, key string, c Component, contract abi.Contract, timeout int) (runtimeResponse, error) {
	rctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()
	op := contract.Operations[0]
	step := abi.Step{Target: "probe", Operation: op.Name}
	if op.Arity == 1 {
		arg := c.Domain.ArgMin
		step.Argument = &abi.Binding{Literal: &arg}
	}
	// A non-exported first operation cannot be invoked by this probe, but startup
	// still validates every operation before the request-level entry check.
	raw, _ := json.Marshal(map[string]any{"transactions": []abi.Transaction{{ID: "build-probe", Instances: map[string]abi.Instance{"probe": {Component: c.Alias, Value: c.Domain.StateMin, Version: 1}}, Steps: []abi.Step{step}}}, "repeats": 1})
	command := exec.CommandContext(rctx, runner, "run", dir, "--key", key, "--request", "-")
	command.Stdin = bytes.NewReader(raw)
	command.Env = []string{"GOMAXPROCS=1"}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	start := time.Now()
	output, err := command.Output()
	if err != nil {
		return runtimeResponse{}, fmt.Errorf("source-free certification failed: %w %s", err, &stderr)
	}
	var response runtimeResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return response, err
	}
	response.ProcessWallNS = time.Since(start).Nanoseconds()
	if command.ProcessState != nil {
		response.ProcessCPUNS = (command.ProcessState.UserTime() + command.ProcessState.SystemTime()).Nanoseconds()
	}
	return response, nil
}
