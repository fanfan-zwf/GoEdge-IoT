# `internal/cloud` — 云端通信（预留 / 工具库）

上级文档：[服务总览 README](../../README.md)

> **当前状态：本包未被任何包 import。**
> `server.go` 是历史 singleton 装配代码的注释残留，`Server()` 为空函数；
> `encryption.go` 是一套**完整可用、但暂未接入**的数据加解密工具。
> 保留它的意义：与 Iot-User-Service / Iot-Config-Service 对接云端 RPC 时可直接复用；
> 也是 SHA-512口令派生 + AES-GCM + GZip + CRC32 这套报文封装格式的**唯一实现参考**。
> 若确认不再需要，可以整包删除（删除前请先全文搜索 `internal/cloud`）。

---

## 一、文件清单

| 文件 | 行数 | 状态 | 职责 |
|------|------|------|------|
| `encryption.go` | 384 | 可用、未接入 | GZip / AES-256-GCM / CRC32 封装与拆解；`;`/`:` 分隔的 KV 串解析工具 |
| `server.go` | 79 | 死代码 | `Server()` 空实现，其余全部注释（早期 singleton + 配置 + MySQL/Redis 初始化） |

---

## 二、报文封装流水线

发送端与接收端**必须镜像对称**，否则解不开：

```
发送：明文 → GzipCompress → AesEncryptGCM(key) → EncodeWithCRC32 → 线路
接收：线路 → DecodeAndVerifyCRC32 → AesDecryptGCM(key) → GzipDecompress → 明文
```

一键调用：

```go
payload, err := cloud.Send__CRC32_Aes_Gzip(rawJSON, aesPasswd)   // 压缩→加密→加校验
rawJSON, err    := cloud.Receive__CRC32_Aes_Gzip(payload, aesPasswd) // 校验→解密→解压
```

| 函数 | 说明 |
|------|------|
| `Send__CRC32_Aes_Gzip(dataBytes, aesPasswd)` | 三步封装；任一步失败返回带上下文的 error |
| `Receive__CRC32_Aes_Gzip(dataBytes, aesPasswd)` | 三步拆解；解压上限硬编码 `20MB`（防 zip bomb） |

---

## 三、各原语细节

### 3.1 `GetPasswordHashGood(password) ([]byte, error)`

`sha512.Sum512` → 返回完整 **64 字节** 摘要。AES 只取前 32 字节做密钥（AES-256）。

> 这是**口令派生**（KDF），不是登录密码存储方案。
> 登录密码一律走 bcrypt：见 [pkg/security](../../pkg/security/README.md) 与 `user_account.Login_Pwd_Hash`。

### 3.2 GZip

| 函数 | 注意 |
|------|------|
| `GzipCompress(data)` | 内存压缩，`defer gz.Close()` 之外还显式 `Close()` 一次以保证 flush（重复 Close 无害） |
| `GzipDecompress(compressed, maxSize)` | 用 `io.LimitReader` 限制**压缩流**读取长度；`maxSize` 由调用方给（当前 20MB） |

### 3.3 AES-256-GCM

| 函数 | 行为 |
|------|------|
| `AesEncryptGCM(plainText, key)` | 密钥 = `SHA512(key)[:32]`；随机 12 字节 nonce；输出 `nonce ‖ ciphertext`（`gcm.Seal(nonce, …)` 直接把 nonce 前缀进结果） |
| `AesDecryptGCM(cipherText, key)` | 前 `NonceSize()` 字节当 nonce，其余解密校验 |

**两个重要语义**：

1. `key == ""` 时**原样透传**（加密返回明文、解密返回入参）——这是"可选加密"的约定，
   而不是"空密钥加密"，配置里留空即等于不加密。
2. GCM 自带认证标签，篡改/密钥不符都会得到 `AES-GCM解密失败` 错误，不必另做签名。

### 3.4 CRC32 附加校验

```
EncodeWithCRC32(json)   →  "<json字节流>｜<十进制CRC32>"      分隔符是最后一个 '|'
DecodeAndVerifyCRC32(b) →  按 LastIndexByte('|') 拆分并校验，通过则返回 json 部分
```

- 多项式表：`crc32.MakeTable(crc32.IEEE)`（包级 `crcTable`，避免每次重建）。
- CRC 以**十进制字符串**存储，不是 4 字节二进制——跨语言对接时注意。
- 空输入直接报错，不返回空切片。

### 3.5 分隔串解析工具（旧配置格式遗留）

早期驱动配置用 `;`/`:` 分隔文本，本包提供过解析原语；**现驱动配置已全面改为 JSON**，
这些函数目前只是工具留存：

| 函数 | 语义 |
|------|------|
| `GetKVValue(str, key)` | 解析 `"name:张三;age:18"`；未找到返回哨兵 `NilError`（区别于真错误） |
| `GetSplitLen(str)` | 按 `;` 分段的段数 |
| `GetSplitStr / GetSplitInt / GetSplitBool / GetSplitFloat64 / GetSplitDuration(str, index, errstr...)` | 取第 `index` 段并按类型转换；失败返回调用方传入的 `errstr`（可变参数，用于自定义错误文案） |

> `NilError` 是"未配置"，不是"解析失败"——调用方据此区分缺省与非法，
> 这一错误语义约定在整个驱动层是一致的（新代码请继续遵守）。

---

## 四、若要接入云端通信，建议的做法

1. `server.go` 里的 singleton 模式**不要恢复**：现在配置来自 [internal/config](../config/README.md)，
   连接来自 `internal/repo/{mysql,redis}` 与 [internal/infra/mqtt](../infra/mqtt/README.md)。
2. 云端 RPC 的 HTTP 客户端与鉴权请复用 [internal/service/configsync](../service/configsync/README.md) 的
   `resty` + Basic/Bearer + 401 重试一套做法，不要在本包再造一个。
3. `AES-GCM + GZip + CRC32` 适合 MQTT 大报文（配置全量下发）；小报文（指令回执）直接明文 JSON 即可，
   当前 [internalmqtt](../service/internalmqtt/README.md) 就是明文，请评估后再决定是否引入。
4. 引入任何加密都必须同时提供**接收侧的对称实现**，并写单测覆盖 `key=""` 的透传分支。

---

## 五、相关文档

- [pkg/security](../../pkg/security/README.md) — bcrypt 密码哈希、随机密钥生成（与本报告"口令派生"分工不同）
- [internal/infra/mqtt](../infra/mqtt/README.md) — 实际在用的 MQTT 通道
- [internal/service/configsync](../service/configsync/README.md) — 实际在用的云端 HTTP 通道
