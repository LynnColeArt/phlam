# Learned runtime: first experiment contract

Status: bounded numerical, composition, independent-component and nonlinear
saturation experiments implemented, 2026-10-25. The newest
[saturation experiment](../experiments/saturating-accumulator/run-0001/REPORT.md)
trains a residual ReLU executor on 102 observed primitives and predicts all 492
held-out accumulator inputs exactly after integer decoding. Counter trains
independently; 224 supported cross-component schedules match the oracle and
526 outside-domain schedules roll back. Startup certification removes warm
contract arithmetic and predicates. This run improves learned latency relative
to concrete checking; lookup remains faster. Learned planning, richer data/control
flow and unbounded symbolic equivalence remain unproven. The broader target follows.

## Architectural destination

Source -> frontend -> semantic contracts + reference traces -> training -> frozen
learned application + exact bindings -> independent runtime invocation.

The learned artifact is the primary application representation. AST/graph IR is
permitted for compilation, the reference evaluator, training, and baseline
measurements. Shipping that graph as the actual executor with a vector lookup
front end would not satisfy PHLAM's goal. The same learned representation should
ultimately support capability discovery; discovery alone does not prove learned
execution.

## First bounded family

Start with Counter; extend to a bounded counter with increment, decrement, read,
and a guarded update with a cap. Use exact signed 64-bit integers and explicit
nonoverflow preconditions. No network, persistence, aliases, or eval in the first
learned experiment. Those remain required language research, not implicitly
solved by this restricted family.

Model input: an authorized stable operation ID, typed arguments, a permitted
state view, and the artifact version. Never use an embedding as object identity.
State handles and versions come from the exact runtime. The model proposes a
small typed operation term with explicit slot reads, arithmetic, comparisons,
conditional expressions, staged writes, and return values. Initially, terms are
pure until an atomic local-state commit. A proposal is data, not host code.

A candidate might compose `read_slot(value)`, `add`, `write_slot(value)`, and
`return` into a complete increment transition. The primitive vocabulary is
fixed; complete execution paths must not just be predefined graph edges. Hold
out combinations of state/arguments and, separately, operation compositions.
Counter alone can establish plumbing, not strong generalization.

## Who does what

| Component | Responsibility | Must not silently do |
| --- | --- | --- |
| Frontend/analysis | Derive exact operational semantics, provenance, known/unknown effects | Treat class purpose prose as a proven contract |
| Reference evaluator | Execute precise semantics, produce observed training labels and test oracle results | Run as a hidden fallback in learned benchmark requests |
| Learned proposer | Select/compose transition terms using frozen trained parameters | Grant authority, choose arbitrary resource handles, mutate weights |
| Semantic checker | Validate a term against the requested operation's exact relation, domain and frame conditions | Accept a merely plausible result or only check types/ranges |
| Exact primitive engine | Perform integer arithmetic, slot access, branch predicates in an accepted term | Supply a missing application-level transition path |
| Effect gate | Bind authorized objects, check versions/budgets, commit staged writes once | Commit before semantic validation, trust model-supplied authority |

Example: for increment(d), checking only `result is int` or `new value >= old
value` is insufficient. The contract must require `new value = old value + d`,
`result = new value`, and preservation of all other slots, with no overflow.
Read must return the current value and leave all state unchanged. A wrong but
well-typed proposal must be rejected.

## Avoiding an oracle disguised as a validator

Current bounded implementation: a worker checks all 4,386 discrete primitive
inputs once, using its actual immutable model, shared quantizer and exact
contracts. It keeps only an in-process certificate, not an output lookup table.
Runtime bounds/shape/score guards plus ordered typed assembly compose those
verified steps by induction. No source or reference evaluator is used in this
certification. This removes warm arithmetic duplication but shifts exact work to
startup, so amortization is measured. It is finite exhaustive verification, not
a solution to unbounded program equivalence. The symbolic strategy below remains
a candidate for a broader domain.

Candidate approach: restrict terms to a small decidable fragment. For straight
line counter arithmetic, symbolically normalize integer expressions and prove
returned value, write set, and final value equivalent to the compiler-derived
relation under checked domain assumptions. Guarded operations can use explicitly
bounded branch partitions. Proof checking can happen per unique term/operation
and be cached only with the contract hash, term hash, domain, primitive semantics
version, and artifact hash. Runtime domain, authority, state-version, and resource
checks still apply every time.

This is a design candidate, not an existing equivalence prover. The implemented
concrete checker re-evaluates relations for one transition; it does not discharge
this symbolic proof obligation. Integer overflow
and branch coverage cannot be waved away by treating machine integers as reals.
Unknown predicates, unbounded loops, dynamic evaluation, or effects outside the
proven fragment must reject as unsupported. A safe rejection is an execution
failure for experiment coverage metrics. General program equivalence is not a
promise of this architecture.

Alternative baseline: evaluate the reference transition to check every proposal.
It can establish correctness on small cases, but must be reported as full oracle
reexecution, including its cost. It does not establish a useful learned runtime.
Even a successful symbolic gate encodes substantial application knowledge in
contracts. Measure and report that work; deterministic primitive execution may
dominate. No performance or novelty claim follows from the diagram alone.

## Candidate models and training inputs

Compare a small trained classifier over bounded operator terms with a small
sequence model that emits typed terms. A classifier is a plumbing baseline; a
fixed menu of whole-method paths is insufficient evidence of composition.
Nearest-neighbor embeddings are a retrieval baseline, not the target executor.
Choose a model only after the reference data format and task are fixed. No
mandatory second language or ML framework is selected in mission 1.

Inputs derive from: compiler facts labeled inferred, class purposes labeled
declared, and real reference runs labeled observed/verified against their test
oracles. Static AST visits are not execution traces. Record source hash, scenario,
seed, initial state, arguments, primitive steps, returns, state deltas, failures,
and provenance. Keep secrets and tenant data out of training.

Split before trace generation by program template and scenario group. Reserve
unseen compositions separately from unseen numeric values. Hash and save split
membership; prevent identical traces or equivalent templates leaking across
train/test. Include deliberately invalid proposals for checker tests, not as
successful execution labels.

## Frozen artifact proposal: phlam.learned.v0

A directory bundle, versioned as a whole:

- `manifest.json`: schema, model architecture, tensor dtype/shapes, primitive ABI,
  source and contract hashes, training code/data/split hashes, seed, toolchain,
  supported domain, validation report hash, and per-file SHA-256 digests.
- `weights.bin`: deterministic numeric weights with documented byte order and
  shape layout, loaded by an allowlisted model implementation. No pickle or
  arbitrary deserialization hooks.
- `contracts.json`: exact relations, argument schemas, frame/effect bounds,
  initialization/lifecycle rules, stable operation IDs and version bindings.
- `bindings.json`: allowlisted primitive/effect ABI IDs, never credentials or
  model-controlled host addresses.
- `provenance.json`: source spans, class purposes with declared labels, derivation
  links, and validation lineage. Source text and executable AST are not required.
- `validation.json`: scenarios, domain coverage, measured failures and costs.

Canonical JSON: sorted object keys, UTF-8, integers exact, no nonfinite floats;
weights have their own fixed binary representation. Hash the sorted file-name
and digest list excluding the detached signature. A deployment principal signs
that root; the loader checks signature, size limits, schema, ABIs and hashes
before allocating model storage. A signature does not prove semantic safety.
Training reproducibility means recording inputs/environment and testing repeat
builds; bitwise reproducibility is unproven until measured.

## Invocation sequence

1. Authenticate and bind operation ID, instance, tenant, artifact, argument types,
   state version, domain, and budgets through deterministic policy.
2. Run frozen model inference over only the authorized view.
3. Parse/type-check the proposal; reject unknown primitives, unbounded terms,
   arbitrary handles, and excessive size/depth.
4. Prove the transition term satisfies the exact operational contract, including
   final return, state delta, frame conditions and effect ordering.
5. Execute exact primitives against a snapshot; stage effects, recheck policy and
   state versions, commit atomically, record observed and rejected events.

No implicit retraining. No reference fallback counted as learned success.
External nontransactional effects require a later outbox/idempotency design;
local-state atomicity does not imply universal rollback.

## Independent component experiment

The scalar ABI now separates each module's model, operation metadata, source path
relations, finite domain, seed and training lineage. Models observe only own state,
argument and own operation index. They do not receive peer IDs/states or joint
training. A shared runtime routes successful int64 returns through prior-result
bindings, confines writes to the bound instance slot, and publishes all affected
frames once. Rejection aborts the full transaction, including earlier successful
primitive calls.

This is owned state plus dataflow in one transaction, not shared-cell aliases.
The compiler lowers guarded branches rather than maintaining hand-coded
Accumulator semantics in the runtime. Certificates store admission intervals for
accepted inputs, not computed transition outputs. The models learn only accepted
transitions; no learned nonlinear rejection or planner is claimed. Separate
source/contract/model comparisons and checker/application-work counters are in
the recorded report. The new worker imports no compiler or reference package.

## Saturating learned transitions

The saturation experiment retains that scalar ABI and learns a nonlinear
accepted transition rather than moving a clamp implementation into admission.
Its generic residual network has 16 learned ReLU units and three output heads.
State, argument and own operation indicators are the only inputs; source paths,
branch labels and peer state never enter training or inference. A sparse subset
of source-observed numerical transitions supplies targets. The 492 held-out
primitive cases include lower/upper saturation, exact thresholds, interior
updates and reads. The same-data affine baseline is substantially less accurate.

Every primitive is checked against real source execution and an independently
stated mathematical fixture specification. Startup exhaustively certifies the
frozen decoded model. Warm execution retains only domain admission intervals;
source relations, including clamp predicates, do not run in the certified checker.
The model learns a piecewise numerical function, not symbolic source-branch IDs
or control-flow plans. Approximate floating-point scores become exact decoded
integer transitions only inside the exhaustively checked finite domain.

Raw predictions, source traces, rollback examples and separate lookup/checker
measurements accompany the report. Existing affine models remain supported;
optional hidden weights are dimension-bounded, signed, sealed and certified
with the rest of the module. A public research key still supplies no production
tenant identity or effects authorization.

## Evaluation and falsification

Run identical scenarios against reference execution, an explicit graph/lookup
baseline, lexical/nearest-neighbor retrieval, and the learned executor. Hide
source and compiler scaffold from the reloaded learned process. Do not call the
reference evaluator from that process; compare results outside it.

Report: proposal accuracy before rejection, accepted-result exactness, rejection
and unsupported rates, completion coverage, held-out composition success, invalid
proposal acceptance, train/inference/check/primitive/commit time separately,
p50/p95 total latency, cold load, memory, artifact size, training cost and version
rebuild cost. Report cold/warm proof-cache performance separately.

Ablate learned weights (zero, random, shuffled) and replace model with trivial
selection. Verify a real pathway loses validity/coverage when learned information
is removed. Test whether contracts or a fixed term menu already solve the task.
Do not force model inference where a trivial baseline is better and call that a
gain. Preserve negative results.

Phase 3 succeeds only when a saved/reloaded frozen learned artifact contributes
to a real verified computation without source, with evidence beyond retrieval.
If the exact checker does all application work, the model merely memorizes paths,
or generalization fails, report that outcome before substituting an index VM.

## Open questions

- Which restricted term language admits useful composition and affordable proof?
- Can the same weights help both exact operation execution and intent discovery?
- How much contract specialization reconstructs a conventional executor?
- How does an eval overlay interact with frozen learned weights without retraining?
  Proposed default: explicit exact overlay mode with separate metrics; never count
  it as learned success. Do not drop caller-scope eval semantics to simplify data.
- What guarantees are feasible beyond the first bounded arithmetic family?
