# `src/views/apikey` — APIKey（AK/SK 接口账号）管理

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

管理供第三方系统调用后端 `/config/*` 接口所用的 **AK/SK 接口账号**（`api_key` 表）。对接 [api 文档](../../api/README.md) 的 `apiKeyApi`，后端实现见 [handler/apikey.go](../../../../Iot-Collector-Service/internal/handler/README.md)。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `index.vue` | 409 | 单页：列表表格 + 新增/编辑弹窗 + 重置密钥 + 删除 |

## 二、表格列（`el-table-column prop`）

`Id` · `Access_Key` · `Name` · 密钥（不直接展示，走"重置"） · `Enable`（开关） · `Expire_At` · `Access_Token_TTL` · `Refresh_Token_TTL` · `Create_Time` · `Update_Time` · 操作（编辑 / 重置密钥 / 删除）

## 三、⚠️ 最容易踩的坑

1. **`secret_key` 只在新增/重置时一次性返回明文**：列表接口不返回 SK，`add`/`resetSecret` 的成功响应里带一次性明文，页面用弹窗提示"仅显示一次，请立即复制"。丢了只能重置。复制用 `navigator.clipboard`（非安全上下文可能 `copyFailed` 告警）。
2. **前端本地分页**：`index.vue:207` 调 `apiKeyApi.query({page:0,pageSize:0})` 拉全量后在前端切片，**不是服务端分页**。数据量大时是性能点，与 `user/rbac` 的服务端分页模式不同，改动别照搬。
3. **重置密钥需二次确认**：`resetSecretConfirm` 弹窗后才 `apiKeyApi.resetSecret({access_key})`，会作废旧 SK。
4. **TTL 字段是字符串时长**：`Access_Token_TTL/Refresh_Token_TTL` 对应后端 `conv.Duration`（如 `"15m"`），前端只做文本透传，格式非法会被后端归一/拒绝。
5. **`enable` 语义**：0 停用 / 1 启用，与登录账号 `userApi` 复用同一套 `account.*` i18n 文案。

## 四、调用的 API（全部来自 `apiKeyApi`）

`query` · `count`（本地分页时未用） · `add` · `update` · `resetSecret` · `del([Id])`

## 五、相关文档

- [api README](../../api/README.md)（`apiKeyApi` 与信封解包）
- 后端 AK/SK 生成与 HMAC 鉴权：[pkg/security README](../../../../Iot-Collector-Service/pkg/security/README.md)
- 姊妹页：[user 登录账号管理](../user/README.md)、[rbac 授权绑定](../rbac/README.md)
