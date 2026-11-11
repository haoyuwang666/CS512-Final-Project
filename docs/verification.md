# 基础框架验证记录

TODO(Haoyu)：后续每次修改按风险更新验证证据；Docker、分区和新研究策略需分别验收。

验证日期：2026-11-11。环境：macOS arm64；Go 1.27.4（本次会话临时工具，不要求团队使用这个临时路径）、Node 24.22.1、pnpm 11.25.0、Python 3.12、系统 FFmpeg/FFprobe。

已执行：

- Go 四个命令实际构建；`go vet ./...`。
- `go test -race ./...`：FSM 双副本门槛、幂等与快照恢复；Peer 坏 Hash、Quota、重启、操作冲突；真实 HTTP 校验/切源；模拟 stall 与取消。
- 前端 `pnpm run build`：TypeScript 检查及 Vite production build。
- `tests/system/smoke.py`：真实 3 Controller + 3 Peer；双副本导入、重复导入、CLI 播放、Session REST/SSE、未实现策略拒绝、Peer 故障切源、Leader 故障恢复、Controller 重启、无多数派写入拒绝。
- FFmpeg 真实 12 秒素材生成、约 2 秒分段、FFprobe 时长和 SHA-256 Manifest 生成。
- baseline runner 的真实 Session 输出、CSV 聚合和诊断 SVG 流水线；仅验证程序运行，不代表正式研究重复实验。
- Python 语法、配置 JSON、Shell 语法、Compose YAML 解析、文件用途 TODO 审计。

验证边界：本机未安装 Docker，未执行镜像构建、Compose 启动或 GitHub CI；YAML 可解析不等于容器已跑通。没有验证持续网络分区、Repair、持久容量 Reservation、Deadline、自适应策略、编码、真实解码或正式配对实验。系统测试数据是很小的确定性字节素材，不能作为性能研究结果。

`make setup/test/build/smoke` 是团队复核入口；README 说明了当前可用范围。自动检查生成的 go.sum 保持标准格式，其用途在 FILE_INDEX 中单独记录。
