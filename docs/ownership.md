# 目录归属与接手说明

TODO(Haoyu)：任务转交时同步本表；这份表记录团队约定的分工，尚非贡献完成声明。

| 唯一负责人 | 负责的目录/文件 | 下一步 |
|---|---|---|
| Haoyu | cmd/controller、internal/domain、internal/protocol、internal/metadata、internal/control、deployments、.github、根级配置/构建/索引/README、tests/system、tests/e2e | 预留、任务状态、故障恢复、Docker 验证 |
| Yuquan | cmd/peer、internal/storage、internal/placement、internal/repair、scripts/media、scripts/ingest | Inventory/Heartbeat、Repair 执行、自适应额外副本 |
| Tian | cmd/client、cmd/demo-api、internal/playback、frontend、experiments、results、tests/integration | 来源预测与 deadline 校准、正式实验 trace/重置、性能评估 |

文档：Haoyu 维护 architecture/api/invariants/decisions/ownership/verification，Yuquan 维护 storage，Tian 维护 playback/experiments/demo。报告：Haoyu metadata/integration/final，Yuquan storage，Tian evaluation。

`scripts/check_scaffold.py` 为 Haoyu 维护的源文件用途检查，不属于 Yuquan 的素材或导入工具。

边界：Yuquan 决定副本放哪里，Tian 决定一次播放从哪里下载，Haoyu 决定元数据和任务如何提交。React/Python 不另写卡顿公式。其他人修改公共 DTO、go.mod、Compose 前先给负责人提变更，不能并行各写一版。

本次框架跨目录创建属于用户明确授权的统一初始化。之后各成员认领任务，基于现有代码继续，不要求再次生成各自的基础版本。分支建议 `feat/haoyu-metadata`、`feat/yuquan-storage`、`feat/tian-playback`，使用独立工作目录。
