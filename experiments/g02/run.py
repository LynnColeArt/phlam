#!/usr/bin/env python3
"""G02 initial scalar reliability: one fit per fresh seed, no fallback credit."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import signal
import subprocess
import tempfile
import time


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, data):
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n")


def tree_rss(pid):
    """Linux /proc sampled simultaneous RSS, summed over live subprocesses."""
    pending, seen, total = [pid], set(), 0
    while pending:
        current = pending.pop()
        if current in seen:
            continue
        seen.add(current)
        try:
            for line in Path(f"/proc/{current}/status").read_text().splitlines():
                if line.startswith("VmRSS:"):
                    total += int(line.split()[1]) * 1024
            for thread in Path(f"/proc/{current}/task").iterdir():
                pending.extend(int(x) for x in (thread / "children").read_text().split())
        except (FileNotFoundError, ProcessLookupError, PermissionError):
            pass
    return total


def supervised(command, stdout_path, stderr_path, budget):
    started = time.monotonic()
    maximum, exceeded = 0, None
    with stdout_path.open("wb") as stdout, stderr_path.open("wb") as stderr:
        process = subprocess.Popen(command, stdout=stdout, stderr=stderr, start_new_session=True, env={**os.environ, "GOMAXPROCS": "1"})
        while process.poll() is None:
            maximum = max(maximum, tree_rss(process.pid))
            if maximum > budget["max_memory_bytes"]:
                exceeded = "sampled process-tree RSS budget"
            elif time.monotonic() - started > budget["total_wall_seconds_per_program"]:
                exceeded = "whole-build wall budget"
            if exceeded:
                os.killpg(process.pid, signal.SIGKILL)
                break
            time.sleep(budget["rss_poll_seconds"])
        code = process.wait()
    return {"exit_code": code, "wall_ns": round((time.monotonic() - started) * 1e9), "sampled_process_tree_peak_rss_bytes": maximum, "budget_exceeded": exceeded}


def project(directory, programs, recipe, seed, campaign, module):
    directory.mkdir()
    sources = []
    components = []
    for p in programs:
        target = directory / p["source"]
        shutil.copyfile(p["source_path"], target)
        assert digest(target) == p["source_sha256"]
        sources.append(p["source"])
        components.append({k: p[k] for k in ["alias", "source", "class", "domain", "entries"]})
    plan = copy.deepcopy(campaign["plan"])
    plan["candidates"] = [dict(recipe, seeds=[seed])]
    save(directory / "training.json", plan)
    save(directory / "phlam.mod", {"schema": "phlame.build.v0", "module": {"name": module, "version": "0.1.0"}, "sources": sources, "training_plan": "training.json", "lock": "phlam.lock", "components": components})
    save(directory / "phlam.lock", {"schema": "phlame.lock.v0", "files": {f: digest(directory / f) for f in sources + ["training.json"]}})


def compare(runtime, bundle, key, request, expected, *, allow_bad_model=False):
    results = {}
    modes = [(e, b) for e in ["learned", "lookup"] for b in ["concrete", "certified"]]
    for engine, boundary in modes:
        with tempfile.TemporaryDirectory(prefix="phlame-g02-source-free-") as directory:
            run = Path(directory)
            for f in bundle.iterdir():
                shutil.copyfile(f, run / f.name)
            shutil.copyfile(runtime, run / "runtime")
            (run / "runtime").chmod(0o755)
            shutil.copyfile(key, run / "key.hex")
            raw = subprocess.check_output([str(run / "runtime"), "run", ".", "--key", "key.hex", "--engine", engine, "--boundary", boundary], input=json.dumps(request).encode(), cwd=run, env={"GOMAXPROCS": "1"})
            inventory = sorted(p.name for p in run.iterdir())
        out = json.loads(raw)
        record = {"preflight_error": out.get("preflight_error"), "certificates": out["certificates"], "load_ns": out["load_ns"], "source_free_files": inventory, "accepted": 0, "rejected": 0, "correct_supported_rejected": 0, "checker_relations": {}, "lookup_application_work": {}, "phase_total_ns": {}, "entry_check_ns": sum(x["ns"] for x in out.get("entry_checks", []))}
        if out.get("preflight_error"):
            assert allow_bad_model and engine == "learned" and boundary == "certified" and not out.get("outcomes")
            results[engine + "/" + boundary] = record
            continue
        assert len(out["outcomes"]) == len(request["transactions"]) * request["repeats"]
        for number, outcome in enumerate(out["outcomes"]):
            tx = request["transactions"][number % len(request["transactions"])]
            wanted = expected[outcome["id"]]
            if outcome["accepted"]:
                assert wanted["accepted"] and outcome["states"] == wanted["states"]
                assert outcome.get("results", []) == wanted["results"] and outcome.get("writes", []) == wanted["writes"]
                record["accepted"] += 1
            else:
                assert outcome["states"] == tx["instances"] and not outcome.get("results") and not outcome.get("writes")
                if wanted["accepted"]:
                    assert allow_bad_model and engine == "learned"
                    record["correct_supported_rejected"] += 1
                record["rejected"] += 1
            for target, values in [("checker_relations", outcome["checker_work"]["relations"]), ("lookup_application_work", outcome["lookup_application_work"]), ("phase_total_ns", outcome["timing"])]:
                for name, value in values.items():
                    record[target][name] = record[target].get(name, 0) + value
        if boundary == "certified":
            assert not any(record["checker_relations"].values())
        results[engine + "/" + boundary] = record
    return results


def single_cases(evidence, program):
    alias = program["alias"]
    report = json.loads((evidence / "report.json").read_text())["components"][alias]
    attempt = report["attempts"][0]
    candidate = evidence / attempt["artifact"]
    artifact = json.loads(candidate.read_text())["payload"]
    observed = json.loads((evidence / alias / "observations.json").read_text())
    cases, expected = [], {}
    for i, o in enumerate(observed):
        op = artifact["contract"]["operations"][o["operation_index"]]
        write, wanted, accepted = op["arity"] == 1, o["state"], True
        if write:
            wanted += o["argument"]
            if alias == "saturating":
                wanted = max(-16, min(16, wanted))
            if alias == "accumulator":
                accepted = -16 <= wanted <= 16
        assert accepted == o["source_accepted"]
        if accepted:
            assert o["prediction"] == {"next": wanted, "result": wanted, "write": write}
        tx = {"id": f"primitive-{i}", "instances": {"x": {"component": alias, "value": o["state"], "version": 1}}, "steps": [{"target": "x", "operation": op["name"]}]}
        if write:
            tx["steps"][0]["argument"] = {"literal": o["argument"]}
        cases.append(tx)
        expected[tx["id"]] = {"accepted": accepted, "states": {"x": {"component": alias, "value": wanted if accepted else o["state"], "version": 2 if accepted and write else 1}}, "results": [{"type": "int64", "value": wanted, "producer": "x"}] if accepted else [], "writes": [{"target": "x", "slot": artifact["contract"]["slot"], "value": wanted}] if accepted and write else []}
    mutator = next(op for op in artifact["contract"]["operations"] if op["arity"] == 1)
    tx = {"id": "rollback-after-write", "instances": {"x": {"component": alias, "value": 0, "version": 1}}, "steps": [{"target": "x", "operation": mutator["name"], "argument": {"literal": 1}}, {"target": "x", "operation": mutator["name"], "argument": {"literal": 100}}]}
    cases.append(tx)
    expected[tx["id"]] = {"accepted": False, "states": tx["instances"], "results": [], "writes": []}
    return candidate.parent, {"transactions": cases, "repeats": 1}, expected


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--tools", type=Path, default=Path("bin"))
    args = parser.parse_args()
    if args.out.exists():
        parser.error("output must be fresh")
    args.out.mkdir(parents=True)
    campaign_path = Path("experiments/g02/campaign.json")
    campaign = json.loads(campaign_path.read_text())
    tools = args.tools.resolve()
    result = {"schema": "phlame.g02.results.v0", "campaign_sha256": digest(campaign_path), "binary_sha256": {name: digest(tools / name) for name in ["phlam", "phlam-train", "phlam-runtime"]}, "trials": []}
    budget_wall = {p["alias"]: 0 for p in campaign["programs"]}
    with tempfile.TemporaryDirectory(prefix="phlame-g02-key-") as directory:
        seedkey = Path(directory) / "key.hex"
        seedkey.write_text(hashlib.sha256(b"PHLAME G02 PUBLIC RESEARCH KEY").hexdigest())
        for strategy in campaign["strategies"]:
            for program in campaign["programs"]:
                for seed in campaign["fresh_seeds"]:
                    name = f'{strategy["id"]}-{program["alias"]}-s{seed}'
                    target = args.out / "trials" / name
                    target.mkdir(parents=True)
                    builds = []
                    comparisons = None
                    for repetition in range(2):
                        evidence = target / f"evidence-{repetition}"
                        output = target / f"deployment-{repetition}"
                        with tempfile.TemporaryDirectory(prefix="phlame-g02-input-") as temp:
                            source = Path(temp) / "source"
                            project(source, [program], strategy["recipe"], seed, campaign, "g02/" + program["alias"])
                            command = [str(tools / "phlam"), "build", str(source / "phlam.mod"), "--out", str(output), "--evidence", str(evidence), "--signing-key", str(seedkey)]
                            resources = supervised(command, target / f"stdout-{repetition}.json", target / f"stderr-{repetition}.txt", campaign["budget"])
                        budget_wall[program["alias"]] += resources["wall_ns"] / 1e9
                        report = json.loads((target / f"stdout-{repetition}.json").read_text())
                        info = report["components"][program["alias"]]
                        assert len(info["attempts"]) == 1
                        attempt = info["attempts"][0]
                        assert attempt["seed"] == seed and attempt["recipe"] == dict(strategy["recipe"], seeds=[seed])
                        within = (resources["budget_exceeded"] is None and attempt["training_cpu_ns"] <= campaign["budget"]["training_cpu_seconds_per_candidate"] * 1e9 and attempt["runtime_verification_process_cpu_ns"] <= campaign["budget"]["verification_cpu_seconds_per_candidate"] * 1e9 and budget_wall[program["alias"]] <= campaign["budget"]["total_wall_seconds_per_program"])
                        passed = resources["exit_code"] == 0 and report["status"] == "built" and attempt["certificate"]["certified"] and within
                        assert output.exists() == (report["status"] == "built")
                        builds.append({"repetition": repetition, "certified_within_budget": passed, "resources": resources, "report": str(evidence.relative_to(args.out) / "report.json"), "attempt": attempt})
                        if repetition == 0:
                            bundle, request, expected = single_cases(evidence, program)
                            save(target / "requests.json", request)
                            save(target / "expected.json", expected)
                            comparisons = compare(tools / "phlam-runtime", bundle, evidence / "development-public-key.hex", request, expected, allow_bad_model=not passed)
                        if repetition == 1:
                            assert builds[0]["certified_within_budget"] == passed
                            assert builds[0]["attempt"]["module_sha256"] == attempt["module_sha256"]
                            first_candidate = target / "evidence-0" / builds[0]["attempt"]["artifact"]
                            assert first_candidate.read_bytes() == (evidence / attempt["artifact"]).read_bytes()
                            assert (first_candidate.parent / "bundle.json").read_bytes() == (evidence / attempt["artifact"]).parent.joinpath("bundle.json").read_bytes()
                    trial = {"strategy": strategy["id"], "program": program["alias"], "seed": seed, "passed_both_builds": all(x["certified_within_budget"] for x in builds), "candidate_bytes_reproduce": True, "builds": builds, "comparisons": comparisons}
                    result["trials"].append(trial)
                    save(target / "result.json", trial)
                    print(f'{name}: {"PASS" if trial["passed_both_builds"] else "FAIL"}', flush=True)
        rates = {strategy["id"]: {program["alias"]: sum(t["passed_both_builds"] for t in result["trials"] if t["strategy"] == strategy["id"] and t["program"] == program["alias"]) for program in campaign["programs"]} for strategy in campaign["strategies"]}
        selected = next((s["id"] for s in campaign["strategies"] if all(n >= campaign["required_certifying_seeds"] for n in rates[s["id"]].values())), None)
        result.update(certifying_seed_counts=rates, selected_strategy=selected, cumulative_program_build_wall_seconds=budget_wall)
        save(args.out / "results.json", result)
        if selected:
            common_seed = next(seed for seed in campaign["fresh_seeds"] if all(any(t["strategy"] == selected and t["program"] == p["alias"] and t["seed"] == seed and t["passed_both_builds"] for t in result["trials"]) for p in campaign["programs"]))
            recipe = next(s["recipe"] for s in campaign["strategies"] if s["id"] == selected)
            for repeat in range(2):
                with tempfile.TemporaryDirectory(prefix="phlame-g02-combined-") as temp:
                    source = Path(temp) / "source"
                    project(source, campaign["programs"], recipe, common_seed, campaign, "g02/combined")
                    output = args.out / f"combined-{repeat}"
                    evidence = args.out / f"combined-evidence-{repeat}"
                    command = [str(tools / "phlam"), "build", str(source / "phlam.mod"), "--out", str(output), "--evidence", str(evidence), "--signing-key", str(seedkey)]
                    subprocess.run(command, check=True, stdout=subprocess.DEVNULL)
            assert {f.name: digest(f) for f in (args.out / "combined-0").iterdir()} == {f.name: digest(f) for f in (args.out / "combined-1").iterdir()}
            request = json.loads(Path("experiments/g01/run-0001/original/requests.json").read_text())
            request["repeats"] = 3
            expected = json.loads(Path("experiments/g01/run-0001/original/expected.json").read_text())
            integration = compare(tools / "phlam-runtime", args.out / "combined-0", args.out / "combined-evidence-0/development-public-key.hex", request, expected)
            save(args.out / "integration.json", {"strategy": selected, "seed": common_seed, "reproducible_deployment": True, "comparisons": integration})
        print(f"fresh-seed counts {rates}; selected {selected}", flush=True)


if __name__ == "__main__":
    main()
