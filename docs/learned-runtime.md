# Learned runtime: first experiment contract

Status: bounded numerical, composition, independent-component, nonlinear
saturation, profiling, size/seed and initialization experiments implemented,
2026-10-26. The newest [paired-initialization sweep](../experiments/paired-seed-sweep/run-0001/REPORT.md)
tests seeds 0–7 and 404 at widths 2/4. Seeds 7 and 404 certify with four units;
none certify with two. The new seed-7 artifact has 47 parameters and preserves
all 594 primitives, 224 supported compositions and 526 domain rollbacks.
Convergence remains fragile, and no minimum possible width is established.
Warm checking still evaluates zero contract arithmetic/predicates. Matched
runtime ranges overlap and establish no speed improvement. Learned planning
and unbounded equivalence remain unproven.

The [broad-feasibility plan](broad-feasibility-plan.md) defines the remaining
acceptance profile, training/proof budgets, richer program corpus and systems
gates. Bounded execution is established; broad compiler/application feasibility
and full-brief research feasibility remain unresolved. Its first implementation
slice is the [general build pipeline](broad-feasibility-next-slice.md).

The [G01 pipeline](../experiments/g01/README.md) is now implemented and accepted
for the existing scalar subset: generic locked builds, reproducible signed
bundles, independent source-free execution, retained candidate failures and
separate signed-entry/scalar-checker measurements. Later gates remain open.

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

## Transaction profiling

The subsequent [profiling study](../experiments/transaction-profile/README.md)
reuses the same frozen modules and oracle-observation data, without retraining.
A source-free profiling process verifies clocked/unclocked outcomes before each
measurement, with separate successful and domain-rejected schedule groups.
CPU and allocation profiles are diagnostic runs; randomized fresh-process
measurements report outer latency, allocation counts, phase clocks and checker
work separately.

Call-local bounded prediction scratch eliminates numerical feature allocations;
publishing the working copy on success eliminates a duplicate original snapshot.
Rejected transactions still discard tentative state and return an owned original
snapshot. All previously recorded raw scores remain bit-identical and all checks
remain active. RunUninstrumented is an opt-in profiling API that omits timestamps,
retaining exact work counters and semantics; the ordinary worker retains Run.

Learned allocation counts now match lookup for this workload, but successful
learned execution remains slower. The remaining gap is predominantly numerical
prediction/decoding. Shared instance-map/output allocation and substantial
phase-clock overhead also remain visible. Finite certificate construction is
reported outside the warm loop; no weaker boundary or exact-result substitution
is used to improve timings.

## Smaller learned artifacts

The [fixed-width survey](../experiments/smaller-saturation/README.md) changes only
training width, retaining the source, 102 training examples, seed and optimizer
budget. Every width freezes before source-free certification. Widths 2 and 4
miss training and nontraining cases and fail startup; width 8 certifies with 79
parameters, compared with 143 for the original 16-unit model. It preserves all
594 primitive transitions and all supported supplied compositions.

The smallest fully certified candidate is selected using the full finite domain;
nontraining accuracy is not an independent architecture-selection holdout. These
fixed-seed failures establish no minimum representational capacity. The retained
failed artifacts show that signatures and smaller models do not bypass exact
certification. Concrete checking rejects their incorrect proposals rather than
repairing outputs; certified failure executes no transactions.

One current source-free binary and one common frozen Counter compare the smaller
model, historical larger model and lookup in randomized fresh-process order.
Runtime semantics, inference, checker logic and allocation counts are unchanged.
The smaller model reduces median successful latency but lookup remains faster
in this run. Startup certificate construction, warm checker work and numerical
proposal costs remain separate. Atomic rollback and owned outputs still apply.

## Bounded seed search

The seed sweep fixes widths 2/4, seeds 0–7 and 404, and the existing 102-example,
24,000-epoch protocol before running. Each signed model reloads source-free and
must satisfy the unchanged full finite domain; first-counterexample failure
prevents every certified transaction. Every candidate fails, including training
inputs, so the result concerns this training setup rather than a proven capacity
limit. All raw predictions and certificate prefix work are retained.

Exact concrete checking still admits only correct complete schedules and rolls
back failed proposals. Lookup and the frozen 8-unit model remain operational
controls, with separate startup, checker and application measurements. No failed
candidate's short rejection path is presented as a successful-execution speedup.
The domain and checker remain unchanged. Candidate selection uses certification
inputs and claims no independent seed-selection test set.

## Training dynamics and initialization

At fixed seed 404, passive checkpoints reproduce historical 2-, 4- and 8-unit
weights exactly. Legacy low-width failures lose useful nonlinear boundaries;
being dead at initialization is not the sole explanation. An offline audit
recomputes activation counts, MSE and gradients and checks legacy initialization
derivatives by finite differences. It does not feed back into training.

Two bounded initialization interventions retain training examples and joint
optimizer settings. Centered opposing random directions certify with four units
and no extra optimization budget. An affine-prefit variant also certifies four
units but explicitly adds 12,000 linear fitting epochs before joint training.
Both two-unit treatments fail. The selected first certifying treatment is frozen
and independently recertified source-free, with full source/lookup comparisons,
atomic rollback and separate checker measurements. Initializer/prefit provenance
is signed metadata; runtime inference is unchanged. Selection uses certification
data and demonstrates no universal optimizer or capacity result.

## Centered paired initialization across seeds

A fixed nine-seed sweep repeats the centered, opposing-direction initializer
without affine prefit, extra epochs or altered examples. Four-unit seeds 7 and
404 fully certify; the eight new seeds contribute one new success, while 404
reproduces the known model. No two-unit seed passes. This demonstrates an
additional successful finite realization without establishing general optimizer
reliability or a minimum capacity. Every failure remains archived.

The first certifying seed per width is selected before timing and compared with
the prior frozen four-unit control and lookup. Full recertification, source-free
execution, oracle comparison, rollback and separate exact work counters remain.
Nontraining certification inputs also serve seed selection; runtime differences
have overlapping ranges and are not presented as a seed-based speed improvement.

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
