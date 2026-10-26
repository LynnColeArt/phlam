# G02 run 0001: fresh-seed reliability on the initial scalar corpus

Recorded 2026-10-26. **Initial scalar milestone passed; full G02 remains in progress.**

[Locked campaign](../campaign.json) · [Reproduction](../README.md) · [All fit results](results.json) · [Integration](integration.json) · [Independent evidence audit](validation.json).

## Decision

A uniform 8-unit paired-affine recipe meets the planned ≥4/5 threshold on each initial program, with **5/5 fresh seeds each**. The same signed model bytes reproduce in both clean builds of all 45 configurations, including failed models. The 90 builds are 45 fits repeated for reproducibility, not 90 independent seed trials. No fallback, curated seed replacement or retry contributes to the reliability counts.

| Recipe | Counter | Rejecting accumulator | Saturating accumulator | Meets all three targets |
|---|---:|---:|---:|---|
| 4-unit paired-centered control | 5/5 | 5/5 | 0/5 | no |
| 8-unit paired-centered | 5/5 | 5/5 | 2/5 | no |
| 8-unit paired-affine | 5/5 | 5/5 | 5/5 | yes |

Seeds are **11, 23, 37, 53, 71**, declared before the run. Historical seeds 7/404 receive no fresh-seed credit. Width-8 paired-centered saturation succeeds only at 11 and 37. All eight failing configurations retain both signed candidates, raw predictions, losses, source observations, certificate counterexamples, resources and safe runtime comparisons. They publish no deployment directory. Failed certification terminates at its first counterexample; it is not recorded as an exhaustive success.

Selection uses the first declared recipe meeting the threshold separately on every program, with no per-program mixing or timing-based selection. The control plus two interventions were fixed before fresh fits. Width adds capacity; affine pretraining addresses the earlier diagnosed interaction between the linear base and nonlinear residual. No new training strategy or bounded follow-up was needed.

## What was trained and frozen

The existing G01 source-to-artifact pipeline builds each source separately with exactly one candidate. Sources, domains, sampling and budgets are hashed in campaign.json. Counter supplies 68 training rows and 262 other admitted rows; rejecting Accumulator supplies 90 and 432, with another 72 contract-rejected inputs; saturation supplies 102 and 492. Actual source-oracle observations are retained and checked against independent fixture mathematics outside the builder.

All recipes use the existing normalized residual model and 24,000 joint Adam epochs. Paired-affine additionally spends **12,000 affine pretraining epochs** on the same training rows, then starts paired residual units with zero output weights before joint optimization. Thus it has a larger optimization budget; this is a comparison of declared recipes, not an isolated causal estimate of initialization alone. Its 8-unit scalar model has 79 parameters. No application thresholds, formulas, manually fitted weights or acceptance counterexamples are injected into training. No training/runtime Go code changed.

The selected recipe is frozen for this initial scalar milestone. Five successes on known scalar source do not establish a population-wide convergence probability, unfamiliar-program reliability, or optimal model size. The broad plan's mandatory G03 families and G05 sealed programs remain outstanding. Their corpus commitment and recipe freeze must precede acceptance work. Full G02 is deliberately not marked passed.

## Recertification and source-free execution

Every build freezes and signs its model, reloads it through the standalone runtime, and attempts the unchanged exact certificate over the full declared primitive domain: 330 Counter inputs and 594 for each accumulator. A successful rejecting accumulator certificate establishes 522 admitted transitions and 72 rejected inputs. Each selected fit has every admitted prediction correct, including all nontraining inputs. Certificate reports are evidence, not trusted executable permission: each certified runtime startup reconstructs them from the frozen artifacts.

Every fit's first clean build is also tested source-free against lookup under both concrete and certified boundaries. Only runtime, signed model/bundle files and an external public key are copied into the execution directory. The byte-identical second build shares the same semantics. Failed learned models are refused wholesale by certified preflight; their concrete execution rejects incorrect predictions without publishing a wrong accepted output. Lookup remains fully checked, including for failed fits. Source independence is established by deployment inventory, dependency and linked-symbol checks; it is not a hostile-filesystem sandbox claim.

The first jointly certifying fresh seed, **11**, is rebuilt as a three-component bundle twice. All deployment files reproduce; each model is byte-identical to its independently built single-component candidate. This composition build does not add seed-success credit.

The combined bundle runs the existing **1,526** oracle/fixture scenarios, repeated three times in all four engine/boundary combinations. Each mode yields **4,341 accepts and 237 rejects**, matching states, versions, typed results and ordered writes. A separate audit reruns all 1,526 cases in eight engine/boundary/clock modes and checks clock equivalence. Cases include cross-component return passing, staged writes followed by a later failure, native rejection after a prior write, invalid arguments/references, unknown exports and state/domain escapes. Every rejection restores original values/versions with no published results or writes. Each isolated fit also exercises a write followed by an out-of-domain argument and checks atomic rollback.

## Exact checker work and timing remain separate

The following totals are for the combined bundle's 4,578 attempted transactions per mode, with phase clocks enabled. Entry checks are measured outside transaction phase totals. These short sequential runs include failed transactions and instrumentation overhead; they do not establish a speed advantage or regression.

| Engine / boundary | Proposal ms | Boundary ms | Commit ms | Total transaction ms | Entry checks ms |
|---|---:|---:|---:|---:|---:|
| Learned / concrete | 0.794 | 1.306 | 0.448 | 6.609 | 0.625 |
| Lookup / concrete | 0.116 | 1.523 | 0.505 | 7.404 | 0.688 |
| Learned / certified | 0.812 | 0.668 | 0.465 | 6.139 | 0.618 |
| Lookup / certified | 0.809 | 0.672 | 0.456 | 6.004 | 0.638 |

Both concrete modes perform 30,534 checker integer operations and 13,059 output comparisons. Both certified modes perform **zero warm relation arithmetic and zero warm output comparisons**; domain, ABI, entry and transaction checks remain. Certified lookup separately records 29,655 application integer operations; learned execution records zero lookup application work. This is the existing lookup baseline's measured work, not neural inference being credited as free.

Learned startup certification separately takes 0.741 ms summed across components in this integration invocation, with 10,149 exact integer operations and 4,338 output comparisons across 1,518 primitive inputs. Full load takes 1.019 ms. Cold certificate cost, warm checker cost, lookup application work, proposal time and entry checks remain separately visible in integration.json and per-trial results.

## Resource bounds and verification

Hardware: AMD Ryzen 5 5600X, Linux amd64, Go 1.26.4, GOMAXPROCS=1. Training child CPU across the 90 isolated builds ranges from 197 to 565 ms; maximum verification-child CPU is 3.762 ms. Maximum sampled simultaneous process-tree RSS is **20,082,688 bytes**. Cumulative isolated build wall times, including all three recipes and both repetitions, are Counter 10.369 s, rejecting accumulator 13.532 s and saturation 15.736 s. All observed runs are within declared CPU/wall/sample-memory budgets.

CPU comes from child process accounting; sampled RSS sums live processes, can miss brief peaks and is not a kernel-enforced 4 GiB peak bound. Sampling/watchdog applies to the 90 isolated builds; combined integration builds retain normal builder candidate CPU/wall measurements but have no additional RSS sampling. The two-hour-per-program limit concerns build work here, not a portable hardware-independent training guarantee.

Validation: **129 passing Go test events**, two existing deferred-semantics skips, no failures; go vet passes. Runtime dependencies exclude compiler/reference packages, and optimizer symbols are absent.

Next: commit the broader corpus specification before richer-family acceptance, then begin G03's typed multi-slot state and learned internal control. Keep this recipe as an explicit development baseline and extend the G02 reliability matrix as those required families become available.
