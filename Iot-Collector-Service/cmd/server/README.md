# `cmd/server` — 程序入口

上级文档：[服务总览 README](../../README.md)

本目录只有一个 `main.go`，负责**按依赖顺序初始化各模块**与**优雅退出**。
所有业务逻辑都在 `internal/` 各包内，入口层刻意保持"只做装配、不写逻辑"。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `main.go` | ~220 | 初始化编排（`app`）、优雅退出（`gracefulExit`/`step`）、信号监听（`main`）、启动期配置同步（`syncConfigOnStartup`） |

---

## 二、启动顺序（`app()`）

顺序**不可随意调整**，每一步都依赖前一步建立的全局状态：

```
1. config 包 init()        # 由 import 触发：读 configs/config.yaml + 初始化日志
2. syncConfigOnStartup()   # 可选：从 Iot-Config-Service 拉取增量配置写入 MySQL
3. mqtt.New()              # 建立全局 MQTT 管理器（含 internal 专用连接）
4. internalmqtt.Start()    # 依赖第 3 步的 internal 连接，注册读写/运维/监控主题
5. handler.Web()           # 启动 Gin HTTP 服务（此时驱动尚未就绪，接口已可用）
6. collector.New()         # 采集核心初始化（当前为预留空实现）
7. influxdb.New()          # 仅当 Influxdb.enable=true；内部 init() 已订阅 updateBus
8. sleep 200ms             # 等待订阅关系与缓冲区就绪
9. driver.Start(collector.Collection_Publisher)  # 查 MySQL 配置 → 创建/连接所有驱动
```

关键约束：

- **`internalmqtt.Start()` 必须在 `mqtt.New()` 之后**：它复用名为 `internal` 的全局连接。
- **`driver.Start` 接收 `collector.Collection_Publisher` 作为回调参数**，而不是让 driver 包直接 import collector——
  这是刻意的依赖注入，用于打断 `service/driver → service/collector → driver` 的 import cycle。
- **接口早于驱动可用**：`handler.Web()` 在第 5 步就启动，此时 `driver.Manager` 必须已经非 nil
  （`Manager` 在包加载时即 `newDriverManager()`，见 [service/driver 文档](../../internal/service/driver/README.md)），
  否则这个窗口内的写值/保存点位请求会 panic 并带走进程。

---

## 三、优雅退出

### 3.1 时间预算

```go
const (
    shutdownTotal = 15 * time.Second // 整体最多等 15s
    shutdownStep  = 5 * time.Second  // 单个步骤最多等 5s
)
```

设计动机：以前某个 PLC 连接卡住会让进程怎么都退不掉，最终只能被 systemd `SIGKILL`。
现在无论哪一步失败/超时/panic，都**只记日志、不阻断后续步骤**。

### 3.2 收尾顺序（`gracefulExit`）

原则：**先断对外入口 → 再停采集推送 → 最后关底层连接**。

| 序号 | 步骤 | 调用 | 为什么在这个位置 |
|------|------|------|------------------|
| 1 | WebSocket 推送 | `handler.WSCloseAll("服务正在关闭")` | 让前端拿到正常的 1001 关闭事件，而不是 TCP 被强断的异常 |
| 2 | HTTP 接口 | `handler.WebShutdown(ctx)` | 不再接新连接，在途请求（写值/导入）跑完再关 |
| 3 | 内部 MQTT 发布 | `internalmqtt.Stop()` | 停止周期发布，避免往待关闭的连接写数据 |
| 4 | 驱动连接 | `driver.CloseAll(ctx)` | 关轮询协程并断开 PLC/Modbus（设备侧看到的是一次正常断连） |
| 5 | MQTT 连接 | `mqtt.CloseAll()` | 向 broker 发正常 DISCONNECT |
| 6 | InfluxDB 缓冲 | `closeInfluxOnce()` | 把客户端缓冲区未发送的数据先落盘（仅 enable 时执行） |
| 7 | Redis / MySQL | `redis.Close()` / `mysql.Close()` | 最底层连接池，前面几步仍会用到，必须最后关 |

### 3.3 三个保护机制

1. **`shutdownOnce sync.Once`**：收尾只跑一次。信号路径（`main`）与 `defer exit()` 都可能触发，第二次直接空转。
2. **`step(ctx, name, fn)`**：每步在独立 goroutine 中执行并 `recover()`，
   panic 降级为错误日志——**绝不让收尾把进程带倒**；总预算 `ctx.Err()` 已触发则跳过该步并打印 `退出超时，跳过[xxx]`。
3. **`influxCloseOnce sync.Once`**：`closeInfluxOnce()` 保证 InfluxDB 客户端只关一次，
   避免 Flush 两次造成多余请求或关闭已关闭的通道。

---

## 四、信号处理

```go
sigChan := make(chan os.Signal, 2)   // 带缓冲 2，避免信号丢失
signal.Notify(sigChan,
    syscall.SIGINT,  // Ctrl+C
    syscall.SIGTERM, // kill <pid>（非 -9）
    syscall.SIGHUP,  // 关闭终端 / 容器窗口
    syscall.SIGQUIT, // Ctrl+\
)
```

- 阻塞在 `sig := <-sigChan`，收到信号后执行 `gracefulExit()`。
- **收尾期间再收到一次信号立即 `os.Exit(1)`**：用于"卡死在某个收尾步骤"时的人工逃生通道
  （代价是未完成收尾，客户端缓冲区数据可能丢失）。
- `kill -9` 无法捕获，属于强制终止路径，不保证缓冲区落盘。

后台还以 `_ "github.com/icattlecoder/godaemon"` 空导入注册了 `-d` 守护进程化 flag（可选）。

---

## 五、启动期配置同步

```go
if config.Config.Config_Service.Enable {
    syncConfigOnStartup()
}
```

内部两步（详见 [configsync 文档](../../internal/service/configsync/README.md)）：

1. `configsync.IsConfigUpdated()` 拿服务端最新更新时间；**零值 = 无更新，直接返回**。
2. 有更新才 `configsync.SyncConfigFromService(t)` 拉全量配置并按 Id 增删改本地 MySQL。

> 这一步必须早于 `driver.Start()`，否则驱动会拿上一轮的旧点位配置去组包。
> 同步失败只记 `ERROR` 日志、不阻断启动——边缘设备断网时仍需以本地配置继续采集。

---

## 六、运行与调试

```bash
# 前台运行（开发）
make run                    # == go run ./cmd/server

# 编译产物
make build                  # == go build -o bin/collector-service ./cmd/server
./bin/collector-service

# 工作目录敏感项：configs/config.yaml 与 third_party/lib/libsnap7.so 都是相对路径，
# systemd / nohup 启动时务必把 WorkingDirectory 指向 Iot-Collector-Service/
```

常见问题：

| 现象 | 原因 | 处置 |
|------|------|------|
| `读取配置文件错误` panic | 工作目录不对，找不到 `configs/config.yaml` | 修正 `WorkingDirectory` / 启动目录 |
| `配置文件版本 x 与软件版本 y 不兼容` | `APP.version` 主/次版本与内置 `Version` 不一致 | 改 `APP.version` 或换配套配置模板 |
| 启动即 `panic: nil pointer` 且栈在 driver | 早期版本的 `Manager` 未就绪窗口 | 现由包级 `var Manager = newDriverManager()` 修复，勿改回 |
| 改了 MQTT 参数不生效 | 连接只在 `mqtt.New()` 建立一次 | 重启服务 |

---

## 七、相关文档

- [internal/config](../../internal/config/README.md) — 配置加载与日志
- [internal/service/driver](../../internal/service/driver/README.md) — 驱动启动与生命周期
- [internal/service/internalmqtt](../../internal/service/internalmqtt/README.md) — 对外 MQTT 主题
- [internal/handler](../../internal/handler/README.md) — HTTP 接口与 WebSocket
- [internal/repo/influxdb](../../internal/repo/influxdb/README.md) — 时序写入与缓冲区
