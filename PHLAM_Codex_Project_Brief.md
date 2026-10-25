# PHLAM: Project Outline and Codex Engineering Brief

**PHLAM** = **Program Hierarchy and Learned Associative Memory** (pronounced *flam*).\
**Status:** Exploratory architecture and phased implementation brief, 25 October 2026.\
**Audience:** Codex and collaborating engineering agents.\
**Goal:** Establish a small, PHP-descended language that compiles and trains applications into frozen learned computational representations, with exact verified operations and effects. The same application powers conventional interfaces and an agent-native capability surface.

> **One-sentence thesis:** Write familiar code, train a learned application substrate, and expose the same verified behavior to people and agents.

## 1. Product vision

PHLAM is **not** a PHP 4 compatibility fork or a vector database with PHP syntax bolted on. PHP 4 / Zend Engine 1 is a historical reference for familiar syntax and expressive dynamic behavior. We are deliberately redesigning the language surface and the execution substrate.

A PHLAM application has a **primary learned computational representation**: a trained embedding or related learned state-space model intended to participate in application execution, not merely retrieve code or documentation. The application is trained from its class contracts, program structure, and verified execution examples. The resulting model is versioned and frozen at deployment.

An **exact semantic/effect layer** remains necessary for discrete truth, state identity, authorization, ordered side effects, and provable behavior. This layer is a correctness boundary, not a pretext to rebrand a traditional graph VM plus vector search as a learned runtime. An AST or graph IR may exist as a *compiler tool and training scaffold*; it is not the target deployable execution substrate.

**Research hypothesis:** can a learned application representation actually participate in choosing or producing valid executable transitions, including paths not simply read from a predefined graph, while a deterministic boundary guarantees program contracts? If not, report the failure explicitly. Do not quietly substitute an indexed interpreter as the product architecture.

An ordinary web UI and an AI agent use the **same backend operations**, with different presentation and authorization contexts. The agent can operate a deployed application without having permission to alter its executable definition.

### Primary outcomes

- A language familiar enough that coding agents can read and write it with modest additional instruction.
- Predictable, class-centered application structure and a small, modular standard library.
- Reproducible training/compilation into an inspectable, versioned and frozen learned application artifact with exact execution safeguards.
- Agent discovery and tool invocation derived from class contracts and compiler evidence.
- Semantic debugging grounded in real execution traces, with Rust-quality diagnostics.
- A first-class experiment in learned application execution and, later, direct agent authorship of learned structures or validated capabilities.

### Explicit non-goals (initially)

- Running existing PHP applications unchanged.
- Reimplementing PHP 8, Laravel, or the complete historical standard library.
- Eliminating exact runtime semantics in favor of approximate vector similarity.
- Letting an LLM silently invent permissions, side effects, or API contracts.
- Providing production write access to the frozen learned application artifact.
- Training a frontier model or building a general-purpose LLM as part of the compiler.
- Optimizing performance before we have a behavioral oracle and benchmarks.

## 2. Language design: familiar PHP, deliberately smaller

**Proposed source extension:** `.phlam` (provisional). **CLI:** `phlam`. Files are always code, never embedded HTML templates.

### 2.1 Syntax decisions

- Remove `<?php` / `?>`. No mixed HTML/code lexer mode.
- Semicolons are **optional at statement boundaries**, but remain legal. Specify deterministic lexical/grammar rules; do not infer structure from arbitrary whitespace. Semicolons remain mandatory as separators inside traditional `for (...)` headers. Parenthesized multiline expressions are supported. Ambiguous continuation cases must produce useful diagnostics.
- Retain `$variable`, `$this`, associative arrays, PHP-like expressions, `if`, loops, `return`, and familiar method calls.
- Require **classes for executable application functionality**: no global application functions and no free-floating top-level executable statements. Manifests designate application entry points.
- Remove the `var` property declaration keyword. Properties use `$name` or `$name = value` within a class.
- No `public`, `private`, or `protected` in v1. All members are accessible **inside the application**. That is not the same as agent/API authorization.
- One object creation form: `new ClassName(...)`; one initializer convention: `init(...)`. No class-named constructors, magic constructor alternatives, or implicit factories. `init` runs once during construction and cannot be invoked as an ordinary method afterward.
- Instances retain independent state. Do **not** turn all classes into global singletons.
- Standardize method and library function names on `snake_case`; classes use `PascalCase`. Rationalize standard-library parameter order rather than preserving PHP's historical inconsistencies.
- Keep PHP's productive dynamic affordances: variable variables (`$$name`), dynamic function/method resolution, references/aliases where defined, and `eval()`.
- No unnecessary modern OO complexity (traits, magic-method lattice, overloaded constructors, reflection free-for-all) in the initial subset. Inheritance can be considered after the object model is stable.

### 2.2 Required class contract

Every class **must begin with a nonempty purpose description** in a recognizable documentation comment. An undocumented class fails compilation. The developer explains what the class is *for*, rather than writing a manual for every method.

The compiler derives an operational contract for each method from the AST/IR: parameter names, observed value shapes where knowable, returns, state reads/writes, calls, declared/imported capabilities, identifiable branches, and effects. The compiler can supplement this with **verified** sandbox execution observations. Unknown dynamic behavior must be marked `unknown` or conservatively bounded, never fabricated.

A class description is a *claim about intent*. Static analysis and tests are *evidence about behavior*. The debugger and agent surface must keep those categories distinct.

### 2.3 Illustrative source (design sketch, not finalized grammar)

```php
/// Holds a counter and supports controlled increments.
/// Each Counter instance owns separate state. Class members
/// have no language-level public/private modifiers.
class Counter {
    $value = 0

    function init($starting_value = 0) {
        $this->value = $starting_value
    }

    function increment($delta = 1) {
        $this->value += $delta
        return $this->value
    }

    function read() {
        return $this->value
    }
}
```

Illustrative dynamic semantics, inside a method:

```php
$name = "value"
$$name = 10
$alias = &$value
eval('$alias += 2')
```

This must modify the caller-visible scoped value in the reference executor. Note that PHP 4 is a **semantic inspiration**, not a blanket compatibility target. Explicitly decide PHLAM's object assignment/copy behavior, coercion tables, array semantics, dynamic name resolution, and error model before declaring the language stable.

### 2.4 Dynamic evaluation versus frozen deployment

`eval()` is a language feature, not a deployment channel. It parses and executes PHLAM statements in the current permitted scope and can access/modify the caller's variables as specified. Runtime-generated code is executed as a **scoped, ephemeral overlay** under the caller's existing capabilities and resource limits. It does not silently mutate the signed base learned application artifact, modify permissions, or create permanently deployed tools. If dynamic class definitions are supported, document their scope and lifetime explicitly. Do not silently drop `eval()` because it complicates the compiler.

## 3. Manifest, dependencies, and standard library

Use one Go-inspired manifest, tentatively `phlam.mod`, and a resolved `phlam.lock`. The manifest declares module identity, standard-library pieces, external dependencies, executable entry points, and **separately** the authority granted to the application and the methods exposed to agents.

Illustrative TOML, **not yet normative syntax**:

```toml
[module]
name = "example/orders"
version = "0.1.0"

[stdlib]
include = ["arrays", "strings", "json", "http"]

[dependencies]
storage = "sqlite@1"

[entrypoints]
web = "WebApp.handle_request"

[capabilities]
allow = ["storage.read", "storage.write"]

[agent]
expose = ["Orders.find_duplicates", "Orders.refund"]
```

**Important separations:** importing an HTTP or storage library is not authorization to use the network or disk. Including a class/method in the executable is not permission to expose it to an agent. Internal class accessibility does not create external authority. Invocation must still check the caller, resource, arguments, effects, and any required confirmation.

The runtime constructs request-scoped entrypoint/service instances through the same `new` + `init` mechanism as ordinary code; any longer-lived instance must have an explicit lifecycle. Design a small modular standard library. Start with `strings`, `arrays`, `math`, `json`, and essential core values. Add file/network/storage modules through explicit capability adapters. A manifest should define the complete deterministic dependency graph, with hashes/versions in the lockfile. No implicit autoloading of ambient code.

## 4. Compiler, training, and learned runtime representation

Proposed pipeline:

```text
.phlam sources + mandatory class purposes + phlam.mod
    -> tokenizer / parser / AST
    -> exact semantic analysis, contracts, effects and reference traces
    -> training examples and a learned application state-space representation
    -> differential validation against an exact reference evaluator
    -> version, sign, freeze learned artifact and exact effect contracts
    -> runtime / web interface / agent interface / semantic debugger
```

Treat Zend Engine 1 as **reference material**, not the execution architecture. An AST or graph IR can help training, validation, debugging, and provenance but should not be assumed to be the deployable computation substrate.

### 4.1 Primary learned artifact

The learned representation should encode useful relationships among program state, class capabilities, transitions, and outcomes. Investigate whether a learned state transition/operator representation can participate directly in computation. Embeddings by themselves do not guarantee exact arithmetic, branching, dynamic scope, or side-effect ordering. Identify and document the smallest deterministic mechanisms required to enforce these semantics; distinguish them from learned behavior.

The deliverable must include a real artifact that is saved, reloaded, and invoked independently of source text. Persist model/version metadata, training lineage, validated contracts, and exact capability/effect bindings. No opaque claims that semantic similarity guarantees correctness.

### 4.2 Reference evaluator versus learned executor

1. **Reference evaluator:** deterministic interpreter for precisely specified PHLAM semantics, useful for generating traces and differential tests. It need not be fast.
2. **Learned experimental executor:** actual learned representation participates in dispatch, transition selection, or computation. Its predictions/proposals must be checked against exact semantics before effects commit. Avoid a graph interpreter with vector lookup presented as the central achievement.

Both must be subjected to identical behavioral scenarios. Measure where learning adds capability, cost, ambiguity, or failure.

### 4.3 Training strategy

Begin with small bounded tasks from compiler facts, documented class purpose, and verified execution traces. Compare learned operations against trivial lexical retrieval, an explicit graph IR baseline, and the deterministic evaluator. Establish train/test splits that assess generalization instead of memorization. Retraining produces a new immutable version; production invocation does not update weights. Never use embedding proximity to authorize anything.

## 5. Unified web and agent interfaces

A normal HTML/JavaScript frontend may be built with conventional tooling. It calls the same application operations exposed to the agent, through an HTTP/runtime adapter. We are **not** requiring a special UI DSL or making the agent click through web pages.

Expose a tiny universal agent bootstrap interface:

```text
app.describe()  -> application summary and authorized domains
app.discover(intent, context?) -> ranked candidate capabilities
app.inspect(capability_id) -> exact parameters, effects, constraints, provenance
app.invoke(capability_id, args) -> checked execution and typed result
app.observe(handle_or_trace_id) -> authorized state/effect observations
```

Use stable IDs for invocation. Embeddings may *rank candidates*, but cannot stand in for exact identities, argument contracts, permission checks, or transaction boundaries. Instance-bound methods require explicit authorized instance/resource handles; agent clients must not bypass construction, ownership, or tenancy by supplying arbitrary IDs. Stateless service operations may be bound to fresh request-scoped instances constructed by the gateway under a documented rule.

Generate method descriptions from the **class purpose comment plus compiler-derived evidence**. Support `verified`, `declared`, `inferred`, and `unknown` provenance labels. Methods with unbounded/unknown effects should not be automatically exposed as high-privilege tools.

An LLM may propose a sequence such as “locate order → inspect status → check eligibility → refund,” but each call is independently authorized. No separate manually maintained MCP implementation should be necessary for standard discover/invoke behavior; an optional MCP adapter can translate the native surface for existing agent ecosystems.

## 6. Security architecture: freeze behavior, not usefulness

**Deployment lifecycle:** author -> compile -> analyze -> generate verified traces -> train -> validate -> freeze/version -> serve. The production learned application artifact and its exact contracts are immutable to normal web users and operating agents. Mutable business data and runtime state live outside the immutable executable artifact.

Required boundaries:

- **Discoverability is not authority.** Agent-visible metadata is permission-filtered; a discovered method cannot be invoked without deterministic authorization.
- **Manifest is not a substitute for policy.** Effective permissions are granted by the deployment principal and gateway, not asserted by source or LLM instructions.
- **Effect isolation.** Database, network, filesystem, process, secrets, and external APIs are separately mediated adapters. No ambient power granted by `eval()` or dynamic dispatch.
- **Artifact integrity.** Version/hash/signature checks, transactional updates, rollback, immutable production snapshots, and source-to-artifact provenance.
- **Training integrity.** Separate trustworthy contracts from untrusted content; prevent learned-representation poisoning; record training artifacts, datasets, and model versions.
- **Prompt-injection resistance.** Descriptions, records, search hits, and user-submitted content are data, not higher-priority instructions. Agent outputs are untrusted requests to the gateway.
- **Bounded dynamic evaluation.** Parse and execute in confined capability context with CPU/memory/time limits and no persistent runtime authority escalation.
- **Tenant isolation and privacy.** Resource-level checks, redacted traces, secrets excluded from embeddings, and no automatic training on user data.

Initial threat model must explicitly cover cross-tenant data leaks, misrouted semantic tools, malicious/poisoned documentation, learned-artifact tampering, code generated by `eval`, malicious dependencies, confused-deputy behavior, unsafe side effects, and trace disclosure.

## 7. Semantic-first debugger

Aim for **Rust-quality error explanations plus causal execution tracing**, not a chatbot that guesses why code failed.

The compiler preserves source spans and stable provenance IDs; the runtime records optional structured provenance: input values where permitted, decisions, state versions, calls, effects, failures, and causal links. The debugger can map facts back to source, contracts, learned proposals, and exact operations.

User experience:

- Conventional breakpoints, stepping, stack/scope inspection, and source diagnostics remain available.
- Semantic query: “Why did this order get refunded?” returns a path grounded in trace evidence.
- Semantic breakpoint: “Stop when a shipped order reaches a refund effect,” compiled into an inspectable exact predicate.
- Contract discrepancy: “Developer says read-only; this path writes to storage.”
- Counterfactual replay: fork a captured trace and change a selected input/state value; replace real external effects with recordings/mocks.
- Patch verification: compare before/after traces and report behavior changed, behavior preserved, and uncovered paths.

LLMs may explain, summarize, or suggest repairs. **They must not invent traces, label hypotheses as observed facts, or authorize repair deployments.** Clearly distinguish observed, inferred, and unverified claims.

Initial CLI sketches:

```text
phlam check
phlam build
phlam test
phlam inspect <artifact>
phlam explain <trace-id>
phlam debug <entrypoint>
phlam train <artifact>      # experimental, reproducible
phlam serve <artifact>
```

## 8. Repository layout (proposed)

```text
phlam/
  README.md
  LICENSE
  phlam.mod                  # self-hosting later; optional at bootstrap
  docs/
    vision.md
    language-spec.md
    learned-runtime.md
    security-model.md
    agent-protocol.md
    debugger.md
    experiments.md
    decisions/
  reference/php4/           # notes, tests, links; do not blindly vendor code
  compiler/
    lexer/
    parser/
    ast/
    semantics/
    ir/
  runtime/
    reference/
    learned/
    effects/
  stdlib/
  agent/
    gateway/
    discovery/
  debugger/
  experiments/
    embeddings/
    geometric-execution/
  examples/
    counter/
    orders-web-and-agent/
  tests/
    syntax/
    semantics/
    differential/
    agent/
    security/
    traces/
  benchmarks/
```

The implementation language is **not yet mandated**. Evaluate Zig vs Go (and any justified alternative) on parser/tooling ergonomics, memory safety, performance, deployment, and agent-maintainability. Record the tradeoff; don't make a multi-language toolchain mandatory without evidence.

## 9. Phased delivery with exit criteria

### Phase 0: Archaeology, decisions, and a testable specification

- Inspect official PHP 4.4.9 source and historical Zend 1 compiler/parser; document which concepts or code could be reused.
- Audit licensing **per actual file and version** before copying any implementation.
- Write the initial language grammar, dynamic-scope rules, object identity semantics, error model, and `eval` rules.
- Define the learned-artifact hypothesis, the exact semantic/effect boundary, and the first threat model. Explicitly contrast this against the earlier executable-index design.
- Establish a test harness and baseline examples.

**Exit:** short architectural decision records, unambiguous grammar examples, documented open questions, and an executable test runner skeleton.

### Phase 1: Small language frontend

- Parse class definitions, required purpose comments, properties, methods, literals, arrays, variables, calls, branches, and returns.
- Implement optional semicolon rules with focused ambiguity tests.
- Parse `phlam.mod`; reject unlisted dependencies, undocumented classes, free functions, and invalid constructor forms.
- Emit AST/IR dumps and source-aware Rust-style diagnostics.

**Exit:** `phlam check` and `phlam inspect` work for the Counter example; positive/negative syntax corpus passes.

### Phase 2: Deterministic reference semantics

- Implement `new`/`init`, instance state, arrays, scope, aliasing, dynamic names, and a minimal exact evaluator.
- Add `eval()` in caller scope, executed in sandboxed ephemeral overlay with the same parser.
- Document and test coercions, errors, reference aliasing, and object-copy/identity rules.

**Exit:** reference evaluator passes explicit behavioral tests including variable variables, references, and `eval`; every failure provides source spans.

### Phase 3: Frozen learned-application proof of concept

- Generate training data and deterministic trace labels from the small reference evaluator.
- Train an initial bounded embedding/state-transition representation for a narrow program family.
- Serialize, reload, freeze, and independently invoke the learned artifact; preserve exact operation and effect gates.
- Test genuinely learned dispatch/transition behavior against the reference and against a nonlearned graph/lookup baseline.
- Report disagreements, generalization failures, training cost, execution cost and any semantic limits.

**Exit:** at least one real computation pathway depends on the frozen learned artifact, runs without source files, and is externally verified against exact tests. If this fails, report why before designing a substitute index VM.

### Phase 4: Web + agent demonstration

- Add capability extraction, class-purpose docs, method contracts, and effect metadata.
- Implement authenticated `describe/discover/inspect/invoke/observe` endpoints.
- Serve a conventional HTML/JS UI and an agent client against the same Orders demo methods.
- Add permissions, resource-bound instance handles, negative authorization tests, and safe effect adapters.

**Exit:** browser and agent perform the same authorized operation and observe consistent results; unauthorized calls reliably fail.

### Phase 5: Semantic debugging

- Structured traces with source, learned-artifact and exact-effect provenance.
- Exact semantic breakpoint predicates and readable compiler/runtime diagnostics.
- Evidence-backed “why” queries, safe counterfactual replay, and trace comparison.

**Exit:** one deliberately introduced bug is diagnosed from an execution trace, reproduced, and verified fixed without fabricated evidence.

### Phase 6: Expand learned computation and associative capability discovery

- Extend the successful bounded learned runtime from Phase 3 toward larger class families and mutable state.
- Evaluate agent semantic tool discovery and composition using the *same* trained representation.
- Test learned transition routing/composition while an independent exact layer enforces contracts.
- Benchmark generalization, correctness, security, latency, memory, retraining, and model drift versus deterministic baselines.

**Exit:** quantitative evidence that the learned representation contributes to execution and tool discovery, or explicit documentation of limitations and failed hypotheses.

### Phase 7: Direct agent authorship (research track)

- Permit agents to propose changes to learned application artifacts or validated contract operations in **development sandboxes only**.
- Validate structural invariants, capabilities, effects, and equivalence tests before proposing deployments.
- Produce human-readable reconstruction/diffs for any machine-authored artifact.

**Exit:** a bounded new capability can be constructed without source, audited, tested, and safely versioned, or the experiment documents why it fails.

## 10. Measurement and evaluation

Maintain baselines and publish results rather than assuming the new substrate is better.

- **Language:** parser success/failure matrix, semicolon ambiguity tests, dynamic variable and `eval` semantics, repeatable builds.
- **Correctness:** differential trace comparisons between reference evaluator and learned executor; no silent divergence; property-based tests where useful.
- **Agent ergonomics:** task completion rate, Recall@k for discovery, wrong-tool invocation rate, token/tool-call budget, developer documentation burden.
- **Security:** cross-tenant denial tests, prompt-injection attempts, poisoned contract retrieval, `eval` confinement, policy bypass attempts, version/signature checks.
- **Debugger:** root-cause localization accuracy against known seeded faults, explanation provenance, replay determinism.
- **Runtime:** cold start, throughput, p50/p95 latency, learned artifact load size, memory, training time, versioned retraining/build time.
- **Scientific integrity:** ablations separating benefit from class structure, explicit contracts, embeddings, and learned execution. Demonstrate more than “ordinary VM plus vector search.”

## 11. Design risks and decisions to document

1. **Learned execution vs. deterministic replay:** Can a frozen learned representation contribute valid transitions beyond memorized traces? Measure against evaluator and graph/lookup baselines; account for all work performed by the exact boundary.
2. **Dynamic `eval` vs. frozen application:** Define transient overlays, lifetime, and effects; preserve caller-scope mutation.
3. **Object assignment:** PHP 4's value-like object behavior versus modern shared identity; choose intentionally and test it.
4. **Semicolon inference:** Formal grammar and diagnostics for newline-sensitive ambiguities.
5. **Agent contracts from code:** Static analysis cannot prove all natural-language claims or all possible dynamic effects; expose unknowns.
6. **State scoping:** Instances, tenant resources, request scopes, concurrency, and persistence must be explicit.
7. **Training integrity:** Embeddings can be poisoned or stale; learned scores can never grant authority.
8. **Performance:** A learned runtime and its exact checks may be slower or larger than conventional execution. Benchmark early.
9. **PHP licensing:** Inspect historical source, generated parser files, bundled libraries, and notices before copying.
10. **Language scope creep:** Protect the minimum PHP-like core; add only features demonstrated necessary by applications/tests.

## 12. Codex: first concrete mission

**Do not attempt to implement the whole vision in a single pass.** Start with a small, falsifiable vertical slice and an evidence trail.

1. Inspect the current repository. If empty, create the proposed skeleton without creating unnecessary frameworks.
2. Read the PHP 4.4.9 historical sources and PHP AST migration notes. Write `docs/decisions/0001-php4-as-reference.md` with real file paths, reuse opportunities, incompatibilities, and licensing cautions.
3. Write initial `docs/language-spec.md`, including `new` + `init`, class-purpose documentation, property syntax, optional semicolon grammar, scope, aliases, and `eval` semantics. Mark any unresolved decision explicitly with a proposed default and test case.
4. Write `docs/learned-runtime.md` outlining training inputs, model candidates, exact execution boundary, reproducible frozen artifact format, and benchmarks against a graph-index baseline. Document what remains unproven.
5. Write `docs/security-model.md`, especially agent permission filtering, immutable deployment, instance ownership, and bounded runtime compilation.
6. Implement the smallest frontend that parses the Counter example and emits a deterministic AST and trace-generating scaffold. Add `phlam check` and `phlam inspect` or equivalently simple commands.
7. Add syntax tests for missing class documentation, illegal top-level functions, bad constructors, semicolon ambiguity, and normal declarations. Add placeholders for differential tests of dynamic variables and `eval` without falsely claiming implementation.
8. Return a concise engineering report: what runs, what fails, evidence paths, decisions made, and the next vertical slice.

**Engineering behavior:** Work autonomously within this brief; resolve ordinary implementation questions with documented assumptions and tests. Do not claim features, benchmarks, or security properties without working evidence. Prefer small verified deliverables to impressive but untestable architecture.

## 13. External historical references

- PHP 4 release archive: https://museum.php.net/php4/
- PHP 4.4.9 release announcement: https://www.php.net/releases/4_4_9.php
- PHP project licensing guidance: https://www.php.net/license/
- PHP AST adoption RFC (useful contrast with Zend 1): https://wiki.php.net/rfc/abstract_syntax_tree
- PHP `eval()` scope behavior: https://www.php.net/manual/en/function.eval.php

---

**North star:** A PHLAM application is a documented hierarchy of capabilities compiled and trained into a frozen learned computational representation. Learning participates in execution and agent navigation; independent exact semantics and capability gates preserve correctness and security. An index graph may be an internal reference structure, not the architectural destination.
