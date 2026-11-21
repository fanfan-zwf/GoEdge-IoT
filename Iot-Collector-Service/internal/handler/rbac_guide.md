# RBAC 鉴权与权限校验体系详解

> 目标：把 `internal/handler` 这套 **认证(Auth) + 权限(RBAC)** 链路彻底讲透——每一层做什么、函数签名长什么样、数据从哪来到哪去、复杂度到底堆在哪、哪些能砍哪些砍不动、以及三种简化方向的改前/改后对照。
> 读者：想读懂、排障、或重构这套权限逻辑的人。
> 版本：2026.11，对应"id 前置进令牌 + AK/SK 凭据缓存 + 判定结果按主题缓存"三处改造之后的现状。

---

## 目录

- [0. 一分钟速览](#0-一分钟速览)
- [1. 请求全链路](#1-请求全链路)
- [2. 文件与分层地图](#2-文件与分层地图)
- [3. 认证层 Auth](#3-认证层-auth)
- [4. 权限层 RBAC：判定模型](#4-权限层-rbac判定模型)
- [5. 主题匹配语法（含判定真值表）](#5-主题匹配语法含判定真值表)
- [6. 判定结果缓存](#6-判定结果缓存)
- [7. 接口级 vs 实例级](#7-接口级-vs-实例级)
- [8. 实例级授权逐 kind 走查](#8-实例级授权逐-kind-走查)
- [9. 数据模型（7 张表 + 加载 SQL）](#9-数据模型7-张表--加载-sql)
- [10. 错误码与排障](#10-错误码与排障)
- [11. 简化候选（改前/改后对照）](#11-简化候选改前改后对照)
- [12. 常见改动落点速查](#12-常见改动落点速查)

---

## 0. 一分钟速览

一个 HTTP 请求要过 **4 层中间件**（`server.go`）：

```go
r.Use(allowAllCors())     // 跨域
r.Use(Response_Use())     // 统一 JSON 响应
r.Use(Auth_Middleware())  // 认证：你是谁？
r.Use(Rbac_Middleware())  // 权限：你能不能干？
```

| 概念 | 一句话 |
|---|---|
| **认证 Auth** | 把"凭证"翻译成"主体身份"：`{主体类型, 主体id}` 写进 Gin 上下文。前端 = Bearer token，接口 = HMAC 签名。 |
| **权限 RBAC** | 拿"主体身份 + 请求路径(=权限主题)"问判定引擎：允许还是拒绝。**默认拒绝**。 |
| **接口级** | 一个布尔：能不能调这个接口（`/config/drive/query`）。 |
| **实例级** | 一批 id：这个接口下我能看/能改**哪几行**（drive 5、7、9 里的哪几条）。← 复杂度真正来源 |
| **fail-closed** | 新增接口只要落在 `/api/v1.0/*` 下就自动受保护；**忘配权限 = 谁都进不去**（不是谁都能进）。 |
| **判定结果缓存** | Redis 按"一个主体 × 一个具体主题 = 一个裁决"缓存，纯 TTL 懒失效（默认 30m）。 |

> **为什么 handler 读起来干净？** 因为判权被全局中间件兜住了，业务 handler 一行权限代码都不用写。不是因为没有鉴权。

---

## 1. 请求全链路

```
                         HTTP 请求
                              │
       ┌──────────────────────▼──────────────────────┐
       │ allowAllCors → Response_Use                  │
       └──────────────────────┬──────────────────────┘
                              │
       ┌──────────────────────▼──────────────────────┐
       │ Auth_Middleware  【认证】                    │
       │  authExemptPath? ── 是 ──▶ 放行(登录/刷新等) │
       │        │否                                    │
       │  有 Authorization: Bearer?                    │
       │   ├─是 通路A 用户: Redis 取 token→User_Id+IP  │
       │   │        IP 变了?→全令牌登出+401            │
       │   │        setLogActor(user, User_Id)         │
       │   └─否 通路B HMAC: loadApiKeyCreds(AK)→SK     │
       │            时间窗/nonce/验签/enable/过期       │
       │            setLogActor(aksk, creds.Id)        │
       └──────────────────────┬──────────────────────┘
                              │  ctx: {LogSource, LogAccount=主体id}
       ┌──────────────────────▼──────────────────────┐
       │ Rbac_Middleware  【权限】                    │
       │  非 /api/v1.0 前缀? ──▶ 放行                 │
       │  缺主体? ──▶ 401                             │
       │  resource = TrimPrefix(path, /api/v1.0)      │
       │  实例级端点? ─是─▶ rbac_instance 接管:        │
       │        缓冲body→算有权id集→注入ctx→无权403    │
       │  否则 接口级: Check{BizUser|ApiClient}        │
       │        err→520 / !allow→403 / allow→Next()   │
       └──────────────────────┬──────────────────────┘
                              │
                    ┌─────────▼──────────┐
                    │ 业务 handler       │
                    │ 需行过滤时读 ctx:   │
                    │ scopeIdsFromCtx /   │
                    │ opIdsFromCtx        │
                    └────────────────────┘
```

**判定时机小结**：接口级"能不能进"在中间件里判完；实例级"能碰哪几行"在中间件算好 id 集合、由 handler 取用。

---

## 2. 文件与分层地图

### 2.1 `internal/handler`（接入层）

| 文件 | 层 | 职责 | 复杂度 |
|---|---|---|---|
| `auth.go` | 认证 | `Auth_Middleware` 双通路；登录/登出/刷新/内部同步；token 与 HMAC 凭据的 Redis 读写 | 中（安全逻辑，必要） |
| `rbac.go` | 权限·接入 | 合并文件（原 rbac_middleware/routes/instance/scope_ctx）：判权入口 + 主题映射 + 实例级 id 授权 + ctx 下发工具 | 中（约 400 行，实例级部分是复杂核心） |
| `point.go`(`valueAllowed`) | 权限·实例级 | 实时读/写值、WS 按点位 id 过滤 | 中 |
| `rbac_admin.go` | 管理面 | 角色/权限字典/三类绑定 CRUD | 中（纯 CRUD） |

> 2026.11 整理：`rbac_middleware/rbac_instance/rbac_routes/rbac_scope_ctx` 四个文件已合并为单个 `rbac.go`；`system_config+system_info→system.go`；`log_api→并入 log.go`。下文按"中间件/路由映射/实例级/scope 工具"分段描述，均指 `rbac.go` 内对应区块。
> 注：下文若仍出现 `rbac_middleware.go` / `rbac_instance.go` 等旧名，指的是合并前原文件，现都在 `rbac.go` 同一文件内。

### 2.2 下层

| 包 | 文件 | 职责 |
|---|---|---|
| `internal/service/rbac` | `rbac.go` | 校验入口 `Check*/Can/ScopeAll/FilterIds/SubjectAuth` + **判定结果 Redis 缓存**（decideOne/decideBatch） |
| `pkg/rbac` | `topic.go` | 纯匹配：切分/校验/占位符/整串正则 |
| `pkg/rbac` | `decide.go` | 纯判定：deny 优先 + 具体度 + match_sort |
| `pkg/rbac` | `scope.go` | 纯 scope：`Can/ScopeAll/FilterIds`（哨兵 "0" 探全量） |
| `internal/repo/mysql` | `rbac.go` | `LoadEffectiveRules`（JOIN+UNION，过滤条件全在 SQL） |
| `internal/repo/redis` | `logic.go` | 令牌会话 `Auth_Token_Info` + HMAC 凭据 `Auth_ApiKey_Creds` 存取 |

> **分层原则**：`pkg/rbac` 是**无 DB、无 config 依赖的纯函数**（可独立 `go test`）；`internal/service/rbac` 负责缓存与降级；`internal/handler` 负责把 HTTP 请求翻译成"主体 + 主题"。

---

## 3. 认证层 Auth

### 3.1 上下文约定（认证写入、权限读取）

认证通过后向 Gin 上下文写三把键，是 Auth 与 RBAC 的**唯一契约**：

| 键 | 类型 | 含义 |
|---|---|---|
| `CtxKeyAccessKey` | `string` | 账号名（access_key），供日志/审计 |
| `CtxKeyLogSource` | `mysql.Log_Source` | 主体来源：`Log_Source_User` / `Log_Source_AKSK` |
| `CtxKeyLogAccount` | `uint` | **主体数字 id**：`user_account.Id` 或 `api_key.Id` |

`Rbac_Middleware` 用 `subjectTypeFromSrc` 把来源映射成 RBAC 主体类型：`Log_Source_User→Rbac_Subject_Biz(1)`，其余 `→Rbac_Subject_Api(2)`。

### 3.2 通路 A · Bearer token（前端用户）

```
Authorization: Bearer {access_token}
```

1. `bearerToken(ctx)` 取令牌；**WebSocket** 端点从子协议位取（不让令牌进 URL / 访问日志）。
2. `redis.Auth_Access_Token_Get(token)` → `Auth_Token_Info{ User_Id, Access_Key, IP, User_Agent, Issued_At }`。
   - 反序列化时 `authTokenUnmarshal` 校验：`Access_Key=="" || User_Id==0` 一律判无效（旧结构无 user_id 的令牌会自然失效一次，强制重登）。
3. `ipChanged(info.IP, 当前IP)` → 不一致则 `Auth_Account_Token_Logout(access_key)` 清掉该账号所有令牌 + 401。
4. `setLogActor(ctx, Log_Source_User, info.User_Id)` —— **请求期不再查 MySQL**。

> ⚠️ 代价（已定稿的"延迟生效"）：不在请求期回查账号状态，管理员**禁用/删除已登录用户**最多等 access 令牌过期（默认 15m）或下次 refresh 才生效。签发点共 3 处都写入了 User_Id：登录、刷新、内部 MQTT 远程签发。

### 3.3 通路 B · HMAC 签名（接口账号 AK/SK）

```
X-AccessKey: {access_key}
X-Timestamp: {unix 秒}          # 时间窗 ±60s
X-Nonce:     {随机串}            # Redis 去重，TTL = 2×窗口
X-Signature: {hex(hmac)}
```

- 验签原文拼接顺序：`timestamp + nonce + path + body`，用 `secret_key` 做 HMAC-SHA256，常量时间比较。
- `loadApiKeyCreds(accessKey)`：**先读 Redis** `auth:apikey_creds:{access_key}`（`{Id,Secret,Enable,ExpireAt}`），未命中或 Redis 故障**回源** `ApiKey__Query_By_Access_Key` 并写回（cache-aside，TTL 30m 固定不续期）。
- `apiKeyCredsDeny(creds)`：`Enable!=1` 或 `ExpireAt!=0 && now>ExpireAt` → 拒。
- `setLogActor(ctx, Log_Source_AKSK, creds.Id)`。
- **写穿**：apikey 增/改/重置、内部同步 AK/SK 成功后 `refreshApiKeyCreds`；删除前 `ApiKey__AccessKeysByIds` 反查 access_key，删库后逐个 `Auth_ApiKey_Creds_Del`（防已删密钥在 TTL 内仍可验签）。

---

## 4. 权限层 RBAC：判定模型

### 4.1 判定入口签名

```go
// internal/service/rbac/rbac.go —— handler 实际调用的入口
func CheckBizUser  (userId uint, resource, action string, params map[string]string) (rbac.Decision, error)
func CheckApiClient(clientId uint, resource, action string, params map[string]string) (rbac.Decision, error)

// 实例级三件套
func Can     (subjectType uint64, subjectId uint, topic, action string) bool
func ScopeAll(subjectType uint64, subjectId uint, base, action string) bool
func FilterIds(subjectType uint64, subjectId uint, base, action string, ids []uint) []uint
```

> `action` 形参**保留只为签名兼容**，不进判定：`t_permission` 无 action 列，`scanRules` 硬置 `Rule.Action="*"`，`compute` 调 `Decide(topic, "*", ...)`。所有 action 恒匹配，等价"纯主题匹配"。

### 4.2 判定核心 `pkg/rbac.Decide`

```go
func Decide(resource, action string, params map[string]string, rules []Rule) Decision
```

流程：

1. **候选** = 对每条规则：`actionMatch`（恒真）且 `Match(topicKey, resource, params)` 命中（内含占位符解析，解析不出即不命中）。
2. 无候选 → `Decision{Allowed:false, Reason:"no_match_default_deny"}`（**默认拒绝**）。
3. 有候选 → 稳定排序，取第一名其 effect 即结果：

| 排序键（高→低） | 说明 |
|---|---|
| ① deny 优先 | `denyRank`：deny=0 排前，allow=1。有一条命中且 deny 就拦（**跨具体度也拦**） |
| ② 具体度 tier | 精确(0) < 单层 `+`/`*`(1) < 尾部 `#`(2) < 整串正则(3)，值小者优先 |
| ③ `match_sort` | 升序，数字小者优先 |
| ④ `PermId` | 升序，保证稳定 |

`Rule` / `Decision` 结构：

```go
type Rule struct {
    PermId uint64; PermCode, TopicKey, Action string
    MatchSort int; Allow bool; Role string   // Role=来源角色码，直接授权为 "direct"
}
type Decision struct {
    Allowed bool; Reason, PermCode string
    HitId uint64; TopicKey string; MatchSort int; Role string  // 溯源，供日志/排障
}
```

---

## 5. 主题匹配语法（含判定真值表）

主题 = 请求路径去掉 `/api/v1.0` 前缀（`resourceTopic`），例 `/api/v1.0/config/drive/query` → `/config/drive/query`。

### 5.1 模式 A：逐段匹配（默认，无反引号）

按 `/` 切段逐段比；段内通配符**必须独占整段**：

| 段写法 | 含义 |
|---|---|
| 字面量 | 与该段完全相等 |
| `+` 或 `*` | 单层通配，恰好匹配**一段**（二者等价） |
| `#` | **仅允许最后一段**，匹配其后**所有段（含零段）** |

**无 `#` 时，模式段数必须与资源段数完全相等**（等长精确）。

判定真值表（`matchSegs`）：

| 模式 topic_key | 资源 resource | 命中? | 原因 |
|---|---|---|---|
| `/config/drive/query` | `/config/drive/query` | ✅ | 逐段全等 |
| `/config/drive/query` | `/config/drive/add` | ❌ | 第 3 段不等 |
| `/config/+/query` | `/config/drive/query` | ✅ | `+` 吃掉 `drive` 一段 |
| `/config/drive/*` | `/config/drive/query` | ✅ | `*` 吃掉 `query` 一段 |
| `/config/drive/#` | `/config/drive/query/5` | ✅ | `#` 吃其余所有段 |
| `/config/rbac/+/del` | `/config/rbac/perm/del/13` | ❌ | 模式 4 段 ≠ 资源 5 段（**等长精确**） |
| `/config/point/query` | `/config/point/query/` | ❌ | 末尾空段 `""` ≠ 缺段 |
| `` `/config/rbac/perm/del/[0-9]+` `` | `/config/rbac/perm/del/13` | ✅ | 反引号内当整串正则 |

### 5.2 模式 B：整串正则（反引号包裹）

主题被 `` ` `` 首尾包裹时，反引号内当一个 Go 正则，对**完整路径串**做隐式锚定 `^(?:...)$`（非前缀匹配）；带 `sync.Map` 编译缓存。要跨段效果（如命中 `/del/13`）必须用正则而非通配符。

### 5.3 占位符 `{userId}/{clientId}/{orgId}`

- 校验/试编译时先把 `{name}` 替成字面量 `0`（避免 RE2 把 `{}` 当量词报假错）。
- **运行时由 `injectSubject` 用认证后的真实 id 强制覆盖**，调用方在 body 里伪造 `{userId}` 无效。
- 任一占位符取不到值 → `ReplacePlaceholders` 返回 `ok=false` → 该规则判**不命中**（安全兜底）。

---

## 6. 判定结果缓存

### 6.1 模型

| 维度 | 取值 |
|---|---|
| 粒度 | **一个主体 × 一个具体主题 = 一个裁决**（不是规则快照） |
| Key | 业务 `rbac:user:{userId}:{主题}`；API `rbac:api:{clientId}:{主题}` |
| Value(JSON) | `cacheValue{ allow, topic(命中规则模式), role(来源角色), sort(优先级) }` |
| 回源 | miss → `LoadEffectiveRules` → `Decide` → `SETEX` 写回 |
| 失效 | **纯 TTL 懒失效**，TTL = `config.RBAC.TTL`（默认 30m）；allow/deny 都缓存；命中**不**续期；**无主动失效** |
| 降级 | Redis Get/MGet 出错视为未命中、Set/Pipeline 忽略错误 → 直连 MySQL，**正确性不受影响**，仅暂时失去加速 |

### 6.2 单查 vs 批量

```go
// decideOne：单主题。GET 命中即用；miss 回源一次规则 + Decide + SETEX
// decideBatch：多主题（FilterIds 用）。MGET 取命中；miss 集合仅回源一次规则，
//              逐个 Decide，再用 Pipeline 一次性 SETEX 写回（避免 N 次往返）
```

> `SubjectAuth`（登录回填前端 roles/perms）低频，**直接回源 MySQL、不经判定缓存**——它要枚举主体全部权限，不是单主题裁决。

---

## 7. 接口级 vs 实例级

| | 接口级 | 实例级 |
|---|---|---|
| 问题 | 能不能**调这个接口** | 这个接口下能看/改**哪几行** |
| 表达 | 一个布尔 | 一组 id（有权子集） |
| 位置 | `rbac_middleware.go` | `rbac_instance.go` + handler 读 ctx |
| 覆盖接口 | 绝大多数（rbac_admin / config 的 add 等 / tagtree / user / apikey…） | drive/point/alarm/history 的增删改查重启、alarm-data、实时读写值/WS |
| 引擎函数 | `Check*` | `ScopeAll` / `FilterIds` / `Can` |

### 7.1 哨兵 "0" 探"全量"

实例级核心难点：项目是**等长精确**匹配，而列表请求天然**没有 id 段**，无法用一条布尔表达"可见全部 vs 仅显式集合"。解法（`pkg/rbac/scope.go`）：

```go
func ScopeAll(rules, params, base, action) bool {
    return Can(rules, params, base+"/0", action)   // 探哨兵主题 base+"/0"
}
```

- 真实行 id 从 **1** 起，`0` 不可能是真实行；
- 只有 `*` / `#` / 正则这类**宽规则**才可能命中 `/0`；
- 命中 → 视为"该主体在 base 下全量可见/可操作"；否则用 `FilterIds` 逐 id 判定求子集。

### 7.2 ctx 注入契约

中间件算好集合后经上下文下发，handler 用 `rbac_scope_ctx.go` 取值：

| ctx 键 | 取用函数 | 含义 |
|---|---|---|
| `CtxKeyRbacScopeIds` | `scopeIdsFromCtx` → `[]uint` | 列表读有权 id；`nil`=全量不过滤 |
| `CtxKeyRbacOpIds` | `opIdsFromCtx` → `(ids, present)` | 批量 update/del 可操作子集；`present=false`=不限制 |

handler 把 idScope 透传给 repo：`Drive_Config__Query(page, size, scopeIdsFromCtx(ctx))`，repo 拼 `WHERE Id IN (...)`（拼在 LIMIT 之前）。

---

## 8. 实例级授权逐 kind 走查

策略表 `instancePolicies`（键 = 去前缀主题），每条目一个 `instancePolicy{kind, base, action, table}`。中间件命中即**缓冲 body → 解析 id → 判权 → 注入 ctx →（拒绝则 403）**。

| kind | 场景 | 判定逻辑 | 拒绝条件 |
|---|---|---|---|
| `kindList` | query/count/search | `ScopeAll(base)`?全量 : 枚举该表全部 id→`FilterIds`→注入 `ScopeIds` | 有权子集为空 → 403 |
| `kindListGate` | alarm-data query/count | 仅接受 `ScopeAll`（未做行过滤） | 非全量 → 403 |
| `kindAdd` | add | `Can(base+"/0")`（仅通配放行，新建无行 id） | 非通配 → 403 |
| `kindUpdate`/`kindDel` | update/del | 解析 body id →`ScopeAll`?全可 : `FilterIds`子集→注入 `OpIds` | 子集为空 → 403 |
| `kindRestart` | restart | 单 id `Can(base+"/"+id)` | 无权 → 403 |
| `kindValueDefer` | 实时读写值/WS | **接口级放行**，逐 id 过滤交 handler(`valueAllowed`) | （由 handler 决定） |

**具体走查示例（批量删 drive，body `{"ids":[5,7,9]}`，主体 user#3）：**

```
rbacInstanceCheck(kindDel):
  ids = [5,7,9]                         # parseBodyIds 解出
  ScopeAll(3, "/config/drive/del")?     # 探 /config/drive/del/0
    ├─是→ ctx[OpIds]=[5,7,9] 放行        # 全可操作
    └─否→ FilterIds(3,"/config/drive/del",[5,7,9])
            逐 id Can("/config/drive/del/5"/7/9)  # MGET 批量判定
            ├─ 子集非空 → ctx[OpIds]=子集 放行
            └─ 子集为空 → 403「无权限操作所选资源」
handler Drive_Del: req.Ids = opIdsFromCtx → 只删有权的
```

**`kindValueDefer` 为什么要"放行+handler 过滤"**：`/point/read/value` 第 3 段是 `value`，而授权规则常是 `/point/read/5`（逐段等长），若中间件按接口级对 `value` 段硬判，会把"仅显式授权具体 id"的主体在进 handler 前提前 403。故此处不拦，进 `point.go` 把 body 解析成点位 id 后 `valueAllowed(ctx, "/point/read", ids)` 逐个过滤。

---

## 9. 数据模型（7 张表 + 加载 SQL）

主体**复用现有表**：业务用户 = `user_account`(id=`user_account.Id`)，API 客户端 = `api_key`(id=`api_key.Id`)。RBAC 新增 7 张：

| 表 | 作用 |
|---|---|
| `t_biz_role` / `t_api_role` | 两套角色（业务 / API 不合表） |
| `t_biz_user_role` / `t_api_client_role` | 主体↔角色绑定 |
| `t_permission` | **全局权限字典**：`perm_code` / `topic_key` / `match_sort` / `effect`(1 allow/0 deny) / `status` |
| `t_role_permission` | 角色↔权限授权（`role_type` 1 业务 / 2 API） |
| `t_direct_permission` | 主体↔权限**直接授权**（预留，UNION 进规则） |

加载某主体有效规则（`LoadEffectiveRules`，过滤全在 SQL）：

```sql
-- biz：三级 JOIN 取角色授权，UNION 直接授权
SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, r.role_code
FROM t_biz_user_role ur
JOIN t_biz_role r         ON r.id = ur.role_id AND r.status = 1
JOIN t_role_permission rp ON rp.role_type = 1 AND rp.role_id = r.id
     AND rp.is_enabled = 1 AND (rp.expire_at IS NULL OR rp.expire_at > NOW())
JOIN t_permission p       ON p.id = rp.perm_id AND p.status = 1
WHERE ur.user_id = ? AND ur.is_enabled = 1 AND (ur.expire_at IS NULL OR ur.expire_at > NOW())
UNION
SELECT p.id, p.perm_code, p.topic_key, p.match_sort, p.effect, 'direct'
FROM t_direct_permission dp JOIN t_permission p ON p.id = dp.perm_id AND p.status = 1
WHERE dp.subject_type = 1 AND dp.subject_id = ? AND dp.is_enabled = 1
     AND (dp.expire_at IS NULL OR dp.expire_at > NOW());
```

- `scanRules`：`effect==1 → Rule.Allow=true`；`Action` 恒 `"*"`；`role_code` 落 `Rule.Role`。
- **管理员自锁风险**：admin(id=1) 权限完全取决于上述 JOIN 链路；链路为空 → 全默认拒绝、连 `/config/rbac/*` 都 403。兜底 = `rbac_admin_role_seed.sql` 建 `super_admin` 并动态绑定全部启用条目。

---

## 10. 错误码与排障

| HTTP code | 触发 | 出处 |
|---|---|---|
| **401** | 缺少鉴权主体 / Bearer IP 变更 | `rbac_middleware.go` / `auth.go` |
| **403** | 判定拒绝（接口级 deny / 实例级无权子集为空） | 两处 |
| **520** | 判权自身故障（回源 MySQL 报错 / 列表枚举失败）→ **fail-closed** | `rbac_middleware.go` / `rbac_instance.go` |
| **417** | 登录参数错误（用户名或密码错误） | `auth.go` |

排障日志形如：

```
WARN RBAC 拒绝 | src=user id=3 | view /config/drive/query | reason=cached_deny
WARN RBAC(实例级) 拒绝 | type=1 id=3 | edit /config/drive/del | reason=无权限操作所选资源
```

`reason` 取值：`no_match_default_deny`（没命中任何规则）/ `matched_deny_intercept`（命中 deny）/ `cached_allow` / `cached_deny`。直接查 Redis：`GET rbac:user:3:/config/drive/query` → 看 `allow/topic/role/sort` 即知"被哪条规则、哪个角色、什么优先级"裁决。

---

## 11. 简化候选（改前/改后对照）

### 11.1 无争议、低风险瘦身（建议直接做）

- **删 `action` 全链路死参数**：`actionFromPath`（`rbac_routes.go`）整个删；`Check*/Can/ScopeAll/FilterIds` 的 `action` 形参去；`instancePolicy.action` 仅留日志或删。
- **合并 5 个 rbac 文件**：接口级 + 辅助 + 实例级收进更少文件，减少跳转。
- **修过时注释**：`server.go:127` 仍写"enable=false 整体跳过，enforce=false 仅观测"，实际开关/超管旁路早已移除（现恒拦截）。

### 11.2 架构方向 —— 判权放哪（需拍板）

#### 方案 A：中间件兜底 + 瘦身（推荐）
保留全局 `Rbac_Middleware` 做"忘不掉"的 fail-closed 保险，仅做 11.1 瘦身；服务层再收敛出更易读的批量入口。
- ✅ handler 仍零权限代码；新增接口默认受保护
- ➖ "路径即主题"约定不直观（靠本文档补足）

#### 方案 B：每 handler 顶部手动判权（你最初的提议）
去掉 RBAC 中间件，rbac 层给两个批量函数，handler 显式调用：

```go
// 提议的服务层接口
func CheckUser  (ctx, topics []string) map[string]bool
func CheckApiKey(ctx, topics []string) map[string]bool
// handler 里
if !rbac.CheckUser(ctx, []string{"/config/drive/query"})["/config/drive/query"] { 403; return }
```

**改前**（现状，handler 干净、中间件自动判）：

```go
func Drive_Query(ctx *gin.Context) {
    var req ...; ctx.BindJSON(&req)
    configs, _ := mysql.Drive_Config__Query(req.Page, req.PageSize, scopeIdsFromCtx(ctx))
    ctx.Set("Response", []any{200, "ok", configs})
}
```

**改后**（每 handler 自带判权）：

```go
func Drive_Query(ctx *gin.Context) {
    // ① 接口级：手写，忘写=裸奔
    if !rbac.CheckUser(ctx, []string{"/config/drive/query"})["/config/drive/query"] {
        ctx.Set("Response", []any{403, "无权限"}); return
    }
    // ② 实例级：还得自己枚举 id → 拼主题 → 过滤（中间件那套并沒消失，只是搬进来）
    all := mysql.Drive_Config__AllIds()
    allow := rbac.FilterIds(ctx, "/config/drive/query", all)
    ...
}
```

- ✅ 判权点显式、读码一眼可见
- ❌ **fail-open**：新增接口忘写判权 = 无鉴权（安全回退）
- ❌ **接口级**只是把布尔从中间件挪进 handler，没省；**实例级**那套 id 过滤从中间件**平移复制到约 40 个 handler**，总量与出错面都上升

#### 方案 C：混合
接口级仍由中间件兜底（防漏），实例级改为 handler 内一行显式调用（去掉 ctx 注入那套隐式契约）。
- ✅ 兼顾"不会漏"与"实例过滤显式可读"
- ➖ 需重构 `rbac_instance.go` 与所有实例级 handler

### 11.3 关于"输入用长 token"的修正

普通业务请求**只带短(access) token**，长(refresh) token 不发；且主体 id 在认证阶段已写进 ctx。判权应直接吃 ctx 里的 `{LogSource, LogAccount}`，**不该让 rbac 层再去解析 token**（会把 Redis 令牌解析重新耦合进来、且拿不到没带的长 token）。

### 11.4 一句话结论

> `map[string]bool` 能干净覆盖**接口级**；但**实例级**（按资源行 id 过滤）本质是"对一批 id 求有权子集"，不是一个布尔能表达——它**不会因简化方案消失，只会在 A/B/C 之间挪位置**。要"既简单又可读"，性价比最高的是 **方案 A + 11.1 瘦身**（必要时叠加 C 把实例级显式化）。

---

## 12. 常见改动落点速查

| 想做的事 | 改哪 |
|---|---|
| 加新业务接口并纳入鉴权 | 路径在 `/api/v1.0/*` 下即自动受接口级保护；在 `t_permission` 加主题条目并授予角色 |
| 某接口改"按行过滤" | `rbac_instance.go` 的 `instancePolicies` 加一条 + handler 读 `scopeIdsFromCtx/opIdsFromCtx` |
| 改判权优先级/匹配语法 | `pkg/rbac/decide.go` / `topic.go`（纯函数，改后 `go test -count=1 ./pkg/rbac/`） |
| 调权限缓存时长 | `configs/config.yaml` → `RBAC.cache_ttl`（默认 30m） |
| 调 token TTL / HMAC 窗口 / 凭据 TTL | token TTL 在登录签发处；`Hmac_Sign_Window`、`ApiKey_Creds_TTL` 在 `auth.go` |
| 管理员自锁兜底 | `rbac_admin_role_seed.sql`（super_admin 动态绑定全部启用条目） |

> **生效前提**：改代码后**重启后端**加载新二进制；改授权/种子后**最多延迟一个 `cache_ttl`（默认 30m）自动生效**（纯 TTL 懒失效，无需手动清缓存）。
