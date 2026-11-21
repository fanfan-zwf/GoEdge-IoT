# `internal/driver/modbus` — Modbus TCP 采集驱动

上级文档：[服务总览 README](../../../README.md) · [驱动管理 service/driver](../../service/driver/README.md)

本包实现 `Modbus_Tcp` 驱动：解析驱动/点位配置 → 按从机地址+功能码分组组包 → 单协程轮询 → 拆包解析 → 回调上报，
并支持 FC05/15/06/16 的批量写入。底层协议栈为 [`github.com/things-go/go-modbus`](https://github.com/things-go/go-modbus)。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `modbus.go` | 742 | 驱动主体：`New`/`UpdatePoints`/组包/连接/重连/轮询/解析/写入 |
| `new.go` | 127 | 配置解析入口：`Drive_Config_Switch`（驱动 JSON）、`Point_Config_Switch`（点位字符串） |
| `packet.go` | 109 | 地址合并算法 `PackAddressPackages`（连续/重叠地址必合并） |

---

## 二、⚠️ 最容易踩的坑

1. **可采集的类型只有 6 个**。组包时按 `Type_byte[point.Value_Type]` 查寄存器数量，查不到就打 `ERROR 无效类型` 并**静默跳过该点位**：

   | 类型编码 | 名称 | 占用寄存器 |
   |------|------|------|
   | 1 | `BOOL` | 1 |
   | 4 | `INT16` | 1 |
   | 5 | `UINT16` | 1 |
   | 6 | `INT32` | 2 |
   | 7 | `UINT32` | 2 |
   | 12 | `FLOAT32` | 2 |

   `INT64/UINT64/FLOAT64/STRING` **都不支持**，配了也采不上来。

2. **决定能否采集/写入的是 `Point_Config.RW_Cancel`，不是功能码**：
   - 轮询组包只收 `RW_Cancel ∈ {2 只读, 4 读写}`；
   - 写入组包只收 `RW_Cancel ∈ {3 只写, 4 读写}`；
   - 上层 [service/driver](../../service/driver/README.md) `DriveWrite` 还会再做一次只读拦截。

3. **写入只支持功能码 1 和 3**。`write_packet` 的分支只覆盖 `FunctionCodeReadCoils(1)` 与 `FunctionCodeReadHoldingRegisters(3)`；
   给 FC2（离散输入）/FC4（输入寄存器）的点位配 `RW_Cancel=3/4`，会在运行期收到 `不支持的写入类型`。

4. **`Write` 的返回值不可信**。它逐包执行、遇错只 `log.Print` 后 `continue`，最终 `return` 的是**最后一个包**的 err；
   最后一个包成功时整体返回 `nil`，前面的失败被吞掉。需要逐点成败回执时请看 [internalmqtt](../../service/internalmqtt/README.md) 的 `/write` 处理或自行改造为错误聚合。

5. **`UpdatePoints` 不回填 `Tag`**（`New` 会回填）。动态更新点位后 `Point_Config_type.Tag` 为空串，
   导致 `analysis`/`Error_External_Mappings` 上报的 `model.Value_type.Tag` 为空 → InfluxDB 序列 tag 缺失、报警日志里点位标识为空。
   触发时机见 [service/driver](../../service/driver/README.md) 的点位增量更新逻辑；改动本函数时请顺手补 `Tag: v.Tag`。

6. **一个驱动 = 一条 TCP 连接 = 一个轮询协程，串行取包**。`Delay_between_polls` 是**每包之间**的 sleep，不是每轮之间；
   包多时一轮耗时 = Σ(包往返 + sleep)，调这个值前先数一下 `轮询包数`（`New`/`UpdatePoints` 的 INFO 日志里有）。

7. **写入值有过期校验**：`model.Value_type.Time` 非零且距今 > 5s 直接拒绝该包（`写入值时间间隔过长`），
   这是防止把队列里积压的旧写入下发的保护，别在构造写入值时填一个很早的时间戳。

8. 以下成员是**历史遗留、当前未被使用**，读码时不要据此推断行为：
   `Modbus_Tcp.first_connect`、`.read_points`、`.Esc_collection`、`Collection_Allback()`（空实现）、`Connect_interface`、`Packet_df` 以外的 `packet.go:IsTimePassed`。
   同理 `new.go` 注释里的 `Write_Coils_Function` / `Write_Register_Function` 两个字段**已不存在于 `Config_type`**，
   写入功能码是硬编码的 FC15/FC16，不可配置。

---

## 三、配置格式

### 3.1 驱动配置（`Drive_Config.Config`，JSON 对象）

```json
{
  "Address": "192.168.1.10:502",
  "Retry_timeout": "3s",
  "Connect_timeout": "10s",
  "Response_timeout": "10s",
  "Delay_between_polls": "0s",
  "Packet_max": 120
}
```

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `Address` | string | **是** | — | `ip:port`；缺失直接报 `Modbus配置缺少必填字段: Address` |
| `Retry_timeout` | `conv.Duration` | 否 | `3s` | 重连间隔（固定间隔，**非**指数退避；Siemens 驱动才是退避） |
| `Connect_timeout` | `conv.Duration` | 否 | `10s` | 传给 `modbus.WithTCPTimeout` |
| `Response_timeout` | `conv.Duration` | 否 | `10s` | 目前只做默认值补全，未单独作用于读超时 |
| `Delay_between_polls` | `conv.Duration` | 否 | `0` | 每包之间的 sleep |
| `Packet_max` | int | 否 | `120` | 单包最大**寄存器数**；`<=0` 会被重置为 120 |

> 所有时长字段都是 `conv.Duration`，JSON 里写 `"3s"`/`"1m30s"` 这种字符串，实现见 [pkg/conv/duration.go](../../../pkg/conv/README.md)。
> 校验入口在 [mysql.Config__Validate_JSON](../../repo/mysql/README.md)：驱动配置必须是 JSON **对象**，数组/标量会被拒。

### 3.2 点位配置（`Point_Config.Config`，冒号分隔字符串）

```
从机地址:功能码:寄存器地址[.子地址]:字节序:数据类型
```

正则：`Regex_Modbus_Point = ^(\d+):([1-4]):(\d+)(?:\.(\d+))?:(\w+):(\w+)$`

| 段 | 范围 | 例 |
|----|------|----|
| 从机地址 | 0–255 | `1` |
| 功能码 | **1–4**（正则限制，写也用它选分支） | `3` |
| 寄存器地址 | 0–65535 | `100` |
| `.子地址` | 可选，0–15；仅 BOOL+寄存器场景用作位偏移 | `.5` |
| 字节序 | `conv.Byte_Value` 表的键，大小写不敏感 | `ABCD` |
| 数据类型 | `conv.ValueTypeStringToInt` 的键，大小写不敏感 | `float32` |

示例：`1:3:100:ABCD:float32`、`1:1:100.5:AB:bool`

- **解析失败不会中断驱动**：`New` 对每个点位单独 `log.Printf("WARN 点位【tag】配置解析失败, 跳过")`，坏点被丢弃，好点照常采集。
- `Point_Config.Value_Type = 0` 时，用点位配置里解析出的数据类型作为默认输出类型（打 WARN）。
  输出类型 ≠ 采集类型时以**输出类型**决定寄存器数量与解析分支，二者在 `analysis` 中是同一个值（见 5.2）。

---

## 四、类型与函数

### 4.1 类型

| 类型 | 关键字段 | 说明 |
|------|----------|------|
| `Config_type` | 见 3.1 | 驱动配置 |
| `Points_type` | `SlaveID, Function, Address, Child_Address, Byte_Order` | 点位协议参数 |
| `Drive_Config_type` | `Id, Type, Config` | 运行时驱动配置（由 mysql 结构体转换而来） |
| `Point_Config_type` | `Id, Tag, Conversion, RW_Cancel, Value_Type, Config` | 运行时点位 |
| `Packet_type` | `SlaveID, Function, Start_Address, Number_Address, Ids[]uint` | 一个轮询/写入包 |
| `Modbus_Tcp` | `Drive, Points, conn, conn_err, packets, closed, Tag_Pointsindex_Map, Read_External_Mappings, Write_value_mu` | 驱动实例 |

### 4.2 方法（对应 [service/driver](../../service/driver/README.md) 的 `driverEntry` 函数槽）

| 方法 | 签名 | 说明 |
|------|------|------|
| `New` | `(Drive mysql.Drive_Config_Query_type, Points []mysql.Point_Config_Query_type) error` | 解析配置 + 建 `Tag_Pointsindex_Map` + 组轮询包；**不连接** |
| `UpdatePoints` | `(Points []mysql.Point_Config_Query_type) error` | 重解析点位 + 重包，不断连 |
| `Connect` | `(Read_External_Mappings func([]model.Value_type) error) error` | 注入数据回调并起轮询协程 |
| `Close` | `() error` | 置 `closed` + 关 TCP + 上报 `"驱动连接已关闭"` |
| `Write` | `(values []model.Value_type) error` | 写入（见坑 4） |
| `packet` | 私有，`(Points, RW_Cancel map[int]bool) ([]Packet_type, error)` | 读写共用，用 `RW_Cancel` 白名单过滤 |
| `analysis` | 私有，`(Packet_type, []byte) ([]model.Value_type, error)` | 字节流 → 值 |
| `polling` / `connect` / `reconnect` / `write_packet` / `id_points_index` | 私有 | 见下 |
| `Error_External_Mappings(msg)` / `Error_External_Mappings_list(ids, msg)` | 全点/指定点上报**只有 Msg、无 Value** 的记录 | 状态与错误通过同一条数据通道上抛 |

### 4.3 包级导出

`FunctionCodeXxx` 常量、`Type_byte`、`Config_type`、`Regex_Modbus_Point`、`Drive_Config_Switch`、`Point_Config_Switch`、`PackAddressPackages`、`IsTimePassed`。

---

## 五、核心流程

### 5.1 组包 `PackAddressPackages(addrList, maxPackageLen)`

1. 丢掉 `DataLen <= 0` 的点（打 WARN）；`EndAddr = StartAddr + DataLen - 1`；
2. 按 `StartAddr` 升序排序；
3. 顺序扫描，只与**上一个包**比较：`point.StartAddr <= lastEnd + 1` 即视为连续/重叠并尝试合并；
   合并后长度超过 `maxPackageLen` 就不合并、另起一包（`maxPackageLen == 0` 表示不限长）；
4. 结果按 (SlaveID, Function) 桶分别生成 `Packet_type`。

推论：**同一个从机+功能码下的点位地址越密集，包越少**；故意把两点位隔得很开只会增加请求数，
中间未配置的寄存器也会被一并读回（浪费带宽但无害）。

### 5.2 解析 `analysis`

以"包内相对位置"定位，`byte_index = cfg.Config.Address - packet.Start_Address`：

| 情况 | 取值方式 |
|------|----------|
| BOOL + FC1/FC2 | 位对象：第 `byte_index/8` 字节的第 `byte_index%8` 位 |
| BOOL + FC3/FC4 | 寄存器对象：第 `byte_index/16` 字节（=该寄存器高字节）的第 `Child_Address` 位 |
| INT16/UINT16 + FC3/FC4 | 从 `byte_index*2` 起取 **2** 字节，按 `Byte_Order` 转换 |
| INT32/UINT32/FLOAT32 + FC3/FC4 | 从 `byte_index*2` 起取 **4** 字节，按 `Byte_Order` 转换 |

之后统一走 `conv.ConvertWithCalc(v, Value_Type, Value_Type, Conversion)` 做表达式换算；
失败不丢点，而是把错误写进 `read.Msg`（`Value` 留空）——错误信息随数据一起上行，前端与历史库都能看到。

> 注意：FC1/FC2 的 BOOL 用**位地址**（子地址被忽略，寄存器地址直接当位地址用），
> FC3/FC4 的 BOOL 才用 `Child_Address` 位偏移。这是同一份配置在两类功能码下语义不同的地方。

### 5.3 轮询与重连

```
Connect(callback)
 ├─ connect() 成功 → go polling()
 └─ connect() 失败 → 返回 nil（不报错！）+ go { reconnect() → polling() }
polling():
  for { closed? 退出
        取 packets[i++]（越界归零）
        sleep(Delay_between_polls)
        按功能码读 → 出错: reconnect() 成功则 continue，失败则全点上报错误并退出轮询
        analysis → Read_External_Mappings(read_list) }
```

- **`Connect` 首次失败不向上层返回错误**：连接失败是运行期状态，由 `closed` 标志 + 后台重连循环接管，
  避免一台离线设备拖垮整条 `DriveResetConfig`/启动链路。设备状态通过 `Msg` 通道体现（见 [service/collector](../../service/collector/README.md)）。
- `reconnect()` 是**固定间隔死循环**，只有连上或 `closed` 才退出；因此 `Close()` 必须被调用，否则协程泄漏。
- 读到的 `err` 会先 WARN 再重连，**不会**给单点写错误；重连彻底失败才用 `Error_External_Mappings` 广播全点。

### 5.4 写入 `Write` → `write_packet`

1. `Write` 把 `[]model.Value_type` 建成 `map[id]value`，逐点查配置（任一点不存在**立即整体返回错误**）；
2. 用 `RW_Cancel ∈ {3,4}` 重新组包（写入包的合并规则与轮询包完全相同）；
3. 每包：校验时间新鲜度（≤5s）→ 校验 `v.Type == cfg.Value_Type` → `conv.ConvertReverse` 反算表达式 →
   按类型拼 `byte_list`：
   - **BOOL + FC3 会先回读该寄存器**（`ReadHoldingRegistersBytes`，同一地址本包内只回读一次），
     再 `SetBit(Child_Address)`，从而**保留同寄存器其他位**不被清零；
   - 数值类型用 `conv.XxxToBytes(..., Byte_Order)` 按配置的字节序落地；
4. 下发：FC1 → `WriteMultipleCoils`（FC05/15 语义合并为多写），FC3 → `WriteMultipleRegistersBytes`（长度 `Number_Address*2`）。

> `Write_value_mu sync.Mutex` 字段已声明但**未使用**，写入目前依赖 go-modbus 客户端自身的串行化；
> 若要在同驱动上并发写，请先补上锁。

---

## 六、扩展指引

- **新增类型**：同时改 `Type_byte`（寄存器数）、`analysis`（解析分支）、`write_packet`（编码分支），
  并确认 [pkg/conv](../../../pkg/conv/README.md) 里已有对应的 `BytesToXxx/XxxToBytes` 与类型编码。
- **新增功能码**：`Regex_Modbus_Point` 的 `([1-4])` 要放宽，`polling`/`write_packet` 的 switch 要加分支。
- **接入新驱动**：实现 `New / Connect(func([]model.Value_type) error) / Close / UpdatePoints / Write` 五个方法即可，
  然后在 [service/driver/manager.go](../../service/driver/README.md) 的 `CreateDriver` 注册类型名——**不需要**改本包。
- 建议顺带修正的已知缺陷（改动前先跑一遍真实设备回归）：`UpdatePoints` 补 `Tag`、`Write` 聚合错误、`Write_value_mu` 生效。

---

## 七、相关文档

- [internal/driver/siemens](../siemens/README.md) — 另一个驱动实现（S7，含指数退避重连，可对比）
- [internal/service/driver](../../service/driver/README.md) — 驱动生命周期、只读校验、忙碌锁
- [internal/service/collector](../../service/collector/README.md) — 数据回调后的换算/报警/历史/发布
- [pkg/conv](../../../pkg/conv/README.md) — 字节序、类型转换、表达式正反算
- [internal/model](../../model/README.md) — `Value_type` 上报契约
