# Cooperative Video Storage Presentation Script

TODO(Tian): Keep the speech aligned with Method through E3 and update slide numbers when the deck order changes.

Speaker: Tian. Covers slides 11–19, Method through E3. Read only the **Presentation Script** aloud. Target: **3–4 minutes** at approximately 135–150 words per minute, with brief pauses for tables and transitions. Use slide titles if numbering changes. The **Notes** contain optional explanations and Q&A material, in the same slide order; they are not part of the timed script.

## Presentation Script — Read Aloud

### Slide 11 Independent processes on one shared host

Thanks, Haoyu. We tested three Controllers and three Peers on one machine, using real HTTP downloads with simulated playback. Low demand means two viewers; high demand means six. Each Peer has limited upload capacity shared by playback and repair.

### Slide 12 Evaluation Metrics and Failure Timeline

We measure playback completion, stalls, repair traffic, and replica recovery. Recovery requires two verified, reachable copies of every affected segment. One-copy exposure measures data size times time with only one reachable copy. The first failure occurs at five seconds; the recovery target is forty seconds later.

### Slide 13 E1: How Peer Failure and Repair Affect Playback

E1 separates failure from repair. N has no failure. OFF stops one Peer without repair. S adds repair under the shared bandwidth limit. Comparing OFF with N measures the failure effect; comparing S with OFF measures the additional repair effect.

### Slide 14 Playback Performance Under Different Scenarios

At low demand, every session completes without stalls. At high demand, failure reduces completion from eighteen to seven sessions. Enabling repair keeps completion unchanged but raises average stalls from 3.24 to 4.46 seconds. These totals combine three runs. Repair can therefore add interference when capacity is limited.

### Slide 15 E2: How Long Should Repair Wait After a Peer Failure

E2 asks how long repair should wait. We compare grace periods of one, three, and eight seconds. During brief outages, the Peer returns after four seconds. During persistent outages, it stays offline. Waiting may avoid unnecessary copies, but already scheduled tasks are not automatically cancelled.

### Slide 16 Repair Traffic and Risk Under Different Settings

For brief outages, one-second grace copies one segment per run; longer grace avoids that traffic. For persistent outages, all settings copy the same amount, but one-copy exposure increases by about sixteen percent. So waiting saves traffic during short outages while extending reduced redundancy during persistent ones.

### Slide 17 E3: How Repair Policies Balance Playback and Recovery

E3 compares shared repair, two fixed caps, and adaptive policy L. Six viewers either start together or arrive in three pairs. We keep content, failures, and total upload capacity matched across policies, so only the repair allowance changes.

### Slide 18 Playback and Recovery Across Repair Policies

L shows no consistent advantage over F50. With simultaneous arrivals, L reduces average stalls and both recover in all three runs. With staggered arrivals, stalls are nearly identical, but L meets the recovery target less often. Completion is identical across policies. We need to consider playback and recovery together.

### Slide 19 Lower Playback Stalls Can Delay Replica Recovery

This example shows the tradeoff. When the slowest Peer fails, all six viewers finish. L saves 1.65 seconds of stalls per viewer, but restores replicas 2.07 seconds later. That is a local benefit with a recovery cost. My teammate will now discuss second failures and Controller recovery.

## Notes — Reference Only, Not Read Aloud

### Slide 11 Independent processes on one shared host

Thanks, Haoyu. I'll explain our experimental setup and the results of E1 through E3.

We run three Controllers and three Peers as separate processes on one machine. The workload contains nine segments, each 128 kibibytes and representing two seconds of playback. The three Peers have different upload limits: 96, 144, and 192 kibibytes per second.

Low demand means two viewers; high demand means six. Each client has a three-segment prefetch window, allows two concurrent downloads, and buffers four seconds of content before starting playback.

Downloads use real HTTP traffic, but a headless client simulates playback consumption. We measure download and buffering behavior, rather than browser decoding or rendering. Let's look at how we evaluate that behavior.

**Additional clarification:** Client details: a three-segment prefetch window, at most two concurrent downloads, and four seconds of media buffered before startup. Startup buffering is content duration, not a fixed four-second wait. Each segment represents two seconds. Download attempts time out after eight seconds; sessions after forty-two seconds. Placement is fixed and source selection is round-robin. All processes share one host; these results do not measure browser decoding or rendering.

### Slide 12 Evaluation Metrics and Failure Timeline

We measure playback completion and average cumulative stall time after the fault. Failed sessions remain included, so a lower stall average does not necessarily mean better playback: a session that fails early stops accumulating stalls.

We also measure replica recovery, repair traffic, and one-copy exposure. Recovery means every affected segment has two verified, reachable copies registered in metadata. One-copy exposure combines how much data has only one reachable copy with how long that condition lasts.

For E1 through E4, the first failure occurs at five seconds, and observation ends at fifty seconds. Our recovery target is forty seconds after the failure.

With those definitions, E1 separates the effect of losing a Peer from the additional effect of repair.

**Additional clarification:** Recovery is a storage replication target, not Raft majority recovery. The forty-second target ends around experiment time forty-five seconds; observation ends at fifty seconds. Verification is sampled about every two seconds, so recovery is bracketed by observations. One-copy exposure is expressed in MiB·s and is not a data-loss probability. E5 uses separate timing origins.

### Slide 13 E1: How Peer Failure and Repair Affect Playback

We compare three conditions. In N, all Peers stay online. In OFF, one Peer fails and repair is disabled. In S, the same failure occurs, but repair shares upload bandwidth with playback.

Comparing OFF with N shows the impact of the failure. Comparing S with OFF shows the additional impact of enabling repair.

We test both low and high demand across three paired failure scenarios, rotating which Peer fails. The next table summarizes those runs.

**Additional clarification:** The three paired contexts rotate the failed Peer across slow, medium, and fast upload capacities. They are not three identical-condition repetitions. Peer failures are injected by stopping a process, rather than caused by overload. N uses the same analysis time marker without stopping a Peer.

### Slide 14 Playback Performance Under Different Scenarios

At low demand, all sessions complete with no observed stalls under all three conditions.

Under high demand, completion falls from eighteen out of eighteen without failure to seven out of eighteen after failure, both with and without repair. These totals combine three runs with six viewers each.

Enabling repair raises average observed stalls from 3.24 to 4.46 seconds per viewer, while completion remains unchanged. So Peer failure already causes substantial playback failures, and repair can add further waiting through shared bandwidth. The size of that effect varies across failure scenarios.

E2 asks whether waiting before creating repair work can avoid unnecessary traffic.

**Additional clarification:** Six low-demand sessions means two viewers times three runs; eighteen high-demand sessions means six times three. Stall averages include failed sessions and cover the post-fault observation period. Early failure can reduce observed stalls, so lower stalls alone do not prove better playback. Remaining capacity, finite retries, and timeouts contribute to failures independently of repair.

### Slide 15 E2: How Long Should Repair Wait After a Peer Failure

We keep policy S and compare grace periods of one, three, and eight seconds. Grace is the waiting period after the system detects insufficient reachable copies.

In the brief-outage case, the failed Peer returns four seconds after failure, with its stored data intact. In the persistent-outage case, it remains offline for the rest of the experiment.

Waiting can avoid copying data that will soon become available again. However, once a repair task has been scheduled, the Peer returning does not automatically cancel that task.

We measure the resulting repair traffic and one-copy exposure.

**Additional clarification:** Grace starts when the Coordinator detects insufficient replicas, not necessarily at the instant of process failure. A repair task identifies one segment, a source Peer, and a replacement target. Task creation and actual byte transfer are separate stages. Brief-outage restoration primarily comes from the original Peer returning with its disk intact; persistent-outage restoration requires replacement copies.

### Slide 16 Repair Traffic and Risk Under Different Settings

During brief outages, one-second grace produces 0.125 mebibytes of repair traffic, or one segment per run. Three- and eight-second grace avoid that traffic in these traces.

During persistent outages, all settings transfer the same 0.750 mebibytes to replace six missing reachable copies. But one-copy exposure rises from 17.42 to 20.20 mebibyte-seconds as grace increases from one to eight seconds, an increase of about sixteen percent.

These values are averages across three runs. Exposure measures a reduced-redundancy window, not a probability of data loss.

The tradeoff is that longer grace avoids unnecessary copying during short outages, but leaves data with fewer reachable copies for longer during persistent outages. E3 examines how to allocate bandwidth once repair is enabled.

**Additional clarification:** The 0.125 MiB brief-outage payload is the total over the run, not bytes necessarily transferred during the four-second outage. Tasks were first observed near the original Peer's return and transfer could continue afterward. All 54 brief-outage playback sessions completed. Exposure is calculated from sampled copy states; see the E2 appendix formulas. Repair payload includes retries and excludes HTTP headers/TCP overhead.

### Slide 17 E3: How Repair Policies Balance Playback and Recovery

E3 compares shared repair, the fixed 25- and 50-percent caps, and the adaptive policy L.

Both arrival patterns have six viewers. With simultaneous arrivals, all six start within the first half-second. With staggered arrivals, pairs start around zero, four, and twelve seconds.

We match the video, initial placement, failed Peer, and failure timing across policies. Each Peer's total upload limit stays unchanged; only the repair allowance changes.

This lets us compare playback outcomes and replica recovery under the same capacity constraints.

**Additional clarification:** Simultaneous arrivals occur within 0–0.5 s. Staggered arrivals occur approximately at 0, 0, 4, 4.2, 12, and 12.2 s. Both have six viewers. S has no additional repair cap; F25/F50 cap repair at 25%/50% of source capacity. L uses 25% when foreground downloads are active and 100% after two seconds idle. Its 50% floor for tasks older than sixty seconds did not activate in these runs. Allowances are caps, not reserved or guaranteed bandwidth.

### Slide 18 Playback and Recovery Across Repair Policies

The main result is that L does not consistently outperform F50.

With simultaneous arrivals, L lowers average stalls from 4.88 to 4.33 seconds, and both policies restore replicas within the target in all three runs. With staggered arrivals, stalls are nearly identical, while L meets the recovery target in one of three runs, compared with two for F50.

All policies complete the same number of sessions: seven of eighteen with simultaneous arrivals, and nine of eighteen with staggered arrivals. Higher observed stalls therefore do not automatically mean fewer completed sessions.

These are small samples, and recovery is checked periodically, so we should be cautious about differences close to the forty-second threshold. A specific scenario illustrates the underlying tradeoff more clearly.

**Additional clarification:** For staggered arrivals, L/F50 recovery counts are 1/3 versus 2/3. These count runs, not viewers. One L observation was 40.05 s, close to the threshold relative to two-second sampling, so avoid strong claims from that count difference. Later viewers competing with repair is a plausible mechanism, not an isolated causal conclusion from this table. Higher staggered stalls coexist with more completed sessions than in simultaneous arrivals. F25 had one staggered run unrecovered at the observation horizon.

### Slide 19 Lower Playback Stalls Can Delay Replica Recovery

Here, the slowest Peer fails under simultaneous arrivals, and all six viewers finish under both F50 and L.

L reduces average stalls from 8.53 to 6.88 seconds, saving 1.65 seconds per viewer. However, verified replica recovery takes 36.06 seconds instead of 33.99 seconds, and one-copy exposure also increases.

So L improves playback in this scenario at the cost of slower replica recovery. This is a local tradeoff, not evidence that L is always the better policy.

Across E1 through E3, repair timing and bandwidth allocation affect both playback and the period of reduced redundancy. I'll now hand over to my teammate, who will examine a second Peer failure and Controller recovery in E4 and E5.

**Additional clarification:** The local example is one paired context under simultaneous arrivals. One-copy exposure rises from 18.82 to 20.38 MiB·s. Do not generalize this example into a consistent policy advantage. The final handoff connects this reduced-redundancy window to E4, followed by Controller failures in E5.
