# 系统架构

TODO(Haoyu)：在引入持久任务和预留时更新交互流程；保持公共协议与实现一致。

控制平面：Controller 的真实 Raft 集群持久化 Manifest、Replica ACK、READY 状态、操作去重结果。FSM 无网络、时间或随机副作用。API 读取前验证领导权并通过 Barrier 同步应用进度。

数据平面：Ingestion 直接 PUT Peer；Peer 在对象及索引持久化后返回 Receipt。Controller 只向启动时配置的 Peer URL 核对对象元信息，再提交位置。GET 字节绕过 Raft。

播放平面：Go Engine 读取已发布 Manifest 与 Locations，用有界窗口和并发下载真实数据，验证后标记逻辑 buffer 可用；按段时长消费，不解码。当前验证后的载荷不长期保留，仅保留可消费状态，该 headless 路径继续用于模拟实验。

显示平面：模拟模式通过 Demo API 调用 Go Engine；视频模式通过独立 Media API 获取已校验 MP4，React 通过 MSE 的同一 MediaSource/SourceBuffer 持续解码播放，浏览器持有有界 ArrayBuffer 与媒体缓冲窗口。已展示逻辑 Buffer 与 Session 内 Peer 吞吐估计；Controller 集群及 Peer 存储全量面板尚未实现。

扩展边界：Placement 与 Repair 只输入快照、输出计划；Control 将计划变成持久任务；Storage 执行复制。此任务闭环尚未实现，不能自动修复或回收。

故障边界：Raft 保护元数据，不保护视频字节；双副本只能容忍每个对象丢失一个持有者。Manifest/Locations 有有界内存缓存，短时故障可回退；冷缓存、过期缓存或尚未查询的片段仍依赖可用 Controller，不能保证完全离线播放。

播放引擎支持 Round Robin 与 Deadline 来源预测、可配置启动缓冲、冻结时间线 deadline miss、实时快照。Demo API 的 JSONL 日志支持重启恢复和 SSE 续传；不是媒体字节存储。配置和计量语义见 playback.md。

真实播放链路：React → POST /media/sessions 获取已发布 Manifest → GET /media/sessions/{id}/segments/{index} → 共用 MetadataClient/Downloader/Scheduler 选源并校验 → 浏览器再次校验大小/SHA-256 → MSE appendBuffer → HTMLVideoElement。Gateway 使容器 Peer DNS 与跨域细节不暴露给浏览器。此路径不运行模拟媒体时钟；浏览器真实指标与 headless Summary 分开。媒体 lease 空闲 30 分钟失效，最多 100 个，停止/断连可取消下载，重启不恢复真实播放位置。视频单次编码后按关键帧拆分 continuous fMP4，保留精确 tfdt，避免换源与重复 AAC priming。旧 independent MP4 需要重新打包，不能混入新时间线。
