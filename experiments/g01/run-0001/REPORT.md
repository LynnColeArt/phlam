# G01 acceptance: reusable scalar source-to-frozen-artifact pipeline

Recorded 2026-10-26. **Verdict: G01 passed for the existing scalar subset.**

[Commands and reproduction](../README.md) · [Locked campaign](../campaign.json) · [Machine report](report.json) · [Validation](validation.json) · [Format ADR](../../../docs/decisions/0003-g01-scalar-build-bundles.md).

## What now works

One reusable authoring API and `phlam build` command consume explicit source/class/domain declarations, a locked training plan and a configured development signing key. They derive contracts, observe source execution, select training examples, run bounded generic candidates, freeze each produced model, ask the standalone runtime to recertify it, and publish a signed bundle only on success. There are no Counter/Accumulator/SaturatingAccumulator arithmetic branches in the builder or runtime. Independent fixture mathematics remains outside the builder.

The three projects exercise the same three source programs in original, class/field-renamed and method-reordered forms. Each is built twice from separate clean input/output directories with the same binaries/key/recipe; all frozen deployment files are byte-identical. These are nine supported source/class cases, not nine new program families or sealed broad-feasibility acceptance programs.

## Locked inputs and deterministic selection

The campaign was recorded before implementation. Strict schema decoding rejects unknown fields and trailing documents. Versioned JSON is used in phlam.mod/phlam.lock and training.json, per ADR 0003. The lock must cover the exact explicit source/plan set; components cannot load unlisted/escaping sources. Each source has one class and one owned scalar slot in this gate.

All programs share the same recipe: affine seed 7, then paired-centered ReLU width 4 seeds 7/404, then width 8 seeds 7/404. Sampling uses state stride 2 and argument stride 4, admitted source observations only. The case cap is 5,000 per component, candidate training/verification wall timeout 15 seconds, whole-build timeout 180 seconds. No application formula, per-class callback, manual weights or application-specific sampling is supplied to the builder.

| Project | Component | Oracle cases | Training rows | Selected candidate index | Attempts retained |
|---|---|---:|---:|---:|---:|
| original | accumulator | 594 | 90 | 0 | 1 |
| original | counter | 330 | 68 | 0 | 1 |
| original | saturating | 594 | 102 | 1 | 2 |
| renamed | accumulator | 594 | 90 | 0 | 1 |
| renamed | counter | 330 | 68 | 0 | 1 |
| renamed | saturating | 594 | 102 | 1 | 2 |
| reordered | accumulator | 594 | 90 | 0 | 1 |
| reordered | counter | 330 | 68 | 0 | 1 |
| reordered | saturating | 594 | 102 | 2 | 3 |

Counter and the rejecting accumulator choose affine models. Saturation first rejects affine prediction, then selects a learned ReLU candidate. Method reordering changes operation indices and training traversal; its first width-4 fit fails, and the same generic bounded policy proceeds to the next seed. Failure and retry costs remain in the evidence instead of being hidden by a successful build.

Each observed primitive is actual source execution with source-node trace provenance. The compiler-derived relation is compared with that observation, and acceptance separately verifies the intended fixture mathematics. Two real trace samples per source (one per operation) are retained in observations.json. The accumulator learns accepted numerical updates; false-return rejection remains exact contract/admission behavior. Saturation is a learned nonlinear transition. G01 adds no new model-family claim.

## Frozen deployment and execution contract

Deployment has exactly four files: three signed model artifacts and signed bundle.json. The bundle binds module identity/version, aliases, filenames, module roots, ABI, explicit-int64 argument policy and exported methods. Loading requires an externally configured public key; a bundle does not nominate its own trust root. A valid model signature alone neither changes entry bindings nor proves semantics.

Every runtime startup reconstructs finite certificates on the current engine for all 1,518 primitive inputs across the three modules. No persisted report or verification bit is accepted as a certificate. Source/training data, optimizer, private key and build evidence are absent from deployment. The runtime dependency graph excludes compiler/reference packages; its binary symbol table contains no linked optimizer routine. The authoring/training tools are separate binaries.

The ABI intentionally preserves explicit method arguments: one argument requires a literal or prior successful result; source defaults are not inserted silently. Source false-as-rejection aborts the enclosing research transaction. Instance value/version maps remain the existing local scalar ABI, not authenticated persistent handles. Network/storage capabilities, tenant policy and durable transactions are not introduced.

## Source-oracle, independent fixture and lookup comparisons

Each project executes 1,518 exhaustive primitive scenarios plus eight additional transaction/negative scenarios, for **1,526** cases. Learned/lookup, concrete/certified and clocks on/off all have identical accepted states, typed results, versions and ordered writes. Five timing repetitions per mode are also checked for semantic equality.

| Project | Accepted | Rejected | Two clean bundles identical | Source-free working-directory files |
|---|---:|---:|---|---|
| original | 1447 | 79 | yes | accumulator.json, bundle.json, counter.json, runtime, saturating.json, trusted-key.hex |
| renamed | 1447 | 79 | yes | accumulator.json, bundle.json, counter.json, runtime, saturating.json, trusted-key.hex |
| reordered | 1447 | 79 | yes | accumulator.json, bundle.json, counter.json, runtime, saturating.json, trusted-key.hex |

The rejected cases are 72 native rejecting-accumulator primitive inputs plus seven additional negatives. The additional success passes Counter’s return into the saturating accumulator and commits both versions together, including a same-value saturation write. The late-domain failure stages a Counter write and a saturation write, then fails on the third call and restores both original states/versions with no published results/writes. Native rejection after a prior write, invalid/forward references, missing arguments, unknown entries and domain escape are also checked.

The harness checks source observations against independent arithmetic fixtures; accepted source comparisons are not inferred merely from matching two learned implementations. The test suite independently covers snapshot ownership, source name/order changes and signed entry narrowing. The deployment runtime never consumes fixture expectations; expected.json belongs to the external acceptance harness.

## Failed builds and interrupted publication

| Recorded negative build | Exit code | Deployment activation |
|---|---:|---|
| failed-candidate | 1 | absent |
| invalid-domain | 1 | absent |
| lock-mismatch | 1 | absent |
| training-timeout | 1 | absent |
| undeclared-source | 1 | absent |
| unknown-recipe | 1 | absent |
| unsupported-source | 1 | absent |

The failed-candidate campaign preserves signed produced models and raw scores even though no complete bundle is published. Timeout failures retain seeds, inputs, error and actual child wall/CPU cost; a killed optimizer cannot supply unavailable intermediate weights. Unsupported eval source produces a B_UNSUPPORTED diagnostic with source span. Invalid lock/domain/declaration/recipe failures are explicit.

Tests additionally cancel an actively running training subprocess and verify no output activation and no remaining handled staging directory. Preexisting outputs/evidence are refused. Files stage next to the output and one directory rename is the publication point after final report writing. This assumes a trusted, single-writer development filesystem; it is not hostile-filesystem race protection, fsync crash durability or production deployment coordination. A hard kill may leave an inactive temporary stage, never an accepted output bundle.

Signed-manifest/model mutation, wrong keys, root/ABI mismatches, validly signed wrong numerical models and unexported entries are rejected in tests. The initial tamper test failed because its attempted mutation of a read-only fixture was not checked; the test now explicitly enables fixture writing, checks mutation success and passes against the signature guard. That was a fixture-write error, not evidence that altered signed bytes were accepted. The first failing test transcript is not counted as validation; the final full-suite transcript is retained.

## Separate checker and execution measurements

Timings below are mean nanoseconds per case, including the primitive-heavy request mix and rejections over five repetitions. They are instrumented diagnostic measurements in fixed mode order, not matched performance claims. The runner retains batch outcomes; JSON I/O is outside these transaction phase clocks. Signed-entry checking has its own exact work counters and clock separate from the scalar transaction checker/total. Startup load/signature and certificate build are separate fields.

| Project / engine / boundary | Scalar checker ns | Scalar total ns | Entry checks ns | Warm contract integer operations |
|---|---:|---:|---:|---:|
| original/learned/concrete | 307.8 | 1551.2 | 137.7 | 50890 |
| original/learned/certified | 155.0 | 1361.6 | 138.8 | 0 |
| original/lookup/concrete | 325.1 | 1420.5 | 129.0 | 50890 |
| original/lookup/certified | 144.7 | 1411.9 | 126.4 | 0 |
| renamed/learned/concrete | 295.7 | 1561.6 | 133.9 | 50890 |
| renamed/learned/certified | 154.2 | 1413.8 | 145.3 | 0 |
| renamed/lookup/concrete | 300.3 | 1434.1 | 125.5 | 50890 |
| renamed/lookup/certified | 146.9 | 1341.1 | 129.0 | 0 |
| reordered/learned/concrete | 280.6 | 1440.3 | 142.3 | 50890 |
| reordered/learned/certified | 149.1 | 1381.9 | 135.1 | 0 |
| reordered/lookup/concrete | 316.8 | 1353.1 | 202.8 | 50890 |
| reordered/lookup/certified | 157.0 | 1561.6 | 158.0 | 0 |

All certified modes record zero warm contract expressions, integer arithmetic, predicates and expected-output comparisons. This is not zero boundary work: entry, frame, binding, range and certificate checks remain. Lookup’s contract interpretation is charged separately as application work in certified mode; concrete lookup coalesces its application evaluation with checking. Numerical inference/decoding and score sanity checks remain proposal work.

Each attempt records training wall/CPU time, final Go managed bytes, numerical loss, raw source-agreement coverage, signature/load time, finite certificate work/build time and the verification child’s wall/CPU time. Training is killed by per-candidate wall timeout; the Go GC soft memory limit and final managed-bytes metric are not an OS RSS cap or memory sandbox. Broader CPU/RSS/practicality gates remain pending.

## Reproduction, verdict and remaining work

Build evidence is separate from deployment and contains locked inputs, observations/real traces, training rows, all produced candidates and build reports. Frozen identity includes source/contracts/domain, training/oracle data hashes, seed/initialization/prefit, ABI/toolchain and the actual builder/trainer/runtime binary fingerprint. Timing data is outside frozen identity. Deployment byte hashes and semantic outcome digests for both clean builds are recorded in report.json.

| Tool | SHA-256 |
|---|---|
| phlam | `c100199456dead3e1df8301b5def03d345f1032ff7f689809b7d7f5c3a24b824` |
| phlam-runtime | `f00973f3d2f198526da425822961a5b79e1955db31d20073ef3ab323520ffc1d` |
| phlam-train | `2a43d5986ebbc2d5de281cad06456c97ab7a77c07fee21c1e158604036b9d74e` |

The final full suite has 129 passing test events; go vet ./... passes. Two pre-existing dynamic-variable/alias and eval tests remain skipped for G06, not misreported as G01 implementations. JSON test events and their digest are retained in validation.json/tests.jsonl. New tests cover all G01 input/build/runtime/negative/publication obligations listed above.

**G01 passes**: one reusable pipeline builds all three supported scalar sources and renaming/reordering variants, publishes reproducible source-free bundles, retains failures, validates independently and preserves exact transactional semantics and separated work measurements. The campaign requires no application-specific builder or runtime arithmetic.

**Broad feasibility remains unresolved.** This gate does not prove reliable convergence across fresh seeds/program families, non-enumerating certification, richer state/control flow, dynamic language semantics, production authority, durable storage, learned discovery or direct authorship. G02 training reliability and G03 semantic/ABI design are the next dependency-ready gates; no later gate is marked passed by this report.
