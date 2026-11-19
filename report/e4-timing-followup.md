# E4 timing follow-up

TODO(team): Preserve the exploratory timing evidence and its separation from the frozen study.

Executed November 15, 2026. This is a small exploratory check, separate from the frozen 81-run study. It tests whether later second failures allow the selected replacement to serve; it does not compare policies or estimate an availability probability.

## Setup

- F50, seed 1101, high demand: the same six-viewer trace, nine 128 KiB segments, three Peers, two initial copies, source limits, three-second grace, and client settings as the original E4 context.
- Peer 1 stops at workload epoch +5 seconds. Peer 3, the other original holder of sentinel `seed-1101--000008`, stops 30, 40, or 50 seconds later. Peer 2 is the replacement target.
- A fresh isolated cluster is used for each run; treatments run serially in shuffled order (30, 50, 40 seconds). Observation lasts 65 seconds from workload epoch, allowing the latest stop and its bounded five-second probe to finish.
- All three executable hashes match the original study. The reused experiment harness differs from its archived source only in module docstrings. No production or policy code was changed.
- Success retains the original definition: a direct GET from the target after the second stop must return the correct size/hash, and target registration must be committed. Receipt and registration are also checked before the second stop.

## Results

| Second stop after first stop | Actual delay | Target receipt before stop | Target committed before stop | Direct GET after stop | Correct hash and committed |
|---|---:|---|---|---|---|
| +30 seconds | 30.022 s | 404, missing | No | 404 | **Fail** |
| +40 seconds | 40.021 s | 200, present | Yes | 200 | **Pass** |
| +50 seconds | 50.047 s | 200, present | Yes | 200 | **Pass** |

In the +40 and +50 runs, the target first appeared in verified-copy samples 35.73 and 36.03 seconds after the first stop, respectively. These are sampled observations, not exact completion times. All intended failure preconditions/effects were confirmed, and all six viewers completed in each run. Viewer completion is separate from the sentinel probe result.

## Interpretation

Allowing more time changed the outcome with the same policy and binaries. At +30 seconds the replacement segment was still absent; at +40 and +50 seconds it was installed and committed, and remained directly deliverable after both original holders stopped. This supports unfinished repair as the explanation for the original failure in this context; the +30-second observation alone was not evidence of a correctness bug.

The follow-up demonstrates both an unavailable and an available outcome. It does not pinpoint an exact transition time, establish that other contexts recover by +40 seconds, prove complete-video availability, or establish a policy advantage. Each delay has one run in one context. The original E4 results remain 0/18 at their original delays and are unchanged.

## Evidence and reproduction

- [Timing summary](../results/e4-timing-followup-2026-11-15-rerun/timing-summary.csv)
- [Structured results](../results/e4-timing-followup-2026-11-15-rerun/timing-summary.json)
- [Configuration](../results/e4-timing-followup-2026-11-15-rerun/study-config.json)
- [Original binary comparison](../results/e4-timing-followup-2026-11-15-rerun/original-binary-comparison.json)
- Raw pre-stop receipts, injections, copy observations, final correctness checks, and process logs are saved in each run directory alongside the summary.

Run from the repository root, using a new output directory:

```sh
python3 experiments/e4_timing_check.py --output-dir results/e4-timing-followup-new
```

The script requires the original study's peer/controller/client binaries and permission to use local network ports. An initial restricted invocation could not bind local ports and produced no measured run; the completed results above come from the subsequent permitted invocation.
