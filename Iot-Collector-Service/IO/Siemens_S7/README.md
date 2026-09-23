# Siemens S7 驱动

本模块基于 [gos7](https://github.com/robinson/gos7) 库实现西门子 S7 协议通信。

## 第三方依赖声明

本模块使用了以下第三方开源库：

**gos7** - https://github.com/robinson/gos7

```
BSD 3-Clause License

Copyright (c) 2018, robinson
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

* Neither the name of the copyright holder nor the names of its
  contributors may be used to endorse or promote products derived from
  this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

**中文概要：**

BSD 3-Clause 许可证。允许自由使用、修改和分发（源码或二进制形式），需满足以下条件：
1. 保留原始版权声明
2. 二进制分发需附带版权声明
3. 不得使用原作者名称进行推广

软件按“原样”提供，作者不承担任何担保或赔偿责任。

## 驱动配置格式

配置字符串格式（分号分隔）：

```
IP;机架号;槽位号;重试间隔;轮询间隔;连接超时;响应超时;组包大小;连接类型
```

### 完整示例

```
192.168.1.1;0;2;3s;0;10s;10s;480;2
```

### 最简配置

```
192.168.1.1;0;2
```

必填字段：IP、机架号、槽位号。其余可选字段缺失时自动使用默认值。

### 字段说明

| 序号 | 字段 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| 0 | IP | 是 | - | PLC 地址，支持端口号如 `192.168.1.1:102` |
| 1 | 机架号 | 是 | - | PLC 的 rack 号（通常为 0） |
| 2 | 槽位号 | 是 | - | PLC 的 slot 号（S7-300=2, S7-1200/1500=1） |
| 3 | 重试间隔 | 否 | 3s | 掉线重连等待时间 |
| 4 | 轮询间隔 | 否 | 0 | 每次读取后的延迟，0 表示不等待 |
| 5 | 连接超时 | 否 | 10s | TCP 连接+读取超时 |
| 6 | 响应超时 | 否 | 10s | 空闲超时 |
| 7 | 组包大小 | 否 | 480 | 单次通信最大字节数（PDU） |
| 8 | 连接类型 | 否 | 2 | 1=PG编程设备 2=OP操作面板 3=Basic |

### PDU 组包大小参考

| PLC 型号 | PDU 大小 |
|----------|----------|
| S7-200 Smart | 240 |
| S7-300 | 240 |
| S7-400 | 960 |
| S7-1200 | 480 |
| S7-1500 | 960 |

默认请求 480 字节，PLC 不支持时会自动协商到更低值。

### 连接类型说明

| 值 | 类型 | 说明 |
|----|------|------|
| 1 | PG | 编程设备，权限最高，可上下载程序 |
| 2 | OP | 操作面板/HMI，只能读写数据（推荐采集用） |
| 3 | Basic | 基础连接，权限最低 |

## 点位配置格式

配置字符串格式（分号分隔）：

```
Area;DBNumber;Start;Type;Child_Address
```

### 示例

```
132;1;0;12;0
```

表示：DB数据块(0x84=132)、DB1、字节偏移0、float32(Real)、子地址0

### 字段说明

| 序号 | 字段 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| 0 | Area | 否 | 132(0x84) | 存储区，见下表 |
| 1 | DBNumber | 是 | - | DB块号 |
| 2 | Start | 是 | - | 字节偏移 |
| 3 | Type | 是 | - | 采集数据类型，见下表 |
| 4 | Child_Address | 否 | 0 | 子地址（bool 类型的位偏移） |

### 存储区（Area）对照表

| 代码 | TIA Portal | 说明 |
|------|------------|------|
| 0x81 (129) | I | 输入映像区 |
| 0x82 (130) | Q | 输出映像区 |
| 0x83 (131) | M | 标志位/中间寄存器 |
| 0x84 (132) | DB | 数据块（最常用） |
| 0x1C (28) | T | 定时器 |
| 0x1B (27) | C | 计数器 |

### 数据类型（Type）对照表

| 代码 | 代码类型 | TIA Portal | 字节数 |
|------|----------|------------|--------|
| 1 | bool | Bool | 1 |
| 2 | int8 | SInt | 1 |
| 3 | uint8 | USInt | 1 |
| 4 | int16 | Int | 2 |
| 5 | uint16 | UInt | 2 |
| 6 | int32 | DInt | 4 |
| 7 | uint32 | UDInt | 4 |
| 8 | int64 | LInt | 8 |
| 9 | uint64 | ULInt | 8 |
| 10 | int | - | 平台相关 |
| 11 | uint | - | 平台相关 |
| 12 | float32 | Real | 4 |
| 13 | float64 | LReal | 8 |
| 14 | float | - | 平台相关 |
| 15 | string | String | 可变 |

## 通信架构

### 组包机制

- 读取和写入分别独立组包，均使用 `[]gos7.S7DataItem` 结构
- 按 `Area + DBNumber` 分组，合并连续地址，受 PDU 大小限制
- 初始化时调用 `Read_Packet()` 和 `Write_Packet()` 完成预组包

### 读取流程

1. `AGReadMulti` 批量读取所有 DataItem（单次最多 20 个，超出自动分批）
2. 逐个 DataItem 检查 `Error` 字段，失败的仅上报该包内的点位
3. `analysis` 解析字节流：用 `byte_util.Get_list_index` + `BytesToXxx` 提取原始值
4. `byte_util.ConvertType` 将采集类型转换为输出类型
5. 回调 `Read_External_Mappings` 上报数据

### 写入流程

1. 校验每个写入值的点位是否允许写入（`RW_Cancel=3` 只写 或 `4` 读写）
2. `AGReadMulti` 批量读取当前数据（读-改-写模式，bool 位操作必需）
3. `writePacket` 逐包更新缓冲区：bool 用位操作，其他用 `byte_util.XxxToBytes` + `copy`
4. `AGWriteMulti` 批量写入 PLC（单次最多 20 个，超出自动分批）

### 字节序

S7 协议固定大端序，所有字节转换通过 `byte_util` 完成：

| 字节数 | 字节序常量 | 说明 |
|--------|------------|------|
| 2 | `BA` | int16 / uint16 |
| 4 | `ABCD` | int32 / uint32 / float32 |
| 8 | `ABCDEFGH` | int64 / uint64 / float64 |

## 错误处理

### 初始化阶段

| 错误信息 | 原因 |
|----------|------|
| `解析驱动配置失败` | 驱动配置字符串格式错误或字段值无效 |
| `点位 id=X 配置解析失败，跳过` | 单个点位配置字符串解析失败，仅跳过该点位 |
| `读取组包失败` / `写入组包失败` | 组包过程中发生错误 |
| `无有效的可读点位` | 所有点位均无效或无可读点位 |
| `无效 Value_Type=X 点位 id=Y，跳过` | 点位的 Value_Type 不在支持范围内 |
| `S7 组包失败` | 地址合并或 PDU 分割失败 |
| `无效 Type=X 点位 id=Y，跳过` | 写入点位的采集类型无效 |
| `无有效的可写点位，跳过写入组包` | 没有 RW_Cancel=3/4 的点位，写入组包跳过 |

### 连接阶段

| 错误信息 | 原因 |
|----------|------|
| `siemens_s7 连接失败` | TCP 连接或 S7 协议握手失败 |
| `驱动已关闭` | 调用 Close 后继续操作 |
| `读取错误: X，尝试重连` | AGReadMulti 通信失败，触发自动重连 |
| `重连最终失败: X，退出轮询` | 重连尝试超时或驱动已关闭 |
| `重连失败: X，Ys 后重试` | 单次重连失败，等待后再次尝试 |

### 轮询阶段

| 错误信息 | 原因 |
|----------|------|
| `无有效轮询包，退出轮询` | 无可用的读取数据包 |
| `DataItem[X] 读取失败: X` | 单个 DataItem 读取失败，仅上报该包内的点位 |
| `数据回调失败: X` | `Read_External_Mappings` 回调执行失败 |

### 写入阶段

| 错误信息 | 原因 |
|----------|------|
| `点位 id=X 不允许写入，RW_Cancel=Y` | 点位的 RW_Cancel 不是 3(只写) 或 4(读写) |
| `AGReadMulti 批量读取失败` | 写入前读取当前数据失败 |
| `AGWriteMulti 批量写入失败` | 批量写入 PLC 失败 |
| `写入准备失败 Area=X DB=Y Start=Z` | 单包缓冲区更新失败，跳过该包继续其他包 |
| `写入值时间间隔过长` | 写入值的时间戳与当前时间相差超过 5 秒 |
| `配置类型与写入值类型不匹配` | 写入值的 Type 与点位的 Value_Type 不一致 |
| `字节偏移越界` | 点位地址超出 DataItem 数据范围 |
| `写入值类型转换失败` | `byte_util.ConvertType` 转换失败 |
| `值类型断言失败: 期望 X, 实际 Y` | 转换后的值类型与预期不符 |
| `子地址超出范围: child_address=X, 最大7` | bool 类型的位偏移超过 0-7 |
| `不支持的写入类型: X` | 点位的采集类型不在 1-15 范围内 |

### 数据解析阶段（analysis）

| 错误信息 | 原因 |
|----------|------|
| `不支持的采集类型: 点位id=X, Type=Y` | 点位的采集类型不在 1-15 范围内 |
| `类型转换失败: 点位id=X, 采集Type=Y 输出Value_Type=Z 实际T` | `byte_util.ConvertType` 采集类型转输出类型失败 |
