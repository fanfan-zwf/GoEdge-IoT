# `internal/model` — 跨层数据模型

上级文档：[服务总览 README](../../README.md)

本包只放**被多层共同引用的最小数据结构**与 Tag 规则工具，刻意不含业务逻辑，
目的是打断 `driver ↔ collector ↔ handler` 之间的循环依赖。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `value.go` | 87 | `Value_type`（统一采集值）、`FullConfig_type`（驱动全配置）、Tag 与设备标识互转工具 |

---

## 二、`Value_type` — 全链路唯一的数据单元

```go
type Value_type struct {
    PointId uint      // 点位 id（MySQL Point_Config.Id）
    Tag     string    // 点位标识（//设备id//标识符）
    Value   any       // 点位值（真实 Go 类型：bool/int16/float32/…）
    Type    int       // 输出类型编码（见 pkg/conv 类型表）
    Msg     string    // 采集状态信息（正常为空；掉线时为错误文本）
    Time    time.Time // 采集时间
}
```

驱动 → collector → InfluxDB/WebSocket/MQTT **整条数据流都只传这个结构**（批量传 `[]Value_type`）。

约定与坑：

| 约定 | 说明 |
|------|------|
| `PointId == 0` | 表示"无效点位"，下游（报警/历史/写入）一律跳过；不要用它做过滤以外的假设 |
| `Value` 与 `Type` 必须一致 | `Type` 是 `Value` 的**声明输出类型**（经 Conversion 换算后的类型），写 InfluxDB 时按 `Value` 实际类型选字段名，两者不一致会导致 field type conflict |
| `Msg` 非空 = 该点采集失败 | 报警/历史判断会据此走错误分支；前端"状态列"直接展示它 |
| `Tag` 可能为空 | 内存与库内部仍以 `PointId` 为键；`Tag` 由上层按需补（InfluxDB 会**跳过 Tag 为空的数据**） |
| 结构体可复制 | 无指针字段，跨 goroutine 传递时按值拷贝切片即可，不必深拷贝 |

---

## 三、`FullConfig_type`

```go
type FullConfig_type struct {
    Drive  mysql.Drive_Config_Query_type
    Points []mysql.Point_Config_Query_type
}
```

"一个驱动 + 它名下所有点位"的成对视图，用于配置装载/同步场景。
注意它已经把 `mysql` 的 `_Query_` 结构带进模型层——这是本包唯一的仓库层依赖，
新增字段请沿用 `*_Query_type`（读视图），不要引入 `*_Add_type`/`*_Update_type`。

---

## 四、Tag 规则（本项目的核心标识约定）

```
//设备id//标识符      ← 完整形态（恰好 2 组 //）
/标识符               ← 简写形态，落库/使用前会被补全为完整形态
```

### 4.1 `TagWithDeviceId(tag string) (string, error)`

把简写 Tag 补全成带设备标识的完整 Tag：

| 输入 | 输出 |
|------|------|
| `/AI01` | `//<APP.Label>//AI01` |
| `//0001//AI01` | 原样返回（仅校验） |
| `AI01`（无 `/` 前缀） | 错误：`tag格式错误，必须以//或/开头` |
| `//a//b//c`（3 组 `//`） | 错误：`tag格式错误，必须恰好包含2组//` |
| `""` | 错误：`tag不能为空` |
| `APP.label` 未配置 | 错误：`设备标识符(Label)未配置` |

### 4.2 `TagSplitDeviceId(tag string) (string, bool)`

反向拆出设备 id：

- `//0001//AI01` → `("0001", true)`
- `/AI01`（不以 `//` 开头）→ `("", false)`
- `//` 组数不等于 2、或找不到第二个 `//` → `("", false)`

> 判定"是否本机点位"就靠它：`deviceId != config.Config.APP.Label` 即非本机数据，
> 内部 MQTT 与跨设备迁移的过滤逻辑都基于此。

### 4.3 与前端/仓库层的关系

| 层 | 位置 | 规则 |
|----|------|------|
| 前端镜像 | `Iot-Collector-Web/src/utils/configFormat.ts` | Tag/Conversion 正则的**前端副本**，改正则必须同步 |
| 仓库层校验 | `mysql.Regex_Tag_Format` / `Config__Validate_*` | 入库前把关，非 `//设备id//标识符` 或 `/标识符` 直接拒 |
| 对外接口 | `handler/point.go`（`idsByTags`/`mergePointIds`）、`service/collector/write.go`（`writeIdsByTags`） | 请求优先用 Tag，进来先反查成 `PointId` |
| 时序库 | `repo/influxdb` | 以 Tag 为序列维度，Tag 为空**不写入** |
| 树形导航 | `mysql/tag_tree` + `handler/tagtree.go` | 由 Tag 路径派生懒加载树 |

---

## 五、扩展指引

- 新增"跨层共用结构"放本包，并保持**无状态、无 IO**；任何要查库/发请求的逻辑都属于 `internal/service/*`。
- 不要在本包引用 `internal/service/*`、`internal/handler`——方向只能是 上层 → model。
- 若需要新的类型编码（如新增一种数值类型），改动点在 [pkg/conv](../../pkg/conv/README.md)（编码表 + 转换函数），本包的 `Type int` 只是承载它。

---

## 六、相关文档

- [pkg/conv](../../pkg/conv/README.md) — 类型编码表、字节序、换算
- [internal/service/collector](../service/collector/README.md) — `Value_type` 的消费方（值变化/报警/历史）
- [internal/repo/influxdb](../repo/influxdb/README.md) — `Value_type` → 时序数据点
- [internal/driver/modbus](../driver/modbus/README.md) / [siemens](../driver/siemens/README.md) — `Value_type` 的生产方
