# TODO(team): Maintain the TeX Live build entry point for the presentation.
.PHONY: all watch
all:
	latexmk -pdf -interaction=nonstopmode -halt-on-error -outdir=dist main.tex

# Recompile automatically whenever an included source file changes.
watch:
	latexmk -pdf -pvc -interaction=nonstopmode -outdir=dist main.tex
