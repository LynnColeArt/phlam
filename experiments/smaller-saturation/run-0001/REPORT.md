# PHLAME smaller saturation: recorded run 0001

Recorded 2026-10-25. [Method and reproduction](../README.md) · [Candidate selection](selection.json) · [Matched measurements](measurements.json) · [Grouped results](summary.json).

## Result

The 8-unit residual ReLU accumulator fully certifies with **79 parameters instead of 143** (44.8% fewer). It is freshly trained, frozen, signed and reloaded without source. All 102 training inputs and all **492 nontraining primitive inputs** decode correctly, covering both saturation regions, exact thresholds, interior updates and reads. All **224 supported compositions** match source execution and lookup; **526 outside-domain schedules roll back**.

In a matched comparison on the existing optimized runtime, successful certified transactions without phase clocks have median process means of **1.656 µs** for learned-8, **1.894 µs** for the frozen learned-16 control and **1.547 µs** for lookup. Smaller learned execution has 12.6% lower median latency than the larger model in this run. Lookup remains faster, and process variation is visible. All three have the same allocation counts and exact checker work.

Widths 2 and 4 were also trained and frozen. Both fail finite certification and remain in the evidence set. Selection uses the full certification domain; the 492 nontraining inputs are not an independent holdout for architecture selection. No minimum possible network size or broad performance claim follows from these three fixed-seed runs.

## Fixed candidate protocol

Before training the survey, candidate widths were fixed at 2, 4 and 8. Each uses the original 102 observations: even states in [-16,16], add arguments divisible by four in [-8,8], and read examples at those states. Seed 404, 24,000 full-batch Adam epochs, feature scale 32, input/target definitions, initialization rule and learning-rate schedule remain unchanged. Width changes the number of random draws and the parameter shapes; it is not pruning or distillation of the original model. No other seeds or optimizer settings were tried.

Every candidate trains twice to check bit-identical reproduction, freezes with the public research signature, reloads in a source-free worker and attempts the same full-domain certification. An exploratory survey used these same settings and obtained the same weights and accuracy; after tests were completed, the recorded run regenerated all three candidates. There was no tuning between the exploratory and recorded runs.

| Hidden units | Parameters | Training correct / 102 | Nontraining correct / 492 | Final MSE | Certified | Signed artifact bytes |
|---|---:|---:|---:|---:|---|---:|
| 2 | 31 | 60 | 310 | 0.000648839297 | no | 3,434 |
| 4 | 47 | 60 | 310 | 0.000648839297 | no | 3,759 |
| 8 | 79 | 102 | 492 | 1.54925763e-12 | yes | 4,389 |
| 16, frozen control | 143 | 102 | 492 | 7.91374624e-13 | yes | 5,730 |

Widths 2 and 4 miss 42 training inputs each, so their failure is not solely a nontraining generalization failure. Their incomplete fits do not prove those widths cannot represent the intended function. Under this fixed optimizer/seed, the two-unit run certifies some prefix before failing at add(state=-15, argument=-1); the four-unit run fails immediately at add(state=-16, argument=-8). Their signed artifacts do not bypass the checker.

In concrete mode each failed model correctly completes 105 of the 224 supported schedules and rejects the other 119; it publishes no incorrect result. In certified mode each fails preflight and executes **zero transactions**. Rejection counts in candidate reports group oracle categories, not necessarily the worker’s first rejection reason. Bad predictions can reject before a later outside-domain step. No break-even estimate is reported when certification fails.

| Nontraining behavior | Width 2 correct | Width 4 correct | Width 8 correct | Cases |
|---|---:|---:|---:|---:|
| lower saturation | 30 | 8 | 30 | 30 |
| upper saturation | 8 | 30 | 30 | 30 |
| threshold | 0 | 0 | 12 | 12 |
| interior | 262 | 262 | 404 | 404 |
| read | 10 | 10 | 16 | 16 |

The selected model has maximum absolute nontraining next-state/return score error of **0.000153334 integer units**, below the 0.5 rounding threshold. Its raw floating-point outputs differ from the 16-unit model; only decoded integer state, return and write behavior is required to match. Training took 481.305 ms in this run; no matched training-speed claim against the historical 16-unit run is made. The same-data affine baseline remains at 200/492 nontraining inputs correct.

The selected signed artifact is 4,389 bytes versus 5,730 (23.4% smaller). The JSON contains contracts, lineage and signature overhead as well as weights, so total artifact reduction is smaller than parameter reduction. The model has five base features, eight learned ReLU units and three linear output heads: 5×8 + 3×(5+8) = 79 weights. There are no branch IDs, clamp primitives or peer features in inference.

## Source-free recertification and rollback

The parent compiler/source oracle reobserves all 594 accumulator and 330 Counter primitive inputs and checks them against source-derived contracts and the independently stated fixture mathematics. Only the selected 102 accumulator observations enter numerical training. All raw observations, prediction scores, training samples and hashes are saved separately under each candidate directory.

Each trained candidate is loaded from a signed artifact in a process without source, compiler or oracle imports. Every successful startup separately seals and checks all 594 accumulator and 330 Counter inputs against the current frozen predictor. The new module root gets its own certificate; no certificate from the larger model is accepted as authority. Admission retains input intervals, not transition outputs. Every admitted call still runs the numerical model.

All 750 supplied schedules remain untrained compositions. They exercise both directions of typed prior-result passing and feedback. The source oracle accepts all 750; 224 stay within every declared primitive input domain and match in all four primary engine/boundary combinations. The remaining 526 leave a declared input domain and publish original states/versions, with no application results or writes. They are scope restrictions, not native source rejections.

[Selected request examples](width-8/request.json), [source traces](width-8/oracle-traces.json) and [source-free results](width-8/example-results.json) preserve upper/lower saturation, a threshold and failure after both components stage writes. In the last example, Counter 0 reaches 1, accumulator 16 successfully writes 16 after adding 1, and passing that return to Counter exceeds its argument domain. Both states and versions roll back. Debug source traces intentionally retain attempted values; the application outcome does not publish staged results.

Zero weights, shuffled heads and removal of the nonlinear layer still fail certification in regression tests. Invalid bindings, root swaps and altered frames remain covered by the existing tests. The ordinary source-free worker and the inference/transaction/checker implementations are unchanged; this slice exposes bounded training width and fixes break-even reporting for failed preflight.

## Matched runtime comparison

One current profiling binary executes all three variants. Learned-8 loads the selected accumulator; learned-16 loads the historical frozen accumulator byte-for-byte; lookup uses the selected contract specification. The same newly frozen Counter file is used for every variant, and its numerical weights, contract, domain and training data match the original Counter control. This isolates accumulator model size from runtime changes or peer models.

The runner verifies that accumulator contracts, domains, training hashes, oracle-observation hashes, seeds and epochs match between the smaller and larger models. Transaction observations are byte-identical to the original study. Signature validation, finite certification and full outcome comparison happen before every warm measurement. All processes produce the same semantic outcome digest, including rejection reason/step, states, versions, ordered writes and typed returns.

Values below are medians of five fresh-process means with phase clocks off. Outer latency includes counter aggregation and retaining the last result in a sink. JSON I/O, startup, validation and warm-up are excluded. These numbers should not be compared directly with earlier full-batch-retention worker timings.

| Certified workload | Learned-8 ns | Learned-16 ns | Lookup ns | Allocations/tx, all three | Bytes/tx, all three |
|---|---:|---:|---:|---:|---:|
| accepted | 1655.6 | 1893.7 | 1547.3 | 5.312 | 828.1 |
| mixed | 1372.6 | 1442.7 | 1305.6 | 5.683 | 1063.1 |
| scope rejected | 1053.3 | 1101.5 | 977.2 | 5.840 | 1163.1 |

| Successful certified execution, clocks off | Median ns | Minimum process mean | Maximum process mean |
|---|---:|---:|---:|
| learned-8 | 1655.6 | 1594.0 | 2250.6 |
| learned-16 | 1893.7 | 1808.7 | 1925.0 |
| lookup | 1547.3 | 1410.4 | 1638.5 |

The smaller model narrows the median gap to lookup from 346.4 to 108.4 ns per successful transaction in this run. Distributions overlap and the smaller-model group has a visible high sample; no statistical-significance or broad speed claim is made. No timing run was repeated to obtain a better result. The mixed group contains 224 successes and 526 domain rejections, and the rejection group includes both early and late failures.

[CPU diagnostics](learned-8.cpu.txt) attribute 16.48% inclusive sampled CPU time to Model.Predict for learned-8, compared with [28.48% for learned-16](learned-16.cpu.txt). These are shares of different diagnostic processes, not an additive latency decomposition. [Allocation sampling](learned-8.allocs.txt) shows no inference allocation; instance snapshots and result/write structures remain the shared allocation cost. Smaller weight count does not alter those containers.

## Separate checker measurements

For 100,000 successful transactions, exact counts match across all three variants, clock modes and repeated processes. Model size does not reduce or remove any exact obligation.

| Checker work | Concrete | Certified |
|---|---:|---:|
| Integer arithmetic | 1,472,742 | 0 |
| Predicates | 1,591,054 | 0 |
| Expressions | 7,843,650 | 0 |
| Expected-output comparisons | 1,387,500 | 0 |
| Binding checks | 462,500 | 462,500 |
| Frame checks | 200,000 | 200,000 |
| Domain checks | 462,500 | 925,000 |
| Certificate uses | 0 | 462,500 |

Certified lookup executes 1,472,742 integer arithmetic operations and 1,591,054 predicates as **application work**, separately from checker counters. Learned proposals run floating-point neural math and decoding. Concrete lookup coalesces its application computation with checking. Zero warm contract evaluation is not zero boundary cost: typed binding, frame ownership, domain admission and certificate use remain.

| Successful certified execution, clocks on | Binding ns | Proposal ns | Checker ns | Commit ns | Overhead ns | Total phase ns |
|---|---:|---:|---:|---:|---:|---:|
| learned-8 | 267.2 | 614.9 | 393.5 | 132.7 | 1519.9 | 2932.1 |
| learned-16 | 272.5 | 876.2 | 399.4 | 140.8 | 1551.5 | 3244.2 |
| lookup | 270.0 | 403.8 | 400.6 | 136.9 | 1531.9 | 2738.3 |

Phase cells are medians of per-process means and need not add exactly. Outer latency additionally includes loop aggregation. Clocked totals are not used as clock-free timing estimates. Full concrete and clocked measurements remain in summary.json and measurements.json.

At startup each successful process rebuilds both certificates with 5,949 integer arithmetic operations, 7,185 predicates and 2,772 output comparisons over the 924 inputs. Load/signature and seal/certificate timings are recorded per process, outside warm latency. Failed candidates stop certification at their first counterexample.

| Successful-group unclocked processes, median startup | Load/signature ms | Seal/certify ms |
|---|---:|---:|
| learned-8 | 0.770 | 0.402 |
| learned-16 | 0.782 | 0.452 |
| lookup | 0.791 | 0.438 |

## Lineage, validation and limits

180 fresh-process measurements cover three variants × two boundaries × two clock modes × three workload groups × five repetitions. Order is shuffled with seed 20261025; each process uses GOMAXPROCS=1 and measures 100,000 transactions after ten group warm-up passes and a GC. Three separate diagnostic processes sample CPU and cumulative allocations over 1.5 million accepted certified transactions each, excluded from the latency comparison. Profile allocation samples also include startup/validation/warm-up; normalized allocation counts come from unprofiled runs.

The comparison directory contains only the profiling binary, two signed accumulator models, one common Counter, two root-pinned manifests, the public research key and transaction-observation data. Diagnostic runs additionally write profiles. No source implementation, training dataset, compiler or oracle is available to the worker. Expected values are only checked before timing and never supplied to inference. Contract specifications intentionally remain in modules. This is source independence, not an OS sandbox.

| Evidence | SHA-256 |
|---|---|
| Selected module payload | `1da6d69e0d46a1bf3b47530c08b7f0395cdb4e8d8365e770bf9de67386f29241` |
| Larger module payload | `dc7f4157157e1c655d207c45335f45f70870043bd22dcfdf0d5ba9af25c2d021` |
| Common Counter payload | `b8282781122ac63e901033b30e8a886ad4125c1ff9133e203ebedbaffe707e9d` |
| Oracle transaction observations | `08dcbdc0bb48ddca2de07244ceb63a63dc9ed2b994c56866ad40185c0a14c003` |
| Profile binary | `15e22ddde4ae533e06427a70e0b69fc7cd209a6d5f9efa27978e8263aa57a877` |
| Shared accumulator training data | `99c5e078f2c4d6ddcda49105f67d2afb66dee6ce4ceedd25190e667078b69124` |
| Shared primitive source observations | `e60409c995fa4000adabaec91ba64fe76859d40c81236f4969bd8192cc5240e2` |
| Selected raw predictions | `177e696bab8fdede2a6af15e0863aac429a60218afc69207379490377226027c` |
| Training implementation | `952c29ac924cb8bf3a886d2366856cd277801b9872d64a2c64daf402a693a7fe` |

Toolchain: go version go1.26.4 linux/amd64. All three binary digests and candidate lineages are in selection.json; model-file digests and manifests are described by metadata.json. Builds use -trimpath -buildvcs=false. Artifacts and public research signatures are frozen before runtime checking; certificates remain process-local results rebuilt against the current engine. Older reports and frozen models are retained unchanged.

go test -count=1 ./... and go vet ./... passed. New tests cover bounded widths, deterministic retraining, rejected candidate certification and the selected 8-unit certificate. Both 8- and 16-unit models run the exhaustive source-oracle, all-composition, ablation, source-free reload and atomic rollback tests. Existing raw-score preservation, zero-allocation inference, clock equivalence, frame/binding and tamper tests remain. Two pre-existing dynamic-language differential tests remain skipped.

This survey uses one fixed training seed and a fixed optimizer budget. Architecture selection uses all nontraining certification inputs; no independent architecture-selection test set is claimed. The source family and integer domain are small, schedules are supplied, and a low-width training failure is not an impossibility result. Five process means have visible noise; no CPU pinning or confidence intervals are used. The baseline is the existing exact relation interpreter, not a specialized handwritten clamp or a precomputed table. Correctness outside the finite domain, learned planning and general symbolic branch discovery remain unproven.
