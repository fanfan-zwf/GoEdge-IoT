# Iot-Collector-Service

工业物联网边缘数据采集服务：支持多种工业协议（Modbus TCP、Siemens S7）的设备数据采集、报警判断、历史记录，并通过 MQTT / WebSocket / HTTP API 与云端和前端交互。

> 🌐 **在线 Demo**：<http://collector.demo.zwf.fan/> —— 本服务与 [前端管理台](../Iot-Collector-Web/README.md) **前后端一体化部署**的完整系统演示实例，可直接打开体验。

> 本文件是**服务总览入口**。每个包都有一份独立的详细 README（含"最易踩的坑""排障速查""扩展指引"），请从下方 [🧭 模块导航](#-模块导航) 点进去。

## 📋 目录

- [技术栈](#-技术栈)
- [🧭 模块导航](#-模块导航)
- [项目结构](#-项目结构)
- [系统架构](#-系统架构)
- [快速开始](#-快速开始)
- [配置说明](#-配置说明)
- [开发指南](#-开发指南)
- [许可证](#-许可证)

## 🔧 技术栈

| 类别 | 技术 | 说明 |
|------|------|------|
| 语言 | Go 1.27 | 主开发语言（`go.mod` 声明 `go 1.27`；部分文件使用 `encoding/json/v2`） |
| Web 框架 | Gin | RESTful API 服务 |
| 工业协议 | goburrow/modbus、snap7 (CGO) | Modbus TCP / Siemens S7 驱动 |
| 消息队列 | Eclipse Paho MQTT | 与云端 / 内部总线通信 |
| 时序数据库 | InfluxDB（可选） | 历史数据存储 |
| 关系数据库 | MySQL | 配置、报警、历史、账号与 RBAC 数据存储 |
| 缓存 | Redis、freecache | 会话 / 热点数据缓存 |
| 表达式引擎 | expr-lang | 报警条件与数值换算动态求值 |
| WebSocket | gorilla/websocket | 前端实时数据推送 |
| HTTP 客户端 | resty.dev/v3 | 向 Config-Service 拉取配置 |
| Excel 导出 | excelize | 配置批量导出 |
| 系统信息 | gopsutil/v3 | 主机 / 进程 / 网络指标采集 |

## 🧭 模块导航

按分层组织，点击"详细文档"进入各包独立 README。

### 程序入口

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `cmd/server` | 初始化顺序、信号监听、优雅退出预算 | [README](cmd/server/README.md) |

### 基础设施与配置

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `internal/config` | YAML 配置结构体、加载与默认值归一化、日志初始化 | [README](internal/config/README.md) |
| `internal/model` | `Value_type` 统一数据模型 | [README](internal/model/README.md) |
| `internal/infra/mqtt` | MQTT 客户端封装 | [README](internal/infra/mqtt/README.md) |
| `internal/cloud` | 云端通信（加密 / 预留连接） | [README](internal/cloud/README.md) |

### 驱动层

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `internal/driver/modbus` | Modbus TCP 连接 / 轮询 / 读写 / 组包 / 地址串解析 | [README](internal/driver/modbus/README.md) |
| `internal/driver/siemens` | Siemens S7（snap7 CGO）连接 / 轮询 / 读写 / PDU 分批 | [README](internal/driver/siemens/README.md) |

### HTTP API 层

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `internal/handler` | 路由 / 鉴权 / 点位 / 配置 / 导出 / 系统 / 日志 / RBAC / 用户 / APIKey | [README](internal/handler/README.md) |

### 业务服务层

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `internal/service/collector` | 采集核心：实时值总线、报警、历史、写值 | [README](internal/service/collector/README.md) |
| `internal/service/driver` | 驱动管理与启动（并发控制、生命周期） | [README](internal/service/driver/README.md) |
| `internal/service/internalmqtt` | 内部 MQTT 总线：数据上报 / 指令下发 / 系统主题 | [README](internal/service/internalmqtt/README.md) |
| `internal/service/configsync` | 从 Config-Service 同步配置（默认关闭） | [README](internal/service/configsync/README.md) |
| `internal/service/rbac` | RBAC 鉴权服务层（规则装载 / 决策 / 数据域） | [README](internal/service/rbac/README.md) |

### 数据访问层

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `internal/repo/mysql` | MySQL 连接、表结构自维护、业务 / 账号 / 日志 / 标签树 / RBAC CRUD | [README](internal/repo/mysql/README.md) |
| `internal/repo/redis` | Redis 连接、会话与缓存、消息队列 | [README](internal/repo/redis/README.md) |
| `internal/repo/influxdb` | InfluxDB 时序写入与历史查询 | [README](internal/repo/influxdb/README.md) |
| `internal/repo/tsdb` | 时序数据库抽象接口 | [README](internal/repo/tsdb/README.md) |

### 公共工具包

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `pkg/conv` | 字节序 / 类型 / 表达式换算 / 时长 转换（⚠️ 含 5 个不可往返字节序） | [README](pkg/conv/README.md) |
| `pkg/rbac` | 类 MQTT 主题的权限匹配与决策 | [README](pkg/rbac/README.md) |
| `pkg/security` | Key 生成、bcrypt 口令 | [README](pkg/security/README.md) |
| `pkg/sysinfo` | gopsutil 系统指标采集封装 | [README](pkg/sysinfo/README.md) |
| `pkg/netutil` | TCP/UDP 客户端工具（**当前全项目零引用**） | [README](pkg/netutil/README.md) |

## 📁 项目结构

```
Iot-Collector-Service/
├── cmd/server/                    # 程序入口（main.go：初始化、信号、优雅退出）
├── configs/config.yaml            # YAML 配置（见下方「配置说明」的实际段结构）
├── internal/
│   ├── cloud/                     # 云端通信（encryption.go / server.go）
│   ├── config/                    # 配置加载与日志（config.go / log.go）
│   ├── driver/
│   │   ├── modbus/                # Modbus TCP 驱动（modbus.go / new.go / packet.go）
│   │   └── siemens/               # Siemens S7 驱动（siemens.go / new.go / packet.go / snap7_cgo.go）
│   ├── handler/                   # HTTP API：router / server / auth / apikey / point / config /
│   │                              #   export / system / log / user / rbac / rbac_admin / tagtree / docs
│   ├── infra/mqtt/                # MQTT 客户端封装（mqttbase.go）
│   ├── model/                     # Value_type 统一定义（value.go）
│   ├── repo/
│   │   ├── influxdb/              # 时序写入（influxdb.go）
│   │   ├── mysql/                 # mysql.go / schema / business / account / log / tagtree / rbac
│   │   ├── redis/                 # redis.go / logic / queue
│   │   └── tsdb/                  # TSDB 抽象接口
│   └── service/
│       ├── collector/             # db_point / alarm / history / write（采集核心）
│       ├── configsync/            # sync.go / ApiConfig.go
│       ├── driver/                # init.go / manager.go
│       ├── internalmqtt/          # 内部 MQTT 总线
│       └── rbac/                  # RBAC 服务层
├── pkg/
│   ├── conv/                      # byte_util / calculate / convertType / duration
│   ├── netutil/                   # tcp_or_udp.go（零引用）
│   ├── rbac/                      # decide / scope / topic（+ 3 个 _test.go）
│   ├── security/                  # key.go / pwd.go
│   └── sysinfo/                   # sysinfo.go
├── logs/                          # 运行日志（按日期分文件）
├── third_party/lib/               # libsnap7.so / snap7.h / snap7_api_cn.md
├── Makefile / go.mod / go.sum
└── run.sh / stop.sh
```

## 🏗️ 系统架构

```
┌──────────────────────────────────────────────────────────────────────┐
│                          Iot-Collector-Service                         │
│  main.go ─▶ 初始化：config → MySQL/Redis → InfluxDB → collector 总线    │
│                     → internalmqtt → handler(Web) → driver 启动         │
├──────────────────────────────────────────────────────────────────────┤
│  驱动管理器 (service/driver)                                            │
│    ├─ Modbus TCP (goburrow/modbus)   ├─ Siemens S7 (snap7 CGO)          │
│         │ 采集值 → conv 字节序/类型还原 │                                │
├─────────┼──────────────────────────────┼───────────────────────────────┤
│         ▼          数据采集核心 (service/collector)                      │
│   collectionBus ─▶ 值变化检测 ─▶ updateBus ─┬─▶ 报警判断(expr) ─▶ alarmBus│
│         │                                   └─▶ 历史判断 ─▶ historyBus   │
├─────────┼───────────────────────────────────────────────────────────────┤
│         ▼              存储 & 分发                                        │
│   InfluxDB(历史)   MySQL(配置/报警/账号/RBAC)   Redis(会话/缓存)          │
│   internalmqtt(上报/指令/系统主题)   handler(HTTP API + WebSocket)       │
│   configsync(可选：拉取 Config-Service 配置)                             │
└──────────────────────────────────────────────────────────────────────────┘
```

数据流细化与各总线订阅者见 [collector 详细文档](internal/service/collector/README.md)。

## 🚀 快速开始

### 环境要求

- Go 1.27+
- MySQL 5.7+
- Redis 6.0+
- InfluxDB 2.x（可选，`configs/config.yaml` 未提供 `Influxdb` 段时默认关闭）
- MQTT Broker（EMQX / Mosquitto）
- snap7 动态库（`third_party/lib/libsnap7.so`，CGO）

### 编译运行

```bash
make build   # 编译
make run     # 运行
go run ./cmd/server
make clean   # 清理产物
```

> 若默认 GOCACHE 不可写，编译需显式指定**绝对路径**：
> `GOCACHE="$(pwd)/.gotmp/gocache" go build ./...`

## ⚙️ 配置说明

`configs/config.yaml` 的顶级段：`APP` `MYSQL` `REDIS` `Influxdb` `LOG` `API` `Internal_Mqtt` `ALARM` `RBAC`。

> ⚠️ 实际文件**不含** `Influxdb` 与 `Config_Service` 段 → 时序写入与配置同步两项可选能力默认关闭；段结构与默认值细节见 [config 详细文档](internal/config/README.md) 与 [configsync 详细文档](internal/service/configsync/README.md)。

### 驱动配置格式

两个驱动的 `Drive_Config.Config` 都是 **JSON 对象**，入库前由 `mysql.Config__Validate_JSON` 把关，非 JSON 直接拒绝。

**Modbus TCP:**
```
{"Address":"IP:端口","Retry_timeout":"3s","Connect_timeout":"10s","Response_timeout":"10s","Delay_between_polls":"0s","Packet_max":120,"Write_Coils_Function":15,"Write_Register_Function":16}
示例: {"Address":"192.168.1.100:502","Retry_timeout":"3s","Packet_max":120}
```

**Siemens S7（PDU 自动协商）:**
```
{"Address":"IP地址","Rack":机架号,"Slot":槽位号,"Retry_timeout":"3s","Connect_timeout":"10s","Response_timeout":"10s","Delay_between_polls":"0s","ConnectionType":连接类型,"Max_reconnect_interval":"30s","Local_TSAP":0,"Remote_TSAP":0}
// ConnectionType: 1=PG 2=OP 3=Basic（默认2）; Max_reconnect_interval: 掉线重连指数退避最大间隔（默认30s）
// Local_TSAP/Remote_TSAP: snap7 原生 uint16 TSAP 覆盖，0=按 ConnectionType+Rack+Slot 自动推导
```
> PDU 长度无需配置：连接时向 PLC 请求上限 960 字节，PLC 自动下调到自身支持值（Smart200/300=240、1200=480、400/1500=960）。早期分号分隔文本已废弃，会被入库校验拒绝。

### 点位配置格式

`Point_Config.Config` **不做统一格式校验**：各驱动的 `Point_Config_Switch` 自己解释，仓库层只过 `mysql.Config__Validate_Basic`（非空、不超长、无控制字符、无 SQL 注入片段）。

**Modbus TCP 点位（冒号分隔）:**
```
从机地址:功能码:寄存器地址[.子地址]:字节序:数据类型
示例: 1:3:100:ABCD:float32、1:1:100.5:ABCD:bool
```
> ⚠️ **字节序陷阱**：14 个字节序里 `BADC / CDAB / BADCFEHG / CDABGHEF / HGFEDCBA` 五个实现有误、不能正确往返，但能通过 `new.go` 的存在性校验，会让 32/64 位值静默错乱。详见 [conv 详细文档](pkg/conv/README.md)（坑 1）。

**Siemens S7 点位（地址串或 JSON 两种都行）:**
```
地址串: DB1.DBD0 Real、DB1.DBX0.5 Bool、M10.0 Byte、DB1.DBB8 USInt
JSON:  {"Area":132,"DBNumber":1,"Start":0,"Child_Address":0}
```

### 字段格式校验一览

| 字段 | 规则 | 校验位置 |
|------|------|----------|
| `Drive_Config.Config` | 必须是合法 JSON 对象 | `mysql.Config__Validate_JSON` |
| `Point_Config.Config` | 只挡注入片段与控制字符 | `mysql.Config__Validate_Basic` |
| `Point_Config.Tag` | `//设备id//标识符` 或 `/标识符` | 仓库层 |
| `Point_Config.Conversion` | 留空、`0` 或含 x 的表达式（可 `round2(x,2)`；反向换算限一元一次） | 仓库层 |

> 前三条正则在前端的镜像是 `Iot-Collector-Web/src/utils/configFormat.ts`，改正则必须同步前端，否则会出现"前端能提交、后端回 417"的裂缝。

## 🔨 开发指南

### 添加新驱动

1. 在 `internal/driver/` 创建新目录（如 `opcua/`）
2. 实现驱动结构体，包含：`New / Connect / Close / UpdatePoints / Write`
3. 在 `internal/service/driver/manager.go` 的 `CreateDriver` 中注册新驱动类型

细节与生命周期（`Close → 6s → New → 3s → Connect`、按驱动 ID 独立锁）见 [driver 详细文档](internal/service/driver/README.md)。

### 添加新 API

在 `internal/handler/` 增加处理函数并在 `router.go` 注册；统一响应信封 `{Code,Msg,Data,Timestamp}` 由 `Response_Use` 生成。鉴权（HMAC / Bearer）、RBAC 主题匹配见 [handler](internal/handler/README.md) 与 [service/rbac](internal/service/rbac/README.md)。

### 数据模型

```go
type Value_type struct {
    PointId uint      // 点位 ID
    Tag     string    // 点位标签
    Value   any       // 点位值
    Type    int       // 值类型编码（见 pkg/conv 的 TYPE_*）
    Msg     string    // 状态信息
    Time    time.Time // 采集时间
}
```

## 📄 许可证

本项目采用 BSD 3-Clause 许可证。主要第三方依赖：goburrow/modbus (MIT)、snap7 (BSD-3)、gin (MIT)、paho.mqtt.golang (EPL/EDL)、influxdb-client-go (MIT)、expr-lang (MIT)、resty (Apache-2.0)、gopsutil (BSD-3)。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 联系方式

作者：范范zwf
