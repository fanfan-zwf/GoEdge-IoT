# `internal/service/driver` — 驱动管理器：实例生命周期、按驱动加锁、写值统一校验

上级文档：[服务总览 README](../../../README.md)

本包是"驱动类型字符串 ↔ 驱动实例 ↔ 业务调用"的唯一枢纽：把 MySQL 里的 `Drive_Config` 变成活的采集协程，并向 HTTP / 内部 MQTT 两条通路提供统一的驱动操作入口。

它**不含任何协议实现**——Modbus 与 S7 的细节在 [internal/driver/modbus](../../driver/modbus/README.md)、[internal/driver/siemens](../../driver/siemens/README.md)。

> 依赖方向：`cmd/server`、`handler/config.go`、`handler/point.go`、`service/collector/write.go`、`service/internalmqtt/subscribe.go` → 本包 → `internal/driver/*` + `internal/repo/mysql` + `internal/model`。
> 采集数据回流靠 **main 注入回调**（`driver.Start(collector.Collection_Publisher)`），因此本包**不 import collector**，避免 import cycle。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `init.go` | 83 | `Start(onData)`（sleep 1s → `InitializeDrivers`）、`InitializeDrivers`（查全量驱动/点位 → 按 Drive_Id 分组 → 逐个初始化）、`initSingleDriver` |
| `manager.go` | 401 | `driverEntry`、`DriverManager`、`Manager` 单例、`CreateDriver`、`DriveNew/Connect/Close/ResetConfig/UpdatePoints/Write`、`IsDriverBusy`、`CloseAll`、`ErrPointNotWritable`、`RW_Cancel` 常量 |

---

## 二、⚠️ 最容易踩的坑

1. **每个驱动操作都会额外"占锁 3 秒"，驱动多时启动极慢。**
   ```go
   func unlockDrive(l *sync.Mutex) { time.Sleep(3 * time.Second); l.Unlock() }
   ```
   所有 `DriveNew / DriveConnect / DriveClose / DriveResetConfig` 都用 `defer unlockDrive(l)`。`defer` 在函数返回时执行，**sleep 期间锁仍被持有**，`TryLock` 的并发请求会立刻失败并得到"驱动 id=N 正在操作中，请稍后再试"。
   启动路径 `initSingleDriver` = `DriveNew`(3s) + `DriveConnect`(3s) = **每个驱动串行 6 秒**，10 个驱动要 60 秒才采起来。这不是 bug（是防抖设计），但排障时"启动后很久没数据"要先想到它。
   改造建议：把 sleep 移出临界区（先 Unlock 再丢弃锁句柄），或把防抖时间做成配置。

2. **`DriveResetConfig` 的总耗时 ≈ 6 + 3 + 3 = 12 秒，而 handler 提示"预计 9 秒完成"。**
   `Close()` → `Sleep(6s)` → `New()` → `Sleep(3s)` → `Connect()`，最后还要走坑 1 的 `defer unlockDrive`（再 3 秒）。
   handler 的 `/config/drive/restart` 是**异步**的，响应只代表"已提交"。前端等到 9 秒去查会发现驱动仍显示"正在重启"——再等 3 秒。

3. **`DriveUpdatePoints` 不加驱动锁。**
   它是唯一**绕过 `tryLockDrive`** 的写路径操作：直接 `getEntry(id)` 后 `e.UpdatePoints(Points)`。
   后果：改点位（`handler/config.go:250`）与 `DriveResetConfig`/`DriveWrite` 可并发进入驱动内部，而驱动的 `UpdatePoints` 会重组轮询包、替换 `Points` 切片。Modbus 的组包与轮询协程之间存在竞态（`packet.go` 与轮询 loop 共享同一份点表）。
   现状能大致相安无事，是因为 Modbus 的 `UpdatePoints` 只做整包替换。**一旦驱动改成增量修改，这里就是数据竞争点**，建议补锁。

4. **`CreateDriver` 覆盖已有 entry 时不会 `Close` 旧实例。**
   `Manager.drivers[driveId] = entry` 直接赋值。若对同一 id 重复 `CreateDriver`（`DriveResetConfig` 只在 entry 不存在时才建，正常路径没问题），旧实例的连接与轮询协程会**泄漏**——它仍持有回调并继续发布数据，且再也拿不到它的句柄去关闭。
   同理：删除驱动时如果只调 `DriveClose` 而不从 map 里摘除 entry，实例会一直留在管理器里（`CloseAll` 退出时仍会再 Close 一次，无害但占内存）。

5. **`Manager` 必须在包加载时就绪，不能只在 `InitializeDrivers` 里 new。**
   `var Manager = newDriverManager()` 是刻意写成这样的：`handler.Web()` 与内部 MQTT 比驱动启动更早可用（`Start` 内部还 sleep 1s 并查库），若 `Manager` 只在初始化时创建，这个窗口内保存点位/写值/MQTT 指令都会 `Manager.mu.RLock()` 空指针 panic，**而 panic 发生在子协程会直接带走整个进程**。
   `InitManager()` 也已改为幂等（仅 `Manager == nil` 时重建），旧实现每次调用都会替换整个管理器、把正在轮询的驱动连同各自的锁一起丢掉。改这里前先读 `manager.go:47-67` 的注释。

6. **`InitializeDrivers` 返回"第一个失败"的错误，但其余驱动照常启动。**
   单个驱动初始化失败只 `log.Printf("ERROR 驱动 id=%d 初始化失败")` 后 `continue`。
   `main.go` 收到非 nil 也只 `log.Printf("ERROR 驱动初始化失败: %v")`，**不会退出、不会重试**。于是可能出现"日志里有 ERROR 但服务看起来正常运行"，实际上少了一个驱动的数据。健康检查必须看 `/system/info` 里的驱动状态，不能只看进程活着。

7. **驱动类型只有 `"Modbus_Tcp"` 和 `"Siemens_S7"` 两个字符串，大小写敏感。**
   `CreateDriver` 的 `switch` 不在列表内直接 `errors.New("不支持的驱动类型: " + driveType)`。`Drive_Config.Type` 是用户可编辑字段，写错一个字母就是这条错误——而且是**每次启动都刷一遍**。

8. **`DriveWrite` 的只读校验是"整批拒绝"，不是"跳过不可写的"。**
   ```go
   if len(denied) > 0 {
       return fmt.Errorf("%w: 以下点位不允许写入（只读/禁止/不存在）: %s", ErrPointNotWritable, strings.Join(denied, ", "))
   }
   ```
   任一目标点位为 `RW_Cancel=1`(禁止)/`2`(只读)/不存在 → **整批失败，一个都不写**。这是有意的（避免"部分成功"造成误解），调用方须用 `errors.Is(err, ErrPointNotWritable)` 区分"参数错"与"设备写失败"：handler 据此映射 `417` vs `500`。
   校验放在这里是为了覆盖 HTTP 与内部 MQTT 两条通路，也修掉了 Modbus 组包静默过滤只读点位后**假返回成功**的老问题。各驱动内部的 `RW_Cancel` 校验保留为第二道防线。

9. **`RW_Cancel` 的四个值里只有 3/4 可写，别把 1 当成"只读"。**
   `1=禁止(N)`、`2=只读(R)`、`3=只写(W)`、`4=读写(R/W)`。常量 `rwCancelWriteOnly=3` / `rwCancelReadWrite=4` 是**包内私有**，外部只能靠数字或注释。

10. **`IsDriverBusy(id)` 对不存在的驱动返回 `false`（"不算忙碌"），不是错误。**
    它靠 `TryLock`+立刻 `Unlock` 实现，无副作用；但驱动不存在时 `getDriverLock` 返回 nil → `false`。`handler/config.go` 用它做重启前置判断时，"空闲"也可能意味着"根本没这个驱动"。

---

## 三、核心数据结构

```go
// driverEntry：函数绑定代替类型断言，调用时无需 switch/assert
type driverEntry struct {
    instance     any          // *modbus.Modbus_Tcp / *siemens.Siemens_S7
    New          func(mysql.Drive_Config_Query_type, []mysql.Point_Config_Query_type) error
    Connect      func(func([]model.Value_type) error) error
    Close        func() error
    UpdatePoints func([]mysql.Point_Config_Query_type) error // 不断连接、只重组包
    Write        func([]model.Value_type) error
    callback     func([]model.Value_type) error              // 存下来供 ResetConfig 重连复用
}

type DriverManager struct {
    drivers     map[uint]*driverEntry // key = Drive_Config.Id
    driverTypes map[uint]string
    driverLock  map[uint]*sync.Mutex  // 每驱动独立锁：不同 id 可并行，同 id 串行
    mu          sync.RWMutex          // 保护三个 map 本身
}

var Manager = newDriverManager()   // 包加载即就绪，见坑 5
```

`CreateDriver` 的绑定方式（新增驱动类型照抄此结构）：

```go
case "Siemens_S7":
    inst := &siemens.Siemens_S7{}
    entry = &driverEntry{instance: inst, New: inst.New, Connect: inst.Connect,
                         Close: inst.Close, UpdatePoints: inst.UpdatePoints, Write: inst.Write}
```

> 两层锁的分工：`Manager.mu` 只保护 map 读写（极短）；`driverLock[id]` 保护该驱动的**业务操作**。取锁一律用 `TryLock` 立即失败而非阻塞，保证接口不会因某个驱动卡在重连上而挂住 HTTP 线程。

---

## 四、对外 API

| 函数 | 谁调用 | 说明 |
|---|---|---|
| `Start(onData) error` | `cmd/server/main.go` | `Sleep(1s)` → `InitializeDrivers`；`onData` 传 `collector.Collection_Publisher` |
| `InitializeDrivers(onData) error` | `Start` | 全量装载；先 `Drive_Config__Query(0,0)`，再 `Point_Config__Query(nil,0,0)`，按 `Drive_Id` 分组（**避免每个驱动遍历全量点位**） |
| `CreateDriver(type, id) (any, error)` | `initSingleDriver`、`DriveResetConfig` | 建实例 + 绑函数 + 建该 id 的锁 |
| `GetDriver(id) (any, error)` | 需要裸实例的场合 | 返回 `any`，调用方自行断言（新代码不建议） |
| `DriveNew(id, drive, points)` | `initSingleDriver` | 解析配置、组包；**加锁 + 3s 防抖** |
| `DriveConnect(id, callback)` | `initSingleDriver` | 启动轮询协程；记 `e.callback = callback` |
| `DriveClose(id)` | 需要停驱动的场合 | 加锁 + 3s 防抖 |
| `DriveResetConfig(id, drive, points)` | `handler/config.go:179`（`/config/drive/restart`） | Close→6s→New→3s→Connect，全程锁住该 id；重连用存下来的 `e.callback` |
| `DriveUpdatePoints(id, points)` | `handler/config.go:250` | 不断连接只重组包；**不加锁**（坑 3）；`e.UpdatePoints == nil` 时返回"不支持动态更新点位" |
| `DriveWrite(id, values)` | `collector.WriteValues` | 统一只读校验入口（坑 8）；`len(values)==0` 直接返回 nil |
| `IsDriverBusy(id) bool` | `handler/config.go:141` | 非阻塞探忙 |
| `CloseAll(ctx)` | `main.go` 优雅退出 | 见第五节 |
| `ErrPointNotWritable` | `handler/point.go:158` 用 `errors.Is` 判 417 | 哨兵错误 |

---

## 五、优雅退出：`CloseAll(ctx)`

```go
1. Manager.mu.RLock() 一次性快照 ids + entries，立即 RUnlock   // 不在持锁期间做 IO
2. ids 升序排序                                                  // 日志顺序固定，便于跨次对比
3. 逐个驱动：
     ctx.Err() != nil → 直接 return（整体受退出预算约束，绝不卡死流程）
     lockWithTimeout(l, driveCloseWait=2s) 失败 → WARN "正在操作中，退出时不主动关闭（交由进程退出回收）" 并跳过
     e.Close() 成功 → INFO；失败 → WARN
     l.Unlock()
```

`lockWithTimeout` 是 `TryLock` + 100ms 轮询直到 deadline，**不使用 `Lock()`**：这样即使某个驱动的死循环占着锁，退出流程也最多为它花 2 秒。
`driveCloseWait = 2 * time.Second`，远小于总退出预算（`shutdownTotal 15s / shutdownStep 5s`），因此 N 个驱动最坏情况是 `N × 2s`——驱动很多时需要注意它与 `shutdownStep` 的关系（`CloseAll` 作为一个 step 调用，`step()` 内部 5s 超时只控制等待者，不会打断 `CloseAll` 本身）。

---

## 六、启动时序（与其他包的交错）

```
init 阶段（Go runtime）
  repo/mysql  init()  → 连 MySQL（失败 panic）+ 表结构自校验
  repo/redis  init()  → 连 Redis（失败 panic）
  repo/influxdb init()→ collector.Update_Subscriber(a)（注册但不初始化客户端）

main
  ├─ handler.Web()                         HTTP 立即可用（此时 Manager 为空 map，接口不 panic，见坑 5）
  ├─ collector.New()                       实时值缓存/报警/历史订阅就绪
  ├─ influxdb.New()（若 Enable）
  ├─ Sleep(200ms)                          等 Web 的致命监听错误暴露
  └─ driver.Start(collector.Collection_Publisher)
        ├─ Sleep(1s)
        └─ InitializeDrivers
             ├─ Drive_Config__Query(0,0) / Point_Config__Query(nil,0,0)
             ├─ InitManager()              幂等
             └─ 逐驱动 initSingleDriver：CreateDriver → DriveNew(+3s锁) → DriveConnect(+3s锁)
                                              └→ 驱动内部启动轮询协程
                                                 └→ 采集值经 onData(=collector.Collection_Publisher) 回流
```

数据回流链路：`驱动轮询协程 → Collection_Publisher → collector 总线 → {实时值缓存, 报警判断, InfluxDB 订阅, WS 推送订阅, internalmqtt}`。

---

## 七、排障速查

| 现象 | 定位 | 处理 |
|---|---|---|
| 启动后 1 分钟仍无数据 | 坑 1（每驱动 6 秒串行） | 看 `INFO 驱动 id=N ... 初始化完成` 的打印间隔；驱动多则考虑并行初始化 |
| 连续点两次"重启驱动"报 `正在操作中，请稍后再试` | `tryLockDrive` 非阻塞 + 3s 防抖 | 属预期；前端应禁用按钮 ~12 秒 |
| 重启后 9 秒查状态仍是旧的 | 坑 2（实际 12 秒） | 改提示文案或压缩 sleep |
| 改点位后偶发读到旧点表/panic | 坑 3（`DriveUpdatePoints` 无锁） | 与重启操作错开；或补 `tryLockDrive` |
| 日志有 `ERROR 驱动 id=X 初始化失败` 但服务正常 | 坑 6（失败不阻断） | 该驱动无数据；用 `/system/info` 核对驱动在线数 |
| `不支持的驱动类型: modbus_tcp` | 坑 7（大小写敏感） | 修 `Drive_Config.Type` |
| 写值返回 417 且消息含 `(RW=2)` | 坑 8/9，只读点位 | 改点位 `RW_Cancel` 为 3/4 |
| 写值整批失败但其中一个点位已删 | 坑 8 的"不存在"也整批拒绝 | 前端提交前过滤已删点位 |
| 退出卡在驱动关闭 | `CloseAll` 的 2s/驱动 | 减少并发退出驱动数或调小 `driveCloseWait` |
| 同一驱动出现两份数据 | 坑 4（旧 entry 未 Close 被覆盖） | 检查是否有重复 `CreateDriver` |

---

## 八、扩展指引：接入一个新驱动

1. **实现驱动包**：参考 modbus/siemens 的 `New / Connect / Close / UpdatePoints / Write` 五件套（`New` 接 `Drive_Config_Query_type + []Point_Config_Query_type`，`Connect` 接 `func([]model.Value_type) error`，`Write` 接 `[]model.Value_type`）。
2. **在 `CreateDriver` 的 `switch` 加一个 case**：创建实例并绑定五个函数引用。**这是唯一需要改本包的地方**——不要引入类型断言或全局 switch 分发。
3. **确认写值语义**：`DriveWrite` 已做 `RW_Cancel` 前置校验，驱动内只需实现"点位地址→设备写请求"的映射，可保留自身校验作第二道防线。
4. **确认点位更新语义**：`UpdatePoints` 要么"整包替换"（原子），要么自己加锁；不要做增量修改（会与轮询协程竞态）。
5. **确认 Close 幂等**：`CloseAll` 可能对一个从未成功 `Connect` 的实例调 `Close`。
6. **同步前端与文档**：`Drive_Config.Type` 的可选项、[handler README](../../handler/README.md) 的配置校验分支、根 README 的模块导航表。

---

## 九、相关文档

- [服务总览 README](../../../README.md)
- [internal/driver/modbus](../../driver/modbus/README.md) — Modbus TCP 驱动实现
- [internal/driver/siemens](../../driver/siemens/README.md) — 西门子 S7 驱动实现（CGO/snap7）
- [internal/service/collector](../collector/README.md) — `Collection_Publisher` 与 `WriteValues`
- [internal/repo/mysql](../../repo/mysql/README.md) — `Drive_Config__Query` / `Point_Config__Query__RW`
- [internal/handler](../../handler/README.md) — `/config/drive/restart`、`/point/write/value`
- [internal/service/internalmqtt](../internalmqtt/README.md) — MQTT 下行写值/重启指令
