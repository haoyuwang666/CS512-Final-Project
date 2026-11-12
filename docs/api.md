# HTTP API v1

TODO(Haoyu)：维护唯一公共契约；引入预留、Heartbeat 和任务前先补请求/响应样例及调用方迁移。

前缀 `/api/v1`。JSON 使用 snake_case，时间为毫秒、空间/流量为字节、吞吐为 bytes/s。ID 只能包含字母、数字、`_`、`-`，最长 128。Manifest 每段最大 64 MiB，JSON 请求体最大 4 MiB，视频目前只有表示版本 1。

## 已实现

| 服务 | 方法及路径 | 请求/响应 |
|---|---|---|
| 全部 | GET `/health` | 本进程可响应；不是集群 quorum 证明 |
| Controller | GET `/cluster` | Leader URL、成员、本机 Raft state/stats、已配置 Peers |
| Controller | POST `/videos` | `{operation_id, manifest}`；创建 DRAFT |
| Controller | POST `/videos/{id}/placement-plan` | 空 JSON；`{advisory:true, placements:{segment_id:[{peer_id,url},...]}}` |
| Controller | POST `/segments/{id}/replicas` | `{operation_id, receipt}`；只接收已配置 Peer 的可核对对象 |
| Controller | POST `/videos/{id}/publish` | `{operation_id, expected_version:1}`；每段至少两个不同 Peer 才提交 READY |
| Controller | GET `/videos/{id}/manifest` | 仅 READY 可读；返回 Manifest |
| Controller | GET `/segments/{id}/locations` | 仅 READY 视频的片段可见；返回 `{segment_id, locations:[{peer_id,url,representation_version}]}` |
| Peer | PUT `/segments/{id}` | 二进制 body，见 header 约定；返回 Receipt |
| Peer | GET `/segments/{id}` | 真实原始字节；Content-Length、X-SHA256 |
| Peer | GET `/segments/{id}/receipt` | 已持久安装对象的 Receipt；不传输视频字节 |
| Peer | GET `/stats` | peer_id、instance_id、stored_bytes、quota_bytes、objects、uploaded_bytes、upload_bytes_per_second |
| Demo | POST `/sessions` | `{video_id,strategy:"round_robin"或"deadline",window?,concurrency?,startup_buffer_ms?}` → HTTP 202 `{session_id}` |
| Demo | GET `/sessions/{id}` | `{session_id,summary,events,buffer,peers,persistence_error?}`；running 阶段返回实时快照 |
| Demo | POST `/sessions/{id}/stop` | 请求取消；最终状态随后可读 |
| Demo | GET `/sessions/{id}/events` | SSE data 为 Event，id 为单调 sequence；支持 Last-Event-ID 和磁盘历史回放 |

PUT header：`X-Operation-ID`、`X-Size-Bytes`、`X-SHA256`、`X-Representation-Version: 1`。Receipt 字段：`operation_id, peer_id, segment_id, representation_version, size_bytes, sha256`。

Manifest：`schema_version:1, video_id, representation, segments`。每段：`segment_id, index, duration_ms, size_bytes, sha256`。index 连续从 0 开始，SHA-256 为 64 位小写 hex。视频 ID 不可覆盖，片段 ID 在全部视频中唯一。

操作 ID 同内容重试返回原结果，同 ID 不同内容冲突。FSM 拒绝结果也被记住；修复条件后必须使用新操作 ID，不能拿原来的失败操作假装是新命令。导入脚本按 Manifest 内容生成稳定 ID。

错误格式：`{error:{code,message,retryable,leader_url?}}`。参数错误 400、缺失 404、版本冲突/未就绪 409、尚未实现 501、不可服务/结果未知 503、容量不足 507。Leader 只做提示，不自动将视频 PUT 重定向。客户端向已配置 Controller 重试；写结果未知保留原 operation_id。

## 明确未实现

`POST /peers/{id}/heartbeat`、`POST /peers/{id}/inventory`、`GET /peers/{id}/tasks`、`POST /tasks/{id}/ack` 返回 501。无 DELETE。未知播放策略返回 400；`round_robin` 和 `deadline` 均可用。放置没有持久预留 ID；不宣称支持并发导入的全局预留保证。Peer 自身原子配额检查始终生效。

SSE 使用 Last-Event-ID 精确续传；无效或超前序号返回 400，无磁盘历史且已超出事件环返回 409 EVENT_HISTORY_EXPIRED。连接期间落后过多会收到 reset 事件，前端重新读取快照。默认持久化目录为 .data/sessions，详情见 docs/playback.md。未来新增 Heartbeat、Task、Inventory DTO 应统一放 `internal/protocol`，不由消费者各自定义协议。

播放请求默认 window=3、concurrency=min(2,window)、startup_buffer_ms=0（首段启动）。窗口 1–64，并发 1–window，启动缓冲 0–3600000 ms；启动前缀超过窗口时 Session 以 failed 结束。新增 Summary 字段：downloaded_bytes、requests、failed_requests、deadline_misses；verified_bytes 现在统计所有验证成功的预取段。事件新增 bytes、elapsed_ms、deadline；持久日志另含 snapshot，REST 事件环与 SSE 不重复发送该快照。


## 浏览器媒体交付

- POST `/media/sessions`：请求字段同 SessionRequest，成功为 201 `{session_id,manifest,window,concurrency,startup_buffer_ms}`。只接受 continuous fMP4 表示；旧 independent MP4 返回 422 REPACKAGE_REQUIRED，fixture 返回 422，启动缓冲超出窗口返回 400，已达 100 个 lease 返回 429。
- GET `/media/sessions/{id}/segments/{index}`：根据 Manifest 查找指定分片，复用副本选源、大小/SHA-256 校验和失败切源，成功返回 `video/mp4`，支持 Range。浏览器不需要直接连接 Peer。
- 响应头：`X-Peer-ID`、`X-Verified-SHA256`、`X-Download-Bytes`、`X-Download-Attempts`、`X-Download-Failures`、`X-Download-Elapsed-MS`。尝试数/响应体流量仅属于本次成功交付请求；Range 也会先校验完整对象。
- POST `/media/sessions/{id}/stop`：幂等取消 lease 和进行中的下载。后续查询返回 404，正在进行且连接仍有效的请求返回 410 MEDIA_STOPPED。最终无法取得有效副本返回 502 MEDIA_UNAVAILABLE。

媒体 lease 空闲 30 分钟失效；不持久化，也不包含模拟 Session 的 Summary/SSE。真实播放状态由浏览器媒体事件测量。

连续媒体的 Manifest 新增可选字段 `media:{mime_type,duration_ms}`。continuous-fmp4-h264-v1 / continuous-fmp4-h264-aac-v1 必须提供此字段，codec MIME 必须为 AVC（可加 AAC LC），总时长须等于分片时长之和。对象自身含相同初始化信息及原始连续时间戳；Range 仍先验证完整对象。旧 Manifest 的 JSON 结构、headless 导入与 Raft 快照保持兼容，但已有运行进程需要重启以接受新字段。
