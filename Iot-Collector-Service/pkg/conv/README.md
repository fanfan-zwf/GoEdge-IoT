# `pkg/conv` — 字节序 / 类型 / 数值表达式 / 时长 的底层转换工具集

上级文档：[服务总览 README](../../README.md)

本包是整个采集服务的"数据地基"：驱动层把裸字节还原成数值、采集层把数值做线性换算、配置层把字符串时长解析成 `time.Duration`，全部依赖这里。它无外部依赖、无状态、被 `internal/driver/*`、`internal/service/*`、`internal/repo/mysql`、`internal/handler` 广泛引用。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `byte_util.go` | 484 | 14 种字节序常量 + `UintX/FloatX/BoolToBytes` / `BytesToX` 系列 + 位操作 `SetBit` + 泛型列表工具 `Get_list_index` / `Update_List_Slice` |
| `convertType.go` | 987 | 15 个类型码 `TYPE_*` + `ValueTypeCode` 映射 + 15 个 `ConvXxx(v any, to int)` + `ConvertType` / `AnyToBool` |
| `calculate.go` | 177 | 基于 expr-lang 的线性换算 `ExpressionCalc` / `ExpressionReverse` / `ConvertWithCalc` / `ConvertReverse` |
| `duration.go` | 42 | `type Duration time.Duration` 的 JSON 编解码（字符串 "3s" ↔ 时长） |

---

## 二、⚠️ 最容易踩的坑（按严重度排序）

### 坑 1（头号）：14 个字节序常量里 5 个不能正确往返 —— 且 Modbus 可被用户配置触发

`byte_util.go` 定义了 14 个字节序（`iota` 连续编号，`AB=0 … HGFEDCBA=13`）。用往返测试（写入→读回应等于原值）实测，**5 个是坏的**：

| 常量 | 值 | 宽度 | 往返结果 | 结论 |
| --- | ---:| --- | --- | --- |
| `AB` | 0 | 16 | ✅ | 可用 |
| `BA` | 1 | 16 | ✅ | 可用 |
| `ABCD` | 2 | 32 | ✅ | 可用 |
| **`BADC`** | 3 | 32 | ❌ `back=11334422`（应 11223344） | **禁用** |
| **`CDAB`** | 4 | 32 | ❌ `back=11442233` | **禁用** |
| `DCBA` | 5 | 32 | ✅ | 可用 |
| `ABCDEFGH` | 6 | 64 | ✅ | 可用 |
| **`BADCFEHG`** | 7 | 64 | ❌ `back=1122337744665588` | **禁用** |
| **`CDABGHEF`** | 8 | 64 | ❌ `back=1133224488665577` | **禁用** |
| `DCBAHGFE` | 9 | 64 | ✅ | 可用 |
| `EFGHABCD` | 10 | 64 | ✅ | 可用 |
| `FEHGDCBA` | 11 | 64 | ✅ | 可用 |
| `GHEFCDAB` | 12 | 64 | ✅ | 可用 |
| **`HGFEDCBA`** | 13 | 64 | ❌ `back=4433221155667788` | **禁用** |

> 注：坏序在"编码→解码"两步里字节的交换不是彼此的逆操作，导致还原值 ≠ 原值。这类字节序（如 `CDAB`=word-swap、`BADC`=byte-swap-within-words）在工业设备里其实是常见的，本包实现有误。

**可达性（关键）**：
- `internal/driver/siemens`：只硬编码 `conv.BA / ABCD / ABCDEFGH`（全在可用列），**不受影响**。
- `internal/driver/modbus`：字节序来自点位地址串第 6 段 `cfg.Config.Byte_Order`，`modbus/new.go:100` **仅校验字符串是否在 `Byte_Value` 表里存在**，不校验是否正确 → 用户填 `CDAB`/`BADC`/`HGFEDCBA` 等能过校验，但采到的 32/64 位值会**静默错乱**。

**排障**：Modbus 采集 32/64 位寄存器数值明显不对但字节本身没读错时，优先怀疑地址串里用了上述 5 个坏序。

### 坑 2：`iota` 跨宽度共号 → 短函数收到"越界"order 静默返全零

字节序常量是全局连续编号，但 `Uint16ToBytes`/`Uint32ToBytes` 各自只 `switch` 自己那几种 order，**且没有 `default` 分支**。于是：
```go
Uint16ToBytes(v, ABCD)   // ABCD=2 不在 uint16 的 switch 里 → 落到"什么都不做" → 返回 [0 0]
Uint32ToBytes(v, order=99) // → [0 0 0 0]，不报错
```
未知/跨宽度 order **不 panic、不报错，直接给全零**，是最难排查的一类"数据莫名变 0"。

### 坑 3：`BytesToUint32(len<4)` / 非 4 倍数长度 → 返回空切片

`BytesToX` 系列对切片长度没有前置校验，`BytesToUint32` 收到 3 字节返回 `[]`（长度 0），调用方若继续按 4 字节消费会 panic 或读到脏值。

### 坑 4：`BytesToBool` 返回长度 = `len(bytes)*8`，与输入 bool 数组不等长

`BoolToBytes([true false true true])` → `[13]`（1 字节 packed），`BytesToBool([13])` → 长度 8 的数组 `[true false true true false false false false]`。往返多出的高位补零，比较时不能直接 `reflect.DeepEqual` 原数组。

### 坑 5：`SetBit(data []byte, bit int, v bool)` 无越界检查

内部 `data[bit/8]`，`bit` 超 `len(data)*8-1` 直接 panic，调用方（写值路径）需自行保证 bit 范围。

### 坑 6：`Conv*` 的第二参数名叫 `to`，实际语义是"v 当前声明的类型"

`convertType.go` 每个 `ConvXxx(v any, to int)` 的 `switch to` 分派的是**输入值 v 现在的类型**，函数名才是"目标类型"。所有调用方一律传 `cfg.Value_Type`（即当前类型）。命名反直觉，读码时极易误解为转换目标。

### 坑 7：整型互转无范围检查 → 静默截断

`ConvUint8(v, TYPE_UINT16)` 走 `uint8(val)`：把 300 转成 `uint8` 得 `44`，不报错不溢出提示。写值 / 换算链路上的越界值会被悄悄截断。

### 坑 8：`TYPE_FLOAT` 与 `TYPE_FLOAT64` 都断言 Go `float64`

`case TYPE_FLOAT64, TYPE_FLOAT:` 里统一 `v.(float64)`。若某值实际是 Go `float32` 却带了 `TYPE_FLOAT` 编码，断言失败 → 返回 `(0, false)`。float32 只能配 `TYPE_FLOAT32`。

### 坑 9：字符串解析用带 bitSize 的 `ParseUint/ParseInt`，越界即失败

字符串→整型分支用 `strconv.ParseUint(str,10,8|16|32)`，超 bitSize 直接 `err` → `(0,false)`；但 `ConvUint`/`ConvInt` 用 bitSize `64`，在 32 位平台 `uint/int` 是 32 位，解析出 64 位值再转 `uint` 可能溢出。

### 坑 10：`ExpressionCalc` 每次调用都 `expr.Compile`（无缓存）

采集/报警换算热路径每算一次都重新编译表达式字符串，CPU 开销大。对比 `collector/alarm.go` 用 `alarmProgramCache` 缓存编译结果，这里没有。高频点位换算应关注此热点。

### 坑 11：`ExpressionReverse` 只支持一元一次方程

反算用两点法：`b=f(0)`、`k=f(1)-f(0)`，`|k|<1e-15` 报"不是有效的一元一次方程"。含 `x*x`、`sin(x)` 等非线性 Conversion 无法反算。

### 坑 12：`toFloat64` 不支持 uint 系与小整型

`calculate.go` 的 `toFloat64` 只处理 `float64/float32/int/int64`；传入 `uint`、`int8~int32`、`uint8~uint64` 会走失败分支。换算前需确保值已是受支持类型（通常经 `ConvFloat64` 已归一）。

### 坑 13：`Update_List_Slice` 注释与实现不符

注释写"超出部分忽略"，实现是**自动扩容**（允许 `startIndex==arrLen` 追加）。以代码为准。`Get_list_index` 的文档注释标题还残留旧名 `GetItemOrDefault`；两者命名是蛇形，与包内其余驼峰风格不一致。

### 坑 14：`ValueTypeIntToString` 遍历 map 反查，顺序不确定 + 依赖唯一性

它 `for name,code := range ValueTypeCode` 找第一个匹配的 code。因每个 code 唯一所以结果确定，但这是"靠数据唯一性撑着"的写法；若将来两个类型名共用一个码会随机返回。未找到返回 `unknown(N)`。

### 坑 15：`conv.Duration` 的 `null` / 空串容错语义

`UnmarshalJSON` 对 `"null"` 与 `""` 置 0 且不报错（注释：由调用方决定是否回退默认）。若业务把 0 当"未配置"要对每个字段显式判 0；`config.go` 里就有把 `==0` 归一化回默认值的动作（如 ALARM TTL）。

---

## 三、常量 / 映射 / API 速查

### 3.1 字节序常量（`byte_util.go`，`iota` 从 0）
`AB=0 BA=1 ABCD=2 BADC=3 CDAB=4 DCBA=5 ABCDEFGH=6 BADCFEHG=7 CDABGHEF=8 DCBAHGFE=9 EFGHABCD=10 FEHGDCBA=11 GHEFCDAB=12 HGFEDCBA=13`
`Byte_Value map[string]int`：以上 14 个字符串名 → 数值，供地址串解析（`Byte_Value[str]`）。

### 3.2 类型码（`convertType.go`，`iota+1` 从 1）
`TYPE_BOOL=1 INT8=2 UINT8=3 INT16=4 UINT16=5 INT32=6 UINT32=7 INT64=8 UINT64=9 INT=10 UINT=11 FLOAT32=12 FLOAT64=13 FLOAT=14 STRING=15`
`ValueTypeCode map[string]int`：类型名 → 码；`ValueTypeStringToInt`（正查，带 ok）；`ValueTypeIntToString`（反查，见坑 14）。

### 3.3 转换函数族
| 方向 | 函数 |
| --- | --- |
| any → 各类型 | `ConvBool/ConvUint8/…/ConvString(v any, to int)`（`to`=当前类型，见坑 6），统一入口 `ConvertType(value, fromType, toType)` |
| any → bool（宽松） | `AnyToBool`（覆盖 bool/所有整型/float/uintptr；**string 与 nil 返回 false**） |
| 字节 ↔ 数值 | `Uint16/Int16/Uint32/Float32/Uint64/Float64ToBytes(v, order)` / `BytesToUintX/IntX/FloatX(b, order)` |
| 位打包 | `BoolToBytes` / `BytesToBool` / `SetBit` |
| 列表 | `Get_list_index[T](arr, i)`（越界补零值、`cnt<=0` 返回空）/ `Update_List_Slice[T](arr, start, sub)`（自动扩容） |
| 换算 | `ExpressionCalc(x, expr)` / `ExpressionReverse(y, expr)` / `ConvertWithCalc(v, fromType, toType, expr)` / `ConvertReverse(...)` |

### 3.4 `ExpressionCalc` 环境
expr 程序里可用变量 `x`（float64），自定义函数 `round2`（1 参 = 四舍五入 2 位，2 参 = 按精度）。`round2` 用 expr v1.17.8 的 `expr.Function(name, fn, new(sig...))` 写法注册双重载。`Conversion=="0"` 或空串视为"不换算"直通。

---

## 四、核心流程（Modbus 采集值还原）

```
地址串 "...:34:CDAB:float32"
   │  modbus/new.go:100  Byte_Value["CDAB"] 存在→通过校验（坑1：实为坏序）
   ▼
读回 4 字节寄存器 raw
   ▼
conv.BytesToFloat32(raw, conv.CDAB)   ← 按坏序交换，float 值错乱且不报错
   ▼
ConvFloat32(val, cfg.Value_Type)       ← to=当前类型（坑6）
   ▼
ConvertWithCalc(v, fromType, toType, point.Conversion)
   │  ExpressionCalc 每次 expr.Compile（坑10）
   ▼
写入 collector 值总线
```

---

## 五、排障速查

| 现象 | 定位 | 处理 |
| --- | --- | --- |
| Modbus 32/64 位值乱、字节却读对了 | 地址串用了 `CDAB/BADC/BADCFEHG/CDABGHEF/HGFEDCBA`（坑 1） | 换用可用序；或本包修复对应 case |
| 数值莫名变 0 | order 与函数宽度不匹配 → 全零（坑 2）；或 `BytesToX` 收到不足长切片（坑 3） | 校验字节数 = 类型宽度，order 匹配宽度 |
| 写值截断（300→44） | 整型互转无范围检查（坑 7） | 上游限幅或用合适类型 |
| float 值转失败 `(0,false)` | float32 配了 `TYPE_FLOAT`（坑 8） | float32 用 `TYPE_FLOAT32` |
| 换算 CPU 高 | 热路径每次重编译表达式（坑 10） | 引入编译缓存（仿 alarm 的 `sync.Map`） |
| 反算报错 | 非线性 Conversion（坑 11） | Conversion 保持线性 |
| 配 "3s" 未生效/为 0 | JSON null/空串被置 0（坑 15） | 调用方显式判 0 回退默认 |

---

## 六、扩展指引

- **新增字节序**：先在 `iota` 块末尾追加常量（勿打乱既有编号），再**同步补齐**所有对应宽度 `*ToBytes/*FromBytes` 的 `switch` 分支并加 `default` 报错（避免坑 2 的全零）；务必写往返测试（可参考本包临时验证脚本的写法）确认 `back==原值`。
- **新增类型码**：`TYPE_*` 常量 + `ValueTypeCode` + 对应 `ConvXxx` 的**全部其它 `ConvYyy` 都要加该 case**（否则跨类型转换漏分支）。
- **表达式换算**：如需性能，把 `ExpressionCalc` 改成按 conversion 字符串缓存 `vm.Program`；注意 expr 版本 API。

---

## 七、相关文档

- Modbus 地址串解析与字节序消费：[internal/driver/modbus README](../../internal/driver/modbus/README.md)
- Siemens 字节序消费（安全子集）：[internal/driver/siemens README](../../internal/driver/siemens/README.md)
- 采集值换算 / 报警表达式消费：[internal/service/collector README](../../internal/service/collector/README.md)
- `conv.Duration` 在配置解析中的使用：[internal/config README](../../internal/config/README.md)
- 类型编码在写值路径的使用：[internal/handler README](../../internal/handler/README.md)
