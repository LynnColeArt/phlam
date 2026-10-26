# PHLAME: training dynamics and initialization

[Recorded findings](run-0001/REPORT.md) · [All configurations and checkpoints](run-0001/selection.json) ·
[Independent diagnostic audit](run-0001/dynamics-analysis.json) · [Runtime comparison](run-0001/summary.json)

The failing seed-404 models lose useful nonlinear features during training. The
2-unit model finishes with one mixed activation boundary and one always-active
unit; the 4-unit model with one mixed boundary, two always-active units and one
dead unit. Both plateau near MSE 0.000648839. Passive instrumentation reproduces
the previous weights exactly. Independent analytic and finite-difference checks
find no gradient discrepancy at the audited checkpoints/initializations.

A fixed change to initial geometry makes the **4-unit, 47-parameter model fully
certifiable**: random hyperplanes start centered, with opposing directions in
pairs. It uses no source thresholds or branch labels. All 102 training inputs,
492 nontraining inputs and 224 supported compositions match; 526 outside-domain
schedules roll back. The source-free worker and exact boundary remain active.
Both tested 2-unit interventions still fail. This establishes a successful
four-unit realization of the finite task, not a minimum possible width.

![Training loss at fixed checkpoints](run-0001/training-loss.png)

## Fixed investigation

After reproducing legacy dynamics, the two target widths use three arms at seed
404, with the original 102 training examples and 24,000 joint Adam epochs:

- **legacy:** original random weights and intercepts.
- **paired-centered:** keep the original random output heads; zero hidden bias
  and operation offsets, and make each odd unit's state/argument direction the
  negative of its preceding even unit. The even directions remain random.
- **paired-affine:** same centered pairs, but first fit a linear model using the
  same examples for **12,000 additional gradient-descent epochs**, copy its base
  output weights and initialize hidden-output coefficients to zero. All weights
  subsequently train together; the affine model is not a runtime fallback.

No thresholds, contracts, held-out targets or peer state reach either initializer
or optimizer. Width, arms, seed and budgets were fixed before trying the
interventions. An 8-unit legacy run is a reference. Both 4-unit treatments pass;
selection takes the first passing arm in the declared order, before benchmarking.
The centered treatment needs no extra prefit and is selected. Its exact success
shows initialization matters here; this combined intervention does not isolate
centering from pairing or establish success on other seeds/programs.

Each model records 11 checkpoints: completed joint epochs 0,1,10,100,500,1000,
3000,6000,12000,18000,24000. Snapshots contain copied weights, training MSE,
per-unit preactivation range, activation counts by operation and current gradient
norms. Snapshots are taken before the next update; the final gradient is evaluated
without another update. For paired-affine, epoch 0 is after its extra prefit.
Observers do not control updates or stopping. A regression test mutates snapshot
copies and verifies that training weights are unaffected.

## Reproduce

From the repository root, with Go and Python 3:

```sh
go build -trimpath -buildvcs=false -o bin/phlame-components-worker ./cmd/phlame-components-worker
go build -trimpath -buildvcs=false -o bin/phlame-components-experiment ./cmd/phlame-components-experiment
go build -trimpath -buildvcs=false -o bin/phlame-components-profile ./cmd/phlame-components-profile
python3 experiments/training-dynamics/study.py --out work/dynamics-reproduction
python3 experiments/training-dynamics/analyze.py work/dynamics-reproduction
go test ./...
go vet ./...
```

Use a fresh output directory. The optional figure uses matplotlib; for example,
`python3 -s experiments/training-dynamics/plot.py work/dynamics-reproduction`.
The figure is a log-scale plot of saved checkpoints, not a dense epoch trace.
The training, audit and runtime-comparison scripts need no external Python
packages. Exploratory diagnostic runs used the same final configurations and
weights; no additional arms or seeds were searched after their outcomes.

For training-only diagnostics without source or contracts as inputs:

```sh
go run ./cmd/phlame-training-dynamics -width 4 -seed 404 -initialization legacy
```

This diagnostic command reads the original training samples and exports snapshots;
it is not an execution or certification worker. Full training/freeze/reload for
one candidate is available through the experiment command:

```sh
bin/phlame-components-experiment -branching -hidden-width 4 -branch-seed 404 \
  -branch-init paired-centered -training-dynamics \
  -out work/paired-four-reproduction -repeats 1
```

Legacy defaults are unchanged. New initializers and prefit counts are signed
lineage metadata, validated on load. They are not inference switches: the same
numerical predictor executes every artifact. Old artifacts still load.

## Source-free execution and evidence

```sh
bin/phlame-components-worker experiments/training-dynamics/run-0001/trials/w4-paired-centered/registry.json \
  experiments/training-dynamics/run-0001/trials/w4-paired-centered/trusted-key.hex learned certified \
  < experiments/training-dynamics/run-0001/common/request.json
```

Every trial is frozen before source-free checking. Successful startups independently
certify all 594 accumulator and 330 Counter primitive inputs. Failed candidates
stop on their first counterexample and execute no certified transactions. Their
full-domain raw predictions are still recorded. No partial domain or old
certificate is substituted. Concrete mode rejects incorrect proposals without
repairing them and discards earlier writes/returns on failure.

Shared observations/training data, requests and real oracle traces are stored
once in common/. Each trial retains its signed modules, root-pinned manifest,
key, raw predictions, report/checkpoints and example response; evidence-refs.json
pins shared files by digest. The source oracle accepts all 750 schedules, of
which 526 exceed learned input domains. The examples include failure after both
components stage writes; original states/versions remain and nothing publishes.

The profiler compares the selected 4-unit model with the frozen 8-unit control
and lookup using one binary and one common Counter. It checks all 750 outcomes
before every measurement and keeps startup, checker and application work
separate. Certified warm checking performs zero contract arithmetic/predicates;
lookup's relation interpretation is application work. The process has neither
source implementation nor compiler/oracle imports. Contract specifications
remain in signed artifacts; expected outcomes validate before timing and never
enter inference.

The matched run has broad timing variation and overlapping ranges. It establishes
no clear latency improvement; lookup has the lowest median. All timings and CPU/
allocation profiles are retained. Nontraining inputs are selection/certification
data, not an independent test set. Correctness is finite decoded integer behavior;
learned planning, universal branch discovery, OS sandboxing and production
identity/authorization are not established by this study.
