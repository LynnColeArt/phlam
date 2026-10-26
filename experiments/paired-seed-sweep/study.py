#!/usr/bin/env python3
"""Fixed paired-centered seed grid; freeze all trials before certification."""
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

WIDTHS = (2, 4)
SEEDS = (0, 1, 2, 3, 4, 5, 6, 7, 404)


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
    parser.add_argument("--control-study", type=Path, default=Path("experiments/training-dynamics/run-0001/trials/w4-paired-centered"))
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
    common = args.out / "common"
    common.mkdir()
    shared = ["accumulator-training.json", "accumulator-primitive.jsonl", "counter-training.json",
              "counter-primitive.jsonl", "counter-predictions.jsonl", "transactions.jsonl",
              "request.json", "oracle-traces.json"]
    for width, seed in itertools.product(WIDTHS, SEEDS):
        trial = f"w{width}-s{seed}"
        directory = args.out / "trials" / trial
        directory.mkdir(parents=True)
        with tempfile.TemporaryDirectory(prefix="phlame-seed-evidence-") as temp:
            full = Path(temp) / "full"
            subprocess.run([str(trainer), "-branching", "-hidden-width", str(width),
                "-branch-seed", str(seed), "-branch-init", "paired-centered", "-worker", str(worker), "-out", str(full), "-repeats", "1"], check=True)
            refs = {}
            for file in full.iterdir():
                if file.name in shared:
                    target = common / file.name
                    if target.exists() and target.read_bytes() != file.read_bytes():
                        raise RuntimeError("shared evidence differs: " + file.name)
                    if not target.exists():
                        shutil.copyfile(file, target)
                    refs[file.name] = {"file": "../../common/" + file.name, "sha256": digest(file)}
                else:
                    shutil.copyfile(file, directory / file.name)
            save(directory / "evidence-refs.json", refs)
        report = json.loads((directory / "report.json").read_text())
        module = report["independent_modules"]["accumulator"]
        if module["seed"] != seed or module["hidden_width"] != width or module["training_samples"] != 102 or module["training_initialization"] != "paired-centered" or module["prefit_epochs"] != 0:
            raise RuntimeError("trainer ignored a fixed trial parameter")
        certified = report["engines"]["learned/certified"]
        passed = (not certified.get("preflight_error") and
                  module["raw_training_correct"] == module["training_samples"] and
                  module["raw_heldout_correct"] == module["heldout_cases"] and
                  certified["accuracy"]["accepted"] == 224 and
                  certified["accuracy"]["supported_rejected"] == 0 and
                  all(c["certified"] for c in certified["certificates"].values()))
        survey.append({"trial": trial, "width": width, "seed": seed, "module": module,
                       "fully_certified": passed, "preflight_error": certified.get("preflight_error"),
                       "certificates": certified["certificates"],
                       "concrete_accuracy": report["engines"]["learned/concrete"]["accuracy"],
                       "certified_accuracy": certified["accuracy"],
                       "lookup_accuracy": report["engines"]["lookup/certified"]["accuracy"]})
        print(f"trial {trial}: raw={module['raw_heldout_correct']}/492 certified={passed}", flush=True)
    selected = {str(width): next((r["trial"] for r in survey if r["width"] == width and r["fully_certified"]), None) for width in WIDTHS}
    selection = {
        "schema": "phlame.paired-seed-sweep.v0", "initialization": "paired-centered", "candidate_widths": WIDTHS, "seeds": SEEDS,
        "epochs": 24000, "training_samples": 102, "selected_by_width": selected,
        "selection_rule": "first fully certified success per width in predeclared seed order; no latency-based selection",
        "selection_uses_full_domain": True, "candidates": survey,
        "binary_sha256": {"trainer": digest(trainer), "worker": digest(worker), "profiler": digest(profiler)},
        "toolchain": subprocess.check_output(["go", "version"], text=True).strip(),
    }
    save(args.out / "selection.json", selection)
    control = args.control_study
    base = args.out / "trials" / survey[0]["trial"]
    if (base / "trusted-key.hex").read_bytes() != (control / "trusted-key.hex").read_bytes():
        raise RuntimeError("research trust keys differ")
    if (common / "transactions.jsonl").read_bytes() != (control / "../../common/transactions.jsonl").read_bytes():
        raise RuntimeError("oracle workload differs")
    control_artifact = json.loads((control / "accumulator.json").read_text())
    counter_artifact = json.loads((base / "counter.json").read_text())
    old_counter = json.loads((control / "counter.json").read_text())
    for field in ["model", "contract", "domain", "training_sha256", "seed", "epochs"]:
        if old_counter["payload"][field] != counter_artifact["payload"][field]:
            raise RuntimeError("Counter control changed: " + field)
    if len(control_artifact["payload"]["model"]["hidden"]) != 4 or control_artifact["payload"].get("training_initialization") != "paired-centered":
        raise RuntimeError("control is not the frozen paired-centered four-unit model")
    variants = {"learned-4-control": ("learned", "registry-control.json"),
                "lookup": ("lookup", "registry-control.json")}
    artifacts = {"control": (control / "accumulator.json", control_artifact)}
    for width, trial in selected.items():
        if trial is not None:
            file = args.out / "trials" / trial / "accumulator.json"
            artifacts[trial] = (file, json.loads(file.read_text()))
            variants["learned-" + trial] = ("learned", "registry-" + trial + ".json")
    for record in survey:
        artifact = json.loads((args.out / "trials" / record["trial"] / "accumulator.json").read_text())
        for field in ["contract", "domain", "training_sha256", "oracle_domain_sha256", "epochs", "training_initialization"]:
            if artifact["payload"][field] != control_artifact["payload"][field]:
                raise RuntimeError("comparison changes a fixed input: " + field)
    results, profiles = [], []
    metadata = {
        "selected_by_width": selected, "iterations": args.iterations, "repetitions": args.repetitions,
        "profile_iterations": args.profile_iterations, "order_seed": 20261025,
        "common_counter_module_sha256": counter_artifact["sha256"],
        "module_sha256": {name: artifact["sha256"] for name, (_, artifact) in artifacts.items()},
        "oracle_observations_sha256": digest(common / "transactions.jsonl"),
        "profile_binary_sha256": digest(profiler),
        "counter_control": "one common frozen Counter for all variants; weights and contract match the earlier control",
        "profile_comparison_role": "successful sweep candidates, if any, plus frozen four-unit and lookup controls",
    }
    with tempfile.TemporaryDirectory(prefix="phlame-seed-source-free-") as directory:
        work = Path(directory)
        for name in ["counter.json", "trusted-key.hex"]:
            shutil.copyfile(base / name, work / name)
        shutil.copyfile(common / "transactions.jsonl", work / "transactions.jsonl")
        shutil.copyfile(profiler, work / "worker")
        (work / "worker").chmod(0o755)
        for name, (file, artifact) in artifacts.items():
            shutil.copyfile(file, work / f"accumulator-{name}.json")
            save(work / f"registry-{name}.json", {
                "abi": artifact["payload"]["abi"], "modules": {
                    "counter": {"file": "counter.json", "module_sha256": counter_artifact["sha256"]},
                    "accumulator": {"file": f"accumulator-{name}.json", "module_sha256": artifact["sha256"]}}})
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
    print(f"saved {args.out}: certified sweep selections={selected}; all measured variants preserve oracle behavior and rollback", flush=True)


if __name__ == "__main__":
    main()
