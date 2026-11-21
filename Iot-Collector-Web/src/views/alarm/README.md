# 报警配置管理

报警（Alarm）配置页面，管理数据报警规则配置。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 报警配置主页面 |

## 🎯 页面功能

- 报警配置列表展示
- 新增/编辑报警配置
- 删除报警配置
- 搜索筛选（支持 Tag 模糊搜索）
- 分页
- 导入/导出 Excel
- 实时状态显示（报警/恢复）
- **查看报警数据**（跳转到报警数据页面）
- **点击 Tag 链接**跳转到报警数据页面

## 📝 数据模型

```typescript
interface AlarmConfig {
  Id: number;           // 报警配置ID
  Point_Id: number;     // 关联点位ID
  Tag: string;          // 报警标识（可点击跳转）
  Config: string;       // 报警规则（expr 表达式）
  Group: number;        // 报警分组
  Creation_Time: string; // 创建时间
}
```

## 📝 报警规则格式

使用 `expr` 表达式引擎，支持以下格式：

| 格式 | 说明 | 示例 |
|------|------|------|
| `==值` | 等于时报警 | `==1` 值等于1时报警 |
| `<>值` | 不等于时报警 | `<>0` 值不等于0时报警 |
| `>值` | 大于时报警 | `>100` 值大于100时报警 |
| `<值` | 小于时报警 | `<10` 值小于10时报警 |

## 🔗 跳转逻辑

点击 Tag 列或"查看数据"按钮会跳转到报警数据页面：

```typescript
const viewAlarmData = (row: any) => {
  // 添加标签页并跳转
  useMultiTagsStoreHook().handleTags("push", {
    path: `/alarm-data/${row.Id}`,
    name: "AlarmData",
    params: { alarmId: String(row.Id) },
    meta: { title: row.Tag || "报警数据" }
  });
  router.push({ name: "AlarmData", params: { alarmId: String(row.Id) } });
};
```

## 🔗 相关链接

- [返回页面目录](../README.md)
- [报警数据页面](../alarm-data/README.md)
