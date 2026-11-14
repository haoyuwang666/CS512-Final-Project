# 实验输出

TODO(Tian)：保存配置、原始事件、session 汇总、run 汇总和正式配对分析。大数据另存时记录地址与校验值。

`make experiment-baseline` 产生的是流程验证数据，`formal_experiment=false`；其 seed 目前只是运行标签，没有生成随机工作负载。不能作为五次独立随机实验的研究证据。失败 session 保留在输出中。

## Compact E1–E5 study

`compact-pilot/` contains separate calibration/instrumentation data. `compact-study/` contains the frozen 81-run exploratory study, CSV tables, PNG/SVG figures and raw per-run journals. Rebuild exhibits with `python3 experiments/analyze_repair_playback.py results/compact-study`. The completed `compact-study/`, separate pilot archives, and validation archive are explicitly included in Git so teammates can open the presentation links after cloning. Other generated experiment directories remain ignored. Preserve the source snapshots and binary fingerprints with the data.

Completed: 72 performance runs + 9 controller runs in 67.9 minutes. See `compact-study/SLIDE-EXHIBITS.md`, `compact-study/RESEARCH-FINDINGS.md`, and `report/compact-study-results.md`. These are exploratory real-transfer/headless results; failures and censored recovery are retained.
