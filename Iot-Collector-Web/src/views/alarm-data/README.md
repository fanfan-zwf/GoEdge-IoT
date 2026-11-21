# 报警数据查询

报警数据查询页面，展示报警历史记录。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 报警数据查询主页面 |

## 🎯 页面功能

- 报警数据列表展示
- 时间范围筛选
- 分页查询
- 状态显示（报警/恢复）
- 动态标签页标题更新

## 📝 数据模型

```typescript
interface AlarmData {
  Id: number;           // 记录ID
  Alarm_Id: number;     // 报警配置ID
  Status: string;       // 状态（报警/恢复）
  Time: string;         // 报警时间
}
```

## 🎯 标签页标题更新

页面加载后会自动更新标签页标题为 `{Tag} 报警`：

```typescript
const updateTabTitle = async () => {
  // 查询报警配置获取 Tag
  const alarms = await configApi.alarm.query({ page: 1, pageSize: 1000 });
  const alarm = alarms.find(a => a.Id === routeAlarmId.value);
  
  if (alarm?.Tag) {
    const title = `${alarm.Tag} 报警`;
    // 更新路由 meta
    route.meta.title = title;
    // 更新浏览器标题
    document.title = `${title} | Iot-Collector`;
    // 更新标签页 store
    const tag = tags.find(t => t.path === route.path);
    if (tag) {
      tag.meta.title = title;
      useMultiTagsStoreHook().multiTags = [...tags];
    }
  }
};
```

## 🔗 相关链接

- [返回页面目录](../README.md)
- [报警配置页面](../alarm/README.md)
