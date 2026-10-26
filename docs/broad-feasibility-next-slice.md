# G01 next slice: reusable build and frozen execution

Date: 2026-10-26. Status: implemented and accepted for the existing scalar subset.
[Recorded G01 verdict](../experiments/g01/run-0001/REPORT.md) ·
[Commands and format](../experiments/g01/README.md).
Parent: [broad-feasibility plan](broad-feasibility-plan.md), gate G01.

## Concrete outcome

Replace the need for a new experiment driver per application with one reusable
build path for the **existing scalar source subset**. A manifest, source files,
specialization domains and a bounded training recipe produce either a frozen,
inspectable deployment bundle or an explicit build failure with retained evidence.
The bundle runs through an independent source-free process. No new numerical
model family or broad-feasibility claim is needed to pass this first slice.

Keep existing Counter/accumulator experiments and CLI behavior working. The new
builder must use generic source-derived relations and sample collection, not
class-name tests, a saturation-specific arithmetic check or application formulas
in the runtime. Independent fixture mathematics remains an external test oracle.

## Inputs and scope to lock first

- Minimal `phlam.mod`/`phlam.lock` schema: module identity, explicit local sources,
  entry methods, resolved content hashes, domains and training-plan references.
  This first schema supports local locked inputs only; network dependency
  acquisition and real authority grants remain deferred to G08.
- A versioned training-plan data schema: generic candidate model kinds/widths,
  initialization, fixed seed order, sampling rule, optimizer/proof budgets and
  deterministic semantic selection. No per-class custom Go callback or manual
  weight edits. Model/seed policy is build data, never permission or inference input.
- Current supported scalar restrictions: one declared owned slot, integer
  state/arguments/results, existing arity and effect limits, supported guarded
  source expressions and identity construction. Unsupported source receives a
  source-aware error. This slice does not imply loops/arrays/eval are implemented.
- Explicit argument/default normalization between reference, public schemas and
  worker ABI. Do not accidentally change the current explicit-argument research
  ABI by silently accepting missing arguments.
- A development trust-key input and a documented signature policy. The fixture
  key is allowed for tests; production-principal trust is still a later gate.

The implemented command contract is:

```text
phlam build <manifest> --out <new-bundle-directory> --evidence <new-evidence-directory> --signing-key <seed.hex>
phlam-runtime run <bundle> --key <trusted-public-key.hex> --request <typed-request>
phlam build-report <report>
```

The authoring CLI may import frontend/compiler/oracle code. The deployed runtime
is a separate binary/process whose dependency graph excludes those packages and
whose working directory contains no base source or training data. Do not present
the existing source-based `phlam run` oracle as artifact execution.

## Work packages

| Package | Deliverable | Meaningful acceptance check |
|---|---|---|
| G01.1 Input contract | Versioned manifest/lock/training-plan schemas, normalized ABI and campaign fixture inventory | Missing/hash-mismatched/undeclared source, invalid domains and unknown recipe options fail deterministically |
| G01.2 Generic build service | Source loading, contract extraction, oracle samples and bounded candidate orchestration behind a reusable API | Three existing programs plus source renaming/operation-order variants build without class-name branches or source-specific trainer code |
| G01.3 Frozen bundle and failure behavior | Candidate staging, signature/lineage, deployment manifest, build report and atomic output publication | Interrupted build or failed certificate emits no deployable success marker; rejected candidates and actual costs remain available |
| G01.4 Source-free execution | Runtime loader, current finite certification, typed requests and explicit run outcomes | Copy only deployable files to a clean temporary directory; no compiler/oracle dependencies, no source/optimizer access, no caller-provided verification bit |
| G01.5 Differential and negative suite | Source-oracle/lookup comparison, startup/warm work and authority/frame regression checks | Exact results/writes/versions; rollback after successful earlier calls; wrong model/root/key/ABI/binding/domain denied; unsupported language fails clearly |
| G01.6 Reproduction and verdict | Two clean deterministic builds, artifact inventory, full report and ledger update | Same declared toolchain/target/seed yields identical normalized frozen bytes; independent reload and measurements reproduce semantics |

G01.1 precedes API/bundle work. Bundle format and loader must agree before the
source-free integration check. Finish semantic/negative checks before collecting
performance results. This table is not an instruction to spawn agents or create
separate chats.

## Fixture and evidence inventory

Use Counter, rejecting Accumulator and SaturatingAccumulator from the repository.
Add renamed class/field variants and reordered-method variants whose observable
semantics remain equivalent under normalized invocation names. These are G01
anti-hardcoding tests, not new program families or sealed G05 acceptance programs.
Retain a known failed low-width candidate as a failure-path fixture. Record all
fixture source hashes and recipe policies before implementation/testing.

For each program collect:

1. Source and derived contract hashes; declared domain/purpose and unknowns.
2. Actual reference observations, training split and trace/source provenance.
3. Candidate recipes/seeds, all raw predictions, training/prefit costs and failures.
4. Frozen parameter/contract/ABI roots, signature/key identity and file inventory.
5. Exact certification attempt counts/cost and source-free runtime results.
6. Oracle and lookup comparisons, including independent fixture checks outside
   the builder, plus work counters and clocked/unclocked timing.

Deployment files contain model parameters, exact specification/bindings, ABI and
integrity metadata. Development evidence may contain source/training data but is
not copied into the runtime deployment directory. Certificate state is rebuilt by
the current runtime, not trusted because a signed report says it once passed.
No signature alone establishes semantic correctness or application authority.

## G01 pass/fail

Pass only when all three supported programs and their anti-hardcoding variants
build through the same API/command, freeze, reload and match the source oracle
and lookup, including atomic rejection and ownership checks. Unsupported syntax,
shape or exhausted search must fail visibly with source locations/evidence and
no usable artifact activation. Two clean builds reproduce normalized bytes.

A valid bundle may still be outside the next gate's fresh-seed reliability target.
Passing G01 means the **pipeline is reusable**, not that broad learned compilation
is reliable or that the six richer families are supported. The next dependent
work is G02 plus the G03 semantic/ABI design, with corpus commitments made before
acceptance feedback. If G01 needs per-application custom code, keep the gate open
and report the coupling rather than broadening the success claim.

The original slice contract above is now backed by the locked G01 campaign,
recorded artifacts and acceptance report. The training plan is referenced in the
manifest and lock rather than overridden by a CLI flag. JSON is the first schema
encoding per ADR 0003. G01 is passed; later broad-feasibility gates remain open.
