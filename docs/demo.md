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
