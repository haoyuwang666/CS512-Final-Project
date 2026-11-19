# COMPSCI 512 Proposal

TODO(team): Maintain the proposal source layout and build instructions.

## 文件组织

- `proposal.tex`：正文入口。
- `references.bib`：独立文献源文件，从原正文内嵌文献原样提取；后续在此编辑。
- `assets/architecture.png`：架构图，来自原 `arch.png`，图片内容未改变。
- `usenix2019_v3.sty`：正文实际使用的原始模板样式。
- `.latexmkrc`：统一编译配置；`Makefile`：可选的一条命令入口。
- `dist/proposal.pdf`：保留的提交版 PDF。

## 编译

在本目录运行：

```sh
latexmk proposal.tex
# 或
make
```

需要 TeX Live 的 pdfLaTeX、BibTeX 和 latexmk。输出 PDF、日志及中间文件均在 `dist/`，Git 只保留最终 PDF。LaTeX Workshop 的现有 `%DIR%/dist` 配置使用相同位置；编辑时以 `proposal.tex` 为主文件。

此次目录整理原样移动了原 PDF，没有重新生成或改变它。后续编译会更新 `dist/proposal.pdf`。在 Overleaf 使用时上传 `.tex`、`.bib`、`.sty` 与 `assets/` 即可，平台输出位置由平台管理。
