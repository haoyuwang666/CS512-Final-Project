# Ten-Minute PowerPoint Outline

Working title: **Balancing Playback Continuity and Replica Recovery in Cooperative Video Storage**

TODO(team): Fill the result slides from archived experiment outputs, then rehearse the exact ten-minute schedule.

This is a slide specification and speaking guide, not a finished deck or a record of measured findings. It follows the research focus in [the literature review](literature-review-and-research-focus.md). The companion [experiment plan](experiment-plan.md) defines experiment IDs E0–E5 and all proposed policies. Existing system behavior and proposed repair controls must remain visibly distinct until implementation and validation finish.

## 1. Presentation argument

**Main question:** How should repair share limited Peer upload bandwidth with video delivery after a Peer fails?

**Research questions:**

- **RQ-A:** When does replica repair interfere with playback, and how does repair timing affect temporary versus persistent failures?
- **RQ-B:** Does a load-aware repair budget improve playback and redundancy recovery compared with simple fixed budgets?
- **RQ-C:** Which correctness properties and recovery behavior does the application retain during Peer and Controller failures?

The presentation should establish the resource conflict, explain the system that lets us study it, demonstrate the working application, and then answer the questions with evidence. Keep the normal-path live demo to two minutes. Use prepared measurements for fault cases.

## 2. Timing and page order

Ten slides total, including the opening slide and demo holding slide. Q&A is assumed to follow the ten-minute presentation.

| Slide | Time | Duration | Title | Main evidence |
|---|---|---|---|---|
| 1 | 0:00–0:40 | 40 s | Playback and Replica Recovery | Resource conflict illustration |
| 2 | 0:40–1:20 | 40 s | Research Question and Prior Work | Narrow contribution and two prior-work anchors |
| 3 | 1:20–2:20 | 60 s | System Architecture | Control/data paths and implementation responsibilities |
| 4 | 2:20–3:10 | 50 s | Publication and Peer Recovery | Protocol sequence and shared upload budget |
| 5 | 3:10–5:10 | 120 s | Live System Demonstration | Cluster configuration, upload, packaging, playback |
| 6 | 5:10–6:00 | 50 s | Experiment Design | Repair policies, controlled variables, metrics |
| 7 | 6:00–7:20 | 80 s | Repair Interference and Timing | E1 and E2 findings |
| 8 | 7:20–8:20 | 60 s | Playback and Recovery Tradeoff | E3 frontier and E4 second-fault consequence |
| 9 | 8:20–9:30 | 70 s | Controller Recovery and Safety | E5 results and failure boundaries |
| 10 | 9:30–10:00 | 30 s | Conclusions and Limits | Direct answers to RQ-A/B/C |

## 3. Slide-by-slide specification

### Slide 1: Playback and Replica Recovery

**Put on the slide**

- Project title, team names, course.
- One simple illustration: a failed Peer, a surviving holder, a viewer, and a replacement Peer. Label the surviving holder's upload capacity and the competing foreground/repair transfers.
- One sentence: “Restoring a replica consumes bandwidth that current viewers also need.”

**Talk about**

“Our project stores video segments across Peers and uses a replicated metadata service to coordinate them. When a Peer fails, another copy can serve the video. But restoring the missing copy also reads from the surviving storage nodes. Under limited upload capacity, recovery itself can interfere with playback. We investigate that tradeoff.”

**Purpose:** Establish importance through a concrete conflict. No performance numbers are needed on this slide. If using numerical bandwidths, label them as an illustration.

### Slide 2: Research Question and Prior Work

**Put on the slide**

Main question in large text:

> How should repair share limited upload bandwidth with ongoing playback after a Peer fails?

Below it, two short prior-work notes:

- CoolStreaming studies deadline/bandwidth scheduling and recovery after departures.
- Total Recall and f4 establish repair timing and protecting online requests from maintenance as prior concerns.

Contribution line: “A defined repair policy and controlled measurements of playback, redundancy recovery, and transfer cost in our prototype.”

**Talk about**

“We do not claim to invent replication, Raft, or deadline scheduling. Prior work already studies these mechanisms, and storage systems already manage repair overhead. Our scoped question is how repair controls affect video playback and the time spent with reduced redundancy under the same resource budget. We evaluate our policy against simple fixed controls.”

**Source notes for the PPT speaker notes:** [CoolStreaming, Sections III-C/III-D](https://www.cs.sfu.ca/~jcliu/Papers/47_01.pdf), [Total Recall, Section 3.3](https://www.usenix.org/legacy/publications/library/proceedings/nsdi04/tech/full_papers/bhagwan/bhagwan_html/index.html), [f4, Section 5.3](https://www.usenix.org/system/files/conference/osdi14/osdi14-paper-muralidhar.pdf).

**Purpose:** State the novelty argument honestly. Do not use “first,” “optimal,” or “guaranteed” without supporting evidence.

### Slide 3: System Architecture

**Put on the slide**

One architecture diagram with clearly labeled paths:

- Three Controllers with one Raft Leader. Label manifests, committed receipts, publication state, and persistent tasks.
- Three Peers with independent persistent indexes/blobs. Label direct ingestion, playback reads, and repair copying.
- Browser/Gateway path and headless experimental-client path. Keep the Gateway outside the Raft cluster.

Small implementation footer: “Go HTTP services, HashiCorp Raft, BoltDB/snapshots, SHA-256 object validation, FFmpeg packaging.”

**Talk about**

“Controllers agree on small metadata decisions through Raft. Peers store and transfer the video bytes outside the Raft log. Our application implements the publication protocol, durable store, task lifecycle, and playback engine. The browser obtains verified segments through the Gateway. The experiment client performs real downloads and simulates media consumption so that we can repeat controlled workloads.”

Explain that HashiCorp supplies the consensus implementation. Attribute its algorithmic foundation in the speaker notes to the [Raft paper](https://raft.github.io/raft.pdf).

**Purpose:** Show distributed-system difficulty and explain why control/data separation fits the research question. Do not present the single Gateway as highly available.

### Slide 4: Publication and Peer Recovery

**Put on the slide**

Two short sequences, preferably revealed in order:

1. Upload: DRAFT, validated durable Peer receipts, two distinct confirmed holders per segment, READY.
2. Peer failure: alternate-source delivery, grace interval and verified shortage, persistent repair task, verified replacement registration.

Highlight one annotation: “Delivery recovery and redundancy recovery have different completion times.”

**Talk about**

“A video becomes publishable only after the required distinct replicas have been confirmed. During playback, the downloader checks size and hash before consuming a segment. If a Peer disappears, the downloader tries another source. The background coordinator separately restores redundancy through a persistent task. The experiment measures both milestones because playback can resume before repair finishes.”

Mention the shared source upload limiter. Clarify that receipt history is not proof of present availability and that repair requires a valid source, a target with capacity, and Controller quorum.

**Purpose:** Connect correctness mechanisms to the measured resource conflict.

### Slide 5: Live System Demonstration

**Put on the slide**

Before switching to the browser, show one short checklist: “Configure cluster, upload and package, publish, play.” Keep this slide available as a return point. No code screenshots are necessary.

**Do and say during the two-minute slot**

| Within demo | Action | Narration |
|---|---|---|
| 0:00–0:25 | Select 3 Peers and 3 Controllers, start, show readiness | “These are independent service processes. Choosing the size configures a deployment.” |
| 0:25–1:15 | Upload a rehearsed short clip and show actual processing stages | “The pipeline packages the media, uploads segments to distinct Peers, and confirms them before publication.” |
| 1:15–1:45 | Play the new video and point to verified source information | “The application delivers and decodes real media. This short demonstration establishes functionality.” |
| 1:45–2:00 | Return to slides | “We use automated workloads to measure failures and the repair tradeoff repeatedly.” |

The UI supports topology selection before startup, not live membership reconfiguration. Do not make a comparative performance claim from this demonstration.

**Preparation:** Rehearse the exact file and target browser. Keep a recording of the same workflow and an already imported clip. If processing overruns, explicitly switch to one of these at the cutoff. The backend media check now validates stored fragment timing and decoding; actual browser continuity still needs target-browser evidence. Do not add an unplanned live crash that takes time away from results.

### Slide 6: Experiment Design

**Put on the slide**

Compact policy table:

| Policy | Repair behavior |
|---|---|
| Shared | Existing common limiter |
| Fixed | Additional repair cap at 25% or 50% of source capacity |
| Load-aware | Busy/idle budget with an age escalation rule |

Below the table: “Same initial replicas, workload/fault trace, source limits, placement, client settings, and observation horizon.”

Name the measured outcomes: completion, playback stalls, verified redundancy-restoration time, repair bytes. Explain the no-repair diagnostic reference verbally.

**Talk about**

“We vary repair behavior while keeping the rest of the system fixed. First we establish whether repair interference exists. Then we vary repair timing for short and persistent failures. Finally we compare the proposed budget against fixed caps under steady and changing demand. We keep failed sessions in the completion denominator.”

**Status:** Fixed and load-aware budgets are proposed until implemented. Before that point, label this slide “Planned experiment design” and use the empirical-only conclusion described below. Thresholds and parameters must match the archived final configuration.

### Slide 7: Repair Interference and Timing

**Put on the slide**

Two measured exhibits:

1. E1: paired change in completed-session stall time, with completion shown alongside, for a failed-Peer run with repair enabled versus disabled. Separate low and higher load.
2. E2: grace interval against repair bytes and reduced-redundancy exposure, with brief/persistent failures distinguished.

Use a takeaway title only after results support it. Otherwise retain the neutral title above. Put run counts, uncertainty method, units, and experiment IDs in small source notes.

**Talk about**

Answer RQ-A in this order: whether the fault actually caused replica shortage, whether repair overlapped playback, how much repair changed stalls/completion, and how timing changed the transient-failure cost. Explain any null result. A disabled-repair run is a diagnostic extreme with prolonged reduced redundancy, not a complete solution.

**Result sentence to complete:** “At [load], repair changed [stall metric] by [paired difference and uncertainty], with [completion outcome]. For [failure duration], increasing grace changed [bytes/exposure] by [measured amount].”

Do not insert illustrative numbers into a result plot.

### Slide 8: Playback and Recovery Tradeoff

**Put on the slide**

- E3's main comparison: verified redundancy-restoration time on the horizontal axis and completed-session post-fault stalls on the vertical axis. Show individual runs or clearly defined summaries, uncertainty, and a companion completion/recovery-within-target table.
- One small E4 result: after a second holder stops at a specified delay, whether the new target can supply a preselected segment with the correct hash.

**Talk about**

“A lower repair rate can protect current playback while leaving fewer available copies for longer. We compare our load-aware rule with both fixed rates rather than only an aggressive baseline. Here is whether it improves the compromise, merely moves along the same frontier, or performs worse. The second-fault experiment makes the cost of delayed repair observable.”

**Result sentence to complete:** “For [workload], the load-aware policy [improved/did not improve] the tradeoff relative to [fixed policy], with [completion and recovery-target results]. After the second fault at [delay], [hash-verified outcome].”

Censored/unrecovered runs must remain visible. A second process crash demonstrates availability under that schedule, not a probability of permanent data loss.

### Slide 9: Controller Recovery and Safety

**Put on the slide**

One E5 timeline: Leader crash, observed replacement Leader, first successful fresh quorum-backed application probe. Next to it, a short evidence table:

| Check | Result to fill |
|---|---|
| Pending repair survives Leader change | Same task ID, validated completion, passes/attempts |
| Majority loss | No successful new metadata acknowledgment during stable isolation |
| Integrity and partial upload | Hash/rejection checks, passes/attempts |

Boundary line: “Requires a communicating Controller majority and a valid surviving data source.”

**Talk about**

“Raft protects committed metadata. Our application persists repair tasks so a replacement Leader can continue the work. We measure usable metadata recovery separately from merely observing a Leader. Warm playback can continue from cached metadata, so it does not alone prove Controller recovery. Without a majority, the service refuses new quorum-backed progress. If all data copies disappear, metadata cannot recreate the video.”

Mention the single Gateway and local-host failure-domain limits. Show actual recovery times and failed attempts. If a known application defect remains unresolved, state its effect on the claim rather than reporting a general pass.

### Slide 10: Conclusions and Limits

**Put on the slide**

One measured answer per research question:

- RQ-A: the conditions under which repair affects playback and timing changes cost.
- RQ-B: the policy comparison and its recovery/completion tradeoff.
- RQ-C: the verified fault boundary and observed application recovery.

One limitation line appropriate to the final study, such as headless performance scope, local-process topology, or an unmet recovery target.

**Talk about**

“We implemented the storage and coordination protocol and used it to measure a specific recovery tradeoff. Our results show [supported finding] under [tested conditions], at the cost of [measured cost]. [Named condition] remains a limit. The contribution is the application design and evidence about the repair policy in this setting.”

If the adaptive policy is unimplemented, replace RQ-B with the measured fixed-grace/fixed-policy characterization. Explicitly state that no adaptive-policy result exists. An unsupported or inconclusive hypothesis is an acceptable conclusion if the experiment and limits are clear.

## 4. Backup pages for Q&A

Keep these outside the ten-slide timed sequence:

| Backup page | Content |
|---|---|
| Policy definition | Exact load-aware rule, parameters, total-cap preservation, cap versus reservation distinction |
| Workload and measurement | Segment size/duration, demand, seeds, injection rule, timeouts, definitions, raw-run provenance |
| Prior-work comparison | CoolStreaming, placement/scheduling studies, Total Recall, f4, and the specific claim we make |
| Publication and repair state | Commands, pending/completed transitions, operation IDs, and unknown outcomes |
| Full correctness matrix | Corruption, partial upload, quota exhaustion, restarts, majority loss, and unsupported fault classes |
| Negative results and limitations | Timeout regressions, missing injections, censored recovery, browser/headless distinction, and remaining defects |

## 5. Evidence and grading map

| Criterion | Timed pages | What must be supported |
|---|---|---|
| Importance | 1, 7–8 | The resource conflict has observable consequences |
| Novelty | 2, 6, 8 | Precise policy and scoped comparison against prior ideas and credible baselines |
| Difficulty | 3–5, 9 | Substantial application protocols and working multi-process implementation |
| Completeness | 7–10 | Direct answers with costs, failures, uncertainty, and limits |

The outline can guide PPT production now. Quantitative slide titles and conclusions must wait for the experiments. The companion experiment plan identifies exactly which outputs each result page needs.
