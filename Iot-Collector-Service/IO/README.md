# IO 层 — 工业协议驱动与数据采集

IO 层负责工业设备的协议通信、数据采集和点位管理。采用**驱动插件化**架构，每个协议驱动独立实现，由 [manager](./manager) 统一调度。

## 目录结构

```
IO/
├── Modbus_Tcp/      # Modbus TCP 协议驱动
├── Siemens_S7/      # 西门子 S7 协议驱动
├── byte_util/       # 字节序转换工具库
├── manager/         # 驱动管理器（统一调度入口）
└── tcp_udp/         # 通用 TCP/UDP 网络连接层
```

## 协议驱动

| 驱动 | 协议 | 说明 |
|------|------|------|
| [Modbus TCP](./Modbus_Tcp/README.md) | Modbus TCP | 支持线圈、离散输入、保持/输入寄存器的读写，自动组包优化 |
| [Siemens S7](./Siemens_S7/README.md) | S7 (Siemens) | 支持 S7-200/300/400/1200/1500 系列 PLC，基于 gos7 库 |

每个驱动独立实现三个核心生命周期方法：

- **New** — 解析配置、构建组包
- **Connect** — 建立连接、启动轮询采集
- **Close** — 关闭连接、上报状态

## 公共模块

| 模块 | 说明 |
|------|------|
| [manager](./manager) | 驱动管理器，负责驱动的创建、初始化、连接、关闭和配置热重置，提供按驱动粒度的并发锁控制 |
| [byte_util](./byte_util) | 字节序转换工具，支持全部字节序排列（AB/BA/ABCD/BADC/CDAB/DCBA）及位级操作 |
| [tcp_udp](./tcp_udp) | 通用 TCP/UDP 网络连接封装，提供收发队列和超时管理 |

## 架构概览

```
┌─────────────────────────────────────────┐
│              manager (调度层)             │
│  CreateDriver / DriveNew / DriveConnect  │
│  DriveClose / DriveResetConfig           │
├──────────────┬──────────────┬────────────┤
│  Modbus_Tcp  │  Siemens_S7  │  更多驱动…  │
├──────────────┴──────────────┴────────────┤
│          byte_util / tcp_udp             │
│            (公共工具层)                    │
└─────────────────────────────────────────┘
```

**数据流：** manager 从数据库获取驱动配置和点位配置 → 调用对应驱动的 New 完成组包 → Connect 建立连接并轮询采集 → 采集结果通过回调函数上报给上层。
