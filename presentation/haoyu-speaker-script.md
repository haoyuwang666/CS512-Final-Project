# Cooperative Video Storage Presentation Script

TODO(Haoyu): Keep the speech aligned with slides 1 to 9 and regenerate the Word export after edits.

Speaker: Haoyu Wang. Covers slides 1 to 9, ending before methods on slide 10. Target length is 3 to 4 minutes. Read only the slide paragraphs aloud, with brief pauses to point to the diagrams.

## Slide 1 Balancing Playback Continuity and Replica Recovery

Hi everyone, I'm Haoyu. Our project looks at how to keep videos playing while rebuilding missing copies after a storage failure. I'll start with the system.

## Slide 2 Our prototype cooperative video storage

Our system's starting idea is P2P, or peer-to-peer. Computers share resources with each other. BitTorrent is an example: users download file pieces and upload them to others. In our design, dedicated Peers store video, and viewers only download.

## Slide 3 System overview

Now let's look at our system structure. Peers store the video segments. Three Controllers use Raft to agree on segment locations and repair tasks. The Gateway asks where to find segments, fetches them from Peers, and sends them to the browser.

## Slide 4 A video journey from upload to playback

So how does a video move through this system? FFmpeg packages it into segments. Each segment is checked and saved on two Peers before publication. For playback, the Gateway fetches segments, and the browser checks and buffers them.

## Slide 5 Playback recovery and redundancy recovery

Now, what happens if a Peer fails? Suppose A and B hold a segment. If A fails, playback can continue from B. But we still need a replacement copy on C. Until that finishes, B may be the only reachable copy.

## Slide 6 One upload budget serves two demands

And that's where the challenge comes in. B must upload data to viewers and, through the Controller, to C for repair. These share B's bandwidth. Repair can slow playback, but waiting leaves us with fewer copies for longer.

## Slide 7 Three questions organize the study

This leads to three questions. When does repair affect playback, and what does waiting change? Can an adaptive policy balance playback and repair better than fixed policies? And can the system preserve committed work and recover under the failures we test?

## Slide 8 Every policy obeys the same total source cap

To answer these, we compare several conditions. N has no failure. OFF has failure without repair. S enables shared repair. F25 and F50 cap repair at 25 and 50 percent of source capacity. L adjusts its cap. All keep the same total upload limit.

## Slide 9 L reacts to source local foreground activity

Finally, here's how L works. While a Peer serves viewer downloads, repair is capped at 25 percent. After two seconds without those requests, it rises to 100 percent. For tasks older than 60 seconds, the cap is at least 50 percent. These caps don't reserve bandwidth.

My teammate will now explain how we tested these policies.
