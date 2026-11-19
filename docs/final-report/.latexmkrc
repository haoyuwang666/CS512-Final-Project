# TODO(team): Keep final-report build products in dist.
$pdf_mode = 1;
$out_dir = 'dist';
$aux_dir = 'dist';
$pdflatex = 'pdflatex -interaction=nonstopmode -halt-on-error -file-line-error -synctex=1 %O %S';
