# Initial security model

Status: required design boundaries, not implemented security guarantees. Mission 1
has a local parser, integer Counter reference oracle, observed traces and a
concrete operational contract checker. A bounded learned Counter loader now
checks artifact hashes/signatures, size/shape/domain and primitive bindings, using
an explicitly public research signing key. That is not a production trust root.
It has no production loader, symbolic learned-term verification, authentication,
external effect adapters, eval runtime or tenant isolation.

## Trust and execution boundaries

Source/class purposes, model outputs, dependency contents, business records,
discovery results and agent prompts are untrusted inputs. Source declarations
request capabilities; the deployment principal grants them. Model scores never
authorize operations. Even a signed model may produce invalid proposals.

A deployment freezes learned parameters, exact contracts, primitive ABI bindings,
and lineage into one version. Normal users and operating agents cannot replace
these. Business data and object state live separately. New training creates a new
version, validated before activation. Hashes/signatures need a configured trusted
principal; neither an untrusted self-signature nor a checksum creates trust.

The gateway authenticates callers and filters describe/discover/inspect/observe
metadata by policy. Invocation rechecks exact operation ID, argument contract,
resource, tenant, allowed effects and any deployment-required confirmation.
Object handles are opaque, scoped to owner/tenant, artifact version and lifecycle;
the model cannot supply arbitrary database identifiers as ownership evidence.
Fresh request services are constructed through new/init; shared services need
explicit lifecycle/concurrency policy. Internal member accessibility is unrelated
to agent exposure.

The semantic checker proves operational relationships, including return values,
state deltas, frame conditions, and ordered effects. Schema validation alone is
insufficient. Rejected or unsupported proposals produce errors and no state
commit. The effect layer performs policy checks and exact primitive execution
against versioned snapshots; local writes commit only after validation. Never
execute model output as a host function or process command.

## Initial threat register and planned negative tests

| Threat | Required control | Test before claiming coverage |
| --- | --- | --- |
| Cross-tenant handles or state | Bind caller/resource/version at gateway and commit | Tenant A invokes or observes B's handle: denial, no disclosure |
| Misrouted semantic tool | Exact requested ID and operational relation validation | Plausible wrong operation with valid argument shapes: reject |
| Poisoned class docs or records | Intent/evidence labels; descriptions are data | Prompt-like purpose cannot change permissions or contract |
| Tampered weights/contracts/bindings | Signed root and bounded strict loader | Flip byte, swap contract, change ABI, oversize tensor: reject |
| eval authority escape | Caller cells, narrowed capabilities, bounded overlay | Host import, nested budget escape, persistent class: reject |
| Malicious dependencies | Locked graph, content hashes, allowlisted adapters | Undeclared or hash-mismatched import: reject before execution |
| Confused deputy | Effective caller policy on every effect and bound resource | Privileged service invoked by restricted caller: denial |
| Incorrect but well-typed transition | Exact relation and frame proof | Increment by wrong delta or extra slot write: reject |
| Unsafe or repeated effect | Staging, ordering, version check, idempotency | Stale proposal/race/retry: no duplicate effect |
| Trace or training disclosure | Authorized observations, redaction, no tenant training | Secret in input cannot appear in metadata/training export |
| Resource exhaustion | Loader and inference/term/eval budgets | Huge proposal, deep nesting, infinite overlay: bounded failure |

The concrete integer tests reject wrong returns/deltas, frame mutations,
extra/reordered writes, overflow and partial state publication. Runtime init
rejection, step/depth errors and rollback are also tested. The table’s tenant,
model, loader, dependency, eval and external-effect controls remain pending.
Frozen-model tests additionally reject payload tampering, wrong keys, invalid
shapes/ABIs and out-of-domain requests; zeroed/shuffled proposals fail without
commits. These local checks are not production sandbox or gateway evidence.

## Limits that need explicit designs

Exact validation must never repair a bad proposal into a different authorized
operation silently. Proof caches are bound to contract, domain, ABI and model
versions; revocation/policy is checked fresh. Freeze prevents accidental mutation,
not model error. Do not load executable serialization formats such as pickle.

Local state atomicity does not provide distributed rollback. Network/payment
adapters need an outbox or explicit idempotency and failure/reconciliation model.
External effects remain out of the first learned arithmetic experiment. Eval
must not publish new agent-visible tools or modify signed contracts. Its explicit
exact overlay mode, if used, remains separate in learned coverage metrics.

Traces must distinguish observed facts, proposed steps, accepted proofs, rejected
requests and actual commits. Local trace capture omits integer payloads by default; synthetic examples opt
in explicitly. Local result/state reports still contain values. Neither capture
mode provides a production observation/redaction policy.
