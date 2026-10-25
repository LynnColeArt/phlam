# First bounded learned Counter experiment

This experiment trains a linear state-transition model from observed reference
runs, freezes its weights with exact contracts, reloads the artifact in a separate
process without source, and compares with the oracle and structural lookup.

The [recorded result](run-0001/REPORT.md) includes correctness, ablations, timing,
checker work, lineage and limitations. The [machine report](run-0001/report.json),
[frozen artifact](run-0001/artifact.json), [training observations](run-0001/training.json)
and [complete split membership](run-0001/splits.json) are committed evidence.

## Reload and invoke without source

```sh
go build -o bin/phlam-counter-worker ./cmd/phlam-counter-worker
bin/phlam-counter-worker experiments/counter/run-0001/artifact.json \
  experiments/counter/run-0001/trusted-key.hex learned \
  < experiments/counter/request.json
```

The worker accepts an artifact, a trusted research public key, an engine name,
and typed cases on stdin. It accepts no source path and imports no reference
oracle. It returns 78 for the increment and -63 for the read, with stage timing
and actual checker counters. `lookup`, `zero`, and `shuffled` select the structural
baseline or weight ablations, not fallbacks.

## Reproduce training and the full comparison

Run from the repository root:

```sh
go build -o bin/phlam-counter-worker ./cmd/phlam-counter-worker
go run ./cmd/phlam-counter-experiment -out work/counter-reproduction -repeats 10
go test ./...
go vet ./...
```

Choose a new output directory each time; existing output is refused. Source is
used only by the training/evaluation parent. The orchestrator launches each worker
in a fresh temporary directory containing exactly `worker`, `artifact.json` and
`trusted-key.hex`. Requests contain no expected labels, oracle traces, source or
AST. The process receives only GOMAXPROCS=1 as an explicit environment entry.
This demonstrates absence of a source dependency, not filesystem sandboxing.

The artifact contains source/contract/training/split/implementation hashes, model
shape/ABI/domain, seed, optimizer settings and toolchain. Source text, training
examples, AST and reference interpreter are not in the artifact. The common
contract package uses compiler source-span types, so the worker dependency graph
includes the frontend schema package; it does not call parsing or derivation.
`go list -deps ./cmd/phlam-counter-worker` has no runtime/reference dependency.

The immutable version is its SHA-256/signature, not merely a file permission.
Signatures use a publicly reproducible research fixture key. This exercises
signature verification and catches tampering under a supplied key; anyone can
reproduce this key, so it supplies no production author identity or malicious
publisher protection. A production deployment needs a separately managed trusted
principal. There is no production gateway or external effect authority here.

## What was learned

Inputs are `[1, state/64, delta/64, is_increment]`; read requests use delta=0.
Three independently initialized linear heads predict next state/64, return/64,
and a write score. Full-batch gradient descent minimizes squared error on labels
from actual oracle execution. The first two outputs are rounded to integers;
write is selected at score >= 0.5. Parameters are frozen during invocation.

This is a direct numerical learned computation, not embedding retrieval or a
menu of prewritten graph paths. It is also an exceptionally simple task: the
features reveal the operation, absent read arguments are encoded as zero, and the
true numerical relationship is affine. No unseen program or operation composition
is tested. Do not infer broader learned-language execution from this result.

The exact checker repeats the relation arithmetic before publishing a local
candidate state/version. It does not create the candidate or repair bad outputs.
Rejected candidates return the original state/version; there is no oracle call.
This concrete baseline does not implement the symbolic term verifier envisioned
in the general architecture.

The lookup baseline maps operation IDs to exact read/add primitives, then uses
the same candidate/check/commit path. It generalizes across the entire numeric
domain, so this comparison does not depend on a weak memorized-example baseline.
Its exact primitive result uses the shared numerical candidate interface; it
performs no training. The oracle remains the external behavioral comparator.

## Measurement limits

Stage times are instrumented microbenchmarks on one host, without CPU pinning or
frequency control. Timer and checker-counter overhead are included. Total time
also reports unassigned dispatch/instrumentation overhead. Accuracy counts unique
held-out cases; learned/lookup times include ten repetitions, ablations one.

Fresh-process artifact load timing excludes process startup and public-key loading;
filesystem caches are warm. Load allocation is a Go TotalAlloc delta, not RSS or
peak runtime memory. The oracle's invoke timing includes heap snapshots, tracing,
method calls and its own uninstrumented contract check; parse/construction are
outside the timed interval. It is not an apples-to-apples speed benchmark against
the lightweight transition worker. No efficiency advantage is claimed.
