# `internal/infra/mqtt` — MQTT 连接基础设施

上级文档：[服务总览 README](../../../README.md)

对 Eclipse Paho 的薄封装：**多连接管理器 + 全局单例 + 包级收发函数**。
本包只负责"连上、发出去、收进来、重连后补订阅"，**不含任何主题语义**；
业务主题（write/data/alarm/status…）全部在 [internal/service/internalmqtt](../../service/internalmqtt/README.md) 里实现。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `mqttbase.go` | 313 | `MqttConfig`、`Manager`（多实例）、全局 `GlobalMQTT`、`New`/`CloseAll`、包级 `Send`/`Subscribe`/`Subscribe_Close` |

依赖：`github.com/eclipse/paho.mqtt.golang`（EPL/EDL 双许可）。

---

## 二、⚠️ 最容易踩的坑：`broker` 参数是"连接名"，不是地址

```go
mqtt.Send("internal", topic, data)   // ✅ "internal" 是 Manager.inst 的 key
mqtt.Send("tcp://127.0.0.1:1883", …) // ❌ 实例不存在: tcp://127.0.0.1:1883
```

目前本服务**只有一条连接**，名字固定为常量：

```go
const InternalBrokerName = "internal"   // 由 config.yaml 的 Internal_Mqtt 段建立
```

要接第二条 broker，就用 `GlobalMQTT.Add("<新名字>", MqttConfig{…})`，之后所有收发都用那个名字。

---

## 三、类型与 API

### 3.1 `MqttConfig`（纯连接参数，无 RPC 字段）

| 字段 | 来源（`Internal_Mqtt`） | 说明 |
|------|--------------------------|------|
| `Broker` | `broker` | `tcp://host:1883` / `ssl://…` |
| `Username` / `Password` | `username` / `password` | 明文写在 yaml 中，注意文件权限 |
| `ClientID` | `client_id` | 留空时由 `config.applyMqttDefaults()` 填成 `APP.label` |
| `SetCleanSession` | `clean_session` | `true` → 重启不接收离线消息 |
| `SetAutoReconnect` | `auto_reconnect` | 生产环境务必 `true` |
| `SetConnectTimeout` / `SetWriteTimeout` / `SetKeepAlive` | `connect_timeout` / `write_timeout` / `keep_alive` | 缺省 `10s` / `5s` / `30s` |

### 3.2 `Manager`

| 方法 | 行为 |
|------|------|
| `NewManager() *Manager` | 建空管理器（`inst` map + `RWMutex`） |
| `Add(name, cfg) error` | 建连并登记；失败返回 error（**不会**在 map 里留下半可用实例） |
| `GetClient(name) (mqtt.Client, error)` | 拿原生 Paho client，需要 QoS/retain/异步等细粒度控制时用 |
| `Exist(name) bool` | 连接是否已建立 |
| `Close()` | 逐个 `Disconnect(200)`（200ms /quiesce），并打日志 |

### 3.3 包级函数

| 函数 | 说明 |
|------|------|
| `New() error` | 进程启动时调用一次：`GlobalMQTT = NewManager()`，若 `Internal_Mqtt.enable` 则建立 `internal` 连接；`broker` 为空只打 WARN 并返回 nil |
| `InitGlobal()` | 只建空管理器、不连接（供测试或自定义装配） |
| `CloseAll()` | 优雅退出调用；`GlobalMQTT == nil` 时安全返回 |
| `Send(name, topic, data) error` | `Publish(topic, QoS=1, retain=false, data)` 并**同步 `token.Wait()`** |
| `Subscribe(name, topic, cb) error` | 登记订阅 + 订阅，3s 超时；超时/出错都**回滚订阅记录** |
| `Subscribe_Close(name, topic) error` | 取消订阅并从记录表移除 |

---

## 四、重连与补订阅

`Add()` 注册了三个 Paho 回调，这是本包真正的价值所在：

```
SetConnectionLostHandler   → 日志：MQTT[internal] 连接断开: %v
SetReconnectingHandler     → 日志：MQTT[internal] 尝试重连...
SetOnConnectHandler        → 连接成功 且 逐条重新订阅 subscriptions 里的全部主题
```

要点：

1. **订阅信息必须存在 `instance.subscriptions` 里**，重连后才可能自动补订阅；
   绕过 `Subscribe()` 直接用 `GetClient().Subscribe()` 的订阅，重连后会静默丢失。
2. 补订阅在 `OnConnect` 回调里**同步执行**，每条 `WaitTimeout(5s)`；
   主题很多时会拖慢重连完成时间，属已知取舍。
3. 每次补订阅前先用 `RLock` 拷贝一份快照再遍历，避免与业务侧新增订阅并发写 map。
4. 订阅回调外层有 `recover()`：业务 panic 只打 `订阅回调 panic: %v`，不会带倒整个 MQTT 收件协程。

QoS 固定为 **1**（收发都是）；未开放 QoS/retain 参数，需要时请改 `Send`/`Subscribe` 签名或走 `GetClient()`。

---

## 五、降级策略（与采集可用性相关）

`mqtt.New()` 的失败**不会终止进程**（`main.app()` 里只 `log.Printf("ERROR …")`）：

- broker 挂了 → 本地采集、报警判断、InfluxDB 写入照常；
- 代价是 `internalmqtt.Start()` 拿不到连接，对外 MQTT 主题（写值/运维/监控）不可用；
- `auto_reconnect: true` 时会一直退避重试，恢复后 `OnConnect` 自动补订阅，无需重启。

> 反过来：`Send()` 是同步 `token.Wait()`，broker 不可达时会阻塞到写超时（`write_timeout`）。
> 周期性发布逻辑（status/data）已放在各自 goroutine 里，不要把它挪进采集热路径。

---

## 六、用法示例

```go
// 发布（服务内已封装，这里示意通用写法）
if err := mqtt.Send(mqtt.InternalBrokerName, "/IOT/Collector/0001/result/write", payload); err != nil {
    log.Printf("WARN 回执发布失败: %v", err)
}

// 订阅（内部指令主题：逐个订阅具体主题，不用 base+"/#" 通配，避免收到自己发布的上行回声）
err := mqtt.Subscribe(mqtt.InternalBrokerName, base+"/write", func(name, topic string, data []byte) {
    dispatch(topic, data)   // 见 internalmqtt.subscribe.go
})
```

---

## 七、扩展指引

| 需求 | 做法 |
|------|------|
| 接第二条 broker | 在 `New()` 内按配置再 `GlobalMQTT.Add("<name>", …)`；不要复用 `internal` 名字 |
| 需要 QoS2 / retain | 加一个新包级函数（如 `SendQos`），保持 `Send` 语义不变以免影响既有调用 |
| 需要"离线期间不丢消息" | 用 `GetClient()` + `SetOrderMatters(false)` / 会话保持（`clean_session=false`），或在上层做本地缓冲 |
| 想监听连接状态给前端 | 在 `Add()` 的三个 handler 里追加回调注册（当前只有日志） |

---

## 八、相关文档

- [internal/service/internalmqtt](../../service/internalmqtt/README.md) — 主题定义、指令分发、周期发布
- [configs/config.yaml](../../../configs/config.yaml) — `Internal_Mqtt` 段
- [cmd/server](../../../cmd/server/README.md) — `mqtt.New()` 的启动位置与退出时的 `CloseAll()`
