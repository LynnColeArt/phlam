# PHLAME transaction profiling: recorded run 0001

Recorded 2026-10-25. [Method and reproduction](../README.md) · [Raw process measurements](measurements.json) · [Grouped results](summary.json) · [Build and fixture digests](metadata.json).

## Findings

Learned execution loses to lookup primarily in numerical prediction and decoding, with substantial shared allocation and clock overhead around both engines. The certified checker is not doing extra semantic work for learned execution: its exact counters are identical across engines and code variants, with zero warm contract relations.

Two bounded changes improve successful certified learned execution without phase clocks from **2.495 to 1.801 µs** (27.8% lower median). Lookup benefits from the same transaction change, moving from **1.741 to 1.408 µs**. Learned execution remains slower by about 393 ns per successful transaction in these process medians. This is not a performance win over lookup.

Learned allocations fall from **18.875 to 5.312 per successful transaction**, exactly matching optimized lookup. Bytes fall from 2809.1 to 828.1. Frozen weights and all 924 previously recorded raw prediction scores are unchanged, bit for bit.

## Evidence from profiles

1. Before optimization, [learned CPU sampling](baseline-learned.cpu.txt) attributes 41.63% inclusive CPU time to Model.Predict, including its allocation descendants. [Allocation sampling](baseline-learned.allocs.txt) attributes 54.51% of allocated bytes to prediction/features and 33.06% to instance-map cloning. The feature slice starts small and repeatedly grows while appending hidden activations.
2. Inference now uses a validated, call-local stack buffer for at most 11 base features and 32 hidden activations. All floating-point arithmetic remains in the original order. Prediction allocates zero heap objects in regression measurements. No pooling, shared mutable scratch, cached transitions or model-specific clamp operator is introduced.
3. Successful transactions now return their working snapshot directly. Previously they allocated both an original snapshot for potential rejection and a working snapshot. Rejection still discards working state and returns an owned copy of the original, so the original and working copies remain necessary on failed calls. Both engines receive this same optimization.
4. After optimization, [learned CPU sampling](optimized-learned.cpu.txt) still attributes 35.49% inclusive CPU time to numerical predict/Decode and descendants (Model.Predict itself: 27.78% inclusive). [Allocation sampling](optimized-learned.allocs.txt) shows no predictor allocation: instance-map snapshots account for 56.26% and transaction result/write structures for 43.18% of sampled allocated bytes. These common costs remain.
5. The [clocked learned profile](optimized-learned-clocked.cpu.txt) puts 24.59% flat CPU samples in time.runtimeNow and 12.50% in runtime.nanotime. Together these clock primitives account for 37.09% of sampled CPU. Clocked and unclocked results must be reported separately at this scale.

CPU percentages are sampled shares of each individual diagnostic process, not independent additive estimates across profiles. Allocation profiles are cumulative process samples, including startup, validation and warm-up; the normalized exact counts below come from unprofiled measurement processes.

## Matched latency and allocation results

Each value is the median of three fresh-process means. Phase clocks are off in this table. Both engine variants run the same fixed models, supplied schedules and generic scalar ABI. Only the last output is retained in a global sink; validation, warm-up, startup, JSON parsing and serialization are outside the timed interval. The outer timer includes work-counter aggregation and sink assignment. These absolute timings are not directly comparable to the earlier worker that retained a full batch of outcomes.

| Certified workload | Engine | Baseline ns | Optimized ns | Baseline alloc/tx | Optimized alloc/tx | Baseline B/tx | Optimized B/tx |
|---|---|---:|---:|---:|---:|---:|---:|
| accepted | learned | 2494.6 | 1801.1 | 18.875 | 5.312 | 2809.1 | 828.1 |
| accepted | lookup | 1740.8 | 1407.7 | 7.312 | 5.312 | 1292.1 | 828.1 |
| mixed | learned | 1720.9 | 1498.8 | 11.722 | 5.683 | 1900.7 | 1063.1 |
| mixed | lookup | 1295.2 | 1161.7 | 6.279 | 5.683 | 1201.6 | 1063.1 |
| scope rejected | learned | 1219.8 | 1074.8 | 8.680 | 5.840 | 1514.5 | 1163.1 |
| scope rejected | lookup | 1032.9 | 948.5 | 5.840 | 5.840 | 1163.1 | 1163.1 |

The accepted group contains 224 cases. The scope-rejected group contains 526 source-valid cases whose execution leaves a declared input domain; it includes early failures as well as failures after tentative writes. The mixed group contains all 750, so its mean conceals some successful inference cost. An explicit saturation-then-failure schedule is additionally checked outside the benchmark mix.

| Successful certified execution, clocks off | Median ns | Min process mean | Max process mean |
|---|---:|---:|---:|
| baseline/learned | 2494.6 | 2442.6 | 2593.3 |
| optimized/learned | 1801.1 | 1795.6 | 1867.4 |
| baseline/lookup | 1740.8 | 1616.1 | 2161.0 |
| optimized/lookup | 1407.7 | 1330.7 | 1493.7 |

Allocations on rejected optimized learned and lookup transactions also match. Failed transactions still allocate an original output snapshot after discarding the working snapshot. Further reducing this shared cost would require a different state/output representation; this study does not weaken ownership or rollback to avoid those allocations.

## Separate checker and application measurements

For 100,000 successful transactions, the source-independent checker performs the following exact work. Counts match across engines, baseline/optimized variants, clock modes and repeated processes.

| Checker operation | Concrete | Certified |
|---|---:|---:|
| Integer arithmetic | 1,472,742 | 0 |
| Predicates | 1,591,054 | 0 |
| Expressions | 7,843,650 | 0 |
| Expected-output comparisons | 1,387,500 | 0 |
| Binding checks | 462,500 | 462,500 |
| Frame checks | 200,000 | 200,000 |
| Domain checks | 462,500 | 925,000 |
| Certificate uses | 0 | 462,500 |

Certified lookup executes 1,472,742 integer arithmetic operations and 1,591,054 predicates as **application work**, separately from checker work. Learned inference has no such integer contract evaluation; its floating-point neural operations and decoding are proposal work. Concrete lookup reuses its exact evaluation for both computation and comparison rather than duplicating it. Certified boundary counters being zero does not mean the boundary is free: range comparisons, binding, frame checks and certificate use remain active.

| Optimized successful execution, phase clocks on | Binding ns | Proposal ns | Checker ns | Commit ns | Overhead ns | Total ns |
|---|---:|---:|---:|---:|---:|---:|
| learned/concrete | 268.0 | 877.3 | 645.6 | 136.9 | 1556.7 | 3476.1 |
| learned/certified | 260.2 | 861.6 | 392.7 | 133.7 | 1496.2 | 3137.7 |
| lookup/concrete | 258.7 | 110.4 | 625.3 | 137.1 | 1494.7 | 2626.1 |
| lookup/certified | 266.1 | 408.1 | 400.4 | 137.1 | 1508.1 | 2723.4 |

In the clocked certified runs, proposal means are 861.6 ns learned versus 408.1 ns lookup, while checker means are 392.7 versus 400.4 ns. This supports locating the remaining engine gap in proposal computation rather than extra learned checker work. Phase cells are medians of per-process means and need not sum exactly. Total phase time excludes the outer measurement loop aggregation.

The accumulator executes a generic dense 16-unit residual ReLU network, followed by finite/range checks, integer rounding and write decoding. Lookup interprets a small three-path scalar relation. The source-free worker contains no special Counter or saturation arithmetic implementation. This fixture is small enough that the dense numerical pathway remains more expensive than exact relation evaluation. That explanation is specific to these models and schedules.

## Timestamp overhead

| Optimized successful certified execution | Clocks off, outer ns | Clocks on, outer ns | Difference ns |
|---|---:|---:|---:|
| learned | 1801.1 | 3253.6 | 1452.5 |
| lookup | 1407.7 | 2837.3 | 1429.6 |

RunUninstrumented skips phase timestamps only; all exact guards, state handling and work counters remain. The normal application worker still uses Run with clocks. These on/off differences include process variation and interactions with allocation/GC, rather than counting clock instructions in isolation. The diagnostic clocked CPU profile independently identifies clock primitives as a substantial cost.

## Source-free validation and startup

Every process verifies both clock modes against the same recorded source-oracle observations before measuring. All variants have the same semantic-outcome digest, including acceptance, rejection reason and failed step, states, versions, ordered writes and typed results. Exactly 224 transactions succeed and 526 domain failures publish original state with no results or writes. An additional four-case fixture covers lower/upper saturation, a threshold and failure after both components stage writes. Tests check independently owned input/output snapshots.

No training or source execution occurs in this study. The process directory contains worker, accumulator.json, counter.json, registry.json, trusted-key.hex and transactions.jsonl; diagnostic runs additionally create profile output files. The binary imports runtime/components and semantics, plus profiling/standard libraries, with no compiler, parser, reference oracle or old Counter executor. Expected outcomes are validation data only and never enter model inference.

Each process reloads signatures and reconstructs both finite certificates on the current worker: 594 accumulator plus 330 Counter cases. One-time contract work remains 5,949 integer arithmetic operations, 7,185 predicates and 2,772 output comparisons. No external certificate is reused. Load and certificate timings are saved per process, excluded from warm loops.

| Successful-group unclocked processes, median startup | Load/signature ms | Seal/certify ms |
|---|---:|---:|
| baseline/learned | 0.808 | 0.700 |
| optimized/learned | 0.969 | 0.557 |
| baseline/lookup | 0.848 | 0.489 |
| optimized/lookup | 0.853 | 0.385 |

## Reproducibility and limits

Baseline revision: `065027d`. Toolchain: go version go1.26.4 linux/amd64. Both binaries are built with -trimpath -buildvcs=false. The baseline includes the same optional-clock profiling API and predates either allocation optimization. Frozen model files, registry and oracle dataset are reused byte-for-byte.

| Binary | SHA-256 |
|---|---|
| baseline | `db1ef711ed015d1eb206718dbe4acc54faf44e05b794603dfcff759f5f29bbe0` |
| optimized | `dd8610e18ecf537fe870a693d6aafa9f4c0dc1121d14ae3943c12f587b99ff20` |

144 measurement processes cover 2 code variants × 2 engines × 2 boundaries × 2 clock modes × 3 workload groups × 3 repetitions. Order is shuffled with seed 20261025. Each measures 100,000 transactions after ten full group warm-up passes and a GC, with GOMAXPROCS=1. Groups are cycled deterministically and may end on a partial cycle. Model certification and the outcome validation pass happen before CPU capture or timing.

Five separate diagnostic processes cover each engine/variant with clocks off and optimized learned with clocks on, at 1.5 million successful certified transactions each. These runs are excluded from latency/allocation summaries. The extra clocked profile was added after the randomized measurement pass to investigate the observed timestamp cost. Binary profiles and rendered text summaries are retained; metadata and raw results retain all digests and startup costs.

go test -count=1 ./... and go vet ./... passed. New tests verify source-free profiling dependencies, all workload groups and clock/boundary/engine combinations, oracle mismatch rejection, zero warm certified relation arithmetic, output ownership, tentative-state rollback, unchanged raw model scores and zero-allocation prediction. Existing exhaustive certification, source oracle comparisons, module tamper/replacement and typed-binding tests remain. Two pre-existing dynamic-language differential tests remain skipped.

This is one fixed machine and finite fixture, without CPU pinning or confidence intervals. Three process samples show visible variation. The lookup baseline is the existing source-derived relation interpreter; a specialized handwritten or precomputed implementation could be faster. Sampling percentages are approximate and validation uses previously recorded oracle evidence rather than re-executing source in the measured process. These limits preclude a broad efficiency claim.

The next useful performance experiments are a denser numerical representation or a smaller learned model, with frozen raw behavior recertified, and a compact owned state/output ABI that benefits both engines. Neither should replace learned execution with an exact clamp or remove the exact boundary. No such additional representation change is included here.
