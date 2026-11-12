# 播放：Headless 与浏览器视频

TODO(Tian)：用代表性负载校准预测参数；正式实验仍需独立 trace、匹配资源预算与环境重置。

## 范围与配置

CLI、Demo API 共用 Go Engine。真实 HTTP 下载与 size/SHA-256 校验后，按媒体时长模拟消费；不保留媒体载荷、不解码，也不提供画面或音频。逻辑结果按 index 顺序消费，窗口有界。

- `strategy`：`round_robin`（默认）或 `deadline`；其他值报错。
- `window`：1–64，默认 3。
- `concurrency`：1–window，默认 2；REST 未指定时取 min(2, window)。
- `startup_buffer_ms`：0–3600000；0 表示首段准备好即可启动。正值等待足够时长的连续前缀；视频短于目标时等待整段视频。此前缀必须能放进 window，否则 Session 失败。
- Metadata HTTP 超时 4 秒，分片 HTTP 总超时 8 秒；CLI Session 默认 2 分钟，REST Session 10 分钟。

```bash
./bin/client -video my-video -strategy deadline \
  -window 4 -concurrency 2 -startup-buffer-ms 4000 \
  -timeout 5m -events-file .data/playback/my-video.jsonl
```

事件仍写 stderr，最终 Summary 写 stdout。`-events-file` 额外保存包含快照的 JSONL；写入失败取消播放并返回失败。

## 下载与策略

固定 worker pool 从按 index 入队的任务中取工作，因此默认请求顺序保持顺序播放所需的紧迫顺序；允许并发完成。Round Robin 按 Manifest index 旋转候选来源。请求失败、超时或校验失败后逐个切源，候选耗尽后强制刷新 Locations 再尝试一轮。缺少来源或最终失败会终止 Session，不无限等待。

Deadline 按预计完成耗时对副本排序，考虑分片大小、每源 EWMA 吞吐、该 Session 的同源并发、连续失败惩罚。吞吐初值 1 MiB/s，EWMA 新样本权重 0.3，固定开销 50 ms，失败惩罚每次 250 ms。它们是明确的预测默认值，不是性能结论。并发与样本仅属于本 Session，互不污染；这不是跨 Session 的 Peer 全局队列估计。

Deadline 请求预算取预计耗时的两倍与正的剩余时间的较小者，限制在 250 ms–8 s。过期请求保留恢复下限，避免所有副本因同一个过期时间立即失败。首播前采用请求起点的临时时间线；首播后采用媒体偏移加累计 stall 的调度时间线。下载进行中不会动态修改已经创建的请求超时。不使用 hedging。

顺序视频的请求顺序本就与 EDF 一致；当前 Deadline 的主要差异是选源和请求超时，不宣称实现了额外的 EDF 排序收益。

## 元数据缓存

Manifest、Locations 分别最多保留 128、4096 项，返回副本避免调用者修改共享缓存。5 秒内直接复用；刷新遇到网络错误、429 或 5xx 时，可使用获取后 5 分钟内的缓存。404、409、无效 JSON/数据及取消不触发旧值回退；明确错误会清除对应缓存。失败重试显式刷新 Locations。

缓存仅在进程内。冷启动、过期或从未查询过的片段仍需要可用 Controller；不保证 Controller 完全失联时任意视频都能播放。当前发布对象不可变，未来引入更新/删除时须重新设计失效规则。

## 指标与实时快照

事件回调串行执行；每个事件可携带一致的 Summary、Buffer、Peer 估计快照。消费和等待卡顿期间约每 250 ms 更新进度。Clock 接口允许测试注入媒体时间；HTTP 超时仍使用真实时间。

- Startup：Run 开始到启动缓冲前缀准备好，包含元数据查询和下载等待。
- Stall：首播后等待下一分片的时间，单独于 startup；正在等待时取消也计入已等待部分。
- Media elapsed：实际已模拟消费的时长，包括取消前的部分片段。
- Rebuffering：stall / (media elapsed + stall)，排除 startup。
- Verified bytes：所有成功验证的下载字节，包括最终尚未消费的预取段。相较旧版本仅统计消费循环已接收段的定义有所扩展。
- Downloaded bytes：本客户端实际读取的分片响应体字节，包含校验失败和中断尝试；不含 HTTP header、未读取的错误响应体或 TCP 重传，不能替代 Peer 侧网络成本。
- Requests / failed requests：分片 HTTP 尝试及失败数，不含元数据请求。
- Deadline misses：实际进入消费的非首段中，验证完成时间晚于首播起点加媒体偏移的数量。使用冻结时间线，不因 stall 移动；未消费/失败段不进入该计数，比较时必须同时报告完成率。

Buffer 展示 queued、downloading、retrying、ready、playing、failed，最多 window 个条目。它是逻辑状态，不是解码缓存。结束前会取消并等待所有 worker，最终统计包含它们已读取的字节。每次运行都发出终态事件，包括早期元数据失败和取消。

## Session、持久化与 SSE

Demo API 默认将 JSONL 保存到 `.data/sessions`，可用 `SESSION_DIR` 指定独立目录。每条事件携带恢复快照，并同步到磁盘；持久化错误会取消播放、在 REST 返回 `persistence_error`，最终状态为 failed。一个目录只能由一个 Demo API 进程使用。

重启恢复最近状态；未结束的 Session 标为 stopped，不会自动重播。截断的尾部 JSON 记录会被移除，再追加恢复事件；其他损坏会导致启动失败，避免静默丢失证据。

最多保留 100 个 Session，每个内存只保留最近 256 个事件。达到上限时回收最早的已结束 Session 及日志，全部活跃则返回 429。需要长期实验归档时，将日志复制到 results 或使用 CLI 独立输出。

SSE 使用 `Last-Event-ID` 仅发送更新事件。内存历史不足时从磁盘流式回放；内存模式无法补齐时返回 409 EVENT_HISTORY_EXPIRED。无效/超前序号返回 400；连接期间落后超过内存环时发出 reset 并关闭，客户端应重新获取快照。前端使用 EventSource 自动重连，并保留 REST 轮询恢复完整状态。


## 浏览器无缝播放

默认“视频播放”模式使用连续 fMP4 与 MSE，不再使用独立 MP4 换源或 Canvas 保帧。一个播放器生命周期仅创建一个 MediaSource URL 和 SourceBuffer；各片段按 segments 模式追加，使用文件本身的精确时间戳，不用 sequence 模式或逐段 timestampOffset 拼凑时间线。原生视频控件与页面进度条都对应整段视频。

准备流程：FFmpeg 只编码一次 H.264（不使用 B 帧）和 AAC，按关键帧输出 moof/mdat；fragment_mp4.py 拆分 BMFF box，给每个对象附上相同 ftyp/moov 初始化信息，保留原始 tfdt。每段起始视频样本必须为关键帧。stream.json 记录从视频关键帧推导的分片时间边界与精确 MIME codec；Manifest 新增可选 media:{mime_type,duration_ms}。各段 duration_ms 的和等于媒体总时长。编码只有一次，所以 AAC priming 不会在每个分片重新插入。

浏览器仍通过 Media API 获取完整、校验正确的对象，并再次验证 size/SHA-256；对象保持在有界 ArrayBuffer 窗口。下载并发和启动缓冲配置继续生效。追加完成后立即释放 JS 中的载荷，SourceBuffer 保留当前窗口和最多两个历史分片；播放进度驱动预取与回收，不能无限缓存整部视频。append/remove 串行执行，浏览器报告 QuotaExceededError 时尝试清理历史缓冲后重试一次。

跳转取消旧窗口的请求，并从目标关键帧所在片段重新填充；不会替换 video.src 或调用 load()。跳回已回收位置会重新下载，播放结束后的跳转可重新打开同一个 MediaSource。只有首次创建播放器与最终清理才装载/卸载媒体源。停止会取消 HTTP 请求、SourceBuffer 操作并撤销唯一的媒体 URL。

真实进度取自 currentTime，启动延迟到首次 playing 事件（包含有声播放授权等待），卡顿取自播放中的 waiting，排除手动暂停与跳转。浏览器字节数为成功验证的响应体；Peer 请求/失败切源统计来自已成功交付的响应，不替代全局网络成本。Headless Summary 不会用作真实播放指标。

只接受 continuous-fmp4-h264-v1 / continuous-fmp4-h264-aac-v1。旧 independent MP4 返回 422 REPACKAGE_REQUIRED，应从原视频用新 ID、新目录重新生成；fixture 返回 422 UNSUPPORTED_MEDIA。需支持清单 MIME codec 的 MSE 浏览器，以及 localhost/HTTPS 上的 Web Crypto。缺少支持时明确报错，不回退到会换源闪烁的实现。

媒体 lease 与模拟 Session 隔离，最多 100 个，空闲 30 分钟失效；服务器不缓存整部视频，单次传输受分片大小与并发上限约束。页面刷新后重新开始真实播放。MSE 消除了分片边界的源重载和编码时间线断裂；资源不足、下载落后于播放或用户跳到未缓冲位置时仍可能等待，不能保证任意网络下零卡顿。

格式参考：[W3C ISO BMFF Byte Stream Format](https://www.w3.org/TR/mse-byte-stream-format-isobmff/)。
