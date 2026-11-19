# COMPSCI 512 Final Report

TODO(team): Complete and review the eight-page English report according to PLAN.md before submission.

最终报告目录。当前提供写作计划和可编译的 LaTeX 骨架，**尚不是可提交的报告**。

- [PLAN.md](PLAN.md)：研究主线、八页分配、证据、分工与截止时间安排。
- `final-report.tex`：英文报告主入口，USENIX Letter 双栏、10pt。
- `sections/`：按章节拆分的正文；当前为显式占位内容。
- `references.bib`：待核实的引用库；正文出现引用后启用主文件的 bibliography 两行。
- `assets/`：报告使用的图表和素材。
- `dist/final-report.pdf`：编译输出；当前是带标识的规划骨架 PDF。

在本目录执行 `make` 或 `latexmk final-report.tex`。需要 TeX Live、latexmk、BibTeX。`.latexmkrc` 将 PDF 和中间文件统一放在 `dist/`；LaTeX Workshop 的现有配置兼容。

课程截止时间：**2026-11-24 11:59 AM（America/New_York）**，按用户提供的要求记录。先按八页包含参考文献、AI 使用说明与贡献分工规划；课程未明确参考文献是否豁免，确认前不自行放宽。最后一节必须为详细成员贡献。

`report/` 继续存放研究说明、实验结果和素材，本目录存放最终提交文稿。原 proposal、checkpoint 和 presentation 不因创建本目录而更改。
