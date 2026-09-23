# db_point — 实时数据与配置管理

本模块是 Iot-Collector-Service 的实时数据核心层，负责 IO 采集数据的分发、值变化检测、报警判断和历史记录。

## 文件说明

| 文件 | 职责 |
|------|------|
| `db_point.go` | 实时值内存存储、值变化检测、IO 采集数据分发（发布/订阅） |
| `alarm.go` | 报警配置管理（三级缓存）、报警业务判断、报警事件发布/订阅 |
| `history.go` | 历史记录配置管理（内存缓存）、历史数据业务判断、记录事件发布/订阅 |
| `write.go` | 点位写入回调注册与聚合下发 |

## 数据流架构

```
IO 驱动采集
    │
    ▼
Collection_Publisher ──── 采集数据发布（所有原始数据）
    │                       │
    │                       ├──▶ History_Judgment_list ──▶ History_Publisher ──▶ 历史存储
    │                       │
    │                       └──▶ Update_Value_Judgment_list ──▶ 值变化检测
    │                                                               │
    │                                                               ▼
    │                                                   Update_Publisher ──── 变化数据发布
    │                                                               │
    │                                                               └──▶ Alarm_Judgment_list ──▶ Alarm_Publisher ──▶ 报警处理
    │
    ▼
Write_value_Publisher ──── 写入回调聚合下发
```

### 订阅注册时机

| 订阅函数 | 注册方式 | 触发条件 |
|----------|----------|----------|
| `Update_Value_Judgment_list` | `init()` 自动注册到 `Collection_Subscriber` | 每次 IO 采集数据到达 |
| `History_Judgment_list` | `init()` 自动注册到 `Collection_Subscriber` | 每次 IO 采集数据到达 |
| `Alarm_Judgment_list` | `init()` 自动注册到 `Update_Subscriber` | 仅在点位值**发生变化**时触发 |

> **关键区别：** 历史记录订阅的是 `Collection`（所有原始数据），报警判断订阅的是 `Update`（仅变化数据）。

## 报警配置（alarm.go）

### MySQL 表结构

表名：`Alarm_Config`

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `Id` | int unsigned | PK, AI, UQ, IDX | 报警配置 id（自增） |
| `Point_Id` | int unsigned | NOT NULL, IDX | 关联点位 id |
| `Name` | varchar(100) | NOT NULL | 报警名称 |
| `Config` | varchar(200) | NOT NULL | 报警条件配置 |
| `Group` | int unsigned | NOT NULL | 报警组号 |
| `Creation_Time` | datetime | NOT NULL | 创建时间 |

### 内存数据结构

```go
type Alarm_Config_type struct {
    AlarmId uint      // 报警配置 id（对应 MySQL Id）
    PointId uint      // 点位 id
    Config  string    // 报警条件配置
    Group   int       // 报警组号
    Status  string    // 运行时状态（开始/恢复/触发）
    Time    time.Time // 最后一次报警时间
}
```

### 三级缓存读取策略

`Alarm_Config__Query` 采用三级缓存，按优先级依次查找：

```
1. freecache（100MB LRU 缓存，key 格式: "alarm:{PointId}"）
       │ 命中 → 返回
       │ 未命中 ↓
2. MySQL（按 Point_Id 查询）
       │ 命中 → 写入 freecache + 同步内存 map → 返回
       │ 未命中 ↓
3. 内存 map（Alarm_Config，sync.RWMutex 保护）
       │ 命中 → 返回
       │ 未命中 → 返回 false
```

> **TTL 配置：** freecache 的 TTL 由 `Init.Config.ALARM.Config_CacheTTL` 控制，设为 0 表示永久保存不过期。

### 报警条件配置格式

`Config` 字段支持以下三种条件：

| 配置值 | 含义 | 判断逻辑 | 触发状态 |
|--------|------|----------|----------|
| `==0` | 值等于 false（0） | 值为 false → 报警开始；值为 true → 报警恢复 | 开始 / 恢复 |
| `==1` | 值等于 true（非 0） | 值为 true → 报警开始；值为 false → 报警恢复 | 开始 / 恢复 |
| `0<>1` | 值在 0 和 1 之间变化 | 任何值变化都触发 | 触发 |

### 报警状态说明

| 状态 | 含义 |
|------|------|
| `开始` | 报警条件满足，报警触发 |
| `恢复` | 报警条件不再满足，报警恢复正常 |
| `触发` | 值发生变化即触发（用于 `0<>1` 模式） |

### 核心函数

| 函数 | 说明 |
|------|------|
| `Alarm_Config__Query(key)` | 三级缓存查询单个点位的报警配置 |
| `Alarm_Config__Query_list(keys)` | 批量查询多个点位的报警配置 |
| `Alarm_Config__Add(configs)` | 新增配置，按 key 分组追加到内存 map 并同步 freecache |
| `Alarm_Config__Status_Update(key, configStr, status, t)` | 按 Config 字符串匹配更新报警状态，同步 freecache |
| `alarmJudgmentSingle(new, cfg)` | 单条报警配置的业务判断 |
| `Alarm_Judgment(new)` | 遍历点位所有报警配置，返回所有触发的报警 |
| `Alarm_Judgment_list(new_list)` | 批量报警判断，汇总后通过 `Alarm_Publisher` 发布 |
| `Alarm_Publisher(v)` / `Alarm_Subscriber(fn)` | 报警事件发布/订阅 |

### 报警判断流程

```
Alarm_Judgment_list(采集数据列表)
    │
    ├── 遍历每个点位值
    │   └── Alarm_Judgment(单点位值)
    │       ├── 校验 PointId != 0
    │       ├── Alarm_Config__Query(key) → 获取该点位所有报警配置
    │       └── 遍历每条配置
    │           ├── alarmJudgmentSingle() → 类型转换 + 条件判断
    │           ├── 命中 → Alarm_Config__Status_Update() 更新状态
    │           └── 汇总所有触发的报警
    │
    └── Alarm_Publisher(所有报警事件) → 通知下游订阅者
```

## 历史配置（history.go）

### MySQL 表结构

表名：`History_Config`

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `Id` | int unsigned | PK, AI, UQ, IDX | 历史配置 id（自增） |
| `Point_Id` | int unsigned | NOT NULL, IDX | 关联点位 id |
| `Config` | varchar(200) | NOT NULL | 历史配置（非空且非 "null" 时启用记录） |
| `Creation_Time` | datetime | NOT NULL | 创建时间 |

### 内存数据结构

```go
// 历史值记录
type History_Value_type struct {
    PointId uint      // 点位 id
    Time    time.Time // 记录时间
    Msg     string    // 状态信息
    Value   any       // 记录值
    Type    string    // 值类型
}

// 历史配置
type History_Config_type struct {
    PointId uint   // 点位 id
    Config  string // 配置（非空且非 "null" 时启用记录）
}
```

### 缓存策略

历史配置仅使用**内存 map**（`sync.RWMutex` 保护），不依赖 freecache 或 MySQL 实时查询：

```
History_Config__Query(key)
    │
    └── 内存 map 查找（History_Config）→ 命中返回 / 未命中返回 false
```

### 核心函数

| 函数 | 说明 |
|------|------|
| `History_Config__Query(key)` | 内存 map 查询单个点位的历史配置 |
| `History_Config__Query_list(keys)` | 批量查询多个点位的历史配置 |
| `History_Config__Add(configs)` | 批量新增配置，一次性加锁写入内存 map |
| `History_Judgment(new)` | 单点位历史判断：校验配置有效 → 类型转换 → 构造记录值 |
| `History_Judgment_list(new_list)` | 批量历史判断，汇总后通过 `History_Publisher` 发布 |
| `History_Publisher(v)` / `History_Subscriber(fn)` | 历史记录事件发布/订阅 |

### 历史记录判断流程

```
History_Judgment_list(采集数据列表)
    │
    ├── 遍历每个点位值
    │   └── History_Judgment(单点位值)
    │       ├── 校验 PointId != 0
    │       ├── History_Config__Query(key) → 获取历史配置
    │       ├── 校验 Config 非空且非 "null"
    │       ├── byte_util.ConvBool() → 类型转换
    │       └── 构造 History_Value_type 返回
    │
    └── History_Publisher(所有历史记录) → 通知下游订阅者
```

## 报警与历史配置对比

| 特性 | 报警配置（alarm.go） | 历史配置（history.go） |
|------|---------------------|----------------------|
| MySQL 表 | `Alarm_Config` | `History_Config` |
| 一个点位可有多条配置 | 是（map value 为切片） | 否（map value 为单条） |
| 缓存层级 | 三级：freecache → MySQL → 内存 map | 单级：仅内存 map |
| 订阅数据源 | `Update`（值变化后触发） | `Collection`（所有原始数据） |
| 条件判断 | 支持 `==0` / `==1` / `0<>1` 三种模式 | 仅判断配置是否有效（非空非 "null"） |
| 运行时状态 | 有（Status 字段：开始/恢复/触发） | 无 |
| 发布事件类型 | `Alarm_type` | `History_Value_type` |

## 公共数据结构

### 点位 ID

所有配置 map 统一使用 `uint`（点位 ID）作为 key：

```go
map[uint]XXX_Config_type
```

### 发布/订阅模式

本模块实现了三套独立的发布/订阅通道：

| 通道 | 数据类型 | 用途 |
|------|----------|------|
| Collection | `[]fullConfig.Value_type` | IO 采集原始数据分发 |
| Update | `[]fullConfig.Value_type` | 值变化数据分发 |
| Alarm | `[]Alarm_type` | 报警事件分发 |
| History | `[]History_Value_type` | 历史记录分发 |

每套通道均提供 `Publisher` / `Subscriber` 函数对，支持多个订阅者。

## 依赖

- `main/IO/manager/fullConfig` — 采集值结构体 `Value_type`
- `main/IO/byte_util` — 值类型转换 `ConvBool`
- `main/db/mysql` — 报警/历史配置持久化 CRUD
- `main/Init` — 读取报警缓存 TTL 等运行配置
- `github.com/coocood/freecache` — 报警配置 LRU 缓存
