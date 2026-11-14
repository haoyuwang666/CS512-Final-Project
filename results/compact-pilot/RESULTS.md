# Compact repair study results

Archived E1–E4 runs: **6/6**. Controller runs: **0**.

Exploratory study: 3 paired workload seeds, 3 Controllers, 3 Peers, 50 s fixed horizon. Real HTTP transfer with headless simulated media consumption. No browser performance claim.

## Slide-ready evidence

| Experiment | Group | Runs | Complete viewers | Mean post-fault stall (s) | Recovery observed | Target met | Repair MiB |
|---|---|---:|---:|---:|---:|---:|---:|
| pilot | F25, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.11 | 1/1 | 1/1 | 0.75 |
| pilot | F50, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.39 | 1/1 | 1/1 | 0.75 |
| pilot | L, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.16 | 1/1 | 1/1 | 0.75 |
| pilot | N, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.76 | 1/1 | 1/1 | 0.00 |
| pilot | OFF, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.00 | 0/1 | 0/1 | 0.00 |
| pilot | S, high, grace 3, persistent, Δ2=0 | 1 | 4/4 | 0.05 | 1/1 | 1/1 | 0.75 |

## Interpretation boundaries

Small samples characterize these workloads; individual paired differences and ranges are in `paired-contrasts.csv`. They do not establish population noninferiority or precise tail latency. Unrecovered runs remain censored; no zero or fabricated recovery time is substituted.

Copy exposure is sampled from hash-valid inventories intersected with quorum-backed registrations. Verification failure is unknown/unreachable, not proof of destroyed data. The first post-fault sample is interval censored at the verification cadence. Repair bytes are ResponseWriter payload, including retries; probe traffic has a separate class. Counter instances are tracked across restart.

See `summary.csv` for every run, `aggregate.csv` for slide tables, `controller-summary.csv` for controller checks, and `figures/` for SVG and PNG exports. Raw event journals, injections, manifests, task/copy observations and logs are under each run directory.

Time budget reduction: grace 1/3/8 s, brief outage 2 s, first fault 5 s, second fault delays 4/14/30 s, recovery objective 40 s. L retains the original 250 ms / 2 s idle / 60 s aging rule; compact runs do not test the age branch statistically. Regression tests exercise that branch.

The final scientific verdict must consider completion, recovery censoring, traffic and exposure together; a completed-only stall statistic cannot establish improvement.
