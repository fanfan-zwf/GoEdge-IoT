# 内部 MQTT 功能说明

内部 MQTT 功能让采集器（Collector）通过一条 **独立的 MQTT 连接** 对外提供「读写 / 运维 / 监控」能力，
可用于被云端或运维平台远程下发指令、采集实时数据与系统状态。

- 代码位置：`internal/service/internalmqtt/`
- 文件划分：`internalmqtt.go`（`Start`/主题前缀/发布入口）、`publish.go`（上行）、`subscribe.go`（下行四类指令）、`types.go`（各主题 DTO）、`log.go`（下行指令的审计日志埋点）
- 连接建立：`internal/infra/mqtt/mqttbase.go` 的 `New()`（连接名固定为 `internal`，为一条专用连接）
- 启动接入：`cmd/server/main.go` 在 `mqtt.New()` 之后调用 `internalmqtt.Start()`
- 配置段：`configs/config.yaml` 的 `Internal_Mqtt`

---

## 一、主题结构

所有主题统一由三段拼成（`joinTopic` 会自动去重斜杠）：

```
{topic_prefix} / {APP.label} / {后缀}
```

- `topic_prefix`：默认 `/IOT/Collector`，可在 config 自定义
- `APP.label`：设备标识（`config.yaml` 中 `APP.label`），拼进主题实现一机一根主题
- `后缀`：见下表，均可在 `Internal_Mqtt` 中自定义

示例根主题：`/IOT/Collector/123e4567-e89b-12d3-a456-426614174000`

订阅侧逐个订阅具体的指令主题（`{根主题}/write`、`/restart`、`/token`、`/config/update`），
避免把本服务自己发布的上行主题回声收回。

---

## 二、配置项（`configs/config.yaml`）

```yaml
Internal_Mqtt:
  enable: true                 # 总开关；false 时 Start() 直接跳过

  # —— 专用连接参数 ——
  broker: tcp://127.0.0.1:1883 # broker 地址
  username: o7aKZOj53p2PkkFJzKzM
  password: lpj4VZ4ZYFg5GmINhRat
  # client_id 留空则默认取 APP.label
  clean_session: True          # 清洁会话（重启不接收离线消息）
  auto_reconnect: True         # 自动重连
  connect_timeout: 10s         # 连接超时
  write_timeout: 5s            # 写超时
  keep_alive: 30s              # 心跳保活

  topic_prefix: /IOT/Collector # 主题前缀（可自定义）

  # —— 订阅（下行）后缀 ——
  sub_write: /write            # 写数据
  sub_restart: /restart        # 重启驱动/服务
  sub_token: /token            # 增加临时 handler token
  sub_config_update: /config/update # 更新所有配置

  # —— 发布（上行）后缀 ——
  pub_data: /data             # 点位实时值
  pub_alarm: /alarm           # 报警事件（配置字段 + 报警值/时间）
  pub_status_prefix: /status  # 监控状态（每功能一个子主题）
  pub_result: /result         # 操作结果回执

  status_interval: 30s        # 监控状态发布周期
  data_full_interval: 10m     # 点位数据全量推送周期
```

内置默认值（`applyMqttDefaults`，yaml 未配置时生效）：`client_id=APP.label`、`connect_timeout=10s`、`write_timeout=5s`、`keep_alive=30s`、`topic_prefix=/IOT/Collector`、各后缀同上、`status_interval=30s`、`data_full_interval=10m`。

> ⚠️ 连接仅在进程启动时（`mqtt.New()`）建立一次；修改 `Internal_Mqtt` 的连接参数需 **重启服务** 生效。

---

## 三、订阅（下行指令）

以下主题为 **你 → 采集器** 方向。除写数据外，异步动作的处理结果统一通过 `{根主题}/result/{动作}` 回执（见第五节）。

### 3.1 写数据 `{sub_write}` = `/write`

payload 为 **数组 JSON**，以全局唯一的 `tag` 为键，带类型、值，可选时间戳（RFC3339，缺省用当前时间）。

```json
[
  { "tag": "//123e4567-e89b-12d3-a456-426614174000//温度_A1", "type": "float64", "value": 23.5, "time": "2026-11-13T10:00:00Z" },
  { "tag": "//123e4567-e89b-12d3-a456-426614174000//阀门_K1",  "type": "bool",    "value": true },
  { "tag": "//123e4567-e89b-12d3-a456-426614174000//设定_P1",  "type": "int32",   "value": 1024 }
]
```

- `tag` 与数据库 `Point_Config.Tag` 完全一致（即 `/data` 发布回来的 `tag`），服务内部自动反查点位 id 与所属驱动
- `type` 合法值：`bool`、`int8/uint8`、`int16/uint16`、`int32/uint32`、`int64/uint64`、`int/uint`、`float32/float64/float`、`string`
- 处理：按 `tag` 反查点位（未命中则刷新缓存重试一次）→ 按 `type` 精确转换 → 分组调用 `driver.DriveWrite`
- 回执动作名：`write`；成功 `data` 形如 `{ "count": 3 }`
- 审计日志：`Type=point`、`Operate=write`，**一个点位一行**（`point_id`/`tag`/`type`/`value`/`time`）；整批拒绝时记 `ok:false` + `reason` + 截断后的原始报文

### 3.2 重启 `{sub_restart}` = `/restart`

用 `target` 字段区分重启对象。

重启指定驱动（`drive_ids` 为空 = 重启全部驱动）：

```json
{ "target": "driver", "drive_ids": [1, 2] }
```

重启服务进程（发送 SIGTERM 让本进程优雅退出，**依赖 systemd/守护进程重新拉起**）：

```json
{ "target": "service" }
```

- 回执动作名：`restart`
- 驱动重启为异步，先回执「已启动」，完成后再回执 `{ "restarted": N }`
- `service` 目标回执后会延时约 1 秒退出，以便回执先发；若无守护拉起，进程不会自动恢复
- 审计日志：驱动重启 `Type=drive`、服务重启 `Type=service`、`Operate=restart`，各记「受理」与「完成（含 `restarted`/`total`）」两行，单个驱动失败另记一行；`target` 非法记 `ok:false`（此时 `Type=all`）

### 3.3 增加临时 handler token `{sub_token}` = `/token`

为给定 `access_key` 生成短(access) / 长(refresh)令牌，**直接写入 Redis**（不落 account / api_key 表）。
可通过 `with_access_token` / `with_refresh_token` **单独签发长令牌或短令牌**。

默认（不传开关）同时签发长、短两种令牌：

```json
{
  "access_key": "your-access-key",
  "access_token_ttl": "15m",
  "refresh_token_ttl": "168h"
}
```

只要长令牌（refresh），不要短令牌（access）：

```json
{
  "access_key": "your-access-key",
  "refresh_token_ttl": "720h",
  "with_access_token": false,
  "with_refresh_token": true
}
```

- `with_access_token` / `with_refresh_token`：是否签发对应令牌；**缺省（不填）视为 `true`**，两者不可同时为 `false`
- TTL 为 Go duration 字符串（如 `15m`、`168h`），留空/非法回退默认：access `15m`、refresh `168h`
- 令牌绑定 IP 置空，以跳过会话 IP 一致性校验，便于远程签发
- 回执动作名：`token`，`data` 返回 **本次实际签发的令牌**（未签发的类型不会出现在回执里）：

两者都签发：

```json
{
  "access_key": "your-access-key",
  "token_type": "Bearer",
  "access_token": "……",
  "expires_in": 900,
  "refresh_token": "……",
  "refresh_expires": 604800
}
```

只签发长令牌（`with_access_token:false`）：

```json
{
  "access_key": "your-access-key",
  "token_type": "Bearer",
  "refresh_token": "……",
  "refresh_expires": 2592000
}
```

> 令牌经 MQTT 明文回执，请确保该内部连接处于可信网络/权限范围内。

- 审计日志：`Type=account`、`Operate=login`（远程签发令牌等同一次登陆），记 `target_access_key`、`user_id`、签发了哪几种令牌与 TTL；**令牌明文绝不入库**（回执里给，日志里无）。

### 3.4 更新所有配置 `{sub_config_update}` = `/config/update`

JSON 仅携带一个 http 下载地址，采集器下载 **表格(xlsx)** 后复用导入逻辑全量导入。

```json
{ "url": "http://example.com/config/collector.xlsx" }
```

- `url` 与 `download_url` 二者取其一（优先 `url`）
- 下载限制：超时 30s、最大 50MB；非 `http/https` 前缀直接拒绝
- 导入含四张 Sheet：`drive_config` / `point_config` / `alarm_config` / `history_config`
- 导入成功后自动刷新点位 tag 缓存，并让各驱动按新点位配置动态重组（不断连接）
- 回执动作名：`config_update`，`data` 为各 Sheet 的 `added/updated/deleted` 统计
- 审计日志：**按 Sheet 各记一行** `Operate=import`，`Type` 由 Sheet 名直接映射（`drive_config`→`drive`、`point_config`→`point`、`alarm_config`→`alarm`、`history_config`→`history`），内容含 `url`/`sheet`/`added`/`updated`/`deleted`；下载、解析、导入失败记 `Type=all` 的 `ok:false` 行

---

## 四、发布（上行数据）

以下为 **采集器 → 你** 方向。

### 4.1 点位实时值 `{pub_data}` = `/data`

payload 为 **数组 JSON**，以全局唯一的 `tag` 为键。触发方式：

1. **变化即推**：任一点位值变化后即时推送（推送变化的点位）
2. **全量周期推**：每 `data_full_interval`（默认 10 分钟）推送一次全部点位当前值

```json
[
  {
    "tag": "//123e4567-e89b-12d3-a456-426614174000//温度_A1",
    "value": 23.5,
    "type": "float64",
    "msg": "",
    "time": "2026-11-13T10:00:05Z"
  }
]
```

- `tag` 即数据库 `Point_Config.Tag`（内部缓存按 `pointId` 补全；启动加载、全量推送与配置更新后刷新），与 `/write` 的 `tag` 一致，可直接回填下发
- `type` 为可读类型名，`time` 为 RFC3339

### 4.2 报警事件 `{pub_alarm}` = `/alarm`

payload 为 **数组 JSON**，**事件驱动即时推送**：每当一批报警触发或恢复（订阅报警事件总线 `collector.Alarm_Subscriber`），就把本批报警一次性发出。每个元素包含 **报警配置的全部字段** + **报警值** + **报警时间**：

```json
[
  {
    "tag": "//123e4567-e89b-12d3-a456-426614174000//温度_A1",
    "alarm_id": 12,
    "config": "x > 100",
    "group": 1,
    "status": "报警",
    "value": 105.3,
    "type": "float64",
    "time": "2026-11-13T10:00:05Z"
  }
]
```

- `alarm_id` / `config` / `group` 来自报警配置（`Alarm_Config` 表），`config` 为 expr 表达式（`x` 代表点位值）
- `status`：`报警`（表达式为真）/ `恢复`（为假）/ 编译或执行错误信息
- `value` 为触发时的点位值，`type` 为其可读类型名，`time` 为报警时间（RFC3339）
- `tag` 同 `/data`，由 `pointId` 反查补全，可直接定位报警点位

### 4.3 监控状态 `{pub_status_prefix}` = `/status`

**每个功能一个子主题**，周期 `status_interval`（默认 30s）采集并分别发布。payload 为 gopsutil 对应结构体的 JSON。

| 子主题 | 内容 |
|--------|------|
| `/status/host` | 主机信息 |
| `/status/sensors` | 传感器 |
| `/status/cpu` | CPU 型号等信息 |
| `/status/cpu/percent` | CPU 使用率（数组） |
| `/status/cpu/cores` | CPU 核数 |
| `/status/mem` | 虚拟内存 |
| `/status/mem/swap` | 交换内存 |
| `/status/disk` | 磁盘分区 |
| `/status/disk/usage` | 根分区使用率 |
| `/status/disk/io` | 磁盘 IO 计数 |
| `/status/load` | 系统负载 |
| `/status/net` | 网络 IO 计数 |
| `/status/net/interfaces` | 网络接口 |
| `/status/process/pids` | 进程 PID 列表 |

> 单个功能采集失败仅记日志并跳过，不影响其余功能发布。

---

## 五、操作结果回执 `{pub_result}` = `/result`

异步动作（写数据 / 重启 / 令牌 / 配置更新）的处理结果统一发布到：

```
{根主题}/result/{动作}
```

例如写数据回执主题：`/IOT/Collector/{label}/result/write`。

payload 结构：

```json
{
  "action": "write",
  "ok": true,
  "msg": "ok",
  "data": { "count": 3 }
}
```

| 动作名 `action` | 来源指令 |
|-----------------|----------|
| `write` | 写数据 |
| `restart` | 重启驱动/服务 |
| `token` | 签发临时令牌 |
| `config_update` | 更新所有配置 |

---

## 六、审计日志

四类下行指令 **全部落库** 到单表 `log_operate`（与 HTTP 通路同一张表，同一套字段）。

MQTT 通路没有 HTTP 那套凭证（`Internal_Mqtt.username` 只是连接凭证，不代表操作人），因此：

| 列 | 写法 |
|---|---|
| `Type` / `Operate` | 与 HTTP 通路同一套**文本枚举**（`point`/`drive`/`service`/`account`/`all` × `write`/`restart`/`login`/`import`），库里直接读得懂，不是数字 |
| `Operator` | 固定 `internalmqtt`，**不归属任何账号** |
| `Ip` | 空串（MQTT 报文无来源 IP 可用） |
| `Content.channel` | `internalmqtt` |
| `Content.topic` | 触发本次动作的 **完整主题**（如 `/IOT/Collector/<label>/write`），相当于“指令从哪里来” |
| `Content.detail` | 业务详情；失败行带 `ok:false`、`reason` 与截断到 1024 字的原始报文 `raw` |

埋点入口在本包 `log.go`（`logMqtt` / `logMqttFail` / `logMqttWrites`），统一经
[`handler.LogExternalEvent`](../handler/README.md) / `handler.LogMqttWriteValues` 落库：

- **异步写、失败只打印**，绝不影响指令处理与回执；
- 日志只写“动了什么”，不写凭证：令牌明文、`secret_key`、密码一律不入库（回执里照旧返回）；
- 可在接口侧查看：`POST /api/v1.0/config/log/query` 传 `{"operators":["internalmqtt"]}`、`{"types":["point"],"operates":["write"]}` 或 `{"keyword":"write"}`（枚举维度就用上面的文本值）。

---

## 七、快速自检

1. `configs/config.yaml` 中 `Internal_Mqtt.enable: true` 且 `broker` 地址、账号可用
2. `APP.label` 已配置（否则 `Start()` 直接报错）
3. 启动日志出现：`INFO 内部MQTT功能启动，独立连接=internal 主题前缀=/IOT/Collector/{label}`
4. 用 MQTT 客户端订阅 `/IOT/Collector/{label}/#`，向 `{根主题}/write` 发一条数组 JSON，观察 `{根主题}/result/write` 回执
5. 指令执行后查 `log_operate`（或 `/api/v1.0/config/log/query` 带 `operators:["internalmqtt"]`），应能看到对应的审计行
