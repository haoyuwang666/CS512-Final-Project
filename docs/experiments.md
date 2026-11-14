# Compact E1–E5 study

Executed: 81 frozen runs in 67.9 minutes, with nine controller cases passing. See [results and conclusions](../report/compact-study-results.md) and the [slide exhibit guide](../results/compact-study/SLIDE-EXHIBITS.md). Raw runs remain under `results/compact-study/`.

The implemented, headless study is described in [compact-experiment-design.md](../report/compact-experiment-design.md). Run `make experiment-compact` for the serial E1–E5 matrix and tables/PNG/SVG figures. Defaults are 72 unique performance runs and 9 controller runs, designed for an approximately 80–95 minute execution including separate pilot/gates. See the frozen JSON scenario in `experiments/scenarios/compact-repair.json`. Earlier baseline limitations below refer to the older runners.

# 实验接口与状态

当前新增 `experiments/paired.py` / `compare.py` 和可执行的 main scenario：独立集群重置、同 seed trace、四策略组、相同副本预算与 Peer 上限、完整失败会话和 run 层配对差值。参见 reliability.md。下面关于 baseline runner 的限制保留；新入口是冷启动 headless 对照，仍明确标记 formal_experiment=false，不代表包含 warm-up、故障矩阵与热度学习的最终研究。

TODO(Tian)：完成独立随机工作负载、故障 trace、环境重置、资源采集和配对不确定性分析后，才能开始正式研究实验。

当前 baseline runner 调用真实 Go CLI，可并发 viewers，记录 session JSONL 与 stderr 事件。seed 只是重复运行标签，没有随机化含义；不自动重置 Peer、Raft、cache、counter、bucket。输出带 `formal_experiment:false`。主实验配置标为未实现，runner 明确退出。

正式方案需要：多视频、固定码率、warm-up 及测量窗口、低/中/高负载、相同 trace、每 Peer quota/upload cap、初始 buffer/并发一致、匹配额外副本的固定放置对照、全部 Repair 和复制成本。首轮放置可 warm-up 后冻结，但应据实命名。

聚合以 run 为单位，Completion 分母保留全部 session；失败/停止另外列出。当前 completed-only 平均卡顿明确在列名中标注，不能作为全体 session 的无条件收益。诊断 SVG 只是数据检查，不提供置信区间。

每轮结果应附配置、实际代码 commit、工具和镜像版本、素材 Hash、随机 trace、实际存储峰值、Peer 上传计数和 controller recovery 事件。上述资源/环境全量记录尚未自动完成。

Playback 已支持 Deadline、启动缓冲、请求事件与实时指标；当前 runner 仍限制为 fixed + round_robin，主实验依赖自适应放置与匹配预算，因此继续显式禁用。Session 的 verified_bytes 定义已扩展为所有验证成功的预取数据；跨代码版本的数据不可直接混合聚合。
