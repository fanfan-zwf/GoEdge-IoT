# Iot-Collector-Service Web 接口

基于 Gin 框架提供的 HTTP API 服务，支持：

- **点位实时操作**：写入控制值、读取实时值、报警状态查询
- **配置管理**：驱动/点位/报警/历史配置的增删改查
- **配置导出**：多表配置导出为 Excel 文件
- **WebSocket 实时推送**：支持点位实时值的 WebSocket 流式推送

## 接口稳定性与优化

### 性能优化
- **值变化检测**：仅推送实际发生变化的数据，减少不必要的网络传输
- **批量数据处理**：按客户端分组批量推送数据，减少序列化开销
- **内存管理**：使用安全上限防止数据点无限增长，优化内存使用
- **API 响应时间**：优化数据库查询，确保 API 响应在合理范围内（平均 < 100ms）
- **并发处理**：使用读写锁保护共享资源，支持高并发访问

### 稳定性保障
- **延迟计算**：提供准确的时间延迟计算，帮助监控数据新鲜度
- **连接管理**：WebSocket 连接采用优雅关闭和重连机制，确保连接稳定
- **数据一致性**：确保实时值与数据库状态的一致性
- **错误处理**：完善的错误处理和日志记录机制，便于故障排查
- **批量处理**：支持批量查询和更新操作，减少网络请求次数

### 实时数据推送优化
- **智能推送**：仅当数据实际变化时才推送，无变化时不推送
- **初始化策略**：页面加载时先通过 API 获取初始值，再连接 WebSocket 接收后续变化
- **持续更新**：曲线图每秒更新一次，即使没有新数据也会保持最新值显示
- **时间同步**：精确计算数据延迟，显示准确的延迟信息

## 服务配置

```yaml
API:
  Enable: true        # 是否启用 API 服务（false 则不启动）
  Ip: "0.0.0.0"      # 监听地址
  Post: 8080          # 监听端口
```

## 文件说明

| 文件 | 职责 |
|------|------|
| `web.go` | 服务启动、CORS 中间件、统一响应中间件 |
| `api.go` | 点位读写、报警状态查询接口 |
| `config.go` | 配置管理 CRUD 接口（驱动/点位/报警/历史） |
| `export.go` | 配置导出 Excel 接口（多 Sheet） |

## 统一响应格式

所有 JSON 接口返回统一结构：

```json
{
  "Code": 200,
  "Msg": "ok",
  "Data": ...,
  "Timestamp": "2026-08-29T12:00:00.000000000+08:00"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| Code | int | 状态码，200 表示成功 |
| Msg | string | 响应消息 |
| Data | any | 响应数据 |
| Timestamp | string | RFC3339Nano 格式时间戳 |

### 常见错误码

| Code | 说明 |
|------|------|
| 200 | 成功 |
| 401 | 未登录 |
| 404 | 资源不存在 / 写入失败 |
| 417 | 请求格式错误 / 参数缺失 |
| 500 | 服务端内部错误 |
| 501 | 响应中间件异常 |
| 520 | MySQL 错误 |
| 521 | Redis 错误 |
| 522 | 正则表达式计算错误 |

---

## 一、点位实时操作

### 1.1 点位实时值读取

查询指定点位的当前实时采集值（内存中的最新值）。

```
POST /api/v1.0/point/read/value
```

#### 请求体

```json
[
  { "PointId": 1 },
  { "PointId": 2 },
  { "PointId": 3 }
]
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| PointId | uint | 是 | 点位 ID |

#### 响应 Data 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| PointId | uint | 点位 ID |
| Value | any | 当前实时值 |
| Type | string | 值类型 |
| Msg | string | 状态信息（`ok` 表示正常） |
| Time | time | 当前值的采集时间 |

> 如果请求的点位尚未采集到数据（未注册），该点位不会出现在返回结果中。

---

### 1.2 点位写入

向指定点位写入控制值，驱动将根据配置将值下发到 PLC 设备。

```
POST /api/v1.0/point/write/value
```

#### 请求体

```json
[
  { "PointId": 100, "Value": 25.5, "Type": "float32" }
]
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| PointId | uint | 是 | 点位 ID |
| Value | any | 是 | 写入值，类型必须与 Type 匹配 |
| Type | string | 是 | 数据类型 |

#### 支持的 Type 值

| Type | 说明 |
|------|------|
| bool | 布尔 |
| int8 / uint8 | 8 位整数 |
| int16 / uint16 | 16 位整数 |
| int32 / uint32 | 32 位整数 |
| int64 / uint64 | 64 位整数 |
| int / uint | 平台相关整数 |
| float32 / float64 | 浮点数 |
| float | 等同于 float64 |
| string | 字符串 |

#### 写入保护机制

- **超时保护：** 写入值的时间戳超过 5 秒时拒绝写入
- **类型校验：** Value 必须与 Type 声明的类型一致
- **点位存在性：** 点位必须已在系统中注册

---

### 1.3 报警状态查询

查询指定点位的当前报警配置和状态。

```
POST /api/v1.0/alarm/status
```

#### 请求体

```json
[
  { "PointId": 100 }
]
```

#### 响应 Data 字段

| 字段 | 类型 | 说明 |
|------|------|------|
| PointId | uint | 点位 ID |
| Config | string | 报警规则配置（如 `==0`、`==1`） |
| Group | int | 报警组号 |
| Status | string | 当前状态：`开始` / `恢复` / `触发` |
| Time | time | 最后一次报警状态变更时间 |

---

### 1.4 历史数据查询

从 InfluxDB 时序数据库中查询指定点位在某个时间范围内的历史数据，支持分页。

```
POST /api/v1.0/history/data/query
```

#### 请求体

```json
{
  "pointId": 100,
  "startTime": "2026-09-23T10:00:00Z",
  "endTime": "2026-09-23T12:00:00Z",
  "page": 1,
  "pageSize": 50
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| pointId | uint | 是 | 点位 ID |
| startTime | string | 是 | 开始时间（RFC3339 格式） |
| endTime | string | 是 | 结束时间（RFC3339 格式） |
| page | uint | 否 | 页码，默认 1 |
| pageSize | uint | 否 | 每页数量，默认 20 |

#### 响应 Data 字段

```json
{
  "data": [
    {
      "Time": "2026-09-23T10:30:25.123Z",
      "Field": "value_int",
      "Value": 100,
      "Msg": "ok"
    },
    {
      "Time": "2026-09-23T10:30:26.456Z",
      "Field": "value_float",
      "Value": 25.5,
      "Msg": "ok"
    }
  ],
  "total": 150
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| data | array | 历史数据列表 |
| data[].Time | string | 数据采集时间（毫秒级精度） |
| data[].Field | string | 值类型字段名（value_bool/value_int/value_uint/value_float/value_string） |
| data[].Value | any | 实际值 |
| data[].Msg | string | 状态信息（绿色显示 ok，红色显示异常） |
| total | int | 符合条件的总记录数 |

#### 数据存储规则

InfluxDB 按数据类型分别存储到不同字段，避免类型冲突：

| Go 类型 | InfluxDB 字段 |
|---------|---------------|
| bool | value_bool |
| int/int8/int16/int32/int64 | value_int |
| uint/uint8/uint16/uint32/uint64 | value_uint |
| float32/float64 | value_float |
| string | value_string |

#### 特性

- **毫秒级精度**：时间戳精确到毫秒，同一毫秒内的多个字段自动合并
- **分页查询**：支持大数据量分页加载，减少网络传输
- **多类型支持**：自动识别并返回对应类型的字段值
- **状态显示**：前端根据 Msg 值显示绿色（正常）或红色（异常）
- **性能优化**：使用 Flux 查询语言，按 measurement 和 tag 高效过滤

---

## 二、配置管理

所有配置管理接口统一使用 `/api/v1.0/config/` 前缀，支持 4 种配置类型：

| 类型 | 路径前缀 |
|------|----------|
| 驱动配置 | `/api/v1.0/config/drive/` |
| 点位配置 | `/api/v1.0/config/point/` |
| 报警配置 | `/api/v1.0/config/alarm/` |
| 历史配置 | `/api/v1.0/config/history/` |

每种配置均支持 4 个操作：`query`、`add`、`update`、`del`。

### 2.1 驱动配置

#### 查询 `POST /api/v1.0/config/drive/query`

```json
// 请求
{ "page": 1, "pageSize": 20 }

// 响应 Data
[
  { "Id": 1, "Type": "Modbus_Tcp", "Name": "驱动1", "Config": "{...}", "Points_Length": 5 }
]
```

| 请求字段 | 类型 | 说明 |
|----------|------|------|
| page | uint | 页码（0 表示不分页） |
| pageSize | uint | 每页数量 |

#### 新增 `POST /api/v1.0/config/drive/add`

```json
[
  { "Type": "Modbus_Tcp", "Name": "驱动1", "Config": "{...}" }
]
```

#### 更新 `POST /api/v1.0/config/drive/update`

```json
[
  { "Id": 1, "Name": "驱动1-改名", "Config": "{...}" }
]
```

#### 删除 `POST /api/v1.0/config/drive/del`

```json
{ "ids": [1, 2, 3] }
```

---

### 2.2 点位配置

#### 查询 `POST /api/v1.0/config/point/query`

```json
{ "driveId": [1, 2], "page": 1, "pageSize": 20 }
```

| 请求字段 | 类型 | 说明 |
|----------|------|------|
| driveId | []uint | 按驱动 ID 过滤（空则不过滤） |
| page | uint | 页码 |
| pageSize | uint | 每页数量 |

#### 新增 `POST /api/v1.0/config/point/add`

```json
[
  { "Drive_Id": 1, "Name": "温度", "Description": "车间温度", "Config": "{...}", "RW_Cancel": 2, "Value_Type": 3 }
]
```

#### 更新 `POST /api/v1.0/config/point/update`

```json
[
  { "Id": 1, "Name": "温度-改名", "Config": "{...}", "RW_Cancel": 4 }
]
```

#### 删除 `POST /api/v1.0/config/point/del`

```json
{ "ids": [1, 2] }
```

---

### 2.3 报警配置

报警规则使用 **expr 表达式引擎**进行条件判断，支持复杂的逻辑运算。

#### 第三方依赖

本模块使用 [expr](https://github.com/expr-lang/expr) 库（v1.17.8）进行动态表达式求值。

**版权声明**：
```
MIT License

Copyright (c) 2018 Anton Medvedev

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

**中文概要**：
- expr 采用 MIT 许可证，允许自由使用、修改、分发和商业用途。
- 使用时需保留原始版权声明和许可声明。
- 软件按"原样"提供，作者不承担任何担保责任。

#### 查询 `POST /api/v1.0/config/alarm/query`

```json
{ "pointId": [100, 101], "page": 1, "pageSize": 20 }
```

| 请求字段 | 类型 | 说明 |
|----------|------|------|
| pointId | []uint | 按点位 ID 过滤（空则不过滤） |
| page | uint | 页码 |
| pageSize | uint | 每页数量 |

#### 新增 `POST /api/v1.0/config/alarm/add`

```json
[
  { "Point_Id": 100, "Name": "高温报警", "Config": "==1", "Group": 1 }
]
```

#### 更新 `POST /api/v1.0/config/alarm/update`

```json
[
  { "Id": 1, "Config": "==0", "Group": 2 }
]
```

#### 删除 `POST /api/v1.0/config/alarm/del`

```json
{ "ids": [1] }
```

#### Config 字段说明

报警配置使用 expr 表达式引擎解析，`Config` 字段支持以下格式：

| 表达式 | 说明 | 示例 |
|--------|------|------|
| `==value` | 等于 | `==1` (值等于1时报警) |
| `!=value` | 不等于 | `!=0` (值不等于0时报警) |
| `>value` | 大于 | `>100` (温度超过100时报警) |
| `<value` | 小于 | `<5` (电压低于5V时报警) |
| `>=value` | 大于等于 | `>=80` |
| `<=value` | 小于等于 | `<=0` |
| `in[a,b,c]` | 在范围内 | `in[1,2,3]` |
| `!in[a,b,c]` | 不在范围内 | `!in[0,99]` |

> **注意**: Config 字段会被缓存到 freecache 中，TTL 默认设置为 1 小时，减少数据库查询压力。

---

### 2.4 历史配置

#### 查询 `POST /api/v1.0/config/history/query`

```json
{ "pointId": [100], "page": 1, "pageSize": 20 }
```

| 请求字段 | 类型 | 说明 |
|----------|------|------|
| pointId | []uint | 按点位 ID 过滤（空则不过滤） |
| page | uint | 页码 |
| pageSize | uint | 每页数量 |

#### 新增 `POST /api/v1.0/config/history/add`

```json
[
  { "Point_Id": 100, "Config": "{...}" }
]
```

#### 更新 `POST /api/v1.0/config/history/update`

```json
[
  { "Point_Id": 100, "Config": "{...}" }
]
```

#### 删除 `POST /api/v1.0/config/history/del`

```json
{ "ids": [1] }
```

---

## 三、配置导出

导出配置为 Excel 文件，支持同时导出多种配置（每种一个 Sheet）。

```
GET /api/v1.0/config/export?type=drive,point,alarm,history
```

### 传参方式

| 方式 | 示例 |
|------|------|
| 逗号分隔 | `?type=drive,point,alarm,history` |
| 数组形式 | `?type=drive&type=point` |

### type 值

| type | Sheet 名 | 导出列 |
|------|----------|--------|
| drive | drive_config | Id, Type, Name, Config, Points_Length |
| point | point_config | Id, Drive_Id, Name, Description, Config, RW_Cancel, Value_Type |
| alarm | alarm_config | Id, Point_Id, Name, Config, Group |
| history | history_config | Id, Point_Id, Config |

### 响应

- **Content-Type**: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- **文件名**: `config_export_20260922_150405.xlsx`（自动带时间戳）
- 全量导出，不分页