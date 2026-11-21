# 历史数据查询

历史数据查询页面，展示点位的历史记录数据。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 历史数据查询主页面 |

## 🎯 页面功能

- 历史数据列表展示
- 时间范围筛选
- 分页查询
- 数据导出

## 📝 数据模型

```typescript
interface HistoryData {
  time: string;   // 时间戳
  value: any;     // 数据值
  msg: string;    // 状态信息
}
```

## 🔗 相关链接

- [返回页面目录](../README.md)
- [历史配置页面](../history/README.md)
