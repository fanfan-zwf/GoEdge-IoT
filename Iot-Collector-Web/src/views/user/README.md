# `src/views/user` — 登录用户（user_account）管理

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

管理**登录进前端管理台的账号**（`user_account` 表，AK + 密码），对接 [api 文档](../../api/README.md) 的 `userApi`。与 [apikey](../apikey/README.md)（第三方接口账号）是两套主体，但共用 `account.*` i18n 文案。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `index.vue` | 340 | 单页：列表 + 新增/编辑弹窗（含改密）+ 删除 |

## 二、表格列

`Id` · `Access_Key`（列名显示为"用户名"） · `Enable` · `Expire_At` · `Access_Token_TTL` · `Refresh_Token_TTL` · `Update_Time` · 操作（编辑 / 删除）

## 三、⚠️ 最容易踩的坑

1. **密码只在新增/改密时提交，列表永不回传**：`userApi.add` 带 `password`，`update` 的 `password` 可选（留空=不改密）。列表接口返回体里没有密码字段，别指望回填。
2. **服务端分页**：`index.vue:171` 是 `userApi.query({page,pageSize})` 全量拉取（`page:0,pageSize:0`），与 apikey 一样偏本地；但 `update/del` 走真实服务端。改分页先核对当前实际调用。
3. **`Access_Key` 是登录凭证本身**：后端用它 + bcrypt 校验（见 [pkg/security](../../../../Iot-Collector-Service/pkg/security/README.md)），改 AK 等于换登录名，历史 token 失效。
4. **删除需确认**：`ElMessageBox.confirm` 后 `userApi.del([row.Id])`。
5. **超管与 RBAC 的联动**：删除/停用某账号可能影响 [rbac](../rbac/README.md) 里以该账号为主体的授权；`enforce:true` 下误删白名单外唯一管理员会自锁，操作前确认还有可用管理员。

## 四、调用的 API（`userApi`）

`query` · `count` · `add` · `update` · `del([Id])`

## 五、相关文档

- [api README](../../api/README.md)
- 后端口令哈希：[pkg/security README](../../../../Iot-Collector-Service/pkg/security/README.md)
- 姊妹页：[apikey](../apikey/README.md)、[rbac](../rbac/README.md)
