# ADR 0002: One Go toolchain for the first frontend

Decision: Go for mission 1, 2026-10-25. Revisit after learned-model requirements
are measured. Local tools observed: Go 1.26.4, Zig 0.16.0, Linux amd64.

| Criterion | Go | Zig |
| --- | --- | --- |
| Parser/tooling ergonomics | Managed allocations, stdlib JSON/hash/testing fit this small frontend | Explicit allocators suit AST arenas; more lifetime plumbing for this bootstrap |
| Memory safety | Managed memory and bounds checks; avoid unsafe/cgo | Explicit lifetime management and runtime safety checks; ownership still needs discipline |
| Performance | GC and model interop may matter; not measured | Allocation/layout control may help; not measured |
| Deployment | One CLI binary with stdlib-only dependencies | Native binaries and cross-target facilities are attractive |
| Agent maintenance | Judgment: fewer moving parts for this parser and corpus harness | Judgment: explicit resource APIs add bootstrap work but can suit later kernels |

Choose Go for the CLI, handwritten lexer/parser, AST serialization and test
harness. No parser generator, ML runtime, network dependency, or multi-language
build is required. This is an implementation-effort decision, not a benchmark
claim that Go will execute learned applications faster.

[Go's FAQ](https://go.dev/doc/faq) documents garbage collection;
[Zig's language reference](https://ziglang.org/documentation/0.16.0/)
documents memory management and runtime safety. The tradeoff assessment above is
our engineering judgment. Do not select training/inference libraries solely to
preserve this choice. First define the bounded model task and exact checker ABI;
add another toolchain only when a concrete model requirement justifies it.

The module path is `github.com/LynnColeArt/phlam`; it has no external dependencies.
No `go.sum` is needed. The repository declares Go 1.26; validation used 1.26.4.
