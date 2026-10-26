# PHLAME centered, paired seed sweep: recorded run 0001

Recorded 2026-10-26. [Protocol/reproduction](../README.md) · [All candidates](selection.json) · [Matched results](summary.json).

## Outcome

**Four-unit seeds 7 and 404 fully certify; no two-unit seed does.** Both widths use the fixed seeds 0,1,2,3,4,5,6,7,404 with centered, opposing random initialization, 102 training examples and 24,000 joint Adam epochs. No affine prefit or extra optimization budget is used. All 18 configurations train, freeze, sign, reload source-free and attempt the unchanged full-domain certification requirement. Every failure is retained.

Seed 7 is one new successful four-unit realization; seed 404 reproduces the known model byte-for-byte. Among only the eight new seeds, success counts are **1/8 at width 4** and **0/8 at width 2**. Including the known control gives 2/9 and 0/9. These fixed-grid counts are not a statistical estimate of general optimizer reliability. Seven of nine four-unit configurations still fail.

Both successes have 47 parameters, exact decoded behavior on all 594 accumulator inputs, all 224 supported compositions matching the source oracle/lookup and all 526 outside-domain schedules rolling back. The two-unit failures establish no minimum representational capacity. The earlier legacy-initialization sweep recorded 0/9 successes per width; centered pairing helps some configurations without making this setup reliable across the grid.

## Fixed protocol and every result

The grid, seed order and selection rule were stated before execution. Initialization zeroes hidden bias/operation intercepts and pairs opposing random state/argument directions; it does not supply a preferred arithmetic direction, saturation thresholds, branch labels, contracts or nontraining targets. Output heads keep their random initialization. Source, domains, training examples, optimizer settings and numerical inference remain unchanged. Counter keeps independent seed 101.

Each candidate trains twice to check bit-identical reproduction, then freezes before source-free checking. Selection chooses the first full success per width in declared order before measuring runtime: width 2 selects none, width 4 selects seed 7. MSE, partial coverage and latency do not override full certification. Nontraining inputs are certification/selection data, not an independent seed-selection holdout.

| Width | Seed | Training correct / 102 | Nontraining correct / 492 | Correct concrete compositions / 224 | Fully certified | First failed add (state,arg) |
|---:|---:|---:|---:|---:|---|---|
| 2 | 0 | 40 | 200 | 68 | no | (-16,-8) |
| 2 | 1 | 40 | 200 | 68 | no | (-16,-8) |
| 2 | 2 | 40 | 200 | 68 | no | (-16,-8) |
| 2 | 3 | 60 | 310 | 105 | no | (-16,-8) |
| 2 | 4 | 60 | 310 | 105 | no | (-16,-8) |
| 2 | 5 | 40 | 199 | 68 | no | (-16,-8) |
| 2 | 6 | 40 | 200 | 68 | no | (-16,-8) |
| 2 | 7 | 40 | 200 | 68 | no | (-16,-8) |
| 2 | 404 | 40 | 200 | 68 | no | (-16,-8) |
| 4 | 0 | 63 | 284 | 126 | no | (-16,0) |
| 4 | 1 | 60 | 310 | 105 | no | (-15,-1) |
| 4 | 2 | 60 | 309 | 105 | no | (-15,-1) |
| 4 | 3 | 40 | 200 | 68 | no | (-16,-8) |
| 4 | 4 | 60 | 310 | 105 | no | (-15,-1) |
| 4 | 5 | 40 | 200 | 68 | no | (-16,-8) |
| 4 | 6 | 63 | 284 | 126 | no | (-16,-8) |
| 4 | 7 | 102 | 492 | 224 | yes | — |
| 4 | 404 | 102 | 492 | 224 | yes | — |

Each failed model also misses training examples, so the failures do not isolate nontraining generalization. Raw scores on all 594 primitive inputs are retained even though startup certification stops at the first counterexample. A failure issues no partial certificate and executes zero certified transactions. In concrete mode all accepted complete schedules match the oracle, and every rejection restores original states/versions with no staged writes or application results. No expected-output repair or domain narrowing is used.

Rejection counts in trial reports group source-oracle categories rather than necessarily the first rejection reason. Failed models may reject a wrong proposal before a later outside-domain step. Lookup completes all 224 supported schedules and rejects the 526 scope cases in every trial; zero incorrect accepted results are observed.

## Successful model and atomic examples

The selected [seed-7 artifact](trials/w4-s7/accumulator.json) contains 47 weights and is 3,780 signed JSON bytes. Maximum nontraining next-state/return score error is 0.000095498 integer units, below the 0.5 rounding threshold. The frozen seed-404 control is 3,786 bytes with error 0.000013880 units. Their raw floats differ, but all decoded integer states, returns and write flags match the source specification.

Every successful startup checks 594 accumulator and 330 Counter inputs independently on the current worker. Certificate keys bind each model root/engine/ABI; seed 7 cannot borrow the prior model’s result. Both registries use the same frozen Counter and have identical Counter proof keys. Input admission intervals retain no next-state/output table, and the numerical model still executes every admitted call.

[Seed-7 example outcomes](trials/w4-s7/example-results.json), [common requests](common/request.json) and [real source traces](common/oracle-traces.json) show upper/lower saturation, a threshold and a later failure after both instances stage writes. Counter 0 reaches 1; accumulator 16 successfully writes 16 after consuming that return; passing 16 to Counter exceeds its argument domain. Both states/versions remain original, with no application writes or results published. The source accepts that schedule; its rejection is explicitly a learned-domain restriction. Debug traces retain attempted values separately from application outcomes.

All 750 schedules are supplied, untrained compositions. The source oracle accepts all 750, of which 224 stay inside every declared primitive input domain. Native source rejection is not conflated with the 526 domain failures. This remains finite decoded execution rather than learned planning or correctness outside the bounded domain.

## Matched runtime measurements

One current source-free profiling binary compares the first certifying candidate (seed 7), the prior frozen four-unit seed-404 control and lookup. It uses one common Counter; contracts, domains, initializer, examples and optimizer settings match across learned variants. Every process reloads signatures, recertifies the complete finite domain, compares clocked/unclocked outcomes and verifies all 750 source-oracle outcomes before measuring. Semantic hashes agree across engines, boundaries and clock modes.

180 fresh-process measurements cover three variants × two boundaries × two clock modes × three groups × five repetitions. Order is shuffled with seed 20261025. Each process measures 100,000 transactions after validation, ten full group warm-up passes and a GC, with GOMAXPROCS=1. Three separate CPU/allocation diagnostic processes use 1.5 million accepted certified transactions each and are excluded from latency summaries.

| Accepted certified execution, phase clocks off | Median process mean ns | Min | Max | Allocations/tx | Bytes/tx |
|---|---:|---:|---:|---:|---:|
| learned-w4-s7 | 1507.2 | 1485.8 | 1546.4 | 5.312 | 828.1 |
| learned-4-control | 1485.9 | 1412.1 | 1643.7 | 5.312 | 828.1 |
| lookup | 1462.3 | 1440.4 | 1565.6 | 5.312 | 828.1 |

Ranges overlap substantially and lookup has the lowest median. This study establishes no speed improvement, statistical equivalence or seed-dependent performance conclusion. No timing run was repeated for a better result. Outer latency includes loop aggregation and retaining the last result in a sink; JSON, startup, validation and full-batch retention are excluded. There is no CPU pinning or confidence interval.

| Accepted certified execution, phase clocks on | Proposal ns | Checker ns | Binding ns | Commit ns | Overhead ns | Total phase ns |
|---|---:|---:|---:|---:|---:|---:|
| learned-w4-s7 | 505.2 | 388.9 | 257.4 | 137.2 | 1505.8 | 2795.4 |
| learned-4-control | 505.3 | 387.8 | 256.3 | 136.4 | 1505.1 | 2790.6 |
| lookup | 505.1 | 424.4 | 278.1 | 147.0 | 1566.0 | 2920.6 |

Phase entries are medians of per-process means and need not sum. Clocked data is reported separately, not used as a clock-free estimate. All concrete, mixed and rejected-group results remain in summary.json and measurements.json.

## Separate checker work

For 100,000 accepted transactions, the following counts match across all three variants, clock modes and repetitions:

| Work | Concrete | Certified |
|---|---:|---:|
| Integer arithmetic | 1,472,742 | 0 |
| Predicates | 1,591,054 | 0 |
| Expressions | 7,843,650 | 0 |
| Output comparisons | 1,387,500 | 0 |
| Binding checks | 462,500 | 462,500 |
| Frame checks | 200,000 | 200,000 |
| Domain checks | 462,500 | 925,000 |
| Certificate uses | 0 | 462,500 |

Certified lookup executes 1,472,742 contract integer operations and 1,591,054 predicates as **application work**, separately from boundary counters. Concrete lookup coalesces application evaluation with checking. Learned proposals execute neural math, integer decoding and numerical sanity checks. Zero warm contract evaluation is not zero binding/frame/domain cost.

Successful startup separately performs 5,949 integer contract operations, 7,185 predicates and 2,772 output comparisons over the 924 primitive inputs. Failed candidates record only their actually attempted certification prefix, stop at the first counterexample and execute no warm certified calls. Load/signature, seal/build, concrete checking and lookup application costs are retained in per-trial reports and measurement records.

## Evidence and verification

All 18 signed accumulator artifacts, raw predictions, reports, first failures and example responses are retained under trials/. Shared source observations, training examples, Counter predictions, workload, requests and oracle traces live once under common/. The runner verifies byte equality before sharing; per-trial evidence-refs.json records relative paths and digests. No failed run is omitted or overwritten.

Trial workers run with only binary, two signed modules, root-pinned manifest and public key. Comparison processes additionally receive oracle-observation data for validation before timing, not as model inputs. The binaries import no source parser/compiler, source implementation or reference oracle. Source-derived specifications remain inside artifacts; source independence is not OS sandboxing. Public signing keys are research fixtures, and certificates remain process-local results rebuilt against the current engine.

| Evidence | SHA-256 |
|---|---|
| Selected seed-7 module | `7778ca1c136a0c060a081f207276c11a53db8f571e4c8bf9223f72d808b45d52` |
| Prior seed-404 module | `aa4a59ecf32103fc5376eda9de69b5f475dc1f98c7ac0c9ca7ab69d3b6fe2f8e` |
| Shared Counter module | `51d2bf0bf344142e094d64b25b2c28151589c0e1348b6e251aa420162e0918a3` |
| Shared oracle observations | `08dcbdc0bb48ddca2de07244ceb63a63dc9ed2b994c56866ad40185c0a14c003` |
| Profile binary | `1d06eaf5356f1317776f9244b40e477569bece7995105c63dac984da144b6df5` |
| Selected raw predictions | `f4186baeff63b6087929a8a0e189e8f8c4453b929cd389b5e5697a56105aefe3` |

Toolchain: go version go1.26.4 linux/amd64. Training implementation/source hashes, all binary digests and candidate lineage are recorded in selection.json and trial reports. Builds use -trimpath -buildvcs=false. No Go inference, optimizer, compiler or checker changes were needed for this sweep. Older models and reports remain unchanged.

go test -count=1 ./... and go vet ./... passed. Existing observer/gradient, initializer lineage, seeded determinism, exhaustive source-oracle, source-free reload, ablation, raw-score, zero-allocation prediction, frame/binding, clock-equivalence and atomic rollback tests remain. Every sweep candidate additionally completes the real frozen-worker/source-oracle/lookup comparison. Two pre-existing dynamic-language differential tests remain skipped.

These are fixed-grid outcomes on one finite source family. Seed 404 was a known positive control, not a fresh random success; only seed 7 adds a new success. Certification data also serves selection, and there is no independent seed-selection holdout or probability estimate. Failed runs miss training inputs as well as nontraining inputs, so failure is not a capacity proof or an isolated generalization result. The findings support some transfer across initializations while showing that convergence is still fragile under the current fixed setup.
