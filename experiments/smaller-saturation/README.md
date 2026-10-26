# PHLAME: smaller learned saturation

A fixed survey trains 2-, 4- and 8-unit residual ReLU models on the same 102
source-observed primitive examples, with seed 404 and the unchanged 24,000-epoch
optimizer. Every candidate freezes separately and attempts source-free finite
certification. All successes and failures are retained.

[Recorded report](run-0001/REPORT.md) · [Selection and failed candidates](run-0001/selection.json) ·
[Matched comparison](run-0001/summary.json) · [Selected frozen model](run-0001/width-8/accumulator.json)

The 8-unit model fully certifies: 79 parameters instead of 143, all 492
nontraining primitive inputs correct, and all 224 supported compositions matching
the source oracle and lookup. Another 526 source-valid schedules leave the
chosen domains and roll back. The 2- and 4-unit runs each get 310/492 nontraining
inputs correct, and neither certifies. They also miss training inputs, so these
failures do not establish a minimum representational capacity.

Architecture selection uses full-domain certification. The 492 inputs are held
out of gradient updates, **not held out of architecture selection**. Width 8 is
the smallest success among these three fixed-seed runs; no other seeds, optimizer
settings, epoch budgets or widths were searched. An initial exploratory survey
used the same settings and produced the same weights and accuracy before the
recorded run.

Successful certified transactions without phase clocks measure 1.656 µs for the
8-unit model, 1.894 µs for the frozen 16-unit control and 1.547 µs for lookup.
These are medians of five fresh-process means with visible variation. Lookup
remains faster. All three have the same 5.312 allocations per successful
transaction and identical checker work, including zero warm contract arithmetic
or predicates. This is a model-size experiment on the existing optimized runtime;
no inference, transaction or checker logic was changed.

## Reproduce

Run from the repository root, with Go and Python 3 and no external packages:

```sh
go build -trimpath -buildvcs=false -o bin/phlame-components-worker ./cmd/phlame-components-worker
go build -trimpath -buildvcs=false -o bin/phlame-components-experiment ./cmd/phlame-components-experiment
go build -trimpath -buildvcs=false -o bin/phlame-components-profile ./cmd/phlame-components-profile
python3 experiments/smaller-saturation/study.py --out work/smaller-saturation-reproduction
go test ./...
go vet ./...
```

Use a fresh output directory. The runner records all three candidate artifacts,
source observations, training examples, raw predictions, source traces and
certification failures. It selects the smallest fully certified model, then
compares it with the historical frozen 16-unit accumulator and lookup using one
current profiling binary and one common frozen Counter. The common Counter's
weights, source contract, domain and training data match the original control;
its new artifact records the current training implementation lineage.

The matched comparison verifies byte-identical oracle observations, source
contracts, domains and training hashes across the smaller and larger models.
Every child process revalidates the signatures, independently recertifies the
finite domain and checks all 750 transaction outcomes before measuring.
Architecture/model aliases do not enter predictor features.

For one explicit candidate, including a failed one:

```sh
bin/phlame-components-experiment -branching -hidden-width 8 \
  -out work/single-small-candidate -repeats 1
```

The historical default remains 16 units. Hidden width is a bounded training
parameter, from 1 through 32; the signed scalar ABI and inference engine are
unchanged. A failed certification produces no transaction execution in certified
mode. Concrete mode may accept only those supplied schedules whose learned
outputs satisfy every exact relation; it never substitutes the expected result.
No break-even estimate is calculated for failed preflight.

## Reload the selected model without source

```sh
bin/phlame-components-worker experiments/smaller-saturation/run-0001/width-8/registry.json \
  experiments/smaller-saturation/run-0001/width-8/trusted-key.hex learned certified \
  < experiments/smaller-saturation/run-0001/width-8/request.json
```

The examples show upper/lower saturation, a threshold and failure after both
components stage writes. In the last, Counter reaches 1 and accumulator 16
successfully saturates at 16, then a return value exceeds Counter's argument
domain. Both states/versions remain original and no application results or
writes publish. The source oracle accepts that schedule; the failure is a
learned-domain restriction. Actual source events are in oracle-traces.json and
the frozen worker's results are in example-results.json. Debug traces expose
attempted values; they are separate from application results.

## Matched measurement

180 fresh-process measurements cover learned-8, learned-16 and lookup, both
concrete/certified boundaries, clocks on/off, mixed/accepted/scope-rejected groups
and five repetitions. Order is shuffled once with seed 20261025; each process
uses GOMAXPROCS=1 and measures 100,000 transactions after validation, warm-up and
GC. Three separate CPU/allocation diagnostic processes use 1.5 million accepted
certified transactions each and do not enter the latency summaries.

The temporary comparison directory contains the profiling binary, one common
Counter, two separately signed accumulator artifacts, two pinned manifests,
public research key and oracle-observation data. Source files, compiler and
reference oracle are absent. Expected results are used only to validate before
timing, never as model inputs. Contract specifications intentionally travel with
the artifacts. Source independence does not imply OS isolation.

The profile API omits only phase clocks when requested. All exact guards,
checker/application counters, output ownership and rollback remain active.
Outer latency includes counter aggregation and retaining the last result in a
sink; input/output JSON, startup and full-batch retention are excluded. Phase
measurements and load/certificate construction remain separate. Certified lookup
charges relation evaluation as application work; concrete lookup coalesces it
with checking.

There is no CPU pinning, confidence interval or broad speed claim. The decoded
integer behavior is exhaustively certified only for this finite domain. Learned
planning, symbolic branch discovery and correctness outside that region remain
unproven. Public signing keys are research fixtures; certificates are rebuilt
process-locally. Older reports and models remain unchanged.
