# `src/views/rbac` — RBAC 权限管理（角色 / 权限字典 / 授权绑定）

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

对接 [api 文档](../../api/README.md) 的 `rbacApi`，维护后端极简 RBAC（类 MQTT 主题匹配）。后端语义务必对照 [pkg/rbac README](../../../../Iot-Collector-Service/pkg/rbac/README.md) 与 [service/rbac README](../../../../Iot-Collector-Service/internal/service/rbac/README.md)，前端 `topic_key` 的写法必须符合后端分段/通配规则。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `index.vue` | 84 | 纯容器：`el-tabs` 三个 Tab，均 `lazy`（进页面只挂载当前 Tab） |
| `role.vue` | 322 | 角色管理：业务/API 类型切换 + 角色 CRUD + 授予权限（穿梭框） |
| `perm.vue` | 263 | 权限字典：`perm_code` / `topic_key` / `effect`(allow/deny) / `match_sort` / `status` CRUD |
| `grant.vue` | 170 | 授权绑定：选主体 → 绑角色 / 直接授权（`subjectRoles`/`subjectPerms` 集合替换） |

> ⚠️ `role.vue`/`perm.vue`/`grant.vue` 是后拆分的独立文件，`index.vue` 只做 Tab 容器；不要再找"单页三 Tab"的旧实现。

## 二、⚠️ 最容易踩的坑

1. **`action` 维度已整体删除**：MySQL `t_permission`、后端 CRUD、本目录表格都**不再有 `action` 列**，别按旧版加回来。
2. **`topic_key` 是权限核心，必须对齐后端匹配语义**：`perm.vue` 表单 placeholder 示例 `/config/drive/query`。后端是"分段等长精确匹配 + `+`/`*`/`#` 通配 + deny 恒先于 allow + 默认拒绝"，写错主题串会导致规则永不命中或误放行。改前读 [pkg/rbac](../../../../Iot-Collector-Service/pkg/rbac/README.md)。
3. **`effect`：1=allow（绿）/ 其它=deny（红）**；`status`：1=启用。二者独立，停用某条不等于改成 deny。
4. **主体类型 `role_type`/`subject_type`：1=业务(登录用户) 2=API(接口账号)**，贯穿三个 Tab 的查询参数，切错类型看到的是另一套主体。
5. **绑定是"集合替换"**：`grant.vue` 的 `subjectRoles.set/subjectPerms.set` 提交完整 `ids` 数组整体覆盖，不是增量。穿梭框全量选项靠 `query({page:0,pageSize:0})`（后端约定不分页返回全量）。
6. **菜单/按钮权限来源已改真实数据**：登录响应下发真实 `roles/permissions`（不再硬编码 `*:*:*`），本目录写按钮用真实 `perm_code`（如 `config:rbac:role:add`）经 `v-perms` 门控。
7. **enforce:true + 白名单为空会自锁**：后端 `enforce:true`、`super_biz_admin_ids:[]`，靠 `rbac_admin_role_seed.sql` 给 admin 建 `super_admin` 角色并绑全部启用条目（当前 62 条）。清库/改字典后要重灌种子并重启后端（freecache 随重启清空），否则谁都看不到菜单。
8. **RBAC 菜单 `meta.roles` 是 `["*","super_admin"]`**：仅 `["*"]` 会让非超管 admin 的动态路由不注册 → 404。

## 三、调用的 API（`rbacApi`）

`role.query/count/add/update/del` · `perm.query/count/add/update/del` · `rolePerms.get/set` · `subjectRoles.get/set` · `subjectPerms.get/set`

## 四、相关文档

- 后端匹配引擎与全部坑：[pkg/rbac README](../../../../Iot-Collector-Service/pkg/rbac/README.md)
- 服务层决策 / 缓存 / 数据域：[service/rbac README](../../../../Iot-Collector-Service/internal/service/rbac/README.md)
- 管理接口（16 个 CRUD）：[handler README](../../../../Iot-Collector-Service/internal/handler/README.md)
- 前端指令 `v-perms/v-auth`：[directives README](../../directives/README.md)
