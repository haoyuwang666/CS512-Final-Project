# Peer 与导入

TODO(Yuquan)：实现 Heartbeat、Inventory、复制任务执行与观测热度放置；后续添加磁盘故障注入测试。

对象落盘为 `sha256.blob`；持久 `index.json` 维护 segment → Receipt 和 operation → Receipt。索引通过临时文件、fsync、rename、目录 fsync 替换。PUT 在一个 Peer 内串行保护配额和索引，同内容可共享物理 blob。Quota 当前计算 blob 载荷占用，不含索引、临时文件及文件系统开销；严格物理配额需后续扩展。

上传临时文件在正常错误时清理；崩溃留下的临时文件暂无自动清理。未登记 blob 保守计入 used，避免空间被重复花掉。不得随意删除孤儿文件或 index；未来修复需要明确恢复流程。运行时没有后台 Bitrot Scrub，下载端仍检查 Hash。

共享 limiter 为字节 token bucket，burst 32 KiB。统计为 ResponseWriter 接受的字节，包括失败连接此前已经写出的部分；不能声称是精确 TCP 线速计量。维护优先级调度尚未实现。

导入工具使用确定性 operation ID，依次创建 Draft、获取静态放置建议、PUT 两份、登记 ACK、发布。放置按本次计划预计空间扣减，但不锁住跨请求容量；并发导入若空间不足由 Peer 拒绝，视频保持 DRAFT，不发布部分视频。
