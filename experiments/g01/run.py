#!/usr/bin/env python3
"""G01 acceptance harness. Fixture mathematics stays outside the generic builder."""
import argparse
import hashlib
import itertools
import json
from pathlib import Path
import shutil
import statistics
import subprocess
import tempfile


def h(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n")


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--tools", type=Path, default=Path("bin"))
    args = parser.parse_args()
    if args.out.exists():
        parser.error("output must be new")
    args.out.mkdir(parents=True)
    tools = args.tools.resolve()
    campaign = json.loads(Path("experiments/g01/campaign.json").read_text())
    report = {"schema": "phlame.g01.acceptance.v0", "projects": {}, "binary_sha256": {name: h(tools / name) for name in ["phlam", "phlam-train", "phlam-runtime"]}, "campaign_sha256": h(Path("experiments/g01/campaign.json"))}
    for project in campaign["projects"]:
        path = Path(project["manifest"])
        assert h(path) == project["manifest_sha256"]
        for name, digest in project["locked_files"].items():
            assert h(path.parent / name) == digest
        variant = path.parent.name
        target = args.out / variant
        target.mkdir()
        with tempfile.TemporaryDirectory(prefix="phlame-g01-clean-") as temp:
            temporary = Path(temp)
            key = temporary / "development-seed.hex"
            key.write_text(hashlib.sha256(b"PHLAME G01 PUBLIC RESEARCH KEY").hexdigest())
            builds = []
            for number in range(2):
                src = temporary / f"source-{number}"
                shutil.copytree(path.parent, src)
                output = target / "deployment" if number == 0 else temporary / "rebuild"
                evidence = target / "evidence" if number == 0 else temporary / "rebuild-evidence"
                command = [str(tools / "phlam"), "build", str(src / "phlam.mod"), "--out", str(output), "--evidence", str(evidence), "--signing-key", str(key)]
                completed = subprocess.run(command, capture_output=True, text=True)
                if completed.returncode:
                    raise RuntimeError(completed.stderr + completed.stdout)
                builds.append(json.loads(completed.stdout))
                if number == 1:
                    first = {f.name: h(f) for f in (target / "deployment").iterdir()}
                    second = {f.name: h(f) for f in output.iterdir()}
                    assert first == second
            pub = target / "trusted-development-key.hex"
            shutil.copyfile(target / "evidence/development-public-key.hex", pub)
            manifest = json.loads((target / "deployment/bundle.json").read_text())["payload"]
            artifacts = {alias: json.loads((target / "deployment" / binding["file"]).read_text())["payload"] for alias, binding in manifest["modules"].items()}
            cases, expected = [], {}
            for alias, component in builds[0]["components"].items():
                observations = json.loads((target / "evidence" / alias / "observations.json").read_text())
                for index, o in enumerate(observations):
                    op = artifacts[alias]["contract"]["operations"][o["operation_index"]]
                    want, accepted, write = o["state"], True, op["arity"] == 1
                    if write:
                        want += o["argument"]
                        if alias == "saturating":
                            want = max(-16, min(16, want))
                        if alias == "accumulator":
                            accepted = -16 <= want <= 16
                    # An independent specification checks actual source observations.
                    assert accepted == o["source_accepted"]
                    if accepted:
                        assert o["prediction"] == {"next": want, "result": want, "write": write}
                    tx = {"id": f"{alias}-{index}", "instances": {"x": {"component": alias, "value": o["state"], "version": 1}}, "steps": [{"target": "x", "operation": op["name"]}]}
                    if write:
                        tx["steps"][0]["argument"] = {"literal": o["argument"]}
                    cases.append(tx)
                    expected[tx["id"]] = {"accepted": accepted, "states": {"x": {"component": alias, "value": want if accepted else o["state"], "version": 2 if accepted and write else 1}}, "results": [{"type": "int64", "value": want, "producer": "x"}] if accepted else [], "writes": [{"target": "x", "slot": artifacts[alias]["contract"]["slot"], "value": want}] if accepted and write else []}
            def extra(tx, accepted, values=None):
                cases.append(tx)
                expected[tx["id"]] = {"accepted": accepted, "states": tx["instances"], "results": [], "writes": []}
                if accepted:
                    result = expected[tx["id"]]
                    result["states"] = {key: dict(value, value=values[key], version=2) for key, value in tx["instances"].items()}
                    result["results"] = [{"type": "int64", "value": values[step["target"]], "producer": step["target"]} for step in tx["steps"]]
                    result["writes"] = [{"target": step["target"], "slot": artifacts[tx["instances"][step["target"]]["component"]]["contract"]["slot"], "value": values[step["target"]]} for step in tx["steps"]]
            base = {"c": {"component": "counter", "value": 0, "version": 1}, "s": {"component": "saturating", "value": 16, "version": 1}}
            steps = [{"target": "c", "operation": "increment", "argument": {"literal": 1}}, {"target": "s", "operation": "add", "argument": {"from_result": 0}}]
            extra({"id": "successful-composition", "instances": base, "steps": steps}, True, {"c": 1, "s": 16})
            extra({"id": "late-rollback", "instances": base, "steps": steps + [{"target": "c", "operation": "increment", "argument": {"from_result": 1}}]}, False)
            extra({"id": "native-rollback", "instances": {"c": {"component": "counter", "value": 3, "version": 1}, "a": {"component": "accumulator", "value": 15, "version": 1}}, "steps": [steps[0], {"target": "a", "operation": "add", "argument": {"from_result": 0}}]}, False)
            for name, value in [("missing-arg", None), ("forward-result", {"from_result": 0}), ("outside-argument", {"literal": 100})]:
                step = {"target": "c", "operation": "increment"}
                if value is not None:
                    step["argument"] = value
                extra({"id": name, "instances": {"c": base["c"]}, "steps": [step]}, False)
            extra({"id": "unknown-entry", "instances": {"c": base["c"]}, "steps": [{"target": "c", "operation": "init"}]}, False)
            extra({"id": "outside-state", "instances": {"c": {"component": "counter", "value": 17, "version": 1}}, "steps": [{"target": "c", "operation": "read"}]}, False)
            save(target / "requests.json", {"transactions": cases, "repeats": 5})
            save(target / "expected.json", expected)
            metrics = {}
            behavior_hashes = set()
            for engine, boundary, clocks in itertools.product(["learned", "lookup"], ["concrete", "certified"], [True, False]):
                with tempfile.TemporaryDirectory(prefix="phlame-g01-source-free-") as directory:
                    run = Path(directory)
                    for file in (target / "deployment").iterdir():
                        shutil.copyfile(file, run / file.name)
                    shutil.copyfile(pub, run / "trusted-key.hex")
                    shutil.copyfile(tools / "phlam-runtime", run / "runtime")
                    (run / "runtime").chmod(0o755)
                    inventory = sorted(f.name for f in run.iterdir())
                    output = subprocess.check_output([str(run / "runtime"), "run", ".", "--key", "trusted-key.hex", "--engine", engine, "--boundary", boundary, "--phase-clocks=" + str(clocks).lower()], input=(target / "requests.json").read_bytes(), cwd=run, env={"GOMAXPROCS": "1"})
                response = json.loads(output)
                assert not response.get("preflight_error")
                assert len(response["outcomes"]) == len(cases) * 5
                first = []
                relation = {key: 0 for key in ["expressions", "integer_arithmetic", "predicates", "output_comparisons"]}
                lookup = dict(relation)
                phase = {key: [] for key in ["binding_ns", "proposal_ns", "checker_ns", "commit_ns", "overhead_ns", "total_ns"]}
                for number, outcome in enumerate(response["outcomes"]):
                    want = expected[outcome["id"]]
                    assert outcome["accepted"] == want["accepted"]
                    assert outcome["states"] == want["states"]
                    assert outcome.get("results", []) == want["results"]
                    assert outcome.get("writes", []) == want["writes"]
                    if outcome["id"] == "late-rollback":
                        assert outcome["failed_step"] == 2
                    for key in relation:
                        relation[key] += outcome["checker_work"]["relations"][key]
                        lookup[key] += outcome["lookup_application_work"][key]
                    for key in phase:
                        phase[key].append(outcome["timing"][key])
                    if number < len(cases):
                        first.append({"id": outcome["id"], "accepted": outcome["accepted"], "states": outcome["states"], "results": outcome.get("results", []), "writes": outcome.get("writes", [])})
                if boundary == "certified":
                    assert not any(relation.values())
                behavior_hashes.add(hashlib.sha256(json.dumps(first, sort_keys=True).encode()).hexdigest())
                metrics[f"{engine}/{boundary}/clocks={clocks}"] = {"cases": len(cases), "accepted": sum(e["accepted"] for e in first), "rejected": sum(not e["accepted"] for e in first), "runtime_load_ns": response["load_ns"], "certificates": response["certificates"], "checker_relations": relation, "lookup_application_work": lookup, "phase_mean_ns": {key: statistics.mean(values) for key, values in phase.items()}, "entry_check_ns_mean": statistics.mean(e["ns"] for e in response["entry_checks"]), "entry_checks": {key: sum(e["work"][key] for e in response["entry_checks"]) for key in ["instance_bindings", "step_targets", "entry_comparisons"]}, "source_free_files": inventory}
            assert len(behavior_hashes) == 1
            value = {"deployment_sha256": {f.name: h(f) for f in (target / "deployment").iterdir()}, "two_clean_builds_identical": True, "behavior_sha256": next(iter(behavior_hashes)), "metrics": metrics}
            save(target / "comparison.json", value)
            report["projects"][variant] = value
            print(f"{variant}: two clean builds identical; {len(cases)} scenarios match oracle/math and lookup in all modes", flush=True)
    failures = {}
    for kind in ["lock-mismatch", "undeclared-source", "invalid-domain", "unknown-recipe", "unsupported-source", "failed-candidate", "training-timeout"]:
        target = args.out / "failures" / kind
        target.mkdir(parents=True)
        with tempfile.TemporaryDirectory(prefix="phlame-g01-negative-") as directory:
            src = Path(directory) / "source"
            shutil.copytree("examples/g01/original", src)
            key = Path(directory) / "seed.hex"
            key.write_text(hashlib.sha256(b"PHLAME G01 PUBLIC RESEARCH KEY").hexdigest())
            manifest = json.loads((src / "phlam.mod").read_text())
            plan = json.loads((src / "training.json").read_text())
            if kind == "lock-mismatch":
                (src / "counter.phlam").write_text("changed")
            elif kind == "undeclared-source":
                manifest["components"][0]["source"] = "ambient.phlam"
            elif kind == "invalid-domain":
                manifest["components"][0]["domain"]["state_max"] = 1000
            elif kind == "unknown-recipe":
                plan["candidates"][0]["kind"] = "application-handler"
            elif kind == "unsupported-source":
                f = src / "counter.phlam"
                f.write_text(f.read_text().replace("$this->value += $delta", "$this->value = eval('1')"))
            elif kind == "failed-candidate":
                plan["candidates"] = [{"kind": "relu", "width": 2, "initialization": "legacy", "seeds": [404]}]
            elif kind == "training-timeout":
                plan["candidate_timeout_ms"] = 1
            save(src / "phlam.mod", manifest)
            save(src / "training.json", plan)
            if kind != "lock-mismatch":
                save(src / "phlam.lock", {"schema": "phlame.lock.v0", "files": {name: h(src / name) for name in manifest["sources"] + [manifest["training_plan"]]}})
            command = [str(tools / "phlam"), "build", str(src / "phlam.mod"), "--out", str(target / "deployment"), "--evidence", str(target / "evidence"), "--signing-key", str(key)]
            result = subprocess.run(command, text=True, capture_output=True)
            assert result.returncode != 0 and not (target / "deployment").exists()
            build_report = json.loads(result.stdout)
            assert build_report["status"] == "failed"
            failures[kind] = {"exit_code": result.returncode, "activation_absent": True, "error": build_report.get("error"), "report": "failures/" + kind + "/evidence/report.json"}
            save(target / "mutation.json", {"kind": kind, "manifest": manifest, "training_plan": plan})
    report["failures"] = failures
    save(args.out / "report.json", report)
    print(f"saved {args.out}")


if __name__ == "__main__":
    main()
