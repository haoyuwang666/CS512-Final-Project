# Cooperative Video Storage Presentation Script

Speaker: Yuquan Hu. Covers slides 19 to 23: E4, E5, and Conclusion. Target: about 3 minutes, including short pauses and slide changes. Read only the paragraphs below aloud; timing and preparation notes are not part of the speech.

## Slide 19 E4: test the second failure near repair completion

E4 asks whether replacement copies can serve after both original holders stop.

We stop the first holder, wait a fixed delay, then stop the second and request a selected segment. Success requires correct bytes and committed registration.

We tested thirty-three through forty seconds, independently repeated thirty-four seconds, and added a control that waits for verified repair completion.

## Slide 20 E4: availability depends on repair readiness

At thirty-four seconds, independent repeats succeeded in two of four trials for S, three of four for F50, and all four for L.

At forty seconds, every policy succeeded. After verified repair completion, all six controls also delivered all nine segments afresh from the sole survivor.

Availability therefore depends on repair readiness when the next failure occurs. Next, we move to Controller failures.

## Slide 21 E5a and E5b: application recovery after Leader loss

For E5, we require a successful application request, not just a new Leader.

In E5a, we stop the Leader during delivery. Fresh metadata requests recover, and both warm and cold sessions complete in all three trials. Median recovery is one point five two seconds.

In E5b, we stop the Leader with a committed repair task pending. The new Leader preserves the task ID and creation time, and the replacement is verified and registered. All three trials pass, with median recovery of one point six seconds.

## Slide 22 E5c: without a majority, writes were refused

E5c removes the Controller majority. During stable majority loss, all nine valid write attempts return HTTP five-oh-three, with no success acknowledgments.

After quorum restoration, the baseline state is retained and the original operation IDs are reconciled. All three trials pass, with median recovery of one point zero four seconds. This timer starts when quorum is restored, excluding the deliberate outage.

## Slide 23 Conclusions: playback, repair, and application recovery

To conclude, our experiments connect playback performance, repair timing, and failure recovery. Repair shares bandwidth with viewers, and the policies balance playback stalls against recovery speed.

E4 shows that repair readiness matters when another Peer fails. With repair complete, replacement copies successfully served all nine segments after both original holders stopped. E5 verifies fresh metadata recovery, repair task continuity, and write refusal without a majority.

Together, these results show how the system maintains data delivery and restores application service under the tested failures. Thank you.

## Rehearsal timing (not spoken)

| Slide | Time | Cumulative |
|---|---|---|
| 19: E4 design | 0:00-0:38 | 0:38 |
| 20: E4 result | 0:38-1:18 | 1:18 |
| 21: E5a / E5b | 1:18-1:58 | 1:58 |
| 22: E5c | 1:58-2:25 | 2:25 |
| 23: Conclusion | 2:25-3:00 | 3:00 |

## Backup points for questions (not spoken)

- E4 fixed-delay trials probe one late-queue sentinel. The six completion-gated controls additionally fetched all nine segments afresh; this is byte delivery, not browser playback.
- The main E4 table presents independent validation at 34 s: two repetitions per context, two contexts, four trials per policy. The complete exploratory sweep and raw results are retained in report/e4-followup-results.md.
- All fault intervals are measured after the first failure, which occurs at experiment time 5 s. A 34 s interval means the second failure is scheduled at experiment time 39 s.
- E5a recovery range: 1.47-2.74 s; E5b: 1.52-1.97 s. Both timers start at Leader stop.
- E5c range: 0.85-1.43 s, measured from quorum restoration. Nine rejected writes are attempts across three trials, not nine separate scenarios.
- E5b also checked task persistence after Controller restart and L configuration after source restart.
- These results do not establish guarantees under sustained network partitions, power loss, or Byzantine faults.
