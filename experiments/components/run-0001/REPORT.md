# PHLAME components 0001: independent models, shared transaction

2026-10-25; go1.26.4, linux/amd64; AMD Ryzen 5 5600X host,
source-free workers GOMAXPROCS=1. [Reproduction and ABI](../README.md).

## Finding

Independently trained Counter and bounded Accumulator models interoperate through
integer value passing and a shared atomic transaction. Neither model receives the
other's identity, state, parameters, gradients or training examples. The worker
has no hard-coded Counter/Accumulator logic and imports no compiler or oracle.

All **188 supported transactions** match the external source oracle, including
returned values, ordered writes, both states and versions. All **416 native
rejections** and **146 source-valid but uncertified transactions** reject without
publishing partial writes or results. There were **zero wrong commits**.

This establishes scoped module interoperability, not learned planning or general
module semantics. The models learn accepted affine transitions; source-derived
contract conditions enforce rejection. Their slots are distinct and owned, joined
by value passing and one transaction, not shared-cell aliasing.

## Independently trained/frozen modules

| Module | Seed | Training examples | Parameters | Observed primitive inputs | Native rejects | Training ms | Artifact bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| accumulator | 202 | 48 | 15 | 594 | 72 | 15.482 | 3,331 |
| counter | 101 | 30 | 15 | 330 | 0 | 10.295 | 1,938 |

Each model has three independently initialized linear heads over bias, own
state/32, argument/32 and a one-hot index of its own operations. They predict next
state, return and write flag. Full-batch gradient descent uses 12,000 epochs and
learning rate 0.3. Counter's state grid uses stride 8 and argument stride 2;
Accumulator uses strides 4/4. Only accepted observations on these grids train.

All 924 primitive inputs are observed from actual source execution and compared with
source-lowered path relations plus an independent arithmetic statement of the
fixtures. Counter has 300 held-out accepted primitive inputs; Accumulator has 474,
plus 72 recorded rejection inputs. Initial/final normalized MSE:

- Counter: 0.370058937349 -> 9.04857392104e-31.
- Accumulator: 0.316841299322 -> 3.75180092189e-31.

Same-input retraining is bit-identical for both in this environment. Reordering
training and replacing one seed do not affect the other model or certificate.
These are new artifacts for the generic one-hot ABI; the older Counter artifacts
remain unchanged. No cross-component schedules train either model.

## Behavioral coverage

750 transactions are generated before worker evaluation: five initial values
(-16,-8,0,8,16) per instance, five literal deltas (-4,-1,0,1,4), and six supplied
schedules. Schedules pass Counter returns to Accumulator, reverse that flow,
feed results back, and read both instances. All are new compositions.

The accumulator source rejects an amount outside [-8,8] or an update outside
[-16,16]. Its class is independently compiled, with three path relations for add
and a read relation. Return false means ABI operation rejection, causing the
outer transaction to abort. Counter's chosen certificate domain is state
[-16,16], argument [-4,4]; Accumulator uses state [-16,16], argument [-8,8], further
restricted by its source guards. Source-valid excursions beyond these model
domains are reported separately from native failures.

| Engine/boundary | Correct accepted | Supported rejected | Rejected: native-failure category | Rejected: uncertified-scope category | Wrong accepted |
| --- | ---: | ---: | ---: | ---: | ---: |
| learned/concrete | 188 | 0 | 416 | 146 | 0 |
| learned/certified | 188 | 0 | 416 | 146 | 0 |
| lookup/concrete | 188 | 0 | 416 | 146 | 0 |
| lookup/certified | 188 | 0 | 416 | 146 | 0 |
| zero/concrete | 0 | 188 | 416 | 146 | 0 |
| shuffled/concrete | 0 | 188 | 416 | 146 | 0 |

Native/scope columns classify the external oracle's transaction, not necessarily
the first rejection reason of a bad model. Zero/shuffled certified mode fails
preflight in Accumulator before any transaction runs; no success percentage is
reported for that unevaluated batch. Concrete ablations accept none. Neither
mode repairs a bad learned output or invokes a reference fallback.

## Cross-component atomicity and frames

The decisive case: Counter 8 increments to 9, then forwards 9 to Accumulator.
The accumulator input is inadmissible. Counter remains 8/version 1, Accumulator
remains 0/version 1, and no intermediate return or write publishes. A separate
case forwards valid amount 4 into Accumulator at 15; its capacity guard rejects
and both states roll back. Successful cases advance each written instance once
at the outer commit, not once per primitive write.

The ABI's private register file stores int64 values with producer handles. Arguments
are exactly one literal or prior-result reference. Model outputs contain no target,
slot, instance handle, permission or component ID. The runtime binds the target
first and can write only that instance's declared slot. Independent frame tests
show that a one-instance operation preserves its peer, including its version.

Tests cover invalid/forward/ambiguous references, missing instances/operations,
forged value types, uninitialized registries, sealed-data mutation, tampered
artifacts, swapped registry roots and malformed model shapes. A semantically valid
replacement gets a new component certificate; the other component's key is
unchanged. An invalid replacement cannot borrow that peer's certificate.

## Certification work versus warm work

Each independently signed module is checked against its own finite contract before
admitting learned calls. The constructor retains admission intervals per state and
operation, not transition/return lookup tables. Rejected source inputs populate
no admitted region. Every actual accepted numerical update still comes from its
model. Certificate keys bind module content, engine and ABI
`phlame.scalar-components.v0`. They are rebuilt in each process, never accepted from caller JSON.

| Learned module | Primitive cases | Admitted inputs | Guard-rejected inputs | Certification ms | Exact integer arithmetic | Predicate evaluations |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| accumulator | 594 | 522 | 72 | 0.267 | 4,200 | 12,831 |
| counter | 330 | 330 | 0 | 0.086 | 1,188 | 0 |

For 7,500 timed learned transactions (750 × 10), warm work:

| Boundary | Relation expressions | Relation integer arithmetic | Relation predicates | Binding checks | Frame checks | Domain checks | Certified step uses |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Concrete | 362,450 | 52,120 | 97,970 | 21,280 | 15,000 | 21,280 | 0 |
| Certified | 0 | 0 | 0 | 21,280 | 15,000 | 37,640 | 15,660 |

Counts include rejected prefixes. The certified boundary removes runtime evaluation
of relation arithmetic/predicates, not all checking. Input binding, frame ownership,
coarse domain and admission interval checks remain. Rejected transactions retain
no application returns/writes, even though local oracle traces preserve evidence
of attempted calls for this research harness.

## Timing: no end-to-end performance win

Means are µs per scheduled transaction, including failures. Separate p50/p95 and
startup records are in report.json. Stage timers/counters add overhead; this is
one local measurement without CPU pinning or frequency controls.

| Stage | Learned concrete | Learned certified | Lookup concrete | Lookup certified |
| --- | ---: | ---: | ---: | ---: |
| binding | 0.282 | 0.220 | 0.204 | 0.207 |
| proposal | 0.231 | 0.242 | 0.047 | 0.274 |
| checker | 0.431 | 0.378 | 0.431 | 0.286 |
| commit | 0.042 | 0.049 | 0.045 | 0.043 |
| overhead | 1.622 | 1.833 | 1.700 | 1.683 |
| total | 2.609 | 2.722 | 2.427 | 2.493 |

The learned checker mean falls from 431.3 ns to
378.5 ns. However, total mean rises from
2.609 µs to 2.722 µs.
**There is no measured total-time break-even** for either primary engine in this
run. The report deliberately leaves the break-even map empty. Binding, state-map,
allocation and instrumentation overhead dominate this small/rejection-heavy task.
Do not infer that less arithmetic guarantees lower latency.

Lookup is a strong deterministic baseline: it evaluates source-derived exact
relations. Concrete mode reuses its expected result as the candidate rather than
recomputing it, charged under checker time. Certified mode evaluates its relation
as application work under proposal time, separate from boundary work. Therefore
lookup stage accounting differs between modes; compare totals as well as the
explicit work counters. Learned proposal time includes inference, rounding and
floating-point sanity checks. Its model never receives the checker's expected
values. Registry binding and source-expression interpretation are not hidden in
learned inference.

Combined-source oracle mean 19.473 µs
includes heap snapshots, per-method execution and tracing, and excludes source
parsing and construction. Its machinery differs; it is a behavioral comparator,
not an apples-to-apples speed baseline.

## Source-free execution, lineage and limits

Measured worker directory:
`accumulator.json`, `counter.json`, `registry.json`, `trusted-key.hex`, `worker`.
No source, training data, compiler or oracle was copied. Repository dependencies
are only runtime/components and semantics. Observed signatures use a publicly
reproducible research key, not a production trust identity. Requests are supplied
schedules and initialized local instance snapshots, not an authenticated service.

- accumulator source: `bda959bd8633e8f23b07db418c453df591ffa80659fd92aeff45844d97a81e36`
- accumulator module root: `3cbaa95889e8884c9cd0ee028becbeb621daeb26c6f0922bf0730f7a47b207e6`
- accumulator training file: `dffed32e05ca84a47cb0bd751913ee433fb416c9d539863e4e837b757b5edfba`
- accumulator primitive observation file: `5a45459bde618c8c0c34256a9778e42a7827ce46b8ac80765a497aba4fe361ee`
- counter source: `fd26a7823ae1fd39b10f99a48ca42c99208acf36e598c7c46cd9b69ea0450d9e`
- counter module root: `95fcdf2f39f1d0c4fce5d126bb269167ddfd112b935f9238a20bcd2521c004b5`
- counter training file: `e986fd51e7146963dbcfa36c82d96d5a4f8d0e8f4e583f20ad0ffd4a1057b490`
- counter primitive observation file: `ce8b76024e2f5717b67ae6dcbed635f48f5dabf0c08b2091c52e4c6c5b1d7c11`
- Combined oracle source: `2f2db8cb1b093cbdb60d6c74aed83f473ee3285b662cd878e89ce7d1d85937e2`
- Cross-component observations: `4d209d1936cf076374e71b188d086a58613b18984d13223fd4a5486bd190fb1c`
- Implementation manifest: `1e3cd5b421356ce938627db7b332eba7781b308181d9947dbb5ce985771ae432`
- Measured worker binary: `eb5bb0dd07a3fa71364222a83582fb0809b0a5a71fab343f1d7afb29b54f178c`

Each source is verified separately for training; the external transaction oracle
concatenates the two class files, giving its traces a separate combined source
hash. Full primitive and transaction observation records, real trace hashes,
training samples and frozen artifacts are committed beside this report. The
implementation digest covers sorted Go-source paths/hashes under
cmd/compiler/runtime/semantics plus go.mod. Rebuilt binary bytes can differ with
Git VCS metadata or toolchain; the recorded hash identifies this measured build.

This is one scalar slot per instance, int64 passing, at most one argument per
operation, false-as-rejection and deterministic schedules of up to 16 calls. It
demonstrates interoperability and independent certificate/frame binding in this
scope. It does not demonstrate shared-cell aliases, arbitrary data shapes,
nonlinear learned rejection, general module learning, planning, tenant isolation,
external effects, durable commits or unbounded proofs.
