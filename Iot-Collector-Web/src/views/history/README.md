# 历史配置管理

历史（History）配置页面，管理数据历史记录配置。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 历史配置主页面 |

## 🎯 页面功能

- 历史配置列表展示
- 新增/编辑历史配置
- 删除历史配置
- 搜索筛选
- 分页
- 导入/导出 Excel
- **查看历史数据**（跳转到历史数据页面）

## 📝 数据模型

```typescript
interface HistoryConfig {
  Id: number;           // 配置ID
  Point_Id: number;     // 关联点位ID
  Config: string;       // 历史配置（record/null）
  Creation_Time: string; // 创建时间
}
```

## 🔗 跳转逻辑

点击"查看数据"按钮会跳转到历史数据页面：

```typescript
const viewHistoryData = (row: any) => {
  // 添加标签页并跳转
  useMultiTagsStoreHook().handleTags("push", {
    path: `/history-data/${row.Point_Id}`,
    name: "HistoryData",
    params: { pointId: String(row.Point_Id) },
    meta: { title: point?.Tag || "历史数据" }
  });
  router.push({ name: "HistoryData", params: { pointId: String(row.Point_Id) } });
};
```

## 🔗 相关链接

- [返回页面目录](../README.md)
- [历史数据页面](../history-data/README.md)
