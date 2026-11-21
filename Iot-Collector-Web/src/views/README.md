# 页面组件

业务页面目录。

## 📁 目录结构

| 目录 | 说明 | 路由 | 文档 |
|------|------|------|------|
| [login/](./login/README.md) | 登录页面 | `/login` | [查看](./login/README.md) |
| [home/](./home/README.md) | 首页 | `/` | [查看](./home/README.md) |
| [point/](./point/README.md) | 点位配置管理 | `/point` | [查看](./point/README.md) |
| [drive/](./drive/README.md) | 驱动配置管理 | `/drive` | [查看](./drive/README.md) |
| [history/](./history/README.md) | 历史配置管理 | `/history` | [查看](./history/README.md) |
| [history-data/](./history-data/README.md) | 历史数据查询 | `/history-data/:pointId` | [查看](./history-data/README.md) |
| [alarm/](./alarm/README.md) | 报警配置管理 | `/alarm` | [查看](./alarm/README.md) |
| [alarm-data/](./alarm-data/README.md) | 报警数据查询 | `/alarm-data/:alarmId` | [查看](./alarm-data/README.md) |
| [empty/](./empty/README.md) | 空页面（示例） | `/empty` | [查看](./empty/README.md) |
| [error/](./error/README.md) | 错误页面 | `/403`, `/404`, `/500` | [查看](./error/README.md) |
| [ioDrive/](./ioDrive/README.md) | Modbus / Siemens 驱动+点位结构化表单 | 内嵌（router 无独立项） | [查看](./ioDrive/README.md) |
| [system-config/](./system-config/README.md) | 在线编辑 config.yaml | `/system/config` | [查看](./system-config/README.md) |
| [system-info/](./system-info/README.md) | 系统运行信息监控 | `/system/info` | [查看](./system-info/README.md) |
| [user/](./user/README.md) | 登录账号管理 | `/system/user` | [查看](./user/README.md) |
| [apikey/](./apikey/README.md) | APIKey（AK/SK）管理 | `/system/apikey` | [查看](./apikey/README.md) |
| [rbac/](./rbac/README.md) | RBAC 角色 / 权限 / 授权 | `/system/rbac` | [查看](./rbac/README.md) |

## 🎯 页面关系图

```
┌─────────────────────────────────────────────────────────────┐
│                        侧边栏菜单                            │
├─────────────────────────────────────────────────────────────┤
│  首页 (home)                                                 │
│  驱动配置 (drive) ──→ 点位配置 (point)                        │
│                        ├──→ 历史配置 (history)                │
│                        │       └──→ 历史数据 (history-data)   │
│                        └──→ 报警配置 (alarm)                  │
│                                └──→ 报警数据 (alarm-data)     │
└─────────────────────────────────────────────────────────────┘
```

## 📝 页面跳转模式

### 列表页 → 详情页

```typescript
// 在列表页中跳转到详情页
const viewDetail = (row: any) => {
  // 1. 添加标签页
  useMultiTagsStoreHook().handleTags("push", {
    path: `/detail/${row.Id}`,
    name: "DetailPage",
    params: { id: String(row.Id) },
    meta: { title: row.Name }
  });
  
  // 2. 路由跳转
  router.push({ name: "DetailPage", params: { id: String(row.Id) } });
};
```

### 详情页更新标签标题

```typescript
// 在详情页 onMounted 中更新标签标题
const updateTabTitle = async () => {
  // 1. 查询数据获取名称
  const data = await api.getDetail(id);
  
  // 2. 更新路由 meta
  route.meta.title = data.Name;
  
  // 3. 更新浏览器标题
  document.title = `${data.Name} | 系统`;
  
  // 4. 更新标签页 store
  const tags = useMultiTagsStoreHook().multiTags;
  const tag = tags.find(t => t.path === route.path);
  if (tag) {
    tag.meta.title = data.Name;
    useMultiTagsStoreHook().multiTags = [...tags];
  }
};
```

## 🔗 相关链接

- [返回源码目录](../README.md)
