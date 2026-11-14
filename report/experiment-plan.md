# Experiment Plan: Playback and Replica Recovery

TODO(team): Maintain the original experiment specification and its link to the superseding compact design.

Prepared: November 14, 2026  
Scope: experiments needed to support the research questions and result pages in [the PowerPoint outline](powerpoint-outline.md).

**Execution update:** The user reduced the budget to one or two hours. The [compact design](compact-experiment-design.md) supersedes the larger matrix and timing values below. All 81 compact E1–E5 runs completed in 67.9 minutes; see [results and scoped conclusions](compact-study-results.md). The larger matrices below remain the original specification, not the executed sample size.

The quantitative harness, classified source accounting, F25/F50 caps, persisted task ages and load-aware L are now implemented and validated under the compact study. Headless consumption remains distinct from browser playback.

## 1. Questions, hypotheses, and required evidence

| Question | Hypothesis or contract | Experiments | Presentation page |
|---|---|---|---|
| RQ-A: when does repair interfere with playback? | H1: shared-source contention can add stalls under constrained load, with smaller effects when capacity is spare | E1 | 7 |
| RQ-A: when should repair start? | H2: grace can avoid some work for brief absence but extends reduced redundancy during persistent absence | E2 | 7 |
| RQ-B: can a load-aware repair budget improve the compromise? | H3: it improves playback/recovery outcomes over simple fixed budgets in some feasible workloads without hiding completion losses | E3 and E4 | 8 |
| RQ-C: what failure behavior does the application retain? | Verified bytes, persistent task continuity, conditional Leader recovery, and refusal of unsafe quorum-backed progress | E0 and E5 | 9 |

**Primary scope:** repair behavior. Hold placement and playback selection constant. The existing four placement/selection groups are a follow-up ablation, not a required extra dimension in every run.

## 2. E0: implementation and measurement gates

Complete these before interpreting performance data:

1. Keep the fixed valid-but-unregistered repair-target regression passing. The coordinator now adopts verified installed bytes through Raft and resumes same-task installed targets without recopying. Unit and isolated cluster checks cover lost responses and offline original holders.
2. Keep the equivalent-PUT/cleanup regressions passing. Installed receipt identity is now stable, maintenance accepts equivalent object receipts, and deletion retires all recorded aliases. Coverage includes old persisted indexes and Peer restart. Cleanup stays disabled during performance experiments.
3. Pass backend build, vet, race tests, and existing smoke/repair checks on the exact study binaries.
4. Keep stored-fragment timeline validation and media decoding checks passing. The previous timestamp assertion used an inferred nominal FFprobe packet duration and has been replaced by exact BMFF timing checks with real-gap/overlap regressions. Validate the presentation browser separately before claiming seamless media playback. Headless studies remain labeled as simulated consumption.
5. Add transfer classification and per-class byte/request counters if claiming repair-specific bandwidth. Preserve the common total Peer upload limit. The current total-byte counter combines foreground and repair traffic.
6. Validate the measurement harness with known artificial delays, failed probes, missed fault preconditions, truncated transfers, and unrecovered tasks. Confirm all produce honest records instead of fabricated recovery times.
7. For proposed policies, verify that ordinary and repair requests obey the same total cap, repair caps behave as defined, and task/byte verification survives a Leader change. Preserve policy settings across Controller restart and record Peer counter instance IDs.

**Existing commands, run from the repository root:**

```bash
make backend
go vet ./...
go test -race ./...
python3 tests/system/smoke.py
python3 tests/system/repair.py
python3 tests/system/cleanup.py
python3 tests/system/delete_video.py
python3 tests/system/media.py
```

These are acceptance checks. Passing them does not produce the quantitative E1–E5 study automatically.

## 3. Treatments and a policy definition to implement

| ID | Definition | Status |
|---|---|---|
| N | No Peer failure with ordinary shared-service repair enabled | Supported by existing mechanisms |
| OFF | Peer failure with automatic repair disabled | Existing configuration, diagnostic reference |
| S | Peer failure with current shared limiter and configured grace | Existing behavior |
| F25 | S plus a repair-specific cap equal to 25% of source Peer upload capacity | Implemented; compact study |
| F50 | S plus a repair-specific cap equal to 50% of source Peer upload capacity | Implemented; compact study |
| L | S plus the bounded load-aware cap rule below | Implemented; compact study |

All policies retain identical total upload capacity. OFF is useful for isolating contention, but it leaves redundancy reduced and cannot win the overall recovery comparison simply by transferring nothing.

### 3.1 Concrete candidate L

Use this as the initial implementation specification, then freeze any pilot changes before the final study:

- Every 250 ms, each source Peer observes active foreground GETs, including requests waiting for upload service.
- When foreground requests are active, limit repair to 25% of that Peer's configured upload cap.
- After two continuous seconds with no active foreground GET, allow repair up to the total upload cap.
- When the oldest still-pending repair exceeds 60 seconds of age, raise the permitted repair cap to at least 50%, even during foreground activity.
- Every transfer still consumes the original common total limiter. A repair-specific cap is an additional restriction and cannot increase total capacity.

Track repair age using committed task identity and stable task-associated timing. The current Task DTO has no creation timestamp. Add a persisted timestamp supplied in the command, or an equivalent stable mechanism, if the rule must retain age semantics across Leader/restart changes. Do not obtain time inside deterministic FSM application. A first-request local timestamp is a different policy and must be documented as such.

The age rule raises an allowance. It does **not** guarantee a reserved share or a hard completion deadline. A formal service guarantee would require a scheduler with defined reservations/fairness. Evaluate whether this simple rule actually makes enough progress.

This candidate uses traffic activity, not browser-buffer telemetry. Call it **load-aware repair**. Treat the candidate as a modest adaptation of established maintenance-budget ideas, not a first-ever repair-throttling algorithm.

## 4. Common setup and controls

### 4.1 Starting configuration for the pilot

| Item | Proposed value | Reason / constraint |
|---|---|---|
| Topology | 3 Controllers and 3 Peers | Controller quorum and an eligible replacement target |
| Content | One video, 32 segments, 2 s logical duration each, 256 KiB each | 64 s consumption and 8 MiB logical data, enough to observe startup and a fault |
| Initial replication | Exactly 2 confirmed copies per segment, no optional copies | A failed holder produces a real repair requirement |
| Peer quota | At least 32 MiB per Peer | Capacity should not confound this study |
| Upload caps | 256, 384, 512 KiB/s for Peers 1–3 | Explicit heterogeneous source constraints |
| Placement / selection | Fixed / round robin | Existing reproducible baseline, held constant |
| Client configuration | Window 3, concurrency 2, startup buffer 4 s | Same across treatments, valid for 2 s segments |
| Steady demand | Low: 2 viewers. Higher: 4 viewers. Seeded arrivals within 2 s | Pilot values, must verify actual source utilization |
| Fault time | 20 s after the common harness workload epoch | Same wall-clock injection schedule for paired treatments |
| Brief absence | Restore the failed Peer after 4 s | Probe temporary unavailability |
| Persistent absence | Leave the failed Peer down until observation ends | Measure recovery through a new holder |
| Grace sweep | 1, 6, 18 s | Existing controller interval is 2 s, so 1 s is not instantaneous detection |
| Session deadline | 180 s | Preserve failures/timeouts and partial progress |
| Fixed observation horizon | 240 s from workload epoch | Includes post-playback redundancy and traffic accounting |

These are proposed pilot values, not measured sweet spots. The steady per-viewer logical bitrate is 128 KiB/s. Even when aggregate surviving bandwidth exceeds demand, segment placement can leave a specific source overloaded. Pilot with no-fault and failed-Peer/OFF runs to determine actual spare versus constrained source conditions. Freeze final settings and include them with every graph.

The current reusable harness fixes three Peers and a 16 MiB quota. Extend its quota/fixture settings as necessary. Do not claim arbitrary Peer-count scaling from this three-Peer study. Record an intentionally limited set of fixed initial placements; do not silently let each treatment get a different placement.

### 4.2 Paired trace generation

Use at least ten independent workload seeds initially. Within a seed, every treatment receives the same content, arrivals, failed Peer ID, injection time, restore schedule, and observation horizon. Rotate the failed Peer across seeds and report effects by source/capacity when useful. Do not pick an easier fault separately for each policy.

Preselect an affected segment and verify its two original holders and replacement eligibility. Also verify that the workload actually demanded affected, not-yet-buffered content after the fault. Save a missed-exposure/injection status when the intended event does not occur. Preserve its evidence and report its frequency, including when the treatment itself caused the miss.

For E3, add a fixed burst trace: start two viewers near the common epoch, add two more around 12 s, and add two around 40 s. All view the same finite fixture and eventually finish or time out, yielding changing foreground activity. Pilot may adjust these arrivals to create reproducible pressure and recovery. Do not guarantee a synchronized idle interval unless the trace creates one.

### 4.3 Environment and resource matching

Run one measured cluster at a time on an otherwise quiet machine. Use new store/Raft directories and process state for each run. Randomize treatment order independently of workload seed. Fresh directories do not necessarily imply a cold OS page cache, so use the same cache procedure and record the limitation.

Match initial physical bytes, source limits, quotas, client settings, timeout values, repair target policy, source-selection rules, and fixture hashes. Record production versus test overrides, including grace and polling intervals. Use unique fixture contents to avoid unexpected content deduplication.

Keep raw failures. Never change a seed's workload or timeout only for a policy that performs poorly. With ten traces, uncertainty may be wide. Increase the final repetition count according to a rule fixed before reading final outcomes. Do not claim population noninferiority from a nonsignificant test.

## 5. Required experiments

### E1: Does repair itself interfere with delivery?

**Answers:** RQ-A / H1. **Code starting points:** `tests/system/cluster.py`, `repair.py`, and the CLI/event collection in `experiments/paired.py`.

**Matrix:** N, OFF, and S at grace 6 s, crossed with low/higher steady demand and ten paired seeds. **60 runs.** Persistent Peer absence.

**Procedure:**

1. Start and ingest a fresh cluster, verify READY, budgets, and holders.
2. Start observers and the viewer trace. At 20 s, kill the preselected actual holder for OFF/S. N records a marker at the same time without killing it.
3. Capture client transfer/consumption events and class-specific source traffic. Record which repairs overlap active playback.
4. Verify current reachable copies and task state until the same 240 s horizon. Preserve all session failures and full logs.

**Contrasts:** OFF minus N estimates the effect of losing the Peer without repair. S minus OFF estimates repair's additional effect for the same failure, including any later benefit from the new copy. It is not necessarily harmful throughout the entire run. S minus N is the full failure-plus-recovery outcome.

**Report:** completion, startup, post-fault stall seconds, conditional rebuffering, total/repair bytes, redundancy trajectory, and recovery censoring. Prove overlap through per-source repair/foreground traces before attributing a stall increase to bandwidth contention.

**Slide output:** E1 paired differences by load, with completion alongside. Null interference under spare capacity is a valid result.

### E2: How does grace change brief and persistent outages?

**Answers:** RQ-A / H2. **Starting point:** existing `REPAIR_GRACE` setting and process restart helper.

**Matrix:** S with grace 1, 6, or 18 s, crossed with 4 s/persistent absence, at higher steady demand and ten seeds. **60 runs**, ten of which match E1's S/higher-demand/persistent cell and can be reused without rerunning or double-counting.

**Procedure:** Inject the same holder failure, restore at the declared time for brief absence, and observe actual detection, task creation, first repair byte, target installation, and verified two-copy restoration. Count copies in flight at restoration as real work, including any that continue after the original returns.

**Report:** actual repair bytes, task attempts, copy completion/failure, source demands, exposure duration, completion and stalls. Record recovery both from original-Peer return and from a new target so they are not confused.

**Interpretation:** Grace is a detection/trigger control. A longer grace may save bytes for short outages but cannot be assumed to cancel an already-started task. For persistent absence, identify how much added exposure comes from grace versus queued serial repairs.

**Slide output:** repair bytes and byte-weighted exposure against grace, separated by outage type.

### E3: Does L improve the playback/recovery compromise?

**Answers:** RQ-B / H3. **Prerequisite:** implement and validate F25, F50, L, and repair accounting.

**Matrix:** S, F25, F50, L, crossed with higher steady/burst demand and ten seeds. **80 runs**, with ten S/higher-steady runs reusable from E1. Hold grace at 6 s and use persistent Peer absence.

**Procedure:** Use the same paired lifecycle and demand/fault schedules. Log the actual repair cap and foreground activity per source at each control update. Verify that L follows its declared busy/idle/age rule, including after a policy-state restart in a separate correctness check.

**Predeclare engineering targets:** Start the pilot with full affected-set verified redundancy restored within 180 s after the first fault. Before final runs, retain or revise that target based on fixture feasibility and report the rationale. Set an acceptable completion-loss margin and a meaningful stall difference before the final analysis. These are study objectives, not guaranteed service properties.

**Report:** completion and time-to-failure/progress, stalls, recovery-within-target fraction, restoration time, exposure, total/repair bytes, failed attempts, and L's actual budget history. A lower stall average among only the surviving sessions is insufficient evidence of improvement.

**Baseline fairness:** Show both fixed caps. If selecting a single tuned fixed policy for E4, select it from separate pilot data under the declared target and freeze it. Do not choose a weak fixed baseline to make L look better or tune each policy using its final test outcomes.

**Decision:** Determine whether L improves one objective without a material measured regression in others, whether it provides a quantified compromise, or whether fixed caps are just as effective. Wide uncertainty warrants an inconclusive verdict.

**Slide output:** completion/recovery-target table plus the stall-versus-restoration-time plot. Unrecovered runs appear explicitly, not as zero or omitted latency.

### E4: What is the consequence of the reduced-redundancy interval?

**Answers:** RQ-B's recovery cost and RQ-C's data-availability boundary. This is an explicit stress test, not a real-world loss-probability estimate.

**Matrix:** S, the pilot-selected fixed policy, and L, with a second-holder failure 10, 40, or 100 s after the first, and five paired repetitions. **45 runs.** Freeze these delays before final execution; they illustrate a range rather than guaranteeing a transition in every workload.

**Procedure:**

1. Before the run, select a segment held by the first failed Peer and one other original Peer. Identify the eligible third target. The segment, node choices, and fault times must match across treatments.
2. Apply the first failure. Immediately before the second fault, record whether a valid committed replacement exists.
3. Kill the remaining original holder at the declared delay, regardless of whether repair has completed.
4. Probe the replacement target directly and verify the segment size/hash. Also query its committed registration through available Controllers. Distinguish an installed but uncommitted copy from a registered usable copy.
5. Record live-viewer outcomes separately. If the selected segment is already buffered, a continuing viewer is not proof that the replacement can serve it.

**Report:** correct target delivery, committed target status, probe latency/failure, copy-in-flight state, and playback outcomes. The whole video's full redundancy can recover later than this single sentinel segment, so keep the two measurements separate.

**Slide output:** target-delivery outcomes by second-fault delay, next to the E3 exposure tradeoff. Do not call stopped processes with intact disks permanent data loss.

### E5: Does metadata/task recovery preserve the application contract?

**Answers:** RQ-C. Keep these runs separate from the main Peer-contention comparisons.

| Case | Injection / procedure | Required checks and measurements | Starting code |
|---|---|---|---|
| E5a Leader failure with active delivery | Kill the actual Leader during a long session, directly probe fresh metadata, keep a warm session running | Replacement role, first usable quorum-backed request, warm-session outcome, cold uncached request outcome, retry counts | `smoke.py`, `media.py` |
| E5b Leader failure during pending repair | Record a committed pending task ID, kill Leader, verify the replacement's work | Same task identity, correct target bytes and committed registration, completion or explicit failure, no silent disappearance | `repair.py` |
| E5c Majority loss and restoration | Commit baseline state, stop two Controllers, wait for stable isolation, submit a unique new command, restore quorum and reconcile | No success acknowledgment during isolation, original state retained, unknown outcomes reconciled using original IDs, application recovery | `smoke.py` |

Use ten repetitions per case initially, **30 runs**. For E5b, slow or gate the copy enough to make the pending state observable. A missed pending precondition is not a successful recovery test. Repeat policy-state persistence for L as a regression without claiming broad statistical coverage from one trace.

The existing ingestion controller helper hides retries. Use direct one-attempt timed probes for measurement. `/health` and a role string alone do not show application availability. A write submitted entirely during sustained isolation must not be acknowledged as committed then, but an ambiguous or queued request can legitimately commit after quorum returns.

**Slide output:** one annotated timeline with application recovery distribution, and a passes/attempts table including missed injections and failed checks.

## 6. Metrics, timing, and collection requirements

### 6.1 Definitions

| Metric | Exact meaning |
|---|---|
| Completion | Completed / all scheduled sessions, retaining start failures and timeouts |
| Startup | Start to first playable prefix in headless mode, separately first `playing` in browser mode |
| Post-fault stall seconds | Portion of stall intervals after the actual first-fault time, including the observed portion before a failure/timeout |
| Rebuffering | Headless stall / (simulated media consumed + stall), excluding startup, reported conditional on completion with completion beside it |
| Alternate-delivery latency | Fault to first relevant post-fault download that validates from a surviving source. Missing exposure produces no such measurement |
| Segment redundancy restoration | First verified observation of at least two distinct reachable hash-valid holders with committed registrations for that segment |
| Full affected-set restoration | All segments whose valid-copy count dropped below two at the first fault satisfy the restoration condition at the same sampled observation |
| Byte-weighted one-copy exposure | Integral over time of the sum of sizes of affected segments with exactly one reachable, valid, committed copy. Units: byte-seconds |
| Zero-copy exposure | Record separately the time and segment bytes with no valid reachable committed copy |
| Repair / foreground bytes | Bytes written by the source Peer for classified requests, including retries and failed attempts. Payload bytes, not TCP/NIC traffic |
| Metadata recovery | Fault to the first successful fresh quorum-backed application probe, separate from role/health observations |
| Recovery target | Fraction of runs meeting the declared restoration deadline, with unrecovered/late runs retained |

The current client failed-request counter covers segment attempts, not metadata requests. The harness must collect metadata retry counts separately. Current deadline-miss counts cover consumed noninitial segments only and cannot replace completion or stall metrics.

### 6.2 Observation without distorting the study

Use the harness's monotonic clock for injection, direct probes, and event receipt. Store client source timestamps too, but do not subtract unrelated process clocks without explicit alignment. If calculating precise post-fault stall intervals, add a documented client elapsed-time alignment or record uncertainty from harness event receipt.

Sample metadata roles/tasks at a declared interval, such as 200–500 ms, and cheap per-Peer stats around once per second. Full inventory and receipt validation recompute hashes, so avoid full-disk scans every 200 ms. Use a declared lower verification cadence, such as two seconds, plus targeted final verification. Keep observer load the same across policies and measure observer overhead in E0. Report restoration time as an observed interval with the corresponding sampling uncertainty.

The task state `completed` or two entries in `/locations` is insufficient proof of two currently valid copies. Conversely, receipt failure can be a probe timeout under contention rather than proven corruption. Preserve reason codes and distinguish observed unreachability from physical destruction.

### 6.3 Analysis

- Aggregate at run/seed level. Viewers sharing a source are correlated and do not become independent repetitions.
- Use paired per-seed differences for matched treatments. Report run counts, uncertainty method, all completion rates, and failure causes.
- With small samples show individual points, median, and range. Do not advertise precise p99 recovery from ten runs.
- Keep censored recovery, setup failures, and missed exposure visible. Do not compute favorable latency only over recovered runs without naming that conditioning.
- Compare the full workload horizon and the playback-overlap interval separately. Repair after all viewers finish still contributes to recovery time and total bytes.
- Freeze workload and policy parameters after a separate pilot. Preserve negative findings, including lower throughput caused by overly restrictive repair caps.

## 7. Scripts and artifacts to build

### 7.1 Implemented study files

- `experiments/repair_playback_study.py`: E1–E4 matrices, paired traces, lifecycle, fault injection, collection, per-run checks.
- `experiments/controller_recovery_study.py`: E5 direct timed probes and task/quorum checks.
- `experiments/analyze_repair_playback.py`: summaries, paired contrasts, recovery target tables, figures, and censored-outcome reporting.

These files are now runnable and reuse the original cluster helper and Go playback engine. Their compact configuration supersedes the original run counts above; see the compact design and archived final data.

### 7.2 Shared lifecycle

```text
validate config and create a unique output directory
start fresh harness-owned processes
verify quorum-backed metadata, READY, hashes, replicas, capacity, and source caps
capture initial counters and start observers
launch the recorded viewer trace
record fault request, confirmed process exit, and the intended affected objects
apply any declared restore or second-fault schedule
observe to the fixed horizon, retaining terminal failures and censored recovery
reconcile ambiguous outcomes and verify scenario invariants
copy raw logs, event streams, final snapshots, and binary/config hashes
stop harness-owned processes in a finally block and clean temporary data
```

Store artifacts outside temporary directories before the cluster helper deletes them. Do not attach to or kill the live demo's processes. Account for counter resets by Peer instance ID after restart.

### 7.3 Required output contract

```text
results/<study-id>/
  study-config.json
  environment-and-provenance.json
  runs.jsonl
  summary.csv
  figures/
  <run-id>/
    config.json
    manifest.json
    workload-and-fault-trace.json
    injections.jsonl
    client-events.jsonl
    sessions.jsonl
    peer-traffic-and-policy.jsonl
    metadata-probes.jsonl
    tasks-and-copy-verification.jsonl
    correctness.json
    process-logs/
```

Each event identifies run, seed, treatment, node/segment/task where relevant, and observer time. A run summary includes precondition status, all session outcomes, every safety check, recovery status/time or censoring reason, and cost counters. Never substitute zero for an unobserved recovery event.

## 8. Execution order, scope, and expected cost

| Phase | Deliverable | Gate for the next phase |
|---|---|---|
| 1 | E0 regression and measurement checks | Correctness blockers resolved for the claimed paths |
| 2 | Small E1 pilot plus brief/persistent examples | Real fault exposure and playback/repair overlap established |
| 3 | Finalize demand, horizons, targets, and policy parameters | Freeze configuration and archive pilot separately |
| 4 | Final E1/E2 | RQ-A has a measured answer, including null results |
| 5 | Implement/validate caps and L, then final E3/E4 | RQ-B has evidence against credible fixed baselines |
| 6 | E5 and presentation-browser validation | RQ-C and demo claims have direct evidence |
| 7 | Analysis and slide exhibits | Every conclusion traces to raw runs |

With the proposed initial replication count, E1–E3 total 200 run slots, with 20 exact shared cells reusable, giving **180 unique performance runs**. At a 240 s observation horizon, that is **12 hours of observation time**, plus setup, verification, and analysis. E4 adds 45 runs and up to three more observation hours. E5 adds 30 shorter correctness/recovery runs. Do not run measured clusters concurrently just to shorten the wall time.

This is a planned study size, not a required precision guarantee. If time is limited, complete E0, E1/E2, and E5 first and present a scoped empirical characterization. Do not claim to answer H3 unless the proposed treatment and credible baseline comparisons actually run. Reduce the scope of the question openly rather than manufacturing an adaptive-policy result.

No current-code-only experiment can establish that an unimplemented repair policy works. Similarly, a ten-minute live demo cannot replace the repeated study.

## 9. Final answer checklist

- [ ] RQ-A states when contention appeared, when it did not, and how grace affected brief/persistent outages.
- [ ] RQ-B compares L against both fixed caps and reports completion, recovery targets, exposure, and bytes.
- [ ] E4 exposes the consequence of a second failure without pretending to estimate real-world loss probability.
- [ ] RQ-C distinguishes role election, usable metadata recovery, cached playback, and data availability.
- [ ] Null, failed, timed-out, and missed-injection outcomes remain in the evidence.
- [ ] Figures identify exact configuration, binary version, trace set, units, and uncertainty.
- [ ] Browser claims come from the target browser, and headless performance remains labeled as simulated consumption.
- [ ] Slide 7 uses E1/E2, slide 8 uses E3/E4, and slide 9 uses E0/E5.

The research is complete when these scoped questions have evidence-backed answers. A supported benefit, a measured tradeoff, or a negative finding can each be a complete answer.
