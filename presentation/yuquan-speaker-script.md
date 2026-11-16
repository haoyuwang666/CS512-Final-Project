# Cooperative Video Storage Presentation Script

TODO(Yuquan): Keep the speech aligned with slides 19 to 23 and regenerate the PDF export after edits.

Speaker: Yuquan Hu. Covers slides 19 to 23: E4, E5, and Conclusion. Target: about 3 minutes, including short pauses and slide changes. Read only the paragraphs below aloud; timing and preparation notes are not part of the speech.

## Slide 19 Can the replacement serve after both holders stop?

The previous experiments compared playback and repair performance. E4 asks what happens if another Peer fails before repair finishes.

We stop the first original holder, wait four, fourteen, or thirty seconds, and then stop the second original holder. These delays are fixed; we do not wait for repair to succeed.

We then request one selected segment near the end of the repair queue. Success requires both correct bytes and committed registration.

## Slide 20 The selected replacement was not ready in time

We compare S, F50, and L, with two paired contexts for each delay. All eighteen probes were unavailable under this success condition.

This shows a limit of unfinished repair, rather than a policy winner. It also does not establish permanent data loss: the stopped Peers kept their disks. A viewer finishing from its buffer would not prove that fresh delivery was available.

Next, we move from Peer failures to Controller failures.

## Slide 21 E5a and E5b: application recovery after Leader loss

For E5, we require a successful application request, not just a new Leader.

In E5a, we stop the Leader during delivery. Fresh metadata requests recover, and both warm and cold sessions complete in all three trials. Median recovery is one point five two seconds.

In E5b, we stop the Leader with a committed repair task pending. The new Leader preserves the task ID and creation time, and the replacement is verified and registered. All three trials pass, with median recovery of one point six seconds.

## Slide 22 E5c: without a majority, writes were refused

E5c removes the Controller majority. During stable majority loss, all nine valid write attempts return HTTP five-oh-three, with no success acknowledgments.

After quorum restoration, the baseline state is retained and the original operation IDs are reconciled. All three trials pass, with median recovery of one point zero four seconds. This timer starts when quorum is restored, excluding the deliberate outage.

## Slide 23 Conclusions: performance and recovery are separate

To conclude, repair competes with playback for bandwidth. Waiting can avoid unnecessary repair, but extends the time with reduced redundancy. L shows a local tradeoff, without a consistent advantage over F50.

Controller recovery and task continuity hold in our tested cases, while E4 shows the risk of repair still being unfinished.

These are short, single-host experiments with simulated consumption. Next steps are multi-host browser measurements and an E4 control after verified repair completion. Thank you.

## Rehearsal timing (not spoken)

| Slide | Time | Cumulative |
|---|---|---|
| 19: E4 design | 0:00?0:35 | 0:35 |
| 20: E4 result | 0:35?1:10 | 1:10 |
| 21: E5a / E5b | 1:10?1:50 | 1:50 |
| 22: E5c | 1:50?2:20 | 2:20 |
| 23: Conclusion | 2:20?3:00 | 3:00 |

## Backup points for questions (not spoken)

- E4 probes one late-queue sentinel segment, not availability of the whole video. Later delays and a completed-repair control were not tested.
- E5a recovery range: 1.47?2.74 s; E5b: 1.52?1.97 s. Both timers start at Leader stop.
- E5c range: 0.85?1.43 s, measured from quorum restoration. Nine rejected writes are attempts across three trials, not nine separate scenarios.
- E5b also checked task persistence after Controller restart and L configuration after source restart.
- These results do not establish guarantees under sustained network partitions, power loss, or Byzantine faults.
