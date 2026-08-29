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
