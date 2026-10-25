# PHLAME: learned saturating transitions

A frozen residual ReLU model learns `clamp(total + amount, -16, 16)` and read,
then composes with an independently trained Counter through the existing scalar
ABI. Saturation happens in the model's predicted transition. The warm certified
boundary evaluates no source branch predicates or transition arithmetic.

[Recorded report](run-0001/REPORT.md) · [Machine report](run-0001/report.json) ·
[Source](../../examples/saturating-accumulator/saturating-accumulator.phlam)

102 source-observed primitive examples train the accumulator; all 492 remaining
primitive inputs predict the exact integer state, return and write flag before
checking. The affine baseline gets 200/492 held-out inputs correct. All 224
supported Counter compositions match the source oracle and lookup; 526 other
source-valid schedules leave the chosen domains and roll back. There is no native
false-return rejection in this saturating fixture. The previous rejecting
accumulator remains in its own experiment.

## Reproduce

```sh
go build -o bin/phlame-components-worker ./cmd/phlame-components-worker
go run ./cmd/phlame-components-experiment -branching -out work/saturation-reproduction -repeats 10
go test ./...
go vet ./...
```

Use a fresh output directory. The optimizer has a fixed seed, width and epoch
budget, without held-out validation, certificate-guided training or parameter
search. A first exploratory run used the same configuration and produced the
same accuracy and weights. Training twice checks bit-identical reproduction.
Source observations and the training subset have separate digests; per-input
raw scores and decoded predictions are also saved separately. Measurement code,
source hashes, artifact roots and toolchain are recorded in report.json.

## Reload without source

```sh
bin/phlame-components-worker experiments/saturating-accumulator/run-0001/registry.json \
  experiments/saturating-accumulator/run-0001/trusted-key.hex learned certified \
  < experiments/saturating-accumulator/run-0001/request.json
```

The four examples exercise upper saturation, lower saturation, an exact threshold,
and rollback after both modules stage writes. The last starts Counter at 0 and
accumulator at 16: Counter increments to 1, the accumulator adds that value and
successfully writes 16, then Counter rejects a passed argument of 16 outside its
[-4,4] domain. Both values and versions remain at their initial values, with no
published results or writes. Source execution accepts that last schedule; its
rejection is explicitly a learned-domain restriction, not a language rejection.

oracle-traces.json contains actual source events and observed branch categories;
example-results.json contains the frozen source-free worker's results. Local
research traces intentionally expose attempted values, including rolled-back
writes. They are debug evidence, not production application returns.

## Executor and exact boundary

The model has five inputs: bias, state/32, argument/32 and two own-operation
indicators. Sixteen learned ReLU units augment those inputs; three learned output
heads predict next state/32, return/32 and write propensity. All 143 parameters
are trained with full-batch Adam for 24,000 epochs. There are no built-in clamp
operators, branch IDs, contract paths or peer features in the predictor. Integers
are decoded by rounding; write flags use a 0.5 threshold. Model floats are
approximate; decoded integer behavior is the quantity exhaustively certified.

The scalar ABI is unchanged. Model metadata adds optional hidden rows to the
existing output weights. Old affine artifacts omit that field and still load;
new hidden dimensions and all weights are bounded and validated before use.
The runtime seals the entire module, including hidden weights. Removing the
nonlinear layer or corrupting the output heads prevents certification.

Startup independently checks all 594 accumulator and 330 Counter inputs against
source-derived exact contracts. The certificate retains admission intervals,
not next-state/return tables. At runtime the frozen numerical model executes
every admitted call. The boundary supplies frame ownership, typed prior-result
bindings, domain checks and atomic publication. Concrete mode recomputes source
relations and rejects incorrect predictions; certified mode checks admission
without rerunning those relations.

The source-free worker imports neither compiler nor oracle. Each measured child
process runs from a temporary directory containing only the binary, two signed
modules, registry and public key. Contracts travel with the modules; source-free
means no source-language implementation or compiler, not absence of a specification
or an OS sandbox. Certification repeats at startup and is process-local.

The lookup baseline interprets the same exact path relations. Concrete lookup
coalesces its application computation with the expected-value computation;
certified lookup reports its application arithmetic separately from checker work.
The independent source oracle also has different heap and tracing costs, so its
timing is not a matched executor speed comparison.

This demonstrates finite, decoded nonlinear transitions and their supplied
compositions. It does not demonstrate learned planning, symbolic source-branch
selection, arbitrary program compilation, real-valued exactness or correctness
outside the declared integer domain. Learned certified execution remains slower
than lookup in the recorded run. Research signing keys are publicly reproducible
fixtures, not production identities.
