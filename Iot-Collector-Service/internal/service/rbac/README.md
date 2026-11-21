# `internal/service/rbac` — 权限校验服务层：判定入口 + 判定结果 Redis 缓存

上级文档：[服务总览 README](../../../README.md)

本层只做两件事：**把"某主体能否访问某具体主题串"这一问题给出答案**，以及**把这个答案缓存起来**。
匹配算法在 [pkg/rbac](../../../pkg/rbac/README.md)（纯函数），规则数据在 [internal/repo/mysql/rbac.go](../../repo/mysql/README.md)，HTTP 侧的拦截与行级过滤在 [internal/handler/rbac.go](../../handler/README.md)。

安全基调（与 handler 层一致，**务必先读**）：恒开启、恒强制拦截、**无 enable/enforce 开关**、**无超管旁路**、deny-by-default、MySQL 故障 fail-closed（返回 error，中间件给 `520`）。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `rbac.go` | 240 | `cacheValue`、`decKey`、`injectSubject`/`scopeParams`、`compute`、`decideOne`、`decideBatch`、`checkSubject`、`CheckBizUser`、`CheckApiClient`、`Can`、`ScopeAll`、`FilterIds`、`SubjectAuth` |

包内只有这一个文件，无 `_test.go`（判定引擎的测试在 `pkg/rbac/*_test.go`）。

---

## 二、⚠️ 最容易踩的坑

1. **改完 RBAC 权限不会立即生效，最长要等一个 `cache_ttl`。**
   缓存是**纯 TTL 懒失效**：`TTL = config.RBAC.TTL`（默认 `30m`），到期 Redis 自动删，下次请求回源 MySQL 重算。
   **没有主动失效**——旧实现里的"写完 `SCAN + DEL` 全清"已经移除，`/config/rbac/*` 的写操作不会碰缓存。
   所以"给用户加了权限但他说还是 403"是**预期现象**，等最多 30 分钟，或手工 `DEL rbac:user:<id>:*`（用 SCAN，别用 KEYS）。

2. **`allow` 与 `deny` 都缓存，且命中不续期。**
   拒绝结果也缓存，是为了防止"无权限的主体反复打接口、每次回源 MySQL"这种放大攻击；固定过期（不滑动）保证**最多 `cache_ttl` 后必然刷新一次**，避免热点 key 永不过期。

3. **`action` 形参全部无效，别指望它能区分 view/edit/delete。**
   `compute` 里写死 `rbacmatch.Decide(topic, "*", params, rules)`，而数据层 `scanRules` 又把所有规则的 `ru.Action` 置为 `"*"`（`t_permission` 根本没有 action 列）。
   → `CheckBizUser(userId, resource, action, params)` / `Can(..., action)` / `ScopeAll(..., action)` / `FilterIds(..., action)` 里的 `action` **只用于可读性，不参与匹配**。权限语义完全由主题串本身承载（`/config/drive/update/…` 就是"改驱动"）。

4. **调用方传进来的 `params` 被整体丢弃，占位符只认 `userId` / `clientId`。**
   `checkSubject` 从不使用入参 `params`，而是自己调 `scopeParams(subjectType, subjectId)` 构造，并**强制覆盖** `{userId}` / `{clientId}` —— 这是防伪造的关键设计（调用方无法通过传 params 把自己伪装成别的 id）。
   副作用：**主题模式里写 `{orgId}`、`{deptId}` 等任何其他占位符，永远解析失败 → 该规则判为不命中 → 落到默认拒绝。** 需要组织维度请先把 id 编进主题串本身。

5. **缓存里没存 `perm_code` 和权限 id，排障时拿不到"命中了哪条权限"。**
   `cacheValue` 只有 `allow / topic / role / sort`。因此从本层返回的 `Decision` 里 **`PermCode` 与 `HitId` 恒为空**，`Reason` 也只有三种：
   | Reason | 含义 |
   |---|---|
   | `cached_allow` | 命中缓存且允许 |
   | `cached_deny` | 命中缓存且拒绝（**分不清"显式 deny"还是"无规则默认拒绝"**） |
   | `load_error` | 规则加载失败（MySQL 故障），调用方必须从严 |
   想看真实原因（`no_match_default_deny` / `matched_deny_intercept`）只能在**缓存未命中当次**的日志里找 `pkg/rbac.Decide` 的输出，或临时清掉对应 key。要长期可观测，需要把 `PermCode`/`HitId` 加进 `cacheValue`（注意会让缓存值变大）。

6. **Redis 故障不影响正确性，只影响性能；MySQL 故障会直接 403/520。**
   `decideOne` 里 `Rdb.Get` 报错 → 视为未命中 → 回源 MySQL；`Rdb.Set` / `pipe.Exec` 报错 → **忽略**。
   只有 `mysql.LoadEffectiveRules` 的 error 会向上传播。调用方（`Rbac_Middleware`）拿到 error 一律从严拒绝。
   注意：**Redis 挂掉期间每个请求都会打一次 MySQL**（`LoadEffectiveRules` 是 4 表 JOIN + UNION），高并发下可能连锁拖垮数据库——这是"降级"的真实代价。

7. **缓存 key 粒度是"主体 × 具体主题串"，实例级鉴权下 key 数量会很多。**
   `rbac:user:7:/config/point/update/123` 这种带行 id 的 key 大量存在（`FilterIds` 会为每个候选 id 生成一个）。规模 ≈ 活跃主体数 × 被访问的资源数，靠 TTL 收敛。
   `decideBatch` 用 `MGET` 一次取、`miss` 集合只回源**一次**规则、`Pipeline` 一次写回，正是为了压住这个放大。

8. **`FilterIds` / `Can` / `ScopeAll` 在出错时返回"零结果"，而不是 error。**
   ```go
   func Can(...) bool { if err != nil { return false } }          // 从严
   func ScopeAll(...) bool { if err != nil { return false } }     // 从严：退化为"非全量"
   func FilterIds(...) []uint { if err != nil { return nil } }    // 从严：空集 → 中间件判 403
   ```
   三者**都没有 error 返回值**，所以调用方无法区分"确实没权限"与"MySQL 挂了"。中间件对接口级走 `checkSubject`（有 error → 520），对实例级走这三个函数（无 error → 表现为 403）。排查"莫名 403"时要回头查 DB 健康。

9. **`ScopeAll` 靠哨兵 `base + "/0"` 探测"是否全量"。**
   约定真实行 id 从 1 起，所以 `0` 只可能被 `*` / `#` / 正则等宽规则命中。
   推论：**任何把 id 段写成精确值的规则都不会被 `ScopeAll` 认成"全量"**，这是有意的（精确授权 id 只给那几个 id）。同理，若将来允许 `id=0` 的业务行，整个哨兵机制会失效。

---

## 三、缓存模型

```
key    rbac:user:{user_account.Id}:{具体主题串}      业务主体（subjectType = mysql.Rbac_Subject_Biz = 1）
       rbac:api:{api_key.Id}:{具体主题串}            API 主体（Rbac_Subject_Api = 2）
value  {"allow":true,"topic":"/config/drive/update/*","role":"ops","sort":10}
       allow  裁决结果
       topic  命中规则的 topic_key 模式串（默认拒绝时为 ""）
       role   来源角色编码；直接授权为 "direct"
       sort   命中规则的 match_sort 优先级
ttl    config.RBAC.TTL（configs/config.yaml 的 RBAC.cache_ttl，默认 30m），固定过期、命中不续期
```

```go
func decKey(subjectType uint64, subjectId uint, topic string) string {
    kind := "api"
    if subjectType == mysql.Rbac_Subject_Biz { kind = "user" }
    return fmt.Sprintf("rbac:%s:%d:%s", kind, subjectId, topic)
}
```

> `kind` 的判定用的是"等于 Biz 就是 user，否则 api"。传一个非法 `subjectType`（既非 1 也非 2）会被静默当成 `api`，而回源的 `LoadEffectiveRules` 会返回 `非法主体类型` error → 最终表现为拒绝。逻辑安全，但日志里看到的 key 前缀会有误导性。

---

## 四、对外 API

### 4.1 判定入口（中间件用）

```go
func CheckBizUser(userId uint, resource, action string, params map[string]string) (rbac.Decision, error)
func CheckApiClient(clientId uint, resource, action string, params map[string]string) (rbac.Decision, error)
```

`action` 与 `params` **都被忽略**（坑 3、4），签名保留只为兼容。返回的 `Decision` 见坑 5 表。

### 4.2 实例级（数据级）授权

主题形如 `/config/drive/query/{id}`，`base` 是去掉 id 的前缀：

```go
func Can(subjectType uint64, subjectId uint, topic, action string) bool
        // 单资源判定：restart 的单 id、add 的通配放行
func ScopeAll(subjectType uint64, subjectId uint, base, action string) bool
        // 探测 base+"/0" → 是否"全部可见/可操作"
func FilterIds(subjectType uint64, subjectId uint, base, action string, ids []uint) []uint
        // 候选 ids → 有序授权子集；len(ids)==0 直接返回 nil
```

handler 侧的三类用法（详见 [handler README](../../handler/README.md)）：

| kind | 用法 |
|---|---|
| `kindList` | 先 `ScopeAll`，非全量则用 `allIdsForTable(table)` 枚举候选 id 再 `FilterIds` 求可见集，塞进 `CtxKeyRbacScopeIds` 供 DAO 的 `idScope` 使用 |
| `kindListGate`（alarm-data） | 只做通配读权限门禁，**不做行过滤** |
| `kindUpdate` / `kindDel` / `kindRestart` / `kindAdd` | 从 body 解析 id 集 → `FilterIds` → 子集为空则 403；`kindAdd` 用 `base+"/0"` 探通配 |
| `kindValueDefer`（`/point/read/value`、`/point/ws`、`/point/write/value`） | 接口级放行，handler 解析出点位 id 后调 `valueAllowed` → 内部即 `ScopeAll` + `FilterIds` |

### 4.3 登录回填

```go
func SubjectAuth(subjectType uint64, subjectId uint) (roles, perms []string, err error)
```

- 直接回源 MySQL，**不经过判定缓存**（它需要枚举全部权限而非单主题裁决）；
- `perms` 只收 `Allow == true` 的规则 `PermCode`（deny 不进前端可见集）；
- `roles` 来自 `LoadRoleCodes`（同样已过滤启用/过期）；
- 登录响应里带 `roles`/`permissions` 供前端做菜单可见性；**刷新令牌响应不带**（见 handler README）。
- handler 里对该调用的错误处理是**降级为空集合**而非拒绝登录——即"权限查不到也能登录，但前端菜单空"。

---

## 五、一次判定的完整链路

```
Rbac_Middleware
  │ 归一主题：req.URL.Path（已去 /api/v1.0 前缀）+ 实例级拼 id
  ├─ CheckBizUser / CheckApiClient → checkSubject
  │     └─ decideOne
  │           ├─ decKey → Redis GET
  │           │     ├─ 命中且 JSON 合法 → 直接返回（allow/deny 都算命中）
  │           │     └─ 未命中/损坏/Redis 报错 → 继续回源
  │           ├─ mysql.LoadEffectiveRules(type, id)     4 表 JOIN UNION 直接授权，SQL 侧过滤 status/is_enabled/expire_at
  │           │     └─ error → Decision{false,"load_error"} + err → 中间件 520
  │           ├─ scopeParams：强制注入 {userId}/{clientId}（防伪造）
  │           ├─ compute → pkg/rbac.Decide(topic, "*", params, rules)
  │           │     候选 = 主题命中 且 动作命中 → deny 优先 → 精确>+/*>#>正则 → match_sort 升序 → perm id 升序 → 取第一
  │           │     无候选 → 默认拒绝
  │           └─ Redis SET（TTL=config.RBAC.TTL），写失败忽略
  └─ 未过 → 403；过了且是 kindList 类 → 再走 ScopeAll/FilterIds 得 idScope
```

批量分支（`FilterIds`）：`MGET` 全部 key → 未命中的 topic 集合只 `LoadEffectiveRules` **一次** → 逐个 `compute` → `Pipeline` 一次写回 → 按入参顺序输出允许子集。

---

## 六、配置

```yaml
RBAC:
  cache_ttl: 30m      # 唯一配置项；代码里对应 config.Config.RBAC.TTL
```

- 无 `enable` / `enforce` / 超管白名单配置（`configs/config.yaml` 的注释已明确写死这一点）。
- `cache_ttl` 是**安全与负载的折中**：调小 → 权限变更更快生效、但 MySQL 回源更频繁（尤其配合坑 6 的 Redis 故障放大）。

---

## 七、排障速查

| 现象 | 定位 | 处理 |
|---|---|---|
| 加了权限仍 403 | 坑 1（最长等一个 cache_ttl） | 等 TTL，或 `SCAN MATCH rbac:user:<id>:<主题>` 后 DEL 对应 key |
| 移除了权限仍能访问 | 同上 | 同上；这是 allow 缓存的正常表现 |
| 所有接口偶发 520 | 坑 6（`LoadEffectiveRules` 失败） | 查 MySQL 与 8 张 RBAC 表是否存在（新库要导备份 SQL） |
| 莫名 403 且权限确实有 | 坑 8（`Can/ScopeAll/FilterIds` 吞掉了 error） | 看是否有 MySQL 慢查询/连接耗尽 |
| 带 `{orgId}` 的规则永不生效 | 坑 4（只注入 `userId`/`clientId`） | 把组织维度编进主题串或换精确规则 |
| 想按 view/edit 区分权限 | 坑 3（action 恒 `*`） | 用不同主题串表达，如 `/config/drive/update/*` vs `/config/drive/del/*` |
| 审计日志里没有 perm_code | 坑 5（缓存值未存） | 清 key 后看当次回源日志，或扩展 `cacheValue` |
| 权限变更后 Redis key 暴涨 | 坑 7（key 粒度含行 id） | 属预期，TTL 收敛；关注 Redis 内存而非清理 |
| 登录成功但前端菜单全空 | 坑 6/`SubjectAuth` 降级为空集合 | 查 RBAC 表；登录响应里 `roles`/`permissions` 是否为空数组 |
| `redis` 挂时 MySQL 压力突增 | 坑 6（每请求回源） | 尽快恢复 Redis；或给 `LoadEffectiveRules` 加进程内短 TTL 兜底缓存 |

---

## 八、扩展指引

- **新增主体类型**：`decKey` 的 `kind`、`scopeParams` 的占位符名、`mysql.Rbac_Subject_*` 常量、`LoadEffectiveRules` 的表名分支四处都要同步；漏改会静默按 `api` 处理（见第三节末）。
- **想让权限变更立即生效**：在 `handler/rbac_admin.go` 的写操作成功后，按 `subjectType` 反查受影响主体 id 集合，精确 `DEL` 它们的 `rbac:<kind>:<id>:*`（**不要用全库 SCAN**），或引入一个全局版本号拼进 key 前缀实现"整体翻篇"。
- **想补 action 维度**：需要给 `t_permission` 加列、改 `scanRules` 不再硬编码 `"*"`、并把 `compute` 的第二个实参从 `"*"` 换成真实 action——三处缺一不可，且要重新审视 `Decision.Reason` 的可观测性（坑 5）。
- **想加进程内 L1 缓存**：放在本层而不是 `pkg/rbac`（后者必须保持纯函数、零依赖，才能被 `mysql` 与测试直接引用）；注意 L1 会削弱"权限撤销的时效性"，TTL 要比 `cache_ttl` 短得多。

---

## 九、相关文档

- [服务总览 README](../../../README.md)
- [pkg/rbac](../../../pkg/rbac/README.md) — 主题匹配与判定优先级纯函数
- [internal/handler](../../handler/README.md) — `Rbac_Middleware`、`instancePolicies`、`kindValueDefer`
- [internal/repo/mysql](../../repo/mysql/README.md) — `LoadEffectiveRules` / `LoadRoleCodes` / RBAC 管理 DAO
- [internal/repo/redis](../../repo/redis/README.md) — 本层缓存所用的 `Rdb`
- [internal/config](../../config/README.md) — `RBAC.cache_ttl`
