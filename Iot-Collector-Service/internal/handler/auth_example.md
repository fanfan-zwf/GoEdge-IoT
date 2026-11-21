# 统一鉴权使用示例

本服务采用「一套 account 表 + 一个鉴权中间件」的双通路方案：

- 通路 1（服务间调用）：请求头携带 `X-AccessKey` / `X-Timestamp` / `X-Nonce` / `X-Signature`，HMAC-SHA256 签名校验。
  签名原文顺序：`timestamp + nonce + request path + request body`；时间窗口 ±60 秒；nonce 存 Redis 防重放。
- 通路 2（Vue 前端）：账号密码登录换取 `access_token` / `refresh_token`，业务请求携带 `Authorization: Bearer {access_token}`。

接口清单（统一前缀 `/api/v1.0/`）：

| 方法 | 路径 | 说明 | 是否鉴权 |
|------|------|------|----------|
| POST | `/api/v1.0/auth/login` | 账号密码登录，返回 access/refresh token | 免鉴权 |
| POST | `/api/v1.0/auth/refresh-token` | 刷新 access_token（同时轮换 refresh_token） | 免鉴权 |
| POST | `/api/v1.0/internal/sync-aksk` | 内部同步账号 AKSK（upsert） | 免鉴权 |
| POST | `/api/v1.0/**`（其余业务接口） | 全部业务接口 | 共用鉴权中间件 |

---

## 一、通路 1：客户端 HMAC-SHA256 签名示例（Go）

```go
package main

import (
    "bytes"
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"
)

const (
    baseURL   = "http://127.0.0.1:8103"
    accessKey = "<你的 AK>" // account.Access_Key
    secretKey = "<你的 SK>" // account.Secret_Key，HMAC 密钥，绝不外泄
)

// genNonce 生成随机 nonce（base64url），可用 collector/pkg/security.GenKey(16) 替代
func genNonce() string {
    b := make([]byte, 16)
    _, _ = rand.Read(b)
    return base64.RawURLEncoding.EncodeToString(b)
}

// sign 计算签名原文：timestamp + nonce + request path + request body
func sign(sk, timestamp, nonce, path string, body []byte) string {
    mac := hmac.New(sha256.New, []byte(sk))
    mac.Write([]byte(timestamp))
    mac.Write([]byte(nonce))
    mac.Write([]byte(path))
    mac.Write(body)
    return hex.EncodeToString(mac.Sum(nil))
}

func main() {
    path := "/api/v1.0/config/drive/query"
    body := []byte(`{"page":0,"pageSize":0}`)

    timestamp := strconv.FormatInt(time.Now().Unix(), 10) // Unix 秒
    nonce := genNonce()
    signature := sign(secretKey, timestamp, nonce, path, body)

    req, _ := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-AccessKey", accessKey)
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    req.Header.Set("X-Signature", signature)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    data, _ := io.ReadAll(resp.Body)
    fmt.Printf("status=%d body=%s\n", resp.StatusCode, data)
}
```

要点：
- `path` 只取 URL 路径（不含 query），必须与服务端 `ctx.Request.URL.Path` 完全一致。
- `body` 为请求体原始字节；GET/无请求体时用空字节序列（`[]byte{}`）。
- `X-Signature` 使用小写 hex；服务端做常量时间比较。
- 客户端与服务端时间需大致同步（偏差 < 60 秒），且每个 `nonce` 在窗口期内只能用一次。

同步账号（首次注入 AK/SK，走内部免鉴权接口）示例：

```go
// POST /api/v1.0/internal/sync-aksk
// {
//   "accounts": [
//     {
//       "access_key": "ak-xxxx",
//       "secret_key": "sk-xxxx",
//       "password": "Init@123",      // 明文，服务端 bcrypt 后落库；或直接给 login_pwd_hash
//       "enable": 1,
//       "access_token_ttl": 900,     // 可选，秒；缺省用默认值
//       "refresh_token_ttl": 604800  // 可选，秒
//     }
//   ]
// }
```

---

## 二、通路 2：Vue + axios 请求示例（token 模式）

```js
// src/utils/request.js
import axios from 'axios'

const service = axios.create({
  baseURL: '/api/v1.0',
  timeout: 10000,
})

// 读写 token（登录后端返回 access_token / refresh_token）
const getAccessToken = () => localStorage.getItem('access_token')
const getRefreshToken = () => localStorage.getItem('refresh_token')
const setTokens = (access, refresh) => {
  localStorage.setItem('access_token', access)
  if (refresh) localStorage.setItem('refresh_token', refresh)
}
const clearTokens = () => {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

// 1) 请求拦截器：注入 Authorization: Bearer {access_token}
service.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }
  return config
})

// 2) 响应拦截器：401 时用 refresh_token 刷新一次并重放原请求
let refreshing = null // 防止并发重复刷新

async function doRefresh() {
  // 用裸 axios，避免再次进入拦截器
  const { data } = await axios.post('/api/v1.0/auth/refresh-token', {
    refresh_token: getRefreshToken(),
  })
  // 后端统一响应体：{ Code, Msg, Data, Timestamp }
  if (data.Code !== 200) throw new Error(data.Msg || '刷新令牌失败')
  setTokens(data.Data.access_token, data.Data.refresh_token)
  return data.Data.access_token
}

service.interceptors.response.use(
  (resp) => {
    const body = resp.data
    // 业务码非 200 可按需处理
    return body
  },
  async (error) => {
    const { response, config } = error
    if (response && response.status === 401 && !config._retried) {
      config._retried = true
      try {
        // 并发场景共享同一个刷新 Promise
        refreshing = refreshing || doRefresh()
        const newToken = await refreshing
        refreshing = null
        config.headers['Authorization'] = `Bearer ${newToken}`
        return service(config) // 重放原请求
      } catch (e) {
        refreshing = null
        clearTokens()
        // 跳转登录页
        return Promise.reject(e)
      }
    }
    return Promise.reject(error)
  }
)

export default service
```

登录与业务调用：

```js
// 登录（免鉴权），返回体 Data 含 access_token / refresh_token
import request from '@/utils/request'

export async function login(accessKey, password) {
  const res = await request.post('/auth/login', {
    access_key: accessKey,
    password: password,
  })
  if (res.Code === 200) {
    localStorage.setItem('access_token', res.Data.access_token)
    localStorage.setItem('refresh_token', res.Data.refresh_token)
  }
  return res
}

// 业务请求（自动带 Bearer token，401 自动刷新）
export function queryDrive(page = 0, pageSize = 0) {
  return request.post('/config/drive/query', { page, pageSize })
}
```

要点：
- 业务请求统一走 `/api/v1.0/` 前缀与同一个鉴权中间件，前端只需保证携带 `Authorization: Bearer {access_token}`。
- `expires_in`（秒）用于前端判断 access_token 何时过期，可提前主动刷新。
- 登录/刷新响应绝不包含 `secret_key`；`secret_key` 仅用于服务间 HMAC，前端无法获取。
- 若走跨域，后端 CORS 已在 `Access-Control-Allow-Headers` 放行 `Authorization` 及 `X-AccessKey/X-Timestamp/X-Nonce/X-Signature`。
