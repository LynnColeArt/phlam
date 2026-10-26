#!/usr/bin/env python3
"""Offline audit: no results are fed back to training or certification."""
import argparse
import json
import math
from pathlib import Path


def dot(a, b):
    result = 0.0
    for x, y in zip(a, b):
        result += x * y
    return result


def features(row):
    op = row["operation_index"]
    return [1.0, row["state"] / 32, row["argument"] / 32, float(op == 0), float(op == 1)]


def scores(model, x):
    z = [dot(row, x) for row in model["hidden"]]
    activation = x + [max(0, v) for v in z]
    return z, activation, [dot(row, activation) for row in model["weights"]]


def target(row):
    prediction = row["prediction"]
    return [prediction["next"] / 32, prediction["result"] / 32, float(prediction["write"])]


def loss_gradient(model, samples):
    width = len(model["hidden"])
    gradient = [[0.0] * len(row) for row in model["hidden"] + model["weights"]]
    loss = 0.0
    for row in samples:
        x = features(row)
        z, activation, prediction = scores(model, x)
        delta = []
        for h, wanted in enumerate(target(row)):
            difference = prediction[h] - wanted
            loss += difference * difference
            delta.append(2 * difference / (3 * len(samples)))
            for j, value in enumerate(activation):
                gradient[width + h][j] += delta[h] * value
        for k in range(width):
            if z[k] <= 0:
                continue
            d = 0.0
            for h in range(3):
                d += delta[h] * model["weights"][h][5 + k]
            for j, value in enumerate(x):
                gradient[k][j] += d * value
    return loss / (3 * len(samples)), gradient


def norm(row):
    return math.sqrt(dot(row, row))


def main():
    parser = argparse.ArgumentParser(__doc__)
    parser.add_argument("study", type=Path)
    args = parser.parse_args()
    selection = json.loads((args.study / "selection.json").read_text())
    samples = json.loads((args.study / "common/accumulator-training.json").read_text())
    primitive = [json.loads(line) for line in (args.study / "common/accumulator-primitive.jsonl").read_text().splitlines()]
    output = {"audit": "independent activation, loss/gradient and legacy finite-difference checks", "trials": []}
    for trial in selection["candidates"]:
        snapshots = trial["module"]["training_dynamics"]
        result = {"trial": trial["trial"], "checkpoints": [], "max_logged_gradient_error": 0.0}
        for snapshot in snapshots:
            model = snapshot["model"]
            width = len(model["hidden"])
            loss, gradient = loss_gradient(model, samples)
            if abs(loss - snapshot["training_mse"]) > 1e-12:
                raise RuntimeError("logged loss mismatch")
            counts = [0] * width
            active = [[0] * width for _ in range(2)]
            minimum, maximum = [math.inf] * width, [-math.inf] * width
            for row in samples:
                z, _, _ = scores(model, features(row))
                for k, value in enumerate(z):
                    minimum[k] = min(minimum[k], value)
                    maximum[k] = max(maximum[k], value)
                    if value > 0:
                        counts[k] += 1
                        active[row["operation_index"]][k] += 1
            for k, unit in enumerate(snapshot["units"]):
                error = abs(norm(gradient[k]) - unit["current_hidden_gradient_l2"])
                result["max_logged_gradient_error"] = max(result["max_logged_gradient_error"], error)
                if error > 1e-10 or counts[k] != unit["active_samples"] or abs(minimum[k] - unit["min_preactivation"]) > 1e-12 or abs(maximum[k] - unit["max_preactivation"]) > 1e-12:
                    raise RuntimeError("unit logging mismatch")
                if any(active[op][k] != unit["active_by_operation"].get(str(op), 0) for op in range(2)):
                    raise RuntimeError("operation activation mismatch")
            out_norm = norm([g for row in gradient[width:] for g in row])
            if abs(out_norm - snapshot["current_output_gradient_l2"]) > 1e-10:
                raise RuntimeError("output gradient logging mismatch")
            accuracy = {}
            for row in primitive:
                _, _, prediction = scores(model, features(row))
                rounded = [int(math.copysign(math.floor(abs(v * 32) + .5), v)) for v in prediction[:2]]
                wanted = row["prediction"]
                correct = rounded == [wanted["next"], wanted["result"]] and (prediction[2] >= .5) == wanted["write"]
                label = "read"
                if row["operation_index"] == 0:
                    total = row["state"] + row["argument"]
                    label = "lower" if total < -16 else "upper" if total > 16 else "threshold" if abs(total) == 16 else "interior"
                group = accuracy.setdefault(label, {"correct": 0, "cases": 0})
                group["cases"] += 1
                group["correct"] += int(correct)
            result["checkpoints"].append({"epoch": snapshot["completed_epochs"], "mse": loss,
                "active_samples": counts, "active_add_samples": active[0],
                "dead_on_training": sum(n == 0 for n in counts),
                "always_active_on_training": sum(n == len(samples) for n in counts),
                "mixed_add_units": sum(0 < n < 85 for n in active[0]),
                "accuracy_by_region": accuracy})
        # Verify actual derivatives numerically at legacy initialization, away
        # from ReLU kinks. These checks are offline and never update weights.
        if trial["initialization"] == "legacy":
            model = snapshots[0]["model"]
            _, analytic = loss_gradient(model, samples)
            if min(abs(z) for row in samples for z in scores(model, features(row))[0]) < 1e-5:
                raise RuntimeError("finite-difference check too near a kink")
            maximum_error, checked = 0.0, 0
            for i, row in enumerate(model["hidden"] + model["weights"]):
                for j, value in enumerate(row):
                    row[j] = value + 1e-6
                    plus, _ = loss_gradient(model, samples)
                    row[j] = value - 1e-6
                    minus, _ = loss_gradient(model, samples)
                    row[j] = value
                    maximum_error = max(maximum_error, abs((plus - minus) / 2e-6 - analytic[i][j]))
                    checked += 1
            if maximum_error > 1e-7:
                raise RuntimeError("finite-difference gradient check failed")
            result["finite_difference"] = {"parameters_checked": checked, "max_absolute_error": maximum_error, "epsilon": 1e-6}
        output["trials"].append(result)
    (args.study / "dynamics-analysis.json").write_text(json.dumps(output, indent=2, sort_keys=True) + "\n")
    print("All checkpoint activations, losses, gradients and legacy finite differences verified.")


if __name__ == "__main__":
    main()
