// TODO(Tian): Keep additive playback fields aligned with the Go API contract.
export interface PlaybackEvent {
  sequence: number; type: string; time: string; segment_id?: string; peer_id?: string;
  message?: string; bytes?: number; elapsed_ms?: number; deadline?: string;
}
export interface Summary {
  video_id: string; status: string; error?: string; startup_delay_ms: number;
  stall_time_ms: number; media_playback_elapsed_ms: number; rebuffering_ratio: number;
  verified_bytes: number; downloaded_bytes: number; requests: number; failed_requests: number;
  deadline_misses: number; segments_played: number; simulated_playback: boolean;
}
export interface BufferEntry { segment_id: string; index: number; duration_ms: number; state: string; peer_id?: string }
export interface PeerEstimate { peer_id: string; bytes_per_second: number; in_flight: number; failures: number }
export interface Session {
  session_id: string; summary: Summary; events: PlaybackEvent[];
  buffer: BufferEntry[]; peers: PeerEstimate[]; persistence_error?: string;
}

export interface MediaSegment { segment_id: string; index: number; duration_ms: number; size_bytes: number; sha256: string }
export interface MediaManifest { schema_version: number; video_id: string; representation: string; media?: { mime_type: string; duration_ms: number }; segments: MediaSegment[] }
export interface MediaConfig { video_id: string; strategy: string; window: number; concurrency: number; startup_buffer_ms: number }
export interface MediaSession { session_id: string; manifest: MediaManifest; window: number; concurrency: number; startup_buffer_ms: number }
