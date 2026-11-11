// TODO(Tian): Update these consumer DTOs only after Haoyu publishes contract changes.
export interface PlaybackEvent { sequence: number; type: string; time: string; segment_id?: string; peer_id?: string; message?: string }
export interface Summary { video_id: string; status: string; error?: string; startup_delay_ms: number; stall_time_ms: number; media_playback_elapsed_ms: number; rebuffering_ratio: number; verified_bytes: number; segments_played: number; simulated_playback: boolean }
export interface Session { session_id: string; summary: Summary; events: PlaybackEvent[] }
