# 端到端验收扩展

TODO(Haoyu)：补充浏览器 UI 与服务 API 的联合验收；当前 `make test-e2e` 复用 tests/system/smoke.py。

系统脚本使用真实 Raft/TCP 和 HTTP、临时磁盘及项目二进制。HTTP 单测中的元数据替身仅用于隔离 downloader 行为，不代表 Raft 验收。浏览器真实视频解码不在当前范围。
