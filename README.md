# Cooperative Video · 可扩展项目基础框架

TODO(Haoyu)：维护项目入口、构建约定和功能状态；每个负责人完成任务时同步相关文档。

所有开发文件统一位于本目录。先读 `docs/ownership.md`，再读 `docs/api.md`。完整文件用途和 TODO 索引见 `FILE_INDEX.md`。仓库根目录即开发入口；计划材料和课程提交资料不纳入代码仓库。

## 已有基础代码与边界

| 能力 | 当前状态 |
|---|---|
| 真实 HashiCorp Raft / BoltDB / FileSnapshotStore | 已实现基础接入，可启动单节点或固定三节点集群 |
| Draft → 双副本确认 → READY | 已实现；修改都经 Raft FSM；Controller 向配置中的 Peer 核对持久对象元信息 |
| Peer 磁盘存储 / SHA-256 / Quota / 幂等上传 | 已实现；内容寻址不可变数据 + 持久索引；保守保留孤儿文件 |
| Peer 共享上传限速 / 已写出字节统计 | 已实现基础版本；计量在应用 ResponseWriter 层，不是 TCP 实际到达字节 |
| 静态双副本放置 / Python 导入 | 已实现；放置建议会扣除本次计划预计空间，但没有跨请求持久预留 |
| Headless Round Robin / 并发预取 / 校验 / 失败切源 | 已实现；真实 HTTP 与模拟媒体消费，默认首段即可启动 |
| Session REST / SSE / React 单页 | 已实现基础版本；前端用快照轮询，完整指标在结束后更新 |
| Heartbeat / Inventory / 任务 API | 501，未实现；Peer 不会虚假上报成功 |
| Repair / 自适应放置 / Deadline 策略 | 接口与 TODO 已建立，未实现；不能用于正式对比 |
| 正式配对实验 / Hedging / Coding / 真实解码 | 未实现；baseline runner 仅检查数据流水线 |

这是基础框架，不是完整项目交付。当前固定三节点集群是实际共识服务，不使用内存 Map 假扮 Raft；FSM 内存视图由真实日志复制和持久恢复驱动。

## 项目结构

```text
cmd/                 controller / peer / client / demo-api 四个入口
internal/domain/     Manifest、Receipt、Location、Task 公共模型
internal/protocol/   DTO、错误码、HTTP 与生命周期基础工具
internal/metadata/   FSM、Raft、元数据 HTTP
internal/control/    任务协调扩展点
internal/storage/    Peer 持久存储与 HTTP
internal/placement/  静态放置和自适应扩展点
internal/repair/     修复决策接口
internal/playback/   下载、调度、逻辑 Buffer、时钟、Metrics、Session
frontend/            React + TypeScript + Vite
scripts/media/       二进制测试素材与真实视频生成、切分、Manifest
scripts/ingest/      复用公共协议的导入工具
experiments/         baseline runner、故障入口、聚合、诊断图
deployments/         多节点 Compose 与 Dockerfile
tests/integration/   真实 HTTP 集成测试
tests/system/        不依赖 Docker 的真实多进程系统测试
tests/e2e/           演示验收扩展说明
docs/                协议、架构、不变量、分工、开发说明
results/             生成的运行数据（默认忽略）
report/              按负责人拆开的报告章节
```

相对 plan：公共 API 工具统一为 `internal/protocol/`，不再创建职责重复的 `internal/api/`；脚本按分工拆到 `scripts/media/` 与 `scripts/ingest/`。`control` 负责组合，不让策略包改 FSM。

## 本地依赖

- Go ≥ 1.24；当前编译/测试使用 Go 1.27.4。
- Node.js 24、pnpm 11.25.0；本次使用 Node 24.22.1 验证。前端固定依赖及 `pnpm-lock.yaml`。与原 plan 的 npm 选择不同，仓库统一使用 pnpm，不混用 lockfile。
- Python ≥ 3.10；测试、导入和基础分析只使用标准库。
- FFmpeg / FFprobe 只用于真实媒体生成。二进制冒烟测试不需要它们。
- Docker + Compose 用于容器演示；本机实际环境没有 Docker，因此容器镜像和 Compose 运行尚未验证。

```bash
make setup
make vet test test-race
make build
make smoke
```

`make smoke` 自行启动临时的 3 Controller + 3 Peer，生成短时长字节素材、导入、模拟播放、停止一个 Peer、停止 Leader、重启 Controller、验证无多数派写入失败；最终关闭其进程。临时目录自动清理。不依赖已有集群，不删除项目数据。`make test-raft` 和 `make test-e2e` 目前复用这项检查，尚不是完整分区测试套件。

## Docker 演示入口（配置已提供，待在有 Docker 的机器验证）

```bash
make up
make fixture
make ingest
# 打开 http://localhost:8080，使用 fixture 启动 Session
make experiment-baseline
make plot
make down
```

Controller 对外端口 8001–8003；Peer 9001–9003；Dashboard 8080。只绑定主机 loopback。Raft 端口只在容器网络使用。初始选主需要数秒；导入工具会重试。

容器返回的 Peer 地址是容器 DNS，因此 `make ingest` 和 `make experiment-baseline` 在同一个 Compose 网络中运行，不能直接把返回的地址拿到宿主机下载。实验容器的挂载目录需要可写；Linux 可设置 `LOCAL_UID`、`LOCAL_GID` 为当前用户 ID。

真实视频素材：

```bash
make media
make ingest MANIFEST=.data/media/segments/manifest.json
# Dashboard 中改用 demo-video
```

`fixture` 是确定性二进制协议测试数据，不是视频；`demo-video` 是 FFmpeg 生成的真实 MP4 片段。两者都通过相同传输代码，但本框架均不解码。独立 MP4 不能直接拼接成无缝浏览器流。

本机手动运行服务的配置见 `docs/demo.md`。前端开发在已有 Demo API 上执行 `cd frontend && pnpm dev`，默认端口 5173 代理到 8080。

## 团队开发入口

- Haoyu：`internal/metadata/`、`internal/control/`、公共协议、部署和集成。下一步是持久容量预留和任务生命周期。
- Yuquan：`internal/storage/`、`internal/placement/`、`internal/repair/`、素材和导入。下一步是 Inventory、Heartbeat 和 Repair 执行。
- Tian：`internal/playback/`、`frontend/`、`experiments/`。下一步是准确 deadline/吞吐估计、完整事件数据与实验重置。

接手代码应先运行现有验证，再认领 TODO；不要重写另一套 Raft、导入器或指标实现。目录的具体负责人见 `docs/ownership.md`。

## 使用范围

只适用于可信 LAN/VPN 或本机实验。没有身份认证、TLS、Byzantine 防护或面向互联网的部署配置。无自动删除；空间不足明确拒绝。不要把未实现优化的 baseline 结果当作研究收益。
