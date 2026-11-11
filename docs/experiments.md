# 实验接口与状态

TODO(Tian)：完成独立随机工作负载、故障 trace、环境重置、资源采集和配对不确定性分析后，才能开始正式研究实验。

当前 baseline runner 调用真实 Go CLI，可并发 viewers，记录 session JSONL 与 stderr 事件。seed 只是重复运行标签，没有随机化含义；不自动重置 Peer、Raft、cache、counter、bucket。输出带 `formal_experiment:false`。主实验配置标为未实现，runner 明确退出。

正式方案需要：多视频、固定码率、warm-up 及测量窗口、低/中/高负载、相同 trace、每 Peer quota/upload cap、初始 buffer/并发一致、匹配额外副本的固定放置对照、全部 Repair 和复制成本。首轮放置可 warm-up 后冻结，但应据实命名。

聚合以 run 为单位，Completion 分母保留全部 session；失败/停止另外列出。当前 completed-only 平均卡顿明确在列名中标注，不能作为全体 session 的无条件收益。诊断 SVG 只是数据检查，不提供置信区间。

每轮结果应附配置、实际代码 commit、工具和镜像版本、素材 Hash、随机 trace、实际存储峰值、Peer 上传计数和 controller recovery 事件。上述资源/环境全量记录尚未自动完成。
