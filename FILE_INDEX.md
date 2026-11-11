# 文件用途与 TODO 索引

TODO(Haoyu)：新增文件后运行 python3 scripts/check_scaffold.py --write-index 更新本表。

每个手写源文件均有用途 TODO。go.sum 是格式受限的自动校验文件，不可插入注释；其用途在下表说明。pnpm-lock.yaml 支持注释，因此保留文件头 TODO。编译产物、依赖目录、缓存和运行数据不纳入源文件要求。

| 文件 | 用途 / 后续负责人 |
|---|---|
| [.dockerignore](.dockerignore) | # TODO(Haoyu): Keep host caches, large datasets and build outputs outside Docker build contexts. |
| [.env.example](.env.example) | # TODO(Haoyu): Document local-only service settings; do not store secrets here. |
| [.github/workflows/ci.yml](.github/workflows/ci.yml) | # TODO(Haoyu): Maintain repository-root build and verification checks. |
| [.gitignore](.gitignore) | # TODO(Haoyu): Keep generated data, local dependencies, binaries and credentials out of version control. |
| [Makefile](Makefile) | # TODO(Haoyu): Maintain these shared entry points; commands run from the repository root. |
| [README.md](README.md) | TODO(Haoyu)：维护项目入口、构建约定和功能状态；每个负责人完成任务时同步相关文档。 |
| [cmd/client/main.go](cmd/client/main.go) | // TODO(Tian): Extend CLI configuration alongside the shared Go engine; never implement a separate CLI scheduler. |
| [cmd/controller/main.go](cmd/controller/main.go) | // TODO(Haoyu): Keep controller startup/configuration here; protocol, FSM and policy logic belong in internal packages. |
| [cmd/demo-api/main.go](cmd/demo-api/main.go) | // TODO(Tian): Extend demo routes with live Peer/buffer snapshots; the UI never bypasses the shared engine. |
| [cmd/peer/main.go](cmd/peer/main.go) | // TODO(Yuquan): Add heartbeat/inventory workers after their real controller endpoints are implemented. |
| [deployments/compose.yaml](deployments/compose.yaml) | # TODO(Haoyu): Validate this topology with Docker; host smoke tests already exercise equivalent real processes. |
| [deployments/dockerfiles/backend.Dockerfile](deployments/dockerfiles/backend.Dockerfile) | # TODO(Haoyu): Verify image tags/platforms in Docker CI; this one image supplies all four backend commands. |
| [docs/api.md](docs/api.md) | TODO(Haoyu)：维护唯一公共契约；引入预留、Heartbeat 和任务前先补请求/响应样例及调用方迁移。 |
| [docs/architecture.md](docs/architecture.md) | TODO(Haoyu)：在引入持久任务和预留时更新交互流程；保持公共协议与实现一致。 |
| [docs/decisions.md](docs/decisions.md) | TODO(Haoyu)：核对课程真实播放要求、正式提交/演示时间，并在变更接口前更新决策。 |
| [docs/demo.md](docs/demo.md) | TODO(Tian)：Docker 验证通过后录制一次 3–5 分钟演示；只展示实际已实现功能。 |
| [docs/experiments.md](docs/experiments.md) | TODO(Tian)：完成独立随机工作负载、故障 trace、环境重置、资源采集和配对不确定性分析后，才能开始正式研究实验。 |
| [docs/invariants.md](docs/invariants.md) | TODO(Haoyu)：将新任务和删除协议的不变量补充为可执行故障测试。 |
| [docs/ownership.md](docs/ownership.md) | TODO(Haoyu)：任务转交时同步本表；这份表记录团队约定的分工，尚非贡献完成声明。 |
| [docs/playback.md](docs/playback.md) | TODO(Tian)：实现预计完成时间选源、deadline miss、缓存元数据、可控时钟和完整实时 Buffer 快照。 |
| [docs/storage.md](docs/storage.md) | TODO(Yuquan)：实现 Heartbeat、Inventory、复制任务执行与观测热度放置；后续添加磁盘故障注入测试。 |
| [docs/verification.md](docs/verification.md) | TODO(Haoyu)：后续每次修改按风险更新验证证据；Docker、分区和新研究策略需分别验收。 |
| [experiments/aggregate.py](experiments/aggregate.py) | """TODO(Tian): Add paired run-level differences and uncertainty; keep failed/stopped sessions in completion denominators.""" |
| [experiments/fault_injection.py](experiments/fault_injection.py) | """TODO(Tian): Add timed trace-driven partitions; current actions require an explicit service and never remove volumes.""" |
| [experiments/plot.py](experiments/plot.py) | """TODO(Tian): Replace this diagnostic SVG with run-level comparison plots and uncertainty for formal experiments.""" |
| [experiments/run.py](experiments/run.py) | """TODO(Tian): Add seeded request/failure traces, fresh-state resets and warm-up; current runner is a baseline plumbing check, not a controlled study.""" |
| [experiments/scenarios/baseline.json](experiments/scenarios/baseline.json) | "_todo": "TODO(Tian): Set media bitrate, fresh cluster state, failure trace and matching resource budgets before formal experiments.", |
| [experiments/scenarios/main.json](experiments/scenarios/main.json) | "_todo": "TODO(Tian/Yuquan): Enable the main study only after adaptive placement, deadline scheduling and matched-budget reset are implemented.", |
| [frontend/index.html](frontend/index.html) | <!-- TODO(Tian): Maintain the React entry HTML; real decoding is outside this starter. --> |
| [frontend/package.json](frontend/package.json) | "_todo": "TODO(Tian): Maintain the dashboard package; generated lockfile purpose is documented in FILE_INDEX.md.", |
| [frontend/pnpm-lock.yaml](frontend/pnpm-lock.yaml) | # TODO(Tian): Generated dependency lock; update using pnpm install and review resolved package changes. |
| [frontend/pnpm-workspace.yaml](frontend/pnpm-workspace.yaml) | # TODO(Tian): Review explicitly permitted dependency build scripts when updating the lockfile. |
| [frontend/src/App.tsx](frontend/src/App.tsx) | // TODO(Tian): Keep app composition here; scheduling and metric calculations stay in Go. |
| [frontend/src/api/client.ts](frontend/src/api/client.ts) | // TODO(Tian): Add versioned API capabilities; display server errors without generating local experimental metrics. |
| [frontend/src/components/MetricsPanel.tsx](frontend/src/components/MetricsPanel.tsx) | // TODO(Tian): Add buffer/Peer views after the backend exposes snapshots; never manufacture missing live values. |
| [frontend/src/hooks/useSession.ts](frontend/src/hooks/useSession.ts) | // TODO(Tian): Add SSE replay with Last-Event-ID; snapshots currently recover state by polling once per second. |
| [frontend/src/main.tsx](frontend/src/main.tsx) | // TODO(Tian): Add top-level error boundaries as dashboard routes grow. |
| [frontend/src/pages/Dashboard.tsx](frontend/src/pages/Dashboard.tsx) | // TODO(Tian): Expand this minimal single-page dashboard after baseline experiments work. |
| [frontend/src/styles.css](frontend/src/styles.css) | /* TODO(Tian): Extend responsive dashboard styling without blocking core system work. */ |
| [frontend/src/types/api.ts](frontend/src/types/api.ts) | // TODO(Tian): Update these consumer DTOs only after Haoyu publishes contract changes. |
| [frontend/tsconfig.json](frontend/tsconfig.json) | "//": "TODO(Tian): Maintain strict browser TypeScript checks; keep protocol types aligned with Go.", |
| [frontend/vite.config.ts](frontend/vite.config.ts) | // TODO(Tian): Add controller/Peer proxy routes when their dashboard panels are implemented. |
| [go.mod](go.mod) | // TODO(Haoyu): Maintain the shared Go module and review dependency upgrades. |
| [go.sum](go.sum) | TODO(Haoyu)：Go 自动管理的依赖完整性校验；格式禁止添加用途注释，本索引代为说明。 |
| [internal/control/coordinator.go](internal/control/coordinator.go) | // TODO(Haoyu): Implement leader-only reservation/task coordination and replay; repair is deliberately unavailable until durable transitions exist. |
| [internal/domain/models.go](internal/domain/models.go) | // TODO(Haoyu): Own the common domain model; extend versioned task and coding types here. |
| [internal/metadata/fsm.go](internal/metadata/fsm.go) | // TODO(Haoyu): Extend the real Raft FSM with reservations and durable repair transitions; never perform network I/O in Apply. |
| [internal/metadata/fsm_test.go](internal/metadata/fsm_test.go) | // TODO(Haoyu): Extend safety tests to durable task transitions and concurrent removals when those features land. |
| [internal/metadata/handlers.go](internal/metadata/handlers.go) | // TODO(Haoyu): Replace advisory placement with durable reservations; heartbeat, inventory and repair endpoints currently return 501. |
| [internal/metadata/raft.go](internal/metadata/raft.go) | // TODO(Haoyu): Add controlled membership changes and quorum/failover tests; this uses persistent HashiCorp Raft, not a mock. |
| [internal/placement/adaptive.go](internal/placement/adaptive.go) | // TODO(Yuquan): Implement popularity/startup-weighted optional replicas after matching fixed-placement storage budgets. |
| [internal/placement/static.go](internal/placement/static.go) | // TODO(Yuquan): Add capacity reservations and matched-budget adaptive placement; current plans are advisory, with Peer quotas authoritative. |
| [internal/playback/buffer.go](internal/playback/buffer.go) | // TODO(Tian): Extend the bounded, per-index result slots with a UI buffer snapshot; do not introduce an unbounded segment cache. |
| [internal/playback/clock.go](internal/playback/clock.go) | // TODO(Tian): Add deterministic clock injection for metric tests; this clock consumes media in real wall time without decoding. |
| [internal/playback/deadline.go](internal/playback/deadline.go) | // TODO(Tian): Implement predicted completion, EWMA throughput and stall-shifted deadlines; do not alias the baseline as an optimization. |
| [internal/playback/downloader.go](internal/playback/downloader.go) | // TODO(Tian): Add per-source throughput estimates and deadline-driven retry; all successful objects must pass size/hash validation. |
| [internal/playback/engine.go](internal/playback/engine.go) | // TODO(Tian): Add configurable startup buffering, cached metadata fallback and deadline metrics; baseline uses bounded concurrent prefetch. |
| [internal/playback/events.go](internal/playback/events.go) | // TODO(Tian): Persist event logs for experiments and implement reconnect replay; callbacks must remain fast and thread-safe. |
| [internal/playback/metadata_client.go](internal/playback/metadata_client.go) | // TODO(Tian): Add cached manifests and controlled refresh during controller outages; retries use configured controller URLs only. |
| [internal/playback/metrics.go](internal/playback/metrics.go) | // TODO(Tian): Add frozen-deadline misses and per-request JSONL; this is the sole definition of session-level playback metrics. |
| [internal/playback/round_robin.go](internal/playback/round_robin.go) | // TODO(Tian): Measure the deterministic baseline before modifying source selection. |
| [internal/playback/scheduler.go](internal/playback/scheduler.go) | // TODO(Tian): Keep source selection separate from downloading; extend assignments with explicit deadline and queue estimates. |
| [internal/playback/sessions.go](internal/playback/sessions.go) | // TODO(Tian): Add persistent session/event retention and active buffer snapshots; current process retains at most 100 sessions and 256 events each. |
| [internal/protocol/dto.go](internal/protocol/dto.go) | // TODO(Haoyu): Freeze these DTOs with both consumers before adding fields or endpoints. |
| [internal/protocol/http.go](internal/protocol/http.go) | // TODO(Haoyu): Maintain HTTP contracts and bounded decoding; add authentication before wider deployment. |
| [internal/protocol/serve.go](internal/protocol/serve.go) | // TODO(Haoyu): Add structured request logging and per-route deadlines; SSE intentionally has no global write timeout. |
| [internal/repair/policy.go](internal/repair/policy.go) | // TODO(Yuquan): Implement grace-period repair planning from verified inventory; this policy must not mutate Raft or delete data. |
| [internal/storage/handlers.go](internal/storage/handlers.go) | // TODO(Yuquan): Add authenticated task execution/telemetry; upload bytes are counted at ResponseWriter, not TCP-wire level. |
| [internal/storage/store.go](internal/storage/store.go) | // TODO(Yuquan): Extend immutable object storage with inventory and authorized cleanup; never acknowledge before durable install. |
| [internal/storage/store_test.go](internal/storage/store_test.go) | // TODO(Yuquan): Add crash-at-each-fsync and concurrent quota tests before implementing cleanup or migrations. |
| [report/evaluation.md](report/evaluation.md) | TODO(Tian)：填写工作负载、配对对照、指标、不确定性及负结果；不将基础流水线数据当正式实验。 |
| [report/final.md](report/final.md) | TODO(Haoyu)：课程格式确认后合并各负责人章节，列出实际贡献、已知限制与 AI 辅助使用；当前不是已完成报告。 |
| [report/integration.md](report/integration.md) | TODO(Haoyu)：说明架构、构建环境、端到端验证、实现与计划的差异和实际贡献。 |
| [report/metadata.md](report/metadata.md) | TODO(Haoyu)：说明实际 Raft 配置、FSM 命令、持久化和故障测试；仅填写有日志支持的结果。 |
| [report/storage.md](report/storage.md) | TODO(Yuquan)：说明对象安装、配额、完整性、放置和后续 Repair 的真实实现与局限。 |
| [results/README.md](results/README.md) | TODO(Tian)：保存配置、原始事件、session 汇总、run 汇总和正式配对分析。大数据另存时记录地址与校验值。 |
| [scripts/check_scaffold.py](scripts/check_scaffold.py) | """TODO(Haoyu): Maintain the per-file purpose index and audit owner TODO markers when extending the scaffold.""" |
| [scripts/ingest/ingest.py](scripts/ingest/ingest.py) | """TODO(Yuquan): Replace advisory placement with durable reservations; preserve operation IDs on all retries.""" |
| [scripts/media/fixture.py](scripts/media/fixture.py) | """TODO(Yuquan): Maintain deterministic byte fixtures for protocol tests; these are NOT decodable video.""" |
| [scripts/media/generate_fixture.sh](scripts/media/generate_fixture.sh) | # TODO(Yuquan): Extend the licensed media fixture library; this generates a real 12-second test video locally. |
| [scripts/media/generate_manifest.py](scripts/media/generate_manifest.py) | """TODO(Yuquan): Extend manifest generation to audio/representations; actual duration is read using FFprobe.""" |
| [scripts/media/segment_video.sh](scripts/media/segment_video.sh) | # TODO(Yuquan): Tune media encoding for research workloads; first version produces independently decodable MP4 segments. |
| [tests/e2e/README.md](tests/e2e/README.md) | TODO(Haoyu)：补充浏览器 UI 与服务 API 的联合验收；当前 make test-e2e 复用 tests/system/smoke.py。 |
| [tests/integration/playback_test.go](tests/integration/playback_test.go) | // TODO(Tian): Add explicit failed-session, cancellation and controlled-stall metric cases to these real HTTP integration tests. |
| [tests/system/smoke.py](tests/system/smoke.py) | """TODO(Haoyu): Extend this real-process Raft smoke test with partitions, disk-loss faults and task recovery.""" |
