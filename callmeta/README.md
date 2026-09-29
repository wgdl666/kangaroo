# callmeta

WG 跨服务调用的共享传播协议。消费者在 `go.mod` 中固定本模块的同一发布版本，生产构建不使用本地副本或路径替换。

- `traceparent` 和 `tracestate` 传递追踪上下文。按当前 Span 注入，不增加第二份 `trace_id` 头。
- `business_key` 和 `business_id` 成对关联特殊业务，例如 Prompt 草稿快照。
- 通用 `Propagator` 传递 `x-wg-business-line` 和 `x-wg-business-scene`。
- ModelHub 请求体承载 `business_metadata`。调用它时用 `RequestBusinessPropagator`，头中只保留追踪和特殊业务关联。
- `subscene` 属于当前模型步骤，由调用点设置并写入请求体，不放入通用传输头。
- `turn_id`、`task_id` 等业务对象 ID 留在各自业务协议和记录中。

异步任务用 `Capture` / `Restore` 或 `SaveTask` / `RestoreTask` 保存、恢复白名单。RabbitMQ 用 `Headers` / `FromHeaders`。不保存取消状态、deadline、认证头或任意 baggage。

每次发送使用新 carrier；`Inject` 不清理复用 carrier 上的旧键。HTTP 使用 `HTTPHeaders` 适配调试关联键，避免含下划线的请求头在网关被丢弃。

本地验证：`GOWORK=off go test -race -count=1 ./...`。消费者发布前以 `GOWORK=off go list -m -json github.com/wgdl666/kangaroo/callmeta` 核对真实版本和空 `Replace`，再运行对应服务测试。
