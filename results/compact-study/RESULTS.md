# Compact repair study results

Archived E1–E4 runs: **72/72**. Controller runs: **9**.

Exploratory study: 3 paired workload seeds, 3 Controllers, 3 Peers, 50 s fixed horizon. Real HTTP transfer with headless simulated media consumption. No browser performance claim.

## Slide-ready evidence

| Experiment | Group | Runs | Complete viewers | Mean post-fault stall (s) | Recovery observed | Target met | Repair MiB |
|---|---|---:|---:|---:|---:|---:|---:|
| E1 | N, high, grace 3, persistent, Δ2=0 | 3 | 18/18 | 3.20 | 0/3 | 0/3 | 0.00 |
| E1 | OFF, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 3.24 | 0/3 | 0/3 | 0.00 |
| E1 | S, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.46 | 3/3 | 3/3 | 0.75 |
| E1 | N, low, grace 3, persistent, Δ2=0 | 3 | 6/6 | 0.00 | 0/3 | 0/3 | 0.00 |
| E1 | OFF, low, grace 3, persistent, Δ2=0 | 3 | 6/6 | 0.00 | 0/3 | 0/3 | 0.00 |
| E1 | S, low, grace 3, persistent, Δ2=0 | 3 | 6/6 | 0.00 | 3/3 | 3/3 | 0.75 |
| E2 | S, high, grace 1, brief, Δ2=0 | 3 | 18/18 | 3.30 | 3/3 | 3/3 | 0.12 |
| E2 | S, high, grace 1, persistent, Δ2=0 | 3 | 7/18 | 4.45 | 3/3 | 3/3 | 0.75 |
| E2 | S, high, grace 3, brief, Δ2=0 | 3 | 18/18 | 3.09 | 3/3 | 3/3 | 0.00 |
| E2 | S, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.46 | 3/3 | 3/3 | 0.75 |
| E2 | S, high, grace 8, brief, Δ2=0 | 3 | 18/18 | 3.18 | 3/3 | 3/3 | 0.00 |
| E2 | S, high, grace 8, persistent, Δ2=0 | 3 | 7/18 | 4.23 | 3/3 | 2/3 | 0.75 |
| E3 | F25, burst, grace 3, persistent, Δ2=0 | 3 | 9/18 | 6.28 | 2/3 | 1/3 | 0.74 |
| E3 | F50, burst, grace 3, persistent, Δ2=0 | 3 | 9/18 | 6.20 | 3/3 | 2/3 | 0.75 |
| E3 | L, burst, grace 3, persistent, Δ2=0 | 3 | 9/18 | 6.21 | 3/3 | 1/3 | 0.75 |
| E3 | S, burst, grace 3, persistent, Δ2=0 | 3 | 9/18 | 6.18 | 3/3 | 2/3 | 0.75 |
| E3 | F25, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.73 | 3/3 | 3/3 | 0.75 |
| E3 | F50, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.88 | 3/3 | 3/3 | 0.75 |
| E3 | L, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.33 | 3/3 | 3/3 | 0.75 |
| E3 | S, high, grace 3, persistent, Δ2=0 | 3 | 7/18 | 4.46 | 3/3 | 3/3 | 0.75 |
| E4 | F50, high, grace 3, persistent, Δ2=4 | 2 | 0/12 | 2.20 | 0/2 | 0/2 | 0.00 |
| E4 | F50, high, grace 3, persistent, Δ2=14 | 2 | 0/12 | 6.97 | 0/2 | 0/2 | 0.12 |
| E4 | F50, high, grace 3, persistent, Δ2=30 | 2 | 7/12 | 4.28 | 0/2 | 0/2 | 0.62 |
| E4 | L, high, grace 3, persistent, Δ2=4 | 2 | 0/12 | 2.19 | 0/2 | 0/2 | 0.00 |
| E4 | L, high, grace 3, persistent, Δ2=14 | 2 | 0/12 | 6.83 | 0/2 | 0/2 | 0.12 |
| E4 | L, high, grace 3, persistent, Δ2=30 | 2 | 7/12 | 4.73 | 0/2 | 0/2 | 0.62 |
| E4 | S, high, grace 3, persistent, Δ2=4 | 2 | 0/12 | 2.25 | 0/2 | 0/2 | 0.00 |
| E4 | S, high, grace 3, persistent, Δ2=14 | 2 | 0/12 | 6.97 | 0/2 | 0/2 | 0.12 |
| E4 | S, high, grace 3, persistent, Δ2=30 | 2 | 7/12 | 4.44 | 0/2 | 0/2 | 0.62 |

## Interpretation boundaries

Small samples characterize these workloads; individual paired differences and ranges are in `paired-contrasts.csv`. They do not establish population noninferiority or precise tail latency. Unrecovered runs remain censored; no zero or fabricated recovery time is substituted.

Copy exposure is sampled from hash-valid inventories intersected with quorum-backed registrations. Verification failure is unknown/unreachable, not proof of destroyed data. The first post-fault sample is interval censored at the verification cadence. Repair bytes are ResponseWriter payload, including retries; probe traffic has a separate class. Counter instances are tracked across restart.

See `summary.csv` for every run, `aggregate.csv` for slide tables, `controller-summary.csv` for controller checks, and `figures/` for SVG and PNG exports. Raw event journals, injections, manifests, task/copy observations and logs are under each run directory.

Time budget reduction: grace 1/3/8 s, brief outage 4 s, first fault 5 s, second fault delays 4/14/30 s, recovery objective 40 s. L retains the original 250 ms / 2 s idle / 60 s aging rule; compact runs do not test the age branch statistically. Regression tests exercise that branch.

The final scientific verdict must consider completion, recovery censoring, traffic and exposure together; a completed-only stall statistic cannot establish improvement.
