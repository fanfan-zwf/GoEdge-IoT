# `internal/config` — 配置加载与日志

上级文档：[服务总览 README](../../README.md)

本包是**全局单例配置**的唯一来源：进程一启动就把 `configs/config.yaml` 解析进
`config.Config`，供所有包直接读取；同时初始化按日期分文件、带精确去重的日志记录器。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `config.go` | 395 | `Config_type` 结构体、`init()` 加载、版本兼容校验、MQTT 默认值补全、`ReloadConfig`/`SaveConfig` |
| `log.go` | 971 | `DedupDailyLogger` 按日切分 + 精确去重日志器、`init_log()` 接管标准库 `log` |

包级导出：`Config`（全局唯一实例）、`Version`、`ConfigFilePath`、一组 `Regex_*` 常量、`ReloadConfig`、`SaveConfig`、日志器类型。

---

## 二、加载时机与副作用

```go
const ConfigFilePath = "configs/config.yaml"

func init() {
    loadConfig()   // 读 yaml → 填 Config → 校验版本 → 补 MQTT 默认值
    init_log()     // 重定向标准库 log 输出到 DedupDailyLogger
}
```

- `init()` 由 `cmd/server/main.go` 的 import 触发，**早于任何业务包**，
  因此其它包的 `init()` 里可以安全读 `config.Config`（如 collector 的报警缓存 TTL）。
- **相对路径**：`ConfigFilePath` 是相对工作目录的，systemd/脚本启动必须设好 `WorkingDirectory`。
- 读文件失败 / yaml 解析失败 / 版本不兼容一律 `log.Panic`——配置错误宁可启动即崩，也不要带着半套配置跑。

---

## 三、配置结构（`Config_type`）

| 段 | 关键字段 | 默认值（yaml 未写时） | 说明 |
|----|----------|------------------------|------|
| `APP` | `version` / `sn` / `passwd` / `label` | 无 | `label` 是设备标识，参与 Tag 与主题拼接；`version` 必须与内置 `Version` 主/次版本一致 |
| `Config_Service` | `enable` / `address` / `setTimeout` / `setRetryCount` / `setRetryWaitTime` | `10s` / `1` / `3s` | 与 Iot-Config-Service 同步配置用；`setRetryCount=1` 限制 401 重试次数避免死循环 |
| `API` | `enable` / `ip` / `post` / `header[]` | `0.0.0.0` / `8102` | `header` 为 CORS 响应头列表，逐条 `Set`；生产应改成本地端口 |
| `MYSQL` | `dsn` | 无 | `Init_sql` 使用 |
| `REDIS` | `ip` / `post` / `passwd` / `database` | 无 | 令牌、RBAC 判定缓存、权限缓存 |
| `Influxdb` | `enable` / `url` / `token` / `org` / `bucket` / `write_timeout` / `buffer_size` / `flush_interval` / `max_retries` / `retry_interval` / `retry_buffer` / `use_gzip` / `write_quantity_tag` | `max_retries=3`、`retry_buffer` 自动计算 | 缓冲/重传参数全部作用于**客户端内置缓冲区**，详见 [repo/influxdb](../repo/influxdb/README.md) |
| `LOG` | `enable` / `path` / `cacheTTL` / `flags` | `./log` / `30` / `Ldate Lmicroseconds Llongfile` | `path` 为日志目录；`cacheTTL` 单位秒，是去重窗口 |
| `Internal_Mqtt` | 连接参数 + `topic_prefix` + 4 个订阅后缀 + 4 个发布后缀 + 2 个周期 | 见下节 | 对外 MQTT 能力，详见 [service/internalmqtt](../service/internalmqtt/README.md) |
| `ALARM` | `Config_CacheTTL` | `1h`（`0s`=永久不过期） | freecache 报警配置 TTL |
| `RBAC` | `cache_ttl` | `30m` | **判定结果**缓存 TTL；RBAC 恒开启、无开关、无超管白名单 |

### 内置版本常量与校验正则

```go
const Version = "v1.0.0"                // 软件版本，配置文件 APP.version 必须与之主/次版本一致
const RFC_FAN   = "2006-01-02 15:04:05.000000"
const User_Permissions = 100

Regex_Name            = `^.{0,6}$`
Regex_Passwd          = `^(?=.*\d)(?=.*[!@#$%&])[A-Za-z\d!@#$%&]{8}$`   // 8 位，必含数字与 !@#$%&
Regex_Passwd_sha3_256 = `^[a-f0-9]{64}$`
Regex_Phone           = `^1[3-9]\d{9}$`
Regex_Email           = `^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`
Regex_URL             = `https?:\/\/[^\s]+`
```

> `Regex_URL` 被 `config` 与 handler 用于校验配置下载地址（内部 MQTT 的 `/config/update` 还会额外强制 `http/https` 前缀）。

---

## 四、版本兼容校验（`checkAppVersion`）

三段子函数协作：`leadingInt` → `parseMinor` → `checkAppVersion`。

规则：

1. **只比较前两段（主/次）的数值**，第三段往后的修订号/构建号/预发布标记一律忽略：
   软件 `v1.0.0` 接受配置 `v1.0` / `v1.0.2` / `v1.0.3` / `v1.0.0.1`，拒绝 `v1.1` / `v2.1`。
2. 段数不限但**至少两段**，每段非空（`v1.0.`、`v1..0` 均非法），前两段必须以数字开头。
3. 前缀 `v`/`V` 可省略；数字允许带后缀（`12-rc1`、`3.1 ` 都能取出 12 / 3）。
4. 配置版本为空 → 直接拒绝，避免拿错版本的配置文件继续启动。

---

## 五、MQTT 默认值补全（`applyMqttDefaults`）

在 `loadConfig` 与 `ReloadConfig` 解析完成后调用，保证 yaml 未显式配置时也能工作：

| 项 | 缺省值 |
|----|--------|
| `client_id` | 取 `APP.label`（label 为空则不填） |
| `connect_timeout` / `write_timeout` / `keep_alive` | `10s` / `5s` / `30s` |
| `topic_prefix` | `/IOT/Collector` |
| 订阅后缀 | `/write` `/restart` `/token` `/config/update` |
| 发布后缀 | `/data` `/alarm` `/status` `/result` |
| `status_interval` / `data_full_interval` | `30s` / `10m` |

> 注意 `configs/config.yaml` 把 `status_interval` 显式改成了 `3s`（比内置默认更频繁），
> 排查"监控主题推送太频繁/太慢"时先看配置而不是这里。

---

## 六、热更新：`ReloadConfig` 与 `SaveConfig`

### `ReloadConfig() error`

- 重读 yaml → `yaml.Unmarshal` 进 `Config`。
- **失败整体回滚**：解析出错或版本不兼容时 `Config = defaultConfig`（先前的完整快照），
  绝不留"半套新版本配置"生效。
- 解析成功后逐项补默认值（`SetTimeout`、`API.Ip/Post`、`ALARM.Config_CacheTTL`、`RBAC.TTL`、`LOG.*`），
  再走一次 `applyMqttDefaults()`。
- 调用入口：HTTP `POST /api/v1.0/system/config/reload`（见 [handler 文档](../handler/README.md)）。

**不会因 Reload 生效的部分**：MQTT 连接参数只在 `mqtt.New()` 时建立一次；
驱动配置需走 `DriveResetConfig`；日志 flag 在 `init_log()` 时已固定。

### `SaveConfig() error`

`yaml.Marshal(&Config)` 后以 `0644` 覆盖写 `configs/config.yaml`。
配合 `POST /api/v1.0/system/config/write` 使用；写盘是**整份结构体重排**，
原文件的注释与键序不会保留——需要注释的模板请走版本管理，不要指望接口回写。

---

## 七、日志（`log.go`）

### 7.1 设计目标

工业现场设备（弱网、PLC 掉线）会产生**海量重复错误日志**，一天几十 MB 且无法定位。
`DedupDailyLogger` 解决两件事：按日期切文件 + 相同内容在窗口期内只落一次并计数。

### 7.2 对外类型

```go
type LogCallback func(timestamp time.Time, microseconds, longfile, message string) error
type CacheStats struct { TotalEntries, TotalHits, Size int }

type DedupDailyLogger struct { /* 文件轮转 + 去重缓存 + 统计 */ }

func NewDedupDailyLogger(logDir, prefix string, cacheTTL time.Duration) *DedupDailyLogger
func (d *DedupDailyLogger) SetCallback(cb LogCallback)      // 每条日志的旁路回调（如入库）
func (d *DedupDailyLogger) Write(p []byte) (n int, err error) // 实现 io.Writer，接管 log 输出
func (d *DedupDailyLogger) GetCacheStats() CacheStats
func (d *DedupDailyLogger) GetTopDuplicates(limit int) []struct{ ... }
func (d *DedupDailyLogger) CleanupOldLogs(retentionDays int)
func (d *DedupDailyLogger) CleanCache()
func (d *DedupDailyLogger) Close() error
```

内部方法：`startCacheCleanup` / `cleanupExpiredCache`（TTL 过期清理）、
`getFilename` / `rotate`（跨天切文件）、`normalizeMessage`（正则归一化后再比对）、
`isDuplicate`（命中则累加计数）、`parseLogMessage`（拆时间戳/微秒/文件名/正文）。

### 7.3 归一化正则

去重前先做"变量归一"，让 `驱动id:12 等待:3000ms` 这类只差数字的日志折叠成同一条：

| key | 作用 |
|-----|------|
| `timestamp` | 折叠完整时间戳 |
| `filepath` | 折叠 `xxx.go:行号` |
| `ip_address` / `port` | 折叠 IP 与端口 |
| `driver_id` / `wait_time` | 折叠中文标签后的数字 |
| `any_number` | 兜底折叠任意数字 |

### 7.4 `init_log()`

- `LOG.enable=false` 时保持标准库默认（仅 stdout）。
- 开启时：创建 `logDir`（默认 `./logs`）、按 `LOG.flags` 字符串解析 `Ldate`/`Ltime`/`Lmicroseconds`/`Llongfile`/`Lshortfile` 等位、
  `log.SetOutput(logger)` 并把 `log.SetFlags(0)`（标志位由 logger 自己解析，避免重复前缀）。
- 另有包级变量 `Log__Add2 func(User_Id uint, Type, Message string) (uint, error)`，
  为"日志入库"预留的函数注入点（由上层在需要时赋值，避免 config 反向依赖 repo）。

### 7.5 输出位置

```
logs/
├── <prefix>-2026-11-20.log     # 按日切分
└── ...
```

> 文件内既有 stdout 的内容（`log.Print*` 走的就是这个 writer），
> 若由 systemd 拉起，stdout 也会同时进 journald。

---

## 八、新增一个配置段的正确姿势

1. 在 `Config_type` 里加内嵌 struct + `yaml:"xxx"` tag（时长用 `time.Duration` 或 `conv.Duration`，见 [pkg/conv](../../pkg/conv/README.md)）。
2. 在 `loadConfig()` 里设默认值（**在 `yaml.Unmarshal` 之前**设，yaml 缺字段时才会保留）。
3. 若希望热更新后仍保留默认，`ReloadConfig()` 里补一次"零值 → 旧值/内置值"回退。
4. 在 `configs/config.yaml` 写示例并加中文注释；根 README 的配置说明同步。
5. 若该段可被前端读写，需在 `handler/system.go` 的读写接口做校验（注意会整份重写 yaml）。

---

## 九、相关文档

- [configs/config.yaml](../../configs/config.yaml) — 实际生效的配置
- [internal/service/internalmqtt](../service/internalmqtt/README.md) — `Internal_Mqtt` 主题语义
- [internal/repo/influxdb](../repo/influxdb/README.md) — `Influxdb` 缓冲参数落点
- [internal/service/collector](../service/collector/README.md) — `ALARM.Config_CacheTTL` 落点
- [internal/service/rbac](../service/rbac/README.md) — `RBAC.cache_ttl` 落点
