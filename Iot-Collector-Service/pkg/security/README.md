# `pkg/security` — 凭证随机数生成与登录口令 bcrypt 哈希

上级文档：[服务总览 README](../../README.md)

本包只有两个文件、三个导出函数，却是全服务**所有凭证实体的唯一出处**：AK/SK、access_token、refresh_token 的随机串，以及用户登录口令的哈希与校验都在这里。任何"需要不可猜测字符串"的地方都应调 `GenKey`，不要自己 `rand.Intn` / `uuid` / 时间戳拼接。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | --- | --- |
| `key.go` | 30 | `GenKey(nBytes int) (string, error)` — `crypto/rand` 取 n 字节随机数，输出 base64url（无填充） |
| `pwd.go` | 34 | `HashPassword(pwd string) (string, error)` / `VerifyPassword(hash, pwd string) bool` — bcrypt 封装 |
| 合计 | 64 | — |

无内部依赖（只依赖标准库 + `golang.org/x/crypto/bcrypt`），可安全被任何层 import。

---

## 二、⚠️ 最容易踩的坑

1. **`GenKey` 的长度参数是"字节数"，不是"字符数"。**
   base64 每 3 字节编成 4 字符，所以输出长度 ≈ `ceil(nBytes*8/6)`：16 字节 → 22 字符，24 字节 → 32 字符，32 字节 → 43 字符。写库或做长度校验时按**字符数**算。

2. **`GenKey` 用 `crypto/rand`，阻塞但安全；性能远低于 `math/rand`。**
   绝对不要在采集/推送热循环里调它。当前所有调用点都在低频路径（登录、签发/刷新令牌、创建/重置 API Key），这是刻意的。

3. **`GenKey` 会返回 error，绝大多数调用点把它映射成 500。**
   实际上 `crypto/rand.Read` 在现代 Linux 上几乎不会失败。看到 `生成随机字节失败` 这种日志，优先怀疑系统熵池或平台异常，而不是代码逻辑。

4. **`nBytes <= 0` 直接返回错误，不 panic。**（`key.go:19-21`）传 0 是常见手误，返回的是 `"GenKey: 字节长度必须大于 0，当前=0"`，字符串里带当前值，排查时直接搜这句。

5. **`VerifyPassword` 的返回值是"匹配与否"的裸 `bool`，吞掉了 bcrypt 的具体错误。**（`pwd.go:33`）
   `bcrypt.CompareHashAndPassword` 其实能区分 `hash magic version mismatch`（库里存的根本不是 bcrypt 串）与密码不匹配。这里一律折算成 `false`，所以**"明文密码被误写进哈希列"这种脏数据表现为"密码错误"而不是"数据异常"**。迁移老库时要单独校验哈希串是否以 `$2a$`/`$2b$`/`$2y$` 开头。

6. **`hash == "" || pwd == ""` 短路返回 `false`。**（`pwd.go:30-32`）
   这挡住了"空哈希 + 空密码 = 登录成功"的经典漏洞；`handler/auth.go:218` 又额外加了 `acc.Login_Pwd_Hash == ""` 判断做双保险。删掉任意一层都要重新评估。

7. **bcrypt 有 72 字节口令上限，超长部分被静默丢弃。**
   `HashPassword` 不做长度校验，`bcrypt.GenerateFromPassword` 也只会在 cost 越界时报错。若前端允许超长口令，两个不同但前 72 字节相同的密码会被判定为同一个。需要长口令/ passphrase 场景请换 `golang.org/x/crypto/argon2` 或先做 SHA-256 预哈希（**改动会兼容不了，属于破坏性变更**）。

8. **`HashPassword` 用 `bcrypt.DefaultCost`（当前为 10），没有做成配置项。**
   想调强度只能改代码并重新哈希全部存量口令；`GenerateFromPassword` 每次返回的串都不同（内含随机盐），**不能**用哈希结果做等值比较或唯一索引。

9. **`HashPassword` 拒绝空串（`pwd.go:18-20`），所以"密码留空表示不修改"这类语义要在上层处理。**
   `handler/user.go:104/166` 两处调用都要求先确认用户确实提交了新密码。

---

## 三、API 与调用点

| 函数 | 签名 | 项目内实际用法 |
| --- | --- | --- |
| `GenKey` | `(nBytes int) (string, error)` | 见下表 |
| `HashPassword` | `(pwd string) (string, error)` | `handler/user.go:104`（建用户）、`:166`（改密码）→ 存 `user_account.Login_Pwd_Hash` |
| `VerifyPassword` | `(hash, pwd string) bool` | `handler/auth.go:218` 登录校验 |

`GenKey` 的调用点与长度约定：

| 位置 | 常量 | 字节数 | 产物 |
| --- | --- | --- | --- |
| `handler/auth.go:227` | `Token_Bytes` | 32 | access_token |
| `handler/auth.go:232` | `Token_Bytes` | 32 | refresh_token |
| `handler/auth.go:340` / `:345` | `Token_Bytes` | 32 | 刷新时轮换出的新 access/refresh token |
| `handler/apikey.go:109` | `Secret_Bytes` | 24 | API 客户端 `secret_key` |
| `handler/apikey.go:218` | `Secret_Bytes` | 24 | 重置 `secret_key` |
| `internalmqtt/subscribe.go:294` / `:309` | `mqttTokenBytes` | 32 | MQTT 通路签发的 access/refresh token |

> `Access_Key` **不是**本包生成的——它由调用方在创建 API 客户端时自行指定（`handler/apikey.go:95` 只校验非空 + 唯一性）。SK 才是随机生成的，且明文只在创建/重置当次响应里返回一次，库里存的就是这个明文（HMAC 签名需要双方持有同一 secret，见 [internal/cloud README](../../internal/cloud/README.md)）。

---

## 四、编码格式说明

`GenKey` 返回的是 `base64.RawURLEncoding`：

| 特性 | 说明 |
| --- | --- |
| 字母表 | `A-Z a-z 0-9 - _`（用 `-`/`_` 替代 `+`/`/`） |
| 填充 | **无 `=` 填充**（`Raw` 前缀） |
| 可直接放的位置 | URL、query、HTTP 头、Redis key（如 `auth:access_token:<token>`）、JSON |
| 不可做的事 | 用标准 `base64.StdEncoding` 去解码它（`-`/`_` 会报错）；也不要指望它等于 hex |

凭证在整个链路里始终以这段文本原样流转：签发 → 存 Redis → 客户端携带 → 服务端拿它当 key 反查。所以**任何一方做了大小写转换或 padding 修补都会导致鉴权失败**，`GenKey` 的输出本身已大小写敏感且无歧义字符。

---

## 五、排障速查

| 现象 | 定位 | 处理 |
| --- | --- | --- |
| 登录总返回密码错误，但密码确认没错 | 库里 `Login_Pwd_Hash` 是不是被写成了明文（坑 5） | 用 `security.HashPassword` 重算并回写；检查是否有绕过 handler 的直连 SQL 写库 |
| 刚建的 API Key 立刻用 SK 签名失败 | SK 只在创建响应里返回一次 | 调 `/apikey/reset`（`apikey.go:218`）换新的，并确认前端保存的是完整 32 字符 |
| token 长度对不上（预期 43 实际 22） | 生成时传的字节数不同 | 见坑 1 的换算表；不同实体长度本就不同，不是 bug |
| `GenKey: 字节长度必须大于 0，当前=0` | 传入了未初始化的常量/配置 | 检查调用处的 const 与赋值顺序 |
| `生成随机字节失败` | `crypto/rand` 异常 | 容器/沙箱环境熵源问题，非业务逻辑 |
| 想批量导入用户并预置密码 | `HashPassword` 每条都要跑 bcrypt（DefaultCost 下约几十毫秒/CPU） | 1 万条会明显卡住；导入前评估并发或降低批量大小，**不要为此改成可逆加密** |

---

## 六、扩展指引

- **新增一类凭证**：直接 `security.GenKey(n)`，并在调用点定义一个具名常量（沿用 `Token_Bytes` / `Secret_Bytes` 的写法），别在函数里写裸数字。
- **需要可校验格式的 ID**（如带前缀的 `ak_xxx`）：在调用方拼前缀，不要改 `GenKey` 的编码方式——它被 Redis key、HTTP 头、URL 多处依赖。
- **升级 bcrypt cost**：必须配套做"登录成功后透明重哈希"（`VerifyPassword` 通过后再用新 cost `HashPassword` 回写），否则老用户永远停留在旧 cost。
- **换算法（bcrypt → argon2id）**：`pwd.go` 是唯一改动点，但哈希串格式变了会导致存量用户全部无法登录，需双算法兼容期（按 `$2` 前缀分派）。

---

## 七、相关文档

- [服务总览 README](../../README.md)
- [internal/handler](../../internal/handler/README.md) — `/auth/login`、`/auth/refresh`、`/user/add`、`/apikey/add`、`/apikey/reset` 的完整签发与校验流程
- [internal/repo/redis](../../internal/repo/redis/README.md) — `GenKey` 产物的存储位置与 key 空间（`auth:access_token:*` 等）
- [internal/cloud](../../internal/cloud/README.md) — HMAC-SHA256 签名（与 bcrypt 平行的另一套鉴权，AK/SK 由本包生成 SK）
- [internal/service/internalmqtt](../../internal/service/internalmqtt/README.md) — MQTT 通路复用同一套 token 签发
