# E3 新分批到达实验结果

TODO(team): Keep this report synchronized with the audited late-staggered results and preserve invalid-run provenance.

已完成 12 个有效场景，共 72 个播放会话。另有一个测量异常的原始 run 被完整保留，并以独立补跑替换；不是因结果不理想而重跑。

## 配置与目的

- 3 Controllers、3 Peers，单机多进程，真实 HTTP 传输、模拟播放消费。
- 六个 viewer 在 **0/0.2、20/20.2、50/50.2 s** 分三批加入。
- 第 5 s 停止一个 Peer，之后不恢复；grace=3 s。
- S、F25、F50、L 各测试三个故障 Peer（上传上限分别为 96/144/192 KiB/s），每种容量场景一次，不是每个场景重复三次。
- 观察窗口延长到 100 s；会话超时仍为 42 s，使最后一批有完整运行时间。
- 九个 128 KiB 分片、每片两秒、两份初始副本和客户端设置均沿用 compact study。

目的是补充原 0/4/12 s 到达实验，直接覆盖修复进行中和完整恢复后的播放；不是执行整个增强实验计划。

## 汇总结果

卡顿为所有预定 viewer 的平均故障后累计观察卡顿。恢复耗时从实际首次故障算起。

| 策略 | 完成会话 | 平均卡顿 / viewer | 40 s 内恢复 | 三个场景的恢复耗时范围 | 平均 one-copy exposure | 平均 repair traffic |
|---|---:|---:|---:|---:|---:|---:|
| S | 18/18 | 0.00 s | 3/3 | 29.45–31.43 s | 14.99 MiB·s | 0.750 MiB |
| F25 | 18/18 | 0.02 s | 3/3 | 31.55–37.48 s | 17.63 MiB·s | 0.750 MiB |
| F50 | 18/18 | 0.00 s | 3/3 | 29.45–31.47 s | 15.48 MiB·s | 0.750 MiB |
| L | 18/18 | 0.02 s | 3/3 | 31.53–35.55 s | 16.87 MiB·s | 0.750 MiB |

## 是否真的覆盖恢复之后

四种策略的第 50 s 批次均在此前最近一次副本检查已经确认完整恢复后启动：12/12 场景，24/24 后到会话完成，平均卡顿为 0。完整恢复被观察到的实验时刻约为第 34.45–42.56 s。

第 20 s 批次启动前，12/12 场景尚未观察到所有受影响分片恢复双副本；该批次仍全部完成。F25 与 L 的少量卡顿出现在这批会话，其跨三个场景的平均累计卡顿均约 0.05 s/viewer。

**但所有场景的受影响分片均未记录到来自替代 Peer 的 segment_verified 交付事件。** 后到会话完成不等于利用了新副本。当前固定 placement、按分片编号排列候选来源的 round-robin 与存活原副本足以服务这些请求；这次数据不支持“repair 提高了后到 viewer 完成率”的因果结论。

## 结论与边界

1. 这次到达安排确实覆盖了完整恢复之后，而且前、中、后三批都能完成播放。
2. 与旧实验分批较密集的到达安排相比，新实验卡顿明显较少、完成率更高；到达间隔变长改变了并发重叠，不能只归因于 repair 或新副本。两批历史实验也不是同步重跑的配对对照。
3. 在这次低播放干扰场景中，各策略完成率均饱和，无法体现 L 的播放优势。S/F50 观察到较早恢复，F25/L 额外限速没有带来可见的完成率收益；样本少，不能推广为固定策略总是更好。
4. 没有 OFF 对照，也没有新副本实际服务证据，不能据此证明启用 repair 提高了完成率。
5. 恢复约每两秒验证一次；报告值是首次观察到恢复的时间，应结合逐 run 的上下界，不能按小数点细微差异排序。

可用于展示的英文结论：

> With wider-spaced arrivals, all policies completed every session and restored replicas within 40 seconds of failure. Late viewers started after recovery, but no delivery from replacement replicas was observed, so the completion benefit of repair remains unestablished.

## 测量审计与复现

原 F50 / seed 1101 run 只有四条会话记录，计划第 20.2/50.2 s 的会话记录为约第 17.4 s 启动，副本验证日志仅到第 16.22 s，时间关系不可能。因此按测量无效处理；具体原因未确定。使用相同配置和二进制重新运行该场景。原文件未删除，[替换清单](../results/late-staggered-e3-run1/replacements.json)记录理由及补跑路径。

全部纳入结果通过：6 个唯一会话、实际到达时间与计划相差不足 1 s、会话结束晚于启动、验证覆盖到至少第 97 s、验证时钟不倒退、无未知副本状态、故障注入确认。应用播放失败本应保留，本次有效 runs 中没有出现。

```sh
make backend
python3 experiments/late_staggered_study.py --output-dir results/new-late-staggered
python3 experiments/analyze_late_staggered.py results/new-late-staggered
```

已有结果请直接分析，不要再次运行到相同非空目录：

```sh
python3 experiments/analyze_late_staggered.py results/late-staggered-e3-run1
```

- [冻结配置](../experiments/scenarios/late-staggered-e3.json)
- [完整汇总与 cohort 表](../results/late-staggered-e3-run1/RESULTS.md)
- [逐场景 CSV](../results/late-staggered-e3-run1/run-details.csv)
- [逐批 viewer CSV](../results/late-staggered-e3-run1/cohort-details.csv)
- 原始 journals、会话、任务、流量、进程日志及源码/二进制 provenance 在 `results/late-staggered-e3-run1/`；补跑在 `results/late-staggered-e3-replacement/`。

新增结果目录沿用仓库忽略策略，不会自动进入 Git；本报告和复现实验脚本可以提交。
