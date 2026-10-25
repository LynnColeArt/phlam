# PHLAME: independently learned scalar components

Two components now train independently, freeze separately, and interoperate
through a component-independent execution ABI. The source-free worker passes
integer return values between instance-owned state frames and publishes a shared
transaction only after every call succeeds.

[Recorded results](run-0001/REPORT.md) · [Machine report](run-0001/report.json) ·
[Source accumulator](../../examples/accumulator/accumulator.phlam)

## What this tests

Counter trains on 30 accepted single-operation examples; Accumulator on 48, using
separate seeds and source files. Each has its own 15 weights, contracts, domain,
training lineage, signed artifact, and finite certificate. There are no shared
weights, joint gradients, peer-state features, or cross-component training
examples. Models receive only their own scalar state, integer argument and own
operation index.

The accumulator returns false when an amount is outside [-8,8] or the resulting
total would leave [-16,16]. Its accepted numerical update is affine. **Rejection
conditions are source-derived contract behavior enforced by the exact boundary;
they are not learned rejection predictions.** This is an interoperability and
transaction experiment, not a demonstration of learning nonlinear branches.

750 supplied schedules exercise both directions of value passing and feedback.
The oracle accepts 188 within scope, rejects 416 natively, and accepts another
146 that the models cannot cover with their chosen domains. Both primary engines
handle all three categories correctly. No learned planning is involved.

## Source-free reload and run

```sh
go build -o bin/phlame-components-worker ./cmd/phlame-components-worker
bin/phlame-components-worker experiments/components/run-0001/registry.json \
  experiments/components/run-0001/trusted-key.hex learned certified \
  < experiments/components/request.json
```

The three examples demonstrate:

1. Counter 2 increments to 3; Accumulator consumes that return and reaches 3.
   Both versions advance from 1 to 2 together. Reads return 3 from each instance.
2. Counter 8 tentatively reaches 9, which exceeds Accumulator's argument domain.
   Both states remain at their initial values/versions; no staged result publishes.
3. Counter 3 tentatively reaches 4, but Accumulator 15 cannot add 4. Its contract
   rejects the update, and both states roll back.

Engines: learned, lookup, zero, shuffled. Boundaries: concrete, certified. A failed
certification returns preflight_error with no transaction execution. Original
phlam paths/commands and GitHub identity remain intact; new material uses the
correct project name PHLAME.

## ABI v0

Each registry alias pins a signed module's content root. An instance binds that
module to an owned scalar value and version. Operations come from module metadata,
not built-in Counter/Accumulator cases. A schedule step specifies the target
instance, operation, and, for one-argument operations, exactly one of:

```json
{"literal": 3}
{"from_result": 0}
```

A result reference must name a successful prior step in this transaction. The
register file contains typed int64 values plus producer handles; the model cannot
produce a resource handle or choose a write target. Zero-argument operations take
no argument binding. Missing/ambiguous/forward bindings reject. Language default
arguments are not normalized by this explicit-argument research ABI.

All calls run on a private working state map. Typed model output can update only
the already bound instance's own declared slot. Results and ordered writes remain
private until the entire schedule succeeds. Any failure restores every state and
version and returns no application results or writes. Each written instance gets
one version increment at the outer commit, including same-value writes; untouched
instances preserve their versions.

One scalar slot per instance, at most one integer argument, successful int64
returns, false-as-rejection, and up to 16 supplied calls are supported. These are
component ABI restrictions, not the full PHLAME language semantics. Shared state
means a transaction namespace of owned slots plus value passing, **not aliased
cells jointly writable by both models**. No production authorization, tenancy,
external effects, durable storage or concurrent transaction protocol is supplied.

## Contracts and certification

compiler/components lowers scalar assignments, arithmetic, guarded branches and
returns into exact path relations. Required class purposes remain declared claims.
Guard/effect evidence comes from the parsed source. The compiler rejects calls,
cross-instance writes, unsupported shapes and excessive symbolic growth. Identity
init and one zero-default property are required for this experiment.

All 330 Counter primitive inputs and 594 Accumulator inputs are checked against
real reference execution before training artifacts are frozen. Additional
independent arithmetic checks verify the intended fixture behavior. Training uses
only the selected accepted subset. Rejected source cases are recorded as evidence,
not supplied as numerical training targets.

At worker startup each module separately checks its frozen predictor against its
contract on all admitted finite inputs. Inputs where the contract returns false
are excluded from that model's admitted region. The worker retains per-state
argument **intervals** for admission, not tables of next states or return values.
The numerical model still executes each admitted call. An outside-region input
rejects; nothing clips or repairs it.

Certificate keys bind module content, engine and ABI. The registry deep-copies
models/contracts and never accepts caller-supplied certificates. A valid replacement
gets a new key without changing the peer's key; an invalid replacement cannot
borrow the peer's certificate. Tests also cover tampering, module-root swaps,
malformed value types, invalid references and write-frame isolation.

The generic worker imports only runtime/components and semantics from this
repository: no parser/compiler, reference oracle or old Counter-specific runtime.
Each measured process directory contains only its binary, the two signed modules,
registry manifest and public research key. This is source independence, not an OS
sandbox. The publicly reproducible signing key is not a production identity.

## Reproduce

```sh
go build -o bin/phlame-components-worker ./cmd/phlame-components-worker
go run ./cmd/phlame-components-experiment -out work/components-reproduction -repeats 10
go test ./...
go vet ./...
```

Use a new output directory. Training and full-domain oracle observations are
recorded separately for each module. transactions.jsonl records the external
combined-source oracle's outcomes and observed trace hashes; its source hash is
separate from each module's original source hash. Reproduction regenerates the
same model weights under the recorded environment; timings vary.

The lookup baseline evaluates exact source-derived path contracts. In concrete
mode that one computation supplies both its result and its expected-value check;
it is not deliberately forced to compute twice. In certified mode its contract
execution is charged as lookup application work, while admission is boundary
work. Learned proposals never receive the checker's expected values. There is no
reference fallback in the worker.

Measurements distinguish binding, numerical proposal/rounding, checker, local
commit and overhead. Proposal timing includes floating-point sanity checks.
Concrete checker work includes evaluation of the operational relation; certified
checker work uses admission rows and frame/value-binding guards. Startup, source
oracle overhead and lookup's application arithmetic are separately reported.
