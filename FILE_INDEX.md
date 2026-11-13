# 文件用途与 TODO 索引

TODO(Haoyu)：新增文件后运行 python3 scripts/check_scaffold.py --write-index 更新本表。

每个手写源文件均有用途 TODO。go.sum 是格式受限的自动校验文件，不可插入注释；其用途在下表说明。pnpm-lock.yaml 支持注释，因此保留文件头 TODO。编译产物、依赖目录、缓存和运行数据不纳入源文件要求。

| 文件 | 用途 / 后续负责人 |
|---|---|
| [.dockerignore](.dockerignore) | # TODO(Haoyu): Keep host caches, large datasets and build outputs outside Docker build contexts. |
| [.env.example](.env.example) | # TODO(Haoyu): Document local-only service settings; do not store secrets here. |
| [.gitattributes](.gitattributes) | # TODO(Yuquan): Keep executable Linux scripts portable across Windows checkouts. |
| [.github/workflows/ci.yml](.github/workflows/ci.yml) | # TODO(Haoyu): Maintain repository-root build and verification checks. |
| [.gitignore](.gitignore) | # TODO(Haoyu): Keep generated data, local dependencies, binaries and credentials out of version control. |
| [Makefile](Makefile) | # TODO(Haoyu): Maintain these shared entry points; commands run from the repository root. |
| [README.md](README.md) | TODO(Haoyu)：维护项目入口、构建约定和功能状态；每个负责人完成任务时同步相关文档。 |
| [cmd/client/main.go](cmd/client/main.go) | // TODO(Tian): Extend CLI configuration alongside the shared Go engine; never implement a separate CLI scheduler. |
| [cmd/controller/main.go](cmd/controller/main.go) | // TODO(Haoyu): Keep controller startup/configuration here; protocol, FSM and policy logic belong in internal packages. |
| [cmd/demo-api/main.go](cmd/demo-api/main.go) | // TODO(Tian): Extend demo routes with live Peer/buffer snapshots; the UI never bypasses the shared engine. |
| [cmd/local-console/main.go](cmd/local-console/main.go) | // TODO(Haoyu): Keep the local console bound to loopback and stop only its own child processes. |
| [cmd/peer/main.go](cmd/peer/main.go) | // TODO(Yuquan): Add report interval controls and inventory pagination for larger libraries. |
| [deployments/compose.yaml](deployments/compose.yaml) | # TODO(Haoyu): Validate this topology with Docker; host smoke tests already exercise equivalent real processes. |
| [deployments/dockerfiles/backend.Dockerfile](deployments/dockerfiles/backend.Dockerfile) | # TODO(Haoyu): Verify image tags/platforms in Docker CI; this one image supplies all four backend commands. |
| [docs/api.md](docs/api.md) | TODO(Haoyu)：维护唯一公共契约；引入预留、Heartbeat 和任务前先补请求/响应样例及调用方迁移。 |
| [docs/architecture.md](docs/architecture.md) | TODO(Haoyu)：在引入持久任务和预留时更新交互流程；保持公共协议与实现一致。 |
| [docs/decisions.md](docs/decisions.md) | TODO(Haoyu)：核对课程真实播放要求、正式提交/演示时间，并在变更接口前更新决策。 |
| [docs/demo.md](docs/demo.md) | TODO(Tian)：Docker 验证通过后录制一次 3–5 分钟演示；只展示实际已实现功能。 |
| [docs/experiments.md](docs/experiments.md) | TODO(Tian)：完成独立随机工作负载、故障 trace、环境重置、资源采集和配对不确定性分析后，才能开始正式研究实验。 |
| [docs/invariants.md](docs/invariants.md) | TODO(Haoyu)：将新任务和删除协议的不变量补充为可执行故障测试。 |
| [docs/ownership.md](docs/ownership.md) | TODO(Haoyu)：任务转交时同步本表；这份表记录团队约定的分工，尚非贡献完成声明。 |
| [docs/playback.md](docs/playback.md) | TODO(Tian)：用代表性负载校准预测参数；正式实验仍需独立 trace、匹配资源预算与环境重置。 |
| [docs/reliability.md](docs/reliability.md) | TODO(Yuquan)：补充持续分区、逐持久化边界断电、热度策略和高负载测试。 |
| [docs/storage.md](docs/storage.md) | TODO(Yuquan)：补逐 fsync 掉电测试、库存分页与观测热度策略；已实现的上报/修复流程见 reliability.md。 |
| [docs/verification.md](docs/verification.md) | TODO(Haoyu)：后续每次修改按风险更新验证证据；Docker、分区和新研究策略需分别验收。 |
| [experiments/aggregate.py](experiments/aggregate.py) | """TODO(Tian): Add paired run-level differences and uncertainty; keep failed/stopped sessions in completion denominators.""" |
| [experiments/compare.py](experiments/compare.py) | """TODO(Yuquan): Validate sensitivity to workload design before interpreting diagnostic confidence intervals.""" |
| [experiments/compare_test.py](experiments/compare_test.py) | """TODO(Yuquan): Extend metric validation when paired reports gain new statistics.""" |
| [experiments/fault_injection.py](experiments/fault_injection.py) | """TODO(Yuquan): Add timed trace-driven faults; explicit Docker/Console actions never remove stored data.""" |
| [experiments/paired.py](experiments/paired.py) | """TODO(Yuquan): Extend controlled cold-start comparisons with warm-up and timed fault traces.""" |
| [experiments/plot.py](experiments/plot.py) | """TODO(Tian): Replace this diagnostic SVG with run-level comparison plots and uncertainty for formal experiments.""" |
| [experiments/run.py](experiments/run.py) | """TODO(Tian): Add seeded request/failure traces, fresh-state resets and warm-up; current runner is a baseline plumbing check, not a controlled study.""" |
| [experiments/scenarios/baseline.json](experiments/scenarios/baseline.json) | "_todo": "TODO(Tian): Set media bitrate, fresh cluster state, failure trace and matching resource budgets before formal experiments.", |
| [experiments/scenarios/main.json](experiments/scenarios/main.json) | "_todo": "TODO(Yuquan): Extend this controlled cold-start scenario with warm-up, fault traces and a broader load matrix before calling it the final study.", |
| [frontend/index.html](frontend/index.html) | <!-- TODO(Tian): Maintain the React entry HTML; real decoding is outside this starter. --> |
| [frontend/package.json](frontend/package.json) | "_todo": "TODO(Tian): Maintain the dashboard package; generated lockfile purpose is documented in FILE_INDEX.md.", |
| [frontend/pnpm-lock.yaml](frontend/pnpm-lock.yaml) | # TODO(Tian): Generated dependency lock; update using pnpm install and review resolved package changes. |
| [frontend/pnpm-workspace.yaml](frontend/pnpm-workspace.yaml) | # TODO(Tian): Review explicitly permitted dependency build scripts when updating the lockfile. |
| [frontend/src/App.tsx](frontend/src/App.tsx) | // TODO(Tian): Keep app composition here; scheduling and metric calculations stay in Go. |
| [frontend/src/api/client.ts](frontend/src/api/client.ts) | // TODO(Tian): Add versioned API capabilities; display server errors without generating local experimental metrics. |
| [frontend/src/components/FaultLab.tsx](frontend/src/components/FaultLab.tsx) | // TODO(Yuquan): Add replayable network-fault traces after process-fault workflows are validated. |
| [frontend/src/components/LocalConsole.tsx](frontend/src/components/LocalConsole.tsx) | // TODO(Haoyu): Keep local lifecycle and upload feedback tied to actual server responses. |
| [frontend/src/components/MetricsPanel.tsx](frontend/src/components/MetricsPanel.tsx) | // TODO(Tian): Validate dashboard terminology alongside future metric schema changes. |
| [frontend/src/components/ReplicaCleanup.tsx](frontend/src/components/ReplicaCleanup.tsx) | // TODO(Yuquan): Add keeper preferences when placement policies expose stable rankings. |
| [frontend/src/components/VideoPlayer.tsx](frontend/src/components/VideoPlayer.tsx) | // TODO(Tian): Keep real decoder metrics separate from headless experiment summaries. |
| [frontend/src/hooks/useSession.ts](frontend/src/hooks/useSession.ts) | // TODO(Tian): Keep snapshot recovery compatible with future event schema versions. |
| [frontend/src/main.tsx](frontend/src/main.tsx) | // TODO(Tian): Add top-level error boundaries as dashboard routes grow. |
| [frontend/src/pages/Dashboard.tsx](frontend/src/pages/Dashboard.tsx) | // TODO(Tian): Keep playback controls aligned with reproducible CLI configurations. |
| [frontend/src/playback/msePlayer.ts](frontend/src/playback/msePlayer.ts) | // TODO(Tian): Extend codec negotiation and eviction policy with real-browser regressions. |
| [frontend/src/playback/segmentBuffer.ts](frontend/src/playback/segmentBuffer.ts) | // TODO(Tian): Preserve verified, bounded prefetch as new transports are introduced. |
| [frontend/src/styles.css](frontend/src/styles.css) | /* TODO(Tian): Extend responsive dashboard styling without blocking core system work. */ |
| [frontend/src/types/api.ts](frontend/src/types/api.ts) | // TODO(Tian): Keep additive playback fields aligned with the Go API contract. |
| [frontend/tests/browser.mjs](frontend/tests/browser.mjs) | // TODO(Tian): Expand browser coverage to additional MSE implementations and codecs. |
| [frontend/tests/msePlayer.test.mjs](frontend/tests/msePlayer.test.mjs) | // TODO(Tian): Keep single-source, seek and bounded-buffer invariants covered for MSE changes. |
| [frontend/tests/run.mjs](frontend/tests/run.mjs) | // TODO(Tian): Register new browser-independent client regressions here. |
| [frontend/tests/segmentBuffer.test.mjs](frontend/tests/segmentBuffer.test.mjs) | // TODO(Tian): Extend client buffer regression tests alongside seek and transport changes. |
| [frontend/tsconfig.json](frontend/tsconfig.json) | "//": "TODO(Tian): Maintain strict browser TypeScript checks; keep protocol types aligned with Go.", |
| [frontend/vite.config.ts](frontend/vite.config.ts) | // TODO(Tian): Add controller/Peer proxy routes when their dashboard panels are implemented. |
| [go.mod](go.mod) | // TODO(Haoyu): Maintain the shared Go module and review dependency upgrades. |
| [go.sum](go.sum) | TODO(Haoyu)：Go 自动管理的依赖完整性校验；格式禁止添加用途注释，本索引代为说明。 |
| [internal/control/coordinator.go](internal/control/coordinator.go) | // TODO(Yuquan): Paginate inventories and bound parallel repair before scaling beyond lab workloads. |
| [internal/control/coordinator_test.go](internal/control/coordinator_test.go) | // TODO(Yuquan): Extend coordinator tests with bandwidth stalls and retry backoff. |
| [internal/domain/models.go](internal/domain/models.go) | // TODO(Haoyu): Own the common domain model; extend versioned task and coding types here. |
| [internal/domain/models_test.go](internal/domain/models_test.go) | // TODO(Haoyu): Extend manifest safety tests when additional media representations are introduced. |
| [internal/localconsole/delete_video.go](internal/localconsole/delete_video.go) | // TODO(Haoyu): Move deletion reconciliation to a background worker for large libraries. |
| [internal/localconsole/delete_video_test.go](internal/localconsole/delete_video_test.go) | // TODO(Haoyu): Exercise deletion reconciliation with a real browser console. |
| [internal/localconsole/faults.go](internal/localconsole/faults.go) | // TODO(Yuquan): Extend supervised faults with explicit trace logging and network faults. |
| [internal/localconsole/faults_test.go](internal/localconsole/faults_test.go) | // TODO(Yuquan): Add long-running concurrency tests for simultaneous browser sessions. |
| [internal/localconsole/manager.go](internal/localconsole/manager.go) | // TODO(Haoyu): Extend local process supervision without changing distributed storage protocols. |
| [internal/metadata/cleanup.go](internal/metadata/cleanup.go) | // TODO(Yuquan): Add paginated cleanup previews for large video libraries. |
| [internal/metadata/cleanup_test.go](internal/metadata/cleanup_test.go) | // TODO(Yuquan): Extend cleanup concurrency tests alongside future membership changes. |
| [internal/metadata/delete_video.go](internal/metadata/delete_video.go) | // TODO(Haoyu): Add bounded retention for deleted video manifests and task history. |
| [internal/metadata/delete_video_test.go](internal/metadata/delete_video_test.go) | // TODO(Haoyu): Extend video deletion tests when replica generations are introduced. |
| [internal/metadata/fsm.go](internal/metadata/fsm.go) | // TODO(Yuquan): Add global capacity reservations and bounded task retention; never perform network I/O in Apply. |
| [internal/metadata/fsm_test.go](internal/metadata/fsm_test.go) | // TODO(Haoyu): Extend safety tests to durable task transitions and concurrent removals when those features land. |
| [internal/metadata/handlers.go](internal/metadata/handlers.go) | // TODO(Yuquan): Replace advisory ingestion plans with global persistent capacity reservations. |
| [internal/metadata/maintenance.go](internal/metadata/maintenance.go) | // TODO(Yuquan): Add report pagination and bounded task retention for larger deployments. |
| [internal/metadata/maintenance_test.go](internal/metadata/maintenance_test.go) | // TODO(Yuquan): Extend durable task tests alongside new lifecycle transitions. |
| [internal/metadata/raft.go](internal/metadata/raft.go) | // TODO(Haoyu): Add controlled membership changes and quorum/failover tests; this uses persistent HashiCorp Raft, not a mock. |
| [internal/placement/adaptive.go](internal/placement/adaptive.go) | // TODO(Yuquan): Add measured popularity only after defining an observation window. |
| [internal/placement/adaptive_test.go](internal/placement/adaptive_test.go) | // TODO(Yuquan): Compare observed-resource placement with workload-specific policies. |
| [internal/placement/plan.go](internal/placement/plan.go) | // TODO(Yuquan): Persist capacity reservations when supporting concurrent ingestion guarantees. |
| [internal/placement/static.go](internal/placement/static.go) | // TODO(Yuquan): Add capacity reservations and matched-budget adaptive placement; current plans are advisory, with Peer quotas authoritative. |
| [internal/playback/buffer.go](internal/playback/buffer.go) | // TODO(Tian): Preserve bounded buffering when adding a media decoder. |
| [internal/playback/clock.go](internal/playback/clock.go) | // TODO(Tian): Keep clock implementations cancellation-aware as playback evolves. |
| [internal/playback/deadline.go](internal/playback/deadline.go) | // TODO(Tian): Calibrate prediction defaults against representative workloads. |
| [internal/playback/downloader.go](internal/playback/downloader.go) | // TODO(Tian): Preserve size and hash verification for every download path. |
| [internal/playback/engine.go](internal/playback/engine.go) | // TODO(Tian): Keep headless consumption and real media decoding as explicit, separately measured modes. |
| [internal/playback/events.go](internal/playback/events.go) | // TODO(Tian): Version event schemas before changing persisted field semantics. |
| [internal/playback/media.go](internal/playback/media.go) | // TODO(Tian): Keep verified media delivery independent of browser buffering policy. |
| [internal/playback/media_test.go](internal/playback/media_test.go) | // TODO(Tian): Extend media gateway tests when new representations are introduced. |
| [internal/playback/metadata_client.go](internal/playback/metadata_client.go) | // TODO(Tian): Revisit cache lifetime when mutable representations or deletion are introduced. |
| [internal/playback/metadata_client_test.go](internal/playback/metadata_client_test.go) | // TODO(Tian): Extend election recovery tests alongside controller discovery changes. |
| [internal/playback/metrics.go](internal/playback/metrics.go) | // TODO(Tian): Keep session metric definitions stable across experiment versions. |
| [internal/playback/playback_test.go](internal/playback/playback_test.go) | // TODO(Tian): Extend deterministic scenarios when media or scheduling contracts change. |
| [internal/playback/round_robin.go](internal/playback/round_robin.go) | // TODO(Tian): Measure the deterministic baseline before modifying source selection. |
| [internal/playback/scheduler.go](internal/playback/scheduler.go) | // TODO(Tian): Extend source prediction only alongside reproducible comparison tests. |
| [internal/playback/sessions.go](internal/playback/sessions.go) | // TODO(Tian): Introduce schema migration before changing persisted session records. |
| [internal/protocol/dto.go](internal/protocol/dto.go) | // TODO(Haoyu): Freeze these DTOs with both consumers before adding fields or endpoints. |
| [internal/protocol/http.go](internal/protocol/http.go) | // TODO(Haoyu): Maintain HTTP contracts and bounded decoding; add authentication before wider deployment. |
| [internal/protocol/serve.go](internal/protocol/serve.go) | // TODO(Haoyu): Add structured request logging and per-route deadlines; SSE intentionally has no global write timeout. |
| [internal/repair/policy.go](internal/repair/policy.go) | // TODO(Yuquan): Implement grace-period repair planning from verified inventory; this policy must not mutate Raft or delete data. |
| [internal/repair/policy_test.go](internal/repair/policy_test.go) | // TODO(Yuquan): Extend repair policy checks when replica budgets become configurable. |
| [internal/storage/cleanup.go](internal/storage/cleanup.go) | // TODO(Yuquan): Add generation-based cleanup retention before pruning tombstones. |
| [internal/storage/cleanup_test.go](internal/storage/cleanup_test.go) | // TODO(Yuquan): Expand cleanup tests with injected fsync failures. |
| [internal/storage/handlers.go](internal/storage/handlers.go) | // TODO(Yuquan): Add authenticated task execution/telemetry; upload bytes are counted at ResponseWriter, not TCP-wire level. |
| [internal/storage/reliability_test.go](internal/storage/reliability_test.go) | // TODO(Yuquan): Add subprocess power-loss injection at each persistence boundary. |
| [internal/storage/reporting.go](internal/storage/reporting.go) | // TODO(Yuquan): Bound/paginate reports for inventories larger than the lab JSON limit. |
| [internal/storage/store.go](internal/storage/store.go) | // TODO(Yuquan): Extend immutable object storage with inventory and authorized cleanup; never acknowledge before durable install. |
| [internal/storage/store_test.go](internal/storage/store_test.go) | // TODO(Yuquan): Add crash-at-each-fsync and concurrent quota tests before implementing cleanup or migrations. |
| [report/evaluation.md](report/evaluation.md) | TODO(Tian)：填写工作负载、配对对照、指标、不确定性及负结果；不将基础流水线数据当正式实验。 |
| [report/final.md](report/final.md) | TODO(Haoyu)：课程格式确认后合并各负责人章节，列出实际贡献、已知限制与 AI 辅助使用；当前不是已完成报告。 |
| [report/integration.md](report/integration.md) | TODO(Haoyu)：说明架构、构建环境、端到端验证、实现与计划的差异和实际贡献。 |
| [report/metadata.md](report/metadata.md) | TODO(Haoyu)：说明实际 Raft 配置、FSM 命令、持久化和故障测试；仅填写有日志支持的结果。 |
| [report/storage.md](report/storage.md) | TODO(Yuquan)：按课程报告格式补实验图表、逐阶段断电测试及实测限制。 |
| [requirements.txt](requirements.txt) | # TODO(Haoyu): Keep Python runtime requirements aligned with scripts and experiments. |
| [results/README.md](results/README.md) | TODO(Tian)：保存配置、原始事件、session 汇总、run 汇总和正式配对分析。大数据另存时记录地址与校验值。 |
| [scripts/check_scaffold.py](scripts/check_scaffold.py) | """TODO(Haoyu): Maintain the per-file purpose index and audit owner TODO markers when extending the scaffold.""" |
| [scripts/ingest/ingest.py](scripts/ingest/ingest.py) | """TODO(Yuquan): Replace advisory placement with durable reservations; preserve operation IDs on all retries.""" |
| [scripts/media/fixture.py](scripts/media/fixture.py) | """TODO(Yuquan): Maintain deterministic byte fixtures for protocol tests; these are NOT decodable video.""" |
| [scripts/media/fragment_mp4.py](scripts/media/fragment_mp4.py) | """TODO(Yuquan): Extend the muxed fMP4 representation only with matching timeline tests.""" |
| [scripts/media/generate_fixture.sh](scripts/media/generate_fixture.sh) | # TODO(Yuquan): Extend the licensed media fixture library; this generates a real 12-second test video locally. |
| [scripts/media/generate_manifest.py](scripts/media/generate_manifest.py) | """TODO(Yuquan): Keep codec/track validation aligned with browser media support.""" |
| [scripts/media/segment_video.sh](scripts/media/segment_video.sh) | # TODO(Yuquan): Preserve one encoder timeline when extending continuous fMP4 packaging. |
| [scripts/run_console.sh](scripts/run_console.sh) | # TODO(Haoyu): Provide one local entry point while keeping tools in the user's own installation. |
| [tests/e2e/README.md](tests/e2e/README.md) | TODO(Haoyu)：补充浏览器 UI 与服务 API 的联合验收；当前 make test-e2e 复用 tests/system/smoke.py。 |
| [tests/integration/playback_test.go](tests/integration/playback_test.go) | // TODO(Tian): Add explicit failed-session, cancellation and controlled-stall metric cases to these real HTTP integration tests. |
| [tests/system/cleanup.py](tests/system/cleanup.py) | """TODO(Yuquan): Extend cleanup acceptance with disk failure injection.""" |
| [tests/system/cluster.py](tests/system/cluster.py) | """TODO(Yuquan): Share isolated real-process fixtures without attaching to user clusters.""" |
| [tests/system/console.py](tests/system/console.py) | """TODO(Haoyu): Exercise local orchestration, real upload/trim/ingest and verified Peer delivery in isolated data directories.""" |
| [tests/system/delete_video.py](tests/system/delete_video.py) | """TODO(Haoyu): Add partial-ingest orphan collection to deletion acceptance.""" |
| [tests/system/media.py](tests/system/media.py) | """TODO(Tian): Add browser decoding automation when a supported browser runner is available.""" |
| [tests/system/repair.py](tests/system/repair.py) | """TODO(Yuquan): Extend repair acceptance with sustained network partitions and power-loss faults.""" |
| [tests/system/smoke.py](tests/system/smoke.py) | """TODO(Haoyu): Extend this real-process Raft smoke test with partitions, disk-loss faults and task recovery.""" |
