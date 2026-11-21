# `internal/service/configsync` — 配置服务对接：拉取中心侧 Drive / Point 配置并覆盖本地 MySQL

上级文档：[服务总览 README](../../../README.md)

本包是采集器与**独立部署的 Iot-Config-Service** 之间唯一的 HTTP 通路。仅在启动时被调用一次（`main.go:171-173`），把中心下发的驱动 / 点位配置同步进本地 MySQL，然后才启动驱动。它是一个"边缘侧被管理"的可选能力，**默认关闭**。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `ApiConfig.go` | 325 | 4 个远程接口的封装：登录取 token、查配置更新时间、拉驱动配置、拉点位配置；URL 常量、响应泛型解析、401 自动刷新重试中间件 |
| `sync.go` | 98 | 同步编排：`IsConfigUpdated`（时间戳比对）+ `SyncConfigFromService`（拉取 → `__Sync` 落库 → 记录同步时间）；同步水位线存在 `APP_Config` 表 |
| 合计 | 423 | — |

对外只暴露 4 个符号：`IsConfigUpdated`、`SyncConfigFromService`（均被 `cmd/server/main.go` 调用）、`Collector_GetBasicAuth`、`FetchWithRetry` 等其余为包内实现或死代码（见坑 7）。

---

## 二、⚠️ 最容易踩的坑

1. **`configs/config.yaml` 里根本没有 `Config_Service` 段 → 本包整条链路默认不执行。**
   `main.go:171` 先判 `config.Config.Config_Service.Enable`，缺段即 `false`。要启用必须**手工补全整段**（注意 yaml 键是小写驼峰，与结构体 tag 一致）：

   ```yaml
   Config_Service:
     enable: true
     address: 127.0.0.1:8101     # 不带协议头会自动补 http://
     setTimeout: 10s             # 缺省 10s
     setRetryCount: 1            # 缺省 1（即"401 重试一次"）
     setRetryWaitTime: 3s        # 缺省 3s
   ```

   三个 `set*` 的默认值在 `config.go:157-159` 硬编码，并在归一化阶段（`config.go:347-353`）把 0 值补回默认——所以写 `0` 与不写等价。

2. **同步是"以中心为准的整表覆盖"，本地手改的驱动/点位会被抹掉。**
   `SyncConfigFromService` 调的是 `mysql.Drive_Config__Sync` / `mysql.Point_Config__Sync`，语义是**按 Id 比对后执行新增 / 更新 / 删除**（见 [repo/mysql README](../../repo/mysql/README.md)）。中心没有的行，本地会被删。

3. **`Drive_Config__Sync` 不带 `Name` 字段** —— 一旦启用配置同步，从中心同步进来的驱动在本地表里 `Name` 会是空值。这是 mysql 层已记录的缺陷（构造 `Drive_Config_Add_type{Id, Type, Config}` 时漏了 `Name`），本包是它的**唯一实际触发路径**。详见 [repo/mysql README 的坑清单](../../repo/mysql/README.md)。

4. **只同步 Drive 与 Point，不同步 Alarm / History / RBAC / 账号。**
   报警与历史配置必须仍然在本机维护。误以为"中心改了就全生效"会导致排障方向错误。

5. **token 是懒取的：第一次请求必然带空 token，先吃一个 401。**
   `collector_AccessToken` 初值为空串（`ApiConfig.go:54`），`IsConfigUpdated → Collector_ConfigUpdate` 发送时中间件塞的是 `Collector_Token: ""`，服务端回 401 → 重试钩子里才调 `Collector_GetBasicAuth` 换新 token → 再重试一次成功。功能正确，但每次冷启动都多两个往返，且日志里会出现一次 401。想省掉就在同步前先调 `Collector_GetBasicAuth()`。

6. **重试次数 = `SetRetryCount`，而它同时用于"网络异常重试"。**
   `AddRetryConditions` 里 `if err != nil { return true }`（`ApiConfig.go:111-115`、`164-169`、`220-225`），网络错误也会重试。默认 `1` 次重试 × `SetTimeout 10s` + `SetRetryWaitTime 3s` ≈ 单个接口最坏阻塞 23s；三个接口串起来最坏约 1 分钟。**这段时间发生在驱动启动之前，进程看起来像卡住**。

7. **`FetchWithRetry` / `refreshToken` / `main()` 是三段死代码，其中还有一个叫 `main` 的函数。**
   `ApiConfig.go:263-325`：`refreshToken()` 是个把 token 硬写成 `"new_token_xxxx_xxxx"` 的 TODO 占位；`FetchWithRetry()` 只被 `main()` 调用；而非 `main` 包里的 `func main()` 永远不会被执行。三者互相引用形成孤岛，读代码时容易误以为有通用请求工具可用。**不要复用 `FetchWithRetry`**，它的鉴权头是 `Authorization: Bearer`，与配置服务实际约定的 `Collector_Token` 不同。

8. **`collector_AccessToken` 无锁保护。**
   包级裸 `string`，写在 `Collector_GetBasicAuth`（可能发生在 resty 重试钩子协程里），读在请求中间件。当前只在启动单线程路径使用所以不触发，但如果以后加入"周期性同步"或并发拉取，就是数据竞争。

9. **登录凭证直接复用 `APP.Label` / `APP.Passwd`。**
   `ApiConfig.go:74`：`base64("<label>:<passwd>")` 放在自定义头 `Collector_Basic` 里。既不是标准 `Authorization: Basic`，也只是 base64 不是加密——**明文 HTTP 下等于裸奔**（而 `configServiceBaseURL` 默认补的就是 `http://`）。生产必须配 `https://` 地址或走内网。

10. **同步失败不阻断启动。**
    `syncConfigOnStartup` 把所有错误降级为日志（`WARN 配置更新检查失败` / `ERROR 配置同步失败`），随后照常 `app()`。也就是说中心挂了，采集器会**用上一次落库的本地配置继续跑**——这是设计意图，但也意味着"同步失败"不会有任何退出码体现，只能靠日志。

11. **同步时间戳写入失败只 `WARN`，会导致下次启动重复同步。**
    `sync.go:91-94`：`setLastConfigUpdateTime` 失败不影响返回值。水位线存在 `APP_Config` 表的 `ConfigUpdateTime` 键（注意 mysql 层这个函数就叫 `APP_Config__GetByKet` / `APP_Config__SetByKet`，`Ket` 是历史拼写，见 [repo/mysql README](../../repo/mysql/README.md)）。

12. **时间比对用的是 `After`，同秒即视为"无更新"。**
    `sync.go:58`：`!configUpdateTime.After(lastSyncTime)` → 相等也算未更新。中心在同一秒内多次改动会被漏掉。存的是 `time.RFC3339`（**秒级精度**），精度损失是双重的。

---

## 三、远程接口一览

| 常量 | 路径 | 方法 | 鉴权头 | 响应 `Data` 类型 | 封装函数 |
| --- | --- | --- | --- | --- | --- |
| `url_collector_GetBasicAuth` | `/api/app/v1.0/login` | POST | `Collector_Basic: base64(label:passwd)` | `string`（token） | `Collector_GetBasicAuth` |
| `url_collector_ConfigUpdateTime` | `/api/app/v1.0/collector/ConfigUpdateTime` | GET | `Collector_Token` | `time.Time` | `Collector_ConfigUpdate` |
| `url_collector_Drive_Config__Query` | `/api/app/v1.0/collector/drive/config/query` | POST | `Collector_Token` | `[]mysql.Drive_Config_Query_type` | `Collector_Drive_Config__Query` |
| `url_collector_Point_Config__Query` | `/api/app/v1.0/collector/point/config/query` | POST | `Collector_Token` | `[]mysql.Point_Config_Query_type` | `Collector_Point_Config__Query` |

响应统一是标准信封，`parseResponseBody[T]` 只做 `json.Unmarshal` 到：

```go
type body_struct_Resp[T any] struct {
    Code      int       // ⚠️ 本包从不在 HTTP 200 之后二次校验 Code
    Msg       string
    Data      T
    Timestamp time.Time
}
```

> 四个函数都只判 `resp.StatusCode() != 200`，**不看 `Code`**。如果配置服务像本服务一样把业务错误放在 `Code`（417/520/…）而 HTTP 仍返 200，这里会被当成成功，`Data` 为 `null` → 反序列化成空切片 → `__Sync` 拿到空列表 → **把本地驱动/点位全部删掉**。接入前务必确认对端的错误约定。

HTTP 客户端为 `resty.dev/v3`，每个函数**各自 `resty.New()` 建一个新客户端**（无连接池复用），统一挂三件套：请求中间件带 token、重试条件（err≠nil 或 401）、重试钩子（401 时刷新 token）。

地址处理由 `configServiceBaseURL` 完成（`ApiConfig.go:26-35`）：空 → 报错提示查 `config.yaml`；无 `http://`/`https://` 前缀 → 补 `http://`；尾部 `/` 会被 `TrimRight` 去掉再拼路径。

---

## 四、核心流程

### 4.1 启动时同步（唯一调用路径）

```
main.go:171  if config.Config.Config_Service.Enable
   └─→ syncConfigOnStartup()                                    [main.go:208-221]
         ├─→ IsConfigUpdated()                                  [sync.go:40-64]
         │      Enable==false → 直接返回零值
         │      Collector_ConfigUpdate()  ←─ 空 token → 401 → Collector_GetBasicAuth → 重试
         │      getLastConfigUpdateTime() → APP_Config__GetByKet("ConfigUpdateTime") → RFC3339 解析
         │      服务端时间 <= 本地水位线 → 打 INFO 配置未更新，返回零值
         │      否则返回服务端时间
         ├─ configUpdateTime.IsZero() → return（不覆盖本地）
         └─→ SyncConfigFromService(t)                           [sync.go:69-98]
                Collector_Drive_Config__Query()
                Collector_Point_Config__Query()
                mysql.Drive_Config__Sync(driveConfigs)   ← 按 Id 增 / 改 / 删（坑 3 的 Name 缺失在此）
                mysql.Point_Config__Sync(pointConfigs)
                setLastConfigUpdateTime(t)               ← 失败仅 WARN
                INFO 配置服务同步完成：驱动 N 条，点位 M 条
   ↓ 之后才 app() → driver.Start(collector.Collection_Publisher)
```

同步**刻意放在驱动启动之前**，所以中心改过的驱动配置本轮就能生效，无需 `driver.DriveResetConfig`。

### 4.2 水位线存储

`APP_Config` 表，键名常量 `appConfigKey_ConfigUpdateTime = "ConfigUpdateTime"`，值是 `time.RFC3339` 字符串。首次运行读不到（空串）→ 返回零时间 → 只要服务端时间非零就必定触发一次全量同步。

---

## 五、排障速查

| 日志 / 现象 | 含义 | 处理 |
| --- | --- | --- |
| 完全没有本包任何日志 | `Config_Service.Enable` 为 false（默认） | 正常，见坑 1 |
| `Config_Service.Address 未配置，请检查 config.yaml` | `enable: true` 但没写 `address` | 补 `address` |
| `INFO 获取基础认证凭证成功，token: xxx` | 401 后刷新 token 成功 | 属正常冷启动首跳，见坑 5；**这行把 token 原样打进日志，注意日志脱敏与访问控制** |
| `ERROR 刷新token失败：...` | `/login` 也失败 | 核对 `APP.Label` / `APP.Passwd` 与配置服务是否一致 |
| `INFO 配置未更新（上次同步时间: ...）` | 水位线判定无变更 | 中心改了配置却不生效 → 检查中心是否更新了 `ConfigUpdateTime`（坑 12 的同秒情况） |
| `WARN 配置更新检查失败: ...` | 连通性 / 解析 / 鉴权异常 | 已降级为用本地配置启动（坑 10） |
| `驱动配置响应解析失败` | 对端返回不是标准信封 | 抓包看原始 body；注意坑 1 的 `Code` 不校验 |
| `INFO 配置服务同步完成：驱动 0 条，点位 0 条` 随后本地被清空 | 对端返 200 + 空 Data | 高度可疑，见坑 3/信封 `Code` 不校验 |
| 启动卡住约 1 分钟才继续 | 三个接口 × (10s 超时 + 1 次重试 + 3s 等待) | 见坑 6；调低 `setTimeout` |
| 同步后驱动名称全变空 | `Drive_Config__Sync` 漏 `Name` | 坑 3，需在 mysql 层修 |

---

## 六、扩展指引

**新增一类要同步的配置（例如 Alarm）**
1. 在 `ApiConfig.go` 加 URL 常量 + 一个与 `Collector_Drive_Config__Query` 同构的函数（复制中间件三件套）；
2. 在 `mysql` 层确认有对应的 `Xxx__Sync`；
3. `SyncConfigFromService` 里追加一次调用；
4. **关键**：报警/历史有内存缓存，同步完必须调 `collector.Alarm_Config__Reload()` / `collector.History_Config__Reload()`（见 [collector README](../collector/README.md)），否则中心改动不生效。

**改造建议（按收益排序）**
1. 先修 `mysql.Drive_Config__Sync` 的 `Name` 缺失（坑 3）；
2. 在 `parseResponseBody` 之后统一校验 `Code != 200` 即报错，并对"空列表覆盖"加保护阈值（如 Data 为空且本地非空时拒绝执行 `__Sync`）；
3. 把 4 个函数重复的 resty 客户端构造 + 中间件 + 重试逻辑抽成一个 `doRequest[T](method, path) (T, error)` 泛型辅助，同时消掉死代码 `FetchWithRetry` / `refreshToken` / `main`（坑 7）；
4. 启动早期主动调一次 `Collector_GetBasicAuth()`（坑 5）；
5. `collector_AccessToken` 换成 `atomic.Value` 或加锁（坑 8）；
6. 水位线改用毫秒精度（`RFC3339Nano`）或改用单调递增版本号（坑 12）。

---

## 七、相关文档

- [服务总览 README](../../../README.md)
- [cmd/server](../../../cmd/server/README.md) — `syncConfigOnStartup` 的位置与"先同步后启动驱动"的时序
- [internal/config](../../config/README.md) — `Config_Service` 各字段的默认值与归一化规则
- [internal/repo/mysql](../../repo/mysql/README.md) — `Drive_Config__Sync` / `Point_Config__Sync` / `APP_Config__GetByKet` 的实现与已知缺陷
- [internal/service/collector](../collector/README.md) — 若扩展同步范围，需要触发的 `*_Reload`
- [internal/service/driver](../driver/README.md) — 同步结果的实际消费者（启动时按 `Drive_Config` 建实例）
- [internal/handler](../../handler/README.md) — 本地手工改配置的另一入口（与中心同步会冲突，见坑 2）
