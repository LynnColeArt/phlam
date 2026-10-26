# PHLAME: bounded low-width seed sweep

[Recorded report](run-0001/REPORT.md) · [All 18 trial results](run-0001/selection.json) ·
[Control measurements](run-0001/summary.json)

Widths 2 and 4 were trained with seeds **0,1,2,3,4,5,6,7,404**, including the
previous failing seed as a control. **None of the 18 configurations fully
certified.** The training examples, source, domains, 24,000-epoch optimizer and
inference engine were held fixed. All signed artifacts, raw predictions and
first counterexamples are retained.

Best nontraining accuracy is 310/492 for either width. Every run also misses
training cases; the best training accuracy is 63/102. These results do not prove
that the architectures cannot represent saturation. They show that this bounded
seed search did not fix training under the current setup. The previously
certified 8-unit model remains the smallest established success in these studies.

Every model was frozen and reloaded in a source-free worker before applying the
unchanged full-domain certification requirement. Certification stops at the
first wrong transition, executes zero transactions on failure, and never grants
a partial certificate. Raw predictions are nevertheless recorded on all 594
primitive inputs. The domain was neither reduced nor patched to obtain a pass.

Concrete mode checks every proposal and safely completes 68–126 of the 224
supported schedules, depending on the failed model; all accepted results match
the source oracle. Lookup completes all 224 in every trial. Every rejected
transaction restores original state and publishes no staged returns or writes.
An 8-unit/lookup control comparison separately preserves all 224 supported
successes and 526 domain rollbacks, with identical exact checker work.

A later [training-dynamics study](../training-dynamics/README.md) reproduces
these failures and tests two fixed initialization interventions. Its centered
4-unit treatment certifies; the negative seed sweep below remains unchanged.

## Reproduce

From the repository root, with Go and Python 3:

```sh
go build -trimpath -buildvcs=false -o bin/phlame-components-worker ./cmd/phlame-components-worker
go build -trimpath -buildvcs=false -o bin/phlame-components-experiment ./cmd/phlame-components-experiment
go build -trimpath -buildvcs=false -o bin/phlame-components-profile ./cmd/phlame-components-profile
python3 experiments/seed-sweep/study.py --out work/seed-sweep-reproduction
go test ./...
go vet ./...
```

Use a fresh output directory. The script fixes both widths and all nine seeds;
it does not continue searching after failure or tune the optimizer. It would
select the first fully certified seed per width in declared order, before any
latency comparison. Both selections are null in this run. An exploratory pass
used the same eight new seeds/settings before the recorded run; no settings were
changed. Each recorded candidate trains twice to verify bit-identical weights.

A single explicit configuration can be reproduced with:

```sh
bin/phlame-components-experiment -branching -hidden-width 2 -branch-seed 0 \
  -out work/seed-zero-reproduction -repeats 1
```

The default branch seed remains 404, preserving the previous experiment command.
Counter retains its own seed 101 and separate training examples. Seed is lineage
metadata and optimizer initialization, not an inference feature or authority.

## Evidence layout and source-free reload

Each trials/wN-sS directory retains the signed accumulator and Counter modules,
root-pinned registry, public key, raw accumulator predictions, report and example
response. Byte-identical training data, source observations, Counter predictions,
workload, request and oracle traces are stored once in common/. Each trial's
evidence-refs.json records the relative location and digest of those shared
files; the runner checks byte equality before sharing them. No failure evidence
is omitted or overwritten.

```sh
bin/phlame-components-worker experiments/seed-sweep/run-0001/trials/w2-s0/registry.json \
  experiments/seed-sweep/run-0001/trials/w2-s0/trusted-key.hex learned certified \
  < experiments/seed-sweep/run-0001/common/request.json
```

That command returns preflight_error and no transaction outcomes, as expected.
Changing certified to concrete exposes safe partial coverage, never exact-output
fallback. Common source traces describe source execution, which accepts all 750
schedules; 526 are outside the learned domains. Debug traces and application
outcomes remain separate.

The ordinary worker imports no compiler or reference oracle and runs in a
temporary directory with only its binary, two signed modules, manifest and key.
The control profiler additionally receives oracle-observation data, used solely
to validate outcomes before timing. Source-derived contract specifications remain
inside artifacts. Certificates are rebuilt process-locally; source independence
is not OS sandboxing, and public research signing keys are not production identities.

## Controls and measurements

No failed low-width model enters the successful-execution timing comparison.
The recorded control comparison uses the unchanged frozen 8-unit accumulator,
one common Counter and lookup. It has 120 fresh processes: two variants, both
boundaries, clocks on/off, three workload groups and five repetitions. Each
measures 100,000 transactions after validation, warm-up and GC, in shuffled order
with GOMAXPROCS=1. Two diagnostic processes separately sample CPU/allocations.

Startup certification work, warm checker work, lookup application work, proposal,
binding, commit and overhead remain separate. Both controls evaluate zero warm
contract arithmetic/predicates in certified mode. Concrete lookup coalesces its
application computation with checking. Counter aggregation and a last-result
sink are included in outer latency; JSON, startup and full-batch retention are
excluded. There is no CPU pinning or statistical-significance claim. The timing
controls establish no speed result for the rejected low-width candidates.

Full-domain certification also serves as model selection. Nontraining inputs
are held out of gradient updates, not an independent seed-selection holdout.
This negative bounded result establishes neither minimum network capacity nor
the effectiveness of other seeds, initialization strategies or training regimes.
