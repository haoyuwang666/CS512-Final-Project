# 开发决策与未决事项

TODO(Haoyu)：核对课程真实播放要求、正式提交/演示时间，并在变更接口前更新决策。

- 采用 Go + React/TypeScript/Vite + Python；公共 API 包统一为 protocol。
- 真实 Raft 固定成员配置，首次只由一个参与者 bootstrap，重启不重建集群。
- 前端统一 pnpm 11.25.0，提交 pnpm-lock，不同时维护 npm lockfile。esbuild 构建脚本在 pnpm-workspace 中显式允许。
- 无自动删除、无 Coding、无 Hedging。现有 Repair 恢复最少两份副本；Adaptive 是资源感知的导入时放置，不是热度学习。任务经 Raft，复制由 Leader 协调，详见 reliability.md。
- 初始 placement 为 advisory，Peer 最终配额防止超用；这与团队目标中的持久 reservation 有差距，需要 Haoyu/Yuquan 后续完成。
- 测试字节 fixture 不要求解码；真实视频由 FFmpeg 另生成。Headless 仅研究交付与模拟消费。
- Docker 地址和宿主机地址不混用：容器内导入/实验使用容器网络；本机系统测试全部使用 loopback。
- 日程暂按原文 11/24，年份与演示日期仍需向课程通知核实。

- 客户端真实播放使用 MSE 连续 fMP4：视频和 AAC 单次编码、按关键帧拆分并保留时间戳，播放器不换源。Headless 模式继续独立存在。旧独立 MP4 必须从源文件重新打包，不混入连续时间线。
