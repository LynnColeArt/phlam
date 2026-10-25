# PHLAM

**Program Hierarchy and Learned Associative Memory**, pronounced *flam*.

PHLAM explores compiling and training small PHP-descended applications into
frozen learned computational representations. Learning must participate in
execution; independent exact contracts, primitives and effect gates preserve
correctness. ASTs and graph IR support compilation, training and baselines.
They are not the target deployed runtime.

## Current state

The Go frontend parses Counter and emits deterministic ASTs and diagnostics.
The reference oracle now executes integer Counter methods with independent
instances, shared object identity, checked arithmetic, real execution traces and
transaction rollback. Compiler-derived operational contracts check exact returns,
state relations and ordered writes before publication.

**Frozen learned Counter execution now covers held-out operation sequences.**
The unchanged model trained on 170 single steps matches the oracle on 1,096
supported compositions across 136 shapes. Another 80 schedules leave its finite
domain and reject atomically. These are caller-supplied schedules, not learned
planning. [Composition results](experiments/counter-compositions/run-0001/REPORT.md).

A load-time finite-domain certificate removes duplicated relation additions from
the warm checker, with startup cost and break-even measured separately. Structural
lookup remains equally correct and faster overall. The
[first numerical experiment](experiments/counter/README.md) remains available.
General symbolic verification, dynamic variables, reference-cell aliases and eval
remain unimplemented.

Read the [project brief](PHLAM_Codex_Project_Brief.md),
[learned executor/exact boundary](docs/learned-runtime.md),
[language draft](docs/language-spec.md), and
[initial security model](docs/security-model.md).
The [Counter execution guide](docs/reference-execution.md) and
[second-slice report](docs/mission-0002-report.md) record current evidence and limits.
The [first mission report](docs/mission-0001-report.md) preserves the bootstrap history.

## Run

Requires Go 1.26+. No external Go dependencies or PHP installation.

```sh
go test -v ./...
go run ./cmd/phlam check examples/counter/counter.phlam
go run ./cmd/phlam inspect examples/counter/counter.phlam
go run ./cmd/phlam contracts examples/counter/counter.phlam
go run ./cmd/phlam run examples/counter/counter.phlam examples/counter/scenario.json
mkdir -p bin
go build -o bin/phlam ./cmd/phlam
./bin/phlam check tests/syntax/invalid/missing-purpose.phlam
```

The last command intentionally fails with P101 and a source location. `check`
validates bootstrap syntax and local declarations only. `inspect` accepts source,
not a trained artifact. Both require an explicit source path. Exit codes: 0
success, 1 source/read/output error, 2 usage error. Output is deterministic for
identical source bytes and contains no timestamp or absolute checkout path.

Implemented: documented classes, properties, init/methods, parameters/defaults,
integer/string/bool/null literals, arrays, variables/indirect variables, alias
syntax, calls/new, arithmetic/comparisons, if/else, return, optional semicolons.
Those are frontend features. The [execution guide](docs/reference-execution.md)
defines the smaller subset that actually runs. Manifests, loops, dynamic member-name syntax, general effect analysis and
full name/type checking are not implemented.

## Structure

- `cmd/phlam`: CLI; `compiler/frontend`: tokens, AST, parser, diagnostics.
- `compiler/contracts`: restricted operational contract derivation and checking.
- `semantics`: exact values and checked integer primitives.
- `runtime/reference`: local snapshot-based oracle; `runtime/trace`: observed events.
- `runtime/learned`: bounded trained predictor, frozen loader, composition
  certificates and atomic checked transitions.
- `cmd/phlam-counter-experiment`, `cmd/phlam-counter-worker`: training/evaluation
  parent and independently invoked source-free executor.
- `cmd/phlam-counter-compositions`, `cmd/phlam-counter-compose-worker`: held-out
  composition study and source-free worker with concrete/certified boundaries.
- `examples/counter`: first source example.
- `tests/syntax`: valid/invalid corpus with expected diagnostic codes.
- `tests/differential`: frozen-model/oracle/lookup comparison plus two skipped
  dynamic-language runtime obligations.
- `docs/decisions`, `reference/php4`: evidence and recorded architectural choices.

The next research question is whether learned execution can contribute beyond
supplied schedules in this affine family, with a checker that scales beyond finite
domain enumeration.
Symbolic checking, generalization across programs/compositions, capability
discovery, stdlib/effects, gateways and debugging remain open. Phase 2 aliases and
eval also remain open. The learned substrate remains the architectural target.

## Licensing

The owner has not selected a license for original PHLAM work. Historical source
is for local research only and is excluded from Git. File-specific observations
are recorded in ADR 0001; no third-party implementation is vendored.
