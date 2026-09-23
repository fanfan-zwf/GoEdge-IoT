# Iot-Collector-Service — IoT 数据采集与边缘计算服务

Iot-Collector-Service 是 GoEdge-IoT 平台的核心数据采集服务，负责工业设备的数据采集、协议转换、实时处理、报警判断和历史数据存储。

## 📋 项目概述

### 核心功能
- **多协议支持**：Modbus TCP、Siemens S7 等工业协议驱动
- **实时数据采集**：毫秒级数据采集与内存缓存
- **智能报警系统**：基于 expr 表达式引擎的动态报警规则
- **历史数据存储**：InfluxDB 时序数据库集成，支持毫秒级精度查询
- **配置管理**：完整的 CRUD API 与 Excel 导出功能
- **WebSocket 推送**：实时数据流式推送到前端
- **远程配置同步**：从 Iot-Config-Service 自动拉取和更新配置

### 技术栈
| 类别 | 技术 |
|------|------|
| 语言 | Go 1.21+ |
| Web 框架 | Gin |
| 数据库 | MySQL (配置存储), InfluxDB v2 (时序数据), Redis (缓存/队列) |
| 工业协议 | Modbus TCP, Siemens S7 |
| 表达式引擎 | [expr](https://github.com/expr-lang/expr) v1.17.8 |
| MQTT | Eclipse Paho MQTT |
| Excel 导出 | excelize |

---

## 📂 模块导航

### 1️⃣ IO 层 — 工业协议驱动与数据采集

[📖 IO/README.md](./IO/README.md)

IO 层负责工业设备的协议通信、数据采集和点位管理。采用**驱动插件化**架构，每个协议驱动独立实现，由 manager 统一调度。

**子模块**：
- **[Modbus_Tcp](./IO/Modbus_Tcp/README.md)** — Modbus TCP 协议驱动（支持 FC01/FC02/FC03/FC04）
- **[Siemens_S7](./IO/Siemens_S7/README.md)** — 西门子 S7 协议驱动（支持 S7-200/300/400/1200/1500）
- **[manager](./IO/manager)** — 驱动管理器（统一调度入口，按驱动粒度并发控制）
- **[byte_util](./IO/byte_util)** — 字节序转换工具库（支持全部字节序排列）
- **[tcp_udp](./IO/tcp_udp)** — 通用 TCP/UDP 网络连接层

**架构特点**：
- ✅ 驱动插件化，易于扩展新协议
- ✅ 自动组包优化，减少网络请求次数
- ✅ 按驱动粒度的并发锁控制
- ✅ 完善的错误隔离机制

---

### 2️⃣ db_point — 实时数据与配置管理

[📖 db/db_point/README.md](./db/db_point/README.md)

本模块是实时数据核心层，负责 IO 采集数据的分发、值变化检测、报警判断和历史记录。

**文件说明**：
- `db_point.go` — 实时值内存存储、值变化检测、IO 采集数据分发（发布/订阅）
- `alarm.go` — 报警配置管理（三级缓存）、报警业务判断、报警事件发布/订阅
- `history.go` — 历史记录配置管理（内存缓存）、历史数据业务判断
- `write.go` — 点位写入回调注册与聚合下发

**核心特性**：
- 🚀 **三级缓存策略**：freecache → MySQL → 内存 map
- ⚡ **发布/订阅模式**：解耦数据采集与业务逻辑
- 🔔 **动态报警规则**：使用 expr 表达式引擎，支持复杂条件判断
- 📊 **智能历史记录**：仅在配置启用时记录，减少存储压力

---

### 3️⃣ Web API — HTTP 接口服务

[📖 web/README.md](./web/README.md)

基于 Gin 框架提供的 HTTP API 服务，支持点位实时操作、配置管理和 WebSocket 推送。

**API 分类**：

#### 点位实时操作
- `POST /api/v1.0/point/read/value` — 读取点位实时值
- `POST /api/v1.0/point/write/value` — 写入点位控制值
- `POST /api/v1.0/alarm/status` — 查询报警状态
- `POST /api/v1.0/history/data/query` — 查询历史数据（InfluxDB）

#### 配置管理
- `/api/v1.0/config/drive/*` — 驱动配置 CRUD
- `/api/v1.0/config/point/*` — 点位配置 CRUD
- `/api/v1.0/config/alarm/*` — 报警配置 CRUD
- `/api/v1.0/config/history/*` — 历史配置 CRUD

#### 配置导出
- `GET /api/v1.0/config/export` — 导出配置为 Excel 文件（多 Sheet）

**性能优化**：
- ✅ 值变化检测，仅推送实际变化的数据
- ✅ 批量数据处理，减少序列化开销
- ✅ WebSocket 连接采用优雅关闭和重连机制
- ✅ 平均 API 响应时间 < 100ms

---

### 4️⃣ api_get_config — 远程配置同步

[📖 app/api_get_config/README.md](./app/api_get_config/README.md)

从 Iot-Config-Service 拉取配置并同步到本地 MySQL，实现配置的集中管理和远程更新。

**核心流程**：
```
main.go 启动
│
├─ syncConfigOnStartup()
│   ├─ IsConfigUpdated() — 检查配置是否有更新
│   │   └─ 有更新 → SyncConfigFromService(configUpdateTime)
│   └─ app() — 启动 MQTT / Web / 数据点 / 驱动
│       └─ manager.Start()
│            └─ InitializeDrivers() — 从 MySQL 读取配置初始化驱动
```

**文件说明**：
- `ApiConfig.go` — 配置服务 API 封装（认证、查询接口、401 自动重试）
- `sync.go` — 同步逻辑（检查更新 + 拉取配置 + 写入 MySQL）

**特性**：
- 🔐 BasicAuth 认证，支持 token 自动刷新
- 🔄 增量同步，仅更新变化的配置
- 📝 同步时间记录，避免重复拉取

---

### 5️⃣ InfluxDB — 时序数据存储

[📖 db/influxdb/influxdb.go](./db/influxdb/influxdb.go)

InfluxDB v2 集成模块，提供历史数据的批量写入和查询功能。

**核心特性**：
- 📦 **批量写入缓冲区**：bufferSize 阈值触发 + flushInterval 定时刷新
- 🔒 **防重复写入**：sync.Map 缓存去重，定期清理过期键
- 🎯 **类型分字段存储**：value_bool/value_int/value_uint/value_float/value_string
- ⏱️ **毫秒级精度**：时间戳精确到毫秒，同一毫秒内字段自动合并
- 📊 **Flux 查询语言**：高效过滤 measurement 和 tag，支持分页查询

**配置参数**：
```yaml
Influxdb:
  enable: true
  url: "http://127.0.0.1:8086"
  token: "..."
  org: "group1"
  bucket: "my"
  write_timeout: 5000        # 写入超时（毫秒）
  buffer_size: 100           # 缓冲区大小阈值
  flush_interval: 2s         # 刷新间隔
```

---

### 6️⃣ 其他模块

#### db/mysql — MySQL 数据库层
- `mysql.go` — 数据库连接管理
- `business.go` — 业务数据表 CRUD 操作
- `CheckSqlStructure.go` — 表结构自动检查和创建

#### db/redis — Redis 缓存与队列
- `redis.go` — Redis 连接管理
- `queue.go` — 消息队列操作
- `logic.go` — 业务逻辑缓存

#### db/tsdb — TDengine 时序数据库（可选）
- `tsdb.go` — TDengine 连接和写入（备用方案）

#### Init — 初始化模块
- `config.go` — YAML 配置文件解析
- `log.go` — 日志系统初始化

#### cloud — 云端通信
- `server.go` — 云端服务器连接
- `encryption.go` — 数据加密传输

#### sharing — 共享数据结构
- `sharing.go` — 全局共享变量和数据结构

---

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Iot-Collector-Service                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │  Web API     │    │  MQTT Base   │    │ Config Sync  │   │
│  │  (Gin)       │    │  (Paho)      │    │ (resty)      │   │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘   │
│         │                   │                    │           │
│         └───────────────────┼────────────────────┘           │
│                             ▼                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              db_point (实时数据核心)                   │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ Value    │  │ Alarm    │  │ History  │           │   │
│  │  │ Manager  │  │ Engine   │  │ Recorder │           │   │
│  │  └──────────┘  └──────────┘  └──────────┘           │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼───────────────────────────────┐   │
│  │                  IO Layer (驱动层)                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ Modbus   │  │ S7       │  │ 更多...  │           │   │
│  │  │ TCP      │  │ Protocol │  │          │           │   │
│  │  └──────────┘  └──────────┘  └──────────┘           │   │
│  └──────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼───────────────────────────────┐   │
│  │              数据存储层                               │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │   │
│  │  │ MySQL    │  │ Redis    │  │ InfluxDB │           │   │
│  │  │ (配置)   │  │ (缓存)   │  │ (时序)   │           │   │
│  │  └──────────┘  └──────────┘  └──────────┘           │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

### 1. 配置文件

编辑 `config.yaml`：

```yaml
APP:
  Label: "collector-01"        # 采集器标识
  SN: "SN001"                  # 序列号

MySQL:
  Host: "127.0.0.1"
  Port: 3306
  User: "root"
  Password: "password"
  Database: "iot_collector"

Redis:
  Addr: "127.0.0.1:6379"
  Password: ""
  DB: 0

Influxdb:
  enable: true
  url: "http://127.0.0.1:8086"
  token: "your-token"
  org: "group1"
  bucket: "my"
  buffer_size: 100
  flush_interval: 2s

API:
  Enable: true
  Ip: "0.0.0.0"
  Post: 8080

MQTT:
  Enable: true
  Broker: "tcp://127.0.0.1:1883"
  ClientId: "collector-01"
```

### 2. 编译运行

```bash
# 安装依赖
go mod download

# 编译
go build -o collector-service main.go

# 运行
./collector-service
```

### 3. 验证服务

```bash
# 检查 API 是否启动
curl http://localhost:8080/api/v1.0/point/read/value \
  -H "Content-Type: application/json" \
  -d '[{"PointId": 1}]'
```

---

## 📊 数据流说明

### 数据采集流程

```
工业设备 (PLC/传感器)
    │
    ▼
IO 驱动 (Modbus/S7)
    │  轮询采集
    ▼
Collection_Publisher (发布所有原始数据)
    │
    ├──▶ History_Judgment_list ──▶ History_Publisher ──▶ InfluxDB
    │
    └──▶ Update_Value_Judgment_list (值变化检测)
            │
            ▼
        Update_Publisher (仅发布变化数据)
            │
            └──▶ Alarm_Judgment_list ──▶ Alarm_Publisher ──▶ 报警处理
```

### 配置同步流程

```
Iot-Config-Service (配置中心)
    │
    ▼  REST API (BasicAuth)
api_get_config 模块
    │
    ▼  增量比对
MySQL (本地配置库)
    │
    ▼  读取配置
manager.Start()
    │
    ▼  初始化驱动
IO 驱动实例
```

---

## 🔧 开发指南

### 添加新协议驱动

1. 在 `IO/` 目录下创建新文件夹，如 `IO/NewProtocol/`
2. 实现三个核心函数：
   - `New()` — 解析配置，构建组包
   - `Connect()` — 建立连接，启动轮询
   - `Close()` — 关闭连接，上报状态
3. 在 `IO/manager/new.go` 中注册驱动类型
4. 编写 `README.md` 说明配置格式和使用方法

参考：[Modbus_Tcp/README.md](./IO/Modbus_Tcp/README.md)、[Siemens_S7/README.md](./IO/Siemens_S7/README.md)

### 添加新的 Web API

1. 在 `web/` 目录下创建或修改 handler 文件
2. 实现 handler 函数，使用 `ctx.Set("Response", ...)` 返回统一格式
3. 在 `web/xxx.go` 的 `register()` 函数中注册路由
4. 更新 [web/README.md](./web/README.md) 文档

示例：
```go
func api_example(ctx *gin.Context) {
    ctx.Set("Response", []any{200, "ok", data})
}

func register(r *gin.Engine) {
    r.POST("/api/v1.0/example", api_example)
}
```

### 自定义报警规则

报警配置使用 [expr](https://github.com/expr-lang/expr) 表达式引擎，`x` 代表点位值：

| 表达式 | 说明 |
|--------|------|
| `==0` | 值为 0 时报警 |
| `>100` | 值超过 100 时报警 |
| `<5 || >95` | 值小于 5 或大于 95 时报警 |
| `in[1,2,3]` | 值在 [1,2,3] 范围内时报警 |

---

## 📝 文档索引

| 文档 | 说明 |
|------|------|
| [IO/README.md](./IO/README.md) | IO 层总览与驱动列表 |
| [IO/Modbus_Tcp/README.md](./IO/Modbus_Tcp/README.md) | Modbus TCP 驱动详解 |
| [IO/Siemens_S7/README.md](./IO/Siemens_S7/README.md) | Siemens S7 驱动详解 |
| [db/db_point/README.md](./db/db_point/README.md) | 实时数据与报警管理 |
| [web/README.md](./web/README.md) | Web API 接口文档 |
| [app/api_get_config/README.md](./app/api_get_config/README.md) | 远程配置同步 |

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 代码规范
- 遵循 Go 官方代码风格
- 所有公共函数必须有注释
- 新增功能需配套单元测试
- 更新相关文档

### 提交规范
```
feat: 添加新功能
fix: 修复 bug
docs: 文档更新
refactor: 代码重构
test: 测试用例
chore: 构建/工具链变更
```

---

## 📄 许可证

本项目采用 MIT 许可证。详见根目录 [LICENSE](../../LICENSE) 文件。

### 第三方依赖声明

| 依赖 | 许可证 | 用途 |
|------|--------|------|
| [gin-gonic/gin](https://github.com/gin-gonic/gin) | MIT | Web 框架 |
| [eclipse/paho.mqtt.golang](https://github.com/eclipse/paho.mqtt.golang) | EPL-2.0 | MQTT 客户端 |
| [influxdata/influxdb-client-go](https://github.com/influxdata/influxdb-client-go) | MIT | InfluxDB 客户端 |
| [expr-lang/expr](https://github.com/expr-lang/expr) | MIT | 表达式引擎 |
| [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | MPL-2.0 | MySQL 驱动 |
| [go-redis/redis](https://github.com/go-redis/redis) | BSD-2-Clause | Redis 客户端 |
| [goburum/gos7](https://github.com/goburum/gos7) | BSD-3-Clause | S7 协议库 |
| [grid-x/modbus](https://github.com/grid-x/modbus) | MIT | Modbus 协议库 |

---

## 📞 联系方式

- 项目主页：[GoEdge-IoT](https://github.com/GoEdge-IoT)
- 问题反馈：提交 GitHub Issue
- 技术支持：查看各模块 README 文档

---

**最后更新时间**：2026-09-23
