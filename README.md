# PHLAM

**Program Hierarchy and Learned Associative Memory**, pronounced *flam*.

PHLAM explores compiling and training small PHP-descended applications into
frozen learned computational representations. Learning must participate in
execution; independent exact contracts, primitives and effect gates preserve
correctness. ASTs and graph IR support compilation, training and baselines.
They are not the target deployed runtime.

## Current state

Mission 1 provides a fresh Go frontend for the Counter example, deterministic JSON
AST inspection, source diagnostics, syntax tests, and a future trace-recorder seam.
**No reference evaluator, trained model, semantic validator or learned execution
exists yet.** Runtime scenarios are explicitly skipped, not represented as passes.

Read the [project brief](PHLAM_Codex_Project_Brief.md),
[learned executor/exact boundary](docs/learned-runtime.md),
[language draft](docs/language-spec.md), and
[initial security model](docs/security-model.md).
The [mission report](docs/mission-0001-report.md) records evidence and limitations.

## Run

Requires Go 1.26+. No external Go dependencies or PHP installation.

```sh
go test -v ./...
go run ./cmd/phlam check examples/counter/counter.phlam
go run ./cmd/phlam inspect examples/counter/counter.phlam
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
These are parsing features, not executed semantics. See the spec for deliberate
restrictions. Manifests, loops, dynamic member-name syntax, effect analysis and
full name/type checking are not implemented.

## Structure

- `cmd/phlam`: CLI; `compiler/frontend`: tokens, AST, parser, diagnostics.
- `runtime/trace`: recording interface for future actual execution events.
- `examples/counter`: first source example.
- `tests/syntax`: valid/invalid corpus with expected diagnostic codes.
- `tests/differential`: skipped runtime obligations and proposed scenarios.
- `docs/decisions`, `reference/php4`: evidence and recorded architectural choices.

The skeleton creates only used directories. Future `runtime/reference`,
`runtime/learned`, `runtime/effects`, training, stdlib, gateway and debugger
packages will follow their first actual implementation.

The next slice is a small reference evaluator with real trace generation and
Counter operational contracts, followed by a separately measured learned
transition experiment. Do not quietly replace that experiment with an index VM.

## Licensing

The owner has not selected a license for original PHLAM work. Historical source
is for local research only and is excluded from Git. File-specific observations
are recorded in ADR 0001; no third-party implementation is vendored.
