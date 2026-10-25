# Slice 0002: exact Counter execution and contract boundary

Date: 2026-10-25. Scope: integer Counter reference execution, observed traces and
operational contracts. No learned-execution milestone is claimed.

## What runs

`phlam run <source> <scenario.json>` constructs and invokes real objects. Counter
supports new/init, per-instance state, shared object identity on assignment,
parameter defaults, exact int64 arithmetic, reads and increments. A source-level
harness also exercises object assignment and nested calls. The sample scenario
returns 12 through both aliases and 0 from an independent instance.

Public calls use a local working heap snapshot. Failures discard all nested
writes/allocations; successful writes advance object versions once per outer
transaction. Init cannot be called as an ordinary method through the runtime API.
Execution errors include node IDs, source spans and active call stacks.

`phlam contracts <source>` derives operational relations from the AST for the
straight-line integer specialization. It records parameter defaults, reads,
ordered writes, checked intermediate operations, frame preservation, final state
and returns. Unsupported analysis stays unknown, including its effects. Class
purpose text remains declared intent.

The standalone concrete checker rejects wrong outputs, hidden/extra/reordered
writes, altered protected fields and overflow. The oracle checks supported
contracts before publication. This is a deterministic relation-evaluation
baseline; it is not symbolic verification of model-generated programs.

Observed traces include transaction/call causality, source provenance, exact
primitive work, staged writes, contract checks, commits and rollback. Integer
capture is explicitly opt-in for synthetic examples. Local result/state reports
still include values; no production redaction claim is made.

## Verification

- `go test ./...` and `go vet ./...` pass, including the existing syntax corpus.
- Reference tests cover defaults, alias identity in both source and host APIs,
  independent instances, read-only versions, runtime init rejection, bad arguments,
  failed construction, overflow, nested transaction rollback, strict branches,
  short-circuiting, undefined handles/methods and step/depth budgets.
- Contracts tests reject wrong delta/return, missing/extra fields and arguments,
  changed frame slots, same-value writes during read, reordered/intermediate
  writes and overflow in an otherwise unused expression. Declared purpose cannot
  override actual writes; unsupported/oversized inference remains unknown.
- Trace tests check deterministic output, source-node membership, transaction and
  parent-call IDs, actual arithmetic/write observations, defensive snapshot copies,
  opt-in capture and absence of commits in a failed transaction.
- CLI tests cover contracts, successful scenarios, deterministic reports, overflow
  diagnostics with retained state, malformed JSON and invalid harness steps.
- Arithmetic fuzzing: 108,960 cases compared with independent `math/big` arithmetic;
  Counter transition fuzzing: 24,102 cases compared with independent exact sums.
  Both bounded two-second runs completed without a failure. These are samples,
  not universal correctness proofs.
- Built CLI runs the Counter sample and emits contracts; the overflow sample
  returns exit 1 and retains its maximum integer value at version 1.

Commands to reproduce are in [reference-execution.md](reference-execution.md).
Local detailed logs and regenerated JSON reports are in ignored `work/`.

## Evidence and limitations

Implementation: `semantics`, `compiler/contracts`, `runtime/reference`,
`runtime/trace`, and `cmd/phlam/scenario.go`. Runnable fixtures are under
`examples/counter`; domain and trace details are in the execution guide.

The checker requires trusted compiler-derived contracts, complete before/after
state, normalized arguments and actual ordered write observations. No signed
contract loader or untrusted proposal commit API exists. A contract check is per
concrete input and performs application arithmetic itself; its cost must be
counted in future learned-runtime experiments. It does not authorize invocation,
prove a proposal equivalent for every input, or supply learned execution.

The oracle is single-threaded, in-memory and source-dependent. Snapshot rollback
covers only local state. There are no external effect adapters, tenant policies,
model weights, trained artifacts, proof caches, training splits or data export
pipeline. Dynamic variables, reference-cell aliases and eval remain three
explicitly skipped runtime/differential obligations. This does not complete
Phase 2 or Phase 3.

## Next boundary to test

Freeze the integer transition vocabulary and use these observed, externally tested
reference runs to define training labels and held-out splits. Compare a bounded
learned proposer with trivial and graph/lookup baselines. Reject bad proposals
before state publication and account separately for learned inference, concrete
checking, exact primitives and commit work. Symbolic term validation remains a
separate design obligation; a failed learned path must not silently call this
reference evaluator and count as success. Remaining language semantics can then
expand the oracle without changing that architectural distinction.
