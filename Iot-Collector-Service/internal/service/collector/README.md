# `internal/service/collector` — 实时数据核心：采集总线、值变化检测、报警 / 历史判断、写值标准入口

上级文档：[服务总览 README](../../../README.md)

本包是整台采集器的**数据枢纽**。驱动采到的原始点位值在这里变成三件事：**实时值内存快照**、**报警事件**、**历史记录事件**；前端与 API 的写值请求也在这里收敛成一条标准链路。它不含任何协议细节（见 [service/driver](../driver/README.md)），也不直接建 SQL（见 [repo/mysql](../../repo/mysql/README.md)）。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `db_point.go` | 268 | 泛型发布/订阅总线 `bus[T]`；实时值 `Value_Map`；值变化检测 `Update_Value_Judgment_*`；`collectionBus` / `updateBus` 两条主干总线的接线 |
| `alarm.go` | 643 | 报警配置三级缓存（freecache → MySQL → 内存 map）；expr 表达式编译缓存与求值；报警事件写 `Alarm_Data` |
| `history.go` | 210 | 历史配置内存缓存；历史事件判断与 `historyBus` 发布 |
| `write.go` | 245 | 写值标准入口：`PrepareWrite`（定位 + 类型转换）与 `WriteValues`（按驱动分组下发）；哨兵错误定义 |
| 合计 | 1366 | — |

包内符号规模：导出函数 20 个、导出类型 6 个、导出变量 5 个（`Value_Map`、`Value_Map_Mu`、`Alarm_Config`、`Alarm_Config_RWMu`、`History_Config`、`History_Config_RWMu`——后四者成对计）。

---

## 二、⚠️ 最容易踩的坑

1. **接线全部发生在 `init()` 里，靠 import 触发。**
   `db_point.go:257` 订 `collectionBus → 值变化检测`；`history.go:207` 订 `collectionBus → 历史判断`；`alarm.go:600` 订 `updateBus → 报警判断` 与 `alarmBus → MySQL 写入`。任何一条链路的包没被 import，对应功能就**静默失效**（不报错、不日志）。

2. **`Value_Map` 和它的锁都是导出的。**
   `internalmqtt/publish.go` 直接 `collector.Value_Map_Mu.RLock()` + 遍历 `collector.Value_Map`。新增读取方要么复用这个模式，要么走 `Value_Map__Query_list`，**不要**不加锁直接读 map。

3. **值变化检测用 `!=` 比较 `any`，遇到不可比较类型会 panic。**
   `db_point.go:210`：`if new.Value != old.Value || new.Msg != old.Msg`。`Value` 是 `any`，若动态类型是 slice / map（驱动层或 MQTT 通路传进来的原始 JSON 解到 `any` 时会出现 `[]any`/`map[string]any`），这一行直接 `runtime error: comparing uncomparable type`。panic 会被 `bus.Publish` 的 recover 兜住（见坑 4），表现为**这一批采集数据整批消失** + 一条 `ERROR 订阅者回调 panic` 日志。当前驱动层只产出标量，所以线上不触发；接入新协议或让写值链路回灌时务必小心。

4. **`bus.Publish` 恒返回 `nil`，错误只进日志。**
   `db_point.go:96-122`：订阅者的 `error` 与 `panic` 都只打印日志，发布方永远拿到 `nil`。所以 `Collection_Publisher` / `Update_Publisher` / `History_Publisher` 的返回值**没有诊断价值**，排障只能看日志。

5. **`Subscribe` 返回的 `unsubscribe` 全项目无人接收。**
   取消订阅是把槽位置 `nil`（不是删除），意味着**订阅者无法真正卸载、也不会泄漏索引**，但也意味着"注册了就一直活着"。`Collection_Subscriber`/`Update_Subscriber`/`Alarm_Subscriber`/`History_Subscriber` 四个包装函数都丢弃了返回值。

6. **`updateBus` 是异步发布，批次之间可能乱序。**
   `db_point.go:246-248`：值变化列表通过 `go func() { _ = Update_Publisher(update_list) }()` 在锁外发布。多个采集周期并发时，**报警判断与 InfluxDB 写入看到的顺序可能与实际采集顺序不符**。同一批次内部仍是同步串行的。

7. **改点位 `Tag` / `Type` 后，`Value_Map` 里的旧值不会刷新。**
   `Update_Value_Judgment_single` 只在**首次出现**时写入 `Tag`/`Type`（`db_point.go:192-207`）；已存在点位只覆盖 `Value`/`Time`/`Msg`/`Last_*`（`210-218`）。而 InfluxDB 的历史序列是按 `Tag` 建的 → 改 Tag 之后新数据仍写到旧 Tag 上。改 Tag 后需重启进程（或手工删掉 `Value_Map` 对应键）才能生效。

8. **`PrepareWrite` 里的 `reflect.Convert` 会 panic。**
   `write.go:140`：`realValue = reflect.ValueOf(realValue).Convert(t).Interface()`。三种典型入参直接崩：
   - `Value` 传 `null` → `realValue` 为 `nil` → `reflect.ValueOf(nil)` 是零 `Value` → `Convert` panic；
   - `Type: "int16"` 而 `Value: "abc"` → `string` 不可转换为整数 → panic；
   - `Type: "bool"` 而 `Value: 1.5` → `float64`→`bool` 非法 → panic。

   panic 发生在 handler 的 goroutine 里，被 Gin Recovery 中间件接住 → 500。同时**超范围静默截断**：`Value: 99999` + `Type: "int16"` 会得到截断后的值而不是报错。改造建议：先判 `reflect.ValueOf(realValue).IsValid()`，再用 `ConvertibleTo` 预检并把溢出显式化为错误。

9. **`WriteValues` 是"部分写入"，不是原子批。**
   `write.go:188-196`：按 `Drive_Id` 分组后 `for driveId := range driveGroups` 遍历——**map 遍历顺序随机**，且**第一个失败就 `return`**。多驱动写值时：已成功下发的驱动不会回滚，未尝试的驱动直接跳过，而且每次失败时"哪些驱动被跳过"都不确定。要整批语义必须在上层自己拆成单点请求。

10. **报警规则里 `Group == 0` 等于禁用该规则。**
    `alarm.go:471-473` 直接 `return Alarm_type{}, false`，无日志。前端建规则时漏填 Group（或默认 0）会得到"配了报警但永远不触发"的现象。同理 `Config == ""` 或 `Config == "null"` 也是静默跳过（`468-470`）。

11. **回源 MySQL 会把运行期报警状态重置成 `"正常"`。**
    `alarmApplyMysqlConfigs`（`alarm.go:169-203`）与 `Alarm_Config__Reload`（`343-389`）构造 `Alarm_Config_type` 时 `Status` 硬编码 `"正常"`。点位正在报警时，只要 freecache 条目过期（默认 `ALARM.Config_CacheTTL = 1h`）触发一次回源，内存里的 `"报警"` 状态就被抹掉，下一轮判断又会当作正常处理。`Time` 同理被清零。

12. **`Alarm_Config__Reload` 存在数据竞争。**
    `alarm.go:353`：`alarmProgramCache = sync.Map{}`。`sync.Map` 的零值整体赋值**不是**并发安全操作，而 `getOrCompileAlarmProgram` 可能同时在读。正确写法是 `alarmProgramCache.Range(func(k, _ any) bool { alarmProgramCache.Delete(k); return true })`。这条会在"改报警配置 + 采集高峰期"同时发生时被 `-race` 抓到。

13. **报警事件没有状态跃迁去重，每次值变化都写一行 `Alarm_Data`。**
    `alarmJudgmentWithCfgs`（`alarm.go:522-542`）对**每条能判断的规则**都产出一个事件（`报警` 或 `恢复` 都算）。由于 `updateBus` 只在值变化时发布，一个持续越限、数值不断跳动的点位会**每个采集周期写一行** `Alarm_Data`。若要"只在正常↔报警跃迁时记录"，需按 `cfg.Status != a.Status` 过滤。

14. **`Alarm_Config__Status_Update` 的返回值被忽略。**
    `alarm.go:534` 不接 `error`。坑 11 之后内存里若没有该 `Config` 字符串的条目（例如并发改了配置），状态更新会静默失败。

15. **历史记录只有内存单级缓存，没有回源。**
    `history.go:36-56` 的 `History_Config` 只由 `History_Config__Reload`（`93-116`）与 `History_Config__Add` 填充。新增历史配置后若没调 `/config/history/reload` 之类的入口，该点位**永远不会进历史**，且与报警不同——这里连一次回源兜底都没有。

16. **`ALARM.Config_CacheTTL: 0s` 并不是"永久保存"。**
    `alarm.go:53-59` 的注释与 `configs/config.yaml:90` 的行内注释都写"0=永久保存不过期"，但 `internal/config/config.go:360-361` 会在归一化阶段把 0 改回 `1h`。**配 0 和不配效果完全一样**，想永久只能给一个很大的值（如 `8760h`）。

---

## 三、总线与数据结构

### 3.1 泛型总线 `bus[T]`（`db_point.go:68-122`）

```go
type bus[T any] struct {
    mu          sync.RWMutex
    subscribers []func(T) error
}
func (b *bus[T]) Subscribe(fn func(T) error) (unsubscribe func())
func (b *bus[T]) Publish(v T) error   // 恒返回 nil
```

| 特性 | 实现细节 | 后果 |
| --- | --- | --- |
| 并发安全 | `Publish` 先 `RLock` 拷贝订阅者快照，再解锁逐个调用 | 持锁期间不执行回调，回调里可安全反向 `Subscribe` |
| 异常隔离 | 每个回调包一层 `func(){ defer recover() }()` | 单个订阅者 panic 不影响其他订阅者，进程不挂 |
| 取消订阅 | 把对应下标置 `nil`，`Publish` 时跳过 | 切片只增不减；调用方全部忽略了返回值，实际未使用 |
| 顺序 | 按 `Subscribe` 调用顺序同步执行 | 但 `init()` 的执行顺序由文件名字典序决定，见 3.2 |
| 错误传播 | 只打日志 | 上层无法感知下游失败 |

### 3.2 四条总线与接线关系

| 总线 | 变量 | 载荷 | 发布入口 | 订阅者 | 注册处 |
| --- | --- | --- | --- | --- | --- |
| 采集 | `collectionBus` | `[]model.Value_type` | `Collection_Publisher`（驱动 / main 调用） | `Update_Value_Judgment_list`、`History_Judgment_list` | `db_point.go:259`、`history.go:209` |
| 变化 | `updateBus` | `[]model.Value_type` | `Update_Publisher`（由值变化检测在 goroutine 里调用） | `Alarm_Judgment_list`（带 PointId≠0 且 Value≠nil 过滤）、InfluxDB、WS 推送、MQTT 上行 | `alarm.go:602`；外部见 `repo/influxdb/influxdb.go`、`handler/point.go`、`internalmqtt/publish.go` |
| 报警 | `alarmBus` | `[]Alarm_type` | `Alarm_Publisher` | 写 `mysql.Alarm_Data__Add` | `alarm.go:629` |
| 历史 | `historyBus` | `[]History_Value_type` | `History_Publisher` | 下游时序存储（当前仅 InfluxDB 侧） | 各下游包自身 `init()` |

> 注意 `init()` 顺序：同一包内按文件名排序执行，`alarm.go` < `db_point.go` < `history.go` < `write.go`。`alarm.go:602` 里订阅的是 `updateBus`，而 `updateBus` 是零值 `bus`（无需初始化即可用），因此不依赖 `db_point.go` 的 `init()`——这点是刻意的，`bus` 用零值 + `make` 惰性策略避开了包内 init 顺序问题。

### 3.3 实时值快照（`db_point.go:26-60`）

```go
var Value_Map map[uint]Update_Value_type   // key = PointId
var Value_Map_Mu sync.RWMutex

type Update_Value_type struct {
    model.Value_type          // PointId / Tag / Value / Type / Time / Msg
    Last_Value any            // 上一次的值
    Last_Time  time.Time      // 上一次值变化的时间
}
```

- 只增不删：点位被删除后 `Value_Map` 中的条目不会清理，直到进程重启。
- `Value_Map__Query_list(keys)` 按 keys 顺序返回命中的条目，不存在的 id 静默跳过（返回值长度 ≤ len(keys)，**不能按下标对齐**）。

### 3.4 报警配置三级缓存（`alarm.go:37-259`）

| 层级 | 载体 | key | 命中条件 | 未命中动作 |
| --- | --- | --- | --- | --- |
| L1 | `alarmCache`（freecache，**固定 100MB**） | `alarm:%d`（PointId） | JSON 反序列化成功；空结果哨兵 `"[]"` 同样算命中 | 进 L2 |
| L2 | MySQL `Alarm_Config` | `Point_Id IN (...)` | 查询成功 → `alarmApplyMysqlConfigs` 回填 L1 与 L3 | 失败 → 进冷却并降级 L3 |
| L3 | 内存 `Alarm_Config map[uint][]Alarm_Config_type` | PointId | 存在且非空 | 返回"无配置" |

关键参数（`alarm.go:62-82`）：

| 常量 | 值 | 作用 |
| --- | --- | --- |
| `alarmNegativeTTL` | `60s` | 空结果哨兵缓存时长；无报警配置的点位占多数，用来压制逐周期回源 |
| `alarmDbBackoff` | `10s` | MySQL 查询失败后的冷却期，期内整体降级到 L3 |
| `alarmQueryChunk` | `500` | 单次 `IN` 查询的点位上限 |
| `alarmCache` 容量 | `100 * 1024 * 1024` | 硬编码，不走配置 |
| L1 TTL | `config.Config.ALARM.Config_CacheTTL` | 默认 `1h`（见坑 16） |

故障日志是"边沿触发"的：`alarmDbDown` 保证进入故障只打一条 `ERROR`，`alarmDbRecover` 保证恢复只打一条 `INFO`（附带故障期间失败次数），避免每个采集周期刷日志。

### 3.5 报警表达式规范（`alarm.go:440-517`）

- 引擎：`github.com/expr-lang/expr`。
- 唯一变量：`x`，值即 `new.Value`，**动态类型就是点位实际类型**（`env = map[string]any{"x": new.Value}`）。
- 期望写法：返回布尔或可用 `conv.AnyToBool` 折算的值（数字非 0 → 报警）。例：`x > 100`、`x == 0`、`x matches "^ERR"`。
- 编译缓存 `alarmProgramCache sync.Map`，**key 是 `AlarmId` 而不是表达式字符串**：相同表达式配在不同点位上会各自编译一次——因为首次编译时的 `x` 实际值会被 `expr.Env` 用于类型推断，混用可能导致类型不符。
- 首次编译会**固化 `x` 的类型**：若首个采到的值是 `int64`，之后同一规则遇到 `string` 值会走 `表达式执行失败` 分支，`Status` 被写成错误文本。
- 结果映射：编译失败 → `Status = compileErr.Error()`；执行失败 → `Status = runErr.Error()`；真 → `"报警"`；假 → `"恢复"`。四种情况**都会产生事件**（除坑 10 的跳过分支）。

### 3.6 历史判断（`history.go:149-205`）

```go
type History_Config_type struct{ PointId uint; Config string }
```

启用条件三连（任一不满足即静默不记录）：`PointId != 0` → `History_Config` 里有该点位 → `Config` 非 `""` 且非 `"null"`。
`Config` 内容**不参与任何逻辑**，只当作"是否启用"的开关。
之后用 `conv.ConvBool(new.Value, new.Type)` 做值/类型一致性校验，不通过则打一条 `ERROR 历史记录类型不匹配` 并放弃记录——这是 history 里唯一有日志的失败分支。

### 3.7 写值标准入口（`write.go`）

API 与内部 MQTT 两条通路共用，分两段：

```
WriteInput{Tag | PointId, Value json.RawMessage, Type string}
        ↓ PrepareWrite
   Tag 批量反查 id（writeIdsByTags）→ 缺 Tag 的整批报错
   只给 PointId 的批量反查 Tag（writeTagsByIds）
   json.Unmarshal → reflect.Convert 到目标类型 → conv.ValueTypeCode 编码
        ↓ []model.Value_type（与入参等长、顺序一致）
   WriteValues
   mysql.Point_Config__Query__DriveIds → 按 Drive_Id 分组
        ↓
   driver.DriveWrite(driveId, vals)   // 只读校验在 DriveWrite 入口统一做
```

- **`Tag` 优先**：同时给了 `Tag` 与 `PointId` 时以 `Tag` 反查出的 id 为准（`write.go:106`）。
- `Type` 的合法取值 = `writeValueTypeMap` 的 15 个键：`bool`、`int8/uint8`、`int16/uint16`、`int32/uint32`、`int64/uint64`、`int/uint`、`float32/float64/float`、`string`。未知类型 → `ErrWriteInvalid`。
- 返回数组顺序与入参一致，调用方可按下标回填原始 JSON 做审计日志。

| 哨兵错误 | 触发场景 | handler 映射 |
| --- | --- | --- |
| `ErrWriteNotFound` | Tag 反查不到 id；PointId 无对应驱动关系 | `404` |
| `ErrWriteInvalid` | Tag 与 PointId 同时为空；类型不存在；值 JSON 解析失败 | `417` |
| `driver.ErrPointNotWritable` | 点位只读，被 `DriveWrite` 拒绝（原样透传，`errors.Is` 可判） | `417` |
| 其他 `error` | MySQL 查询失败、驱动下发失败等基础设施错误 | `500` |

> 注意：定位/转换阶段**任一**条目出错即整批中止（`PrepareWrite` 返回 error，不产出部分结果）；下发阶段则是坑 9 的"部分成功"。两者语义不一致，是刻意的——前者是无副作用的校验，后者已经产生设备副作用。

---

## 四、核心流程

### 4.1 采集 → 分发 → 报警 / 历史

```
驱动协程（modbus / siemens）
        │  onData → collector.Collection_Publisher([]model.Value_type)
        ▼
collectionBus.Publish
   ├─→ Update_Value_Judgment_list          [db_point.go:230]
   │       Value_Map_Mu.Lock()  整批一次
   │       逐条 Update_Value_Judgment_single：新键 / Value|Msg 变化 → 记入 update_list
   │       Unlock()
   │       └─ go Update_Publisher(update_list)         ← 异步、可能乱序（坑 6）
   │              ▼
   │        updateBus.Publish
   │           ├─→ 报警过滤(PointId≠0 && Value≠nil) → Alarm_Judgment_list   [alarm.go:602]
   │           │      alarmConfigFetch(ids)  ← freecache→MySQL(≤500/批)→内存 三级
   │           │      逐点 alarmJudgmentWithCfgs：
   │           │         expr 求值 → Status(报警/恢复/错误文本)
   │           │         Alarm_Config__Status_Update（错误被忽略，坑 14）
   │           │      Alarm_Publisher → alarmBus → mysql.Alarm_Data__Add(...)
   │           ├─→ InfluxDB 时序写入（repo/influxdb 的 init 订阅）
   │           ├─→ WebSocket 前端推送（handler/point.go）
   │           └─→ MQTT 上行（internalmqtt/publish.go）
   └─→ History_Judgment_list               [history.go:189]
          逐条 History_Judgment（读内存 History_Config + ConvBool 校验）
          └─ History_Publisher → historyBus → 下游存储
```

关键点：**历史记录挂在 `collectionBus` 上（每个采集周期都判），报警挂在 `updateBus` 上（只在值变化时判）**。所以"值不变也要周期性落历史"是成立的，而"值不变时报警状态不再刷新"也是成立的。

### 4.2 配置变更 → 重载

`/config/*` 的增删改走 [handler/config.go](../../handler/README.md)：落库成功后调用
`collector.Alarm_Config__Reload()` / `collector.History_Config__Reload()`，
两者都是「全量拉 MySQL → 内存建 map → 加写锁整体替换」，报警额外做 `alarmCache.Clear()`。
`Alarm_Config__Add` / `History_Config__Add` 是**增量追加**语义（`append` 到点位切片），只供启动阶段与内部通路使用，前端改配置不要用它（会重复追加）。

### 4.3 写值链路

见 3.7。API 侧完整链路（含鉴权、审计日志落库）在 [handler README](../../handler/README.md) 的 `/point/write/value` 一节；MQTT 侧在 [internalmqtt README](../internalmqtt/README.md) 的下行订阅一节。

---

## 五、排障速查

| 现象 | 先查 | 定位 / 处理 |
| --- | --- | --- |
| 前端实时值不动，但采集日志正常 | `Collection_Publisher` 是否被调用；`Value_Map` 是否有该 PointId | 驱动 `onData` 回调是否注入（[service/driver](../driver/README.md) 坑：`CreateDriver` 覆盖实例） |
| 日志出现 `ERROR 订阅者回调 panic` | panic 栈里的类型名 | 大概率是坑 3（`any` 不可比较）；确认驱动产出的 `Value` 是标量 |
| 报警完全不触发 | 规则 `Group` 是否为 0；`Config` 是否为空/`"null"`；freecache 里是否有负缓存 | 坑 10；改完配置确认调了 Reload（坑 15 同理适用于历史） |
| 报警状态莫名变回"正常" | `Alarm_Config__Status_Update` 与回源时刻 | 坑 11；调大 `ALARM.Config_CacheTTL` 或修正回源逻辑保留 Status |
| `Alarm_Data` 表暴涨 | 该点位值是否高频抖动；规则条数 | 坑 13；加状态跃迁去重 |
| 报警写值 500 且日志是 Gin Recovery 栈 | 请求体里 `value` 是否为 `null` / 与 `type` 不匹配 | 坑 8；入参先做类型合法性校验 |
| 多点位跨驱动写值"只写进去一部分" | 返回的错误是哪个 driveId | 坑 9；上层拆成单点写或实现回滚 |
| 改了点位 Tag 后历史数据查不到 | `Value_Map` 里该点位的 `Tag` 字段 | 坑 7；重启进程或清理该键 |
| `ERROR 查询报警配置失败，10s 内不再回源 MySQL` | MySQL 连通性 / `Alarm_Config` 表是否存在 | 见 [repo/mysql README](../../repo/mysql/README.md)；这是降级保护，不是 collector 的 bug |
| `-race` 报在 `alarmProgramCache` | 是否与 `Alarm_Config__Reload` 并发 | 坑 12；改用 `Range+Delete` |
| 历史某点位一直不落库 | `History_Config` 里有没有该 PointId；`Config` 是否非空 | 坑 15；`ERROR 历史记录类型不匹配` 则是值与声明 `Type` 不符 |
| 配了 `Config_CacheTTL: 0s` 想永久缓存 | 日志里 TTL 是多少 | 坑 16；0 会被归一化成 1h |

---

## 六、扩展指引

**新增一类"值变化后要做什么"（例如推 Kafka）**
1. 在 `alarm.go` / `history.go` 之外新建文件，`func init() { updateBus.Subscribe(yourHandler) }`；
2. 回调签名必须是 `func([]model.Value_type) error`；返回值只会被打日志，失败请自带重试；
3. 想"每个采集周期都跑"就订 `collectionBus`，想"值变了才跑"就订 `updateBus`；
4. 若放在**别的包**，注意 `repo/influxdb` 那种"repo 反向 import service"的分层破例已被记录在 [influxdb README](../../repo/influxdb/README.md)，不要再增加此类依赖。

**新增一种写值类型**
同步改三处：`writeValueTypeMap`（`write.go:55`）、`conv.ValueTypeCode`（[pkg/conv](../../../pkg/conv/README.md)）、驱动侧的类型编码表（[modbus](../../driver/modbus/README.md) / [siemens](../../driver/siemens/README.md)）。

**改造建议清单（按收益排序，均已在上文标注行号）**
1. `bus.Publish` 返回聚合错误，或至少返回首个错误 + 失败计数（坑 4）；
2. `Update_Value_Judgment_single` 改用 `reflect.DeepEqual` 或按 `Type` 分派比较（坑 3、7）；
3. `PrepareWrite` 用 `IsValid()` + `ConvertibleTo()` + 溢出检查替代裸 `Convert`（坑 8）；
4. `WriteValues` 改为收集全部错误后 `errors.Join` 返回，并按 driveId 排序保证可复现（坑 9）；
5. `alarmApplyMysqlConfigs` / `Reload` 保留同 `AlarmId` 的既有 `Status`/`Time`（坑 11）；
6. `Reload` 改 `Range+Delete` 清理表达式缓存（坑 12）；
7. 报警加状态跃迁去重（坑 13）；
8. `History_Config` 增加与报警同构的按需回源（坑 15）。

---

## 七、相关文档

- [服务总览 README](../../../README.md)
- [cmd/server](../../../cmd/server/README.md) — `collector.New()` 与 `Collection_Publisher` 的最初注入点、优雅退出顺序
- [internal/service/driver](../driver/README.md) — 上游数据来源与 `WriteValues` 的下游
- [internal/repo/mysql](../../repo/mysql/README.md) — `Alarm_Config` / `Alarm_Data` / `History_Config` / `Point_Config` 的表结构与查询封装
- [internal/repo/influxdb](../../repo/influxdb/README.md) — `updateBus` / `historyBus` 的主要订阅者
- [internal/handler](../../handler/README.md) — `/point/write/value`、`/point/realtime`、`/config/*/reload`
- [internal/service/internalmqtt](../internalmqtt/README.md) — MQTT 上行订阅 `updateBus`、下行写值调 `WriteValues`
- [internal/service/configsync](../configsync/README.md) — 配置中心下发后触发 Reload
- [internal/model](../../model/README.md) — `Value_type` 定义
- [internal/config](../../config/README.md) — `ALARM.Config_CacheTTL` 的读取与归一化
- [pkg/conv](../../../pkg/conv/README.md) — `ValueTypeCode`、`ConvBool`、`AnyToBool`
