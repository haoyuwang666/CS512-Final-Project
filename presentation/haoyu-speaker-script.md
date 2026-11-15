# Cooperative Video Storage Presentation Script

Speaker: Haoyu Wang. Covers slides 1 to 10 in the current presentation, ending before methods. Read the paragraphs aloud; the last paragraph of each slide leads into the next. Allow about 8 to 10 minutes, including pauses and pointing to diagrams.

## Slide 1 Balancing Playback Continuity and Replica Recovery

Hi everyone. I'm Haoyu, and our team is presenting our cooperative video storage project.

We built a system that stores copies of video across several storage nodes and can play it back. We use it to study what happens after a storage node fails, especially how to keep serving viewers while rebuilding missing copies.

I'll introduce the system, our research questions, and the repair policies we compare. My teammates will explain the experiments and results.

## Slide 2 Our prototype cooperative video storage

The starting idea is peer-to-peer, or P2P. Participating nodes contribute storage and upload bandwidth. BitTorrent is a familiar example: users download pieces of a file and also upload pieces to other users.

Our design uses dedicated storage nodes, which we call Peers. Viewers download video, but they do not upload it to other viewers. A separate group of Controllers manages the information about the video and its copies. So our design is closer to storage servers working together than to a pure P2P network.

Let's look at how these parts connect.

## Slide 3 System overview

On the left, the uploader sends video segments to the storage Peers. The Peers store the actual video bytes on disk.

Above them are three Controllers. They use Raft to agree on metadata, such as where segments are stored and which repair tasks need to run. One Controller is the leader, and committing updates requires a majority. Raft protects this shared control information; video bytes stay on the Peers.

On the right, the Gateway asks the Controllers where to find a segment, fetches and checks it, and sends it to the browser. Dashed arrows show control information, and solid orange arrows show video data.

Now let's follow one video through that system.

## Slide 4 A video journey from upload to playback

The top row shows upload and publication. First, FFmpeg packages the video into small MP4 segments. We upload each segment to two different Peers. Each Peer checks its hash, saves the bytes, and returns a receipt confirming storage. Once the Controllers confirm two copies of every segment, they mark the video READY through Raft.

The bottom row shows playback. The Gateway gets the segment locations from the Controllers and downloads from a Peer. The Gateway and browser check the data, and the browser adds the segments to its playback buffer using MediaSource Extensions, or MSE.

That is the browser path. Our experiments transfer real bytes but simulate video consumption; my teammate will explain that setup.

These steps rely on a few main tools.

## Slide 5 Application implementation and validation

We use HashiCorp's Raft library, BoltDB, and snapshots to support persistent metadata and repair tasks. The storage services use Go and HTTP, with SHA-256 hashes to check data integrity. FFmpeg and FFprobe handle media processing and checks, and the management console uses React and TypeScript.

The archived checks passed for areas including repair, cleanup, video deletion, and media decoding. These checks support the implementation; the experiments examine its behavior under load and failures.

To understand those experiments, we first need to separate two meanings of recovery.

## Slide 6 Playback recovery and redundancy recovery

Imagine one segment has copies on Peer A and Peer B. If A fails, the client can try B, check the bytes, and continue delivery. That is playback recovery. It uses a copy that already exists.

But B may now be the only reachable copy. We also need to create a replacement on Peer C. The Controller detects the shortage, waits through a configured grace period, records a repair task, and copies the segment. Repair finishes after the replacement is verified and registered. That is redundancy recovery.

So a video can keep playing while the system still has reduced protection against another failure. This is why we measure both playback and replica restoration.

Rebuilding that missing copy, however, uses resources that playback also needs.

## Slide 7 One upload budget serves two demands

This diagram zooms in on the surviving Peer. The upper path serves viewers. The lower path supplies repair data to a Controller, which forwards it to a replacement Peer.

Both paths use the surviving Peer's limited upload bandwidth. Repair can compete with playback and increase waiting. But postponing repair leaves fewer reachable copies for longer.

The goal is to understand this tradeoff: how should we use the available bandwidth to serve viewers and restore redundancy?

That leads to our research questions.

## Slide 8 Three questions organize the study

Our first question is about interference and timing. When does repair disrupt playback, and what changes if we wait before starting it? Experiments E1 and E2 address this.

Our second question is about the policy choice. Can a repair policy that responds to current activity give a better balance between playback and replica restoration than fixed policies? E3 compares these policies, and E4 examines what happens when a second holder fails.

Our third question is about correctness and recovery. Under the failures we test, can the application preserve its committed work and recover service? E4 and E5 examine those boundaries.

To investigate these questions, we compare the following conditions.

## Slide 9 Every policy obeys the same total source cap

N is normal service with no failure. OFF introduces a Peer failure but disables repair. Comparing them helps us understand the effect of the failure itself.

S enables repair under the shared upload limit. Comparing S with OFF helps us identify the additional effect of repair traffic.

F25 and F50 add fixed repair rate caps of 25 and 50 percent of the source's upload capacity. L changes its repair allowance based on current activity.

All conditions keep the same total source capacity. For example, with a total limit of 100 kilobytes per second, F25 caps repair at 25. Combined traffic still shares the total limit of 100. This cap does not reserve bandwidth for repair.

The last slide explains how L changes that allowance.

## Slide 10 L reacts to source local foreground activity

L watches foreground download activity at each source Peer. While those downloads are active, its repair cap is 25 percent. After two seconds without foreground activity, the cap rises to 100 percent so repair can use available bandwidth.

If an active repair task is older than 60 seconds, the repair cap is at least 50 percent, even during foreground activity. This is intended to help older tasks make progress. The policy updates every 250 milliseconds, and these allowances are rate caps, not guaranteed transfer speeds.

The 60-second rule was checked in regression tests, but it did not activate during our experiment runs. L responds to source activity rather than directly reading viewers' playback buffers.

That completes the system and policy introduction. I'll now hand over to my teammate to explain how we tested these questions and what the results show.
