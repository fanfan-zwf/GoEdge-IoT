# Modbus TCP 驱动

本模块基于 [go-modbus](https://github.com/things-go/go-modbus) 库实现 Modbus TCP 协议通信，支持线圈、离散输入、保持寄存器和输入寄存器的读写操作。

## 第三方依赖声明

本模块使用了以下第三方开源库：

**go-modbus** - https://github.com/things-go/go-modbus

```
MIT License

Copyright (c) 2019 things-go

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

**中文概要：**

MIT 许可证。允许自由使用、修改、分发和 sublicense，只需保留原始版权声明和许可声明。软件按"原样"提供，作者不承担任何担保或赔偿责任。

## 驱动配置格式

配置字符串格式（分号分隔）：

```
IP;重试间隔;连接超时;响应超时;轮询间隔;组包最大长度
```

### 完整示例

```
192.168.1.1;3s;3s;180s;1s;256
```

### 最简配置

```
192.168.1.1
```

必填字段：IP。其余可选字段缺失时自动使用默认值。

### 字段说明

| 序号 | 字段 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| 0 | IP | 是 | - | Modbus TCP 设备地址，如 `192.168.1.1` |
| 1 | 重试间隔 | 否 | 3s | 连接失败后的重试等待时间 |
| 2 | 连接超时 | 否 | 3s | TCP 连接建立超时 |
| 3 | 响应超时 | 否 | 180s | 等待从机响应的最大超时 |
| 4 | 轮询间隔 | 否 | 1s | 每次读取请求之间的延迟 |
| 5 | 组包最大长度 | 否 | - | 单次通信最大寄存器数量，**必须为 2 的倍数且大于 0** |

> **注意：** 组包最大长度（Packet_max）控制单次 Modbus 请求中合并的寄存器数量上限。值越大通信效率越高，但受限于设备 PDU 大小。常见 Modbus 设备单次最大读取 125 个寄存器（250 字节）。

## 点位配置格式

配置字符串格式（分号分隔）：

```
从机地址;功能码;寄存器地址;字节序;数据类型
```

### 示例

```
1;3;100;ABCD;float32
```

表示：从机地址 1、读保持寄存器、寄存器地址 100、大端字节序、float32 类型。

### 字段说明

| 序号 | 字段 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| 0 | 从机地址 | 是 | - | Modbus 从机站号（1-247） |
| 1 | 功能码 | 是 | - | Modbus 功能码，见下表 |
| 2 | 寄存器地址 | 是 | - | 寄存器地址（1-based，内部自动减 1 转换为 0-based） |
| 3 | 字节序 | 否 | - | 多字节数据的字节排列顺序，见下表 |
| 4 | 数据类型 | 是 | - | 采集数据类型，见下表 |

### 寄存器地址格式

- **普通类型：** 直接写寄存器地址，如 `100`
- **bool 类型（寄存器模式）：** 使用 `地址.子地址` 格式，如 `100.3` 表示第 100 号寄存器的第 3 位（bit3），子地址范围 0-15

### Modbus 功能码对照表

| 功能码 | 名称 | 说明 | 读写 |
|--------|------|------|------|
| 1 (0x01) | ReadCoils | 读线圈 | 只读 |
| 2 (0x02) | ReadDiscreteInputs | 读离散输入 | 只读 |
| 3 (0x03) | ReadHoldingRegisters | 读保持寄存器 | 读/写 |
| 4 (0x04) | ReadInputRegisters | 读输入寄存器 | 只读 |

> **写入支持：** 当前写入操作仅支持功能码 1（线圈）和功能码 3（保持寄存器），分别使用 WriteMultipleCoils(15) 和 WriteMultipleRegistersBytes(16) 进行批量写入。

### 字节序说明

| 字节序 | 字节数 | 内存排列 | 说明 |
|--------|--------|----------|------|
| AB | 2 | 低字节在前 | 小端（Little-Endian） |
| BA | 2 | 高字节在前 | 大端（Big-Endian，Modbus 默认） |
| ABCD | 4 | 标准大端 | 大端（Big-Endian） |
| BADC | 4 | 字内交换 | 常用小端（Word-Swap） |
| CDAB | 4 | 字间交换 | 中端（Mid-LittleEndian） |
| DCBA | 4 | 完全反转 | 全反小端 |

> **2 字节类型**（bool、int16、uint16）使用 `AB`/`BA`；**4 字节类型**（int32、uint32、float32）使用 `ABCD`/`BADC`/`CDAB`/`DCBA`。

### 数据类型（Type）对照表

| 类型名 | 字节数 | 说明 |
|--------|--------|------|
| bool | 1 | 布尔/开关量 |
| int16 | 2 | 有符号 16 位整数 |
| uint16 | 2 | 无符号 16 位整数 |
| int32 | 4 | 有符号 32 位整数 |
| uint32 | 4 | 无符号 32 位整数 |
| float32 | 4 | 32 位浮点数 |

## 组包逻辑

驱动自动将地址连续或重叠的点位合并为一次 Modbus 请求，减少通信次数：

1. 按 **从机地址 + 功能码** 对点位进行分组
2. 每组内按寄存器地址排序
3. 连续或重叠的地址自动合并为一个请求包
4. 合并后的包大小不超过配置的 `组包最大长度`
5. 不连续的点位拆分为独立的请求包

## 读写模式

点位的读写模式由数据库 `RW_Cancel` 字段控制：

| RW_Cancel 值 | 含义 | 说明 |
|---------------|------|------|
| 0 | 禁用 | 不参与读取和写入 |
| 1 | 只读 (R) | 仅参与采集读取 |
| 2 | 只读 (R) | 仅参与采集读取 |
| 3 | 只写 (W) | 仅参与写入操作 |
| 4 | 读写 (R/W) | 同时参与采集读取和写入操作 |

## 错误处理策略

- **单点位失败隔离：** 单个点位配置解析失败时跳过该点位，不影响其他点位正常工作
- **错误信息上报：** 通信异常时通过 `Error_External_Mappings` 将所有受影响点位的错误信息写入 `Msg` 字段，确保上层能感知每个点位状态
- **写入超时保护：** 写入值的时间戳超过 5 秒时拒绝写入，防止过期数据污染设备
- **类型安全：** 所有写入操作使用 comma-ok 惯用语法进行类型断言，避免运行时 panic
- **FC03 bool 同地址缓存：** 同一寄存器地址的多个 bool 点位只读一次 PLC，避免重复通信

## 错误信息说明

### 初始化阶段

| 错误信息 | 原因 |
|----------|------|
| `解析驱动配置失败` | 驱动配置字符串格式错误或字段值无效 |
| `点位 id=X 配置解析失败，跳过` | 单个点位配置字符串解析失败，仅跳过该点位 |
| `点位 id=X 输出类型未配置，使用采集类型 Y 作为默认值` | 点位的 Value_Type 未配置，自动回退到采集类型 |
| `组包失败` | 读取组包过程中发生错误 |

### 连接阶段

| 错误信息 | 原因 |
|----------|------|
| `modbus_tcp 驱动:X 连接成功 地址:Y` | 连接成功（日志信息） |
| `驱动连接已关闭` | 调用 Close 后继续操作，上报所有点位错误 |
| `连接失败，Xs 后重试` | TCP 连接失败，等待后重试 |

### 轮询阶段

| 错误信息 | 原因 |
|----------|------|
| `驱动:X 无有效轮询包，退出轮询` | 无可用的读取数据包 |
| `驱动:X 连接已关闭，退出轮询` | 驱动已关闭，正常退出 |
| `点位 id=X 索引查找失败，跳过` | 点位在配置中找不到，跳过该点位 |
| `设备id:X 包 id=Y 读取错误: Z` | Modbus 通信读取失败，上报包内所有点位错误 |
| `Unknown function code` | 不支持的功能码，上报包内所有点位错误 |
| `点位 id=X 值类型转换失败, 采集类型=Y, 输出类型=Z, 实际T, 跳过` | `byte_util.ConvertType` 采集类型转输出类型失败 |

### 写入阶段

| 错误信息 | 原因 |
|----------|------|
| `写入值不存在, 点位id: X` | 写入列表中找不到该点位的值 |
| `写入值时间间隔过长, 点位id=X, 间隔=Y` | 写入值的时间戳与当前时间相差超过 5 秒 |
| `配置类型与写入值类型不匹配, 点位id=X, 配置类型=Y, 值类型=Z` | 写入值的 Type 与点位的 Value_Type 不一致 |
| `写入值类型转换失败, 点位id=X, 配置类型=Y, 值类型=Z` | `byte_util.ConvertType` 转换失败 |
| `点位 id=X 值类型断言失败: 期望 Y, 实际 Z` | 转换后的值类型与预期不符 |
| `点位 id=X 子地址超出范围: child_address=Y, 最大15` | FC03 bool 类型的位偏移超过 0-15 |
| `点位 id=X 读取保持寄存器失败` | FC03 bool 写入前读取当前寄存器失败 |
| `不支持的写入类型: type=X, function=Y, 点位id=Z` | 类型与功能码组合不在支持范围内 |
| `未匹配到有效写入功能码: function=X` | 写入包的功能码不是 1 或 3 |
| `ERROR 组包失败` | 写入组包过程中发生错误 |
