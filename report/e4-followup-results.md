# E4 follow-up: second-holder failure and repair readiness

TODO(Yuquan): Keep this report aligned with the archived E4 follow-up evidence.

Executed November 15, 2026. All 48 trials completed with valid injections: 30 exploratory sweep trials, 12 independent fixed-delay validation trials, and six completion-gated positive controls. The original compact-study archive is unchanged.

## Design and timing

- Three real Controllers and three real Peers; the original synthetic nine-segment fixture, high demand, rates, grace, placement, and client settings were retained. Transfers use real HTTP; media consumption is simulated.
- First Peer failure is scheduled at experiment time 5 s. Delta is the interval between the two failures, not time from experiment startup. Delta 34 s schedules the second failure at experiment time 39 s. Actual injection times are archived.
- Sweep: Delta = 33, 34, 35, 36, 40 s; S/F50/L; seeds 1101/1102; one trial per cell. Order shuffled with seed 20261115. Trials run serially on one shared host.
- The fixed-delay observation horizon is 55 s, allowing time after the latest second failure. The original archive used 50 s. The success check still occurs immediately after the second stop, independently of repair success.
- Target success requires a fresh direct response for the preselected late-queue sentinel with correct size and SHA-256 AND quorum-backed committed target registration.
- Delta 34 s was selected during exploration. The selection and independent validation plan were recorded before validation. Validation repeats each context-policy cell twice, giving four new trials per policy. Exploration and validation are not pooled.

## Exploratory sweep

| Delay after first failure | S | F50 | L |
|---|---:|---:|---:|
| 33 s | 1/2 | 1/2 | 1/2 |
| 34 s | 2/2 | 2/2 | 0/2 |
| 35 s | 2/2 | 2/2 | 2/2 |
| 36 s | 2/2 | 2/2 | 2/2 |
| 40 s | 2/2 | 2/2 | 2/2 |

Counts are successful target deliveries / all scheduled trials. At 33 s, all policies failed in context 1101 and succeeded in context 1102. At 34 s, S and F50 succeeded in both contexts, while L was not ready in either. At 35, 36, and 40 s all tested cells succeeded.

## Independent validation at Delta 34 s

| Policy | Successful target deliveries |
|---|---:|
| S | 2/4 |
| F50 | 3/4 |
| L | 4/4 |

| Context and repeat | S | F50 | L |
|---|---|---|---|
| 1101, repeat 1 | unavailable | ready | ready |
| 1101, repeat 2 | unavailable | unavailable | ready |
| 1102, repeat 1 | ready | ready | ready |
| 1102, repeat 2 | ready | ready | ready |

The apparent policy ordering reversed between exploration and validation. Delta 34 s is useful for demonstrating timing-sensitive mixed availability, but these results do NOT establish a stable winner or a deterministic recovery threshold. Each context is reused across policies; the trials do not establish population confidence bounds.

## Positive control after verified repair completion

The control waits until EVERY segment affected by the first failure has two reachable, size/hash-valid, committed copies. It then stops the second original holder and probes the sentinel. Finally it fetches ALL nine segments afresh from the sole surviving Peer and verifies size, SHA-256, and committed registration for each. This bypasses any pre-existing viewer buffer. Controls have a 75 s horizon; their fault time is completion-gated, not fixed.

| Policy | Sentinel delivery | Fresh delivery of all 9 segments |
|---|---:|---:|
| S | 2/2 | 2/2 |
| F50 | 2/2 | 2/2 |
| L | 2/2 | 2/2 |

Actual control fault intervals ranged from 31.618 to 34.091 s after the first stop. All 54 fresh per-segment checks passed. This is full-fixture byte delivery, not browser decoding, MSE continuity, or a full new playback-session measurement.

## Answer to the team discussion

1. The original 4/14/30 s all-zero E4 table did not cover the readiness transition; it is not evidence that repaired replicas cannot serve.
2. Use Delta 34 s (experiment time 39 s) to show mixed availability. Present all independent validation counts and disclose that exploratory policy ordering did not repeat.
3. Keep a late fixed-delay point (40 s, 2/2 for every policy) and the completion-gated control (2/2 for every policy). Together they distinguish unfinished repair from a broken replacement-delivery path.
4. Do not claim a universal minimum repair time of 36 s, permanent data loss, stable policy superiority, or browser playback continuity. Stopped disks were retained during fault injection.

## Reproduce

Run in the project Linux environment after building current backend binaries. Use fresh output directories (or --resume only with matching configuration and binaries).

```sh
make backend
python3 experiments/e4_followup.py --output-dir results/e4-followup/scan
python3 experiments/e4_followup.py --output-dir results/e4-followup/validation --delays 34 --repeats 2
python3 experiments/e4_followup.py --output-dir results/e4-followup/completed-control --control
python3 experiments/analyze_e4_followup.py results/e4-followup
```

## Evidence

- `results/e4-followup/all-outcomes.csv`: every trial, actual fault interval, pre-fault target readiness, post-fault correctness, and full-fixture control checks.
- `results/e4-followup/aggregate.csv`: separate phase summaries; `audit.json`: raw-summary consistency and injection checks.
- Each phase includes its configuration, matrix, source snapshot, binary hashes, and environment provenance. All three phases used identical peer/controller/client binary hashes.
- Each trial retains injections, pre-fault registration and receipt probes, target delivery evidence, task and copy-verification histories, client events, and process logs.
- `validation-plan.json` records selection of 34 s before independent validation; `SHA256SUMS` covers the completed follow-up archive.
