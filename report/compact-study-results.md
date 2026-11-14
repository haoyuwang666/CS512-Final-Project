# Compact E1–E5 results

TODO(team): Keep reported findings and limitations traceable to the archived compact study.

Executed November 14, 2026. **All 81 frozen study runs completed in 67.9 minutes:** 72 unique E1–E4 runs and nine E5 runs. Six policy pilots and three controller pilots are archived separately and excluded from these estimates. The revised one-to-two-hour execution budget was met.

The implemented treatments are N (no failure), OFF (failure without repair), S (shared upload service), F25, F50, and L (load-aware repair). All retain the same total source upload limiter. Backend build/static/race, measurement, smoke, repair, cleanup, deletion, exact fragment-timing, and real-media decoding gates passed. No frontend changes were needed.

This is an exploratory real-process study with three paired workload/fault contexts; E4 has two paired contexts per cell. It transfers real bytes over HTTP under configured upload limits and simulates media consumption. It does not measure browser playback or establish population confidence bounds.

## Setup and evidence

Three Controllers, three Peers, fixed placement, round-robin selection, exactly two initial copies, and no optional copies. The content has nine distinct 128 KiB segments of two logical seconds each. Source limits are 96/144/192 KiB/s. Low demand has two viewers; high demand has six; burst arrivals add viewers at 4 and 12 seconds. Client window/concurrency/startup buffer are 3/2/4 seconds. The existing eight-second download-attempt timeout remains unchanged; the session timeout is 42 seconds.

The first holder failure occurs at five seconds. Observation ends at 50 seconds. Brief absence lasts four seconds; persistent absence is not restored. Grace values are 1/3/8 seconds. The recovery objective is **verified** full affected-set redundancy within 40 seconds after the first fault. Source failure rotates across all three capacities. Seeds 1101/1102/1103 correspond to failed Peers 1/2/3. The same finite viewer trace, content hashes, initial placement, and failure schedule are reused within each paired context. Treatment order is independently randomized.

Data: [full run tables](../results/compact-study/summary.csv), [aggregate tables](../results/compact-study/aggregate.csv), [paired differences](../results/compact-study/paired-contrasts.csv), [396 viewer outcomes](../results/compact-study/sessions.csv), [measurement audit](../results/compact-study/analysis-audit.json), and [configuration/provenance](../results/compact-study/environment-and-provenance.json). Raw journals, manifests, injections, task/copy observations and process logs remain under each run directory. [The compact design](compact-experiment-design.md) explains the reduced matrix.

## RQ-A: when does repair interfere with delivery? (E1)

| Demand | Treatment | Completed viewers | Mean observed post-fault stall per scheduled viewer |
|---|---|---:|---:|
| Low | N | 6/6 | 0.00 s |
| Low | OFF | 6/6 | 0.00 s |
| Low | S | 6/6 | 0.00 s |
| High | N | 18/18 | 3.20 s |
| High | OFF | 7/18 | 3.24 s |
| High | S | 7/18 | 4.46 s |

At low demand, repair overlapped foreground requests but added no observed stalls. At high demand, the mean paired S−OFF difference was **+1.22 seconds/viewer**, with individual differences **−0.13, +0.33, and +3.46 seconds**. Three source/fault contexts are too few to generalize that mean.

The most interpretable feasible high-load context is loss of the slowest Peer: all six viewers finished under N, OFF and S. Observed post-fault stalls were **3.19 / 4.01 / 7.47 seconds**, respectively. Shared repair added **3.46 seconds/viewer** relative to the matched failure without repair. Same-source repair/foreground overlap was observed in all six E1/S runs.

Loss of the middle or fastest Peer produced 1/6 or 0/6 completion under both OFF and S. Thus, Peer loss and the existing client retry/timeout budget explain a major part of service failure independently of repair. A lower stall average in an early-failed session is not a service improvement. The N-versus-failure completion gap must stay beside the stall plot.

**Verdict:** H1 is supported in a scoped sense: the tested spare-capacity workload showed no interference, while a feasible high-load context showed additional stalls during shared repair. The effect is not uniform across failed-source capacities.

## RQ-A: when should repair start? (E2)

| Outage | Grace | Mean repair payload/run | Mean sampled one-copy exposure |
|---|---:|---:|---:|
| Brief, 4 s | 1 s | 0.125 MiB | 4.08 MiB·s |
| Brief, 4 s | 3 s | 0 MiB | 4.08 MiB·s |
| Brief, 4 s | 8 s | 0 MiB | 4.13 MiB·s |
| Persistent | 1 s | 0.750 MiB | 17.42 MiB·s |
| Persistent | 3 s | 0.750 MiB | 18.24 MiB·s |
| Persistent | 8 s | 0.750 MiB | 20.20 MiB·s |

Each one-second-grace brief-outage run copied one 128 KiB segment. Both longer grace settings avoided that work in all three contexts. All 54 brief-outage viewer sessions completed. Their observed full restoration was around 5.3–5.5 seconds after the fault, through original-Peer return; this includes observation lag after the four-second absence.

Persistent failures required the same 0.75 MiB repair payload at all grace settings. Mean sampled exposure rose from 17.42 to 20.20 MiB·s between one- and eight-second grace, about 16%. Full restoration was not strictly monotonic in grace for every context: source contention, viewer termination, two-second coordinator ticks and serial task execution also affect it. Actual first-task and first-repair-byte observations are retained in [measurement-audit.csv](../results/compact-study/measurement-audit.csv).

**Verdict:** H2 is supported for these traces: grace avoids some short-outage work, while persistent absence retains a recovery/exposure cost. Grace is a trigger control, not an assurance that a started task will cancel when the original returns.

## RQ-B: does L improve the playback/recovery compromise? (E3)

| Demand | Policy | Completed viewers | Mean observed stall | Verified by 40 s | Median observed restoration | Unrecovered at horizon |
|---|---|---:|---:|---:|---:|---:|
| High | S | 7/18 | 4.46 s | 3/3 | 34.03 s | 0 |
| High | F25 | 7/18 | 4.73 s | 3/3 | 39.91 s | 0 |
| High | F50 | 7/18 | 4.88 s | 3/3 | 33.93 s | 0 |
| High | L | 7/18 | 4.33 s | 3/3 | 34.13 s | 0 |
| Burst | S | 9/18 | 6.18 s | 2/3 | 37.98 s | 0 |
| Burst | F25 | 9/18 | 6.28 s | 1/3 | 37.08 s* | 1 |
| Burst | F50 | 9/18 | 6.20 s | 2/3 | 37.83 s | 0 |
| Burst | L | 9/18 | 6.21 s | 1/3 | 40.05 s | 0 |

*F25's burst median excludes one censored run and must not be read as evidence that it recovered faster overall.*

All four policies had identical viewer completion counts in each matched context. L−F50 mean observed stall was **−0.55 seconds/viewer under steady demand** and **+0.01 seconds under burst demand**. The steady mean is driven by one context; its median paired difference is essentially zero. These aggregate changes do not demonstrate the predeclared one-second meaningful improvement across the tested conditions.

There is a concrete tradeoff in the feasible slowest-Peer/high-load context: L reduced stalls from **8.53 to 6.88 seconds/viewer** versus F50, while verified restoration rose from **33.99 to 36.06 seconds** and one-copy exposure rose from **18.82 to 20.38 MiB·s**. This is a measured compromise, not dominance.

F25 had later steady restoration in all three paired contexts, with no completion gain, and one burst run remained unrecovered by the horizon. L did not consistently beat F50: in burst demand, their stalls and completion were nearly identical, and L was first verified after the target in one extra context. One of those observations was 40.05 seconds, so the two-second verification cadence precludes a precise claim about a 50-millisecond deadline difference.

Recorded policy samples matched the implemented rule with **zero observed rule violations**. L's 250 ms activity sampling and two-second idle relaxation operated; its 60-second age floor did not activate in the compact performance horizon. Boundary/age and shared-cap behavior have regression coverage, and committed task ages survived the E5 failovers.

**Verdict:** no consistent overall L advantage over the credible fixed F50 baseline is established. The data supports one workload-specific playback/recovery compromise and a negative/inconclusive broader H3 result. It does not establish population noninferiority or prove that adaptation cannot help elsewhere.

## RQ-B/RQ-C: consequence of a second holder failure (E4)

| Policy | Second fault +4 s | +14 s | +30 s |
|---|---:|---:|---:|
| S | 0/2 | 0/2 | 0/2 |
| F50 | 0/2 | 0/2 | 0/2 |
| L | 0/2 | 0/2 | 0/2 |

Cells count hash-valid **and committed** direct deliveries of a prespecified late-queue sentinel from its replacement target after both original holders stop. All 18 attempts were unavailable at the chosen times. Some continuing viewers still completed because they had already buffered the content.

**Verdict:** these cells expose the availability boundary during reduced redundancy. They do not discriminate among the policies' second-failure tolerance because this particular sentinel was not ready at any tested delay. A later-delay transition was not measured. Stopped processes retained their disks: this is not a claim of permanent data loss or an estimate of real-world loss probability.

## RQ-C: application recovery and safety (E5)

| Case | Passed | Fresh application recovery, median [range] |
|---|---:|---|
| Leader loss during delivery | 3/3 | 1.52 s [1.47, 2.74] after Leader stop |
| Leader loss during pending repair | 3/3 | 1.60 s [1.52, 1.97] after Leader stop |
| Majority loss / restoration | 3/3 | 1.04 s [0.85, 1.43] after quorum restoration |

Warm and cold uncached sessions completed in all E5a runs. All E5b runs retained the same committed pending task identity and creation time, installed hash-valid target bytes, and committed the replacement registration. Controller restart retained the task; source-Peer restart retained L configuration and produced a new counter instance ID.

During stable majority loss, **all nine valid write attempts returned HTTP 503**, with no success acknowledgments. Restoring quorum retained the baseline state and allowed reconciliation using the original operation ID. Quorum-isolation duration is recorded separately; the E5c timing above excludes that deliberate isolation. The general controller distribution plot also retains fault-relative timing.

**Verdict:** the application contract held in these nine tested process-crash/quorum scenarios. A Raft role or health response was not used as the availability endpoint; recovery requires a fresh quorum-backed application response. This is not sustained-network-partition, Byzantine, power-loss, or universal reliability coverage.

## Audit, uncertainty and reuse

All 72 performance lifecycles completed; this status does not mean all viewers succeeded. All **396 scheduled viewer outcomes** are saved, with no missing or late outcomes. All intended injections and post-fault exposure checks succeeded. Original raw journals are unchanged; original observed run-summary rows are preserved alongside the enriched summaries. Source/binary fingerprints and the executed source snapshot are archived. Final analysis source is archived separately in `analysis-source/`.

The initial static/race and all nine final correctness gates passed. Final figure headers/layouts were rendered and visually checked. PNG and SVG exports are available under [figures](../results/compact-study/figures/); the [slide guide](../results/compact-study/SLIDE-EXHIBITS.md) provides captions and selection advice.

Important limits: three paired contexts, one context per failed capacity; two E4 contexts; one host; configured application bandwidth rather than physical NIC conditions; fresh stores/processes but no OS page-cache flush; and the existing client retry/timeout semantics. Exact restoration is bracketed by sampled verification, and exposure is a step estimate. Counters measure ResponseWriter payload, including retries, rather than TCP/NIC overhead. A counter snapshot is taken just before each crash; a small amount of payload written between that snapshot and process exit may be unobserved. No population confidence intervals, precise p99, browser continuity, or hard service guarantees are claimed.

To regenerate all final exhibits without rerunning the clusters:

```sh
python3 experiments/finalize_compact_study.py results/compact-study
```
