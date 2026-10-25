# Held-out Counter compositions and a smaller warm boundary

This slice reuses the **unchanged frozen model** from the first Counter
experiment. Its only 170 training examples are single operations. No composition
examples, new weights or new training are introduced.

The [recorded results](run-0001/REPORT.md) compare 1,176 supplied operation
sequences with an atomic reference oracle and structural lookup, under both the
original concrete boundary and a finite-domain certified boundary. All 1,096
in-domain sequences match; 80 sequences escape the domain and reject atomically.
These are rollouts of learned numerical transitions on unseen schedules, not a
learned planner discovering or selecting those schedules.

## Source-free execution

```sh
go build -o bin/phlam-counter-compose-worker ./cmd/phlam-counter-compose-worker
bin/phlam-counter-compose-worker experiments/counter/run-0001/artifact.json \
  experiments/counter/run-0001/trusted-key.hex learned certified \
  < experiments/counter-compositions/request.json
```

The demo returns `[15,15,8,8,8]`, final state 8/version 2, ordered writes
`[15,8,8]`. The second sequence would leave the certified input domain after its
first step; it rejects with original state 64/version 1 and no published results
or writes, even though the requested second step would return the state to 64.
The last output of a sequence may be outside the input domain; an additional
step at that output requires a new domain check and will reject.

Engines: `learned`, `lookup`, `zero`, `shuffled`. Boundaries: `concrete` and
`certified`. Zero/shuffled certification fails before executing compositions;
this is reported as `preflight_error`, not as a completed sequence run. Wire deltas are explicit integer arguments; a missing delta decodes as zero, not
the language method's default of one. This harness does not normalize omitted
language arguments. The worker
accepts no source path and imports no runtime/reference package.

## Reproduce the full comparison

From the repository root:

```sh
go build -o bin/phlam-counter-compose-worker ./cmd/phlam-counter-compose-worker
go run ./cmd/phlam-counter-compositions -out work/compositions-reproduction -repeats 10
go test ./...
go vet ./...
```

Use a new output directory. The parent verifies the original artifact/source and
training-file lineage, generates unseen schedules before evaluation, and records
real oracle results/write traces. Each worker is launched in a temporary directory
containing exactly the worker binary, original artifact and public research key.
Only typed sequences, never oracle outputs or source, are sent on stdin. This is
source independence, not an OS filesystem sandbox or production trust system.

`observations.jsonl` stores one independently observed sequence per line, including
support classification, oracle outputs/state/version, actual ordered writes and
trace hash. The full report records data/code/artifact hashes, ablations, checker
work, startup and warm timings, and estimated break-even. The original artifact
is referenced by hash rather than retrained or replaced.

## Boundary design

`concrete` builds and checks each predicted primitive transition against its
operational contract. It repeats the same relation arithmetic as the first slice.
No intermediate writes publish; the whole sequence commits once at the end.

`certified` first enumerates all **4,386 primitive inputs**: both allowed operations,
all integer input states -64..64 and increment deltas -16..16 (read delta is zero).
It invokes the actual frozen predictor and shared rounding function and checks
every resulting concrete transition. For this artifact all pass, including
exact return values, frame behavior and write flags. This is a complete finite
check, not a general symbolic equivalence proof. It checks no compositions.

The resulting certificate belongs to a private sealed copy of artifact, engine
and composition ABI. It is rebuilt in each worker on the worker's actual float64
implementation. Caller-provided JSON certificates are never trusted. Its key
binds the artifact/model/contracts, engine and ABI. Certification and runtime use
the same predictor and quantizer. No input/output lookup table is retained.

At runtime, each step checks its operation/argument shape and that its actual
input state lies in the certified domain. The frozen model still predicts every
next state, return and write decision; its rounded output becomes the next step's
input. Score sanity checks remain. The first step is exact by the finite check;
if the next input is in-domain, the same guarantee applies again. Induction then
covers the whole staged result/write sequence. Outside-domain steps reject rather
than invoke the oracle, repair outputs, or trust extrapolation.

The candidate type has one Counter slot. Writes/results are assembled in the
requested order by the trusted runtime. This combination of finite primitive
verification, sealed parameters, domain guards and ordered assembly avoids
rechecking relation terms or recomputing additions on the warm path. Changed
artifacts/engines need a new certificate. Neither an empty Composer nor an invalid
certificate can commit.

This shifts substantial exact work to startup. Cold certification cost and its
17,028 exact additions are reported separately, as are runtime guards. Reuse only
pays off after enough compositions; short-lived workers may be slower. This is
not a proof technique for unbounded integers, dynamic state or arbitrary programs.

## Scope and remaining questions

Only one instance/slot, read and increment, and supplied schedules of 1..16 steps
are supported. Tests cover unseen sequence lengths 2,3,4,5,6,8,12,16 and all binary
operation shapes through length 6. The model does not invent schedules or choose
new operations. There is no learned planning, branching composition, alias/eval,
external effect or agent permission system.

The reference oracle executes the same calls in one private heap transaction,
with one version increment for any writes. Its own per-method contracts and trace
machinery remain active. Oracle timing includes that overhead and is a behavioral
comparison, not a matched microbenchmark. Lookup uses exact primitive dispatch
and the same boundaries; it remains the stronger practical performance baseline.

Timers/counters add overhead; workers use GOMAXPROCS=1 without CPU pinning or
frequency controls. Warm timings and startup amortization are local observations,
not guarantees. The original signing key remains publicly reproducible research
material, not a production identity. No persisted proof cache is accepted across
processes, workers, models or ABIs.
