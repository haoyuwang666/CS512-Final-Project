# Evaluation 章节草稿

TODO(Tian)：填写工作负载、配对对照、指标、不确定性及负结果；不将基础流水线数据当正式实验。

## 当前可复现入口与诊断结果

新增 `experiments/paired.py` 重置真实集群，以 seeded 多视频到达 trace 比较
fixed/adaptive × round_robin/deadline，配置上传 cap、相同副本字节预算和播放参数。
`compare.py` 按 seed 检查配对并生成 run 层差值/诊断 bootstrap 区间。
源码快照、二进制哈希、配置、原始事件和 Peer 字节计数均可归档。

首次 20 run / 60 session 诊断中，fixed-deadline 完成 14/15，会话失败因片段请求超时；
其他三组各完成 15/15。保留失败分母，不用仅成功会话的卡顿均值掩盖失败。
本次有并行系统验收带来的时序干扰，只验证流水线，formal_experiment=false。
完整研究仍需独占环境、warm-up、负载矩阵和可重放故障 trace，不能由这次诊断推断普遍收益。
实际命令、配置及验证边界见 docs/reliability.md、docs/verification.md。
