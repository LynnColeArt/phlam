# G01: reusable source-to-frozen-artifact builds

[Acceptance report](run-0001/REPORT.md) · [Locked campaign](campaign.json) ·
[Machine report](run-0001/report.json) · [Format decision](../../docs/decisions/0003-g01-scalar-build-bundles.md)

One manifest-driven builder now compiles, observes, trains, freezes and certifies
all three existing scalar source programs without application-specific build
code. Original, renamed and method-reordered projects each contain Counter,
rejecting accumulator and saturating accumulator semantics. Every project builds
twice from clean project directories with byte-identical deployment files.

Each project runs 1,526 scenarios across learned/lookup, concrete/certified and
phase-clocks on/off: 1,447 accepted results and 79 rejections match the source
oracle and independent fixture mathematics. Cases include shared value passing,
native refusal, failure after both components stage writes, invalid references,
missing arguments, unknown entries and domain escape. All checks remain active.

## Build and run

Go 1.26+, no external Go dependencies. From the repository root:

```sh
go build -trimpath -buildvcs=false -o bin/phlam ./cmd/phlam
go build -trimpath -buildvcs=false -o bin/phlam-train ./cmd/phlam-train
go build -trimpath -buildvcs=false -o bin/phlam-runtime ./cmd/phlam-runtime
python3 experiments/g01/run.py --out work/g01-reproduction
go test ./...
go vet ./...
```

Use a fresh output directory. The runner creates a publicly reproducible
**development-only** signing fixture; no production secret or real authority is
being configured. It records seven real failed builds and preserves the failed
candidate artifacts where training produced them. The full tests also interrupt
an active training process and test signed-manifest/model tampering, root/key/ABI
mismatch, export restrictions and independently owned state snapshots.

For your own supported scalar project:

```text
phlam build path/to/phlam.mod --out NEW_DEPLOYMENT --evidence NEW_EVIDENCE \
  --signing-key CONFIGURED_DEVELOPMENT_SEED_HEX
phlam-runtime run NEW_DEPLOYMENT --key TRUSTED_PUBLIC_KEY_HEX --request request.json
phlam build-report NEW_EVIDENCE/report.json
```

These are now implemented commands; the upper-case tokens designate user-provided
paths. Signing input is a file containing a 32-byte seed as hexadecimal. The
standalone runner requires a separately supplied trusted public key. No key or
success report inside a bundle can authorize its own loading. The authoring CLI
finds phlam-train and phlam-runtime beside itself unless explicit paths are given.
The existing source-based `phlam run` remains the reference oracle command.

The training plan is referenced and hashed by the manifest/lock, not silently
overridden by a CLI flag. Both output and evidence directories must be new and
disjoint. Local sources/plan are locked by content; one explicit class per source,
one scalar property, supported identity init, integer operations/returns and
rejection false are the current compiler subset. Source method defaults remain
language semantics; artifact arguments are explicitly supplied under the signed
`explicit-int64-v0` policy. Unsupported source gets a diagnostic instead of an
exact-interpreter fallback.

## Inspect the recorded artifacts

```sh
bin/phlam-runtime run experiments/g01/run-0001/original/deployment \
  --key experiments/g01/run-0001/original/trusted-development-key.hex \
  --request experiments/g01/run-0001/original/requests.json
```

Add `--engine lookup`, `--boundary concrete` or `--phase-clocks=false` for the
comparison modes. The default is learned/certified with clocks. Each startup
loads and checks all three modules independently. Deployment has exactly four
files: three signed modules and signed bundle bindings. No source, training data,
optimizer, signing key, report or stored proof bit is present.

Source-free tests copy only these files, the standalone runtime binary and an
externally configured public key to a temporary directory. The dependency graph
excludes compiler/reference packages; a symbol check confirms optimizer routines
are not linked into the runtime. Contract specifications remain in modules, as
required for current finite certification and the lookup baseline.

Build evidence is separate: locked inputs, source observations, real trace
samples, selected training rows, every attempted candidate, raw predictions and
training/proof costs. Export checks are measured separately from the scalar
runtime relation checker. Certified warm checking uses no contract arithmetic or
predicates, while lookup application evaluation has its own counters. Timings
include instrumentation and batch retention and are diagnostic, not speed claims.

## Scope of the gate

G01 establishes reusable scalar compilation/build packaging and execution. It
adds no new learned model family and does not establish broad-language support,
training reliability, scalable proof, tenant authorization, durable storage or
OS sandboxing. Current startup certification still enumerates finite domains.
Known failed candidate models remain validly signed but cannot activate certified
execution. No signature is a substitute for correctness.
