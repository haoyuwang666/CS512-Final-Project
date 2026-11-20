# COMPSCI 512 Final Report

TODO(team): Complete and review the eight-page English report according to PLAN.md before submission.

最终报告目录。第 1–6 节、AI Tool Use 和成员贡献均已有英文初稿，包含十篇论文的引用、公式与可编辑 TikZ 架构图。摘要仍为占位，全文仍需团队最终复核，**尚不是可提交的报告**。

- [PLAN.md](PLAN.md)：研究主线、八页分配、证据、分工与截止时间安排。
- `final-report.tex`：英文报告主入口，USENIX Letter 双栏、10pt。
- `sections/`：按章节拆分的正文；01–08 已有初稿，00 摘要待补全。
- `references.bib`：十篇已核对的论文，均在 Section 2 实际引用；注释记录原文依据与版本，主文件已启用 BibTeX。
- `assets/`：报告使用的图表和素材。
- `dist/final-report.pdf`：编译输出；当前是带标识的部分初稿预览，并非最终八页报告。

在本目录执行 `make` 或 `latexmk final-report.tex`。需要 TeX Live、latexmk、BibTeX。`.latexmkrc` 将 PDF 和中间文件统一放在 `dist/`；LaTeX Workshop 的现有配置兼容。

也可执行 `tectonic --outdir dist --keep-logs final-report.tex`。已对 XeTeX/Tectonic 禁用模板中仅适用于 pdfTeX 的 microtype spacing/kerning，并让 XeTeX 使用 hyperref 的原生 URL 换行以避免 breakurl 的 PostScript 输出；标准 pdfLaTeX 排版参数保持不变。

课程截止时间：**2026-11-24 11:59 AM（America/New_York）**，按用户提供的要求记录。先按八页包含参考文献、AI 使用说明与贡献分工规划；课程未明确参考文献是否豁免，确认前不自行放宽。最后一节必须为详细成员贡献。

`report/` 继续存放研究说明、实验结果和素材，本目录存放最终提交文稿。原 proposal、checkpoint 和 presentation 不因创建本目录而更改。
