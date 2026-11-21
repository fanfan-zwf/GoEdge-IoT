# Iot-Collector-Service 标准 Go 项目布局重构

## 一、当前结构问题

| 问题 | 说明 |
|------|------|
| 无分层 | 所有代码在根目录，`IO/`、`db/`、`web/`、`app/` 扁平并列 |
| 巨型文件 | `db/mysql/business.go` 2221 行，混合了数据模型定义 + CRUD 逻辑 |
| 包命名不规范 | `Modbus_Tcp`、`Siemens_S7`、`db_point`、`api_get_config` 含下划线/大写 |
| 死代码 | `cloud/`（全注释）、`sharing/`（空文件） |
| 入口不独立 | `main.go` 在根目录，`db/tsdb/tsdb.go` 含 `main()` 函数 |
| module 名不规范 | `go.mod` 中 module 名为 `main` |

## 二、目标结构

```
Iot-Collector-Service/
├── cmd/
│   └── server/
│       └── main.go                      # 入口（原根目录 main.go）
│
├── internal/
│   ├── config/                          # 配置加载 + 日志初始化（原 Init/）
│   │   ├── config.go                    # Config_type 结构体 + yaml 加载
│   │   └── log.go                       # 日志初始化
│   │
│   ├── handler/                         # HTTP 传输层（原 web/）
│   │   ├── server.go                    # Gin 引擎启动、CORS、信号处理
│   │   ├── middleware.go                # Response_Use 统一响应 + allowAllCors
│   │   ├── router.go                    # 路由注册（原 api.go 的 gui_api）
│   │   ├── point.go                     # 点位读写 + WebSocket API
│   │   ├── config.go                    # 配置管理 CRUD API
│   │   ├── export.go                    # Excel 导入/导出 API
│   │   └── util.go                      # create_token、parseXLSXFromStream
│   │
│   ├── service/                         # 核心业务逻辑
│   │   ├── collector/                   # 数据采集引擎（原 db/db_point/）
│   │   │   ├── bus.go                   # 泛型 pub/sub 总线 + IO 采集/更新总线
│   │   │   ├── valuemap.go              # 实时值内存存储 + 值变化检测
│   │   │   ├── write.go                 # 写入回调注册与聚合下发
│   │   │   ├── alarm.go                 # 报警判断 + freecache 缓存
│   │   │   └── history.go              # 历史记录判断
│   │   │
│   │   ├── driver/                      # 驱动生命周期管理（原 IO/manager/）
│   │   │   ├── manager.go              # DriverManager + 锁 + CRUD
│   │   │   └── init.go                 # InitializeDrivers + Start
│   │   │
│   │   └── configsync/                  # 配置同步（原 app/api_get_config/）
│   │       ├── check.go                # IsConfigUpdated
│   │       └── sync.go                 # SyncConfigFromService
│   │
│   ├── model/                           # 数据模型（从 business.go 拆出）
│   │   ├── point.go                     # Point_Config 系列结构体 + 正则常量
│   │   ├── drive.go                     # Drive_Config 系列结构体
│   │   ├── alarm.go                     # Alarm_Config/Data 系列结构体
│   │   ├── history.go                   # History_Config 系列结构体
│   │   ├── app.go                       # APP_Config 系列结构体
│   │   ├── collector.go                 # Collector_Config 系列结构体
│   │   └── value.go                     # Value_type + FullConfig_type + Tag 工具函数（原 fullConfig.go）
│   │
│   ├── repo/                            # 数据访问层
│   │   ├── mysql/                       # MySQL 实现（原 db/mysql/）
│   │   │   ├── mysql.go                # 连接初始化 + DB 全局变量
│   │   │   ├── schema.go              # CheckSqlStructure（原 CheckSqlStructure.go）
│   │   │   ├── drive.go               # Drive_Config CRUD（从 business.go 拆出）
│   │   │   ├── point.go               # Point_Config CRUD
│   │   │   ├── alarm.go               # Alarm_Config/Data CRUD
│   │   │   ├── history.go             # History_Config CRUD
│   │   │   ├── app.go                 # APP_Config CRUD
│   │   │   └── collector.go           # Collector_Config CRUD
│   │   │
│   │   ├── redis/                       # Redis（原 db/redis/）
│   │   │   ├── redis.go               # 连接初始化
│   │   │   ├── logic.go              # 业务逻辑
│   │   │   └── queue.go              # 队列
│   │   │
│   │   ├── influxdb/                    # InfluxDB（原 db/influxdb/）
│   │   │   └── influxdb.go
│   │   │
│   │   └── tsdb/                        # Badger KV（原 db/tsdb/，需移除 main()）
│   │       └── tsdb.go
│   │
│   └── driver/                          # 工业协议驱动实现（原 IO/ 下协议目录）
│       ├── modbus/                      # Modbus TCP（原 IO/Modbus_Tcp/）
│       │   ├── modbus.go              # 主逻辑（原 Modbus_Tcp.go）
│       │   ├── packet.go              # 组包/解包（原 Packet.go）
│       │   └── new.go
│       │
│       └── siemens/                     # Siemens S7（原 IO/Siemens_S7/）
│           ├── siemens.go             # 主逻辑（原 Siemens_S7.go）
│           ├── packet.go              # 组包/解包
│           ├── new.go
│           └── snap7_cgo.go           # CGO 绑定
│
├── pkg/                                 # 可复用工具包
│   ├── conv/                            # 类型转换（原 IO/conv/）
│   │   ├── byte_util.go
│   │   ├── calculate.go
│   │   └── convertType.go
│   │
│   └── netutil/                         # 网络工具（原 IO/tcp_udp/）
│       └── tcp_or_udp.go
│
├── third_party/                         # 第三方库
│   └── lib/
│       ├── libsnap7.so
│       ├── snap7.h
│       └── snap7_api_cn.md
│
├── configs/
│   └── config.yaml                      # 配置文件（原根目录 config.yaml）
├── logs/                                # 日志（保持不变）
├── go.mod                               # module 名改为 collector
├── go.sum
└── Makefile                             # 构建脚本（新增）
```

## 三、关键变更详解

### 3.1 go.mod 模块名变更

```
module collector    # 原: module main
```

所有 import 路径从 `main/xxx` 改为 `collector/internal/xxx` 或 `collector/pkg/xxx`。

### 3.2 business.go 拆分（2221 行 → model + repo）

**拆出到 `internal/model/` 的内容**（纯结构体定义 + 常量）：
- `Collector__Carry_type`, `Collector_Config_*_type` → `model/collector.go`
- `Drive_Config_*_type` → `model/drive.go`
- `Point_Config_*_type` + `RW_Cancel_map` + 3 个正则常量 → `model/point.go`
- `Alarm_Config_*_type`, `Alarm_Data_*_type` → `model/alarm.go`
- `History_Config_*_type` → `model/history.go`
- `APP_Config_*_type` → `model/app.go`

**保留在 `internal/repo/mysql/` 的内容**（CRUD 函数）：
- 各 `__Query`, `__Add`, `__Update`, `__Del`, `__Sync`, `__Search`, `__Count` 函数
- 正则常量移到 model 后，repo 文件 import model 包使用

### 3.3 db_point → service/collector 文件映射

| 原文件 | 目标文件 | 说明 |
|--------|----------|------|
| `db_point.go` | `bus.go` + `valuemap.go` | 拆分总线/值存储 |
| `alarm.go` | `alarm.go` | 保持 |
| `history.go` | `history.go` | 保持 |
| `write.go` | `write.go` | 保持 |

### 3.4 包命名规范化

| 原包名 | 新包名 | 目录 |
|--------|--------|------|
| `Init` | `config` | `internal/config/` |
| `web` | `handler` | `internal/handler/` |
| `db_point` | `collector` | `internal/service/collector/` |
| `manager` | `driver` | `internal/service/driver/` |
| `fullConfig` | `model` | `internal/model/`（合并） |
| `api_get_config` | `configsync` | `internal/service/configsync/` |
| `Modbus_Tcp` | `modbus` | `internal/driver/modbus/` |
| `Siemens_S7` | `siemens` | `internal/driver/siemens/` |
| `tcp_udp` | `netutil` | `pkg/netutil/` |
| `conv` | `conv` | `pkg/conv/`（不变） |

### 3.5 类型名规范化（去下划线）

包名已提供命名空间，类型名去掉冗余前缀：

| 原类型 | 新类型 | 所在包 |
|--------|--------|--------|
| `Modbus_Tcp.Modbus_Tcp` | `modbus.Modbus` | `internal/driver/modbus` |
| `Siemens_S7.Siemens_S7` | `siemens.Siemens` | `internal/driver/siemens` |
| `fullConfig.Value_type` | `model.ValueType` | `internal/model` |
| `fullConfig.FullConfig_type` | `model.FullConfig` | `internal/model` |
| `mysql.Drive_Config_Query_type` | `model.DriveConfigQuery` | `internal/model` |
| `mysql.Point_Config_Query_type` | `model.PointConfigQuery` | `internal/model` |
| `db_point.Update_Value_type` | `collector.UpdateValue` | `internal/service/collector` |

> 注意：类型名规范化涉及面非常广（所有文件都引用了这些类型），可选择 **仅改包名不改类型名** 来降低风险。建议第一版只改包名和目录结构，类型名保持原样（如 `model.Drive_Config_Query_type`），后续再逐步规范化。

### 3.6 清理

| 操作 | 目标 |
|------|------|
| 删除 | `cloud/`（全注释代码，无实际功能） |
| 删除 | `sharing/`（空文件，仅 package 声明） |
| 处理 | `db/tsdb/tsdb.go` 中的 `main()` 函数需移除或改为测试 |
| 删除 | 根目录 `collector-service` 二进制文件 |

### 3.7 新增 Makefile

```makefile
.PHONY: build run clean

build:
	go build -o bin/collector-service ./cmd/server

run:
	go run ./cmd/server

clean:
	rm -rf bin/
```

## 四、Import 路径变更速查

| 原路径 | 新路径 |
|--------|--------|
| `main/Init` | `collector/internal/config` |
| `main/web` | `collector/internal/handler` |
| `main/db/mysql` | `collector/internal/repo/mysql` |
| `main/db/redis` | `collector/internal/repo/redis` |
| `main/db/influxdb` | `collector/internal/repo/influxdb` |
| `main/db/db_point` | `collector/internal/service/collector` |
| `main/IO/manager` | `collector/internal/service/driver` |
| `main/IO/manager/fullConfig` | `collector/internal/model` |
| `main/IO/Modbus_Tcp` | `collector/internal/driver/modbus` |
| `main/IO/Siemens_S7` | `collector/internal/driver/siemens` |
| `main/IO/conv` | `collector/pkg/conv` |
| `main/IO/tcp_udp` | `collector/pkg/netutil` |
| `main/app/mqttbase` | `collector/internal/infra/mqtt`（或保持 `internal/mqtt`） |
| `main/app/api_get_config` | `collector/internal/service/configsync` |

## 五、执行步骤

按依赖顺序分步执行，每步完成后 `go build ./...` 验证：

1. **创建目录骨架** — 新建所有目标目录
2. **移动 model 层** — 拆分 `business.go`，移动 `fullConfig.go` → `model/`
3. **移动 repo 层** — `db/mysql/` → `repo/mysql/`（引用 model）
4. **移动 infra 层** — `db/redis/`、`db/influxdb/`、`db/tsdb/` → `repo/` 下
5. **移动 driver 层** — `IO/Modbus_Tcp/`、`IO/Siemens_S7/` → `driver/`
6. **移动 pkg** — `IO/conv/` → `pkg/conv/`，`IO/tcp_udp/` → `pkg/netutil/`
7. **移动 service 层** — `db/db_point/` → `service/collector/`，`IO/manager/` → `service/driver/`
8. **移动 handler 层** — `web/` → `handler/`
9. **移动入口** — `main.go` → `cmd/server/main.go`，`config.yaml` → `configs/`
10. **更新 go.mod** — module 名改为 `collector`
11. **全量更新 import** — 所有 `.go` 文件的 import 路径
12. **清理** — 删除 `cloud/`、`sharing/`、旧目录
13. **编译验证** — `go build ./...` 确保零错误
14. **新增 Makefile**

## 六、风险与注意事项

- **CGO 路径**：`siemens/snap7_cgo.go` 中 `#cgo LDFLAGS` 引用的 `lib/` 路径需更新为 `third_party/lib/`
- **config.yaml 加载路径**：`config.go` 中 `ioutil.ReadFile("config.yaml")` 需更新为 `configs/config.yaml`（或通过 `-config` flag 指定）
- **tsdb 的 main()**：`tsdb.go` 含 `func main()`，移到 `internal/infra/tsdb/` 后必须移除，否则编译冲突
- **init() 链**：当前 `mysql.go` 和 `redis.go` 通过 `init()` 自动初始化，重构时保持此行为或改为显式初始化
- **类型名是否同步规范化**：建议第一版仅改目录和包名，类型名暂不变，降低出错风险
