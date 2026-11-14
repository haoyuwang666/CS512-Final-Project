# 存储、修复与配对验证

TODO(Yuquan)：补充持续分区、逐持久化边界断电、热度策略和高负载测试。

## Recovery reconciliation update (2026-11-14)

Before recopying a pending repair, the coordinator checks the target's verified receipt. If it matches the same task, the task can complete even when the original source has since stopped. When a copy attempt failed after installation or its metadata acknowledgment was lost, verified inventory from configured Peers can recover the unregistered replica through a Raft confirmation. This path needs quorum but no new disk space or surviving original holder; it does not recreate missing bytes or collect unindexed orphan files. Failed task history remains failed even if the installed copy is later adopted.

Equivalent PUT attempts retain one canonical installed receipt and separate operation records. Cleanup/deletion matches immutable object identity, retires all recorded operation aliases, and remains retryable across Peer restart. Keep-copy verification also accepts equivalent operation receipts while rejecting changed Peer, segment, version, size, or hash. Repair ACK validation rejects malformed task receipts before proposing the stable completion ID so a bad request cannot poison later valid retries.

Regression coverage includes lost repair responses, pending-target recovery with a dead source, adoption after both original holders stop, equivalent PUTs before deletion, old persisted receipt/index layouts, and rejection of retired aliases. `make test-media` also checks stored BMFF decode timelines instead of assuming demuxed nominal packet durations describe the initial alignment sample.

## 当前实现

Peer 的 `GET /api/v1/inventory` 扫描持久索引，重新计算对象大小和 SHA-256。
损坏、丢失、孤儿对象不会出现在有效库存中，已有 blob 不自动删除或覆盖。
重复 PUT 也重新检查已有 blob，避免损坏数据被重复 ACK。
磁盘/索引错误返回 500；参数与哈希错误返回 400；容量不足返回 507。

Peer 设置 `CONTROLLERS` 后每 2 秒向可用 Leader 上报 heartbeat/inventory。
上报是临时观测，不是共识副本记录；Leader 切换后会重报。
Controller 修复前独立从启动时配置的 Peer URL 拉取并核对库存，不能靠伪造上报发布副本。
当前库存是全量且需要读盘哈希，适合小型课程实验；没有分页或增量 scrub。

Leader 每 2 秒协调一次。已发布分片不足两份可达且有效的已登记副本时，
等待 `REPAIR_GRACE`（默认 6s）。至少有一个有效来源和一个有容量的新目标才创建任务。
已登记但损坏的 Peer 不作为覆盖目标；两份均不可达时不虚构恢复。
`REPAIR_ENABLED=false` 可禁用修复，默认启用。

任务通过 Raft FSM 依次提交 `pending → completed` 或 `pending → failed`。
同一分片不能同时拥有多个 pending 任务，任务 ID 是目标 PUT 的 operation ID。
Leader 先恢复 pending 工作，再规划新任务；复制经来源 Peer 的共享上传限速器，
检查大小/哈希后 PUT 目标。目标持久化返回匹配 Receipt 后，FSM 原子登记副本并完成任务。
失败任务保留记录，后续观察周期可创建新任务；提交结果未知时按原 ID 重放。
每周期最多处理一片，无删除；旧 Leader 的在途写最多产生额外不可变对象，不能绕过 quorum 发布。
复制由 Controller 协调执行 HTTP GET/PUT，Peer 的持久存储执行落盘；没有单独 Peer 任务执行守护进程。

任务本身跨 Controller 重启和 Leader 切换持久存在，容量仍由目标 Peer 原子配额检查兜底。
**没有跨并发导入的全局持久容量预留，也没有 GC、任务压缩或维修流量优先级。**
修复目标是最少两份有效副本，不会在故障后强制恢复所有可选第三副本。

## 放置策略与预算

Controller 环境变量 `PLACEMENT_POLICY=fixed|adaptive`，默认 fixed。
adaptive 根据实时空闲空间、Peer 配置上传上限、可用性排序；前 3 片提高带宽权重。
它是资源感知的导入时放置，不宣称已经实现观测热度学习或自动迁移现有视频。
`EXTRA_REPLICA_BYTES`（默认 0）是每个视频、每次放置计划的额外副本字节预算。
两种策略都先放每片两份，再在预算内按片段顺序增加第三份，比较时必须使用相同预算。
计划内扣减容量，但计划仍为 advisory；额外预算不是全局存储上限。

## 接口

所有服务数据接口以 `/api/v1` 开头。

| 接口 | 行为 |
|---|---|
| Peer GET `/inventory` | `PeerReport`：peer_id、instance_id、stored_bytes、quota_bytes、upload_bytes_per_second、receipts |
| Controller POST `/peers/{id}/heartbeat`、`/inventory` | 同一 PeerReport JSON；仅配置过的 Peer；Follower/无 quorum 返回 503 |
| Controller GET `/maintenance` | 最近上报及接收时间、Raft 任务映射；只有可读 Leader 应答 |
| Controller GET `/peers/{id}/tasks` | 目标为该 Peer 的 pending 任务 |
| Controller POST `/tasks/{id}/ack` | `{operation_id,receipt}`；向配置中的目标核对 Receipt，再提交完成状态；使用任务派生的稳定提交 ID |

`Task` 增加 size_bytes、sha256、reason，状态为 pending/completed/failed。
旧快照缺少 tasks 字段时按空任务集恢复。既有 Manifest/Receipt 字段不变。
仍只适用于可信 LAN/VPN；无身份认证，不能公开暴露管理接口。

## 故障验收

在 Linux/WSL、仓库根目录：

```bash
make test-repair
```

使用临时目录和随机端口：双副本导入 → 停第一个持有者 → 等任务 pending →
停 Leader → 新 Leader 恢复同一任务 → 完成后停第二个原始持有者 →
从第三个 Peer 下载并核验 SHA-256 → Controller 重启/切主后任务仍存在 → 无多数派拒绝 ACK。
不接管正在运行的 Console，也不删除用户视频。

对于正在使用的 Console，可通过管理服务注入其自有 Peer 故障：

也可以直接在网页视频库下方的“故障测试台”操作：Peer、Controller 和播放 API
均有挂掉/恢复/重启按钮；可一键挂掉当前 Leader、恢复全部已退出节点。
面板每 2 秒查询可读 Leader 的修复状态，显示多数派门槛，并可导出本次操作记录。
单 Controller 被挂掉后，管理服务仍可恢复它；测试自动选主须启动 3 Controller。
上传处理期间禁用故障操作，节点数据保留。操作记录最多 100 条，管理服务重启后清空。

```bash
python3 experiments/fault_injection.py stop peer-1 --backend console
python3 experiments/fault_injection.py start peer-1 --backend console
```

API 为 POST `/api/local/peers/{id}/stop|start`，只允许当前管理服务创建的 Peer。
统一节点接口为 POST `/api/local/nodes/{id}/stop|start|restart`，仅支持当前实例的
Peer、Controller 和 Demo API。`{expected_role:"Leader"}` 会在停止前核对角色，
避免状态过期时误挂 Follower。GET `/api/local/maintenance` 提供独立于 Demo API
的可读性、任务和操作日志；无 quorum 时返回 available=false，恢复接口仍可使用。
它保留节点数据。不要用宿主 Windows 的进程命令操作 WSL PID。
Docker 的旧命令保留，默认 `--backend docker`。

## 配对实验入口

```bash
make experiment-main
# 或小规模诊断，输出目录必须新建/为空
python3 experiments/paired.py --seeds 11 --output-dir results/my-diagnostic
python3 experiments/compare.py results/my-diagnostic/runs.jsonl
```

默认五个 seed、fixed/adaptive × round_robin/deadline 四组，共 20 个独立集群。
seed 实际生成多视频选择及到达时间；每组重置 Peer/Raft/counter/limiter，seed 内复用 trace。
Peer quota/upload cap、初始副本字节预算、窗口/并发/启动缓冲一致；预算不匹配会失败。
记录配置、代码 commit/dirty 标记、环境、素材清单哈希、事件、成功/失败会话、
初始/最终 Peer 计数、无删除情况下的存储峰值和修复任务。上传字节包含片段 GET 与维修 GET，
是应用层 ResponseWriter 字节，不含 HTTP/TCP 开销。

compare 按 seed 配对，检查 trace 与初始存储匹配，输出差值和 run 层 bootstrap 区间；
少于两对不输出区间。失败会话保留在完成率分母，卡顿均值明确只针对完成会话。
数据标记 controlled_experiment=true、formal_experiment=false：这是可复现的冷启动
headless 对比入口，还不是完整正式研究。没有 warm-up、低中高负载矩阵、定时故障 trace、
真实浏览器 QoE 或热度学习，不能直接声称策略普遍更优。

## 本机环境

本次使用 Ubuntu WSL、Go 1.24.0；原生 Windows 的目录 fsync 和 Unix 进程接口
仍不受支持。`.gitattributes` 固定 shell 脚本 LF 换行。视频播放使用已验证的 Edge；
原浏览器无法加载此素材的问题与 Peer 传输修复分开处理。
