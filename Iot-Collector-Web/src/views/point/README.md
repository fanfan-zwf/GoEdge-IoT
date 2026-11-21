# 点位配置管理

点位（Point）配置页面，管理数据采集点位。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 点位配置主页面 |

## 🎯 页面功能

- 点位列表展示（表格）
- 新增/编辑点位配置
- 删除点位
- 搜索筛选
- 分页
- 导入/导出 Excel
- WebSocket 实时值显示
- ECharts 实时曲线图

## 📝 数据模型

```typescript
interface Point {
  Id: number;           // 点位ID
  Tag: string;          // 点位标识
  Drive_Id: number;     // 所属驱动ID
  Config: string;       // 点位配置（地址、类型等）
  Conversion: string;   // 换算公式
  Creation_Time: string; // 创建时间
}
```

## 🔗 相关链接

- [返回页面目录](../README.md)
