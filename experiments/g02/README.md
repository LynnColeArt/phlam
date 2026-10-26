# G02: fresh-seed training reliability

[Recorded results](run-0001/REPORT.md) · [Locked campaign](campaign.json) · [Machine results](run-0001/results.json) · [Validation](run-0001/validation.json)

The initial scalar milestone meets the planned ≥4/5 fresh-seed target: one uniform 8-unit paired-affine recipe certifies **5/5 seeds separately on Counter, rejecting Accumulator and SaturatingAccumulator**. Every fit reproduces in a second independent build. Both simpler controls remain in the evidence. This is development recipe selection on known source; full G02 remains open for the mandatory richer families and sealed acceptance programs.

From the repository root, with Go and Python 3 installed:

```sh
go build -trimpath -buildvcs=false -o bin/phlam ./cmd/phlam
go build -trimpath -buildvcs=false -o bin/phlam-train ./cmd/phlam-train
go build -trimpath -buildvcs=false -o bin/phlam-runtime ./cmd/phlam-runtime
python3 experiments/g02/run.py --out work/g02-reproduction
go test -count=1 -json ./... > work/g02-tests.jsonl
go vet ./...
python3 experiments/g02/validate.py --run work/g02-reproduction
```

The runner requires a fresh output directory. It uses the five declared seeds without fallback or retries, retains failed candidates, and creates an explicitly public research signing key in temporary storage. Input sources and training plans are copied into each build's evidence; source-free runtime directories receive only signed deployment files, the runtime executable and a separately configured public key. Validation additionally checks combined/individual model identity, clocks on/off, oracle fixture equivalence, rollback and runtime dependency isolation.

Training/verification CPU, build wall time and sampled process-tree RSS are recorded separately. RSS sampling is an observed watchdog, not proof of an OS-enforced peak-memory limit. Timings are diagnostic measurements from this campaign, not a performance benchmark. No optimizer, inference, exact checker or transaction implementation changed for G02.
