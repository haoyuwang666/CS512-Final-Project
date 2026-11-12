# 启动和演示

TODO(Tian)：Docker 验证通过后录制一次 3–5 分钟演示；只展示实际已实现功能。

最快验证：从本目录执行 `make smoke`。它是短时自动化验收，不会留下长期运行服务。

Docker 方式见 README。第一次运行先 `make up`、`make fixture`、`make ingest`，再打开 8080。Controller 内部返回的 Peer DNS 只有容器可达，故导入与实验都由 Compose tools 服务执行。`make down` 保留数据卷；测试重置和删除卷必须明确区分，不把停止服务当成清空环境。

不用 Docker 时，可在独立终端运行：

```bash
HTTP_ADDR=127.0.0.1:9001 PEER_ID=peer-1 DATA_DIR=.data/peer-1 ./bin/peer
HTTP_ADDR=127.0.0.1:9002 PEER_ID=peer-2 DATA_DIR=.data/peer-2 ./bin/peer
HTTP_ADDR=127.0.0.1:9003 PEER_ID=peer-3 DATA_DIR=.data/peer-3 ./bin/peer
PEERS='[{"peer_id":"peer-1","url":"http://127.0.0.1:9001"},{"peer_id":"peer-2","url":"http://127.0.0.1:9002"},{"peer_id":"peer-3","url":"http://127.0.0.1:9003"}]' ./bin/controller
python3 scripts/media/fixture.py
python3 scripts/ingest/ingest.py .data/fixture/manifest.json --controllers http://127.0.0.1:8001
./bin/demo-api
```

这是单 Controller 本地开发模式，不能作为三节点高可用性演示。三节点启动需要给每个进程设置不同 NODE_ID、HTTP_ADDR、RAFT_BIND、DATA_DIR，相同 RAFT_MEMBERS，且只有第一个 BOOTSTRAP=true；具体可复用 `tests/system/smoke.py` 中的配置方式。

前端开发：`cd frontend && pnpm dev`，打开 5173。生产静态服务：先 `make frontend`，Demo API 从 frontend/dist 提供文件。

故障入口：`python3 experiments/fault_injection.py stop peer-1`，之后用 start 恢复。当前无自动 Repair，不能把切源成功展示为已补齐副本。Leader 故障由现存多数派选主；完整 Controller 断线期间不能保证新增播放查询成功。


## 实际画面与声音

保持 Peer、Controller、Demo API 和前端运行。将原视频保存在 .data/uploads/my-video.mp4 后，在根目录执行（团队示例 ID 统一为 my-video；已有同名旧数据时需使用干净、隔离的实验环境）：

```bash
bash scripts/media/segment_video.sh .data/uploads/my-video.mp4 .data/media/my-video/segments
python3 scripts/media/generate_manifest.py .data/media/my-video/segments --video-id my-video
python3 scripts/ingest/ingest.py .data/media/my-video/segments/manifest.json --controllers http://127.0.0.1:8001
```

选择“视频播放”，输入 my-video，点击“播放视频 / 重新加载”。可暂停、调节音量、全屏；整段跳转用下方进度条。旧 my-video 的独立 MP4 需要重新生成连续分片，旧音轨如已删除也只能从原文件恢复。fixture 仅用于“Headless 模拟实验”。使用 localhost 或 HTTPS 确保浏览器可执行哈希校验。

`make test-media` 需要 FFmpeg/FFprobe，临时启动独立集群，验证带声音的视频导入、Peer 故障切源、网关字节校验，并解码返回的所有片段；不会修改现有视频。`make test-player` 验证浏览器预取、跳转取消、ArrayBuffer 回收、SourceBuffer 串行更新与损坏数据拒绝；`make test-media-browser` 则执行真实浏览器交互。


无缝播放准备完成后，重新启动新版 Controller 与 Demo API，再导入统一的 my-video。已有同名旧视频时，应隔离 Controller、全部 Peer 与素材目录，保留原数据后在新实验环境重新导入。若输出目录已有 segment-*.mp4，切片脚本会拒绝覆盖；请选择新目录。连续分片仍通过相同双副本导入流程。

浏览器验收入口 `make test-media-browser` 使用可选的 Playwright 安装，需环境变量 PLAYWRIGHT_MODULE 指向 playwright/index.mjs；可通过 PLAYWRIGHT_BROWSERS_PATH 指定浏览器位置。验收实际跨四个连续片段播放、确认只有一次 loadstart、首次 playing 后没有 waiting，并检查回跳、暂停、前跳与停止。工具不属于部署运行时依赖。
