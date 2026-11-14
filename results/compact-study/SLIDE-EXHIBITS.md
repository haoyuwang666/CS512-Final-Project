# Slide exhibits: compact E1–E5 study

**81 frozen study runs; 67.9 minutes; all nine controller cases passed.** PNG images can be inserted directly into slides; the matching SVG files preserve vector quality. The ZIP bundle contains all figures, tables, findings, configuration and audit summaries. Raw logs and source snapshots remain in the full study directory.

Use “real HTTP transfer with simulated media consumption” in the slide methodology. There are three paired workload/failure contexts (one per failed capacity); E4 has two. These are exploratory measurements, not population confidence intervals or browser playback results.

## Slide 7: repair interference and grace

Suggested title: **Repair can add stalls under contention; grace avoids brief-outage work.**

Use [E1 paired effects](figures/E1-paired-effects.png) beside [E1 completion/stalls](figures/E1-contention.png). Blue points are paired context differences; orange lines are means. Identical low-load points overlap or are horizontally offset for visibility.

Suggested caption: “At low demand, all viewers completed with zero observed post-fault stalls. In the feasible six-viewer/slowest-Peer-failure context, shared repair added 3.46 s of post-fault stall per viewer relative to the same failure without repair, with all viewers finishing.”

Keep completion visible: high-load N completed 18/18 viewers; OFF and S each completed 7/18. Early failures can make observed stall time smaller, so a negative stall difference alone is not a benefit.

Use [E2 grace](figures/E2-grace.png) to show traffic and exposure together.

Suggested caption: “For four-second absences, one-second grace copied 128 KiB per run; three- and eight-second grace copied nothing. Persistent failures needed 0.75 MiB at every grace; sampled one-copy exposure increased from 17.42 to 20.20 MiB·s between one- and eight-second grace.”

## Slide 8: policy tradeoffs and second-failure boundary

Suggested title: **Load awareness showed a local compromise, not a consistent win over F50.**

Use [E3 policy table](figures/E3-policy-table.png) with [E3 tradeoff](figures/E3-tradeoff.png). If space is tight, use the table as the main exhibit and the tradeoff figure as backup. [Source-specific steady](figures/E3-high-by-failed-peer.png) and [burst](figures/E3-burst-by-failed-peer.png) tables expose the completion differences behind the averages.

Suggested caption: “All four policies completed 7/18 steady and 9/18 burst viewers. L−F50 mean observed stall was −0.55 s/viewer under steady demand and +0.01 s under bursts. One slow-source context saved 1.65 s of stall but restored redundancy 2.07 s later. No consistent overall advantage over F50 was established.”

Do not rank F25 using its 37.08 s burst median alone: that median excludes one unrecovered run. Censored runs have no fabricated restoration time. Target counts use first verified observations; the approximately two-second cadence matters for near-40-second outcomes.

Optional mechanism exhibit: [L budget trace](figures/L-budget-trace.png). Caption: “Source-local foreground activity changes the additional repair allowance; all transfers still obey the common upload cap. The example is high-load seed 1103, with 0/6 completed viewers: its idle period follows failures, not successful playback. The 60-second age floor was regression-tested but not activated in these compact performance runs.”

For the availability boundary use [E4 direct-delivery counts](figures/E4-target-delivery-counts.png).

Suggested caption: “After both original holders stopped, the selected late-queue replacement was unavailable in all 18 probes at 4/14/30 s delays. Some buffered viewers still finished. These delays expose the reduced-redundancy boundary but did not discriminate policy availability benefits.”

This is not permanent data loss: stopped processes retained disks. Do not interpret two trials per cell as a real-world loss probability.

## Slide 9: application recovery and safety

Suggested title: **Application recovery and task continuity held in all nine tested controller scenarios.**

Use [E5 safety table](figures/E5-safety-table.png) with [E5 annotated timeline](figures/E5-annotated-timeline.png).

Suggested caption: “Fresh quorum-backed metadata recovered in a median 1.52 s during delivery and 1.60 s during pending repair. Pending task identity, creation time and target bytes survived Leader loss. All nine isolated valid writes were rejected; after quorum returned, fresh metadata recovered in a median 1.04 s and the original operation IDs reconciled.”

E5a/b times start at Leader stop; the E5c safety-table time starts at quorum restoration. The general E5 distribution retains fault-relative time, including deliberate isolation. Warm and cold uncached sessions both completed in all three active-delivery cases.

These are local process-crash/quorum tests, not sustained network partitions, power loss, Byzantine behavior, or a universal availability guarantee.

## Tables, data and reproduction

- [Research findings](RESEARCH-FINDINGS.md): complete scoped answers and limitations.
- [Aggregate CSV](aggregate.csv): grouped values for rebuilding slide tables.
- [Paired CSV](paired-contrasts.csv): every matched effect and range.
- [Run CSV](summary.csv) and [viewer CSV](sessions.csv): all 72 performance runs and 396 scheduled viewer outcomes.
- [Controller CSV](controller-summary.csv): all nine controller cases.
- [Measurement CSV](measurement-audit.csv): first task/byte observations, restoration mechanism, counter timing and policy checks.
- [Audit JSON](analysis-audit.json), [execution JSON](execution.json), [frozen configuration](study-config.json), and [provenance](environment-and-provenance.json).

Rebuild final exports without rerunning experiments:

```sh
python3 experiments/finalize_compact_study.py results/compact-study
```

The full dataset keeps raw per-run journals, immutable fixture hashes, original observed summary rows, process logs, source snapshot, binary fingerprints and final analysis source. Raw journals were not changed during analysis. Source payload counters have a pre-crash snapshot; a small in-flight amount between that snapshot and process exit may be unobserved. Physical TCP/NIC overhead is outside their scope.
