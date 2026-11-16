# Cooperative Video Storage Presentation Script

TODO(Tian): Keep the speech aligned with Method through E3 and update slide numbers when the deck order changes.

Speaker: Tian. Covers slides 11 to 19 in the current presentation, from methods through E3. Read the paragraphs aloud; the last paragraph of each slide leads into the next. Allow about 7 to 8 minutes, including pauses and pointing to tables. Section titles are the reference if slide numbers change. This is a full script for this section, not a ten-minute script for the entire team presentation.

## Slide 11 Independent processes on one shared host

Thanks, Haoyu. I'll explain our experimental setup and the results of E1 through E3.

We run three Controllers and three Peers as separate processes on one machine. The workload contains nine segments, each 128 kibibytes and representing two seconds of playback. The three Peers have different upload limits: 96, 144, and 192 kibibytes per second.

Low demand means two viewers; high demand means six. Each client has a three-segment prefetch window, allows two concurrent downloads, and buffers four seconds of content before starting playback.

Downloads use real HTTP traffic, but a headless client simulates playback consumption. We measure download and buffering behavior, rather than browser decoding or rendering. Let's look at how we evaluate that behavior.

## Slide 12 Evaluation Metrics and Failure Timeline

We measure playback completion and average cumulative stall time after the fault. Failed sessions remain included, so a lower stall average does not necessarily mean better playback: a session that fails early stops accumulating stalls.

We also measure replica recovery, repair traffic, and one-copy exposure. Recovery means every affected segment has two verified, reachable copies registered in metadata. One-copy exposure combines how much data has only one reachable copy with how long that condition lasts.

For E1 through E4, the first failure occurs at five seconds, and observation ends at fifty seconds. Our recovery target is forty seconds after the failure.

With those definitions, E1 separates the effect of losing a Peer from the additional effect of repair.

## Slide 13 E1: How Peer Failure and Repair Affect Playback

We compare three conditions. In N, all Peers stay online. In OFF, one Peer fails and repair is disabled. In S, the same failure occurs, but repair shares upload bandwidth with playback.

Comparing OFF with N shows the impact of the failure. Comparing S with OFF shows the additional impact of enabling repair.

We test both low and high demand across three paired failure scenarios, rotating which Peer fails. The next table summarizes those runs.

## Slide 14 Playback Performance Under Different Scenarios

At low demand, all sessions complete with no observed stalls under all three conditions.

Under high demand, completion falls from eighteen out of eighteen without failure to seven out of eighteen after failure, both with and without repair. These totals combine three runs with six viewers each.

Enabling repair raises average observed stalls from 3.24 to 4.46 seconds per viewer, while completion remains unchanged. So Peer failure already causes substantial playback failures, and repair can add further waiting through shared bandwidth. The size of that effect varies across failure scenarios.

E2 asks whether waiting before creating repair work can avoid unnecessary traffic.

## Slide 15 E2: How Long Should Repair Wait After a Peer Failure

We keep policy S and compare grace periods of one, three, and eight seconds. Grace is the waiting period after the system detects insufficient reachable copies.

In the brief-outage case, the failed Peer returns four seconds after failure, with its stored data intact. In the persistent-outage case, it remains offline for the rest of the experiment.

Waiting can avoid copying data that will soon become available again. However, once a repair task has been scheduled, the Peer returning does not automatically cancel that task.

We measure the resulting repair traffic and one-copy exposure.

## Slide 16 Repair Traffic and Risk Under Different Settings

During brief outages, one-second grace produces 0.125 mebibytes of repair traffic, or one segment per run. Three- and eight-second grace avoid that traffic in these traces.

During persistent outages, all settings transfer the same 0.750 mebibytes to replace six missing reachable copies. But one-copy exposure rises from 17.42 to 20.20 mebibyte-seconds as grace increases from one to eight seconds, an increase of about sixteen percent.

These values are averages across three runs. Exposure measures a reduced-redundancy window, not a probability of data loss.

The tradeoff is that longer grace avoids unnecessary copying during short outages, but leaves data with fewer reachable copies for longer during persistent outages. E3 examines how to allocate bandwidth once repair is enabled.

## Slide 17 E3: How Repair Policies Balance Playback and Recovery

E3 compares shared repair, the fixed 25- and 50-percent caps, and the adaptive policy L.

Both arrival patterns have six viewers. With simultaneous arrivals, all six start within the first half-second. With staggered arrivals, pairs start around zero, four, and twelve seconds.

We match the video, initial placement, failed Peer, and failure timing across policies. Each Peer's total upload limit stays unchanged; only the repair allowance changes.

This lets us compare playback outcomes and replica recovery under the same capacity constraints.

## Slide 18 Playback and Recovery Across Repair Policies

The main result is that L does not consistently outperform F50.

With simultaneous arrivals, L lowers average stalls from 4.88 to 4.33 seconds, and both policies restore replicas within the target in all three runs. With staggered arrivals, stalls are nearly identical, while L meets the recovery target in one of three runs, compared with two for F50.

All policies complete the same number of sessions: seven of eighteen with simultaneous arrivals, and nine of eighteen with staggered arrivals. Higher observed stalls therefore do not automatically mean fewer completed sessions.

These are small samples, and recovery is checked periodically, so we should be cautious about differences close to the forty-second threshold. A specific scenario illustrates the underlying tradeoff more clearly.

## Slide 19 Lower Playback Stalls Can Delay Replica Recovery

Here, the slowest Peer fails under simultaneous arrivals, and all six viewers finish under both F50 and L.

L reduces average stalls from 8.53 to 6.88 seconds, saving 1.65 seconds per viewer. However, verified replica recovery takes 36.06 seconds instead of 33.99 seconds, and one-copy exposure also increases.

So L improves playback in this scenario at the cost of slower replica recovery. This is a local tradeoff, not evidence that L is always the better policy.

Across E1 through E3, repair timing and bandwidth allocation affect both playback and the period of reduced redundancy. I'll now hand over to my teammate, who will examine a second Peer failure and Controller recovery in E4 and E5.

## Rehearsal Notes — Not Read Aloud

- Follow the slide titles if the team changes the ordering. The script corresponds to `03-method.tex`, `04-e1.tex`, `05-e2.tex`, and `06-e3.tex`.
- Say “two verified, reachable copies” for replica recovery. This is the storage replication target; it should not be described as Raft majority recovery.
- “Recovered by 40 s” counts experiment runs, not viewers. The clock starts at the first Peer failure.
- Stall summaries include failed sessions and use the post-fault observation period. Avoid treating a lower mean alone as evidence of better playback.
- The E2 brief-outage payload is measured over the run; it does not imply the segment finished transferring during the four-second outage.
- L's 60-second task-age floor did not activate in these runs. Its allowance is a cap, not guaranteed bandwidth.
- Later arrivals competing with repair is a plausible explanation for delayed recovery, not an isolated causal result established by these aggregate tables.
- If the entire presentation must fit ten minutes, shorten this section during team rehearsal. Skip reading configuration values and table cells already visible, while retaining the main findings and their limits.
