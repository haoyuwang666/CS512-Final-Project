# Experiment explanation: E0–E5

TODO(team): Keep experiment procedures, measurements and limitations aligned with archived evidence.

This document explains **what each experiment asks, how the implemented harness performs it, what is measured, and what the results permit us to conclude**. It describes the executed compact study, not the earlier larger proposal.

Companion: [section-based presentation layout](presentation-layout.md). Full results: [compact-study-results.md](../report/compact-study-results.md).

## 1. Study at a glance

**Execution:** 81 unique frozen study runs in 67.9 minutes: 72 performance/Peer-failure runs and nine Controller-recovery runs. There are 396 scheduled performance-viewer outcomes. Six policy pilots and three Controller pilots are separate and excluded from final estimates.

| Experiment | Question | What changes | Main evidence |
|---|---|---|---|
| E0 | Are the implementation and measurements usable? | Correctness/measurement test scenarios | Archived validation gates |
| E1 | Does repair add interference beyond losing a Peer? | N/OFF/S and low/high demand | Completion, observed stalls, repair/foreground overlap |
| E2 | What does waiting before repair change? | Grace 1/3/8 s and brief/persistent absence | Repair payload, restoration mechanism and exposure |
| E3 | Does L improve the compromise over fixed caps? | S/F25/F50/L and steady/burst demand | Completion, stalls, restoration, target verification, exposure |
| E4 | What happens if the other original holder fails too? | Policy and second-failure delay | Direct hash-valid, committed replacement delivery |
| E5 | Does metadata/task recovery preserve the contract? | Leader failure or majority loss | Fresh application recovery, task identity, byte validation and write refusal |

**Research questions:** RQ-A concerns interference and repair timing; RQ-B concerns the playback/recovery compromise; RQ-C concerns correctness and failure boundaries. E1/E2 answer RQ-A, E3/E4 inform RQ-B, and E0/E4/E5 support RQ-C.

## 2. What “multiple machines” means in this study

We emulated three Controllers and three Peers as **separate operating-system processes** on one host. They use distinct HTTP ports, separate storage/Raft directories, and real HTTP/TCP communication. Controllers also have their own Raft endpoints. Killing a process makes that service unavailable; restarting a Peer reopens its persisted store.

Ports distinguish endpoints. Process separation creates independent service lifetimes and state directories. Neither alone creates independent physical failure domains. The processes still share CPU, RAM, disks and the host network stack. Upload limits are application-level controls, not measurements of separate physical NICs. The study did not inject WAN latency, packet loss or sustained network partitions.

Suitable presentation wording:

> A three-Controller, three-Peer deployment emulated on one host, with separate processes, storage, bandwidth limits and injected process failures.

Source: [tests/system/cluster.py](../tests/system/cluster.py).

## 3. Content and consumption model

The performance fixture has nine segments, each 128 KiB with a logical duration of two seconds. It therefore represents **18 seconds and 1.125 MiB of logical media**. Exactly two copies initially exist, so initial stored payload is 2.25 MiB across the cluster. Each Peer has a 32 MiB quota, keeping quota exhaustion outside this comparison.

The fixture is deterministic synthetic bytes with unique segment hashes, not a downloaded encoded video. The original Go client performs actual network downloads and size/SHA-256 validation, then advances a simulated consumption clock. The content bit pattern does not change the configured payload limiter, and deterministic content makes paired treatments comparable.

The logical per-viewer bitrate is 128 KiB / 2 s = **64 KiB/s**. Two viewers demand 128 KiB/s; six demand 384 KiB/s before retries. This is offered logical demand, not a guarantee of achieved source throughput.

A separate real-media gate checks stored-fragment timing and decoding. That gate does not turn the synthetic performance runs into browser playback measurements. No claim of target-browser continuity is made by E1–E5.

## 4. Common E1–E4 configuration

| Control | Executed setting |
|---|---|
| Controllers / Peers | 3 / 3 |
| Placement / source selection | Fixed / round robin |
| Initial replicas | Exactly two per segment; no optional replicas |
| Peer 1/2/3 upload caps | 96 / 144 / 192 KiB/s |
| Shared limiter burst | Original 32 KiB token burst |
| Low / high viewers | 2 / 6 |
| Steady arrival times | Seeded arrivals within 0.5 s |
| Burst arrival times | Two seeded near 0; then 4, 4.2, 12 and 12.2 s |
| Client prefetch / concurrency | Window 3 / two downloads |
| Startup buffer | Four logical seconds, bounded by available content |
| Download-attempt timeout | Existing eight-second client default |
| Per-session deadline | 42 s from client start |
| First fault | 5 s after workload epoch |
| Observation horizon | 50 s after workload epoch |
| Main grace | 3 s |
| Grace sweep | 1 / 3 / 8 s |
| Brief absence | Four seconds, restart requested at epoch+9 s |
| Recovery objective | Full affected set first verified at two copies within 40 s of first fault |
| Source statistics / task / copy cadence | Nominal 1 / 0.5 / 2 s, subject to bounded request time |

The configured aggregate upload cap is 432 KiB/s before a failure. Losing Peer 1, 2 or 3 leaves 336, 288 or 240 KiB/s. Segment placement and queuing can overload an individual source even when a cluster-wide sum looks sufficient.

Seeds 1101/1102/1103 select failed Peers 1/2/3, respectively. Thus the three repetitions are three paired workload/fault contexts, with one trace per failed capacity. Capacity and seed are not independently crossed factors. E4 uses only 1101 and 1102.

Authoritative settings: [frozen study-config.json](../results/compact-study/study-config.json). Implementation: [repair_playback_study.py](../experiments/repair_playback_study.py).

## 5. Repair treatments

| ID | Behavior | Purpose |
|---|---|---|
| N | No first Peer failure; a marker is recorded at the same time | Intact-cluster reference |
| OFF | Same holder failure, automatic repair disabled | Isolate loss of service capacity without copying |
| S | Shared total upload service, repair enabled after grace | Existing repair behavior |
| F25 | S plus an additional repair rate cap of 25% of the source cap | Conservative fixed budget |
| F50 | S plus an additional 50% repair cap | Middle fixed budget |
| L | Additional repair cap adapted to source foreground activity and pending task age | Candidate adaptive treatment |

S's additional repair limiter is configured at the full shared rate and is nonbinding relative to the common limiter. Every segment GET class still consumes the unchanged total limiter. F25/F50/L add a restriction; they cannot raise total source capacity. Token bursts and already reserved chunks mean the rate is not a perfectly smooth instantaneous percentage. None of these caps reserves a share or guarantees a completion deadline.

### L's exact rule

1. Every 250 ms, sample active foreground GETs at each source, including requests waiting for upload service.
2. Use a 25% repair allowance while foreground is active, and during the initial idle waiting period.
3. After two continuous seconds with no active foreground GET, allow up to 100%.
4. If the oldest active committed pending task is more than 60 seconds old, raise the allowance to at least 50%, including during foreground traffic.
5. The final transfer still waits on the shared total limiter.

The coordinator normally executes repairs serially. Task creation time is supplied before the Raft command is applied, persisted as part of the task, and passed on repair GETs. It is not obtained inside deterministic FSM application, nor replaced by a new timestamp on retry. Legacy tasks without a timestamp do not get an invented creation age.

The short performance horizon did **not** empirically activate the 60-second branch. Boundary/age behavior is covered by regression tests; E5 also checks persistence of committed age across Leader changes. The recorded example [L-budget-trace.png](../results/compact-study/figures/L-budget-trace.png) is seed 1103/high load with 0/6 completed viewers. Its later idle period follows failures; it must not be presented as idle after successful playback.

Sources: [traffic.go](../internal/storage/traffic.go), [traffic_test.go](../internal/storage/traffic_test.go), [coordinator.go](../internal/control/coordinator.go), [task model](../internal/domain/models.go).

## 6. Pairing, lifecycle and run reuse

Each E1–E4 run starts an isolated, harness-owned cluster with new stores and Raft directories. The harness ingests the fixture, verifies publication and initial replica/capacity conditions, records the trace and starts observers and viewers. It then performs the scheduled failures/restores, observes to the horizon, records outcomes, archives process logs, and stops its own processes. It does not attach to a live demonstration cluster.

Within one seed/load cell, treatments receive the same content, initial placement, viewer arrivals, failed holder and declared fault schedule. Actual request and confirmed fault times are recorded separately. Treatment order is shuffled with a separate fixed seed, 73191. The study runs measured clusters serially. Fresh stores/processes do not mean the OS page cache was flushed.

| Experiment | Comparison memberships | Reused memberships | Unique runs added |
|---|---:|---:|---:|
| E1 | 3 policies × 2 demands × 3 contexts = 18 | 0 | 18 |
| E2 | 3 graces × 2 outage types × 3 contexts = 18 | 3 | 15 |
| E3 | 4 policies × 2 demands × 3 contexts = 24 | 3 | 21 |
| E4 | 3 policies × 3 second delays × 2 contexts = 18 | 0 | 18 |
| E5 | 3 cases × 3 repetitions = 9 | 0 | 9 |
| Total | 87 experiment memberships | 6 | **81** |

The shared S/high/grace-3/persistent cell belongs to E1, E2 and E3. There are three such actual runs, not nine independent observations. Reusing each twice saves six runs. Analysis must not sum all experiment memberships as independent repetitions.

## 7. E0 — Correctness and measurement gates

**Purpose:** verify that measurement and application paths are credible before interpreting the performance comparisons. E0 is validation, not an additional performance-policy cell.

The backend was built, and these nine serial gates were archived: static checks; Go race tests; harness regression tests; smoke; repair; cleanup; video deletion; fragment timing regressions; and real-media system validation. They include the existing installed/unregistered-replica and receipt/maintenance reliability fixes. Harness tests exercise failed probes, artificial delay, truncated bodies, open stalls, counter instance changes, pairing and censored recovery. Policy tests check rule boundaries and limiting behavior.

All gates passed. See [validation/gates.json](../results/compact-study/validation/gates.json), its adjacent logs and [validate_compact_study.py](../experiments/validate_compact_study.py).

**Interpretation:** this supports the tested implementation and measurement paths. It is not exhaustive correctness proof, a browser benchmark or coverage of every fault model.

## 8. E1 — Does repair itself interfere with delivery?

**Question:** after accounting for losing a Peer, does repair add playback interference?

**Design:** N/OFF/S × low/high demand × three paired contexts = 18 runs. S uses three-second grace. Failures are persistent.

**Procedure:**

1. Verify initial two-copy publication, hashes, capacity and source caps.
2. Start the same seeded viewer schedule and observers.
3. At five seconds, stop the selected actual holder for OFF/S; N records the corresponding marker without stopping it.
4. Record client request/verification/consumption events, source class counters, policy activity, tasks and current verified copy counts.
5. Keep failures and partial progress through the fixed horizon; check whether repair and foreground overlap at the same source.

**Comparisons:** OFF−N measures the observed consequence of losing the Peer without repair. S−OFF measures repair's additional observed effect for the same failure; it can include later benefits of a replacement, not only interference. S−N is the combined failure/recovery outcome.

| Demand | N completed | OFF completed | S completed | Mean stall N / OFF / S |
|---|---:|---:|---:|---|
| Low | 6/6 | 6/6 | 6/6 | 0 / 0 / 0 s |
| High | 18/18 | 7/18 | 7/18 | 3.20 / 3.24 / 4.46 s |

High-load mean paired S−OFF was +1.22 s/viewer; individual differences ranged from −0.13 to +3.46 s. In the slowest-Peer failure context, all viewers finished and S added 3.46 s/viewer versus OFF. Other high-load contexts already lost completion without repair. Same-source overlap was observed in all six E1/S runs.

**Conclusion:** repair interference appeared under a feasible contended condition, while low-load runs showed none. Do not interpret low observed stalls from early-failed sessions as improved playback.

**Evidence:** [E1-contention.png](../results/compact-study/figures/E1-contention.png), [E1-paired-effects.png](../results/compact-study/figures/E1-paired-effects.png), E1 rows in [aggregate.csv](../results/compact-study/aggregate.csv), and [paired-contrasts.csv](../results/compact-study/paired-contrasts.csv).

## 9. E2 — How does repair grace affect brief and persistent outages?

**Question:** can waiting avoid unnecessary repair after a brief disappearance, and what exposure cost remains when failure persists?

**Design:** S × grace 1/3/8 s × brief/persistent absence × three contexts = 18 memberships, of which three are reused from E1.

**Procedure:** fail the same holder at five seconds. For brief absence, restart it four seconds later using the same store and configuration; for persistent absence, leave it stopped. Record first task observation, first observed repair payload, task outcomes, traffic, original return and verified restoration. A task that already started may continue after the original returns; its traffic remains real work.

| Outage | Grace | Mean repair payload | Mean sampled one-copy exposure |
|---|---:|---:|---:|
| Brief | 1 s | 0.125 MiB | 4.08 MiB·s |
| Brief | 3 s | 0 | 4.08 MiB·s |
| Brief | 8 s | 0 | 4.13 MiB·s |
| Persistent | 1 s | 0.750 MiB | 17.42 MiB·s |
| Persistent | 3 s | 0.750 MiB | 18.24 MiB·s |
| Persistent | 8 s | 0.750 MiB | 20.20 MiB·s |

Every one-second-grace brief run copied one segment; the longer settings avoided that copy. All 54 brief-outage viewers completed. Persistent absence needed the same repair payload, but eight-second grace had about 16% more mean sampled exposure than one-second grace. Exact full restoration is not strictly monotonic with grace in every context because serial task execution, source demand, viewer termination and coordinator ticks also matter.

**Conclusion:** grace is a useful trigger tradeoff, not free recovery or guaranteed cancellation. Brief restoration here comes through original-Peer return, not proof of replacement-copy completion.

**Evidence:** [E2-grace.png](../results/compact-study/figures/E2-grace.png), [measurement-audit.csv](../results/compact-study/measurement-audit.csv), and the E2 grouped rows. Example [brief trace](../results/compact-study/peer-high-S-1-brief-0-1101/workload-and-fault-trace.json).

## 10. E3 — Does L improve the playback/recovery compromise?

**Question:** does adapting repair allowance improve outcomes compared with both fixed caps, rather than only with unrestricted sharing?

**Design:** S/F25/F50/L × high steady/burst demand × three contexts = 24 memberships, including three E1/E2 cells reused once. Main grace is three seconds, with persistent first failure.

**Procedure:** run the same paired lifecycle under each policy, preserving total source caps, fixture, failed holder, arrivals and timeouts. Log actual cap/activity history. Evaluate completion before interpreting stalls; then compare verified restoration, recovery target, exposure and repair payload.

**Predeclared engineering criteria:** first verified full restoration within 40 seconds after fault; one-second meaningful stall difference; five-percentage-point acceptable completion-loss margin. Three contexts cannot establish population noninferiority for that margin.

All four policies completed 7/18 steady and 9/18 burst viewers, with identical counts in each paired context. Steady target verification was 3/3 for every policy. Burst target verification was 2/3 for S/F50 and 1/3 for F25/L. One F25 burst run was unrecovered by the horizon; its restoration time remains null.

L−F50 mean observed stall was −0.55 s/viewer under steady demand and +0.01 s under bursts. In the slowest-Peer/high-load context, L reduced stalls 8.53→6.88 s/viewer but restoration was observed later, 33.99→36.06 s, with higher sampled exposure. This is a local compromise, not an overall advantage. One L burst observation was 40.05 s; the verification cadence does not support precise claims about a 0.05 s physical deadline miss.

**Conclusion:** no consistent L superiority over F50 was established. A negative or conditional result answers the question honestly. It does not prove adaptation cannot help under other workloads. The 60-second floor was not empirically evaluated by this horizon.

**Evidence:** [E3-policy-table.png](../results/compact-study/figures/E3-policy-table.png), [E3-tradeoff.png](../results/compact-study/figures/E3-tradeoff.png), [steady source-specific table](../results/compact-study/figures/E3-high-by-failed-peer.png), [burst source-specific table](../results/compact-study/figures/E3-burst-by-failed-peer.png), and paired CSV rows.

## 11. E4 — What happens if both original holders stop?

**Question:** does the chosen replacement actually serve an affected segment after the remaining original holder fails?

**Design:** S/F50/L × second-failure delay 4/14/30 s × two contexts = 18 runs. F50 was prespecified as the middle fixed cap, retained after the separate pilot, not selected from final outcomes. First failure remains epoch+5 s; second failures are epoch+9/+19/+35 s.

**Procedure:**

1. Before outcomes are known, select a late-queue affected segment, identify both original holders and the third target.
2. Stop the first holder on schedule.
3. Before the second stop, record target receipt and committed registration state.
4. Stop the other original holder regardless of repair progress.
5. GET that specific segment directly from the replacement target with a bounded five-second probe; verify size/hash and query committed registration separately.
6. Keep live-viewer outcomes separate, because the selected segment may already be buffered.

A successful installed body without committed registration is not counted as a committed replacement delivery. A completed viewer is not substituted for a direct target probe.

**Result:** every policy/delay cell was 0/2 successful hash-valid-and-committed target deliveries, total 0/18. The late-queue sentinel was not ready at these times. Some viewers nevertheless completed from previously fetched content.

**Conclusion:** this demonstrates the availability boundary during reduced redundancy. These chosen delays did not discriminate policies or observe the later transition to replacement availability. The processes retained disks; this is not permanent data loss or a real-world loss probability.

**Evidence:** [E4-target-delivery-counts.png](../results/compact-study/figures/E4-target-delivery-counts.png). Example [precondition snapshot](../results/compact-study/peer-high-F50-3-persistent-30-1101/second-fault-precondition.jsonl), [correctness/probe result](../results/compact-study/peer-high-F50-3-persistent-30-1101/correctness.json).

## 12. E5 — Controller recovery and application safety

E5 uses separate correctness/recovery scenarios. **Do not apply every E1–E4 fixture, five-second trigger or horizon to E5.** There are three repetitions per case, nine runs total. Script: [controller_recovery_study.py](../experiments/controller_recovery_study.py).

### E5a: actual Leader fails during delivery

- Ingest a nine-segment main fixture and a separate one-segment, 4 KiB cold fixture.
- Start a warm client on the main fixture. About four seconds after its workload epoch, identify and kill the actual current Leader.
- Launch a new uncached client on the cold fixture; it has no previously fetched metadata.
- Directly probe Controller roles and fresh application metadata with bounded one-attempt requests. The application endpoint requires quorum-backed service; health or role alone is insufficient.
- Verify a different Leader, fresh application recovery, and completion of both clients.

All three cases passed. Fresh application recovery median was 1.52 s, range 1.47–2.74 s after Leader stop. Warm/cold outcomes are correctness observations with different fixtures, not a controlled latency comparison between equally sized videos.

**Evidence:** [E5-annotated-timeline.png](../results/compact-study/figures/E5-annotated-timeline.png), [E5a-1101 metadata probes](../results/compact-study/E5a-1101/metadata-probes.jsonl), client events and injections in the same directory.

### E5b: actual Leader fails with a committed pending repair

- Use one main segment. Source caps are deliberately reduced to **32/48/64 KiB/s**, one third of the normal caps; repair policy is L and grace is one second.
- Stop an actual original holder and wait until a committed pending task can be observed. Save its ID and persisted creation timestamp. A missed pending precondition is not a successful test.
- Identify and kill the Leader while that task is pending.
- Observe fresh application recovery and require the replacement to complete the same task; completion must not silently switch to a new identity.
- Directly GET and hash-check target bytes and verify committed target registration.
- Restart the interrupted Controller and check task persistence. Restart the source Peer with its original configuration and check that L remains configured while the counter instance ID changes.

All three cases passed, including identity/age, target bytes, registration and restart checks. Median fresh metadata recovery was 1.60 s, range 1.52–1.97 s. **These are not repair-completion latencies.** The task can finish later; its state snapshots record that progress separately.

**Evidence:** [pending-task.json](../results/compact-study/E5b-1101/pending-task.json), [task snapshots](../results/compact-study/E5b-1101/tasks-and-copy-verification.jsonl), [correctness.json](../results/compact-study/E5b-1101/correctness.json), [policy-restart.jsonl](../results/compact-study/E5b-1101/policy-restart.jsonl).

### E5c: majority loss, refusal and restoration

- Commit baseline state, identify the actual Leader, and stop both other Controller processes.
- Wait 2.5 s for stable isolation.
- Submit a valid new uniquely identified create command three times during isolation. Record actual responses; the test is no success acknowledgment, not a fabricated assumption that all unknown requests disappear.
- Record the quorum-restore event, restart the followers, and directly observe fresh application service.
- Verify baseline metadata is retained. Reconcile the new command with its **original operation ID** and check that the object exists.

All three cases passed. All nine isolated write attempts returned HTTP 503. Baseline retention and reconciliation passed in all three cases. Median fresh application recovery was 1.04 s, range 0.85–1.43 s **from the recorded restore event**. That event starts the restart procedure; the latency includes startup/election until usable service. It is not timed from a separately proven healthy-quorum instant.

The raw fault-relative application times are longer because they include deliberate isolation. Never compare those directly with the restore-relative table without naming the origin. This is process-stop testing, not a sustained network partition with both sides still running.

**Evidence:** [isolation-writes.jsonl](../results/compact-study/E5c-1101/isolation-writes.jsonl), [quorum-timing.json](../results/compact-study/E5c-1101/quorum-timing.json), [reconciliation.jsonl](../results/compact-study/E5c-1101/reconciliation.jsonl), [correctness.json](../results/compact-study/E5c-1101/correctness.json).

### E5 combined interpretation

All nine tested Controller cases passed. The result supports application recovery, task continuity, integrity checks and safe refusal in these specific scenarios. It is not universal reliability proof, nor Byzantine, power-loss or arbitrary network-partition coverage.

Use [E5-safety-table.png](../results/compact-study/figures/E5-safety-table.png) for the main presentation. [E5-controller.png](../results/compact-study/figures/E5-controller.png) preserves the fault-relative distribution, so its E5c values include isolation.

## 13. How metrics are measured

| Metric | Meaning and important qualification |
|---|---|
| Completion | Completed viewers / all scheduled viewers; start failures and timeouts stay in the denominator |
| Post-fault observed stall | Portions of stall intervals after the actual fault, including the observed portion in failed sessions; startup excluded |
| Startup | Client start to first playable contiguous prefix; reported conditional on completion where the column says so |
| Rebuffering ratio | Stall / (consumed media + stall); completed-only columns must remain labeled |
| Full affected-set restoration | First sampled observation when all initially affected segments have two distinct reachable, hash-valid, committed holders |
| Restoration interval | Previous verification sample to first restored sample, relative to the fault; not an exact millisecond completion timestamp |
| One-copy exposure | Step estimate of segment sizes × time at exactly one valid reachable committed copy, summed across affected segments; bytes·s or MiB·s |
| Zero-copy / unknown exposure | Recorded separately; an unsuccessful observation is not silently treated as zero valid physical copies |
| Repair/foreground/probe payload | Source ResponseWriter byte counters for classified GETs, including retries and partial transfers; not TCP/NIC traffic |
| Recovery target | Whether full restoration was first verified by the declared 40 s after-fault objective |
| Fresh metadata recovery | Actual fault/restore event to first successful direct quorum-backed application response; separate from observing a role |

Client elapsed time is retained and anchored at client launch; observer event receipt times are also saved. The harness uses monotonic time for its own schedule. Probe start/end, status and error are recorded instead of hiding retries in an ingestion helper.

Copy verification intersects hash-valid Peer inventory with quorum-backed registrations and checks process liveness. A completed task flag or two historical location entries alone is insufficient. The nominal two-second cadence produces observation brackets and step-estimate uncertainty.

Source statistics are nominally sampled each second and also just before stopping a Peer. Instance IDs distinguish counter resets on restart. Final aggregation uses each class/instance's monotonic maximum. A small amount written between the final pre-crash snapshot and process exit may be unobserved. This limitation applies to payload accounting; no physical-wire bandwidth claim is made.

## 14. Analysis, audit and artifact map

Analysis treats the run/context as the repetition, not each viewer as an independent replication. Paired differences use the same seed/load/fault conditions. Individual values and ranges are retained; three contexts do not justify precise tail latencies, population confidence bounds or noninferiority claims.

Unobserved recovery remains null with a censoring reason. Setup failure, missed exposure and missed pending-task preconditions are distinct from a passed scenario. A run's lifecycle status `completed` means its observation finished, not that every viewer completed.

The final audit found all 72 performance lifecycles complete, all intended injections/exposures present, all 396 scheduled outcomes present and terminal within the horizon, and zero observed policy-sample rule violations. These are observed checks, not mathematical proof of fairness or service guarantees. Original raw journals were not changed during final analysis.

| Artifact | What it contains |
|---|---|
| [study-config.json](../results/compact-study/study-config.json) | Executed settings |
| [matrix.json](../results/compact-study/matrix.json) | Actual cells, order and shared experiment memberships |
| [environment-and-provenance.json](../results/compact-study/environment-and-provenance.json) | Environment, source fingerprints and binary hashes |
| `source-snapshot.tar.gz` | Executed study source snapshot, including dirty local code |
| [runs.jsonl](../results/compact-study/runs.jsonl), [summary.csv](../results/compact-study/summary.csv) | Enriched run summaries |
| `runs.observed.jsonl` | Original observed summaries preserved before final enrichment |
| [aggregate.csv](../results/compact-study/aggregate.csv) | Grouped slide-ready numbers |
| [paired-contrasts.csv](../results/compact-study/paired-contrasts.csv) | Matched per-context effects and ranges |
| [sessions.csv](../results/compact-study/sessions.csv) | All performance-viewer outcomes |
| [controller-runs.jsonl](../results/compact-study/controller-runs.jsonl), [controller-summary.csv](../results/compact-study/controller-summary.csv) | E5 results and restored-quorum-relative metrics |
| [analysis-audit.json](../results/compact-study/analysis-audit.json) | Outcome, timing, counter and policy audit |
| [execution.json](../results/compact-study/execution.json) | Unique run counts and elapsed execution time |
| `analysis-source/` | Final plot/audit scripts used for the enriched exports |
| `SHA256SUMS` | Final artifact checksums |

A performance run directory also contains `config.json`, `manifest.json`, `workload-and-fault-trace.json`, `injections.jsonl`, `client-events.jsonl`, `sessions.jsonl`, `peer-traffic-and-policy.jsonl`, `metadata-probes.jsonl`, `tasks.jsonl`, inventory/copy verification, `correctness.json`, `summary.json`, and `process-logs/`. E4 adds second-fault preconditions. E5 directories have case-specific task, role, restart or isolation/reconciliation files.

Example full run directory: `results/compact-study/peer-high-S-3-persistent-0-1101/`. Fields in the name identify case, demand, treatment, grace, outage, second delay and seed. Use its recorded `experiments` membership to determine reuse rather than counting directory names again for each experiment.

## 15. Reproduction and reuse

To regenerate current figures/tables from saved data, without rerunning services:

```sh
python3 experiments/finalize_compact_study.py results/compact-study
```

For a fresh study, build the backend and run the validation script, then invoke the E1–E4 and E5 runners with a **new output directory**. These commands launch real local processes and require local ports:

```sh
make backend
python3 experiments/validate_compact_study.py
python3 experiments/repair_playback_study.py --output-dir results/compact-study-new
python3 experiments/controller_recovery_study.py --output-dir results/compact-study-new
python3 experiments/finalize_compact_study.py results/compact-study-new
```

Do not overwrite the archived final study. `--resume` is only for continuing a compatible existing study with the same configuration/binaries, not mixing changed settings. A rerun is a new observation and need not reproduce every scheduling-dependent latency exactly.

The canonical numerical evidence stays under `results/compact-study/`; this `presentation/` folder contains explanatory documents. The bundle is a convenient copy of exhibits, not a substitute for raw provenance. If regenerating exports, rebuild the ZIP/checksum manifest as needed before distributing a refreshed bundle.

## 16. What this project can claim

The experiments give scoped answers: repair interference depended on load/source failure; grace saved brief-outage work; L showed a local compromise without consistent superiority over F50; a second holder loss exposed unavailable replacement data; and all nine tested Controller scenarios preserved the checked application contract.

The study is sufficient for presenting an implemented prototype and an exploratory evaluation. It is not evidence for an optimal algorithm, universal availability, physical multi-machine fault isolation, permanent-loss probabilities, browser performance, or broad population superiority. Those stronger claims require different or larger experiments.
