# api_get_config

从 Iot-Config-Service 拉取配置并同步到本地 MySQL。

## 调用入口

```
main.go → syncConfigOnStartup()       // 启动前检查并同步配置
  ├─ IsConfigUpdated()                // Enable=false → 跳过
  │   └─ 有更新 → SyncConfigFromService(configUpdateTime)
  └─ app()                            // 启动 MQTT / Web / 数据点 / 驱动
       └─ manager.Start()
            └─ InitializeDrivers()    // 直接从 MySQL 读取配置 → 初始化驱动
```

## 文件说明

| 文件 | 职责 |
|------|------|
| `ApiConfig.go` | 配置服务 API 封装（认证、查询接口、401 自动重试） |
| `sync.go` | 同步逻辑（检查更新 + 拉取配置 + 写入 MySQL） |

## 核心函数

### sync.go

| 函数 | 说明 |
|------|------|
| `IsConfigUpdated() (time.Time, error)` | 对比服务端更新时间与 MySQL 记录，返回非零值表示有更新 |
| `SyncConfigFromService(configUpdateTime)` | 拉取驱动/点位配置 → 调用 `__Sync` 按 Id 比对增删改 → 记录同步时间 |
| `getLastConfigUpdateTime() (time.Time, error)` | 从 `APP_Config` 表读取上次同步时间（key=`ConfigUpdateTime`） |
| `setLastConfigUpdateTime(t)` | 将同步时间写入 `APP_Config` 表 |

### ApiConfig.go

| 函数 | 说明 |
|------|------|
| `Collector_GetBasicAuth() (string, error)` | 认证，获取 access token |
| `Collector_ConfigUpdate() (time.Time, error)` | 查询配置最新更新时间（401 自动刷新 token 重试） |
| `Collector_Drive_Config__Query()` | 获取驱动配置列表 |
| `Collector_Point_Config__Query()` | 获取点位配置列表 |

## 同步流程

```
main.go 启动
│
├─ syncConfigOnStartup()
│   ├─ IsConfigUpdated()
│   │   ├─ Enable=false → 返回零值，跳过
│   │   ├─ Collector_ConfigUpdate() → 获取服务端更新时间
│   │   ├─ getLastConfigUpdateTime() → 从 MySQL APP_Config 读取上次时间
│   │   └─ 比对 → 未更新返回零值，有更新返回 configUpdateTime
│   └─ configUpdateTime 非零 → SyncConfigFromService(configUpdateTime)
│       ├─ Collector_Drive_Config__Query() → 获取全量驱动配置
│       ├─ Collector_Point_Config__Query() → 获取全量点位配置
│       ├─ mysql.Drive_Config__Sync() → 按 Id 比对增删改
│       ├─ mysql.Point_Config__Sync() → 按 Id 比对增删改
│       └─ setLastConfigUpdateTime() → 写入 MySQL
│
└─ app() → MQTT / Web / 数据点 / 驱动初始化（从 MySQL 读取）
```

## 认证机制

- 所有业务请求自动带 `Collector_Token` 头
- 遇到 401 响应时自动调用 `Collector_GetBasicAuth()` 刷新 token 并重试
- 无需在调用方手动认证

## 依赖

- `main/Init` — 读取 `Config_Service` 配置（Enable、Address、超时、重试）
- `main/db/mysql` — 配置持久化存储（`APP_Config` 表 + 各配置表 CRUD）
