# `pkg/rbac` — 权限主题匹配与判定内核（纯函数、零依赖、可单测）

上级文档：[服务总览 README](../../README.md)

本包是极简 RBAC 的**算法层**：给定"资源主题 + 主体的有效规则集"，输出允许/拒绝。它**不查数据库、不碰 Redis、不感知 HTTP**，因此可以脱离服务单测（包内自带 3 个 `_test.go`，共 280 行）。缓存、加载、主体识别全部在 [internal/service/rbac](../../internal/service/rbac/README.md)，表结构在 [internal/repo/mysql](../../internal/repo/mysql/README.md)。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `topic.go` | 168 | 主题归一化 / 切段 / 模式校验 / 占位符替换 / 两种匹配模式（分段 & 整串正则）/ 正则编译缓存 |
| `decide.go` | 122 | `Rule`、`Decision` 结构体；`specificity` 具体度分层；`Decide` 候选筛选 + 优先级排序 + 默认拒绝 |
| `scope.go` | 33 | 实例级（数据级）授权便利函数：`Can` / `ScopeAll` / `FilterIds` |
| `topic_test.go` | 96 | 归一化、切段、模式校验、分段/正则匹配、占位符兜底的表驱动用例 |
| `decide_test.go` | 110 | deny 优先、具体度分层、match_sort、默认拒绝等判定顺序用例 |
| `scope_test.go` | 74 | 哨兵 id 探测全量、逐 id 过滤保序等用例 |

**导入方**（4 处，全部在 internal 层）：`internal/service/rbac/rbac.go`（别名 `rbacmatch`）、`internal/handler/rbac.go`（`rbacmatch.Decision`）、`internal/handler/rbac_admin.go`（`rbacmatch.ValidatePattern`）、`internal/repo/mysql/rbac.go`（`Rule` 类型与加载期校验）。

---

## 二、⚠️ 最容易踩的坑

1. **两种匹配模式互斥，判据是"字符串里有没有反引号"。**
   `Match` 与 `ValidatePattern` 都用 `strings.Contains(pattern, "`")` 分流。只要主题里出现**任意位置**的反引号就走正则分支；此时若反引号不是"成对且恰好在首尾"，`regexBody` 返回 `ok=false` → **整条规则直接判不匹配**（`ValidatePattern` 则会明确报错）。段中间夹一个反引号是配不出正则模式的。

2. **分段模式是"等长精确"，不是前缀匹配。**
   `matchSegs` 最后一行 `return len(pat) == len(res)`（`topic.go:167`）。所以 `/config/rbac/+/del`（4 段）**不命中** `/config/rbac/perm/del/13`（5 段）。想覆盖到实例层必须写 `*`（占一段）或 `#`（吞剩余所有段）。这是最常见的配置误解。

3. **`+` 与 `*` 完全等价**，都表示"恰好一段"。
   `topic.go:160`：`if pat[i] == "+" || pat[i] == "*"`。与标准 MQTT 语义（`#` 多层）不同的地方是这里没有引入 `*` 的多层含义，别按通配符直觉把 `*` 当 `#` 用。

4. **`#` 只能出现在最后一段，且必须独占整段。**
   `ValidatePattern`（`topic.go:68-76`）两条检查：`#` 非末段 → 报错；任何段内含 `#+*` 但不等于 `#`/`+`/`*` → 报错（禁止 `a+`、`+b`、`x*y`、`ab#`）。加载期会拦，但**运行时 `Match` 不重复校验**，手工往库里塞脏数据会得到未定义行为（`matchSegs` 遇到非末段 `#` 会提前 return）。

5. **`#` 匹配"零段"也算命中。**
   `topic.go:155`：`return len(res) >= i`。所以 `/system/info/#` 同时命中 `/system/info`（2 段）和 `/system/info/a/b`。写 `#` 时不要以为必须有下级路径。

6. **占位符取不到值 → 该规则判不匹配（安全兜底，不是报错）。**
   `ReplacePlaceholders`（`topic.go:113-124`）只要**任一** `{name}` 在 `params` 里不存在就返回 `ok=false`，`Match` 随即返回 `false`。因此：模式里多写一个没被注入的占位符，等于把这条规则**永久作废**，且没有任何日志。
   叠加服务层的现实：`internal/service/rbac` 只注入 `userId` **或** `clientId` 一个键（其余 params 被丢弃），所以 `{orgId}` 之类**永远不命中** → 默认拒绝。参见 [service/rbac README](../../internal/service/rbac/README.md)。

7. **正则模式是"隐式整段锚定"，不是前缀匹配。**
   `anchoredMatch` 编译的是 `"^(?:" + body + ")$"`（`topic.go:102`）。写 `` `/config/drive/query/[0-9]+` `` 才能命中 `/config/drive/query/13`；写 `` `/config/drive/query` `` 不会命中它下面的任何子路径。

8. **校验期与运行期对占位符的处理不同。**
   `ValidatePattern` 试编译前把 `{name}` 替成字面量 `0`（`topic.go:58`），因为 RE2 会把 `{3,5}` 当量词、对 `{orgId}` 报假错。运行期 `Match` 则是**真替换**。副作用：正则里**故意**写的量词 `{2}` 在校验时会被"占位符替换"误伤吗？不会——`placeholderRe` 只匹配 `\{([A-Za-z0-9_]+)\}`，纯数字 `{2}` 不匹配，会原样进 `regexp.Compile`。但 `{2,5}` 含逗号也不匹配，同样安全。

9. **`Decision.PermCode` / `HitId` 只有在真命中规则时才有值。**
   默认拒绝分支（`decide.go:91`）返回的 `Decision` 只填了 `Allowed=false, Reason="no_match_default_deny"`，其余字段全零。做审计日志时不要指望拒绝原因里能拿到"缺哪条权限"。

10. **`action` 参数在本项目里是摆设。**
    `actionMatch`（`decide.go:62-64`）逻辑正确（`rule.Action == "*"` 或全等），但 `t_permission` 表**没有 action 列**、`mysql.scanRules` 硬置 `Rule.Action = "*"`，服务层调用 `Decide` 时也固定传 `"*"`。三层叠加后动作维度实际不参与判定——`scope.go:14` 的注释已明说这点。设计新权限模型时不要以为"配个 delete 就能区分"。

11. **`Rule` 与 `Decision` 的 JSON tag 是 `snake_case`，与项目其它结构的 `PascalCase` 不一致。**
    `perm_id` / `topic_key` / `match_sort` …… 这是给管理端接口用的。改动 tag 会直接破坏 [handler/rbac.go](../../internal/handler/README.md) 返回给前端的字段名。

12. **`rxCache` 只增不减，且编译失败也缓存 `nil`。**
    `topic.go:94-109`：键是"已解析占位符后的正则原文"。若正则里带 `{userId}` 这类会随用户变化的占位符，缓存键会随 id 数量线性膨胀（不会淘汰）。当前 RBAC 主题里正则模式极少，暂无风险，但接入按 id 变化的正则前需要评估。

---

## 三、API 一览

### 3.1 `topic.go`

| 函数 | 签名 | 说明 |
| --- | --- | --- |
| `Normalize` | `(topic string) string` | `TrimSpace` + **只裁一个**前导 `/`；使 `/a/b` 与 `a/b` 等价 |
| `Split` | `(topic string) []string` | 归一化后按 `/` 切段；空串返回 `nil` |
| `ValidatePattern` | `(topic string) error` | 加载/录入期校验；正则分支查反引号成对 + 可编译，分段分支查 `#+*` 独占与 `#` 末段 |
| `Match` | `(pattern, resource string, params map[string]string) bool` | 唯一匹配入口，内含占位符解析 |
| `ReplacePlaceholders` | `(topic string, params map[string]string) (string, bool)` | 暴露给服务层做主题拼装/诊断用 |

### 3.2 `decide.go`

```go
type Rule struct {
    PermId    uint64 `json:"perm_id"`
    PermCode  string `json:"perm_code"`
    TopicKey  string `json:"topic_key"`   // 类 MQTT 模式，可含 +/*、尾部 #、{占位符}、反引号正则
    Action    string `json:"action"`      // 本项目恒为 "*"（坑 10）
    MatchSort int    `json:"match_sort"`  // 越小越优先
    Allow     bool   `json:"allow"`       // true=allow，false=deny
    Role      string `json:"role"`        // 来源角色编码；直接授权为 "direct"
}

type Decision struct {
    Allowed bool; Reason string; PermCode string; HitId uint64
    TopicKey string; MatchSort int; Role string
}

func Decide(resource, action string, params map[string]string, rules []Rule) Decision
```

`Reason` 的**全部取值**（做日志/排障就靠这四个字面量）：

| Reason | 触发 | Decision 其余字段 |
| --- | --- | --- |
| `matched_allow` | 排序后首条规则 `Allow==true` | 全部填命中规则的值 |
| `matched_deny_intercept` | 排序后首条规则 `Allow==false`（deny 拦截） | 全部填命中规则的值 |
| `no_match_default_deny` | 无任何候选规则 | 其余字段零值 |
| `cached_allow` / `cached_deny` / `load_error` | **不在本包**，由服务层缓存命中时改写 | 见 [service/rbac README](../../internal/service/rbac/README.md) |

### 3.3 `scope.go` — 实例级授权三件套

| 函数 | 语义 |
| --- | --- |
| `Can(rules, params, topic, action)` | 单主题判定，包一层 `Decide().Allowed` |
| `ScopeAll(rules, params, base, action)` | 探测 `base + "/0"`：**只有 `*` / `#` / 正则等宽规则能命中哨兵 `"0"`** → 为真表示"全量可见/可操作" |
| `FilterIds(rules, params, base, action, ids)` | 逐 id 拼主题判定，返回**保序**子集；`len(ids)==0` 返回 `nil` |

> 哨兵机制依赖一条硬约定：**业务行 id 从 1 起**（`scope.go:5` 的注释）。真实数据里若出现 id=0 的行，它会被"全量规则"和"仅 id 0 规则"混淆。

### 3.4 具体度分层（`specificity`，值小者优先）

| 层 | 模式样例 | 值 |
| --- | --- | --- |
| 精确 | `/config/drive/query` | 0 |
| 单层通配（含 `+` 或 `*` 段） | `/config/drive/*` | 1 |
| 尾部 `#` | `/system/info/#` | 2 |
| 整串正则 | `` `/config/drive/[0-9]+` `` | 3 |

---

## 四、判定流程（`Decide`）

```
入参 (resource, action, params, rules)
  │
  ├─ 逐条筛候选：actionMatch(rule.Action, action)  &&  Match(rule.TopicKey, resource, params)
  │      （Match 内含占位符解析，解析失败即不命中 —— 安全兜底）
  │
  ├─ 候选为空 ────────────────────────────→ Decision{Allowed:false, Reason:"no_match_default_deny"}
  │
  ├─ sort.SliceStable 四级排序：
  │      ① denyRank  （deny=0 在前 → 拒绝优先拦截）
  │      ② tier      （specificity：精确 > +/* > # > 正则）
  │      ③ MatchSort （数据库配的优先级，升序）
  │      ④ PermId    （升序，保证同前四级完全打平时结果稳定）
  │
  └─ 取 cands[0].rule 的 effect 作为最终结果
         Allow → "matched_allow"
        !Allow → "matched_deny_intercept"
```

要点：**只看排名第一的规则，不做"任一 allow 即通过"的并集**。因此"拒绝优先"是全局硬性质——只要存在一条能命中且排在最前的 deny 规则，就无法被其它 allow 规则翻案（deny 的 denyRank 恒为 0）。这是"给某角色开全集、再单独封一个接口"这种配法能够成立的根基。

---

## 五、排障速查

| 现象 | 本包侧原因 | 验证方法 |
| --- | --- | --- |
| 规则配了不生效（403） | 分段模式段数不等（坑 2） | 数一下模式与请求路径各几段；缺的用 `*`/`#` 补 |
| 规则配了不生效（403） | 占位符未被注入（坑 6） | 服务层只注入 `userId`/`clientId`；`go test ./pkg/rbac/` 可用 `Match` 单跑验证 |
| 规则配了不生效 | `Group`/`Allow` 写反，或 deny 抢在前 | 看 `Decision.Reason`：`matched_deny_intercept` 会带出命中 deny 的 `PermCode` |
| 明明只封一个 id，结果整块都不能用 | 该角色有一条宽 `#`/正则 deny | `specificity` 只影响同层排序，deny 恒先于 allow |
| 拒绝日志里 `perm_code` 为空 | 走的是默认拒绝分支（坑 9） | 属正常，说明是"压根没授权"而不是"被显式拒绝" |
| 正则模式一直 403 | 反引号不成对 / 未在首尾（坑 1） | 先过 `ValidatePattern`；`` `x` `` 两侧不能有多余字符 |
| 列表接口比预期少数据 | `FilterIds` 从严返回空 | 服务层 `err != nil → nil`（见 [service/rbac README](../../internal/service/rbac/README.md)），与本包无关 |
| 想确认匹配语义 | 直接跑单测 | `go test ./pkg/rbac/ -run TestMatch -v` |

---

## 六、扩展指引

**新增一种通配语法**（例如段内集合 `{a,b}`）
1. 在 `matchSegs` 里加分支（`topic.go:152`）；
2. **同步**改 `ValidatePattern` 的"通配符须独占整段"检查，否则脏模式能通过校验却在运行期未定义（历史上 `[..]` 集合表达式就是这么加了又整体删掉的，含 `strconv` 与 specificity 的一层）；
3. 在 `specificity` 里给它插一个合适的层级值——注意层级是**全局排序依据**，插错位置会让原有deny/allow 优先级关系发生变化；
4. 补 `topic_test.go` + `decide_test.go` 表驱动用例；
5. 前端文案同步：`Iot-Collector-Web` 的 `locales/zh-CN.json`、`en.json` 里 `rbac.pageTip` / `rbac.topicTip`。

**恢复 action 维度**（当前已废弃）
需要同时动 4 处：`t_permission` 加回 `action` 列、`mysql.scanRules` 不再硬置 `"*"`、服务层 `compute` 把真实 action 传给 `Decide`、`handler` 侧按 HTTP 方法/接口派生 action。**只改本包不够**——本包的 `actionMatch` 一直是正确的。

**性能**
`Decide` 是 `O(规则数 × 匹配开销)`，规则数按主体加载后通常个位数到几十；分段匹配无内存分配，正则走 `rxCache`。真正的开销在服务层的规则加载与缓存（Redis + freecache），见 [service/rbac README](../../internal/service/rbac/README.md)。

---

## 七、相关文档

- [服务总览 README](../../README.md)
- [internal/service/rbac](../../internal/service/rbac/README.md) — 规则加载、Redis 缓存、params 注入、`Reason` 改写
- [internal/repo/mysql](../../internal/repo/mysql/README.md) — `t_permission` / `t_role` / `t_user_role` 等 RBAC 9 表与 `LoadEffectiveRules`
- [internal/handler](../../internal/handler/README.md) — 四类 kind 的主题拼装与 `Decision` 出参
- [internal/cloud](../../internal/cloud/README.md) — HMAC + nonce 鉴权（RBAC 之前的第一道门）
