# PHLAME: centered, paired seed sweep

[Recorded report](run-0001/REPORT.md) · [All 18 configurations](run-0001/selection.json) ·
[Matched runtime results](run-0001/summary.json)

Widths 2 and 4 use centered, opposing random initialization with seeds
**0,1,2,3,4,5,6,7,404**. Everything else stays fixed: the same 102 examples,
24,000 joint Adam epochs, full integer domain and exact boundary. There is no
affine prefit or additional training budget.

**Two 4-unit runs certify: seeds 7 and 404. No 2-unit run certifies.** Seed 7
adds a new success; seed 404 reproduces the prior frozen model byte-for-byte.
Among the eight new seeds, the observed counts are 1/8 successes at width 4 and
0/8 at width 2. The known control is reported separately; these fixed-grid
counts are not a statistical estimate of general convergence probability.

Both successful models have 47 parameters and preserve all 594 primitive
transitions, 224 supported compositions and 526 domain rollbacks. Every failed
artifact, raw prediction and first counterexample is retained. The earlier
legacy-initialization grid had no successes at either width. This experiment
shows an additional four-unit success, while convergence remains fragile.

## Fixed selection and reproduction

The first fully certifying seed in declared order is selected per width before
runtime measurement: width 2 has no selection; width 4 selects seed 7. No seed is
selected using latency, MSE alone or partial composition coverage. Full-domain
certification also serves as selection data; nontraining cases are not an
independent seed-selection holdout.

Run from the repository root with Go and Python 3, using a fresh output directory:

```sh
go build -trimpath -buildvcs=false -o bin/phlame-components-worker ./cmd/phlame-components-worker
go build -trimpath -buildvcs=false -o bin/phlame-components-experiment ./cmd/phlame-components-experiment
go build -trimpath -buildvcs=false -o bin/phlame-components-profile ./cmd/phlame-components-profile
python3 experiments/paired-seed-sweep/study.py --out work/paired-seed-reproduction
go test ./...
go vet ./...
```

For one configuration:

```sh
bin/phlame-components-experiment -branching -hidden-width 4 -branch-seed 7 \
  -branch-init paired-centered -out work/paired-seven-reproduction -repeats 1
```

Each candidate trains twice to verify bit-identical weights, freezes with signed
initialization lineage, reloads without source and attempts full certification.
The source/contract oracle compares all primitives; raw predictions cover all
594 accumulator inputs. A failed certificate stops at its first counterexample,
executes no certified transaction and does not grant a smaller partial domain.
Concrete mode may complete only schedules whose every numerical proposal passes
the exact relation; it never repairs outputs.

## Source-free reload and evidence

```sh
bin/phlame-components-worker experiments/paired-seed-sweep/run-0001/trials/w4-s7/registry.json \
  experiments/paired-seed-sweep/run-0001/trials/w4-s7/trusted-key.hex learned certified \
  < experiments/paired-seed-sweep/run-0001/common/request.json
```

The examples demonstrate upper/lower saturation, a threshold and failure after
both components stage writes. Counter reaches 1, accumulator 16 successfully
writes 16, then the returned value exceeds Counter's argument domain. Both states
and versions remain original, and no application results or writes publish.
The source oracle accepts that schedule; rejection is a learned-domain restriction.
Debug source traces are retained separately from application outcomes.

All trials retain signed accumulator/Counter artifacts, key, root-pinned registry,
raw accumulator predictions, report and example response. Shared training data,
source observations, workload, request and oracle traces live once in common/;
evidence-refs.json pins their paths and hashes after byte-equality checks.

The worker runs with binary, signed modules, manifest and public key, without
source implementation, compiler or reference oracle. Control profiling additionally
receives oracle-observation data only to validate outcomes before timing; expected
values never enter model inputs. Contract specifications remain in artifacts.
Certificates are rebuilt process-locally, not imported from previous runs.

## Runtime comparisons and checker work

The first certifying candidate is compared with the frozen 4-unit seed-404 model
and lookup using one current profiling binary and one common frozen Counter.
All contracts, domains, examples, optimizer settings and initialization mode
match across learned variants. Only the accumulator's training seed/weights differ.
Counter retains its own seed and independently trained weights.

180 shuffled fresh-process measurements cover three variants, both boundaries,
clocks on/off, three workload groups and five repetitions. Each measures 100,000
transactions after validation, warm-up and GC with GOMAXPROCS=1. Three separate
CPU/allocation diagnostic processes use 1.5 million successful certified
transactions each. Diagnostic timings do not enter latency summaries.

Startup build, warm checker work, lookup application work and transaction phases
remain separate. Certified warm checking evaluates zero contract arithmetic,
predicates or output comparisons. Concrete lookup coalesces its relation
computation with checking; certified lookup charges that computation as
application work. All frame, binding, domain and atomic-publication rules remain.

Successful certified transactions without phase clocks have median process means
of 1.507 µs for seed 7, 1.486 µs for seed 404 and 1.462 µs for lookup. Process
ranges overlap; no speed improvement or statistical equivalence is claimed.
Allocations and exact checker counts are identical. Outer timing includes loop
aggregation and retaining the last outcome; JSON, startup, validation and
full-batch retention are excluded. There is no CPU pinning or confidence interval.

This bounded result does not establish minimum network capacity, broad initializer
reliability, learned planning or correctness outside the certified finite domain.
Source independence is not OS isolation. Public research keys are not production
identities or permissions. Prior models and reports remain unchanged.
