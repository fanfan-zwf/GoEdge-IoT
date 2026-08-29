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
