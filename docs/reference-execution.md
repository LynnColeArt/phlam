# Counter oracle, observed traces, and concrete contracts

Implemented slice, 2026-10-25. This is reference execution and a concrete contract
checker. A separate [bounded learned experiment](../experiments/counter/README.md)
now uses this oracle and checker. The oracle itself remains source-based; symbolic
equivalence checking is still unimplemented.

## Run a real scenario

```sh
go run ./cmd/phlam contracts examples/counter/counter.phlam
go run ./cmd/phlam run examples/counter/counter.phlam examples/counter/scenario.json
```

The scenario constructs Counter `a` at 10, binds `b` to the same object, constructs
an independent Counter at 0, increments through `b` by 2, and reads both objects.
Results are 12, 12, and 0. The same identity behavior also runs in PHLAM source in
`TestSourceAssignmentAndNestedCalls`, rather than relying solely on host bindings.

`phlam run` takes source plus a `phlam.scenario.v0` JSON file. This local harness
accepts `new`, `alias` (copy an object handle, not a reference-cell alias), and
`call` steps with exact integer arguments. It does not introduce top-level PHLAM
statements or replace the future entrypoint manifest. Scenario decoding rejects
unknown fields, trailing JSON documents, missing handles and invalid steps; the
harness accepts 1..1000 steps.

Output schema `phlam.reference-run.v0` includes the backend label, source hash,
step results, local bindings, committed objects/versions, and real events. Output
is deterministic for identical input bytes/options. Exit 1 on an execution error
still emits the JSON failure report, with a readable source diagnostic and call
stack on stderr. Usage errors return 2. Source `check` and AST `inspect` retain
their earlier behavior.

## Exact implemented domain

- Signed int64 literals and checked addition, subtraction, multiplication and
  negation. The minimum negative literal is accepted exactly. No numeric-string
  coercion, wraparound or float64 detour.
- Boolean/null values, strict boolean conditions, integer comparisons, typed
  equality, short circuit boolean operators, if/else and returns in the oracle.
- Fresh objects, integer literal property defaults (or null for an absent
  default), one construction-only init, positional arguments/defaults and local
  variables. Method calls get separate local frames and a read-only this binding.
- Object assignment copies identity; new allocations have independent state.
  Static-name instance calls and nested construction execute. Ordinary init calls
  are rejected at the runtime API too.
- Missing symbols/properties/handles, wrong arity, unsupported operations, invalid
  types, overflow and exhausted execution/depth budgets fail explicitly with
  source-node IDs, spans, and the active causal stack.

The initial oracle deliberately rejects string/array evaluation, dynamic names,
reference-cell aliases, eval, host/stdlib calls, division and floating point.
Parsing these constructs is not execution support. No ambient host effects exist.
This is a scoped integer experiment, not completion of Phase 2.

## Snapshot publication and failures

Each public Construct/Call/Compose copies the private local heap. Compose is an
ordered list of ordinary method invocations inside one transaction; its results
and all nested state writes publish together only on success. All nested calls share
that working snapshot. A successful invocation publishes the entire snapshot;
objects with writes advance their version once per public transaction, including
same-value writes. Reads do not advance versions. Failed init publishes no object.
A nested call may validate locally and still be discarded by a later outer
failure. Allocated IDs are never reused, even after failure.

Events report work on the snapshot; only explicit committed events claim
publication. Execution errors discard staged changes, object allocations and
version updates. There are no filesystem/network/database effects to roll back.
This single-threaded in-memory oracle has step and call-depth budgets, not a
production memory sandbox, concurrent transaction protocol, or durable storage.

## Operational contract derivation

`compiler/contracts` derives a restricted straight-line relation from method ASTs.
For Counter, contracts include exact integer argument/default information,
source/hash provenance, declared purpose, read/write sets, ordered writes,
checked intermediate operations, final-state relations and a return relation.
IDs such as `Counter.increment` are provisional operation names, version-bound
by source and contract hashes; manifests/public capability identity are pending.

The supported domain is explicitly `int64_state_and_arguments`: every declared
slot and effective argument must be int64. This specializes dynamic PHLAM methods
for the bounded experiment; it is not a claim that an initializer's source syntax
proves all future callers will pass integers. The oracle enforces that domain for
methods with these inferred contracts. In particular, Counter.init accepts integer
starting values in this slice. Other value shapes remain unsupported here.

An increment contract says:

```text
pre:    all slots/arguments are int64; each checked step does not overflow
writes: [(value, old.value + delta)]
post:   new.value = old.value + delta; every other slot is preserved
return: new.value
```

Read requires zero writes, unchanged slots and return equal to old.value. Init
relates the already allocated/defaulted state to its starting argument and has a
null result. Class purpose prose is copied under `purpose_declared`; it never
overrides inferred writes or establishes correctness by itself.

Substitution preserves the checked intermediate steps. For example, an unused
`old.value + 1` before returning old.value still has an overflow obligation.
Ordered writes catch wrong intermediate values, reordered writes, repeated writes
and writes that restore the final state. Final-state equality alone cannot do so.

Unsupported branches, calls, references, noninteger defaults, and other methods
are marked `unknown` with a reason. Unknown effects/read/write sets are null,
never empty claims of purity. Derivation caps symbolic expansion at 1024 nodes
per expression and 256 statements per method. The reference evaluator can still
execute supported oracle code with an unknown contract and records
`contract_unknown`; it does not label that transition contract-validated.

The contract digest is SHA-256 of deterministic Go JSON with its digest field
empty, bound to the source hash. This local metadata hash is not an authenticated
deployment signature or the future frozen artifact format.

## Concrete exact boundary baseline

`contracts.Validate` accepts a trusted compiler-derived contract and a concrete
transition: complete before/after state, effective arguments, return value and
ordered write observations. It rejects unknown contracts, bad domain/frames,
overflow, incorrect writes, changed protected slots and incorrect returns. It is
pure: no mutation, AST evaluator call, policy decision, or reference fallback.
The oracle checks inferred contracts before publishing state. The separate
[composition experiment](../experiments/counter-compositions/README.md) uses this
API as its external source-based comparator. Exported state,
contracts and event snapshots cannot mutate its private internal relations.

This checker evaluates operational relation terms for the current input. That is
real deterministic application work, including repeated arithmetic. It is a
concrete baseline, not a proof that a generated term is equivalent for all inputs,
and not evidence of a performance improvement. A later learned executor must
supply its own genuinely learned path and cannot count this oracle as a fallback
success. Checking model-supplied operation terms, authorization, resource binding,
proof caching and external effect ordering are still future work.

## Trace evidence and capture

`runtime/trace` stores monotonically numbered events with transaction IDs,
invocation/parent IDs, source-node spans, object IDs, state versions, operation
names and error codes. The source hash is in the enclosing report and the node
IDs. Events include actual literal/argument evaluation, state reads, arithmetic,
staged writes, selected branches, call returns, concrete contract checks, commits,
failures and rollbacks. Untaken branches produce no executed primitive events.

Integer observations are captured only when the scenario sets
`capture_integers: true` (or a local API caller enables CaptureIntegers). The
Counter samples opt in using synthetic data. Without opt-in, events retain
metadata but no integer payloads. Output results and committed state are still
present in the local execution report; this is not a production redaction policy.
Both values and metadata may be sensitive. No agent observation endpoint exists.

These are observations, not automatically verified training examples for the
whole language. Their integer relationships are checked against contracts where
supported, and tests compare expected outcomes with independent oracles. The first bounded experiment now records numeric splits, observed training
labels, frozen learned artifacts and lineage separately. General language
training and program/composition generalization remain unproven.

## Failure example

```sh
go run ./cmd/phlam run examples/counter/counter.phlam examples/counter/overflow.json
```

The example starts at 9223372036854775807 and increments by the default 1.
Execution fails with R_OVERFLOW. The report retains the original value and
version 1, records the failed call and rollback, and has no commit for that failed
transaction. `go run` reports its own wrapper exit as well; the built phlam
binary itself returns 1.

## Cross-instance value-passing oracle

The independent-component study uses `ComposeBound`: a list of bound instance
calls with literal/prior-return arguments, inside one private heap transaction.
Its research ABI can treat a source return of false as rejection of the whole
transaction. Ordinary language false returns do not themselves become exceptions.
All application results are discarded on failure, while local value-bearing
BoundObservation records preserve attempted calls for this explicitly captured
research evidence. This is not a production trace disclosure policy.

Counter and Accumulator source files train independently. The transaction oracle
concatenates those class files, so its observed nodes have a separate combined
source hash. [The component report](../experiments/components/run-0001/REPORT.md)
records both forms of lineage and the source-free comparison.
