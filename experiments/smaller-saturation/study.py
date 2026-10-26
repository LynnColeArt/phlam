#!/usr/bin/env python3
"""Fixed-width training survey followed by one matched source-free comparison."""
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

WIDTHS = (2, 4, 8)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--trainer", type=Path, default=Path("bin/phlame-components-experiment"))
    parser.add_argument("--worker", type=Path, default=Path("bin/phlame-components-worker"))
    parser.add_argument("--profiler", type=Path, default=Path("bin/phlame-components-profile"))
    parser.add_argument("--large-study", type=Path, default=Path("experiments/saturating-accumulator/run-0001"))
    parser.add_argument("--iterations", type=int, default=100000)
    parser.add_argument("--repetitions", type=int, default=5)
    parser.add_argument("--profile-iterations", type=int, default=1500000)
    args = parser.parse_args()
    if args.out.exists():
        parser.error("output must be fresh")
    if not (1 <= args.iterations <= 10000000 and 1 <= args.repetitions <= 20 and 1 <= args.profile_iterations <= 10000000):
        parser.error("iteration/repetition bounds")
    trainer, worker, profiler = args.trainer.resolve(), args.worker.resolve(), args.profiler.resolve()
    args.out.mkdir(parents=True)
    survey = []
    for width in WIDTHS:
        directory = args.out / f"width-{width}"
        subprocess.run([str(trainer), "-branching", "-hidden-width", str(width), "-worker", str(worker), "-out", str(directory), "-repeats", "1"], check=True)
        report = json.loads((directory / "report.json").read_text())
        module = report["independent_modules"]["accumulator"]
        certified = report["engines"]["learned/certified"]
        passed = (not certified.get("preflight_error") and
                  module["raw_training_correct"] == module["training_samples"] and
                  module["raw_heldout_correct"] == module["heldout_cases"] and
                  certified["accuracy"]["accepted"] == 224 and
                  certified["accuracy"]["supported_rejected"] == 0 and
                  all(c["certified"] for c in certified["certificates"].values()))
        survey.append({"width": width, "module": module, "fully_certified": passed,
                       "preflight_error": certified.get("preflight_error"),
                       "concrete_accuracy": report["engines"]["learned/concrete"]["accuracy"],
                       "certified_accuracy": certified["accuracy"]})
    passed = [r for r in survey if r["fully_certified"]]
    selection = {
        "schema": "phlame.smaller-saturation.v0", "candidate_widths": WIDTHS,
        "seed": 404, "epochs": 24000, "training_samples": 102,
        "selection_rule": "smallest predeclared width that fully certifies and matches all supported oracle compositions",
        "selection_uses_full_domain": True, "candidates": survey,
        "selected_width": min((r["width"] for r in passed), default=None),
        "binary_sha256": {"trainer": digest(trainer), "worker": digest(worker), "profiler": digest(profiler)},
        "toolchain": subprocess.check_output(["go", "version"], text=True).strip(),
    }
    save(args.out / "selection.json", selection)
    if not passed:
        raise RuntimeError("No predeclared smaller candidate certified; all failure evidence saved")
    width = selection["selected_width"]
    selected = args.out / f"width-{width}"
    large = args.large_study
    if (selected / "trusted-key.hex").read_bytes() != (large / "trusted-key.hex").read_bytes():
        raise RuntimeError("research trust keys differ")
    if (selected / "transactions.jsonl").read_bytes() != (large / "transactions.jsonl").read_bytes():
        raise RuntimeError("oracle workload differs")
    small_artifact = json.loads((selected / "accumulator.json").read_text())
    large_artifact = json.loads((large / "accumulator.json").read_text())
    counter_artifact = json.loads((selected / "counter.json").read_text())
    for field in ["contract", "domain", "training_sha256", "oracle_domain_sha256", "seed", "epochs"]:
        if small_artifact["payload"][field] != large_artifact["payload"][field]:
            raise RuntimeError("comparison changes more than model architecture/lineage: " + field)
    old_counter = json.loads((large / "counter.json").read_text())
    for field in ["model", "contract", "domain", "training_sha256", "seed", "epochs"]:
        if old_counter["payload"][field] != counter_artifact["payload"][field]:
            raise RuntimeError("Counter control changed: " + field)
    if len(large_artifact["payload"]["model"]["hidden"]) != 16:
        raise RuntimeError("large model is not the frozen 16-unit control")
    variants = {f"learned-{width}": ("learned", "registry-small.json"),
                "learned-16": ("learned", "registry-large.json"),
                "lookup": ("lookup", "registry-small.json")}
    results, profiles = [], []
    metadata = {
        "selected_width": width, "iterations": args.iterations, "repetitions": args.repetitions,
        "profile_iterations": args.profile_iterations, "order_seed": 20261025,
        "common_counter_module_sha256": counter_artifact["sha256"],
        "small_module_sha256": small_artifact["sha256"], "large_module_sha256": large_artifact["sha256"],
        "oracle_observations_sha256": digest(selected / "transactions.jsonl"),
        "profile_binary_sha256": digest(profiler),
        "counter_control": "same frozen Counter file for all three variants; numerical weights match the original control",
        "large_model_file_sha256": digest(large / "accumulator.json"),
        "small_model_file_sha256": digest(selected / "accumulator.json"),
    }
    with tempfile.TemporaryDirectory(prefix="phlame-smaller-source-free-") as directory:
        work = Path(directory)
        for name in ["counter.json", "trusted-key.hex", "transactions.jsonl"]:
            shutil.copyfile(selected / name, work / name)
        shutil.copyfile(selected / "accumulator.json", work / "accumulator-small.json")
        shutil.copyfile(large / "accumulator.json", work / "accumulator-large.json")
        shutil.copyfile(profiler, work / "worker")
        (work / "worker").chmod(0o755)
        for size, artifact in [("small", small_artifact), ("large", large_artifact)]:
            save(work / f"registry-{size}.json", {
                "abi": artifact["payload"]["abi"], "modules": {
                    "counter": {"file": "counter.json", "module_sha256": counter_artifact["sha256"]},
                    "accumulator": {"file": f"accumulator-{size}.json", "module_sha256": artifact["sha256"]}}})
        metadata["source_free_files"] = sorted(p.name for p in work.iterdir())
        jobs = list(itertools.product(range(args.repetitions), variants, ["concrete", "certified"], [False, True], ["mixed", "accepted", "scope_rejected"]))
        random.Random(metadata["order_seed"]).shuffle(jobs)
        for number, (repeat, variant, boundary, clocks, group) in enumerate(jobs):
            engine, registry = variants[variant]
            raw = subprocess.check_output([str(work / "worker"), "-registry", registry,
                    "-engine", engine, "-boundary", boundary, "-phase-clocks=" + str(clocks).lower(),
                    "-group", group, "-iterations", str(args.iterations)], cwd=work, env={"GOMAXPROCS": "1"}, text=True)
            result = json.loads(raw)
            result.update(variant=variant, repetition=repeat, execution_order=number)
            results.append(result)
            if number % 12 == 0:
                print(f"matched measurement {number + 1}/{len(jobs)}", flush=True)
        for variant, (engine, registry) in variants.items():
            raw = subprocess.check_output([str(work / "worker"), "-registry", registry,
                    "-engine", engine, "-group", "accepted", "-iterations", str(args.profile_iterations),
                    "-cpuprofile", "cpu.pprof", "-memprofile", "allocs.pprof"], cwd=work, env={"GOMAXPROCS": "1"}, text=True)
            result = json.loads(raw)
            result["variant"] = variant
            profiles.append(result)
            for kind, flags in [("cpu", []), ("allocs", ["-alloc_space"])]:
                target = args.out / f"{variant}.{kind}.pprof"
                shutil.copyfile(work / f"{kind}.pprof", target)
                top = subprocess.check_output(["go", "tool", "pprof", "-top", "-nodecount=25", *flags, str(profiler), str(target)], text=True)
                (args.out / f"{variant}.{kind}.txt").write_text(top)
            print(f"profiled {variant}", flush=True)
    if len({r["behavior_sha256"] for r in results + profiles}) != 1:
        raise RuntimeError("models/engines/boundaries/clocks changed semantic behavior")
    for result in results + profiles:
        if (result["verified_cases"], result["verified_accepted"], result["verified_rejected"]) != (750, 224, 526):
            raise RuntimeError("source-oracle/rollback coverage changed")
        if result["boundary"] == "certified" and any(result["total_checker_work"]["relations"].values()):
            raise RuntimeError("certified boundary recomputes transition")
    summary = []
    for variant, boundary, clocks, group in itertools.product(variants, ["concrete", "certified"], [False, True], ["mixed", "accepted", "scope_rejected"]):
        rows = [r for r in results if (r["variant"], r["boundary"], r["phase_clocks"], r["group"]) == (variant, boundary, clocks, group)]
        cell = {"variant": variant, "boundary": boundary, "phase_clocks": clocks, "group": group}
        for field in ["ns_per_transaction", "bytes_per_transaction", "allocations_per_transaction"]:
            values = [r[field] for r in rows]
            cell[field] = {"median": statistics.median(values), "min": min(values), "max": max(values)}
        cell["phase_mean_ns"] = {k: statistics.median(r["total_phase_ns"][k] / r["iterations"] for r in rows) for k in rows[0]["total_phase_ns"]}
        for field in ["total_checker_work", "total_lookup_application_work"]:
            if len({json.dumps(r[field], sort_keys=True) for r in rows}) != 1:
                raise RuntimeError("work differs across repetitions")
            cell[field] = rows[0][field]
        summary.append(cell)
    for boundary, group in itertools.product(["concrete", "certified"], ["mixed", "accepted", "scope_rejected"]):
        if len({json.dumps(r["total_checker_work"], sort_keys=True) for r in results if r["boundary"] == boundary and r["group"] == group}) != 1:
            raise RuntimeError("comparison changed checker work")
    for name, value in [("measurements.json", results), ("profile-runs.json", profiles), ("summary.json", summary), ("metadata.json", metadata)]:
        save(args.out / name, value)
    print(f"saved {args.out}: selected width {width}, all three variants preserve oracle behavior and rollback", flush=True)


if __name__ == "__main__":
    main()
