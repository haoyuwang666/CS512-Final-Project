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
| Demo | POST `/sessions` | `{video_id,strategy:"round_robin"}` → HTTP 202 `{session_id}` |
| Demo | GET `/sessions/{id}` | `{session_id,summary,events}`；running 阶段最终指标尚未汇总 |
| Demo | POST `/sessions/{id}/stop` | 请求取消；最终状态随后可读 |
| Demo | GET `/sessions/{id}/events` | SSE data 为 Event，id 为单调 sequence；最多保留 256 个事件 |

PUT header：`X-Operation-ID`、`X-Size-Bytes`、`X-SHA256`、`X-Representation-Version: 1`。Receipt 字段：`operation_id, peer_id, segment_id, representation_version, size_bytes, sha256`。

Manifest：`schema_version:1, video_id, representation, segments`。每段：`segment_id, index, duration_ms, size_bytes, sha256`。index 连续从 0 开始，SHA-256 为 64 位小写 hex。视频 ID 不可覆盖，片段 ID 在全部视频中唯一。

操作 ID 同内容重试返回原结果，同 ID 不同内容冲突。FSM 拒绝结果也被记住；修复条件后必须使用新操作 ID，不能拿原来的失败操作假装是新命令。导入脚本按 Manifest 内容生成稳定 ID。

错误格式：`{error:{code,message,retryable,leader_url?}}`。参数错误 400、缺失 404、版本冲突/未就绪 409、尚未实现 501、不可服务/结果未知 503、容量不足 507。Leader 只做提示，不自动将视频 PUT 重定向。客户端向已配置 Controller 重试；写结果未知保留原 operation_id。

## 明确未实现

`POST /peers/{id}/heartbeat`、`POST /peers/{id}/inventory`、`GET /peers/{id}/tasks`、`POST /tasks/{id}/ack` 返回 501。无 DELETE。`strategy=deadline` 返回 501（CLI 返回失败）。放置没有持久预留 ID；不宣称支持并发导入的全局预留保证。Peer 自身原子配额检查始终生效。

SSE 暂不使用 Last-Event-ID 精确续传；重连后读取快照并按 sequence 去重。前端当前只用快照轮询。未来新增 Heartbeat、Task、Inventory DTO 应统一放 `internal/protocol`，不由消费者各自定义协议。
