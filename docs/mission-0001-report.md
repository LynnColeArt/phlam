# Mission 0001 engineering report

Date: 2026-10-25. Scope: the brief's first concrete mission, not completion of the
whole Phase 1 or a learned runtime proof of concept.

## Delivered

- Corrected stale index-runtime language in the brief's lifecycle, security,
  debugger, layout, measurements and risks. Reference graphs remain legitimate
  training/baseline tools; the target is a frozen learned application.
- Inspected the official PHP 4.4.9 archive and AST migration RFC. Recorded actual
  file paths, the archive hash, incompatibilities and per-file license notices in
  [ADR 0001](decisions/0001-php4-as-reference.md). No historical code was copied
  into the implementation; downloads stay in ignored `work/`.
- Documented Go versus Zig and chose a stdlib-only Go bootstrap in
  [ADR 0002](decisions/0002-bootstrap-toolchain.md).
- Wrote the [language draft](language-spec.md), including layout, construction,
  identity, aliases and eval proposals, with pending decisions called out.
- Defined the [learned executor and exact boundary](learned-runtime.md): proposed
  typed transition terms, restricted semantic proof, exact primitive execution,
  staged effects, frozen artifact layout, data lineage, generalization splits,
  ablations, and accounting for validator work.
- Wrote the [initial threat model](security-model.md), with negative test
  obligations clearly marked unimplemented.
- Implemented `phlam check <source>` and `phlam inspect <source>` for the bootstrap
  grammar. JSON is an AST scaffold with byte spans and source-version-local IDs,
  not a learned application artifact or deployment-ready executable.
- Added `runtime/trace` as a recorder seam for future real execution. It neither
  evaluates ASTs nor turns static visits into observed training labels.
- Initialized Git on main and created the public GitHub remote
  [LynnColeArt/phlam](https://github.com/LynnColeArt/phlam) through gh.

## Verification observed

Environment: Go 1.26.4, Linux amd64. Commands run from the repository root:

| Check | Observed outcome |
| --- | --- |
| `go test -v ./...` | CLI, frontend structure/provenance/precedence/nesting tests and 36 syntax fixtures pass; three runtime obligations explicitly SKIP |
| Syntax corpus | 10 accepted inputs, 26 rejected inputs with expected diagnostic codes |
| `go vet ./...` | Pass |
| `go build -o bin/phlam ./cmd/phlam` | Pass |
| `bin/phlam check examples/counter/counter.phlam` | Exit 0; one class; explicitly identifies bootstrap syntax checks |
| `bin/phlam inspect examples/counter/counter.phlam` | Deterministic JSON AST produced |
| Same source bytes copied to a different path, inspect + cmp | Byte-identical AST output |
| Check missing-purpose fixture | Exit 1; P101, line/column, source excerpt and repair hint |
| Parser fuzzing, initial 4-second run | 284,274 executions, no crash |
| Parser fuzzing after final parser changes, 2-second run | 98,021 executions, no crash |

Local detailed test output and inspected AST are under ignored `work/` and can be
regenerated using README commands. Short fuzz runs are robustness samples, not
proofs of parser correctness or runtime confinement.

Final review added regression coverage for indirect-call binding (`$$name()`),
unique IDs on chained expressions, multiline predicates and escaped-newline
string rejection. No runtime/security/learned success is inferred from syntax.

## What does not run

No semantic execution yet: Counter cannot increment at runtime. Variable
variables, aliases and eval are parsed, not evaluated. The three differential
placeholder tests skip with reasons; `tests/differential/scenarios.json` records
expected future behavior, including caller-scope eval returning 12.

No manifest/lock parser, complete Phase 1 name/contract analysis, loops, dynamic
member syntax, trained weights, model loader, proof checker, effect commit gate,
web/agent gateway or semantic debugger exists. Security requirements and
reproducible-training claims await working evidence. License selection for
original PHLAM remains the owner's decision.

## Next vertical slice

Implement exact Counter reference semantics with new/init, shared object identity
and independent instances, integer overflow checks, real primitive traces and
compiler-derived operational/frame contracts. Then add variable-cell aliases and
eval overlay semantics against the recorded scenarios. Keep these oracle steps
small and separately tested.

Use those contracts/traces to run the bounded learned transition experiment in
`learned-runtime.md`. Before selecting a model, fix the candidate term grammar and
checker domain. Verify rejected proposals cannot write state. Run a reloaded
learned artifact without source and compare externally against the oracle and
explicit graph baseline. Report model, checker and primitive work separately;
no hidden reference fallback or claim of generalization from Counter alone.
