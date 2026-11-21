# `internal/repo/redis` — Redis 持久层：令牌会话、AK/SK 凭据缓存、nonce 防重放

上级文档：[服务总览 README](../../../README.md)

本包只有 **一套现役 API（`Auth_*` 前缀）**，服务对象是鉴权中间件：登录态令牌、账号令牌索引、HMAC 验签凭据缓存、nonce 防重放、AK/SK 登陆日志节流。其余 `Refresh_Token_*` / `Access_Token_*` / `Authority_User_*` / `Api_*` / `Write_Key_list` / `Read_Key*` / `Redis_scanKeys` / `Queue_struct` 全部**没有任何外部调用方**，属历史遗留。

> 依赖方向：`handler` / `service/internalmqtt` → `repo/redis`。本包只依赖 `internal/config`。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `redis.go` | 78 | `var Rdb *redis.Client`、`Init_rdb(Addr, Passwd, Database)`、包级 `init()`（连不上即 `panic`）、`Close()`、`Redis_scanKeys`（遗留） |
| `logic.go` | 666 | **439 行起为现役 `Auth_*`**；前 438 行是遗留令牌/权限/批量 KV 函数 |
| `queue.go` | 114 | `Queue_struct` 简单列表队列（LPUSH / BRPOP / Lua 全量取空），**当前无人使用** |

---

## 二、⚠️ 最容易踩的坑

1. **`init()` 连不上 Redis 直接 `panic("连接失败: …")`，服务起不来。**
   与 MySQL 同样的启动强依赖。`Init_rdb` 里 `Ping` 失败是 `panic` 而不是返回 error，所以**没法做"Redis 可选"的降级**。若要在无 Redis 环境跑测试，必须隔离掉本包。
   另外 `Init_rdb` **没有设置任何连接池参数**（走 go-redis 默认 `PoolSize = 10 * GOMAXPROCS`），高并发 HMAC 验签下如果发现连接饥饿，需要在此处补 `PoolSize`/`MinIdleConns`。

2. **现役函数只有一组，别用 IDE 自动补全选到遗留函数。**
   `logic.go` 前半段的 `Access_Token_Add` / `Refresh_Token_Add` 与现役的 `Auth_Access_Token_Add` / `Auth_Refresh_Token_Add` 名字极像，但**键空间完全不同**（`Access_Token:%s` vs `auth:access_token:%s`），混用会造成"写入成功但永远读不到"。遗留函数已全量确认无调用方，可安全删除。

3. **`Get` 未命中时会把 `redis.Nil` 当错误打日志。**
   遗留的 `Access_Token_Query` / `Refresh_Token_Query` 等都是 `val, err := Rdb.Get(...); if err != nil { log.Printf("ERROR %v", err) }`，键不存在（正常业务）也会刷一条 ERROR。现役 `authTokenGet` 已用 `errors.Is(err, goredis.Nil)` 区分"不存在（`ok=false, err=nil`）"与"Redis 故障（`err!=nil`）"。

4. **令牌 value 解析失败一律视为无效、强制重新登录。**
   `authTokenUnmarshal` 要求 JSON 可解析 **且** `Access_Key != ""` **且** `User_Id != 0`。因此：旧格式（value 是裸 access_key 字符串）的令牌、以及没有 `user_id` 的历史令牌，升级后**全部失效，用户需重新登录一次**。这是有意设计（避免"半新半旧"的会话），但要提前告知运维。

5. **`Auth_Account_Token_Logout` 是"账号级全端下线"，粒度比想象中大。**
   它 `SMembers` 出该账号所有令牌 key 后一次性 `DEL`，并删掉索引集合本身。登录时检测到 IP 变化就会调用它（见 [handler README](../../handler/README.md)），所以**同一账号在两个不同网络间切换会把两端都踢下线**，不是"只踢旧端"。

6. **索引集合的 TTL 是"续期到不小于令牌 TTL"，不是取最大值。**
   `authTokenIndexAdd` 每次都 `Expire(setKey, ttl)`。签发顺序若为「先长 TTL 的 refresh、再短 TTL 的 access」，集合会被**缩短**到 access 的 TTL。实际影响：集合比里面的某些令牌 key 先过期后，"IP 变化全端下线"会漏掉那些已不在集合中的令牌（它们仍活到自己的 TTL 才自然过期）。改造建议：写入前先 `TTL(setKey)`，仅在新 ttl 更大时续期。

7. **`Auth_Nonce_Add` 的 key 按 `access_key` 隔离，粒度是"每 AK 每 nonce"。**
   `auth:nonce:<ak>:<nonce>`。好处是不同 AK 撞 nonce 不互相影响；代价是**大量 AK 时 nonce key 数量 = 请求数**（受 TTL 收敛）。Redis 若被当持久化实例用（AOF/RDB 落盘），这些短命 key 会带来无谓写放大，建议用纯内存实例或独立 DB 号。

8. **`Queue_struct` 不满足 `Queue_interface`，且 `Read()` 在空队列会 panic。**
   - `Queue_interface` 声明 `New(key string) error`，实现是 `func (c *Queue_struct) New(key string)`（无返回值）→ 接口断言不成立，这个接口目前**只能当文档看**。
   - `Read()` 先取 `result[0]` 再判 `err`；`BRPop` 超时（100ms 内无数据）时 go-redis 返回 `nil, redis.Nil`，`result` 是空切片 → **`index out of range` panic**。若将来启用队列，务必先判 `errors.Is(err, goredis.Nil)`。
   - `Read_list()` 用 Lua 原子 `LRANGE + DEL`，并用了带检查的类型断言（注释里明确说明"裸断言会 panic 并干掉消费协程"）——这是 `queue.go` 里唯一稳妥的地方。

---

## 三、Key 空间一览

| 前缀 | 类型 | Value | TTL | 写入方 |
|---|---|---|---|---|
| `auth:access_token:<token>` | String | `Auth_Token_Info` JSON | 账号 `Access_Token_TTL`（默认 `15m0s`） | `Auth_Access_Token_Add` |
| `auth:refresh_token:<token>` | String | `Auth_Token_Info` JSON | 账号 `Refresh_Token_TTL`（默认 `168h0m0s`） | `Auth_Refresh_Token_Add` |
| `auth:account_tokens:<access_key>` | Set | 该账号所有令牌 **Redis key** | 同上传入的 ttl | `authTokenIndexAdd` |
| `auth:apikey_creds:<access_key>` | String | `Auth_ApiKey_Creds` JSON | `ApiKey_Creds_TTL = 30m`（handler 侧常量） | `Auth_ApiKey_Creds_Set` |
| `auth:nonce:<access_key>:<nonce>` | String（`SetNX`，值 `1`） | 占位 | `Auth_Nonce_TTL = 120s` | `Auth_Nonce_Add` |
| `auth:aksk:touch:<access_key>` | String（`SetNX`，值 `1`） | 占位 | `AkSk_Login_Log_Window = 10m` | `Auth_AkSk_Touch_Throttle` |

```go
// Auth_Token_Info：令牌绑定信息，用于会话一致性校验
type Auth_Token_Info struct {
    User_Id    uint      `json:"user_id"`     // user_account.Id，请求期直接取用免查库
    Access_Key string    `json:"access_key"`
    IP         string    `json:"ip"`           // 签发时客户端 IP
    User_Agent string    `json:"user_agent"`   // 签发时 UA
    Issued_At  time.Time `json:"issued_at"`    // RFC3339Nano
}

// Auth_ApiKey_Creds：HMAC 热路径的验签凭据缓存，避免每请求查 MySQL
type Auth_ApiKey_Creds struct {
    Id       uint   `json:"id"`
    Secret   string `json:"secret"`
    Enable   int    `json:"enable"`     // 1=启用 0=禁用
    ExpireAt int64  `json:"expire_at"`  // Unix 秒；0=永不过期
}
```

---

## 四、现役 API

### 4.1 令牌（Bearer 通路）

```go
func Auth_Access_Token_Add(token string, info Auth_Token_Info, ttl time.Duration) error
func Auth_Access_Token_Get(token string) (Auth_Token_Info, bool, error)
func Auth_Access_Token_Del(token string) error

func Auth_Refresh_Token_Add(token string, info Auth_Token_Info, ttl time.Duration) error
func Auth_Refresh_Token_Get(token string) (Auth_Token_Info, bool, error)
func Auth_Refresh_Token_Del(token string) error

func Auth_Account_Token_Logout(accessKey string) error   // 全端下线
```

`Get` 的三态语义（**所有调用方都必须按这个约定分支**）：

| 情况 | `info` | `ok` | `err` | 调用方动作 |
|---|---|---|---|---|
| 命中且格式合法 | 有效值 | `true` | `nil` | 继续校验 IP/UA |
| 不存在 / 已过期 / JSON 损坏 / 缺 `user_id` | 零值 | `false` | `nil` | 返回 `401 未登录`，让前端跳登录 |
| Redis 故障 | 零值 | `false` | 非 `nil` | 返回 `521 服务异常`，**不能当作未登录**（否则 Redis 抖动会全站踢人） |

`Del` 会顺带 `SRem` 把 key 从账号索引集合摘除（读到绑定信息才摘，读不到也照常 `DEL`），因此刷新轮换后不会留下"僵尸索引项"。

### 4.2 AK/SK 凭据缓存（HMAC 通路）

```go
func Auth_ApiKey_Creds_Set(accessKey string, c Auth_ApiKey_Creds, ttl time.Duration) error
func Auth_ApiKey_Creds_Get(accessKey string) (Auth_ApiKey_Creds, bool, error)
func Auth_ApiKey_Creds_Del(accessKey string) error   // AK 删除/禁用/改密时失效
```

这是"读多写极少"的旁路缓存（cache-aside）：`handler/auth.go` 的 `loadApiKeyCreds` 先查 Redis，未命中回源 MySQL 再 `Set` 30 分钟。**写失败被有意忽略**（`_ = redis.Auth_ApiKey_Creds_Set(...)`），因为不影响本次验签。数据损坏（JSON 解析失败）返回 `ok=false, err=nil`，触发回源重建。

### 4.3 防重放与登陆日志节流

```go
func Auth_Nonce_Add(accessKey, nonce string, ttl time.Duration) (fresh bool, err error)
func Auth_AkSk_Touch_Throttle(accessKey string, ttl time.Duration) (first bool, err error)
```

两者都基于 `SetNX`，语义相反，**读代码时极易看反**：

- `Auth_Nonce_Add` → `fresh=true` 表示 nonce **首次出现**，放行；`false` 表示重放，拒绝。
- `Auth_AkSk_Touch_Throttle` → `first=true` 表示窗口内**第一次使用**，调用方才写一条登陆日志；`false` 表示已记过，跳过。

Redis 故障时两者都返回 `false` + `err`。注意 handler 侧对 throttle 的调用是 `if first, err := …; err == nil && first`——**Redis 挂了就不写登陆日志，而不是每次都写**，这是有意的（避免故障期间日志爆量）。

---

## 五、遗留 API（无调用方，仅供参考）

| 分组 | 函数 | 键 | 备注 |
|---|---|---|---|
| 用户令牌 | `Access_Token_Add/Query`、`Refresh_Token_Add/Query` | `Access_Token:%s`、`Refresh_Token:%d:%s` | 早期"只记 id"的令牌模型，已被 `Auth_*` 取代 |
| 接口账号令牌 | `Api_Access_Token_Add/Query`、`Api_Refresh_Token_Add/Query` | `Api_Access_Token:%s`、`Api_Refresh_Token:%d:%s` | 含 `Allow_Ip` 字段，思路被 `Auth_Token_Info.IP` 继承 |
| 用户权限 | `Authority_User_Query/Add/Del_Value` | Hash：field key=`<User_Id>`，field=`Authority_User:<主题>` | **已被 RBAC 完全取代**（见 [service/rbac](../../service/rbac/README.md)）；注意它用**裸数字做 Redis key**，可能与业务 key 撞名 |
| 密码错误锁定 | `User_Passwd_Input_Error_Record` / `_Record_Del` | List `User_Passwd_Input_Error:%d` | 当前限流逻辑未接入 |
| 批量 KV | `Write_Key_list([]KeyValue)`、`Read_Key`、`Read_Key_list` | 任意 | `Write_Key_list` 用 `MSET` + 逐个 `EXPIRE`，**非原子**：中途失败会留下"已写入但无 TTL"的永久 key |
| 键扫描 | `Redis_scanKeys(ctx, rdb, pattern, count)` | — | 游标遍历，>1000 轮判定为死循环并报错；唯一显式接收 `*redis.Client` 参数的函数 |
| TTL 常量 | `Default_Expiration_Second` / `Short_Expiration_Second` | — | **两者当前都是 `1h`**（"短"与"默认"没有区分度），是遗留代码里的一处失效设定 |
| 队列 | `Queue_struct` + `Queue_interface` | List，key 由 `New(key)` 指定 | 见坑 8 |

---

## 六、优雅退出

```go
// cmd/server/main.go
step(ctx, "Redis 连接", func() error { return redis.Close() })
```

`Close()` 对 `Rdb == nil` 安全返回，所以 `init()` 之前调用也不会 panic。`Close` 只关连接池，不删任何 key —— 令牌寿命完全交给 Redis TTL。

---

## 七、排障速查

| 现象 | 定位 | 处理 |
|---|---|---|
| 启动 `panic: 连接失败: …` | `Init_rdb` 的 Ping | 检查 `configs/config.yaml` 的 `redis.ip/post/passwd/database` |
| 升级后所有人都要重新登录 | 坑 4（旧 value 被判无效） | 预期行为；需要平滑可加一次兼容读 |
| 一端切换网络导致另一端掉线 | 坑 5（`Auth_Account_Token_Logout`） | 属既定安全策略；若要"只踢旧端"需改成按 UA 精确匹配 |
| Redis 故障时大量用户被踢到登录页 | 调用方把 `err!=nil` 当成"未登录" | 按 4.1 的三态表区分 `521` 与 `401` |
| HMAC 请求随机被拒"重放" | `Auth_Nonce_Add` 窗口 | 客户端时钟偏移 > ±60s，或同一 nonce 被复用；检查 `Auth_Nonce_TTL` 与时间窗 |
| 改了 AK 的 SK 但旧签名仍能用 | 凭据缓存 30 分钟未失效 | 确认改密路径调了 `Auth_ApiKey_Creds_Del` |
| Redis 里出现大量裸数字 key | `Authority_User_*`（遗留） | 若日志里没有对应调用说明已停用，可 `SCAN` 后清理 |

---

## 八、扩展指引

**新增一类会话缓存**：沿用 `Auth_` 前缀 + `auth:` 键命名空间；`Get` 必须返回 `(值, ok, err)` 三态并用 `errors.Is(err, goredis.Nil)` 区分未命中与故障；所有 key **必须带 TTL**（本包没有"永久 key"的约定）。

**需要真正的队列/锁**：不要复用 `queue.go`（见坑 8），应新写文件并显式判 `redis.Nil`；分布式锁建议改用 Lua `SET key val NX PX` + 值比对的释放脚本。

**改令牌绑定信息结构**：`Auth_Token_Info` 加字段是安全的（JSON 向后兼容）；**删字段或改 `json` tag 会让全部现存令牌失效**（坑 4）。

---

## 九、相关文档

- [服务总览 README](../../../README.md)
- [internal/config](../../config/README.md) — `redis.ip/post/passwd/database` 来源
- [internal/handler](../../handler/README.md) — 双通路鉴权与 `Auth_*` 的唯一调用方
- [internal/service/internalmqtt](../../service/internalmqtt/README.md) — MQTT 登录会话复用 `Auth_Access_Token_Add`
- [internal/service/rbac](../../service/rbac/README.md) — 取代 `Authority_User_*` 的新权限模型
