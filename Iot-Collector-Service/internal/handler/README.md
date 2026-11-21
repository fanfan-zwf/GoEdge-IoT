# `internal/handler` — HTTP / WebSocket 接入层

上级文档：[服务总览 README](../../README.md)

本包是服务对外的**唯一 HTTP 入口**：Gin 路由注册、统一响应、双通路鉴权、RBAC 权限裁决、
点位读写值与 WebSocket 推送、配置 CRUD、Excel 导入导出、操作日志、系统信息、接口清单页。
**不含业务规则**：数据流交给 [service/collector](../service/collector/README.md)，
驱动生命周期交给 [service/driver](../service/driver/README.md)，持久化交给 [repo/mysql](../repo/mysql/README.md)。

> 本包有两份**长篇专题文档**，遇到权限/鉴权问题优先读它们：
> - [auth_example.md](auth_example.md) — 双通路鉴权使用示例（含签名算法与各语言示例）
> - [rbac_guide.md](rbac_guide.md) — Auth + RBAC 全链路详解、数据流、简化方向对照

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|------|------|------|
| `server.go` | 192 | `Web()` 建服务、中间件装配、`Response_Use`、CORS、`create_token`、`guardGoroutine`、`WebShutdown` |
| `router.go` | 46 | `gui_api()` 依次调用 11 个 `*_api_register` |
| `auth.go` | 607 | 统一鉴权中间件 + 登录/登出/刷新/内部同步 AKSK |
| `rbac.go` | 399 | RBAC 中间件、主题映射、实例级（行 id）授权策略表 |
| `rbac_admin.go` | 395 | 角色 / 权限字典 / 绑定关系的管理端 API |
| `point.go` | 774 | 实时值读写、WebSocket 推送、报警状态、历史查询、Tag↔id 适配、按点位授权辅助 |
| `config.go` | 928 | 驱动 / 点位 / 报警配置 / 报警数据 / 历史配置 的 query·count·add·update·del·search·restart |
| `export.go` | 970 | 配置 Excel 导出 / 导入（4 类配置，各一 Sheet） |
| `apikey.go` | 285 | `api_key`（AK/SK 接口账号）管理 |
| `user.go` | 253 | `user_account`（登录用户）管理 |
| `log.go` | 513 | 操作者上下文、双通路（HTTP + 内部 MQTT）日志埋点、日志查询/计数/删除 API |
| `system.go` | 220 | 配置文件读/写/重载 + `system/info/*path` 系统信息通配路由 |
| `tagtree.go` | 75 | `tag_tree` 懒加载子节点、节点下点位、全量重建 |
| `docs.go` | 168 | 零依赖接口清单页（HTML / `?format=json`） |

---

## 二、⚠️ 最容易踩的坑

1. **处理器不直接写响应，一律 `ctx.Set("Response", []any{code, msg, data}})`**。
   `Response_Use` 在 `ctx.Next()` **之后**才读取并 `ctx.JSON`，因此：
   - 忘记 `Set` → 客户端拿到 Gin 默认的空 200 响应；
   - 自己调了 `ctx.JSON` 又 `Set` → 会重复写响应（Gin 打 `http: superfluous` 日志）；
   - 数组第 3 项可省略（`data` 为 `null`）；类型不对（code 非 int / msg 非 string）会返回 **HTTP 501** 的裸字符串，属编程错误而非业务错误。

2. **HTTP 状态码 == 业务码**。`ctx.JSON(code, …)` 用的是同一个 `code`，所以 401/403/417/520/521 会真的反映到状态行上；
   前端拦截器不能只判 200。

3. **中间件顺序不可调换**（`server.go`）：
   `allowAllCors` → `Response_Use` → `Auth_Middleware` → `Rbac_Middleware`。
   `Response_Use` 必须在鉴权**之前注册**（它在 `Next()` 后收尾，才能把鉴权中间件写入的 401/403 也包装成标准 JSON）。

4. **RBAC 恒开启、无开关、无超管白名单**。
   `Rbac_Middleware` 上那段"enable=false 整体跳过、enforce=false 仅观测"的注释**是历史遗留，当前代码无此分支**；
   配置里也只有 `RBAC.cache_ttl`（见 [internal/config](../config/README.md)）。
   没有超级用户旁路：管理员靠 `super_admin` 角色拿真实权限。**新主体默认无任何权限**（deny-by-default）。

5. **`action` 维度已废弃**，仅用于日志展示，**不参与权限匹配**。
   授权只看主题（路径）。`actionFromPath` 的兜底是 `view`——新加接口不会自动放行，必须显式授予对应主题（或 `*`）。

6. **实例级端点会缓冲并重放 body**：`rbacInstanceMiddleware` 用 `io.ReadAll` 读干 body 再 `NopCloser` 回填。
   因此这些接口里 `ctx.BindJSON` 仍可用，但**不要**再自己 `io.ReadAll(ctx.Request.Body)`。

7. **`/point/*` 三个接口在中间件里"故意不拦截"**（`kindValueDefer`），
   改由 handler 解析出点位 id 后调 `valueAllowed` 逐个过滤。
   写权限与读权限是两个主题：`/point/read/{id}`（读值 + WS 订阅共用）、`/point/write/{id}`（写值）。

8. **`/alarm/status` 与 `/history/data/query` 没有行级过滤**，只有接口级权限（`/alarm/status`、`/history/data/query` 归 `view`）。
   同理 `/config/export` **导出的 always 是全量**（`export.go` 调 `Drive_Config__Query(0,0)` 未传 `idScope`）：
   有导出权限即可看到所有行，需要限制可见范围时请改造 `export.go`。

9. **写值是"两段式 + 全批校验"**：`collector.PrepareWrite` 任一条非法 → **整批拒绝、零部分写入**；
   之后 RBAC 过滤掉无权点位（可能只剩子集），无权子集被**静默丢弃**，只有全部无权才 403。

10. **WebSocket 无法带自定义头**，`/point/ws` 的令牌只能借 `Sec-WebSocket-Protocol` 子协议位传（仅该路径接受此方式）。
    服务端必须按 RFC 6455 **回选一个子协议**，且只能通过 `Upgrade(..., responseHeader)` 传
    （gorilla 会 Hijack，改 `ctx.Writer.Header()` 无效）。前端写法：`new WebSocket(url, [accessToken])`。

11. **`wsClient.send` 满了直接丢**（`select … default`），慢客户端会丢帧而不是拖垮推送；
    关闭用 `done` channel 而非 `close(send)`（订阅回调对已关闭 channel 发数据会 panic）。

12. **业务码语义**（本包硬编码，见 `server.go` 常量与各处使用）：

    | 码 | 常量 / 含义 |
    |----|-------------|
    | 200 | 成功 |
    | 401 | `Status_No_Login`：无凭证、令牌失效、签名错、重放、IP 变化、缺主体 |
    | 403 | RBAC 拒绝、账号禁用/过期、驱动忙碌、无权限订阅 |
    | 404 | 点位/驱动定位失败（Tag 不存在、驱动不存在） |
    | 417 | 请求参数非法、点位不可写（`ErrPointNotWritable`）、系统信息功能名未知 |
    | 500 | 未归类的 DB / 设备写入失败 |
    | 501 | 响应包装本身写错（编程错误） |
    | 520 | `StatusMysql`：MySQL 错误、RBAC 校验自身异常（fail-closed） |
    | 521 | `StatusRedis`：Redis / 令牌 / nonce 服务不可用 |
    | 522 | `StatusRegex`：表达式计算错误 |

---

## 三、请求处理链路

```
HTTP 请求
  ↓ allowAllCors        按 config.API.Header 逐条 Set；OPTIONS 直接 200 并 Abort
  ↓ Response_Use        ctx.Next() 后把 ctx["Response"] 包成 {Code,Msg,Data,Timestamp}
  ↓ Auth_Middleware     免鉴权路径豁免 → Bearer token（含 WS 子协议位）→ HMAC 四头 → 否则 401
  │                     写入 ctx：Access_Key、Log_Source、Log_Account_Id
  ↓ Rbac_Middleware     命中 instancePolicies → 实例级集中判定（缓冲 body）
  │                     否则 → CheckBizUser/CheckApiClient(主题) → 403/520
  ↓ 业务 handler         只与 service / repo 打交道
```

### 3.1 双通路鉴权（`auth.go`）

| 通路 | 凭证 | 校验 |
|------|------|------|
| 服务间 | `X-AccessKey` + `X-Timestamp` + `X-Nonce` + `X-Signature` | `HMAC-SHA256(secret, timestamp‖nonce‖path‖body)`，输出**小写 hex**；时间窗口 `±60s`；nonce 按 access_key 隔离、TTL `120s`（=2×窗口）防重放；比较用 `hmac.Equal` 常量时间 |
| 前端 | `Authorization: Bearer {access_token}`（WS 走子协议位） | Redis 取 `Auth_Token_Info`（JSON，含 `User_Id/Access_Key/IP/User_Agent/Issued_At`），校验 **IP 一致** |

- **AK/SK 凭据缓存**：`loadApiKeyCreds` cache-aside（Redis `ApiKey_Creds_TTL=30m`，未命中回源 MySQL）；
  管理端 add/update/reset-secret/内部 sync 会调 `refreshApiKeyCreds` 立即回源刷新；账号已删则清缓存。
- **IP 变化的后果是"该账号全端登出"**（`Auth_Account_Token_Logout`），不只是本次请求 401。
  `ipChanged` 任一侧为空则跳过校验（兼容历史令牌与拿不到 IP 的代理）。
- **登录响应**额外回填 `roles` / `permissions`（仅供前端菜单/按钮显隐；API 拦截由中间件独立裁决）；
  取数失败**降级为空集合、不阻断登录**。刷新响应**不含** roles/permissions。
- 令牌 TTL 来自**账号行**：`user_account.Access_Token_TTL` / `Refresh_Token_TTL`（duration 字符串，默认 `15m0s` / `168h0m0s`）。
- **刷新是轮换**：签发新 access+refresh，**旧 refresh 立即删除**（重放旧 refresh 会 401）。
- **禁用/删除用户不会立即掐断已签发的 access token**（请求期不再查库）：
  在 access 令牌自然过期、或下次 refresh 时才生效。要立刻踢人请调 `Auth_Account_Token_Logout`。
- 免鉴权路径（`authExemptPath`，同时免 RBAC）：`/auth/login`、`/auth/logout`、`/auth/refresh-token`、
  `/internal/sync-aksk`、`/docs`、`/api/v1.0/`、`/`。
- 提示语防账号枚举：用户名不存在 / 未设密码 / 密码错误 → 统一 `401 用户名或密码错误`。

### 3.2 RBAC 主题与实例级授权（`rbac.go`）

- 主题 = 去掉 `ApiPrefix`（`/api/v1.0`）后的路径：`/api/v1.0/config/drive/query` → `/config/drive/query`。
- 主题匹配是"类 MQTT"的通配语义，实现见 [pkg/rbac](../../pkg/rbac/README.md)；
  判定入口见 [service/rbac](../service/rbac/README.md)；表结构与授权 SQL 见 [repo/mysql](../repo/mysql/README.md)。
- **实例级 = 把行 id 编进主题**：`/config/drive/query/17`。策略表 `instancePolicies`（键=主题）：

  | kind | 端点 | 行为 |
  |------|------|------|
  | `kindList` | `query` / `count` / `search` | `ScopeAll` 命中→不过滤；否则 `AllIds()` 枚举 + `FilterIds` 求有权集合，**空则 403**，非空则经 `CtxKeyRbacScopeIds` 下发给 handler 作 `idScope` |
  | `kindListGate` | `alarm-data/query`、`alarm-data/count` | **只认通配** `/config/alarm-data/query/*`，不做行级过滤 |
  | `kindAdd` | `add` | 新建无行 id → 用哨兵 `base+"/0"` 探测（`Can`），只有通配规则能命中 |
  | `kindUpdate` / `kindDel` | `update` / `del` | 从 body 解析 id（update=对象数组的 `Id`；del=`{ids:[]}`），求有权**子集**并写 `CtxKeyRbacOpIds`；全无权限→403；解析不到 id→交 handler 处理 |
  | `kindRestart` | `drive/restart` | body 单 id，无权直接 403 |
  | `kindValueDefer` | `/point/read/value`、`/point/ws`、`/point/write/value` | 中间件放行，过滤交由 handler `valueAllowed` |

  注意：`count/search` 的 base 被**合并**指向 `query`（`/config/drive/query`），授了列表读即同时拥有 count/search 的行过滤口径。
- handler 侧读取：`scopeIdsFromCtx(ctx) []uint`（`nil`=全量不过滤）、`opIdsFromCtx(ctx) ([]uint, bool)`（`present=false`=未做实例级限制）。
- 主体类型映射：`Log_Source_User → Rbac_Subject_Biz(1)`、`Log_Source_AKSK → Rbac_Subject_Api(2)`。
- **fail-closed**：判定本身出错（Redis/MySQL 不可用）→ 520 `权限校验暂时不可用`，绝不放行。

---

## 四、路由总表

`Web()` 在 `config.API.enable=false` 时**整体不启动**（只打一条 INFO）。

### 4.1 鉴权与内部（免鉴权）

| 方法 | 路径 | 处理 |
|------|------|------|
| POST | `/api/v1.0/auth/login` | `api_auth_login`：`{access_key,password}` → 令牌 + roles/permissions |
| POST | `/api/v1.0/auth/logout` | `api_auth_logout`：尽力而为删 access（+可选 `refresh_token`），**恒 200** |
| POST | `/api/v1.0/auth/refresh-token` | `api_auth_refresh`：`{refresh_token}` → 轮换双令牌 |
| POST | `/api/v1.0/internal/sync-aksk` | `api_internal_sync_aksk`：`{accounts:[…]}` 按 access_key upsert `api_key`，随后刷新验签缓存 |

> `sync-aksk` **无鉴权**，只能靠网络隔离/反向代理限制来源；不要把它暴露到公网。

### 4.2 实时数据与历史

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1.0/point/read/value` | body 裸数组，元素可为 Tag 字符串或点位 id（可混用）：`["//0001//AI01",12]`；返回内存实时值缓存快照 |
| POST | `/api/v1.0/point/write/value` | body `[{tag|pointId,value,type}]`；两段式写入 + 审计 |
| GET | `/api/v1.0/point/ws?points=[...]` | 只读推送；`points` 为 Tag/id 的 JSON 数组 |
| POST | `/api/v1.0/alarm/status` | body 同读值；返回报警**状态**（非历史） |
| POST | `/api/v1.0/history/data/query` | `{tag|pointId,startTime,endTime,page,pageSize}` → InfluxDB |

- 时间参数是 **RFC3339**（`time.Time`），`startTime > endTime` → 417。
- `alarmStatusRow` 里点位标识只能叫 **`Point_Tag`**（`Alarm_Config_type.Tag` 已被"报警规则名"占用，两个含义不能撞车）；
  未报警时 `Time` 输出 `null`（用 `*time.Time` 覆盖内嵌零值，避免前端显示 `1/1/1`）。
- WS 推送：`writePump` 会**排空 `send` 合并成一批**再序列化，一帧可能包含多个采集周期的数据；30s 定时 Ping。

### 4.3 配置管理（`config.go`）

统一形态：`POST /api/v1.0/config/{资源}/{动词}`，请求体都是**裸数组或 `{ids:[]}`**，成功返回 `200 "ok"`。

| 资源 | 动词 |
|------|------|
| `drive` | `query` `count` `add` `update` `del` `search` `restart` |
| `point` | `query` `count` `add` `update` `del` `search` |
| `alarm` | `query` `count` `add` `update` `del` `search`（报警**配置**） |
| `alarm-data` | `query` `count`（报警**历史**，仅读） |
| `history` | `query` `count` `add` `update` `del`（历史存储配置） |

入参口径（对应 [repo/mysql](../repo/mysql/README.md) 的函数签名）：

- `add` / `update`：`[]Xxx_Add_type` / `[]Xxx_Update_type`（**批量**）；
- `del`：`{"ids":[…]}`；
- `point/query`、`alarm/query`、`history/query`：`{"driveId"|"pointId":[…],"page":n,"pageSize":n}`；
- `search`：名称/Tag 模糊 + 关联 id 过滤 + `quantity` 限数；
- `page=0` 或 `pageSize=0` = **不分页取全量**（重启与导出都依赖这一语义）；
- 所有 query/count/search 都会把 `scopeIdsFromCtx(ctx)` 作为可变参 `idScope` 传给 repo 做行过滤。

`drive/restart` 特殊：先 `driver.IsDriverBusy(id)` → 403；再全量查驱动+点位配置；
**异步**执行 `driver.DriveResetConfig`（`defer guardGoroutine("驱动重启")`），立即返回 `"驱动重启已启动，预计9秒完成"`。
真实耗时来自 `DriveResetConfig` 的固定等待（Close→6s→New→3s→Connect），所以**响应里的"成功"只代表任务已提交**。

### 4.4 账号管理

| 资源 | 路径 | 要点 |
|------|------|------|
| `apikey` | `/config/apikey/{query,count,add,update,reset-secret,del}` | `add` 时服务端用 `security.GenKey(Secret_Bytes=24)` 生成 SK；**`update` 不改 SK**（要换必须 `reset-secret`）；列表/详情**不回显** `secret_key` |
| `user` | `/config/user/{query,count,add,update,del}` | `password` 明文入参、服务端 **bcrypt** 存 `Login_Pwd_Hash`；`update` 时 `password` 为空 = 保持原密码 |

- 两个资源的 TTL 字段都是 `conv.Duration`（`"15m0s"` 形式），空/0 = 用默认值；`expire_at` 零值 = 永不过期。
- **`/config/user`、`/config/apikey` 不在 `instancePolicies` 里**，只有接口级权限，没有"按账号行"的细粒度授权。

### 4.5 RBAC 管理端（`rbac_admin.go`，组前缀 `/api/v1.0/config/rbac`）

| 路径 | 语义 |
|------|------|
| `role/{query,count,add,update,del}` | 角色（`t_biz_role` / `t_api_role`，按 `subject_type` 分表）；`del` 在事务内级联清理绑定 |
| `perm/{query,count,add,update,del}` | 权限字典 `t_permission`（`topic_key` + 匹配模式） |
| `role/perms` · `role/perms/set` | 查角色权限集合 · **集合替换**（提交全量，服务端算增删） |
| `subject/roles` · `subject/roles/set` | 查主体角色 · 集合替换 |
| `subject/perms` · `subject/perms/set` | 查主体**直接**权限（绕过角色）· 集合替换 |

> `set` 是**全量替换**语义，不是追加；并发编辑会互相覆写，UI 侧请做"打开时取全量、提交时传全量"。

### 4.6 日志 / 系统 / 树 / 文档

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1.0/config/log/{query,count,del}` | 单表 `log_operate` 入口，入参 `{"types":["point","drive"],"operates":["write","login"],"operators":["user:1","internalmqtt"],"keyword","start","end","page","pageSize"}`（各维度为空=不限，`types`/`operates` **直接传库里的文本值**，不认识的枚举报 417）；查询结果每行附带 `type_name`/`operate_name`/`channel` |
| POST | `/api/v1.0/system/config/{read,write,reload}` | 读/整份写 `configs/config.yaml` / 重载（见 [config 文档](../config/README.md) 的"注释与键序不保留"警告） |
| GET/POST | `/api/v1.0/system/info/*path` | 通配到 [pkg/sysinfo](../../pkg/sysinfo/README.md) 的 22 个采集函数：`host`、`host/users`、`host/virtualization`、`sensors`、`cpu`、`cpu/cores`、`cpu/percent`、`cpu/times`、`mem`、`mem/swap`、`mem/swapdevices`、`disk`、`disk/usage`、`disk/io`、`load`、`load/misc`、`net`、`net/interfaces`、`net/protocols`、`net/connections`、`process`、`process/pids`；开关走 query（`percpu`/`logical`/`all`/`pernic`/`path`/`kind`/`protocols`） |
| POST | `/api/v1.0/tagtree/{children,points,rebuild}` | `{tag_path:""}` 空=根层；`children` 返回 `{"子路径":[…文件夹],"文件":[…点位]}`；`points` 返回与 `/config/point/query` 同构的列表 |
| GET | `/api/v1.0/docs`、`/api/v1.0/` | 接口清单页；`?format=json` 给机器可读清单（`{total,generated,items}`）；`/` → 302 到 `/api/v1.0/docs` |

- `system/info` 的主题包含子路径段（`/system/info/cpu/percent`），**授权时请写 `/system/info/*` 或 `/system/info/#`**，
  只授 `/system/info` 不生效。
- `docs` 清单在**请求时**由 `r.Routes()` 生成，与注册顺序无关；只输出方法/路径/handler 名/是否需要鉴权，不含业务数据与凭证。

---

## 五、审计日志埋点（`log.go`）

全通路共用**一张表 `log_operate`**：`Type`（操作对象）+ `Operate`（操作方式）定位事件，两列存的都是**可读文本而非数字**
（库里看到 `drive`/`write`，代码里的常量也是它，无需查对照表），`Operator` 区分操作人，
其余细节一律序列化进 `Content`（JSON）。`Operator` 只有三种固定形态：`user:<id>` / `aksk:<id>` / `internalmqtt`。

| 函数 | 用途 |
|------|------|
| `setLogActor(ctx, src, accountId)` | 鉴权中间件调用，写 `CtxKeyLogSource` / `CtxKeyLogAccount`（操作日志归属 **与** RBAC 主体来源） |
| `currentActor(ctx)` | 取回 `(src, accountId, accessKey, ip)`；`ok=false` = 该路由未鉴权或拿不到 id（无法归属，直接跳过埋点） |
| `logOperate(ctx, typ, operate, detail)` | HTTP 通路主入口；`Content` = 通路字段（channel/method/path/query/user_agent/ip）+ `detail`，序列化后**截断到 `Log_Content_Max=8000`** |
| `LogExternalEvent(src, topic, typ, operate, detail)` | **非 HTTP 通路**（内部 MQTT）的统一入口：`src=mysql.Log_Source_Mqtt` → `Operator=internalmqtt`，`Ip` 列为空串、broker 主题写进 `Content.topic` |
| `logWriteValues(ctx, []Log_Write_item)` / `LogMqttWriteValues(topic, …)` | 写值审计：**一个点位一行**（`Log_Write_item{point_id,tag,type,value,time}`），`value` 存**请求原始 JSON 报文** |
| `logLogin` / `logLogout` | 登陆（密码登录、令牌续期、AK/SK 首次使用、MQTT 远程签发令牌）与登出；AK/SK 无登录动作 → `Auth_AkSk_Touch_Throttle` 在 `AkSk_Login_Log_Window=10m` 内只记一条 |
| `clip` / `marshalLogContent` | 长度裁剪与 JSON 化；`Ip` 列统一 `normalizeIP` 后截断 `Log_Ip_Max=45`，`Operator` 截断 `Log_Operator_Max=64` |

- 枚举在 [repo/mysql/business_log.go](../repo/mysql/README.md)，均为命名 string 类型（`Log_Type` / `Log_Operate`，
  两个参数传反会编译报错）：`Log_Type_*`（`all` 全部/`drive` 驱动/`point` 点位/`alarm` 报警/`log` 日志/`history` 历史/
  `account` 账号权限/`system` 系统配置/`service` 服务进程）、`Log_Operate_*`（`add` 增/`del` 删/`update` 改/`query` 查/
  `write` 写值/`login` 登陆/`logout` 登出/`restart` 重启/`import` 导入/`export` 导出）。
  中文展示名映射 `Log_Type_Name` / `Log_Operate_Name`，同时充当查询入参的合法值白名单。
- 失败事件**不加列**：在 `Content.detail` 里写 `"ok":false,"reason":…`（MQTT 还会带上截断后的原始报文 `raw`）。
- 密码、`secret_key`、令牌明文**绝不入库**；`config.yaml` 只记字节数与指纹。
- 埋点全部异步（`go mysql.Log__Add`）且**失败只打印、不影响接口响应**；拿不到操作人身份时降级为 WARN 并跳过。
- 例行列表/搜索查询**不记**日志，只有审计相关读（导出、系统配置读、日志删除）才落库，防表膨胀。

---

## 六、Tag ↔ 点位 id 适配层

对外一律**以 Tag 为主**（与 InfluxDB 序列维度一致），内部缓存/报警/驱动写入仍以点位 **id** 为键，故有这层换算：

| 函数 | 语义 |
|------|------|
| `splitIdTagArray([]any)` | 把裸数组拆成 `ids` / `tags` / `bad`（非字符串非整数、负数、非整值、`>maxSafePointId=1<<53` 都进 `bad`） |
| `idsByTags(tags)` | `map[tag]id` + `missing`；空 Tag 列表**不发 SQL** |
| `mergePointIds(tags, ids)` | 合并两路定位结果为 id 列表，`missing` 非空 → handler 返 404 |
| `tagsByIds` / `tagsByIdsSoft` / `fillValueTags` | 按 id 批量补 Tag（`Soft` 版失败只 WARN，用于响应展示） |
| `resolveTag(tag, id)` | 单点：Tag 优先，否则用 id 反查 Tag（历史查询用） |
| `resolvePointIds([]pointLocator)` | 配置类接口批量换算；**只换算、不回写请求里的同名字段**（`Alarm_Config.Tag` 是报警规则名，回填会覆掉用户输入） |

> id 上限 `1<<53` 是因为 JSON 数字经 `float64` 解析，超过它已无法精确表示整数——视为非法入参而不是静默截断。

---

## 七、扩展指引

**新增一个业务接口**：

1. 在合适的文件里写 `api_xxx(ctx *gin.Context)`，全程只用 `ctx.Set("Response", …)`；
2. 在对应 `xxx_api_register` 里注册（路径务必以 `/api/v1.0/` 开头，否则**不经 RBAC**——中间件只处理带 `ApiPrefix` 的主题）；
3. 若涉及行级权限：在 `instancePolicies` 加一条并选好 `kind`，handler 里用 `scopeIdsFromCtx` / `opIdsFromCtx`；
4. 若要审计：HTTP 走 `logOperate` / `logWriteValues`，非 HTTP 通路走 `LogExternalEvent`；
5. 打开 `/api/v1.0/docs` 确认已注册；
6. **给主体授权**（新接口默认无人可访问）：`/api/v1.0/config/rbac/perm/add` + `role/perms/set`。

**异步任务**：一律 `go func(){ defer guardGoroutine("名字"); … }()`——子 goroutine 的 panic 会带走整个采集进程，
`guardGoroutine` 把 panic 降级成一条带堆栈的 ERROR 日志。

**优雅退出**：`WSCloseAll(reason)`（发 1001 关闭帧）与 `WebShutdown(ctx)` 由 [cmd/server](../../cmd/server/README.md) 的 `gracefulExit` 调用，新增需要收尾的全局状态请照此暴露成对函数。

---

## 八、相关文档

- [auth_example.md](auth_example.md) — 双通路鉴权与签名示例（各语言客户端）
- [rbac_guide.md](rbac_guide.md) — Auth + RBAC 全链路详解
- [pkg/rbac](../../pkg/rbac/README.md) — 主题通配匹配与判定算法
- [internal/service/rbac](../service/rbac/README.md) — 判定入口与结果缓存
- [internal/service/collector](../service/collector/README.md) — 实时值缓存、报警、历史、写值标准入口
- [internal/service/driver](../service/driver/README.md) — 重启、忙碌锁、只读拦截
- [internal/repo/mysql](../repo/mysql/README.md) — `idScope` 行过滤 SQL 与日志表
- [internal/config](../config/README.md) — `system/config/*` 背后的配置模型
