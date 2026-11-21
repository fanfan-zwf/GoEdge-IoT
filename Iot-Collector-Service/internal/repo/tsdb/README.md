# `internal/repo/tsdb` — 嵌入式本地时序存储（Badger v4，**实验性，当前未接入**）

上级文档：[服务总览 README](../../../README.md)

基于 [dgraph-io/badger](https://github.com/dgraph-io/badger)（纯 Go LSM KV）做的**进程内**点位历史存储，思路是"把 InfluxDB 搬进边缘设备"：不依赖外部时序服务，直接落本地磁盘。

> ⚠️ **本包目前没有任何调用方**：全仓库搜索 `repo/tsdb` 只命中包自身，`badger-data/` 目录也不存在。
> 它既没被 `cmd/server` 初始化，也没被 `collector` 订阅，属于**预留/实验代码**。想启用请先读完第二节。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `tsdb.go` | 193 | `PointData`/`PointQuery`/`PointResult` 三个结构体、`InitDB`、`SavePoint`、`BatchQueryByTimeRange`、`BatchGetLastValue`、`buildKey`；末尾有一段被注释掉的 Badger 指标监控协程 |

包内只有这一个文件，无 `_test.go`，**也没有 `Close()`**。

---

## 二、⚠️ 最容易踩的坑（启用前必修）

1. **`InitDB()` 里 `log.Fatal(err)` —— 打不开 Badger 就直接退出进程。**
   ```go
   opts := badger.DefaultOptions("./badger-data")
   db, err = badger.Open(opts)
   if err != nil { log.Fatal(err) }
   ```
   目录不可写、或**已有另一个进程持有同一目录的锁**（Badger 独占目录锁，双实例/重启残留都会踩），服务立刻终止且没有降级路径。与 `repo/mysql`、`repo/redis` 的 `init()` panic 属同一类"启动强依赖"。

2. **数据目录是硬编码的相对路径 `./badger-data`。**
   不受任何配置项控制 → 落点取决于**进程工作目录**：`run.sh` 里 `cd` 到服务根目录才符合预期；用 systemd 时若 `WorkingDirectory` 没配就会写到别处（甚至只读文件系统）。启用前必须改成可配置（建议 `TSDB.Path`），并纳入磁盘占用监控。

3. **`BatchQueryByTimeRange` / `BatchGetLastValue` 在 `db == nil` 时会 panic。**
   只有 `SavePoint` 判了 `if db == nil { return fmt.Errorf("db not init") }`，两个查询函数都直接 `db.View(...)`，对 nil `*badger.DB` 调方法会崩。调用前必须自行确保 `InitDB()` 已成功。

4. **key 用 `RFC3339Nano` 字符串，字典序 ≠ 时间序。**
   ```go
   func buildKey(pd *PointData) []byte {
       return []byte(pd.Tag + "|" + pd.Time.Format(time.RFC3339Nano))
   }
   ```
   `time.RFC3339Nano` **会去掉小数秒末尾的零**：`.100000000` 变成 `.1`。于是
   `…T00:00:00.1+08:00` 与 `…T00:00:00.05+08:00` 做字符串比较时后者"更大"，而时间上恰恰相反。
   受影响的三处：`BatchQueryByTimeRange` 的 `it.Seek(startKey)` 起点、`keyStr > string(endKey)` 终止判断、`BatchGetLastValue` 靠倒序取"最新"（可能取到错误的最新值）。
   正确做法是定长可比较编码：零填充的 `20060102150405.000000000`、`math.MaxInt64 - unixNano` 的固定宽度十进制（想要天然倒序），或 `Tag + 8 字节大端时间` 的二进制 key。

5. **`Tag` 里含分隔符 `|` 会串台。**
   `keySeparator = "|"`，且 `buildKey` 不做任何转义。Badger 的 key 是任意字节，所以不会报错，只会静默地把
   `a|b` + 时间 与 `a` + `|b…` 混在同一前缀空间里，`strings.HasPrefix(keyStr, prefix)` 判定随之失真。
   点位 Tag 当前约定是 `//dev//name` 或 `/name`，不含 `|`，**但报警规则名/描述性 Tag 不受此约束**。加一层长度前缀或转义更稳妥。

6. **`BatchQueryByTimeRange` 提前 `return err` 会漏关迭代器。**
   迭代器在循环体里 `it := txn.NewIterator(...)`，只有走到 `it.Close()`（注释里称"优化点：避免 defer 在循环中累积"）才释放。
   但 `ValueCopy` 或 `json.Unmarshal` 出错时 `return err` **跳过了 `it.Close()`**。单次 `db.View` 结束后 Badger 会回收事务资源，不至于永久泄漏，但属于不干净的退出路径；`BatchGetLastValue` 里同类分支倒是显式补了 `it.Close()`——**两个函数写法不一致**。

7. **范围查询是"跨前缀线性扫描 + 手工 break"，没有用 `Seek` 到精确定位之外的优化。**
   `it.Seek(startKey)` 之后逐条 `Next()`，靠 `keyStr > string(endKey)` 和 `HasPrefix` 双条件退出。由于坑 4 的字符串序不可靠，**这两个边界判断都可能在错误的时刻生效或失效**（比如提前 break 丢数据，或扫过同 Tag 的后续数据）。区间越宽，代价越高。

8. **`SavePoint` 的 `TTL` 直接交给 Badger，`data.TTL == 0` 表示永不过期。**
   ```go
   entry := badger.NewEntry(key, val).WithTTL(data.TTL)
   ```
   调用方忘记填 `TTL` → 磁盘无限增长。边缘设备上是硬伤，务必在写入侧强制一个默认保留期。

9. **`BatchGetLastValue` 用 `if it.Rewind(); it.Valid()` 的紧凑写法。**
   Badger 在 `opts.Reverse = true` 时 `Rewind()` 定位到**逻辑末尾**（即最新 key），所以这个写法是对的；但很容易在读代码时误判为 bug 而"顺手改坏"。注意 `opts.Prefix = prefix` 已经把迭代范围限制在该 Tag 内。
   另有小坑：结果只在 `last.Value != nil` 时才写入 map，因此"最后一条记录的值恰好是 JSON `null`"会被当作"该点位无数据"。

10. **没有 `Close()`，进程退出不会做 Badger 的 sync/关闭。**
    其他 repo 包都向 `cmd/server` 的优雅退出步骤列表注册了 `Close`，本包没有。直接 `kill` 会依赖 Badger 自身的 WAL 恢复（能恢复，但下次启动慢、且可能丢弃未落盘写入）。

---

## 三、数据结构与 API

```go
type PointData struct {
    Tag   string          // 点位标识（也是 key 的前缀）
    Value json.RawMessage // 任意类型值，原样存 JSON
    Time  time.Time       // 业务时间戳，参与 key 构造
    TTL   time.Duration   // 过期时间，0 = 永不过期（见坑 8）
}

type PointQuery struct {
    Tag       string
    StartTime time.Time
    EndTime   time.Time    // 含端点（用 keyStr > endKey 才 break）
}

type PointResult struct {
    Time  string          // RFC3339Nano 字符串（不是 time.Time）
    Value json.RawMessage // 原始 JSON
}
```

| 函数 | 语义 | 备注 |
|---|---|---|
| `InitDB()` | 打开 `./badger-data` | 失败 `log.Fatal`；无返回值、无参数（坑 1、2） |
| `SavePoint(data PointData) error` | 单点写入，key = `Tag\|RFC3339Nano(Time)`，value = 整个 `PointData` 的 JSON | 每点一次 `db.Update` 事务，**没有批量写接口**（Badger 批量应使用 `NewWriteBatch`） |
| `BatchQueryByTimeRange(queries []PointQuery) (map[string][]PointResult, error)` | 多 Tag 多区间批量读，返回 `map[Tag][]PointResult` | 单个 `db.View` 事务内套多迭代器 |
| `BatchGetLastValue(pointIDs []string) (map[string]PointResult, error)` | 多 Tag 各取最新一条（倒序首条） | 无数据的 Tag **不出现在结果 map 中** |
| `buildKey(pd *PointData) []byte` | key 构造 | 见坑 4、5 |

value 里存的是**完整的 `PointData` JSON**（含 Tag/Time/TTL），因此 key 与 value 有冗余；好处是读回不需要解析 key，代价是写放大。

---

## 四、与 InfluxDB 方案的对比

| | `repo/influxdb` | `repo/tsdb` |
|---|---|---|
| 形态 | 外部服务（HTTP v2） | 进程内嵌入式 KV |
| 是否接入 | ✅ 可选启用（`Influxdb.enable`） | ❌ 未接入，无调用方 |
| 序列组织 | `measurement(APP.Label) + tag`，field 按类型分列 | `Tag|时间` 单 key，value 为 JSON |
| 写入语义 | 非阻塞缓冲区批量落盘 + 自动重传 | 每点一个 Badger 事务 |
| 覆盖/幂等 | 同 measurement+tag+时间戳为覆盖 | 同 key 覆盖 |
| 保留期 | 靠 bucket 的 retention 策略 | 靠 `PointData.TTL`（坑 8） |
| 查询能力 | Flux（窗口聚合、降采样可扩展） | 只能前缀扫描 |
| 主要风险 | `page=0` panic、`Value_Type_Confirm` 类型断言 | key 时间序不可靠、硬编码路径、无 Close |

需要"断网也能存历史、且不想到处部署 InfluxDB"时，本包是候选底座；但坑 4（key 编码）必须先解决，否则范围查询与"取最新"都不可信。

---

## 五、启用清单（若将来要接入）

1. `InitDB` 改为 `InitDB(path string, opts …) error`，**去掉 `log.Fatal`**，路径来自 `config.yaml`（新增 `TSDB.path` / `TSDB.enable`）。
2. 重写 `buildKey` 为定长可比较编码（坑 4），同时约束/转义 `Tag` 中的 `|`（坑 5）。
3. 补 `Close()` 并注册进 `cmd/server` 的优雅退出步骤（与其他 repo 一致，用 `sync.Once` 包一层）。
4. 三个查询/写入函数入口统一加 `if db == nil` 保护（坑 3）。
5. 循环内错误分支改为 `defer`-友好的写法或显式 `it.Close()`（坑 6）。
6. `SavePoint` 增加批量版本（`badger.WriteBatch`），并在写入侧强制默认 TTL（坑 8）。
7. 订阅 `collector.Update_Subscriber` 落数据，或由 `service/collector/history.go` 按 `History_Config` 周期写入——**与 InfluxDB 二选一，避免两份历史存储不一致**。

---

## 六、相关文档

- [服务总览 README](../../../README.md)
- [internal/repo/influxdb](../influxdb/README.md) — 现役时序方案
- [internal/service/collector](../../service/collector/README.md) — 历史采集（`History_Config`）的生产者
- [internal/model](../../model/README.md) — `Value_type` 结构体（若改造为统一入参）
