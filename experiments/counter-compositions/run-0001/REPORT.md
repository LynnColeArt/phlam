# Counter compositions 0001: observed result

2026-10-25; go1.26.4, linux/amd64; AMD Ryzen 5 5600X host,
worker GOMAXPROCS=1. [Reproduction and boundary design](../README.md).

## Outcome

The unchanged model trained on **170 single-operation examples** correctly rolls
out **1,096 supported, untrained compositions** across **136 operation shapes**.
There were **zero incorrect accepted results**. Structural lookup matches the
same oracle outcomes. Another **80 of the 1,176 schedules** leave the certified
input domain and reject atomically under both boundaries; they are counted here,
not filtered away. Supported coverage is 93.20% of scheduled cases.

This is length/sequence generalization by composition of learned transitions.
Schedules are supplied by the caller. It is **not learned planning or invention
of new program structures**, and no new model weights are trained.

The certified boundary reduces warm relation additions from **130,280 to zero**
over the timed workload, while retaining domain/score guards. The cost is a
**5.479 ms** startup certificate with **17,028 exact additions**.
Measured learned checker mean falls **85.2%**. Structural lookup with the
same optimization remains faster overall; there is no learned-over-lookup win.

## Held-out schedules and oracle evidence

- Every composition has at least two operations; the only training examples have
  exactly one. The artifact and its training data are unchanged.
- All binary read/increment shapes of length 2..6 are covered; longer cases use
  read-only, increment-only, alternating, and IIR patterns at lengths 8,12,16.
- Initial values are -48,0,48; increment seeds -9,0,11, with a fixed sign reversal
  at step index mod 3 = 1. Exact duplicate schedules are removed before execution.
- Oracle results come from actual method calls in one atomic reference
  transaction, checked additionally against independent math/big sums. Ordered
  write observations and causal trace hashes are saved in observations.jsonl.
- Supported means every step's actual input lies in state -64..64, with allowed
  operation/delta values. Final outputs can leave the input domain; a subsequent
  step there cannot use this certificate.

| Sequence length | Evaluated | Supported | Learned accepted | Domain rejected |
| --- | ---: | ---: | ---: | ---: |
| 2 | 30 | 30 | 30 | 0 |
| 3 | 66 | 66 | 66 | 0 |
| 4 | 138 | 134 | 134 | 4 |
| 5 | 282 | 262 | 262 | 20 |
| 6 | 570 | 526 | 526 | 44 |
| 8 | 30 | 26 | 26 | 4 |
| 12 | 30 | 26 | 26 | 4 |
| 16 | 30 | 26 | 26 | 4 |

## Acceptance (one count per unique schedule)

| Engine/boundary | Accepted correct | Supported rejected | Outside-domain rejected | Wrong accepted |
| --- | ---: | ---: | ---: | ---: |
| learned/concrete | 1,096 | 0 | 80 | 0 |
| learned/certified | 1,096 | 0 | 80 | 0 |
| lookup/concrete | 1,096 | 0 | 80 | 0 |
| lookup/certified | 1,096 | 0 | 80 | 0 |
| zero/concrete | 8 | 1,088 | 80 | 0 |
| shuffled/concrete | 0 | 1,096 | 80 | 0 |

Both zeroed and shuffled models fail certified preflight on the first primitive
case, so no certified compositions run for those models. Their concrete mode
results above remain visible. Zero weights coincidentally solve eight zero-state,
read-only schedules; that is not evidence of learned capability. All erroneous
outputs reject; no hidden oracle fallback or output repair exists.

## Exact certificate and warm work

Each worker deep-copies/seals the artifact, predicts and concretely checks all
4,386 primitive inputs using its own compiled predictor and quantizer. Both
learned and lookup certify. The startup verification does not enumerate schedules.
A finite primitive guarantee plus actual per-step domain checks gives the
composition guarantee by induction. No transition table is kept. The model is
still executed for every step.

The certificate key binds the immutable model/contracts/artifact, engine and
composition ABI `phlam.counter.finite-domain-composition.v0`. It is an in-process constructor result,
not an externally loadable proof bit. Another process rebuilds its certificate.
Mutation of the original executor cannot change a sealed Composer.

Learned startup certificate work:

- 51,213 relation term evaluations.
- 17,028 exact additions (four per increment input).
- 4,257 argument checks,
  8,772 field checks, and
  4,257 ordered-write checks.

Over 11,760 timed schedules (1,176 × 10), warm learned work was:

| Boundary | Relation terms | Exact relation additions | Concrete checker calls | Domain checks | Score checks | Certificate step uses |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Concrete | 420,230 | 130,280 | 61,960 | 144,620 | 194,580 | 0 |
| Certified | 0 | 0 | 0 | 144,620 | 194,580 | 61,960 |

Input guards and failed partial rollouts are included in these totals. Certificate
step uses count completed certified compositions; failed prefixes contribute
domain/score/inference work but publish no certified result. The private
candidate contains one slot and ordered predictions; the trusted runtime assembles
returns/writes in requested order and publishes once. There is no external effect,
durable commit or tenant authorization claim. Failed sequences publish original
state/version and no results/writes.

## Timing and amortization

Means below are µs per scheduled composition, including the 80 domain failures.
Each primary engine has 11,760 timed invocations; ablations have one repetition.
Timers/counters add overhead. This is one local run without CPU/frequency pinning.
Per-length results, supported-only timings, p50 and p95 are in report.json.

| Stage | Learned concrete | Learned certified | Lookup concrete | Lookup certified |
| --- | ---: | ---: | ---: | ---: |
| proposal | 0.433 | 0.452 | 0.303 | 0.251 |
| candidate | 0.246 | 0.287 | 0.279 | 0.224 |
| checker | 2.743 | 0.405 | 2.574 | 0.326 |
| commit | 0.140 | 0.146 | 0.135 | 0.116 |
| overhead | 1.107 | 1.135 | 1.085 | 1.083 |
| total | 4.669 | 2.426 | 4.376 | 1.999 |

Learned checker p50/p95:
2.460/5.450 µs concrete,
0.370/0.740 µs certified.
Startup certification: learned 5.479 ms,
lookup 5.160 ms. The concrete constructor also
has preparation cost (learned 0.111 ms), included
in its startup metadata; the conservative estimates below charge the full
certified constructor cost as extra work.

Estimated learned break-even on this case mix:

- **2,344 compositions** using checker-time savings alone.
- **2,442 compositions** using measured total-time savings.
- **1,538 compositions** to amortize startup additions in arithmetic-count terms.

These estimates divide startup cost by measured average savings; they are not
service latency guarantees. A new worker pays again, and short-lived runs may
lose. Startup is reported separately from warm invocation.

Fresh-process artifact-load times (excludes key loading/process startup; warm
filesystem cache): learned concrete 0.597 ms, learned certified
0.593 ms. Oracle composition mean
19.440 µs includes
heap snapshots, per-method checks and trace recording, but excludes parsing and
construction. It is a behavioral comparator, not a matched performance benchmark.

## Source-free execution and lineage

Each worker directory contained exactly
`artifact.json`, `trusted-key.hex`, `worker`.
Requests contain supplied operation schedules, not source, AST, reference output,
training data or a proof token. A dependency test forbids runtime/reference in
the worker; the shared frontend source-span schema remains a transitive type
dependency. Original signatures still use a publicly reproducible research key,
not a production trust identity.

- Source SHA-256: `fd26a7823ae1fd39b10f99a48ca42c99208acf36e598c7c46cd9b69ea0450d9e`
- Reused artifact SHA-256: `7073fa5e197930d01a17e806f89a6603aefb2efa08c44e70bb28b5b513297b07`
- Original training SHA-256: `20b2bdfab486878e6ce4cae682ef28ef670929393dff378804bd32280da2344b`
- Observation JSONL SHA-256: `51c98c559c117dffb5dea21d17a2cbeac2b468ded4733ba0d779068b3cf82962`
- Composition implementation manifest SHA-256: `8ee363c3698547dbe8f9017ca6e193444ca5b1ed0fad6b25df1880cc15ebbe30`
- Measured worker binary SHA-256: `a465e25d23d225ef3937beca2f1d65cf08b006332a6270db4eaf03a3de9eccca`
- Learned certificate key: `e558555f1a70ea490323b45778706fbdcda1732151b2e59ae8d7e53dac45d2f7`

Implementation manifest uses sorted paths/hashes of go.mod and Go files under
cmd/compiler/runtime/semantics. Git VCS metadata and toolchain changes can change
rebuilt binary bytes; the recorded binary hash identifies this measured build.
No new learned artifact was produced or source definition changed.

## Remaining research

Warm exact arithmetic duplication is removed for this finite immutable family,
not for arbitrary programs. Verification has been moved to a measured startup
proof, with runtime guards and strict scope. Broader domains may make enumeration
infeasible and require a different proof technique. Branches, multiple objects,
resource effects, new operation selection and learned planning remain unproven.
Lookup remains equally correct and faster. The next research question is which
richer task would make learned composition useful, rather than merely executable.
