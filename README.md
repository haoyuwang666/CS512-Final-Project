# Cooperative Video · 分布式视频传输实验框架

TODO(Haoyu)：维护项目入口、构建约定和功能状态；每个负责人完成任务时同步相关文档。

Go 后端提供 Raft 元数据服务、Peer 分片存储和播放 Session API；React + TypeScript 前端展示 Session 状态与下载指标。下面的本地快速开始使用 **1 个 Controller + 3 个 Peer + Demo API + 前端**，不需要 Docker。

所有命令从仓库根目录开始执行，仓库中已没有 `code/` 这一层。已有本地项目可直接进入项目目录；新环境先克隆：

```bash
git clone https://github.com/haoyuwang666/CS512-Final-Project.git
cd CS512-Final-Project
```

## 1. 准备环境

| 工具 | 版本 / 用途 |
|---|---|
| Go | ≥ 1.24；后端编译与测试，本项目已使用 1.27.4 验证 |
| Node.js | 24；本项目已使用 24.22.1 验证 |
| pnpm | 11.25.0；安装前端依赖和运行 Vite |
| Python | ≥ 3.10；生成测试数据、导入和实验脚本 |
| make | 执行根目录 Makefile 的构建命令 |
| FFmpeg / FFprobe | 可选，仅生成或切分真实视频时需要 |
| Docker + Compose | 可选，仅容器集群启动时需要 |

先确认终端能找到这些命令：

```bash
go version
node -v
pnpm -v
python3 --version
make --version
```

若已经安装 Node.js 和 npm，但提示 `pnpm: command not found`，执行：

```bash
npm install -g pnpm@11.25.0
pnpm -v
```

如果 `node` 或 `npm` 也找不到，先安装 Node.js 24 并重新打开终端，再安装 pnpm。

当前这台 Mac 也可以临时使用 Codex 已下载的工具（仅适用于这些路径存在的本机环境）：

```bash
export PATH="/private/tmp/cs512-toolchain/go/bin:$HOME/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$HOME/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/fallback:$PATH"
```

该设置只对当前终端生效；新开终端若找不到命令，需要重新执行。`/private/tmp` 下的 Go 可能被系统清理，长期开发应自行安装 Go。不要把这些本机路径当成团队成员的安装要求。

Python 脚本当前全部只使用标准库，`requirements.txt` 因此仅包含说明，没有第三方包。它不负责安装 Go、Node.js 或 pnpm。若需要独立 Python 环境，可选执行：

```bash
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r requirements.txt
```

## 2. 首次安装依赖与编译

在仓库根目录执行：

```bash
make setup
make backend
```

`make setup` 下载 Go 依赖，并使用锁文件安装前端依赖；`make backend` 生成 `bin/controller`、`bin/peer`、`bin/client` 和 `bin/demo-api`。已经成功运行过这两条命令，可以直接进入下一步。

## 3. 启动后端

打开 **5 个终端标签页**，每个先进入同一个仓库根目录。当前本机路径为：

```bash
cd /Users/wanghaoyu/Desktop/cs512Project
```

其他机器请替换为自己的仓库路径。以下每个服务会持续运行，各占一个终端；看到监听日志后保留窗口，再打开下一个。

**终端 1：Peer 1**

```bash
HTTP_ADDR=127.0.0.1:9001 PEER_ID=peer-1 DATA_DIR=.data/peer-1 ./bin/peer
```

**终端 2：Peer 2**

```bash
HTTP_ADDR=127.0.0.1:9002 PEER_ID=peer-2 DATA_DIR=.data/peer-2 ./bin/peer
```

**终端 3：Peer 3**

```bash
HTTP_ADDR=127.0.0.1:9003 PEER_ID=peer-3 DATA_DIR=.data/peer-3 ./bin/peer
```

**终端 4：Controller**

```bash
PEERS='[{"peer_id":"peer-1","url":"http://127.0.0.1:9001"},{"peer_id":"peer-2","url":"http://127.0.0.1:9002"},{"peer_id":"peer-3","url":"http://127.0.0.1:9003"}]' ./bin/controller
```

Controller 默认 HTTP 地址为 `127.0.0.1:8001`，Raft 地址为 `127.0.0.1:7001`。启动后等待数秒，让单节点完成选主。

**终端 5：Demo API**

```bash
./bin/demo-api
```

Demo API 默认监听 `127.0.0.1:8080`，通过 Controller 查询元数据，供前端创建和查询 Session。服务直接读取进程环境变量，**不会自动加载 `.env` 或 `.env.example`**。

这个模式适合本地前后端联调；三节点 Controller 高可用演示请使用下方 Docker 配置或参考 `docs/demo.md`。

## 4. 导入测试数据并启动前端

打开 **终端 6**，进入仓库根目录，然后执行：

```bash
python3 scripts/media/fixture.py
python3 scripts/ingest/ingest.py .data/fixture/manifest.json --controllers http://127.0.0.1:8001

cd frontend
pnpm dev
```

打开 **http://localhost:5173**，输入视频 ID `fixture`，点击 **“开始 Session”**。Vite 会把 `/api` 请求代理到 `http://127.0.0.1:8080`，因此 Demo API 也必须保持运行。Vite 默认使用 5173；若端口被占用，请打开终端实际打印的地址。

预期能看到 Session 指标和真实后端事件。`fixture` 只有 4 个短分片，Session 很快结束属于正常现象。它是确定性二进制测试数据，不是可播放的视频；目前前端展示传输与模拟消费过程，不提供视频画面或音频解码。

数据已成功导入且后端仍在运行时，以后只需从根目录执行以下命令启动前端：

```bash
cd frontend
pnpm dev
```

如果只想先查看页面，也可以直接启动前端；创建 Session 需要后端与已导入的数据。

### 不使用 Vite 的前后端合并访问

先停止正在运行的 Demo API，在根目录构建前端，再启动 API：

```bash
make frontend
./bin/demo-api
```

此时直接打开 **http://localhost:8080**，无需 `pnpm dev`。Controller 和 Peer 仍需运行。前端源文件修改后需要重新执行 `make frontend`。

## 5. 停止与再次启动

在前端、Demo API、Controller 和 3 个 Peer 的各自终端按 `Ctrl+C` 即可停止。数据保存在 `.data/`，停止服务不会清空数据。下次使用相同配置启动后，通常无需重新生成或导入已经持久化的 `fixture`。

常见问题：

| 现象 | 处理方式 |
|---|---|
| `pnpm: command not found` | 按环境准备部分安装 pnpm，或在当前终端设置本机工具路径 |
| `go: command not found` | 安装 Go 并加入 PATH，或使用上述本机临时 Go 路径 |
| `./bin/peer: No such file or directory` | 回到仓库根目录，先运行 `make backend` |
| 页面能打开，创建 Session 失败 / 代理连接被拒绝 | 确认 Demo API 在 8080、Controller 在 8001、Peer 在 9001–9003 运行，且测试数据已经导入 |
| 导入时暂时找不到 Leader | 等待 Controller 选主完成，再重试导入；导入工具自身也会重试 |
| `address already in use` | 检查并停止之前启动的同一个服务，避免重复运行 |

## 验证与测试

在仓库根目录执行：

```bash
make vet test test-race
make build
make smoke
python3 scripts/check_scaffold.py
```

`make smoke` 自行启动临时的 3 Controller + 3 Peer，生成短时长字节素材、导入、模拟播放、停止一个 Peer、停止 Leader、重启 Controller、验证无多数派写入失败；最终关闭其进程。临时目录自动清理，不依赖已有集群，也不会留下长期运行的前后端。`make test-raft` 和 `make test-e2e` 目前复用这项检查，尚不是完整分区测试套件。

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

### 上传自己的视频

原视频统一放在项目根目录的 `.data/uploads/`。当前 Mac 的完整路径是 `/Users/wanghaoyu/Desktop/cs512Project/.data/uploads/`；其他机器使用自己的项目路径。`.data` 是隐藏目录，Finder 中按 `Command + Shift + .` 可显示隐藏文件，也可以按 `Command + Shift + G` 输入完整路径前往。

**目前没有自动监控目录或前端上传按钮。复制文件后，需要手动切片并执行导入。** 以下以 `my-video.mp4`、视频 ID `my-video` 为例；需要本机已安装 FFmpeg、FFprobe 和 Python 3。

1. 在项目根目录创建入口目录，将自己的视频复制进去，并命名为 `my-video.mp4`：

   ```bash
   mkdir -p .data/uploads
   # 将视频复制到 .data/uploads/my-video.mp4
   ```

2. 将原视频切成约 2 秒的独立 MP4 片段，生成导入清单：

   ```bash
   bash scripts/media/segment_video.sh \
     .data/uploads/my-video.mp4 \
     .data/media/my-video/segments
   python3 scripts/media/generate_manifest.py \
     .data/media/my-video/segments \
     --video-id my-video
   ```

   当前切片脚本会重新编码为 H.264，并去掉音轨。输出目录包含 `segment-*.mp4`、供上传使用的 `my-video--*.bin` 和 `manifest.json`；`.bin` 内容就是对应 MP4 片段的字节。不同视频应使用不同的视频 ID 和输出目录；ID 只允许字母、数字、下划线和连字符，长度为 1–96 个字符。

3. 如果使用 Docker 集群，先在 Docker Desktop 运行的情况下启动服务（已启动则跳过），然后导入：

   ```bash
   make up
   make ingest MANIFEST=.data/media/my-video/segments/manifest.json
   ```

   如果使用前文的本机单 Controller 模式，保持各服务运行，改用：

   ```bash
   python3 scripts/ingest/ingest.py \
     .data/media/my-video/segments/manifest.json \
     --controllers http://127.0.0.1:8001
   ```

4. 导入成功会输出 `status: READY` 的 JSON。在前端输入视频 ID `my-video`，即可启动下载与缓冲模拟 Session；当前不显示视频画面或播放音频。

文件存放位置：

| 内容 | 位置 |
|---|---|
| 原视频 | 本机项目目录 `.data/uploads/my-video.mp4`，导入不会移动原文件 |
| 准备好的片段与清单 | 本机项目目录 `.data/media/my-video/segments/` |
| Docker 模式下的片段副本 | 各 Peer 独立的 named volume，挂载到各自容器的 `/data/`；存为 `<sha256>.blob`，由 `index.json` 索引 |
| 本机模式下的片段副本 | 按前文配置，分别在 `.data/peer-1/`、`.data/peer-2/`、`.data/peer-3/` |
| 视频清单、发布状态和副本位置 | Controller 的持久化数据中 |

导入工具会将每段上传到两个不同 Peer，并登记元数据；不要直接把 MP4 复制到 Peer 的 `/data`，那样不会完成索引与登记。Docker 的三个 `/data` 相互独立，也不对应本机 `.data/peer-*`。`make down` 保留数据卷，删除卷会丢失其中的副本。`.data/` 已被 Git 忽略，团队成员首次使用时需自行创建目录；不要将视频素材提交到仓库。

## 团队开发入口

- Haoyu：`internal/metadata/`、`internal/control/`、公共协议、部署和集成。下一步是持久容量预留和任务生命周期。
- Yuquan：`internal/storage/`、`internal/placement/`、`internal/repair/`、素材和导入。下一步是 Inventory、Heartbeat 和 Repair 执行。
- Tian：`internal/playback/`、`frontend/`、`experiments/`。下一步是准确 deadline/吞吐估计、完整事件数据与实验重置。

接手代码应先运行现有验证，再认领 TODO；不要重写另一套 Raft、导入器或指标实现。目录的具体负责人见 `docs/ownership.md`。

## 使用范围

只适用于可信 LAN/VPN 或本机实验。没有身份认证、TLS、Byzantine 防护或面向互联网的部署配置。无自动删除；空间不足明确拒绝。不要把未实现优化的 baseline 结果当作研究收益。
