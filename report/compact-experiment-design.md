# Compact E1–E5 study (frozen after separate pilot)

TODO(team): Preserve the executed compact study design and document its exploratory limits.

The user revised the time constraint on November 14, 2026: all experiments must finish within one or two hours. This design supersedes the run counts and fixture/horizon settings of `experiment-plan.md`, while retaining its questions and policy definitions. It is an **exploratory characterization**, not a powered confirmatory study.

**Executed:** All 81 frozen runs completed in 67.9 minutes. See [the results](compact-study-results.md).

## Configuration

- Three Controllers and three Peers; fixed placement and round-robin selection.
- Nine unique 128 KiB segments, two logical seconds each (18 seconds of simulated media). Exactly two initial verified replicas; 32 MiB quota per Peer.
- Source caps 96/144/192 KiB/s; identical across policies, including the existing 32 KiB shared token burst.
- Two low-load or six high-load viewers, paired arrivals within 0.5 seconds. Burst: two at epoch, two at 4/4.2 seconds, two at 12/12.2 seconds.
- Client window 3, concurrency 2, startup buffer 4 seconds; unchanged eight-second per-download attempt timeout and metadata retry budget.
- First fault at 5 seconds, fixed 50-second observation horizon, 42-second per-session timeout. A failed holder rotates over all three Peers with the three seeds.
- Grace 1/3/8 seconds; main grace 3 seconds. Brief absence 4 seconds. E4 second failure after 4/14/30 seconds, independent of repair success.
- Recovery objective: all affected segments currently hash-valid on two distinct reachable committed holders within 40 seconds after the first fault. Completion-loss margin 5 percentage points; meaningful stall difference 1 second. These are engineering comparison criteria, not statistical guarantees.
- Candidate L retains 250 ms updates, 25% while busy, 100% after two seconds idle, at least 50% when its oldest active committed task is older than 60 seconds. Coordinator repair execution is serial, so there is ordinarily one committed pending task; its creation time is persisted in the command. Age does not include pre-task failure-detection time. The compact horizon does not evaluate the 60-second branch empirically; regression coverage does.

The treatment S uses the shared limiter and a nonbinding repair limiter equal to the shared rate. F25/F50 impose additional caps; no treatment raises aggregate source capacity. Limiter updates have chunk/reservation granularity (32 KiB), so short intervals can include a previously reserved chunk and initial token bursts. These are application payload counters, not NIC bandwidth.

## Matrix and cost

| Experiment | Matrix | Slots | Reused | Unique |
|---|---|---:|---:|---:|
| E1 | N/OFF/S × low/high × 3 paired seeds | 18 | 0 | 18 |
| E2 | S × grace 1/3/8 × brief/persistent × 3 seeds | 18 | 3 | 15 |
| E3 | S/F25/F50/L × high/burst × 3 seeds | 24 | 3 | 21 |
| E4 | S/F50/L × delay 4/14/30 × 2 seeds | 18 | 0 | 18 |
| E5 | active Leader loss / pending-task Leader loss / majority loss × 3 seeds | 9 | 0 | 9 |
| Total | | 87 | 6 | **81** |

E1–E4: 72 unique runs × 50 seconds = 60 minutes of observation. Setup and per-run figure export add roughly 8–15 minutes. E5 is shorter and adds roughly 5–10 minutes; six separate policy pilots take about six minutes. The intended experiment execution total is approximately 80–95 minutes. The serial runner has an additional elapsed-time limit and saves every completed run immediately. It never launches concurrent measured clusters.

The first four-viewer pilot had little added stall, so final high demand was raised to six viewers before final measurements. Pilot runs are archived separately and are not pooled into final estimates. The first baseline pilot overlapped acceptance activity and is instrumentation evidence only.

F50 is prespecified for E4 as the middle cap; F25 remains in E3. Pilot comparison may support retaining that choice but final data never choose the comparator. Three performance seeds cover the three first-failure Peer capacities. E4 has only two paired repetitions per delay and should be shown as counts, not a reliable loss probability.

## Measurement and analysis

The original Go playback engine transfers real bytes over HTTP and simulates consumption. Deterministic video-sized payloads are used because codec content does not affect the application byte limiter. They provide matching hashes and avoid packaging/media-duration differences across treatments. They do not establish decoded-video continuity. Real-media acceptance remains a separate gate; no frontend work or browser claim is included.

Each run gets fresh stores, Raft state and processes. OS page caches are not flushed. Treatment order uses its own fixed random seed, separate from workload seeds. No live demo process is touched. Logs are copied before temporary storage is removed.

Raw artifacts include configuration, source/binary fingerprints, immutable fixture manifests, scheduled viewer/fault traces, actual injections, source-class byte/request counters by process instance, policy history, client events and outcomes, direct metadata probes, tasks, current copy verification, and process logs. Full-source hashes are verified at two-second cadence through Peer inventory and intersected with quorum-backed registrations. This gives interval-censored observed restoration, not an exact completion timestamp. Unreachable/failed observations are labeled unknown rather than destroyed data.

All scheduled viewers remain in completion denominators. Failed sessions retain observed stalls. Metadata probe failures, missing exposure, missing pending-task injections and censored restoration remain visible. Known artificial delays, truncated responses, counter resets and open stalls have harness regression coverage. The independent observer uses monotonic time; client elapsed time is anchored at process launch, and observer receipt times remain archived to assess alignment uncertainty.

Analysis uses per-run/seed means, individual points, paired differences and observed ranges. There is no p99 or noninferiority claim with three seeds. Figures show incomplete recovery explicitly. `aggregate.csv` supplies slide tables; `paired-contrasts.csv` supplies matched E1/E3 effects. Every figure exports SVG and PNG.

## Commands

```sh
make backend
go vet ./...
go test -race ./...
python3 experiments/study_test.py
python3 experiments/repair_playback_study.py --pilot --output-dir results/compact-pilot
python3 experiments/repair_playback_study.py --output-dir results/compact-study
python3 experiments/controller_recovery_study.py --output-dir results/compact-study
python3 experiments/analyze_repair_playback.py results/compact-study
```

Use `--resume` only for the same frozen configuration and binaries. Each completed run has a durable summary. Failed partial runs are preserved for inspection; do not silently overwrite them. The recorded data and final evidence summary determine the verdict, including negative or inconclusive findings.
