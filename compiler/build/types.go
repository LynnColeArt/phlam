// Package build implements the authoring-only scalar source build pipeline.
package build

import (
	"context"
	"crypto/ed25519"
	"github.com/LynnColeArt/phlam/runtime/bundle"
	abi "github.com/LynnColeArt/phlam/runtime/components"
	"github.com/LynnColeArt/phlam/runtime/trace"
)

type Component struct {
	Alias   string     `json:"alias"`
	Source  string     `json:"source"`
	Class   string     `json:"class"`
	Domain  abi.Domain `json:"domain"`
	Entries []string   `json:"entries"`
}
type Manifest struct {
	Schema       string          `json:"schema"`
	Module       bundle.Identity `json:"module"`
	Sources      []string        `json:"sources"`
	TrainingPlan string          `json:"training_plan"`
	Lock         string          `json:"lock"`
	Components   []Component     `json:"components"`
}
type Lock struct {
	Schema string            `json:"schema"`
	Files  map[string]string `json:"files"`
}
type Sampling struct {
	StateStride int64 `json:"state_stride"`
	ArgStride   int64 `json:"argument_stride"`
}
type Recipe struct {
	Kind           string  `json:"kind"`
	Width          int     `json:"width"`
	Initialization string  `json:"initialization"`
	Seeds          []int64 `json:"seeds"`
}
type Plan struct {
	Schema             string   `json:"schema"`
	Sampling           Sampling `json:"sampling"`
	MaxCases           int      `json:"max_primitive_cases"`
	CandidateTimeoutMS int      `json:"candidate_timeout_ms"`
	BuildTimeoutMS     int      `json:"build_timeout_ms"`
	Candidates         []Recipe `json:"candidates"`
}
type TrainTask struct {
	Schema  string       `json:"schema"`
	Count   int          `json:"operation_count"`
	Seed    int64        `json:"seed"`
	Recipe  Recipe       `json:"recipe"`
	Samples []abi.Sample `json:"samples"`
}
type TrainResult struct {
	Model        abi.Model `json:"model"`
	Initial      float64   `json:"initial_mse"`
	Final        float64   `json:"final_mse"`
	ManagedBytes uint64    `json:"managed_bytes_at_end"`
}
type Observation struct {
	Trace      []trace.Event  `json:"trace_events,omitempty"`
	Operation  int            `json:"operation_index"`
	State      int64          `json:"state"`
	Arg        int64          `json:"argument"`
	Accepted   bool           `json:"source_accepted"`
	Prediction abi.Prediction `json:"prediction"`
	Training   bool           `json:"training"`
	TraceHash  string         `json:"observed_trace_sha256"`
}
type Attempt struct {
	LoadNS             int64           `json:"runtime_load_ns"`
	VerificationNS     int64           `json:"runtime_verification_process_wall_ns"`
	VerificationCPUNS  int64           `json:"runtime_verification_process_cpu_ns"`
	Recipe             Recipe          `json:"recipe"`
	Seed               int64           `json:"seed"`
	Root               string          `json:"module_sha256,omitempty"`
	Artifact           string          `json:"artifact,omitempty"`
	TrainingNS         int64           `json:"training_wall_ns"`
	TrainingCPUNS      int64           `json:"training_cpu_ns"`
	ManagedBytes       uint64          `json:"trainer_managed_bytes_at_end"`
	Initial            float64         `json:"initial_mse"`
	Final              float64         `json:"final_mse"`
	RawTrainingCorrect int             `json:"raw_training_correct"`
	RawOtherCorrect    int             `json:"raw_nontraining_correct"`
	Certificate        abi.Certificate `json:"certificate"`
	Error              string          `json:"error,omitempty"`
}
type ComponentReport struct {
	Source          string    `json:"source"`
	Class           string    `json:"class"`
	SourceHash      string    `json:"source_sha256"`
	TrainingHash    string    `json:"training_sha256"`
	ObservationHash string    `json:"oracle_domain_sha256"`
	OracleNS        int64     `json:"oracle_wall_ns"`
	OracleCases     int       `json:"oracle_cases"`
	TrainingCases   int       `json:"training_cases"`
	Attempts        []Attempt `json:"attempts"`
	Selected        int       `json:"selected_attempt"`
}
type Report struct {
	Schema             string                      `json:"schema"`
	Status             string                      `json:"status"`
	Error              string                      `json:"error,omitempty"`
	ManifestHash       string                      `json:"manifest_sha256"`
	PlanHash           string                      `json:"plan_sha256"`
	ImplementationHash string                      `json:"implementation_sha256"`
	Toolchain          string                      `json:"toolchain"`
	ArgumentPolicy     string                      `json:"argument_policy"`
	Components         map[string]*ComponentReport `json:"components"`
	Limitations        []string                    `json:"limitations"`
}
type Options struct {
	Output             string
	Evidence           string
	Trainer            string
	Runtime            string
	SigningKey         ed25519.PrivateKey
	ImplementationHash string
	Context            context.Context
}
