# `internal/repo/influxdb` — InfluxDB v2 时序库：历史数据写入与查询

上级文档：[服务总览 README](../../../README.md)

本包负责把采集到的点位值落到 InfluxDB，并向 `/history/data/query` 提供历史查询。**它是可选组件**：只有 `config.yaml` 里 `Influxdb.enable: true` 时才会初始化（仓库内附带的 `configs/config.yaml` **没有 `Influxdb` 段**，即默认关闭）。

> 依赖方向（注意有一处反常）：
> - 被 `cmd/server`（`New`/`Close`）和 `internal/handler/point.go`（`QueryByTagAndTime`）引用；
> - 本包**反向 import 了 `internal/service/collector`**，在自己的 `init()` 里注册采集订阅回调。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `influxdb.go` | 811 | 全部实现：`Connect_struct`、`initInfluxDB`、`Write`/`Flush`/`Close`、`New_Point`、`queryByTagAndTime`（+包装 `QueryByTagAndTime`）、`Read`、订阅回调 `a`、`Value_Type_Confirm` |

包内只有这一个文件，无 `_test.go`。

---

## 二、⚠️ 最容易踩的坑

1. **`page=0` 会让历史查询接口 panic（真实缺陷）。**
   `queryByTagAndTime` 的分页是 `startIdx := int((page - 1) * pageSize)`，`page`/`pageSize` 都是 `uint`：
   `page=0` 时 `0-1` 下溢成 `2^64-1`，乘 `pageSize` 回绕后转 `int` 得到**负数**，随后的 `allData[startIdx:endIdx]` 直接 `slice bounds out of range`。
   而 `handler/point.go` 的 `api_history_data_query` **只校验了起止时间，没有校验 `page >= 1`**，把请求里的 `page` 原样透传。
   触发条件：`pageSize > 0` 且 `page == 0`。
   最小修复（任选其一）：
   ```go
   // 方案 A：DAO 侧兜底（推荐，能同时保护将来的调用方）
   if page == 0 { page = 1 }
   // 方案 B：handler 侧校验
   if req.Page == 0 { ctx.Set("Response", []any{417, "page 从 1 开始"}); return }
   ```

2. **`Read()` 在消费结果流之前就 `cancel` 了查询上下文。**
   ```go
   queryCtx, queryCancel := context.WithTimeout(context.Background(), c.request_timeout)
   result, err = queryAPI.Query(queryCtx, fluxQuery)
   queryCancel()          // ← 流还没读完就取消
   for result.Next() { … } // ← 这里读的是同一个 HTTP 响应体
   ```
   Flux 查询是**流式**的，`Query()` 返回时只拿到了响应头+首块，`Next()` 才继续读 body。提前 cancel 会中断 body 读取，表现为"数据条数少于预期"或 `result.Err()` 报 context canceled。
   另外 `request_timeout` 同时也当查询超时用（配置项名字却是 `write_timeout`），语义混用。
   好在 `Read()` **对外不可达**（`c` 是包内私有变量，`Connect_struct` 没有导出实例），当前不影响生产；若将来要暴露批量历史读取，必须先改成 `defer queryCancel()` 并拆出独立的 `query_timeout`。

3. **`Value_Type_Confirm` 对绝大多数类型必然失败。**
   它按 `Value.(int8)` / `Value.(int16)` / `Value.(uint16)` … 逐个断言，但 InfluxDB 的数值只会以 **`int64` / `float64`** 返回，字符串是 `string`、布尔是 `bool`。
   → 传 `"int16"`、`"uint16"`、`"int32"`、`"uint32"`、`"int8"`、`"uint8"`、`"float32"` 时**恒返回"类型不是 xxx"**。
   只有 `"int64"`、`"uint"`/`"int"`（同理，实际拿不到）、`"float64"`/`"float"`、`"string"`、`"bool"` 有成功可能。
   附带两处复制粘贴错误：`case "int32"` 的报错文案写成"类型不是 int16"，`case "uint32"` 写成"类型不是 uint16"；`case "uint16"` 分支里多了一个无意义的提前 `return`。
   **结论：写回读时一律用 `"float"` 或 `"int64"` 做类型确认**，或改用 `conv` 包的通用转换（见 [pkg/conv README](../../../pkg/conv/README.md)）。

4. **"取第一个 `value_*` 字段"依赖 map 遍历顺序，结果不确定。**
   `queryByTagAndTime` 里：
   ```go
   for key, val := range item {
       if len(key) >= 6 && key[:6] == "value_" { mainValue = val; mainField = key; break }
   }
   ```
   同一个时间点如果存在多个 `value_*` 字段（例如同一 Tag 先后以 `int64` 和 `float64` 写入过——**字段名不同就是不同 field，不会互相覆盖**），随机取一个。表现为"历史值一会儿是整数一会儿是小数"。
   根治要么在写入侧固定单一 field，要么查询侧按 `mainField` 优先级显式排序取第一个命中。

5. **`measurement` 恒等于 `APP.Label`（设备标识），换 Label 等于换库。**
   `Write` / `queryByTagAndTime` / `Read` 都从 `config.Config.APP.Label` 取 measurement，空则报错。
   历史数据是按 measurement 组织的：**改了 `app.label`，之前写的所有历史数据立刻查不到**（不是丢，是查不到）。同样，Tag 为空的点位数据会被**静默跳过**并汇总成一条 `WARN InfluxDB 跳过 N 条未携带 Tag 的数据`。

6. **初始化失败不阻断启动，之后每批数据都会刷一条写入错误日志。**
   `cmd/server/main.go` 里 `influxdb.New()` 失败只 `log.Printf("ERROR InfluxDB 初始化失败: %v")`；`New` 内部失败时已经调过 `Close()`，把 `closed=true`。于是采集回调 `a` 每批都会拿到 `InfluxDB 客户端未连接` 并打印 ERROR。
   排障时看到大量这类日志，**先回去翻启动日志里的第一条 InfluxDB 错误**（配置不完整 / Ping 失败），别在运行期日志里找原因。

7. **`Write` 是"入缓冲区"而非"落库"，返回 `nil` 不代表数据已存。**
   `writeAPI.WritePoint()` 非阻塞：攒够 `buffer_size` 条 **或** 到 `flush_interval` 才发一次 HTTP；失败进重试缓冲区按指数退避重发，**超出 `max_retries` 后该批被丢弃**，错误只能从 `Errors()` 通道异步看到（`watchWriteError` 已持续消费并打日志）。
   需要"立即落盘"（查询前、退出前）请显式调 `Flush()`。服务退出路径里 `Close()` 内部会 `Flush`，因此**优雅退出时最后一段数据不丢**，`kill -9` 则会丢最多一个 `flush_interval` 的数据。

8. **「本批写入点数」统计点会多出一条时间线。**
   每批数据末尾追加一个虚拟点位：`Tag = //<设备标识>//write_quantity`（可由 `Influxdb.write_quantity_tag` 改名），`Value = len(value)`，`Type = conv.TYPE_INT`，`Time = time.Now()`。
   - 该 Tag 的生成依赖 `APP.Label`；**Label 未配置时统计点整体跳过**（只跳过它，真实点位照常写）。
   - 追加用的是 `value = append(value, …)`，可能改写调用方切片的底层数组。当前订阅方各自只读、且本包是唯一的 append 使用者，暂无副作用；新增订阅方时注意。
   - 统计点同样进 `Write`，因此 `write_quantity` 的时间线**也会计入历史查询**（如果拿它当点位查）。

9. **Flux 语句里的 Tag 必须转义，别绕过 `flux_escape`。**
   Tag 是用户可配置内容，含 `"` 或 `\` 会破坏查询。本包所有拼 Flux 的地方都走了 `flux_escape`（先 `\\` 后 `\"`，顺序不能反）。新增查询函数时**必须复用**它——这等同于 SQL 里的占位符。

---

## 三、数据模型

```
measurement  = APP.Label（设备标识，整批相同）
tags         = { "tag": <点位 Tag> }        // 常量 Tag_Key = "tag"
fields       = { "msg": <状态>, "value_<kind>": <值> }
timestamp    = v.Time
```

`New_Point` 按 Go 运行时类型分 field，**避免 InfluxDB 同 field 跨类型冲突**：

| Go 类型 | field 名 |
|---|---|
| `bool` | `value_bool` |
| `int/int8/int16/int32/int64` | `value_int` |
| `uint/uint8/uint16/uint32/uint64` | `value_uint` |
| `float32/float64` | `value_float` |
| `string` | `value_string` |
| 其他 | `value_string`（`fmt.Sprintf("%v", …)` 兜底） |

> 序列由 `measurement + tag` 唯一确定；InfluxDB 对「相同 measurement + tag + 时间戳」是**覆盖语义**，天然幂等，重传不会产生重复点。

---

## 四、配置项

```yaml
Influxdb:
  enable: true
  url: "http://127.0.0.1:8086"
  token: "<API token>"
  org: "GoEdge"
  bucket: "collector"
  write_timeout: 5000          # 毫秒；同时被当查询超时复用（见坑 2）
  buffer_size: 1000            # 单批点数阈值
  flush_interval: 1s           # 缓冲区刷新间隔
  max_retries: 3               # 写入失败最大重试次数，0=用默认值
  retry_interval: 5s           # 重试等待间隔（配合指数基数 2）
  retry_buffer: 0              # 重试缓冲区上限点数，0=自动算 batch_size*(max_retries+1)
  use_gzip: false              # 边缘带宽有限，建议开
  write_quantity_tag: ""       # 统计点 Tag；空=默认 //<APP.Label>//write_quantity
```

内置默认值（配置缺失或为 0 时兜底）：

```go
default_batch_size       = 1000
default_flush_interval   = 1 * time.Second
default_request_timeout  = 5 * time.Second
default_max_retries      = 3
default_retry_interval   = 5 * time.Second
default_exponential_base = 2        // 退避基数，未开放配置
```

`initInfluxDB` 的兜底顺序：`batch_size==0` → 1000；`flush_interval<=0` → 1s；`request_timeout<=0` → 5s；`max_retries==0` → 3；`retry_interval<=0` → 5s；`retry_buffer==0` → `batch_size*(max_retries+1)`（保证短暂断网不丢数据）。
统计点 Tag 先尝试 `model.TagWithDeviceId()` 归一化（允许写 `/标识符` 或 `//设备//标识符`），非法则回落默认值并告警。

四个必填项 `url/token/org/bucket` 任一为空 → `New()` 直接返回 `InfluxDB 配置不完整，请检查配置文件`。

---

## 五、对外 API

```go
func New() error                         // 仅当 Influxdb.Enable 时由 main 调用
func Close() error                       // 置 closed=true 并 client.Close()（内含 Flush），幂等
func Flush()                             // 强制立即落盘缓冲区
func QueryByTagAndTime(tag string, start, end time.Time, page, pageSize uint) (
        data []map[string]interface{}, total int64, err error)   // 历史查询，handler 唯一用到的读接口
func Value_Type_Confirm(Type string, Value any) (any, error)     // ⚠️ 见坑 3
```

包级私有但重要的实现：

| 符号 | 说明 |
|---|---|
| `var c Connect_struct` | 包内单例；因此 `Connect_interface` 只是文档性接口（且 `initInfluxDB` 是未导出方法，外部永远无法通过接口调用） |
| `init()` → `collector.Update_Subscriber(a)` | **无条件注册**；`a` 内部再判 `Enable` 早退，所以关闭时零开销 |
| `a(value []model.Value_type) error` | 采集流回调：追加统计点 → `c.Write(value)`；错误只记日志、**始终返回 nil**（不让 InfluxDB 故障影响采集链路） |
| `Write / queryByTagAndTime / Read` | 方法形式；`Read(scopes []Read_Scope_type)` 目前无人调用（见坑 2） |

`QueryByTagAndTime` 返回的每条记录形如：

```json
{ "Time": "2026-11-21T10:00:00+08:00", "Tag": "//0001//AI01", "Field": "value_float", "Value": 3.14, "Msg": "ok" }
```

`total` 是**时间范围内全部记录数（分页前）**，但实现方式是把区间内所有点读进内存再切片——**没有 Flux 侧的 `limit`**。长区间 + 高采集频率会一次性占住大量内存，这是当前最主要的性能天花板。

---

## 六、生命周期

```
main:  config.Influxdb.Enable ?
        └─ influxdb.New()
             ├─ 必填校验（url/token/org/bucket）
             ├─ 默认值兜底 + 统计点 Tag 归一化
             ├─ influxdb2.NewClientWithOptions + WriteAPI + QueryAPI（各建一次，全局复用）
             ├─ go watchWriteError(writeAPI.Errors())   ← 必须在首次写入之前调用
             └─ Ping(2s)  ← 失败只 WARN，不阻断（数据进重试缓冲区等重传）

采集:  collector 发布 → a(values) → [追加 write_quantity 统计点] → c.Write() → 客户端缓冲区
       → 攒够 buffer_size 或到 flush_interval → HTTP（可选 GZip）→ 失败进重试缓冲、指数退避

main(退出): step("InfluxDB", influxCloseOnce → Close())   // sync.Once 保护，client.Close() 内部 Flush
```

`mu sync.RWMutex` + `closed bool` 的作用是：**关闭后禁止写入**。向已关闭的写入 API 发数据会 panic，所以 `Write`/`Flush` 取读锁、`Close` 取写锁。

---

## 七、排障速查

| 现象 | 定位 | 处理 |
|---|---|---|
| 历史查询接口报 500 且日志有 `slice bounds out of range [-N:]` | 坑 1（`page=0`） | 前端保证 `page>=1`，或按方案 A 在 DAO 兜底 |
| 历史查询有时少数据 / `context canceled` | 坑 2（`Read` 提前 cancel） | 当前只影响未启用的 `Read`；启用前必修 |
| 历史值一会儿整数一会儿小数 / `Field` 名字变来变去 | 坑 4（多 `value_*` 随机取） | 规范写入类型，同一 Tag 不要跨类型 |
| `类型不是 int16` 但值明显是 int16 | 坑 3 | 改用 `int64`/`float` 做类型确认 |
| 改了 `app.label` 后历史全空 | 坑 5（measurement 变了） | 查询侧同步换 label，或数据迁移 |
| `WARN InfluxDB 跳过 N 条未携带 Tag 的数据` | 坑 5 | 采集链路没补 Tag，检查 `Point_Config.Tag` 是否为空/订阅顺序 |
| 断开 InfluxDB 后恢复，中间数据缺了一段 | 超出 `max_retries` 或 `retry_buffer` 上限 | 调大 `max_retries`/`retry_buffer`，或缩短 `flush_interval` |
| 刚写入就查不到 | 默认最多延迟一个 `flush_interval` | 查询前调 `Flush()` |
| `ERROR InfluxDB 客户端未连接` 刷屏 | 坑 6（启动期初始化失败） | 回溯启动日志第一条 InfluxDB 错误 |
| 启动日志 `WARN InfluxDB 统计点 Tag 无法生成：设备标识(APP.Label)未配置` | `APP.Label` 为空 | 配置 label，或显式给 `write_quantity_tag` |
| `WARN InfluxDB 连接探测失败(数据将进入重试缓冲区等待重传)` | InfluxDB 未就绪 | 属预期告警，服务起来后会自动重传 |

---

## 八、扩展指引

- **新增查询函数**：一律基于包内单例 `c`，Flux 字符串里所有外部输入必须过 `flux_escape`，`queryCtx` 用 `defer cancel()` 且必须覆盖到 `result.Next()` 循环结束，`result.Close()` 在循环后显式调用（循环内不许用 `defer`）。
- **想支持多 measurement / 多 bucket**：把 `measurement` 从 `config.APP.Label` 提成函数参数，注意 `queryByTagAndTime` 的返回结构里 `Tag` 是回填的入参而非查询结果。
- **想把 `Read` 开放给上层**：先修坑 2，并新增导出实例（例如 `func Default() *Connect_struct` 或直接把方法改为包级函数），同时把 `Connect_interface` 的 `New(key)` 类签名问题一并清理。
- **加入"按时间跨度自动降采样"**：在 Flux 里加 `|> aggregateWindow(every: …, fn: mean)`，可顺带缓解第 5 节提到的"全区间读进内存"问题。

---

## 九、相关文档

- [服务总览 README](../../../README.md)
- [internal/config](../../config/README.md) — `Influxdb.*` 与 `APP.Label`
- [internal/model](../../model/README.md) — `Value_type` 与 `TagWithDeviceId`
- [internal/handler](../../handler/README.md) — `/history/data/query` 入口
- [internal/service/collector](../../service/collector/README.md) — 采集流订阅机制（`Update_Subscriber`）
- [internal/repo/tsdb](../tsdb/README.md) — 另一套（未启用）本地嵌入式时序方案
- [pkg/conv](../../../pkg/conv/README.md) — 类型转换（替代 `Value_Type_Confirm` 的推荐做法）
