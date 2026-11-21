# `src/api` — 后端接口封装与全局 HTTP 实例

上级文档：[前端总览 README](../../README.md)

本目录是与后端 `Iot-Collector-Service` 通信的唯一入口。**所有请求都必须复用 `config.ts` 里的全局 `http` 实例**，不要再 `axios.create` 第二个实例，否则会丢失 baseURL 动态解析、Bearer 注入与 401 自动刷新。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `config.ts` | 523 | 全局 `http` 实例 + 请求/响应拦截器 + 服务器地址解析 + 各业务 API 分组（config/auth/user/apikey/rbac）+ 导入导出 |
| `user.ts` | 43 | 仅 `UserResult` / `RefreshTokenResult` 类型定义（供 store 使用），**不含请求逻辑** |

> ⚠️ 本目录**没有** `routes.ts`。路由来自本地 `src/router/modules`，非后端接口。

## 二、⚠️ 最容易踩的坑

1. **只有一个 http 实例**：`config.ts:44` 的 `export const http` 是全局单例，`timeout:10000`，`baseURL` 在**请求拦截器里每次动态解析**（`resolveBaseURL()`），因此新增接口直接 `http.post/get`，切勿自带 baseURL。
2. **鉴权接口免注入 token**：`AUTH_FREE_PATHS = ["/auth/login","/auth/refresh-token"]`，`isAuthUrl()` 命中的请求不注入 Bearer、也不触发 401 刷新（否则会死循环）。
3. **401 单飞刷新 + 重放一次**：并发 401 时 `refreshPromise` 只发一次刷新请求（`config.ts:80`），其余等待同一 Promise；刷新成功后带新 token 重放原请求（`_retry` 标记保证只重放一次）；**刷新失败会真正登出**（动态 `import("@/store/modules/user")` 调 `logOut()`），而不是只弹提示。
4. **响应信封解包**：后端统一返回 `{Code,Msg,Data,Timestamp}`。响应拦截器里 `Code∈[200,300)` 直接返回整个 `res`（**注意是 `res` 不是 `res.Data`**，业务层自己取 `.Data`）；其余 code 按区间弹 success/info/warning/error 消息并 `reject`。
5. **blob 透传**：`responseType==="blob"`（导出 Excel）时**不走信封解包**，直接返回完整 response。后端异常可能以 `200 + application/json` 返回，`exportConfig` 里按 MIME 兜底识别并弹 `Msg`。
6. **服务器地址优先级**（`getBaseServer`，`config.ts:24`）：
   `localStorage 持久化的当前地址(baseServerOverride) > platform-config.json 的 BaseServer > 构建注入 VITE_BASE_SERVER > window.location.host 同源兜底`。
   `resolveBaseURL()`：解析出的 host 与当前页面同源时返回相对路径 `/api/v1.0`（天然继承协议/端口，避免混合内容），异机时拼 `http://{host}/api/v1.0`。
   **不要写死内网 IP**：`.env.development` 里的 `VITE_BASE_SERVER="172.21.1.231:8103"` 仅开发用，生产 `.env.production` 留空。
7. **登录返回字段是 snake_case**：`authApi.login` 的 `Data = {access_token, refresh_token, token_type, expires_in, access_key}`（后端风格），而 `user.ts` 里的 `UserResult.data` 是 camelCase（`accessToken/refreshToken/expires`）—— 那是 store 组装后的形态，不是接口原始返回。别混用。
8. **`systemInfo.get()` 是聚合接口**：并行 `Promise.allSettled` 拉 8 个 `/system/info/*` 原始接口，在前端拼成旧版视图模型 `{Timestamp,Host,OS,CPU,Memory,Disk}`，让 system-info/home 页面模板不改也能显示。单接口失败不影响整体（`pickData`）。裸接口（host/cpu/mem/…）保留供调试直接取原始 gopsutil 结果。
9. **RBAC 绑定是"集合替换"语义**：`rolePerms/subjectRoles/subjectPerms` 的 `get` 返回 `{ids:number[]}`，`set` 提交 `{...,ids:number[]}` 整体覆盖，不是增量 add/remove。
10. **`/point/read/value` 等仍发点位 id 数组**：后端已支持用 Tag 定位（详见后端 handler README），但前端 API 目前传的是 `pointIds:number[]` / `{PointId,...}`，迁移需同步改这里。

## 三、API 分组总览（`config.ts`）

| 导出 | 路径前缀 | 覆盖 |
| --- | --- | --- |
| `configApi.drive` | `/config/drive/*` | query/count/add/update/del/restart/search |
| `configApi.point` | `/config/point/*`、`/point/*` | CRUD + search + `readValue`/`writeValue` |
| `configApi.alarm` | `/config/alarm/*`、`/alarm/status` | CRUD + search + status |
| `configApi.alarmData` | `/config/alarm-data/*` | query/count |
| `configApi.history` | `/config/history/*` | CRUD |
| `configApi.historyData` | `/history/data/query` | InfluxDB 历史查询 |
| `configApi.tagTree` | `/tagtree/*` | children/points/rebuild |
| `configApi.systemConfig` | `/system/config/*` | read/write/reload（编辑 config.yaml） |
| `configApi.systemInfo` | `/system/info/*` | 裸接口 + `get()` 聚合 |
| `authApi` | `/auth/*` | login/refresh/logout |
| `userApi` | `/config/user/*` | 登录账号管理（密码仅新增/改密提交，列表不回传） |
| `apiKeyApi` | `/config/apikey/*` | AK/SK 接口账号（`secret_key` 仅新增/resetSecret 一次性返回） |
| `rbacApi` | `/config/rbac/*` | role/perm + rolePerms/subjectRoles/subjectPerms（role_type/subject_type：1=业务 2=API） |
| `exportConfig(type)` | `/config/export` | 带鉴权的 axios(blob) 下载（浏览器 `window.open` 无法带 Authorization 头） |
| `importConfig(file)` | `/config/import` | multipart 上传 Excel |

## 四、请求生命周期

```
业务调用 configApi.xxx → http.post(url, params)
  │ 请求拦截器: config.baseURL = resolveBaseURL(); 非鉴权 URL 注入 Bearer access_token
  ▼
后端 → 响应拦截器
  ├─ blob?      → 原样透传
  ├─ Code 2xx   → return res（业务自取 .Data）
  └─ 其它 Code  → showMessage + reject
  error 分支:
  ├─ 401 且非鉴权且未重放 → refreshAccessToken() 单飞 → 重放一次；失败则 logOut()
  └─ 其它 → 解析 res.Msg（blob 先转 JSON）→ showMessage + reject
```

## 五、扩展指引

新增接口：直接在 `config.ts` 对应的 API 分组里加一行 `xxx: (params) => http.post("/path", params)`，复用全局 `http`；**不要**新建 axios 实例、不要自带 baseURL、不要在业务层再解一层信封（拦截器已处理非 2xx）。

## 六、相关文档

- 后端接口与鉴权/RBAC：[后端 handler README](../../../Iot-Collector-Service/internal/handler/README.md)、[后端 pkg/rbac README](../../../Iot-Collector-Service/pkg/rbac/README.md)
- 运行时服务器地址配置：[src/utils README](../utils/README.md)（`serverAddr.ts`）
- 平台配置加载：[src/config README](../config/README.md)
- 令牌读写（`getToken/setToken`）：[src/utils README](../utils/README.md)（`auth.ts`）
