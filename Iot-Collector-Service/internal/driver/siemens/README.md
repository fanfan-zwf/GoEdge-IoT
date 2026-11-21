# `internal/driver/siemens` — 西门子 S7 采集驱动（snap7 / CGO）

上级文档：[服务总览 README](../../../README.md) · [驱动管理 service/driver](../../service/driver/README.md)

本包通过 **snap7 C 动态库**（CGO 绑定）连接 S7-300/400/1200/1500/Smart200 等 PLC，
用 `ReadMultiVars` / `WriteMultiVars` 批量读写 I/Q/M/DB/C/T 存储区。
相比 [modbus 驱动](../modbus/README.md)，本包额外做了：**PDU 协商、连接代数控制、弱网自适应重连、并发快照**。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `siemens.go` | 1229 | 驱动主体：`New`/`UpdatePoints`/组包/连接/退避重连/轮询/解析/写入 |
| `new.go` | 218 | 配置解析：`Drive_Config_Switch`（驱动 JSON）、`Point_Config_Switch`（S7 地址串或 JSON）、类型名映射 |
| `packet.go` | 285 | 常量（PDU/MaxVars/开销）、`ValueTypeToByteLen`、`PackS7AddressPackages` 合并算法 |
| `snap7_cgo.go` | 1779 | snap7 全量 CGO 绑定：Client（同步+异步）/Server/Partner、错误码、`S7DataItem` |

依赖的本地库：`third_party/lib/`（`libsnap7.so`、`snap7.h`、`snap7_api_cn.md`）。

---

## 二、⚠️ 编译与运行前置（最容易失败的一步）

`snap7_cgo.go` 顶部的 cgo 指令：

```go
#cgo CFLAGS: -std=c11 -I${SRCDIR}/../../../third_party/lib
#cgo LDFLAGS: -L${SRCDIR}/../../../third_party/lib -lsnap7 -Wl,-rpath,$ORIGIN/third_party/lib
```

- 编译需要 **gcc + libc6-dev**（纯 `go build` 无 CGO 环境会失败；`CGO_ENABLED=0` 时本包不可用）。
- `-Wl,-rpath,$ORIGIN/third_party/lib` 中的 `$ORIGIN` 是**可执行文件所在目录**：
  二进制输出到 `bin/` 后 `$ORIGIN` = `bin/`，那里没有 `third_party/`，**运行期会报 `libsnap7.so: cannot open shared object file`**。
  [run.sh](../../../run.sh) 已通过 `export LD_LIBRARY_PATH="$SCRIPT_DIR/third_party/lib"` 兜住，
  手工直接跑 `./bin/collector-service` 或用 systemd/容器部署时，**必须自己设置 `LD_LIBRARY_PATH`**（或把库装到 `/usr/lib`）。

---

## 三、⚠️ 行为要点与坑

1. **没有任何可读点位 → `New` 直接失败**。`Read_Packet()` 在可读点（`RW_Cancel ∈ {2,4}`）为空时返回 `无有效的可读点位`，
   进而 `New` 返回 `读取组包失败: …`，驱动创建失败。纯写入型驱动请至少配一个只读点位。
2. **`Type`（采集类型）与 `Value_Type`（输出类型）是两件事**：
   - `Config.Type` 决定**缓冲区长度与字节流解析方式**（PLC 侧真实宽度）；
   - `Value_Type` 决定**对外呈现类型**；两者靠 `Conversion` 表达式连接。
   - 缓冲区按**采集类型**分配，若误按输出类型分配，`int16 → int64` 这类配置会踩内存/越界。
   - 采集类型缺省（0）时回落为 `Value_Type` 并打 WARN——旧配置能跑，但请补齐 TIA 类型名。
3. **`string` 只按 4 字节缓冲处理**：`ValueTypeToByteLen(TYPE_STRING) == 4`。
   超过 4 字节的 S7 `String` 采不全；写入 `string` 直接 `copy` 进缓冲、**不写 S7 的 maxlen/len 前两字节**，
   与 TIA 的 `String` 布局不兼容。要读写字符串请用 `Byte/USInt` 数组逐字节建模，或改造本函数。
4. **`Char/WChar/WString/Time/LTime/Date/TOD/DT` 未实现**，`parseS7Type` 会返回"未知的 TIA 类型名"并跳过该点位。
5. **Timer/Counter 特殊处理**：`Area = 0x1C(C) / 0x1D(T)` 时 `Start` 是**编号**不是字节偏移，
   `Amount=1`、`DBNumber` 强制 0、`WordLen=S7WLTimer/S7WLCounter`、固定返回 2 字节，且**不参与地址合并**（每点独占一包）。
   这两个区只能通过 **JSON 格式**点位配置指定（地址正则不匹配 C/T）。
6. **写入准备阶段的错误只打日志、不回传**：`Write` 第 5 步 `writeUpdateBuffer` 失败时只 `log.Printf("ERROR 写入准备失败…")`，
   第 6 步仍会把该 item 的缓冲区（可能是全零或预读原值）写进 PLC。
   写入值类型不匹配/偏移越界**可能导致把 0 写下去**，接线前务必用只读点验证配置，或改造为收集错误后跳过该 item。
7. **`Write` 会先做全量可写校验**（任一目标点 `RW_Cancel ∉ {3,4}` → 整体返回错误），
   上层 [service/driver](../../service/driver/README.md) `DriveWrite` 还有一层同样的拦截 + `ErrPointNotWritable`。
8. **bool 写入是"读-改-写"**：含 bool 的包先 `ReadMultiVars` 预读，再 `SetBit(item.Data, byteOffset*8 + Child_Address, v)`，
   以保留同字节其他位；`Child_Address > 7` 报错。预读失败会**直接返回错误终止本次写入**（不会盲写）。
9. **单驱动串行、多批**：一轮 = 若干 `ReadMultiVars`（每批 ≤20 个 item 且报文 ≤ 协商 PDU），全部基于同一份快照；
   某批原地重试仍失败 → 立刻中断本轮，剩余批不读。
10. **`UpdatePoints` 不重置 `negiatedPDU`**，用的是上次连接协商到的值重组包，这是期望行为；
    但若 PLC 侧更换型号导致 PDU 变小，需重连（`DriveResetConfig`）才会重新协商。

---

## 四、配置格式

### 4.1 驱动配置（`Drive_Config.Config`，JSON 对象）

```json
{
  "Address": "192.168.1.1",
  "Rack": 0,
  "Slot": 1,
  "ConnectionType": 2,
  "Retry_timeout": "3s",
  "Connect_timeout": "10s",
  "Response_timeout": "10s",
  "Delay_between_polls": "0s",
  "Max_reconnect_interval": "30s",
  "Local_TSAP": 0,
  "Remote_TSAP": 0
}
```

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `Address` | string | **是** | — | PLC IP（不带端口，S7 端口 102 由 snap7 处理） |
| `Rack` | int | 否 | 0 | 机架号，合法范围 **0–7** |
| `Slot` | int | 否 | **2** | 槽位号，合法范围 **0–31**；S7-1200/1500 通常是 **1**（`Slot<=0` 会被改成 2，务必显式写 1） |
| `ConnectionType` | int | 否 | 2 | 1=PG 编程设备、2=OP 操作面板、3=Basic |
| `Retry_timeout` | `conv.Duration` | 否 | `3s` | 指数退避的**初始**间隔 |
| `Max_reconnect_interval` | `conv.Duration` | 否 | `30s` | 指数退避封顶 |
| `Connect_timeout` / `Response_timeout` | `conv.Duration` | 否 | `10s` | 分别映射 `ParamSendTimeout` / `ParamRecvTimeout`（毫秒） |
| `Delay_between_polls` | `conv.Duration` | 否 | `0` | 每**轮**结束后的 sleep |
| `Local_TSAP` / `Remote_TSAP` | uint16（JSON 里写十进制数） | 否 | `0`=自动推导 | 手动覆盖，见 4.3 |

PDU 不可配置：连接前固定请求 `s7RequestPDU = 960`，连接后取 PLC 协商值（`GetPduLength`），协商值 ≤0 时回落 960。
相关常量：`s7DefaultPDUSize=480`、`s7MaxItemsPerRead=20`、`s7ItemOverhead=17`、`s7PduBaseOverhead=18`。

### 4.2 点位配置（`Point_Config.Config`）

支持两种写法，**以 `{` 开头即按 JSON 解析**。

**A. 地址字符串（推荐，人写）**：`<S7地址> [TIA类型名]`，类型名可省略。

| 例 | 解析结果 |
|----|----------|
| `DB1.DBX0.5 Bool` | Area=0x84, DB=1, Start=0, Child_Address=5, Type=1 |
| `DB1.DBD0 Real` | Area=0x84, DB=1, Start=0, Type=12 |
| `DB1.DBB8 USInt` | Area=0x84, DB=1, Start=8, Type=3 |
| `DB1.DBW10` | Area=0x84, DB=1, Start=10, Type=5（后缀 W → UInt） |
| `M10.0` | Area=0x83, Start=10, Child_Address=0, Type=1（后缀 X → Bool） |
| `I0` | Area=0x81, Start=0, Type=0 → 回落 `Value_Type` |

正则：`Regex_S7_Point_Config = ^(?:(DB)(\d+)\.(DB[XBWDL])|([IQM])([XBWDL]?))(\d+)(?:\.(\d+))?$`（地址段会先 `ToUpper`）

**后缀 → 默认采集类型**（仅在未显式写类型名时生效）：`X→Bool(1)`、`B→USInt(3)`、`W→UInt(5)`、`D→UDInt(7)`、`L→ULInt(9)`。

**TIA 类型名 → 采集类型代码**（大小写不敏感，`s7TypeMap`）：

| 代码 | Go 类型 | TIA 名（别名） | 宽度 | 字节序 |
|----|---------|----------------|------|--------|
| 1 | bool | Bool | 1 B（按位） | `Child_Address` 0–7 |
| 2 | int8 | SInt | 1 | — |
| 3 | uint8 | USInt / **Byte** | 1 | — |
| 4 | int16 | Int | 2 | 大端 `BA` |
| 5 | uint16 | UInt / **Word** | 2 | 大端 `BA` |
| 6 | int32 | DInt | 4 | 大端 `ABCD` |
| 7 | uint32 | UDInt / **DWord** | 4 | 大端 `ABCD` |
| 8 | int64 | LInt | 8 | 大端 `ABCDEFGH` |
| 9 | uint64 | ULInt / **LWord** | 8 | 大端 `ABCDEFGH` |
| 10 | int | —（Go 原生别名） | 8 | 按 LInt 采集 |
| 11 | uint | —（Go 原生别名） | 8 | 按 ULInt 采集 |
| 12 | float32 | Real | 4 | IEEE-754 `ABCD` |
| 13 | float64 | LReal | 8 | IEEE-754 `ABCDEFGH` |
| 14 | float | —（Go 原生别名） | 4 | 等价 Real |
| 15 | string | String | **4（限制，见坑 3）** | 截断到首个 `0` 字节 |

**B. JSON（程序生成 / C、T 区必须用它）**：`{"Area":132,"DBNumber":1,"Start":0,"Child_Address":0,"Type":12}`
（`Area` 用十进制：132 = 0x84 = DB）

> 解析失败不中断驱动：坏点 `WARN 点位【tag】配置解析失败, 跳过`，其余点照常组包。

### 4.3 TSAP：自动推导与手动覆盖

```
local  = ConnectionType << 8
remote = ConnectionType << 8 + Rack*0x20 + Slot
```

| 设备 | connType/Rack/Slot | remote |
|------|--------------------|--------|
| S7-300 | 2 / 0 / 2 | `0x0202` |
| S7-1200/1500 | 2 / 0 / 1 | `0x0201` |
| S7-400 | 2 / 0 / 3 | `0x0203` |

**什么时候必须手填**：CP 343-1 Advanced（6GK7 343-1EX30）等"（S7 连接需在 NETPRO 里预配置、使用自定义 TSAP）"的设备。
把 STEP7 里那条连接的 TSAP 填进来（STEP7 显示 `0x0102` → JSON 里填十进制 `258`），两者都填 0 才是自动推导。
手动值与自动值不同时，连接日志会打 `INFO … 使用手动 TSAP local=0x… remote=0x…（自动推导为 …）`，排障先看这行。

---

## 五、类型与函数

### 5.1 `Siemens_S7` 关键字段

| 字段 | 保护 | 说明 |
|------|------|------|
| `Drive_Id`, `Config` | — | 驱动标识与配置 |
| `Points`、`Read_Packet_S7DataItem`、`Read_Packet_S7DataItem_Point`、`negotiatedPDU` | `mu sync.RWMutex` | **派生数据与 Points 一起整体发布新引用**；读方取快照后基于本地引用操作，已发布对象不再原地改 |
| `client`、`closed`、`reconnectGen` | `lifeMu sync.Mutex` | 连接生命周期状态 |
| `retryCount`、`readFailStreak` | 仅重连/轮询协程 | 退避与失败计数 |
| `Read_External_Mappings func([]model.Value_type) error` | `Connect` 注入 | 数据上报回调 |

### 5.2 方法

| 方法 | 说明 |
|------|------|
| `New(Drive, Points) error` | 解析配置 + 首次组包（按默认 PDU）；不连接 |
| `UpdatePoints(Points) error` | 重解析 + 重组包，不断连，不重置协商 PDU |
| `Connect(cb) error` | 递增代数作废旧循环 → `connect(gen)` → 起 `polling(gen)`；**首次连接失败返回 nil** |
| `Close() error` | `closed=true` + `reconnectGen++` + `Disconnect/Destroy` + 上报"驱动连接已关闭" |
| `Write(values) error` | 见 §6.4 |
| `Read_Packet() error` | 导出风格的私有组包函数（首字母大写但在包内） |
| `Error_External_Mappings(msg)` / `Error_External_Mappings_byPacketIndex(i,msg)` | 全点 / 单包错误上报（只有 `Msg`，无 `Value`） |
| `loopAlive(gen)` / `snapshotClient()` / `connect(gen)` / `reconnect(gen)` / `polling(gen)` / `analysis(...)` / `computeReadBatches` / `batchS7Items` | 私有 |

包级导出：`Config_type`、`S7Points_type`、`Point_Config_type`、`Siemens_S7`、`Regex_S7_Point_Config`、
`Drive_Config_Switch`、`Point_Config_Switch`、`ValueTypeToByteLen`、`PackS7AddressPackages`、`PackAddressPackages`、`IsTimePassed`、
`snap7_cgo.go` 的全量常量与 `Snap7Client`/`S7DataItem`/`S7Error` 等绑定。

> `packet.go:IsTimePassed` 与 `PackAddressPackages`（`uint16` 版）当前**未被调用**，是从 modbus 版本复制来的历史遗留；
> S7 实际使用的是 `PackS7AddressPackages`。

---

## 六、核心流程

### 6.1 组包 `PackS7AddressPackages(points, maxPacketLen)`

1. `maxPacketLen <= 0` → 用 `s7DefaultPDUSize(480)`；
2. **数据区上限** `maxData = PDU - s7PduBaseOverhead(18) - s7ItemOverhead(17)`（PDU 是**整包**大小，含 S7 头与每 item 协议开销）；
3. 按 `Area + DBNumber` 分组，组内按 `Start` 排序；
4. `C/T` 区每点独占一包（`ByteLen=2`）；普通区**连续或重叠即合并**（`point.Start <= last.End + 1`），合并后超 `maxData` 则另起一包。

推论：同一 DB 内地址越密集包越少；跨 DB **永不合并**。

### 6.2 连接

```
New → Read_Packet(默认PDU) → Connect(cb)
   lifeMu: closed=false, reconnectGen++=gen
   connect(gen):
     NewSnap7Client → buildAutoTSAP(可被 Local_TSAP/Remote_TSAP 覆盖) → SetConnectionParams
     SetParam(ParamPDURequest, 960) → SetParam(ParamSendTimeout/RecvTimeout)
     Connect() 失败 → Disconnect() + Destroy()（主动发 FIN，避免半开连接堆积）
     lifeMu 下发布：若 closed 或 gen 已过期 → 丢弃这条连接并返回"连接结果已作废"
     GetPduLength → negotiatedPDU → 用真实 PDU 重新 Read_Packet()
   成功 → go polling(gen)；失败 → go { reconnect(gen) → polling(gen) } 且 Connect 返回 nil
```

**代数（gen）机制是本包并发安全的核心**：保存配置会走 `Close → New → Connect`，
若无代数控制，旧的重连循环会与新连接并发刷同一台 PLC（连接风暴）；旧循环在每个检查点调 `loopAlive(gen)`，
一旦 `gen != reconnectGen` 或 `closed` 就自我退出。修改本包时**任何新起的循环都必须带 gen 并在每轮检查**。

### 6.3 轮询与弱网自适应

```
polling(gen):
  for { loopAlive? 否 → 退出
        RLock 取快照 items/mapping/points/pdu
        batches = computeReadBatches(items, pdu)      // PDU 净空间 + MaxVars(20) 双重限制
        逐批 ReadMultiVars：失败先原地重试 s7ReadRetries(1) 次，间隔 s7ReadRetryPause(500ms)
        任一批最终失败 → readFailStreak++
             streak < s7ReconnectAfterFailures(2) → 全点上报错误 + sleep + 下一轮（保留连接）
             streak 达阈值 → reconnect(gen)（失败则退出轮询）
        全部成功 → streak=0
        逐 item：item.Result != 0 → 按包上报错误；否则 analysis(...) → Read_External_Mappings
        sleep(Delay_between_polls) }
```

重连策略：每轮先做 `s7ConnectBurst(2)` 次**快速建连尝试**（间隔 `s7ConnectBurstPause(1s)`），
两次都失败才 `sleep(interval)` 并令 `interval *= 2`，封顶 `Max_reconnect_interval`；连上即复位 `retryCount`。
设计动机见代码注释：*重连要在烂链路上重新赌一次 ISO/COTP 握手，代价远高于一次读超时*。

### 6.4 解析 `analysis`

`byteOffset = point.Config.Start - item.Start`，按**采集类型**决定读取宽度与解析方式（S7 固定大端），
再 `conv.ConvertWithCalc(v, Config.Type, Value_Type, Conversion)` 做"采集类型 → 换算 → 输出类型"。
任何异常（类型无效、偏移越界、换算失败）都**不丢点**，而是把原因写进 `model.Value_type.Msg` 随数据上行。

### 6.5 写入 `Write`

1. `map[PointId]value` + 快照 `points/pdu` + `snapshotClient()`（nil → `驱动未连接或已关闭, 写入取消`）；
2. 可写校验（`RW_Cancel ∈ {3,4}`）；
3. 按 `Area+DBNumber` 分组 → `PackS7AddressPackages` 动态组包 → 生成 `writeGroup{item, pointIndices, hasBool}`；
4. 含 bool 的包先 `ReadMultiVars` 预读（按 `batchS7Items` 分批），结果回写 groups；
5. `writeUpdateBuffer` 逐点校验（时间 ≤5s、`v.Type == Value_Type`、偏移不越界）+ `conv.ConvertReverse(v, Value_Type, Config.Type, Conversion)` 反算 + 编码进缓冲区；**错误只记日志**（见坑 6）；
6. `WriteMultiVars` 按 `batchS7Items` 分批下发，任一批失败 → 返回错误（**后续批不再执行**）。

---

## 七、排障速查

| 现象 | 先看 |
|------|------|
| `libsnap7.so: cannot open shared object file` | `LD_LIBRARY_PATH`；见 §2 |
| 一直 `连接结果已作废（被新一代连接取代或驱动已关闭）` | 有并发的 `Close/Connect`（保存配置抖动），属保护性日志，非故障 |
| 连不上但 TCP 通 | TSAP 不匹配 → 手填 `Local_TSAP/Remote_TSAP`；或 `Slot` 用了默认 2 而设备是 1200/1500（应为 1） |
| 某点恒为 0 / 明显错位 | 采集类型宽度配错（如把 `Real` 配成 `DWord`）；看 `DEBUG 驱动…DataItem…` 行的 `Start/End/BufLen` |
| `字节偏移越界` | 合并包的 `Start` 与点位 `Start` 不一致 → 检查同 DB 内地址重叠 |
| 写入生效但同字节其他位被清 | 只有 bool 走预读-改-写；非 bool 是整段覆盖，同字节混布点位需拆成两个 bool 点 |
| `无有效的可读点位` → 驱动创建失败 | 该驱动下所有点 `RW_Cancel` 都是 1/3 |
| 读超时后整轮不刷新 | 弱网下先按 `s7ReadRetries/s7ReconnectAfterFailures` 重试，最长 2 轮才判重连；可上调 `Max_reconnect_interval` 观察 |

---

## 八、扩展指引

- **新增 TIA 类型**：改 `s7TypeMap`（名字映射）+ `ValueTypeToByteLen`（宽度）+ `analysis`（解码）+ `writeUpdateBuffer`（编码），
  并保持 `S7Points_type.Type` 注释里的映射表同步。
- **新增存储区**：`Point_Config_Switch` 的地址正则需扩展，`Read_Packet`/`Write` 的 `WordLen/Amount/DBNumber` 特判要加分支（参考 C/T）。
- **不要**绕过 `computeReadBatches`/`batchS7Items` 直接调 `ReadMultiVars`：超过 20 个 item 或超 PDU，C 层直接报错。
- 新驱动接入方式与 modbus 相同，见 [modbus README §六](../modbus/README.md) 与 [service/driver](../../service/driver/README.md)。

---

## 九、相关文档

- [internal/driver/modbus](../modbus/README.md) — 对照实现（无 CGO、固定间隔重连）
- [internal/service/driver](../../service/driver/README.md) — 驱动生命周期、忙碌锁、只读拦截
- [internal/service/collector](../../service/collector/README.md) — 数据落库/报警/上行
- [pkg/conv](../../../pkg/conv/README.md) — 类型编码、字节序、表达式正反算
- [internal/model](../../model/README.md) — `Value_type` 上报契约
- `third_party/lib/snap7_api_cn.md` — snap7 C API 中文说明
