# Literature Review and Recommended Research Focus

Date: November 14, 2026  
Purpose: refine the research direction in `research-and-presentation-plan.md` after a targeted review of prior work.

TODO(team): Run the contention pilot, choose the final policy scope, and collect evidence before treating the hypotheses below as findings.

**Recommendation:** Focus on the tradeoff between playback continuity and replica recovery under shared bandwidth constraints. Use Peer failures to create the resource conflict. Treat Controller failover, integrity, and publication as supporting correctness results. Avoid claiming unconditional smooth playback or novelty merely from combining familiar techniques.

This is a targeted prior-art review, not an exhaustive novelty search. I reviewed accessible model/design and evaluation passages from the five full papers below. The sixth entry was available through its university-hosted abstract and bibliographic record; it is explicitly not a full-text review. No claim of being the first system to consider repair interference follows from this review.

## 1. Relevant papers and what they change

### 1.1 CoolStreaming/DONet — Zhang, Liu, Li, and Yum; INFOCOM 2005

[Author-hosted full paper](https://www.cs.sfu.ca/~jcliu/Papers/47_01.pdf). Relevant passages: Section III-C, scheduling; III-D, failure recovery; V-B/V-C, continuity under stable/dynamic environments.

The design uses data-availability exchanges, supplier bandwidth, and segment deadlines to schedule downloads. It reschedules against remaining suppliers after departures and evaluates continuity under churn.

**Implication for us:** Deadline-aware source selection and sustaining playback despite departing suppliers are established. Our persistent video storage and replicated metadata differ from its live overlay, but that difference alone does not establish a new scheduling contribution.

### 1.2 Optimal Content Placement for Peer-to-Peer Video-on-Demand Systems — Tan and Massoulie

[Full paper, arXiv version revised August 2011](https://arxiv.org/pdf/1004.4709). Relevant passages: system model and distinction between Distributed Server Networks/Pure P2P Networks; Section IV, placement objective and assumptions.

The paper studies storage placement to use upload resources efficiently and derives results under an explicit admission/loss-network model. It distinguishes external viewers served by storage boxes from users that also contribute resources.

**Implication for us:** Our dedicated storage Peers and separate viewers resemble the distributed-server setting more closely than a pure P2P overlay. Placement is established research, and analytical optimality under that paper's model does not transfer automatically to our finite-timeout, segment-by-segment implementation.

### 1.3 Statistical Modeling and Analysis of P2P Replication to Support VoD Service — Zhou, Fu, and Chiu; INFOCOM 2011

[Author-hosted full paper](https://home.ie.cuhk.edu.hk/~dmchiu/p2pvod_infocom11.pdf). Relevant passages: Sections II–III, fair-sharing model and replication; IV-B, heterogeneous peers; V–VI, adaptation and evaluation design.

The paper relates storage allocation and upload supply to VoD demand and origin-server load. Its replication reasoning includes load balancing and heterogeneous capacities under a specified sharing model.

**Implication for us:** Ranking nodes by bandwidth is not a new research question. We need controlled finite-load measurements and must explain our different objective: playback disruption and recovery, rather than origin-server offload under a fluid model.

### 1.4 Total Recall: System Support for Automated Availability Management — Bhagwan et al.; NSDI 2004

[USENIX full text](https://www.usenix.org/legacy/publications/library/proceedings/nsdi04/tech/full_papers/bhagwan/bhagwan_html/index.html). Relevant passages: Section 3.3, eager/lazy repair; Section 6, delivered availability and repair overhead.

The work explicitly studies redundancy, repair timing, transient departures, availability, and bandwidth cost. Delaying repair can avoid work caused by transient absence, while different redundancy arrangements change the available tradeoffs.

**Implication for us:** Delayed repair and repair/storage/bandwidth tradeoffs are not novel. Our two-copy prototype must measure the duration of reduced redundancy; it cannot borrow the availability claims of a system with different redundancy and prediction assumptions.

### 1.5 f4: Facebook's Warm BLOB Storage System — Muralidhar et al.; OSDI 2014

[USENIX full paper](https://www.usenix.org/system/files/conference/osdi14/osdi14-paper-muralidhar.pdf). Relevant passage: Section 5.3, backoff nodes and rebuilder nodes, especially proceedings pages 389–390.

The system separates online requested-object reconstruction from background block reconstruction. Rebuilders limit their activity because reconstruction imposes storage/network load and can interfere with user reads.

**Implication for us:** Even protecting foreground reads from repair is prior art. Our opportunity is a narrower implementation and evaluation of a streaming-specific stall/recovery tradeoff; simple repair throttling must be attributed as an established baseline, not claimed as a new principle.

### 1.6 Additional close prior art — abstract-level review only

[A Unifying Model and Analysis of P2P VoD Replication and Scheduling — Zhou, Fu, and Chiu; INFOCOM 2012](https://researchers.mq.edu.au/en/publications/a-unifying-model-and-analysis-of-p2p-vod-replication-and-scheduli/).

The authors' university record describes how scheduling assumptions change preferred replication strategies, and a model parameterized by the number of Peers serving a request. I did not obtain a readable full text in this review.

**Implication for us:** The broad interaction of replication and scheduling has already been investigated. Obtain and examine the full paper before making a close comparative claim about our placement/selection study.

## 2. How the research focus should change

| Candidate focus | Assessment | Role in our project |
|---|---|---|
| Handle crashes and elect a new Leader | Essential implementation work; standard consensus/failover behavior is not itself new | Correctness and engineering evidence |
| Guarantee smooth playback | Too broad without a capacity model, admissible demand, buffering assumptions, and bounded failures | Replace with measurable continuity targets in a stated regime |
| Combine adaptive placement and deadline selection | Feasible existing ablation, but substantial close prior art | Secondary evaluation or a limited empirical-study fallback |
| Balance repair with current playback under a fixed resource budget | Concrete, measurable conflict already present in our code; still has prior art | Recommended central question |

The recommendation is an inference from the literature and our implementation, not a claim that this last topic is unexplored. A course-level contribution could be a precisely defined policy plus rigorous characterization of where it improves on strong simple baselines. Whether that meets the instructor's exact novelty threshold is not established merely by renaming the question.

## 3. Recommended title, question, and hypotheses

**Title:** Balancing Playback Continuity and Replica Recovery in Cooperative Video Storage

**Primary research question:** After a storage Peer fails, how should the system allocate limited upload bandwidth between foreground video delivery and restoring missing replicas, and when does a load-aware repair policy improve that tradeoff over fixed repair policies?

**Supporting question:** Can the application preserve verified data and recover committed metadata/repair work across a Controller Leader crash, within its quorum and retry assumptions?

Proposed, falsifiable hypotheses:

- **H1 — Interference:** Under constrained surviving-source bandwidth, repair can increase playback stalls relative to an otherwise matched failed-Peer run without repair. Under spare capacity, that effect can be small or absent.
- **H2 — Timing:** A longer grace period can avoid some repair traffic for brief departures but increases exposure to reduced redundancy for persistent departures. Measure actual bytes; cancellation and in-flight tasks can weaken the expected savings.
- **H3 — Policy benefit:** A bounded load-aware repair budget can reduce playback disruption relative to unclassified shared service, while meeting a declared redundancy-restoration target in feasible tested workloads. It should be compared with tuned fixed caps as well as the current implementation.

H3 requires implementation beyond the current code. Failure to outperform a tuned fixed policy is a legitimate negative result. Do not report a benefit before measuring it.

## 4. Why this fits the existing system

The current Peer GET handler uses a shared upload limiter. Both foreground segment downloads and Controller-initiated repair downloads pass through it. The repair coordinator has a configurable grace interval and serial task execution, but no explicit foreground/repair service classification or adaptive repair budget.

Consequently, the research conflict already exists: copying from a surviving holder consumes upload capacity that could serve current playback. The implementation also has the persistent task state needed to discuss recovery across Leader changes.

**Illustration, not a measurement:** If a surviving source can send 1 MiB/s and viewers need 0.8 MiB/s from that source, little headroom remains for repair. A competing repair can exhaust the buffer. Conversely, delaying the repair leaves the segment dependent on fewer holders. Actual traffic scheduling need not divide bandwidth equally, so this arithmetic is only motivation.

Use two initial replicas and an eligible third Peer in the principal fault experiment. An optional third copy present before failure changes the repair requirement. Measure source-specific demand and headroom; spare aggregate bandwidth elsewhere may be unusable for the affected segment.

## 5. A feasible experiment and implementation progression

### Stage A — Test whether the conflict is material

Hold placement, selection, client settings, initial replica bytes, and failure schedule fixed. Compare:

1. No fault, ordinary repair setting: normal reference.
2. The same Peer failure with repair disabled: foreground failover reference and redundancy-exposure extreme, not a deployable recovery policy.
3. The same failure with the current repair behavior.
4. The same failure with several fixed grace intervals.

Cross low/high foreground demand with brief/persistent Peer absence. Use sessions and an observation horizon long enough for repair to overlap playback. Rotate the failed node and keep workload seeds paired. Verify that the intended holder and source were affected and that a third target had capacity.

This stage can mostly use existing code and script extensions. If interference is negligible across realistic constrained conditions, report that result and avoid building an elaborate controller without evidence of a problem.

### Stage B — Add a small, explicit repair-budget policy

For a stronger design contribution, add foreground/repair classification and per-class counters at the source Peer while retaining one authoritative total upload cap. Compare:

| Policy | Behavior | Implementation status |
|---|---|---|
| Current shared service | Repair and foreground use the same limiter without a separate repair budget | Exists |
| Fixed repair cap | Repair has an additional static cap; all traffic still obeys the total cap | Proposed baseline |
| Bounded load-aware repair | Adjust repair allowance using observed foreground demand; retain a minimum service allowance or explicit escalation rule to avoid indefinite postponement | Proposed treatment |

A cap is not a bandwidth reservation or a strict-priority guarantee. If the treatment requires foreground reservations, implement an actual scheduler and describe its semantics. A policy observing only traffic load should be called load-aware. Calling it playback-buffer-aware requires new, trustworthy buffer/deadline telemetry, which the current Controller does not have for all clients.

Declare the maximum acceptable recovery interval and minimum repair progress before the final study. Check that object sizes, repair budgets, and finite copy timeouts are compatible; an overly low cap can merely cause repeated failed repairs. Validate byte integrity, task idempotency, and failover behavior after policy changes.

### Stage C — Test the limits of the compromise

After one holder fails, stop the remaining original holder at several declared delays in separate fault runs. Check whether a repaired valid copy is already usable. This turns reduced-redundancy exposure into an observed consequence under a specified second-fault schedule.

It does not estimate real-world data-loss probability. Distinguish temporarily stopped nodes with intact disks from permanent destruction of copies. The previously identified unregistered-repair-target defect now has a reconciliation fix and unit/system regressions. Retain that evidence gate when changing the repair policy.

## 6. Metrics and the main result figure

Report all of the following together:

- Completion and failures for every scheduled session.
- Playback stalls and startup; keep headless simulation separate from actual browser metrics.
- Time to verified alternate delivery after the first fault.
- Time to two reachable, valid, committed copies, across all affected segments.
- Time/byte-weighted duration with only one valid reachable copy; record zero-copy intervals separately. These measure exposure, not a calibrated loss probability.
- Actual repair bytes, total Peer transfer bytes, retries, and unsuccessful copies.
- Outcome of the separately scheduled second fault.

**Main figure:** plot redundancy-restoration time against playback disruption for each repair policy, with completion and unrecovered counts shown alongside. Look for policies that reduce one cost without worsening the other, or explicitly quantify the cost of the chosen compromise. If a policy never restores redundancy in the observation window, display it as censored/unrecovered rather than dropping it.

Grace-period sweeps explain when repair begins; rate-budget sweeps explain resource contention after it begins. Do not conflate these two controls.

Keep placement and selection fixed in the primary experiment. After establishing the repair effect, repeat a small subset with the existing four-policy ablation if it answers a specific follow-up. Avoid a large uncontrolled matrix that obscures the central result.

## 7. What the report may conclude

The result should identify an operating region and a tradeoff, for example:

> Under the tested bandwidth and demand conditions, policy X changed playback disruption by the measured amount while restoring verified redundancy within the measured time. Under tighter capacity or a second failure before repair completed, that objective was not met.

Do not insert numbers until collected. A useful conclusion may also be that fixed caps perform as well as the adaptive policy, or that limited capacity makes the desired playback/recovery targets mutually infeasible.

Correctness tests for Leader failure, quorum loss, hashes, and interrupted uploads remain necessary. In a ten-minute presentation, use them to establish the reliability substrate, then devote the main experimental argument to the playback/repair tradeoff.

If no new policy work is feasible, retain the existing placement/selection comparisons and add the grace/load/failure study as an empirical characterization. Label that contribution honestly. It is a more modest novelty case than a demonstrated new policy, and no amount of presentation wording can substitute for an actual research distinction.
