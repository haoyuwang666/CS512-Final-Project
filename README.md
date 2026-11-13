# Cooperative Video · 分布式视频传输实验框架

TODO(Haoyu)：维护项目入口、构建约定和功能状态；每个负责人完成任务时同步相关文档。

Go 后端提供 Raft 元数据服务、Peer 分片存储和播放 Session API；React + TypeScript 前端支持浏览器视频播放，也可展示 headless Session 状态与下载指标。下面的本地快速开始使用 **1 个 Controller + 3 个 Peer + Demo API + 前端**，不需要 Docker。

## 推荐：本机网页控制台

存储与修复扩展：已接入 Heartbeat/校验库存、Raft 持久修复任务、资源感知放置和隔离配对实验。
运行与边界见 [docs/reliability.md](docs/reliability.md)。原生 Windows 请使用 WSL；浏览器播放已在 Edge 手工验证。

安装 Go ≥ 1.24、Node.js、pnpm 11.25.0、Python 3、FFmpeg/FFprobe 后，在项目根目录只需运行：

```bash
make console
```

打开 **http://127.0.0.1:8090**，选择 2–8 个 Peer 和 1 或 3 个 Controller，点击“启动集群”。状态表会查询实际健康状态、Raft 角色、端口、分片对象数、存储字节和 Peer 累计发送量。等待集群就绪，再选择视频文件并点击“上传并切片导入”。可选填写起始秒数与保留时长；留空时长表示保留到结尾。上传最大 512 MB，处理过程展示真实阶段和错误，不伪造编码进度百分比。导入成功后，在视频库点击“播放”即可直接加载，无需填写视频 ID 或参数；下载策略和缓冲参数收在播放器的“高级设置”，默认保持不变即可。播放器显示分片来源、校验字节、重试与实际播放等待。

管理服务启动后保留一个终端即可，网页不直接执行用户输入的命令。管理端口为 8090，Demo API 为 18080，Controller HTTP 为 18001 起、Raft 为 17001 起，Peer 为 19001 起。此模式仅允许本机访问，不会接管、停止原有 8001/8080/9001–9003 手动服务。集群数据存储于 `.data/console/p数量-c数量/`；同一规模重启保留视频，不同规模使用独立数据。新上传视频自动生成唯一 ID，上传不会覆盖旧记录。原手动模式的 sample-5s 等视频不会自动出现在新集群，请通过网页重新上传原文件。

更改节点数量需先停止集群；这不是运行中动态扩缩容。网页停止只停止控制台创建的子进程，保留磁盘数据。退出管理服务会终止其子进程；页面关闭不会停止集群。上传处理期间禁止停止集群。重启时未完成任务不自动续传，可重新上传。视频文件应来自可信本机用户。

开发前端时，先运行 `./bin/local-console`，再在 frontend 中运行 `pnpm dev`；Vite 默认代理到 8090。传统手动模式需要用 `API_TARGET=http://127.0.0.1:8080 pnpm dev` 连接原 Demo API。

真实流程回归：`make backend` 后运行 `python3 tests/system/console.py`，使用独立临时目录，测试前需确保没有控制台占用 8090。

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

打开 **5 个终端标签页**，每个先进入自己克隆的 `CS512-Final-Project` 仓库根目录。以下服务会持续运行，各占一个终端；看到监听日志后保留窗口，再打开下一个。

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

## 4. 导入视频并启动前端

### A. 播放真实视频（默认模式）

保持第 3 节的后端运行，打开 **终端 6**，进入仓库根目录，首次生成并导入带声音的 12 秒测试视频：

```bash
make media
python3 scripts/ingest/ingest.py \
  .data/media/segments/manifest.json \
  --controllers http://127.0.0.1:8001

cd frontend
pnpm dev
```

`make media` 需要 FFmpeg/FFprobe，会单次编码 H.264/AAC，生成用于无缝播放的连续 fMP4 分片。输出目录已有 `segment-*.mp4` 时脚本会拒绝覆盖：已有新版清单可以直接导入；旧的独立 MP4 请按下方“升级与旧视频迁移”使用新目录重新生成。

打开 **http://localhost:5173**，选择 **“视频播放（画面与声音）”**，将视频 ID 改为 **`demo-video`**，点击 **“播放视频 / 重新加载”**。原生控件支持整段进度、暂停、音量和全屏；页面下方进度条也可跳转。若页面提示需要手动开始，点击播放器里的播放按钮。

播放器通过同一个 MediaSource/SourceBuffer 连续追加分片，换段不重新加载视频源。使用支持清单中 H.264/AAC codec 的 MSE 浏览器，并通过 `localhost` 或 HTTPS 打开页面，让浏览器能执行 SHA-256 校验。网络供给不足或跳到未缓冲位置时仍可能等待。

Vite 把 `/api` 代理到 `http://127.0.0.1:8080`，因此 Demo API 也必须保持运行。Vite 默认端口 5173；若被占用，请打开终端实际打印的地址。

播放自己的视频见下方“使用自己的视频”。默认输入框为 `my-video`，需要先导入同名视频；它不是新环境自带的数据。

### B. Headless 模拟实验（不需要 FFmpeg）

用确定性字节素材验证分布式传输与模拟消费，在根目录执行：

```bash
python3 scripts/media/fixture.py
python3 scripts/ingest/ingest.py \
  .data/fixture/manifest.json \
  --controllers http://127.0.0.1:8001
```

前端仍用 `cd frontend && pnpm dev` 启动。页面选择 **“Headless 模拟实验”**，输入 **`fixture`**，点击 **“开始模拟 Session”**。`fixture` 只有 4 个短分片，结束很快是正常现象。它不是视频，不能在视频模式解码。此模式显示 Go 引擎的模拟指标；真实播放模式显示浏览器的实际媒体状态。

数据已导入、后端仍运行时，以后只需在根目录执行 `cd frontend && pnpm dev`。只查看页面无需后端；创建播放或模拟 Session 需要已导入的数据及后端服务。

### 不使用 Vite 的前后端合并访问

先停止正在运行的 Demo API，在根目录构建前端，再启动 API：

```bash
make frontend
./bin/demo-api
```

此时直接打开 **http://localhost:8080**，无需 `pnpm dev`。Controller 和 Peer 仍需运行。前端源文件修改后需要重新执行 `make frontend`。

## 5. 停止与再次启动

在前端、Demo API、Controller 和 3 个 Peer 的各自终端按 `Ctrl+C` 即可停止。数据保存在 `.data/`，停止服务不会清空数据。下次使用相同配置启动后，通常无需重新生成或导入已经持久化的新版视频或 `fixture`。

常见问题：

| 现象 | 处理方式 |
|---|---|
| `pnpm: command not found` | 按环境准备部分安装 pnpm |
| `go: command not found` | 安装 Go 并加入 PATH |
| `./bin/peer: No such file or directory` | 回到仓库根目录，先运行 `make backend` |
| 页面能打开，创建 Session 失败 / 代理连接被拒绝 | 确认 Demo API 在 8080、Controller 在 8001、Peer 在 9001–9003 运行，且测试数据已经导入 |
| 导入时暂时找不到 Leader | 等待 Controller 选主完成，再重试导入；导入工具自身也会重试 |
| `metadata HTTP 404` / `video not found` | 确认输入 ID 与清单的 video_id 相同，并已向 Demo API 使用的 Controller 导入成功；仅复制原视频或生成分片还不算导入 |
| `REPACKAGE_REQUIRED` | 旧独立 MP4 需要重新生成连续 fMP4；团队示例统一为 my-video，同名旧数据需隔离实验环境，见下一节 |
| `UNSUPPORTED_MEDIA` | `fixture` 只能用于 Headless 模拟实验，视频模式需导入真实连续 fMP4 |
| `unknown field "media"` | 运行中的 Controller 仍是旧二进制；重新编译并按原配置重启所有 Controller 和 Demo API |
| 输出目录已有分片 | 改用新目录；已有有效清单可直接导入，不必再次切片 |
| 启动缓冲超过预取窗口 | 降低启动缓冲时长，或增大窗口；0 表示首段准备好即可启动 |
| Web Crypto / MSE 格式不支持 | 用 localhost 或 HTTPS 访问，并使用支持清单 codec 的 MSE 浏览器 |
| `address already in use` | 检查并停止之前启动的同一个服务，避免重复运行 |

## 升级与旧视频迁移到无缝播放

团队文档、前端默认值和实验示例统一使用视频 ID **`my-video`**。新版清单新增 `media` 信息，真实播放需要 `continuous-fmp4-h264-v1` 或 `continuous-fmp4-h264-aac-v1`。旧 `independent-mp4-*` 分片需要从原始文件重新生成。

如果原集群已发布旧格式的 `my-video`，该 ID 和 Peer 分片索引不能直接覆盖。请保留原数据，在独立、干净的实验环境中重新生成并导入统一的 `my-video`；Controller、全部 Peer 的数据目录和素材输出目录均需隔离，不能只清空或更换 Controller。以下命令针对没有同名旧数据的环境。

以下 1–4 步用于**本机模式**；Docker 模式按第 3 步准备新分片后，使用本节末尾的 Docker 更新与导入命令。

1. 在旧 Controller 和 Demo API 的终端按 `Ctrl+C`，然后从仓库根目录重新编译：

   ```bash
   make backend frontend
   ```

2. 按第 3 节或自己原有的配置重启 Controller 和 Demo API，没有同名旧数据时继续使用原来的 `DATA_DIR`、`PEERS` 和 Raft 配置；有同名旧数据时按上面的要求使用隔离环境。三 Controller 集群需更新并重启全部 Controller，再写入新版清单。

3. 从原视频生成连续分片，统一使用视频 ID `my-video`（输出目录应为空）：

   ```bash
   bash scripts/media/segment_video.sh \
     .data/uploads/my-video.mp4 \
     .data/media/my-video/segments
   python3 scripts/media/generate_manifest.py \
     .data/media/my-video/segments \
     --video-id my-video
   ```

   **若该目录已存在本次生成的新版 `manifest.json`，跳过此步，直接导入。** 新克隆的仓库不包含 `.data` 素材。

4. 本机单 Controller 模式导入：

   ```bash
   python3 scripts/ingest/ingest.py \
     .data/media/my-video/segments/manifest.json \
     --controllers http://127.0.0.1:8001
   ```

   成功应输出 `"status": "READY"`。刷新前端，选择“视频播放”，输入 **`my-video`** 并开始播放。Vite 模式刷新页面即可；8080 合并访问使用第 1 步已构建的前端。

Docker 模式用 `make up` 重建并更新服务，然后执行：

```bash
make ingest MANIFEST=.data/media/my-video/segments/manifest.json
```

导入必须与播放使用同一套 Controller。旧切片中删除的声音只能从含音轨的原视频恢复，不能靠修改清单或重命名分片恢复。

## 验证与测试

在仓库根目录执行：

```bash
make vet test test-race
make build
make smoke
make test-player
make test-media
python3 scripts/check_scaffold.py
```

`make smoke` 自行启动临时的 3 Controller + 3 Peer，生成短时长字节素材、导入、模拟播放、停止一个 Peer、停止 Leader、重启 Controller、验证无多数派写入失败；最终关闭其进程。临时目录自动清理，不依赖已有集群，也不会留下长期运行的前后端。`make test-raft` 和 `make test-e2e` 目前复用这项检查，尚不是完整分区测试套件。

`make test-player` 检查客户端有界预取、跳转取消、SourceBuffer 串行更新和媒体源复用。`make test-media` 需要 FFmpeg/FFprobe，启动独立临时集群，验证带声音视频的导入、故障切源、播放会话中 Leader 下线后的分片获取、哈希、音视频时间戳连续性和解码；不修改已有集群。

### 可选：真实浏览器验收

从仓库根目录安装独立的测试工具并运行：

```bash
npm install --prefix .data/browser-tools playwright
export PLAYWRIGHT_MODULE="$PWD/.data/browser-tools/node_modules/playwright/index.mjs"
export PLAYWRIGHT_BROWSERS_PATH="$PWD/.data/browser-tools/browsers"
.data/browser-tools/node_modules/.bin/playwright install chromium
make test-media-browser
```

若 Playwright 提示缺少系统依赖，按其提示安装后重试。工具仅用于测试，不是应用运行时依赖。验收会启动临时集群，在 Chromium 中验证完整播放只有一次媒体源加载、跨分片无新增等待，以及暂停、结束后回跳、前跳和停止；随后清空网关缓存，单独验证正在播放时杀掉 Leader 后，无需重新加载即可恢复并播放完毕。

播放链路会对 Controller 连接失败、5xx 和 429 进行退避重试，每次元数据查询最多等待 8 秒。耗尽后返回可重试的 503，前端保留当前 MediaSource 和缓冲，最多再请求两次；停止或跳转会取消相关等待。404、409、无效响应和分片校验失败不会按选主故障重试。若选举耗时超过已缓冲时长，可能短暂卡顿；多数派长期不可用时仍会在重试耗尽后报错。

如需检查本文已生成的带音轨短视频 `my-video`，沿用上述环境变量后执行：

```bash
BROWSER_TEST=1 \
MEDIA_MANIFEST=.data/media/my-video/segments/manifest.json \
python3 tests/system/media.py
```

此命令把指定清单导入临时集群进行检查，不依赖当前正在运行的服务。

## 从控制台删除视频

视频库中点击“删除视频”，确认后会停止当前控制台的播放并提交删除请求。删除不可撤销，范围包括该视频所有已登记的 Peer 分片副本，以及控制台保存的源视频、转码产物和视频库记录。

Controller 通过 Raft 将视频标为 `DELETING` 并登记逐副本删除任务，停止发布 Manifest 和分片位置。Peer 离线时页面保留视频并显示删除进度；恢复节点后继续。若该视频已有修复或冗余清理任务，删除会等待它们结束再提交。全部副本确认删除后变为 `DELETED`，控制台轮询会移除本地文件和列表项。控制台重启后，启动原集群配置并打开页面即可继续处理本地删除记录。

重复请求不会重复删除；共享文件仍被其他视频引用时会保留。Raft 中保留删除标记和任务历史，原视频 ID 不再用于重新导入。已缓冲的播放内容可能继续短暂显示。失败导入产生的未登记孤儿对象不属于此功能的清理范围。

```bash
make test-delete-video
```

## 手动清理冗余副本

Peer 恢复后，在视频库选择视频，展开“清理当前视频的冗余副本”，点击“预览清理计划”。核对保留的两份副本和删除位置后，点击“执行此清理计划”。故障测试台的“副本维护任务”显示进度。

每次计划对每个分片最多清理一份；存在更多冗余时，完成后再次预览。清理按全体视频的已确认副本逻辑字节量估算节点负载，先处理较大的分片，优先删除负载较高节点上的冗余副本；每生成一条计划就扣减预计负载。负载相同时按 Peer ID 决定顺序，保留的两份副本均须实际验证。该策略尽量均衡现有副本，不迁移数据；分片大小、已有分布和节点健康状态可能使结果无法完全相等。显示的大小是逻辑大小，共享哈希文件只有最后一个引用被清理时才释放空间。

清理任务通过 Raft 持久化，并立即从 locations 中移除目标。Peer 删除前向配置中的 Controller 查询授权并重新验证两份保留副本；同一分片的其他修复/清理任务在此期间等待。目标离线、无多数派或保留副本不可验证时，任务保留 pending，恢复节点后重试；没有强制删除或取消入口。旧播放缓存仍可能访问已删除副本，现有播放切源负责重试其他位置。

Peer 必须配置 `CONTROLLERS` 才会接受删除；`make console` 和 Compose 已配置。重放旧上传/确认不会恢复被删除的副本。清理日志和删除标记暂不自动回收。本功能依赖可信实验网络，不提供身份认证。

```bash
make test-cleanup
```

该测试使用临时三节点集群，覆盖目标离线、Leader 切换、删除后的 Peer 重启及保留副本验证，不操作已有集群。

## 已有基础代码与边界

| 能力 | 当前状态 |
|---|---|
| 真实 HashiCorp Raft / BoltDB / FileSnapshotStore | 已实现基础接入，可启动单节点或固定三节点集群 |
| Draft → 双副本确认 → READY | 已实现；修改都经 Raft FSM；Controller 向配置中的 Peer 核对持久对象元信息 |
| Peer 磁盘存储 / SHA-256 / Quota / 幂等上传 | 已实现；内容寻址不可变数据 + 持久索引；保守保留孤儿文件 |
| Peer 共享上传限速 / 已写出字节统计 | 已实现基础版本；计量在应用 ResponseWriter 层，不是 TCP 实际到达字节 |
| 静态双副本放置 / Python 导入 | 已实现；放置建议会扣除本次计划预计空间，但没有跨请求持久预留 |
| Headless Round Robin / 并发预取 / 校验 / 失败切源 | 已实现；真实 HTTP 与模拟媒体消费，默认首段即可启动 |
| Session REST / SSE / React 单页 | 实时 Summary/Buffer/来源估计、SSE 续传、持久 Session 与有界保留 |
| Heartbeat / Inventory / 任务 API | 已实现；上报是临时观测，维修任务与完成状态通过 Raft 持久化 |
| Deadline / 吞吐预测 / 启动缓冲 / 元数据缓存 | 已实现 headless 引擎版本；配置和指标边界见 docs/playback.md |
| Repair / 自适应放置 | 已实现最少双副本修复、任务跨 Leader 恢复、资源感知导入放置；未实现热度学习或迁移 |
| 浏览器视频播放 | 已实现 MSE 连续 fMP4、连续音视频时间戳、音量/暂停/全屏、整段跳转、有界预取与哈希校验 |
| 配对实验 / Hedging / Coding | 已有独立重置、真实 seeded trace 和同预算对比入口；完整正式研究、Hedging、Coding 仍未完成 |

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
# 打开 http://localhost:8080，选择 Headless 模拟实验，使用 fixture
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
# Dashboard 选择视频播放，输入 demo-video
```

`fixture` 是确定性二进制协议测试数据，不是视频；`demo-video` 是 FFmpeg 单次编码生成的连续 fMP4 片段。`fixture` 使用 headless 模拟实验；`demo-video` 可在“视频播放”模式下由浏览器解码画面与声音。新播放器通过同一 MediaSource/SourceBuffer 追加连续片段，换段不重置视频源或解码器。网络供给不足时仍会缓冲。

本机手动运行服务的配置见 `docs/demo.md`。前端开发在已有 Demo API 上执行 `cd frontend && pnpm dev`，默认端口 5173 代理到 8080。

## 使用自己的视频（本机 / Docker）

原视频统一放在项目根目录的 `.data/uploads/`。`.data` 是隐藏目录；macOS Finder 可用 `Command + Shift + .` 显示隐藏文件。所有路径均相对于仓库根目录。

**目前没有自动监控目录或前端上传按钮。复制文件后，需要手动切片并执行导入。** 以下以 `my-video.mp4`、视频 ID `my-video` 为例；需要本机已安装 FFmpeg、FFprobe 和 Python 3。

1. 在项目根目录创建入口目录，将自己的视频复制进去，并命名为 `my-video.mp4`：

   ```bash
   mkdir -p .data/uploads
   # 将视频复制到 .data/uploads/my-video.mp4
   ```

2. 将原视频单次编码并拆成约 2 秒的连续 fMP4 片段，生成导入清单（输出目录须为空）：

   ```bash
   bash scripts/media/segment_video.sh \
     .data/uploads/my-video.mp4 \
     .data/media/my-video/segments
   python3 scripts/media/generate_manifest.py \
     .data/media/my-video/segments \
     --video-id my-video
   ```

   当前切片脚本只编码一次 H.264/yuv420p 与 AAC，按关键帧拆分并保留连续 tfdt 时间戳，避免每段重启 AAC 编码器产生重复填充（无音轨的视频同样支持）。输出目录包含 `segment-*.mp4`、记录连续媒体时间线的 `stream.json`、供上传使用的 `my-video--*.bin` 和 `manifest.json`；`.bin` 内容就是对应 MP4 片段的字节。本文示例 ID 固定为 `my-video`；重新编码的版本若与原集群同名，应在隔离实验环境重新导入，避免覆盖不可变对象。如果该示例目录已有有效的连续 fMP4 清单，可跳过切片直接导入。ID 只允许字母、数字、下划线和连字符，长度为 1–96 个字符。

3. 如果使用 Docker 集群，先确保 Docker 服务运行并启动集群（已启动则跳过），然后导入：

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

4. 导入成功会输出 `status: READY` 的 JSON。在前端输入视频 ID `my-video`，选择“视频播放”并点击“播放视频 / 重新加载”，即可播放画面和已有音轨。原生控件支持整段进度、暂停、音量和全屏，下方进度条也可整段跳转。若浏览器阻止有声自动播放，点击播放器中的播放按钮。

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
- Yuquan：存储可靠性、库存上报、修复闭环与资源感知放置；下一步为分区/掉电覆盖、全局预留和热度策略。当前实现跨原分工目录，按实际变更复核。
- Tian：`internal/playback/`、`frontend/`、`experiments/`。已补齐 MSE 无缝视频播放、浏览器验收，以及 headless Deadline/吞吐估计、实时快照与事件持久化；下一步是正式实验 trace、资源预算与环境重置。

接手代码应先运行现有验证，再认领 TODO；不要重写另一套 Raft、导入器或指标实现。目录的具体负责人见 `docs/ownership.md`。

## 使用范围

只适用于可信 LAN/VPN 或本机实验。没有身份认证、TLS、Byzantine 防护或面向互联网的部署配置。无自动删除；空间不足明确拒绝。不要把未实现优化的 baseline 结果当作研究收益。
