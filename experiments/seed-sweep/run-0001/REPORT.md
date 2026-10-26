# PHLAME bounded seed sweep: recorded run 0001

Recorded 2026-10-25. [Method and reproduction](../README.md) · [All trial results](selection.json) · [Control measurements](summary.json).

## Result

**None of the 18 low-width configurations fully certified.** Widths 2 and 4 each used seeds 0,1,2,3,4,5,6,7,404, with the previous seed included as a control. Every model was trained, frozen, signed, reloaded without source and subjected to the unchanged full-domain certification requirement. Every failure is retained.

The best nontraining accuracy is **310/492** for either width. All candidates also miss training inputs; the best training accuracy is **63/102**. This bounded search did not resolve training under the fixed setup. It does not prove that 2- or 4-unit networks cannot represent saturation. The existing certified 8-unit model remains the smallest established success in these experiments.

Failed certification executes **zero transactions** for every candidate. Concrete checking permits only completely correct schedules and safely rejects the rest, without substituting expected outputs. No incorrect accepted result or state leak was observed. Lookup correctly completes all 224 supported schedules in every trial; another 526 source-valid schedules leave the chosen input domains and roll back.

## Fixed protocol and complete results

The width/seed grid was fixed before the sweep. Source, integer domains, the 102 training observations, 24,000 full-batch Adam epochs, learning-rate schedule, initialization distribution, decoding and exact boundary remain unchanged. Counter retains independent seed 101 and separate examples. There is no additional seed search, optimizer tuning, domain narrowing or inference change. Each recorded configuration trains twice to verify bit-identical weights.

An exploratory pass covered the same eight new seeds at each width before tests and the recorded sweep. The recorded models reproduce those weights; seed 404 reproduces the earlier failed controls. No extra seeds or budgets were introduced after seeing failures. Candidate selection would take the first full success per width in the declared order, without considering latency. Both selected_by_width entries are null.

| Width | Seed | Training correct / 102 | Nontraining correct / 492 | Correct concrete compositions / 224 | First failed add (state,arg) | Startup cases attempted |
|---:|---:|---:|---:|---:|---|---:|
| 2 | 0 | 46 | 184 | 72 | (-16,-8) | 1 |
| 2 | 1 | 60 | 310 | 105 | (-16,-8) | 1 |
| 2 | 2 | 40 | 200 | 68 | (-16,-8) | 1 |
| 2 | 3 | 40 | 200 | 68 | (-16,-8) | 1 |
| 2 | 4 | 60 | 310 | 105 | (-16,-8) | 1 |
| 2 | 5 | 63 | 284 | 126 | (-16,0) | 9 |
| 2 | 6 | 60 | 310 | 105 | (-15,-1) | 25 |
| 2 | 7 | 40 | 200 | 68 | (-16,-8) | 1 |
| 2 | 404 | 60 | 310 | 105 | (-15,-1) | 25 |
| 4 | 0 | 63 | 285 | 126 | (-16,0) | 9 |
| 4 | 1 | 60 | 310 | 105 | (-16,-8) | 1 |
| 4 | 2 | 40 | 200 | 68 | (-16,-8) | 1 |
| 4 | 3 | 60 | 310 | 105 | (-15,-1) | 25 |
| 4 | 4 | 40 | 200 | 68 | (-16,-8) | 1 |
| 4 | 5 | 60 | 310 | 105 | (-15,-1) | 25 |
| 4 | 6 | 63 | 284 | 126 | (-16,0) | 9 |
| 4 | 7 | 63 | 284 | 126 | (-16,-8) | 1 |
| 4 | 404 | 60 | 310 | 105 | (-16,-8) | 1 |

Every candidate has zero certified transaction executions and zero wrong concrete acceptances. Concrete completion varies from 68 to 126 supported schedules. Raw accuracy and composition coverage need not rank candidates identically; neither is used to relax the full-domain requirement. Every lookup run completes 224 supported schedules and rejects all 526 scope cases. Rejection columns in per-trial reports classify source-oracle categories, not necessarily the first runtime rejection reason.

Only certification-prefix work is performed on a failing model: startup stops at the first counterexample and issues no certificate. This is the unchanged full-domain requirement, not an attempt to certify a smaller admitted subdomain. Exhaustive raw scores and decoded outcomes on **all 594 primitive inputs** are retained separately for every candidate. Successful full certification would require all 594 accumulator and all 330 Counter inputs to pass.

## Exact boundary work and atomicity

The failed startups exhibit three measured prefix lengths. Values below are for the accumulator; Counter certification is not reached because sorted registry order encounters the failing accumulator first. These are actual measured operations, not estimates of an unperformed full certification.

| Attempted inputs | Contract expressions | Integer arithmetic | Predicates | Output comparisons |
|---:|---:|---:|---:|---:|
| 1 | 31 | 4 | 10 | 3 |
| 9 | 296 | 41 | 93 | 27 |
| 25 | 945 | 150 | 280 | 75 |

Each trial’s report.json retains startup build/signature times, prefix checker counters, concrete checker work, lookup application work and phase measurements for learned/lookup/zero/shuffled engines. Failed certified engines have no warm transaction work. No learned break-even estimate is calculated for failed preflight.

Concrete mode compares each numerical proposal with its exact source-derived relation before staging it. A wrong later proposal discards earlier correct writes and returns an owned original state snapshot, with no published returns or writes. The parent checks every accepted state, version, typed return and ordered write against the source oracle, and checks original-state rollback on every rejection. There is no repair or exact-output fallback.

The source oracle accepts all 750 schedules; 224 remain within every declared primitive input domain and 526 do not. Thus scope rejection is not source-language rejection. The supplied schedules are not training examples and are not learned plans. All class/frame bindings, typed value passing, range checks and atomic publication remain unchanged.

## Operational controls, not low-width speed results

Because no sweep model fully certifies, only the unchanged frozen 8-unit accumulator and lookup enter the successful-execution timing comparison. A common newly frozen Counter is used for both; its weights, contract and training data match the earlier control. Each process reloads signatures, reconstructs finite certificates and verifies all 750 outcomes against the recorded source oracle before timing. All outcome hashes agree across engines, boundaries and clock modes.

120 fresh processes cover two controls × two boundaries × two clock modes × three groups × five repetitions. Each measures 100,000 transactions after validation, ten warm-up group passes and a GC, with GOMAXPROCS=1 and shuffled order seed 20261025. Two separate CPU/allocation diagnostic processes measure 1.5 million accepted certified transactions each and do not enter the latency summaries.

| Successful certified control, clocks off | Median process mean ns | Minimum | Maximum | Allocations/tx | Bytes/tx |
|---|---:|---:|---:|---:|---:|
| learned-8-control | 1651.4 | 1638.3 | 1733.6 | 5.312 | 828.1 |
| lookup | 1439.0 | 1406.9 | 1514.1 | 5.312 | 828.1 |

These are control results on already certified software, not a successful-execution speed comparison for the rejected low-width models. Outer timing includes counter aggregation and retaining the last result in a sink; startup, validation, JSON and full-batch retention are excluded. There is no CPU pinning, confidence interval or broad performance claim.

| Successful certified controls, clocks on | Proposal ns | Checker ns | Binding ns | Commit ns | Overhead ns | Total phase ns |
|---|---:|---:|---:|---:|---:|---:|
| learned-8-control | 604.9 | 383.7 | 262.4 | 146.8 | 1504.4 | 2894.9 |
| lookup | 424.7 | 402.9 | 279.9 | 143.9 | 1594.5 | 2854.9 |

Phase entries are medians of per-process means and need not add exactly. Clocked timings are separate from unclocked results. Profile files and text summaries remain available for both controls.

For 100,000 accepted transactions the following exact checker counts are identical across both controls, clock modes and repeated processes:

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

Certified lookup executes the relation’s integer arithmetic and predicates as separately counted **application work**. Concrete lookup coalesces that computation with checking. Learned control proposals use neural math and decoding. Every successful startup separately checks 924 inputs with 5,949 contract integer operations, 7,185 predicates and 2,772 output comparisons; startup is outside the warm loop. Zero warm contract work does not mean free binding, domain and frame enforcement.

## Source independence and evidence

Each trial first runs the ordinary worker in a directory containing only binary, two signed modules, root-pinned registry and public key. The process imports neither source parser/compiler nor reference oracle. Control profiling additionally receives oracle-observation data, used only to validate before timing and never supplied to prediction. Source-derived contract specifications intentionally travel with modules. This is source independence, not OS isolation.

Every trials/wN-sS/ directory retains the signed model, its Counter peer, key, registry, full raw accumulator predictions, report and example response. Shared source observations/training data, oracle workload, Counter predictions, source traces and request are stored once under common/. Before sharing, the runner compares files byte-for-byte; each trial’s evidence-refs.json records relative paths and SHA-256 digests. Distinct failed model evidence is never discarded.

[common/oracle-traces.json](common/oracle-traces.json) retains actual source events for upper/lower saturation, a threshold and a later domain-failure schedule. Source execution accepts those schedules; failed low-width certified example responses contain preflight_error and no outcomes. Existing source-free tests also exercise the certified control’s saturation-then-domain-failure rollback after both components stage writes. Debug traces expose attempted values separately from application results.

| Evidence | SHA-256 |
|---|---|
| Control accumulator module | `1da6d69e0d46a1bf3b47530c08b7f0395cdb4e8d8365e770bf9de67386f29241` |
| Common Counter module | `9e37928d456da3247d13a2ac13882ccd707264d878859d4c5b1e2f8c2f73c9a0` |
| Oracle transaction observations | `08dcbdc0bb48ddca2de07244ceb63a63dc9ed2b994c56866ad40185c0a14c003` |
| Profiling binary | `15e22ddde4ae533e06427a70e0b69fc7cd209a6d5f9efa27978e8263aa57a877` |
| profiler binary | `15e22ddde4ae533e06427a70e0b69fc7cd209a6d5f9efa27978e8263aa57a877` |
| trainer binary | `a34686c85c63b0de0db6cb6455b4fdff9db449d8f785c7aa2e7e4900b9964d04` |
| worker binary | `5dce5d157467e10c771b87d80a2df342e60f0c5946d8db5a9589106242445cfe` |

Toolchain: go version go1.26.4 linux/amd64. Full candidate source/training/implementation digests and module roots are recorded in selection.json and per-trial report.json. Builds use -trimpath -buildvcs=false. Public reproducible signing keys are research fixtures; certificates are process-local results rebuilt against the current engine. Old models and recorded reports remain unchanged.

## Validation and interpretation

go test -count=1 ./... and go vet ./... passed. Added tests check low-width seed reproducibility, independent initialization and inability to borrow the prior certified model’s certificate. Existing bounded-width, exhaustive source-oracle, source-free reload, raw-score, zero-allocation prediction, frame/binding, clock equivalence and atomic rollback tests remain. Two pre-existing dynamic-language differential tests remain skipped.

This is a fixed finite family and bounded initialization search. All runs fail some training inputs, so the outcome does not isolate nontraining generalization and does not establish a minimum network capacity. Full-domain certification would also be seed-selection data; no independent seed-selection holdout is claimed. These observations support investigating the training dynamics or initialization before inferring that a larger model is mathematically necessary. No such follow-up training change is included in this sweep.
