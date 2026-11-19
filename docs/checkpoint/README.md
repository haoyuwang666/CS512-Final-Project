# COMPSCI 512 Checkpoint

`dist/checkpoint.pdf` 是英文 checkpoint，沿用 proposal 中的题目方向及三名作者，采用 USENIX 双栏格式，正文与参考文献合计三页。

## 内容及状态

- 研究动机和可检验假设。
- 背景与六篇相关工作的比较。
- 高层架构图、元数据、存储放置、修复、纠删码和播放调度设计。
- 当前进展、实验基线、消融实验、指标和故障验证计划。
- 建议成员分工、截至最终报告的时间安排，以及本次 AI 使用说明。

用户表示实现进展、实验结果和分工尚未确认，因此正文没有声称原型已经完成或给出虚构测量结果。成员分工明确标为建议，Go 技术栈、实验规模、策略参数和里程碑均为计划，而非已落实事实。提交前可根据团队的实际进展更新这几项。

## 编译

在此目录执行：

```powershell
latexmk checkpoint.tex
```

也可将 `checkpoint.tex`、`references.bib` 和 `usenix-2020-09.sty` 一起上传到 Overleaf，选择 pdfLaTeX 编译。架构图用 TikZ 绘制，无外部图片依赖。样式文件直接复制自项目的 `example/usenix-2020-09.sty`，未缩小正文或修改页面边距。

## 文献来源

正文采用编号引用。下列链接用于核对原始资料：

- [USENIX 格式要求与模板](https://www.usenix.org/conferences/author-resources/paper-templates)：Letter 纸型、双栏、7 × 9 英寸正文区域、10pt 字体。
- [HACFS / A Tale of Two Erasure Codes in HDFS](https://www.usenix.org/conference/fast15/technical-sessions/presentation/xia)：快码与紧凑码之间的自适应转换；区别于本项目拟研究的热段副本与冷段纠删码。
- [Quality-Aware Segment Transmission Scheduling in Peer-to-Peer Streaming Systems](https://www2.cs.sfu.ca/~mhefeeda/Papers/mmsys10_segmentScheduling.pdf)：调度问题与近似算法。
- [SPACE](https://doi.org/10.1109/ACCESS.2023.3252365)：预取、边缘缓存及流行度策略；内容核对自仓库 `reference/SPACE_Segment_Prefetching_and_Caching_at_the_Edge_for_Adaptive_Video_Streaming.pdf`。
- [Raft](https://raft.github.io/raft.pdf)：复制日志与元数据一致性。
- [MapReduce](https://research.google/pubs/mapreduce-simplified-data-processing-on-large-clusters/)：周期性汇总设计的背景。
- [When to Hedge in Interactive Services](https://www.usenix.org/conference/nsdi21/presentation/primorac)：冗余请求的收益与拥塞代价。

## 本次检查

PDF 已编译，包含三页 Letter 页面；检查了页面渲染、引用和架构图。没有未解析的引用或 overfull box。模板自身可能输出 `breakurl` 与 `microtype` 提示，不影响此次 PDF 生成。

## 文件组织

- `checkpoint.tex`：正文入口；`references.bib`：文献源文件。
- `usenix-2020-09.sty`：当前使用的原始模板样式。
- `.latexmkrc`：统一输出到 `dist/`，适用于命令行与 LaTeX Workshop。
- `Makefile`：在本目录执行 `make`，等价于 `latexmk checkpoint.tex`。
- `dist/checkpoint.pdf`：保留的提交版 PDF；重新编译会更新此文件。
- `dist/verification/`：旧预览与验证产物；`dist/legacy/`：未使用的旧样式备份，均不提交。

所有命令需在本目录执行。编译中间文件和日志统一进入 `dist/`，Git 只保留该目录下的最终 PDF。此次目录整理原样移动了原 PDF，没有重新生成或改变它。
