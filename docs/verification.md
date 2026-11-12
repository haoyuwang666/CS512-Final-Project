# 基础框架验证记录

TODO(Haoyu)：后续每次修改按风险更新验证证据；Docker、分区和新研究策略需分别验收。

## 本机网页控制台（2026-11-12）

新增 local-console 进程管理入口与网页上传。已运行后端全量 `go test -race ./...`、前端 TypeScript 检查、5 项播放器测试和生产构建。

`tests/system/console.py` 在独立临时数据目录启动真实进程，验证：拒绝单 Peer 配置、拒绝跨来源管理请求、2 Peer + 1 Controller 启动及健康检查、6 秒素材裁剪至约 4 秒、自动切片与双副本导入、经 Media API 从两个 Peer 返回并校验全部分片 SHA-256、停止 Peer 2 后切源、停止/重启后视频库保留、3 Peer + 3 Controller 选主、无效文件失败反馈。测试不接管已有服务。

新增控制台的进程控制只针对本实例的子进程；关闭页面不终止节点。不同规模隔离数据；未实现运行中成员增减、跨管理服务重启的处理中任务恢复、自动副本修复或持久任务协调。此扩展没有使用 Docker。

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

## Playback 扩展复核（2026-11-12）

环境：Linux arm64，Go 1.27.3、Node 26.13.1、pnpm 11.25.0。此条为本次增量记录；上方 macOS 记录保留其当时边界。

已通过：

- `go test -race ./...`：全量后端测试及新增 9 项 playback 测试，无 race 报告。
- 新增覆盖：注入 Clock 后精确的 80 ms stall / 10 ms media / frozen deadline miss；连续启动缓冲、有界 Buffer、部分消费取消；EWMA 与同源队列排序；真实 HTTP Deadline 超时切源；缓存副本隔离、503 回退、404 和取消不回退；损坏载荷流量与失败终态；截断 JSONL 修复、重启恢复、跨 256 事件的 SSE 回放；活跃 Session 不被回收；API 配置验证及持久终态；无效/超前/过期 SSE 游标。
- `make vet smoke`：四个后端程序构建、vet、真实 3 Controller + 3 Peer 的基础链路和故障检查通过，额外执行带启动缓冲的 Deadline Session。
- 前端 `pnpm run build`：TypeScript 与 Vite 生产构建通过。UI 已接入策略/窗口/并发/启动缓冲、实时快照、EventSource 及轮询回退；未执行浏览器自动化交互测试。
- `git diff --check`、`python3 scripts/check_scaffold.py --write-index` 通过。

此次验证不构成 Deadline 性能收益结论；未运行 Docker、正式配对实验或视频解码。缓存只能覆盖已获取且未过期的元数据；Session 日志保留有上限，长期实验应单独归档。完整语义见 playback.md。

## 浏览器媒体交付扩展复核（2026-11-12）

- 全量 `go test -race ./...`、`make vet` 通过。新增 4 项 Media API 测试覆盖：校验后切源与 Range、拒绝 fixture/无效配置、损坏数据不交付、停止播放取消进行中的 Peer 下载。
- 前端 `pnpm test` 的 2 项客户端缓冲测试通过：并发限制、跳转取消、Blob URL 释放及大小/哈希错误拒绝；`pnpm run build` 通过。
- `make smoke` 的真实三 Controller/三 Peer 回归通过。
- `make test-media` 通过：生成带 AAC 声音的 H.264 视频，切片并双副本导入临时集群；停止一个 Peer 后通过 Media API 取回全部 2 个片段，核对大小/SHA-256、检查视频和音频轨道，并对返回字节执行 FFmpeg 完整解码；最后停止媒体 lease。
- 源文件用途索引与 `git diff --check` 通过。

验证边界：此环境没有可用浏览器测试运行器，没有执行真实浏览器的画面截图、自动播放/暂停/跳转/全屏交互或音频设备播放验收。FFmpeg 解码和客户端状态单元测试不能替代浏览器验收。当前独立 MP4 换段允许短暂停顿；不宣称已实现无缝连续流、MSE/HLS 或正式播放性能收益。先前 headless-only 的验证记录保留历史含义，新媒体路径与其分开计量。


## 换段黑闪修复复核（2026-11-12）

前端新增 FrameHold，保留上一段已解码帧直到下一段帧可显示；稳定媒体框尺寸，并提供容器全屏入口。`pnpm test` 共 5 项通过，新增覆盖：不能因 loadedmetadata 提前撤掉画面、等待 seek 完成、连续跳转的过时回调隔离、旧浏览器绘制回退和清理。`pnpm run build`、文件用途检查与 diff 空白检查通过。仍未进行真实浏览器视觉验收；此修复不宣称实现连续音视频无缝拼接。


## 连续 MSE 无缝播放复核（2026-11-12）

此记录取代此前独立 MP4/Canvas 保帧方案的当前状态；历史记录保留。现在使用单次编码的连续 fMP4、唯一 MediaSource URL 与 SourceBuffer，不再逐段 load() 或用 Canvas 遮挡换段。

通过：

- `go test -race ./...`：新增连续 Manifest 的 codec/时长校验和 MediaInfo 缓存深复制测试，原有全量测试通过。
- `make vet smoke`：vet、四个程序构建及真实三 Controller/三 Peer 回归通过。
- `pnpm test` 共 5 项通过：并发预取/跳转取消/JS 载荷回收、损坏数据拒绝、唯一媒体源与串行 append、结束后回跳与回收、时间线索引。TypeScript/Vite 生产构建通过。
- `make test-media-browser` 使用临时 Playwright 与 Chromium 158.0.8127.18，运行真实带声音 8 秒素材的四个片段：只有一次 loadstart，首次 playing 后没有 waiting，实际解码 192 帧。结束后回跳、回收后重新下载、暂停、前跳、停止均通过。
- 通过 MEDIA_MANIFEST 指定统一示例 my-video 对应的连续素材，在独立临时集群重复浏览器验收：11 个片段、21.377777 秒、1280 帧，同样只有一次 loadstart，首次 playing 后没有 waiting，回跳/暂停/前跳/停止通过。
- 两次媒体系统检查均验证所有交付对象的大小、SHA-256、相同初始化信息；去重初始化后组合整段，验证每条轨道相邻 packet 的 DTS 精确等于上一 packet 的 DTS + duration，没有缺口或重叠；所有单片与组合流均通过 FFmpeg 音视频解码。故障 Peer 停止后仍能交付。
- `python3 scripts/check_scaffold.py --write-index`、`git diff --check` 通过。

工具安装在 /tmp/cs512-browser，不是运行时依赖。浏览器检查使用临时集群，不修改现有集群。统一示例的素材目标路径为 .data/media/my-video/segments/；历史测试使用过临时素材目录，命名调整不搬移已有素材。实际使用需准备该路径下的连续 fMP4 清单并重启新版 Controller 与 Demo API，同名旧数据应在隔离实验环境重新导入。

边界：此次浏览器验收为 Chromium；尚未验证 Safari/Firefox、物理扬声器输出或高负载/低带宽网络。测试证明换段不重载、媒体时间线连续，不保证资源不足、跳到未缓冲位置或断网时零等待。

团队示例命名现已统一为 my-video；上方素材名称按统一命名表述，历史测试日志可能保留当时的临时 ID。命名统一不修改已有集群的不可变视频记录。
