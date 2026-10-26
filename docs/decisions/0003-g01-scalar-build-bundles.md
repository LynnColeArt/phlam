# ADR 0003: reusable scalar build and deployment bundles

Date: 2026-10-26. Status: implemented for G01, not a stable language-wide format.

The existing experiment drivers prove bounded learned execution but hardwire
source paths, domains and recipes. G01 moves these inputs into versioned data and
uses one authoring service plus a separate frozen-artifact runner.

## Decision

`phlam.mod`, `phlam.lock` and training plans use JSON in this initial version.
The brief's TOML example was explicitly illustrative. Go's standard JSON tools
allow bounded decoding, unknown-field/trailing-document rejection and no new
runtime dependency. A future syntax change needs a new schema/migration; JSON
formatting is not inferred from file extension.

The manifest declares module identity/version, explicit local source files,
source/class/component bindings, exported entry methods, specialization input
domains, lock and training-plan path. Every source has one class and one scalar
property in this slice. Inputs must be regular files resolving within the
project directory. The exact source/plan set must match lockfile content hashes.
No ambient source discovery, dependency download, host import or authority grant
is introduced. Unlisted unrelated files are never loaded; a component may not
reference an unlisted source, and undeclared lock entries are rejected.

A data-only plan specifies sampling strides, generic candidate recipes and seed
order, maximum primitive cases, per-candidate training/verification wall timeout
and whole-build wall timeout. Affine and residual ReLU models share the existing
numerical engine. No class names, application formulas or expected fixture
outputs are used to choose target behavior in the builder. Purpose text remains
declared intent, not a contract proof or permission.

The authoring service derives exact relations and observes actual reference
execution over the finite domain, checking their agreement. It trains only its
selected admitted examples in a cancellable optimizer subprocess, freezes each
produced candidate, and invokes the standalone runtime to rebuild the finite
certificate. Incorrect candidates retain their artifact/raw predictions/failure.
The first fully certifying candidate matching source observations is selected.
Independent fixture mathematics belongs in acceptance tests, outside the builder.

Deployment consists of signed module files and a separately signed `bundle.json`
that pins aliases, file names, module roots, module identity, ABI and entry names.
This prevents an unsigned alias/export edit from redirecting a valid signed model.
An external `--key` is required: the bundle cannot nominate its own trust root.
The development key does not create caller/tenant/resource authority.

The normalized ABI is `explicit-int64-v0`. Source defaults are neither removed
from the language nor implicitly inserted into artifact requests. Zero-argument
operations have no argument binding; one-argument operations require exactly one
literal/prior-result binding. Source false-as-rejection aborts the enclosing
research transaction; the application receives a structured rejection, not a
published staged return. Instance state/version inputs are the current local
scalar research ABI, not authenticated persistent object handles.

`phlam` remains the authoring/source-oracle CLI; `phlam-train` is its optimizer
sidecar. `phlam-runtime` imports neither compiler nor reference evaluator, links
no optimizer routines and never trains. It verifies bundle/modules, reconstructs
certificates for the current engine, checks signed exports, then uses the existing
transactional scalar runtime. Signature/load, export-check, certificate, proposal,
relation-check, binding and commit measurements remain distinct. Turning phase
clocks off preserves all work counters and semantics.

## Publication, reproducibility and limits

Build evidence is in a separate new directory, including locked inputs, actual
source observations/selected real trace samples, training rows, candidate scores,
failures, wall/CPU measurements, current Go managed-memory-at-end metric and build
report. It is never copied into deployment. The training sidecar uses a Go GC soft
limit; neither that metric nor G01 is an OS memory/security sandbox.

Selected deployment files stage in a sibling temporary directory. The final
report is written before one same-filesystem directory rename publishes the
bundle. Handled cancellation/failure removes staging and never publishes output;
a hard process kill can leave an inactive staging directory, without an output
bundle. Existing output/evidence paths are refused. Publication assumes a trusted,
single-writer development filesystem; it does not claim hostile-filesystem race
resistance, fsync-based crash durability or production concurrent deployment.
G07/G08 own those later systems guarantees.

Frozen identity includes source, contract, domain, samples, oracle observations,
model, seed, initialization/prefit, toolchain and an implementation fingerprint
of the authoring/trainer/runtime binaries. Measurements remain outside frozen
bytes. Two clean project builds on the same declared toolchain reproduce the
modules and signed bundle exactly. Runtime never accepts the report or a stored
`verified` flag as a semantic certificate.

Scalar bounds and compiler restrictions remain explicit. Arrays, loops, aliases,
eval, general dependency resolution, production keys/policy, durable state and
fresh-seed training reliability are not delivered by G01.
