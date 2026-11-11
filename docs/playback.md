# Headless 播放基础

TODO(Tian)：实现预计完成时间选源、deadline miss、缓存元数据、可控时钟和完整实时 Buffer 快照。

基线使用 Manifest index 旋转候选副本，失败逐个切源，再刷新 Locations 一次。Go HTTP 请求总超时 8 秒；Metadata 请求超时 4 秒；CLI 默认 Session 超时 2 分钟。缺少来源报告失败，不无限等待。暂不使用 hedging。

窗口默认 3、最大 64；并发默认 2、不能大于窗口。每段必须完成真实下载及 size/SHA-256 验证，逻辑结果按 index 消费。验证后丢弃载荷，因此不能直接提供解码；扩展播放器时需把结果槽改为可读媒体对象或临时文件。

模拟 startup：开始请求到第一段可消费。Stall：第一段播放后等待下一片段的 wall-clock 时间。Rebuffering：stall / (已消费媒体时长 + stall)，排除 startup。正在消费时取消计入实际已消费部分，状态 stopped。Verified bytes 当前只计消费循环已接收的完整验证段，不是网络总流量，网络成本必须从 Peer 侧补充。

Session 事件带序号，可并发下载但回调序列化。Demo API 保留最多 100 个 Session、各 256 个事件；最终 Summary 在结束后更新，尚非实时统计。正式事件持久化和活跃 Session 保留策略需要后续完善。

顺序播放的 EDF 可能与 index 顺序相同；未来比较必须区分选源、重试与请求排序的收益。不要把 round_robin 包装成 deadline 返回成功。
