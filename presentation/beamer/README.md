# English presentation — editable Beamer source

TODO(team): Keep the editing guide and experimental figures synchronized with the slides.

这是一份标准 LaTeX Beamer 工程，使用 TeX Live 编译，文字、表格和系统示意图均可直接编辑。无需 Python 或 PowerPoint 插件。共 31 页：封面、25 页主体、1 页可选演示、4 页附录。

## 编译

在此目录执行：

```sh
make
# 或直接运行：
latexmk -pdf -interaction=nonstopmode -halt-on-error -outdir=dist main.tex
```

输出：`dist/main.pdf`。已使用 TeX Live 2025/Debian 的 pdfLaTeX 实际编译。需要安装 `latexmk`、Beamer、TeX Gyre Heros 和 TikZ；Ubuntu/Debian 常用安装包为 `latexmk texlive-latex-recommended texlive-latex-extra texlive-fonts-recommended tex-gyre`。

`make watch` 会监听文件变化并重新编译；按 Ctrl+C 退出。也可以在编辑器中将 `main.tex` 设置为主文件，选择 pdfLaTeX/latexmk。

## 从哪里修改

| 文件 | 修改内容 |
|---|---|
| `main.tex` | 标题、作者、日期、章节顺序；所有编译都从这里开始 |
| `theme.tex` | 配色、字体、页脚、边距，以及图示的公共样式 |
| `slides/01-system.tex` | 动机、架构、发布与播放流程、实现情况、演示 |
| `slides/02-policies.tex` | 研究问题、N/OFF/S/F25/F50/L 策略 |
| `slides/03-method.tex` | 实验环境、指标与样本数量 |
| `slides/04-e1.tex` 至 `08-e5.tex` | E1–E5 实验设计、数值与结论 |
| `slides/09-conclusion.tex` | 结论与局限 |
| `slides/10-appendix.tex` | 附录图表 |
| `dist/figures/` | 可独立分发的实验图表副本 |

每个 `frame` 对应一页，没有用绝对坐标摆放正文，也没有生成器覆盖手动修改。编辑后直接重新编译即可。例如添加一页：

```tex
\begin{frame}{Your slide title}
  \begin{itemize}
    \item First observation.
    \item Second observation.
  \end{itemize}
  \takeaway{The message the audience should remember.}
  \note{Speaker notes, hidden in the audience PDF by default.}
\end{frame}
```

可以把它放进已有章节；新增章节文件则在 `main.tex` 中加入 `\input{slides/your-file}`。不要直接编译章节文件。

表格内以 `&` 分列、`\\` 换行。普通文字中的 `%`、`&`、`_`、`#` 分别写成 `\%`、`\&`、`\_`、`\#`。正文为英文，当前字体配置不包含中文支持。

## 演示、讲稿和篇幅

- 在 `main.tex` 中将 `\showdemotrue` 改为 `\showdemofalse`，隐藏可选演示页。
- 取消 `\setbeameroption{show notes}` 的注释，可生成带单独讲稿页的版本；页数会增加。
- 删除或注释不需要的 `frame` 可以缩短展示。不要只隐藏页码，因为内容仍会留在 PDF 中。
- 删除 `\input{slides/10-appendix}` 可以去掉附录。
- 一页过满时优先拆成两页；编译日志中的 `Overfull` 表示需要检查溢出。

## 数据与图片更新

结果依据仓库中的 `report/compact-study-results.md`、`results/compact-study/compact-study-results.md`、`results/compact-study/SLIDE-EXHIBITS.md`。主体数据是可编辑的 LaTeX 表格；附录 PNG 来自 `results/compact-study/figures/`。修改实验结果时同步更新表格、结论与图片，不能仅替换图表。

注意保持口径：单机多进程实验；真实 HTTP 下载但模拟媒体消费；失败会话保留；E5c 主体恢复时间从 quorum 恢复算起，而附录图采用相对故障的时间。不要将这些结果表述为浏览器播放性能或一般化的故障保证。

独立源码包保留了 `dist/figures/`，解压后无需原仓库即可编译。PDF 是展示文件；后续编辑请以 `.tex` 为源文件。

## 字体与版式

当前使用 TeX Gyre Heros 无衬线字体、12 pt Beamer 基准字号。标题采用粗体，内容块用青色标题与浅色背景；列表和表格统一设置间距。字体与公共样式集中在 `theme.tex`，基准字号位于 `main.tex` 的文档类选项中。Beamer 页面尺寸较小，字号不宜直接照搬 PowerPoint 的数值。
