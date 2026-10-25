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

**No trained model, learned executor or symbolic term verifier exists yet.**
Reference execution is the oracle for that experiment. Dynamic variables,
reference-cell aliases and eval remain unimplemented runtime obligations.

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
defines the smaller subset that actually runs. Manifests, loops, dynamic member-name syntax, effect analysis and
full name/type checking are not implemented.

## Structure

- `cmd/phlam`: CLI; `compiler/frontend`: tokens, AST, parser, diagnostics.
- `compiler/contracts`: restricted operational contract derivation and checking.
- `semantics`: exact values and checked integer primitives.
- `runtime/reference`: local snapshot-based oracle; `runtime/trace`: observed events.
- `examples/counter`: first source example.
- `tests/syntax`: valid/invalid corpus with expected diagnostic codes.
- `tests/differential`: skipped runtime obligations and proposed scenarios.
- `docs/decisions`, `reference/php4`: evidence and recorded architectural choices.

The skeleton creates only used directories. Future `runtime/learned`, `runtime/effects`, training, stdlib, gateway and debugger
packages will follow their first actual implementation.

The oracle and concrete contract checker now support designing the bounded
learned transition experiment. It still needs learned parameters, independent
inference, and honest accounting for all exact-checker work. Phase 2 aliases and
eval also remain open. Do not quietly replace that experiment with an index VM.

## Licensing

The owner has not selected a license for original PHLAM work. Historical source
is for local research only and is excluded from Git. File-specific observations
are recorded in ADR 0001; no third-party implementation is vendored.
