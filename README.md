# PHLAME

**Program Hierarchy and Learned Associative Memory**, pronounced *flam*.

PHLAME explores compiling and training small PHP-descended applications into
frozen learned computational representations. Learning must participate in
execution; independent exact contracts, primitives and effect gates preserve
correctness. ASTs and graph IR support compilation, training and baselines.
They are not the target deployed runtime.

## Broad-feasibility roadmap

The [broad-feasibility plan](docs/broad-feasibility-plan.md) separates established
bounded execution from the remaining compiler/application and full-brief research
questions. It defines ten implementation gates, budgets, acceptance/failure
criteria and the final development-only authorship experiment. The
[first slice](docs/broad-feasibility-next-slice.md) is a reusable build/freeze/run
pipeline. [G01 now passes](experiments/g01/README.md) for the finite scalar subset,
with reproducible signed bundles and source-free execution across original,
renamed and reordered programs. The [gate ledger](docs/feasibility-gates.json)
records [G02 scalar reliability](experiments/g02/README.md): one uniform recipe
certifies all five fresh seeds per initial program, twice reproducibly. Full G02
and broad feasibility remain unresolved pending the required richer families and
sealed acceptance programs.

## Current state

**A reusable build pipeline now runs the existing scalar programs.**
`phlam build` uses locked manifests/recipes and an external development signing
key; `phlam-runtime` independently loads, recertifies and executes the frozen
bundle. [G01 commands, results and limits](experiments/g01/README.md).

The Go frontend parses Counter and emits deterministic ASTs and diagnostics.
The reference oracle now executes integer Counter methods with independent
instances, shared object identity, checked arithmetic, real execution traces and
transaction rollback. Compiler-derived operational contracts check exact returns,
state relations and ordered writes before publication.

**Independently trained components now interoperate through a shared ABI.**
Counter and a bounded Accumulator train separately, freeze as separate modules,
and pass typed integer returns through one atomic transaction. All 188 supported
cross-component transactions match the oracle; another 416 native rejections and
146 uncertified-scope cases roll back both states. The generic source-free worker
imports no compiler or oracle. [Results and reproduction](experiments/components/README.md).

**The next slice learns nonlinear saturation.** A 143-parameter ReLU executor
trained on 102 primitive examples gets all 492 held-out accumulator inputs right,
including both saturation regions and exact thresholds. With the independently
trained Counter, all 224 supported compositions match the oracle and lookup;
526 outside-domain schedules roll back. Warm certification runs no contract
arithmetic or predicates. [Results and reproduction](experiments/saturating-accumulator/README.md).

[Transaction profiling](experiments/transaction-profile/README.md) identifies
feature-buffer allocation, duplicate successful state copies and timestamp cost.
Bounded stack scratch and one state copy on success reduce learned allocations
to match lookup. Successful certified execution without phase clocks improves
from 2.495 to 1.801 µs; lookup remains faster at 1.408 µs in the matched study.

The [smaller-model study](experiments/smaller-saturation/README.md) trains and
freezes widths 2, 4 and 8. The 8-unit model certifies with 79 parameters, preserving
all 492 nontraining primitive inputs and 224 supported compositions. Widths 2 and
4 fail certification and are retained. In a matched comparison, successful
certified transactions without clocks measure 1.656 µs for 8 units, 1.894 µs
for the frozen 16-unit model and 1.547 µs for lookup. Architecture selection uses
the finite certification domain; this does not establish a minimum network size.

The [bounded seed sweep](experiments/seed-sweep/README.md) tests seeds 0–7 and
404 for widths 2 and 4. None of the 18 configurations certifies; every signed
artifact and counterexample is retained. The unchanged 8-unit model remains the
smallest established success. Failed models publish no certified transactions,
and concrete checking preserves safe partial coverage and rollback.

[Training-dynamics investigation](experiments/training-dynamics/README.md)
reproduces the failure and finds a certifiable **4-unit, 47-parameter model** by
centering and pairing random initial directions. It preserves all 594 primitive
transitions, 224 supported compositions and 526 rollbacks. Two-unit treatments
still fail. Logged gradients pass independent checks; timing remains noisy and
lookup retains the lowest median.

The [centered, paired seed sweep](experiments/paired-seed-sweep/README.md)
finds four-unit successes at seeds 7 and 404 across the fixed nine-seed grid;
all two-unit runs fail. Both 47-parameter successes retain all finite transitions,
supported compositions and atomic rollbacks. Convergence remains fragile;
matched timing ranges overlap and establish no speed improvement.

Certified learned execution is faster than concrete checking in the saturation run, but
lookup remains faster. Supplied schedules are not learned planning. The
[earlier Counter composition study](experiments/counter-compositions/README.md)
and [first numerical experiment](experiments/counter/README.md) remain available.
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
- `compiler/components`: guarded scalar path relations lowered from source.
- `runtime/components`: generic independently certified modules, value passing
  and atomic instance-frame transactions.
- `semantics`: exact values and checked integer primitives.
- `runtime/reference`: local snapshot-based oracle; `runtime/trace`: observed events.
- `runtime/learned`: bounded trained predictor, frozen loader, composition
  certificates and atomic checked transitions.
- `cmd/phlam-counter-experiment`, `cmd/phlam-counter-worker`: training/evaluation
  parent and independently invoked source-free executor.
- `cmd/phlam-counter-compositions`, `cmd/phlam-counter-compose-worker`: held-out
  composition study and source-free worker with concrete/certified boundaries.
- `examples/counter`, `examples/accumulator`, `examples/saturating-accumulator`:
  independent affine, rejecting and saturating source examples.
- `cmd/phlame-components-profile`: source-free CPU/allocation profiling with
  optional phase clocks and oracle-derived outcome validation.
- `cmd/phlame-components-experiment`, `cmd/phlame-components-worker`: independent
  module training, source-free execution and oracle/lookup measurements.
- `tests/syntax`: valid/invalid corpus with expected diagnostic codes.
- `tests/differential`: frozen-model/oracle/lookup comparison plus two skipped
  dynamic-language runtime obligations.
- `docs/decisions`, `reference/php4`: evidence and recorded architectural choices.

The next research questions concern richer state and control flow, a checker
that scales beyond finite domain enumeration, and an end-to-end benefit over
strong deterministic baselines.
Symbolic checking, generalization across programs/compositions, capability
discovery, stdlib/effects, gateways and debugging remain open. Phase 2 aliases and
eval also remain open. The learned substrate remains the architectural target.

## Licensing

The owner has not selected a license for original PHLAM work. Historical source
is for local research only and is excluded from Git. File-specific observations
are recorded in ADR 0001; no third-party implementation is vendored.
