#!/usr/bin/env python3
"""Measure frozen workers in randomized process order; no source enters workers."""
import argparse
import hashlib
import itertools
import json
from pathlib import Path
import random
import shutil
import statistics
import subprocess
import tempfile


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--baseline", required=True, type=Path)
    parser.add_argument("--optimized", required=True, type=Path)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--study", type=Path, default=Path("experiments/saturating-accumulator/run-0001"))
    parser.add_argument("--iterations", type=int, default=100000)
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--profile-iterations", type=int, default=1500000)
    args = parser.parse_args()
    if args.out.exists():
        parser.error("output must be a fresh directory")
    if not (1 <= args.iterations <= 10000000 and 1 <= args.profile_iterations <= 10000000 and 1 <= args.repetitions <= 20):
        parser.error("iteration/repetition bounds")
    binaries = {"baseline": args.baseline.resolve(), "optimized": args.optimized.resolve()}
    args.out.mkdir(parents=True)
    results = []
    profile_results = []
    files = ["accumulator.json", "counter.json", "registry.json", "trusted-key.hex", "transactions.jsonl"]
    metadata = {
        "schema": "phlame.transaction-profile.v0", "seed": 20261025,
        "baseline_revision": "065027d", "iterations": args.iterations,
        "repetitions": args.repetitions, "profile_iterations": args.profile_iterations,
        "binary_sha256": {k: digest(v) for k, v in binaries.items()},
        "fixture_sha256": {f: digest(args.study / f) for f in files},
        "toolchain": subprocess.check_output(["go", "version"], text=True).strip(),
        "source_free_files": sorted(files + ["worker"]),
    }
    # Both variants are measured in the same randomized schedule, not two passes
    # in before/after order. Each cell is a fresh process and fresh registry.
    jobs = list(itertools.product(range(args.repetitions), binaries, ["learned", "lookup"], ["concrete", "certified"], [False, True], ["mixed", "accepted", "scope_rejected"]))
    random.Random(metadata["seed"]).shuffle(jobs)
    with tempfile.TemporaryDirectory(prefix="phlame-profile-source-free-") as directory:
        work = Path(directory)
        for name in files:
            shutil.copyfile(args.study / name, work / name)
        for number, (repeat, variant, engine, boundary, clocks, group) in enumerate(jobs):
            shutil.copyfile(binaries[variant], work / "worker")
            (work / "worker").chmod(0o755)
            command = [str(work / "worker"), "-engine", engine, "-boundary", boundary,
                       "-phase-clocks=" + str(clocks).lower(), "-group", group,
                       "-iterations", str(args.iterations)]
            raw = subprocess.check_output(command, cwd=work, env={"GOMAXPROCS": "1"}, text=True)
            result = json.loads(raw)
            result.update(variant=variant, repetition=repeat, execution_order=number)
            results.append(result)
            if number % 12 == 0:
                print(f"measurement {number + 1}/{len(jobs)}", flush=True)
        # CPU/allocation profiles are diagnostic runs, excluded from latency and
        # allocation comparisons. Startup/validation/warm-up precede CPU capture.
        profile_jobs = [(variant, engine, False) for variant, engine in itertools.product(binaries, ["learned", "lookup"])]
        profile_jobs.append(("optimized", "learned", True))
        for variant, engine, clocks in profile_jobs:
            shutil.copyfile(binaries[variant], work / "worker")
            (work / "worker").chmod(0o755)
            command = [str(work / "worker"), "-engine", engine, "-boundary", "certified",
                       "-group", "accepted", "-phase-clocks=" + str(clocks).lower(), "-iterations", str(args.profile_iterations),
                       "-cpuprofile", "cpu.pprof", "-memprofile", "allocs.pprof"]
            raw = subprocess.check_output(command, cwd=work, env={"GOMAXPROCS": "1"}, text=True)
            result = json.loads(raw)
            result["variant"] = variant
            profile_results.append(result)
            for kind, flags in [("cpu", []), ("allocs", ["-alloc_space"])]:
                stem = f"{variant}-{engine}" + ("-clocked" if clocks else "")
                target = args.out / f"{stem}.{kind}.pprof"
                shutil.copyfile(work / f"{kind}.pprof", target)
                top = subprocess.check_output(["go", "tool", "pprof", "-top", "-nodecount=25", *flags, str(binaries[variant]), str(target)], text=True)
                (args.out / f"{stem}.{kind}.txt").write_text(top)
            print(f"profiled {variant}/{engine}, clocks={clocks}", flush=True)
    if len({r["behavior_sha256"] for r in results + profile_results}) != 1:
        raise RuntimeError("behavior differs across worker variants/engines/boundaries/clocks")
    for result in results + profile_results:
        if (result["verified_cases"], result["verified_accepted"], result["verified_rejected"]) != (750, 224, 526):
            raise RuntimeError("oracle/rollback coverage changed")
        if result["boundary"] == "certified" and any(result["total_checker_work"]["relations"].values()):
            raise RuntimeError("certified boundary evaluates transition relations")
    summary = []
    for variant, engine, boundary, clocks, group in itertools.product(binaries, ["learned", "lookup"], ["concrete", "certified"], [False, True], ["mixed", "accepted", "scope_rejected"]):
        rows = [r for r in results if (r["variant"], r["engine"], r["boundary"], r["phase_clocks"], r["group"]) == (variant, engine, boundary, clocks, group)]
        cell = {"variant": variant, "engine": engine, "boundary": boundary, "phase_clocks": clocks, "group": group}
        for field in ["ns_per_transaction", "bytes_per_transaction", "allocations_per_transaction"]:
            values = [r[field] for r in rows]
            cell[field] = {"median": statistics.median(values), "min": min(values), "max": max(values)}
        cell["phase_mean_ns"] = {k: statistics.median(r["total_phase_ns"][k] / r["iterations"] for r in rows) for k in rows[0]["total_phase_ns"]}
        # Counts must be exact across repeat processes for the identical schedule.
        for field in ["total_checker_work", "total_lookup_application_work"]:
            if len({json.dumps(r[field], sort_keys=True) for r in rows}) != 1:
                raise RuntimeError("work counters differ between identical repeats")
            cell[field] = rows[0][field]
        summary.append(cell)
    for name, value in [("measurements.json", results), ("profile-runs.json", profile_results), ("summary.json", summary), ("metadata.json", metadata)]:
        (args.out / name).write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")
    print(f"saved {args.out}; all variants preserve oracle results and rollback", flush=True)


if __name__ == "__main__":
    main()
