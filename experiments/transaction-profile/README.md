# PHLAME transaction machinery profile

This study profiles the existing frozen saturation and Counter modules without
retraining them. It identifies inference allocations, duplicate successful state
copies and phase-clock overhead, then measures two small allocation changes
against an unchanged baseline in randomized process order.

[Recorded findings](run-0001/REPORT.md) · [All measurement samples](run-0001/measurements.json) ·
[Grouped results and checker counters](run-0001/summary.json)

On successful certified transactions without phase clocks, learned execution
improves from 2.495 to 1.801 microseconds; lookup improves from 1.741 to 1.408.
Learned allocations fall from 18.875 to 5.312 per transaction, matching lookup.
The remaining learned disadvantage is numerical prediction/decoding, while
instance-map and output allocation remain shared costs. These are medians of
three fresh-process measurements, not general performance guarantees.

## What changed

`Model.Predict` uses a call-local bounded stack buffer for validated features and
ReLU activations. It preserves the original floating-point operation order;
all 924 previously recorded raw Counter/accumulator scores remain bit-identical.
Individual prediction now allocates zero heap objects. No model weights, clamp
special case, transition table, exact fallback or training data enters inference.

The transaction runtime publishes its working state copy on success. It creates
an owned copy of the original state only when rejecting, rather than eagerly
copying original and working state for every successful transaction. Failures
still discard all tentative state, writes and return values. Both engines get
the same transaction change; caller inputs and returned snapshots remain
independently owned.

`Registry.Run` retains phase clocks. `Registry.RunUninstrumented` skips only
phase timestamps; it retains the same checker operations, work counters, output
ownership and transactional behavior. Its Timing fields are zero. The profiling
worker checks the entire clocked and unclocked outcome, including work counters,
for equality after zeroing clocks before measuring. This is an opt-in profiling
API; the existing application worker continues using the clocked Run method.

## Reproduce both revisions

Requires Go and Python 3, with no external packages. Run from the repository root:

```sh
mkdir -p work/profile-baseline-src
git archive --format=tar --output=work/profile-baseline.tar 065027d
tar -xf work/profile-baseline.tar -C work/profile-baseline-src
go build -C work/profile-baseline-src -trimpath -buildvcs=false -o ../phlame-profile-baseline ./cmd/phlame-components-profile
go build -trimpath -buildvcs=false -o work/phlame-profile-optimized ./cmd/phlame-components-profile
python3 experiments/transaction-profile/profile.py \
  --baseline work/phlame-profile-baseline \
  --optimized work/phlame-profile-optimized \
  --out work/profile-reproduction
go test ./...
go vet ./...
```

Use fresh output and baseline extraction directories. Revision 065027d adds the
profiling harness and optional clocks, before either allocation change. Both
variants therefore have the same profiling harness and timestamp controls.
Builds disable VCS metadata and trim paths; binary and fixture digests are saved.
Recorded measurements remain unchanged when reproducing into a new directory.

The runner makes a temporary source-free directory containing only the worker,
two signed modules, manifest, public key and oracle-observation data. Expected
values are used solely to validate outcomes before timing; the predictor never
receives them. Every worker verifies all 750 cases, matching 224 supported
successes and 526 domain rejections against the recorded source oracle. A test
also supplies the explicit saturation-then-failure example. No compiler or
reference oracle is linked into the profiling process.

144 fresh processes cover both variants, learned/lookup, concrete/certified,
clocks on/off, mixed/accepted/scope-rejected groups, and three repetitions. Order
is shuffled with seed 20261025. Each process uses GOMAXPROCS=1 and measures
100,000 transactions after validation, warm-up and GC. Registry load and
certification are outside the warm interval and separately recorded. JSON input
and output are outside the interval. An outer timer includes result/counter
aggregation and a global sink; phase totals cover just transaction execution.

Five separate diagnostic runs sample CPU and allocations over 1.5 million
successful certified transactions: each variant/engine without phase clocks,
plus optimized learned execution with phase clocks. These are excluded from
latency/allocation comparisons. CPU capture begins after warm-up; allocation
profiles are cumulative process samples and include startup/validation/warm-up.
Exact per-transaction allocation counts come from the separate unprofiled runs.
Only the last outcome remains in a global sink; full-batch result retention and
JSON serialization are excluded. This differs from the earlier experiment
worker, so absolute latency should not be compared directly across those studies.
The profiles and text summaries are saved together:

```sh
go tool pprof -top experiments/transaction-profile/run-0001/optimized-learned.cpu.pprof
go tool pprof -top -alloc_space experiments/transaction-profile/run-0001/optimized-learned.allocs.pprof
```

The canonical lookup remains an interpreter for exact source-derived contracts.
Its concrete expected-value computation is coalesced with application work;
certified lookup's application work is separate from its checker counters.
Source-free execution still includes contract specifications and startup finite
certification. It does not imply OS isolation or portable proof certificates.

There is no CPU pinning, statistical significance claim or comparison to a
specialized handwritten clamp. The small fixture, fixed models and supplied
schedules limit conclusions. Clock instrumentation is substantial at this scale;
clocked and unclocked results are reported separately. Atomic rollback, exact
admission and output ownership were preserved throughout.
