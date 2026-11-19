# Final report 写作计划

TODO(team): Freeze claims against archived evidence, confirm individual contributions, and finish the eight-page report by November 24.

制定日期：2026-11-19。状态：待执行的写作计划，不代表正文或成员贡献已确认。

## 1. 课程约束与完成标准

- 英文、USENIX 格式、八页；截止 **2026-11-24 11:59 AM，America/New_York**。
- 按官方模板采用 US Letter、双栏、10pt Times 类字体、7×9 英寸正文区域，不压缩边距或字号凑页。
- 八页预算暂包含图表、参考文献、AI 披露和贡献；若课程明确参考文献豁免，再重新分配。
- 报告最后一节详细描述三位成员的实际贡献，不能用 ownership 计划代替完成声明。
- 明确说明实际如何使用 AI，以及人如何核实代码、引用和实验结论。
- 每个研究问题都给出有证据的答案，允许“不支持一致优势”；完整性不等于证明预期方法获胜。

格式来源：https://www.usenix.org/conferences/author-resources/paper-templates
课程要求以本次用户提供的原文为准，未自行推断额外提交规则。

## 2. 最终研究主线

暂定题目：**Balancing Playback Continuity and Replica Recovery in Cooperative Video Storage**。

核心问题：在上传能力有限且异构的存储 Peer 上，故障后的副本修复如何影响播放；等待和带宽限额如何改变播放与副本恢复的权衡？

按最终实现收束范围：专用存储 Peers + 独立观看者 + Raft Controllers，而非所有 viewer 都上传的纯 P2P 网络。Raft 复制 metadata，视频字节不进入 Raft 日志。

贡献拟写为：

1. 可运行的分布式视频存储/播放原型：发布前验证两副本、可校验分片、持久化修复任务、Controller 故障恢复、浏览器播放与管理工具。
2. 在固定总上传能力下实现 S/F25/F50/L，并提供配对故障实验和按流量类别记录的观测工具。
3. 实证分析 repair 的干扰、grace 权衡、局部策略收益与恢复代价，以及第二次故障下修复就绪状态的重要性。

Novelty 定位为该原型与明确工作负载下的经验研究，须经相关工作支持；不声称首次提出复制、动态限速或 Raft，不声称 L 普遍优于已有方法。Difficulty 用具体实现和验证说明，不靠代码行数；Importance 解释播放可用性与恢复保护之间的现实冲突。

原 proposal/checkpoint 的纠删码、冷热分层、流行度、MapReduce 汇总、deadline 优化等逐项核实；未落实/未测量内容放 scope evolution 或 future work，不能沿用早期计划写成已完成贡献。

## 3. 研究问题及可支持的答案

| 问题/假设 | 实际证据 | 最终可写结论 |
|---|---|---|
| H1：repair 会在带宽竞争下额外影响播放吗？ | E1 N/OFF/S；低/高负载；三个不同失败容量场景 | 高负载配对平均观察卡顿增加；低负载未观察到增加，影响依赖场景 |
| H2：grace 能减少短故障复制，但延长风险窗口吗？ | E2 1/3/8 s grace，4 s 暂时离线与持续离线 | 短故障避免部分复制；持久故障 one-copy exposure 增长约16% |
| H3：L 是否稳定优于固定额度？ | E3 主研究与独立 late-staggered 扩展 | 不支持稳定优势；存在单场景减少卡顿但恢复变慢的 trade-off |
| H4：修复是否帮助应对第二个持有者故障？ | 原 E4、后续扫描、34 s 独立验证、完成修复后的正对照 | 可用性取决于替代副本是否就绪；完成修复后可新取全部九片，未建立稳定策略排序 |
| H5：Controller 故障时应用状态与服务能否恢复？ | E5a/b/c，各3次，fresh metadata、任务身份和写入拒绝检查 | 所测试场景保留已提交状态并恢复应用请求；不外推到分区、掉电或 Byzantine |

## 4. 八页分配（包含所有部分，合计8页）

| 部分 | 页数预算 | 内容/产物 |
|---|---:|---|
| 标题、摘要、Introduction | 0.90 | 问题、研究问题、三项贡献与范围 |
| Background / Related work | 0.75 | 约6–10项经原文核实的相关工作及差异 |
| Design / Implementation | 1.40 | 架构图、发布/播放/repair 协议、策略定义、持久状态 |
| Methodology | 0.80 | 配置表、指标与时间原点、配对和采样定义 |
| Evaluation | 2.45 | E1 0.45；E2 0.35；E3+late 0.65；E4 0.55；E5 0.45 |
| Discussion / Limitations / Conclusion | 0.55 | 回答假设、scope evolution、限制与下一步 |
| References | 0.60 | 仅引用正文实际使用且核实的文献 |
| AI Tool Use | 0.20 | 具体用途和验证方式 |
| Team Member Contributions（最后） | 0.35 | 每人具体实现、实验、写作与验证贡献 |

页数为排版目标，可在部分之间微调；不是靠缩小图中文字保证。先写论证和表格，再写摘要，最后控制页数。没有必要把所有原始表格塞进正文。

## 5. 各节写作任务与证据

### Design / Implementation

以 `docs/architecture.md`、`docs/invariants.md`、`docs/reliability.md`、`docs/playback.md` 为导读，核对 `internal/metadata`、`internal/control/coordinator.go`、`internal/storage`、`internal/playback`、`cmd/demo-api` 和 `frontend`。

必须说清：DRAFT→READY 的提交条件；哈希/大小检查；已提交注册与单纯文件存在的区别；故障检测、grace、任务创建、复制和注册的阶段；controller 重启保留任务；repair 与 foreground 共用源限速器。L 是前台请求活跃/空闲驱动的额度控制，不读取浏览器缓冲，也不保证公平份额。60 s age floor 未在主性能研究激活。

展示浏览器的 MSE/fMP4 实现与 headless 模拟消费的区别。cleanup、删除、Peer 状态面板作为实现完整性简述，不挤占研究主线。

### Methodology

3 Controllers、3 Peers，单机多进程；9×128 KiB、每片2 s，总模拟媒体18 s；限速96/144/192 KiB/s；低2/high6 viewers；窗口3/并发2/启动4 s；尝试8 s/会话42 s。给出故障时刻5 s、故障后40 s恢复目标、原观察50 s和晚到扩展100 s。

区分 simultaneous、early-staggered（0/4/12）和 late-staggered（0/20/50），均说明每批人数。主表的 stall 是故障后累计观察卡顿，全会话纳入；startup delay 单独定义。失败会话提前结束使卡顿累计受截断，故须共同报告完成率。

恢复定义为全部受影响分片拥有两个已验证、已注册、可达副本，不称为 Raft majority recovery。写出 one-copy exposure 简式及单位；约2 s采样，因此恢复时间有区间。任务创建时间不能混同故障时间。

### Evaluation：必须包含的具体结果与边界

- E1：高负载 N/OFF/S 完成18/18、7/18、7/18；平均观察卡顿3.20/3.24/4.46 s。第二张局部案例表与总体均值必须明确区分。
- E2：brief 修复0.125/0/0 MiB；persistent 均0.750 MiB；exposure17.42/18.24/20.20 MiB·s。grace 不自动取消已有任务，短故障流量不是只计四秒内传输。
- E3：L 无稳定总体优势；最慢 Peer 故障时 L 相对 F50 减少1.65 s卡顿、恢复晚2.07 s。并列完成率及暴露量，保留采样误差说明。
- Late extension：12个有效场景/72会话，全部完成，晚到批次在恢复后启动；无替代副本实际交付事件，无 OFF 对照，不能归因于 repair 提高完成率。说明时间异常 run 的剔除依据与补跑，数据不与原研究直接合并。
- E4：原4/14/30 s全零只是未覆盖就绪转折。后续48次=30扫描+12独立验证+6正对照，三个阶段分开。34 s验证S/F50/L=2/4、3/4、4/4，但扫描排序相反，不宣称L优胜。完成修复后6/6新取全9片成功，是字节交付，不是完整浏览器播放实验。
- E5：新鲜 metadata 和 pending task 恢复中位数1.52/1.60 s从leader停止计时；quorum恢复后1.04 s计时起点不同。多数派缺失期间9次有效写入均503；只针对已测试故障模型。

### 证据映射

| 用途 | 文件/目录 |
|---|---|
| 主实验设计和结果 | `report/compact-experiment-design.md`、`report/compact-study-results.md` |
| 主实验原始证据 | `results/compact-study/`：summary、sessions、paired contrasts、journals、validation、provenance |
| E4 后续研究 | `report/e4-followup-results.md`、`results/e4-followup/`，尤其 validation-plan 和 all-outcomes |
| 晚到研究 | `report/late-staggered-e3-results.md`、`results/late-staggered-e3-run1/`、`results/late-staggered-e3-replacement/` |
| AI 辅助文献梳理 | `report/literature-review-and-research-focus.md`；每项引用仍需读原文 |
| 初始目标演进 | `docs/proposal/`、`docs/checkpoint/` |
| 既有章节素材 | `report/metadata.md`、`storage.md`、`integration.md`、`evaluation.md` |
| 实际个人贡献核对 | `docs/ownership.md`（只作线索）、Git history、PR/提交、成员确认 |

不要把81个主研究 runs、48个E4后续 trials、12个late有效runs简单当成同分布样本；pilot、异常和补跑另行记账。验证各阶段二进制/源码版本后再解释差异。

## 6. 图表计划

1. 一张架构与两条数据路径图：Controller metadata/Raft、Peer数据、浏览器/Gateway与headless实验。
2. 一张实验配置表和策略表，可合并压缩。
3. E1完成率+卡顿成对图（所有失败保留）；避免只画成功会话。
4. E2流量与exposure双面板或紧凑表，标明不同单位。
5. E3总体对照+局部trade-off；晚到扩展用短段落/小表，不把历史差异画成因果改善。
6. E4三阶段小表，E5场景/检查/恢复时间表。

优先矢量PDF与可编辑图源；每张图保留生成命令、原始CSV、版本和caption。正文不复制低分辨率slides截图，图表不重复同一结论占页。

## 7. 建议写作分工（须确认，不是最终贡献声明）

| 成员 | 建议初稿职责 | 交叉审阅 |
|---|---|---|
| Haoyu | Introduction；metadata/Raft与持久任务；E5；主文整合与USENIX编译 | 核查E1–E3统计口径与控制条件 |
| Tian | Methodology；播放实现；E1–E3与late扩展；指标/图表复现 | 核查E4的新取数据与缓存区别 |
| Yuquan | Peer存储/placement/repair策略；E4后续；修复相关工作 | 核查架构、恢复语义与E5边界 |
| 全员 | Discussion、AI披露、逐人实际贡献确认、最终通读 | 任何实验结论至少另一人检查 |

贡献最后一节每人写实际做过的模块、关键机制、具体实验/验证、论文写作责任；以提交与产物为依据。共同工作具体说明协作，不用虚构比例。姓名按现有文稿为Haoyu Wang、Yuquan Hu、Tian Xia。

AI披露至少涵盖已知使用：代码实现/排错辅助、实验脚本和分析、图表/LaTeX/英文措辞、文献线索和讲稿。逐人确认工具与实际用途；不推断未知模型名或未发生的核验。写清哪些结果由真实进程生成、如何保留原始证据、如何处理测量异常，以及最终责任由团队承担。

## 8. Nov 19–24 工作安排

- **Nov 19**：确认研究问题与目录，冻结证据清单、页数范围；成员认领初稿并提供贡献事实。
- **Nov 20**：完成Design/Method/Evaluation主体和第一版表格；核查每个数字来源，整理bibliography。
- **Nov 21**：完成Introduction/Related work/Discussion，形成完整英文初稿；团队确认贡献和AI使用。
- **Nov 22**：交叉复核所有结论、计时起点、分母与版本；检查E4探索/验证分离、late异常说明；压缩到八页。
- **Nov 23**：在干净目录编译，逐页检查与通读；完成引用、贡献末节和AI披露；冻结提交PDF。
- **Nov 24 10:30 AM前**：最终下载/打开检查与提交准备；课程截止11:59 AM。此为团队内部建议缓冲，不改变课程期限。

不默认运行新的大规模实验。若正文暴露某个关键证据缺口，先判断能否缩小结论；若确需补测，独立记录、说明探索性与额外开销，不在最后一天重写主假设。

## 9. 提交前验收

- [ ] 八页，模板字号/页边距未改，所有图可读，无溢出或未解析引用。
- [ ] 原始数据可定位；每张表分母、单位、时间起点明确；没有把全局平均与单场景混用。
- [ ] 测量无效与应用失败分开；负结果、删失、采样误差和小样本限制明确。
- [ ] 当前实现/已测试能力与早期计划分开；无未支持的“无缝、必然、最优、首次”保证。
- [ ] 学术引用逐项核实；已有论文方法与自身工作差异具体。
- [ ] AI使用披露真实；贡献是全员确认的已完成工作，且为最后一节。
- [ ] 原proposal/checkpoint不被覆盖，最终PDF在`dist/final-report.pdf`。
- [ ] 用独立输出目录编译，完整解析并渲染所有页后发布；在Windows阅读器或下载端再次打开检查。
