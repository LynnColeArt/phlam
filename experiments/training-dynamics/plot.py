#!/usr/bin/env python3
"""Optional static figure; requires matplotlib. No effect on training or timing."""
import argparse
import json
from pathlib import Path
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt

parser = argparse.ArgumentParser(__doc__)
parser.add_argument("study", type=Path)
args = parser.parse_args()
trials = json.loads((args.study / "selection.json").read_text())["candidates"]
figure, axes = plt.subplots(1, 2, figsize=(11, 4.6), sharey=True)
colors = {"legacy": "#924a85", "paired-centered": "#087e8b", "paired-affine": "#bd5b16"}
labels = {"legacy": "Legacy", "paired-centered": "Centered opposing pairs", "paired-affine": "Pairs + affine prefit (12k extra)"}
for axis, width in zip(axes, [2, 4]):
    for trial in trials:
        if trial["width"] != width:
            continue
        snapshots = trial["module"]["training_dynamics"]
        arm = trial["initialization"]
        axis.loglog([s["completed_epochs"] + 1 for s in snapshots], [s["training_mse"] for s in snapshots], marker="o", markersize=3, color=colors[arm], label=labels[arm])
    if width == 4:
        snapshots = next(t for t in trials if t["width"] == 8)["module"]["training_dynamics"]
        axis.loglog([s["completed_epochs"] + 1 for s in snapshots], [s["training_mse"] for s in snapshots], "--", color="#666666", label="8-unit legacy reference")
    axis.set_title(f"{width} hidden units")
    axis.set_xlabel("Completed joint epochs + 1 (log scale)")
    axis.grid(alpha=.2, which="major")
    axis.legend(fontsize=8, loc="lower left")
axes[0].set_ylabel("Training MSE over 3 normalized output heads")
figure.suptitle("Fixed seed 404: initialization changes the four-unit training outcome", fontsize=13)
figure.tight_layout(rect=(0, .035, 1, .94))
figure.text(.5, .012, "11 saved checkpoints; connecting lines are guides. Affine prefit uses only the same 102 training examples.", ha="center", fontsize=8)
figure.savefig(args.study / "training-loss.png", dpi=170)
figure.savefig(args.study / "training-loss.svg")
