# 系统架构

TODO(Haoyu)：在引入持久任务和预留时更新交互流程；保持公共协议与实现一致。

控制平面：Controller 的真实 Raft 集群持久化 Manifest、Replica ACK、READY 状态、操作去重结果。FSM 无网络、时间或随机副作用。API 读取前验证领导权并通过 Barrier 同步应用进度。

数据平面：Ingestion 直接 PUT Peer；Peer 在对象及索引持久化后返回 Receipt。Controller 只向启动时配置的 Peer URL 核对对象元信息，再提交位置。GET 字节绕过 Raft。

播放平面：Go Engine 读取已发布 Manifest 与 Locations，用有界窗口和并发下载真实数据，验证后标记逻辑 buffer 可用；按段时长消费，不解码。当前验证后的载荷不长期保留，仅保留可消费状态，因此还不能接真实播放器。

显示平面：Demo API 运行同一 Engine；React 只发送 Session 命令并展示事件和汇总。Controller/Peer/Buffer 详细面板尚未实现。

扩展边界：Placement 与 Repair 只输入快照、输出计划；Control 将计划变成持久任务；Storage 执行复制。此任务闭环尚未实现，不能自动修复或回收。

故障边界：Raft 保护元数据，不保护视频字节；双副本只能容忍每个对象丢失一个持有者。尚未实现缓存 Manifest，因此 Controller 无法读取时，即使视频字节可达，新增播放/片段查询仍可能失败。
