# Counter experiment 0001: observed result

Recorded 2026-10-25; go1.26.4, linux/amd64; AMD Ryzen 5 5600X host.
Worker GOMAXPROCS=1. This is the first instrumented bounded numerical experiment,
not validation of a general learned language runtime.

## Outcome

Trained **12 scalar parameters** on **170 observed transitions**.
The reloaded model's rounded proposals matched the oracle on **all 4,216
held-out cases**, before concrete checking. The structural lookup baseline also
matched all cases. Zeroed and shuffled weights produced no correct held-out
proposals; all were rejected without changing state. No incorrect proposal was
accepted. There was no oracle fallback in the worker.

This establishes a source-independent learned numerical pathway in the declared
finite domain. It establishes neither unseen program composition nor a practical
advantage over exact lookup. The exact checker still recomputes the Counter
relationship. The trivial baseline already solves the task, and the learned path
was slightly slower in this measurement.

## Data and generalization

The complete domain contains 4,386 cases: states -64..64, reads without delta,
and increments with delta -16..16. Splits are assigned before trace collection.
Train: states -32..32 in steps of four, increment deltas -8..8 in steps of two,
plus reads at those states (170 total). All remaining state/argument tuples are
held out, with no tuple shared across splits.

Held out: 1,000 interpolation cases and 3,216 extrapolation cases. These are numeric
splits of **one program**, not held-out source templates, control flow or operation
compositions. State and delta are engineered features; reads encode delta=0.
No test case or label is fed to the optimizer. No model/hyperparameter search is
performed by the experiment runner.

Labels come from actual constructed/invoked reference instances, are compared
with independent math/big sums, and carry observed trace hashes. The source hash
binds labels and contracts to the Counter program. The training data and split
membership are saved beside this report. Full raw traces are regenerable from
the source and scenarios; training.json stores their hashes, not their contents.

## Accuracy (unique held-out cases)

| Engine | Raw candidates correct | Accepted | Accepted wrong | Rejected |
| --- | ---: | ---: | ---: | ---: |
| learned | 4,216/4,216 | 4,216 | 0 | 0 |
| lookup | 4,216/4,216 | 4,216 | 0 | 0 |
| zero | 0/4,216 | 0 | 0 | 4,216 |
| shuffled | 0/4,216 | 0 | 0 | 4,216 |

Both learned and lookup reach 100% on interpolation and extrapolation separately.
Ablations are fixed zero/permuted weights, not retrained alternatives. These show
that the learned path actually depends on its weights; they do not show that
learning is necessary to solve Counter.

## Training, freezing and loading

- Full-batch gradient descent: 8,000 epochs, learning rate 0.3, seed 20261025.
- Initial normalized MSE 0.37635796785; final 1.29049652403e-27.
- First training run: 56.372 ms; oracle data generation: 311.930 ms.
- Second same-input training produced bit-identical weights in this environment.
  Cross-toolchain/architecture reproducibility is not established.
- Frozen artifact: 3,018 bytes, including weights, exact contracts and lineage.
- Fresh-process learned artifact load: 0.525 ms;
  87,512 bytes allocated during load. This excludes
  process startup/key loading; filesystem cache is warm. Allocation is not RSS.
- Worker directory contained only `artifact.json`, `trusted-key.hex`, `worker`.
  No source, AST, labels or oracle executable was copied there. A dependency test
  rejects a runtime/reference import in the worker.

The artifact is signed using an explicitly public, reproducible **research key**.
This is an integrity-mechanics fixture, not a production trust identity. Invocation
never updates weights. The version is its signed content, not the chmod bit.

## Timings (nanoseconds per transition)

Learned/lookup each have 42,160 timed runs: 4,216 cases × 10 repetitions. Stage
counters and clocks are inside the timed path. Means, medians and p95 are empirical
microbenchmark observations, without an efficiency claim.

| Stage | Learned mean | Learned p50 | Learned p95 | Lookup mean | Lookup p50 | Lookup p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| proposal | 55.6 | 60 | 60 | 44.0 | 40 | 50 |
| candidate | 149.4 | 120 | 200 | 138.9 | 110 | 190 |
| checker | 490.7 | 450 | 500 | 445.7 | 440 | 480 |
| commit | 22.5 | 20 | 30 | 22.7 | 20 | 30 |
| overhead | 211.8 | 210 | 230 | 242.9 | 210 | 220 |
| total | 930.0 | 860 | 970 | 894.3 | 830 | 930 |

The concrete checker accounts for about
52.8%
of measured learned total time, versus
6.0%
for inference. The local commit is simply publication of the checked candidate
state/version; there is no durable storage, authorization or external effect.

Oracle invoke mean: 4.955 µs,
p50 3.490 µs,
p95 10.410 µs.
This includes reference heap snapshots, trace recording, method dispatch and its
own concrete check, and excludes parse/construction. Different machinery means
this is **not an apples-to-apples speed comparison** with the transition worker.
Structural lookup is the stronger practical comparison here.

## Exact checker work

Across the 42,160 learned runs (41,040 increments, 1,120 reads), the measured
checker performed:

- 493,600 relation-term evaluations.
- 164,160 exact integer additions:
  **four per increment**, zero per read. The checker evaluates the addition in
  checked steps, the ordered write, final state and return independently.
- 41,040 argument checks,
  84,320 field checks and
  41,040 ordered write checks.

Lookup incurred the same checker work. This accounting makes clear that the
boundary repeats the whole application arithmetic; inference is not replacing
that deterministic work. There is no symbolic proof cache or general equivalence
proof. Incorrect concrete predictions reject rather than get repaired.

## Lineage and reproduction

- Source SHA-256: `fd26a7823ae1fd39b10f99a48ca42c99208acf36e598c7c46cd9b69ea0450d9e`
- Training file SHA-256: `20b2bdfab486878e6ce4cae682ef28ef670929393dff378804bd32280da2344b`
- Split file SHA-256: `ee39c2b64e022e3ad89705ed4bf07956c56e11f8d23f72e14fe00ee457e24404`
- Frozen artifact file SHA-256: `7073fa5e197930d01a17e806f89a6603aefb2efa08c44e70bb28b5b513297b07`
- Implementation manifest SHA-256: `5cc94faace7aaadcbcde2521d236ea60c385c432851803493544474a22972df3`
- Worker binary SHA-256: `b0e05c79976d5f7c6169bd1cf90570f523d14b4c831f5902dd9d29f639b5a28c`

Implementation digest covers sorted paths/hashes of go.mod and Go files under
cmd, compiler, runtime and semantics. The worker binary hash identifies the measured build; Git VCS metadata or a
different build environment can change rebuilt bytes. The binary is not committed. Timings and allocation can
vary across runs. [Reproduction commands](../README.md) generate a new versioned
result directory rather than overwrite this recorded artifact.

## Remaining hypothesis

The numerical portion passed its bounded accuracy/source-free/weight-dependence
tests. There is no retrieval gain, unseen composition, trained capability discovery
or reduced exact-checker burden demonstrated. Before extending this architecture,
test a task with genuinely new compositions and measure whether the exact
boundary can do less work without weakening correctness. A negative result there
must remain a negative result rather than become an index VM redesign by stealth.
