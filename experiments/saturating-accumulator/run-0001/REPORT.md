# PHLAME learned saturation: recorded run 0001

Recorded 2026-10-25. [Experiment and reproduction](../README.md) · [Machine report](report.json).

## Result

The frozen accumulator predicts all **492/492 held-out primitive inputs** correctly before any checker repairs or rejects them. It trains on 102 source-observed examples, with 143 learned parameters. Both saturation branches and exact thresholds are covered. The same-data affine baseline gets 200/492 correct.

Counter trains independently on 30 examples. Both modules freeze separately, reload in a source-free child process and compose through typed prior-result bindings. All **224 supported schedules match the source oracle and lookup**, including ordered writes and one version increment per written instance. Another **526 source-valid schedules leave the declared domains and roll back**. There are no native false-return rejections in this saturating fixture. No incorrect transition is published.

Certified warm checking evaluates zero contract expressions, arithmetic, predicates or expected-output comparisons. Learned certified mean total time improves relative to concrete checking in this run, but remains slower than lookup. This is a finite semantic result, not an efficiency victory over the baseline.

## Training and raw primitive predictions

Accumulator input domain: state [-16,16], add argument [-8,8], plus read at every state. There are 561 add inputs and 33 reads. Training selects even states and arguments divisible by four: 85 adds and 17 reads, totaling 102/594 primitive inputs. No compositions, peer state, source paths or branch labels enter training. Labels in the following table are assigned afterward from observed inputs.

| Held-out behavior | Correct | Cases |
|---|---:|---:|
| lower saturation | 30 | 30 |
| upper saturation | 30 | 30 |
| threshold | 12 | 12 |
| interior | 404 | 404 |
| read | 16 | 16 |

All 102/102 training inputs also decode correctly. Maximum absolute held-out next-state/return score error after scaling to integer units: **0.000124429**, below the 0.5 rounding threshold. These floating-point scores are approximate; exactness refers to decoded integer state/return/write behavior. Raw scores and decoded outputs are in [accumulator-predictions.jsonl](accumulator-predictions.jsonl), separately from [source observations](accumulator-primitive.jsonl) and [training samples](accumulator-training.json).

The generic residual network uses five inputs (bias, state/32, argument/32, own add/read one-hot), 16 learned ReLU units, and three linear output heads over base inputs plus hidden activations. The 143 weights include all hidden and output parameters. Seed 404; full-batch Adam, 24,000 epochs; rate 0.003 for 18,000 epochs then 0.0003; beta values 0.9/0.999; epsilon 1e-8. Initial MSE 0.339120438404, final MSE 7.91374623839e-13; measured training 808.109 ms. Same-seed retraining is bit-identical. The affine baseline uses the same samples and seed, and the earlier fixed 12,000-epoch gradient-descent trainer.

One initial exploratory run used this same architecture, seed, sampling and training budget. Its accuracy and weights matched this recorded run. No parameter search, held-out stopping or certification-guided retraining was performed. The Counter retains its independently trained 15-parameter affine architecture and has 300/300 held-out primitive inputs correct.

## Composition and rollback

750 supplied schedules combine five initial values per instance, five Counter deltas and six deterministic patterns, including both directions of return-value passing and feedback. No composition is a training example and no schedule is learned. Supported coverage is 224/750; unsupported inputs reject without clipping or extrapolating. The source oracle accepts all 750 schedules, so the 526 scope rejections must not be reported as source rejections.

| Engine / boundary | Correct accepted | Scope rejected | Supported rejected | Wrong accepted |
|---|---:|---:|---:|---:|
| learned/concrete | 224 | 526 | 0 | 0 |
| learned/certified | 224 | 526 | 0 | 0 |
| lookup/concrete | 224 | 526 | 0 | 0 |
| lookup/certified | 224 | 526 | 0 | 0 |
| zero/concrete | 0 | 526 | 224 | 0 |
| shuffled/concrete | 0 | 526 | 224 | 0 |

Rejection columns group transactions by the source oracle's category, not by the worker's first rejection reason. Corrupted models can fail before reaching a later outside-domain call.

Accepted compositions include 21 lower-saturation calls, 21 upper-saturation calls, 28 exact-threshold calls and 209 interior adds. Counts are calls, not disjoint transaction categories.

Zero and shuffled-head engines cannot certify: each fails on add(state=-16, argument=-8) before any transaction executes. Shuffling preserves hidden weights but rotates output-head/feature weights. Removing the nonlinear layer also fails certification in regression tests. The concrete checker rejects bad proposals without substituting its expected output.

[request.json](request.json) and [example-results.json](example-results.json) demonstrate upper saturation, lower saturation, an exact threshold and rollback after saturation. [oracle-traces.json](oracle-traces.json) contains actual source events with tentative values and source identities. In the rollback example, Counter 0 increments to 1; accumulator 16 consumes 1 and successfully writes 16; passing that return to Counter fails its argument domain. Both states and versions return to their initial values, with no application results or writes published. The source oracle accepts that schedule; its trace illustrates the successful source computation, while the learned worker reports a domain rejection and atomic rollback.

## Exact boundary work

Each frozen module is signed, loaded, deeply sealed and independently certified at startup against its exact source-derived path relations. The certificate binds module root, engine and ABI and cannot be supplied by a caller. Per-state argument admission intervals retain no predicted outputs. Every admitted runtime call still runs the numerical predictor. No warm clamp implementation supplies learned outputs.

| Learned startup module | Cases | Integer arithmetic | Predicates | Build ms |
|---|---:|---:|---:|---:|
| counter | 330 | 1188 | 0 | 0.091 |
| accumulator | 594 | 4761 | 7185 | 0.632 |

Combined one-time certification work is 5,949 integer operations, with measured build time 0.723 ms. Timings include registry sealing and validation. Load/signature verification is separately recorded as load_ns.

| Learned warm checker, 7,500 transactions | Concrete | Certified |
|---|---:|---:|
| Expressions | 290,930 | 0 |
| Integer arithmetic | 57,110 | 0 |
| Predicates | 55,410 | 0 |
| Output comparisons | 51,420 | 0 |
| Binding checks | 22,400 | 22,400 |
| Frame checks | 15,000 | 15,000 |
| Domain checks | 22,400 | 39,540 |
| Certificate uses | 0 | 17,140 |

Certified admission still performs integer range comparisons and metadata checks; zero contract work does not mean zero boundary work. Neural multiply/add/ReLU, rounding and numerical sanity checks belong to proposal work. Lookup application arithmetic is recorded separately; concrete lookup reuses its relation evaluation for both its result and checking rather than doing duplicate work.

## Timing

All values below are mean ns per transaction across the same 750-case mix repeated ten times. Measurements include instrumentation, successful and rejected schedules. Each engine/boundary uses a separate source-free child process with GOMAXPROCS=1; no CPU pinning, randomized process order or confidence intervals. Phase timings can vary with allocation and GC.

| Engine / boundary | Binding | Proposal | Checker | Commit | Overhead | Total |
|---|---:|---:|---:|---:|---:|---:|
| learned/concrete | 273.5 | 1006.0 | 503.9 | 55.5 | 2105.9 | 3944.8 |
| learned/certified | 238.1 | 948.9 | 302.4 | 52.7 | 1717.7 | 3259.9 |
| lookup/concrete | 244.4 | 56.4 | 456.3 | 62.4 | 2130.5 | 2950.1 |
| lookup/certified | 247.4 | 225.6 | 386.0 | 61.1 | 1916.6 | 2836.7 |

Learned checker mean falls from 503.9 to 302.4 ns. Total mean falls from 3.945 to 3.260 µs; lookup certified total is 2.837 µs. The estimate charging the full startup build against this observed learned total saving breaks even after 1,057 transactions. This workload-specific estimate is not a robust performance guarantee and does not imply a win over lookup.

The source oracle mean is 29.520 µs, including different heap, invocation and trace machinery, excluding parse/construction. It is not a matched speed baseline. Machine data retains p50/p95 and separate application-work counters.

## Lineage and reproduction

Toolchain: go1.26.4; platform: linux/amd64. The source-free directory contains exactly: accumulator.json, counter.json, registry.json, trusted-key.hex, worker. No source files, training data, parser, reference oracle or old Counter runtime are available to the worker. This is source independence, not OS sandboxing; source-derived specifications are intentionally included in signed modules.

| Evidence | SHA-256 |
|---|---|
| Implementation | `14b164c539b7a6d5d10ee45a039a5c586a6bc371e3fe8fef2ae417c602beca25` |
| Worker binary | `5137a27406ddf084f98e47ab9a99c3bf9cf62d65bd2187b899471643a2f388f9` |
| Combined oracle source | `7b8bbb072f159c0ce838ecf17320b921fd87d7510a77b8f5a63e39bfdc229de4` |
| Transaction observations | `08dcbdc0bb48ddca2de07244ceb63a63dc9ed2b994c56866ad40185c0a14c003` |
| accumulator module | `dc7f4157157e1c655d207c45335f45f70870043bd22dcfdf0d5ba9af25c2d021` |
| accumulator source | `8c75c3cb73c83d670a26266fca83f94d6f73b844b2c4063e1c74aab17e9cffbb` |
| accumulator training | `99c5e078f2c4d6ddcda49105f67d2afb66dee6ce4ceedd25190e667078b69124` |
| accumulator primitive observations | `e60409c995fa4000adabaec91ba64fe76859d40c81236f4969bd8192cc5240e2` |
| accumulator raw predictions | `fa996219871bc3caa2f6f52fb6c8191b34d4d917afff40408008ac3dcefe4e10` |
| counter module | `1be91c575b054ee2be476bf41c6369c2e49ec7d41dbc31c659f8b4166b6c76ce` |
| counter source | `fd26a7823ae1fd39b10f99a48ca42c99208acf36e598c7c46cd9b69ea0450d9e` |
| counter training | `e986fd51e7146963dbcfa36c82d96d5a4f8d0e8f4e583f20ad0ffd4a1057b490` |
| counter primitive observations | `ce8b76024e2f5717b67ae6dcbed635f48f5dabf0c08b2091c52e4c6c5b1d7c11` |
| counter raw predictions | `d1de4886c8b250a2f0426018ab823f1f49260bcd4a5aebc2e871ee43239399b1` |

The implementation digest covers sorted Go sources/tests under cmd, compiler, runtime and semantics plus go.mod. The binary digest identifies the measured build; VCS metadata may change binary bytes on a later rebuild. Existing reports and frozen artifacts were retained unchanged. Module sizes: Counter 1,938 bytes, accumulator 5,730 bytes. Public reproducible research signing keys demonstrate integrity mechanics, not production identity or authorization.

## Validation and limits

go test -count=1 ./... and go vet ./... passed. Regression tests cover exhaustive source-derived saturation paths against an independent mathematical specification, raw held-out accuracy, deterministic training, all schedules in all four primary engine/boundary combinations, same-value saturation writes, rollback after two staged components, ablation failure, hidden-layer validation/sealing and source-free signed reload. Existing tamper, root-swap, invalid binding and frame-isolation tests remain. Two pre-existing dynamic-variable/alias and eval differential tests remain skipped.

This is a single finite integer family and fixed training seed. Learning the numerical piecewise function does not establish symbolic branch discovery, a learned scheduler, arbitrary program compilation or generalization outside the enumerated domain. Certificates are rebuilt process-locally; no portable proof verifier or external durable effect system is claimed. Scalar instance-owned slots do not establish shared-cell alias semantics. Learned certified latency remains worse than the lookup baseline.
