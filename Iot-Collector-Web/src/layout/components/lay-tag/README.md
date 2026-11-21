# 标签页组件

LayTag 标签页导航组件，管理页面标签的显示、切换、关闭等操作。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.vue` | 标签页主组件 |
| `index.scss` | 标签页样式 |
| `components/TagChrome.vue` | Chrome 风格标签背景 |

## 🎯 核心功能

### 标签页管理

```typescript
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";

// 添加标签页
useMultiTagsStoreHook().handleTags("push", {
  path: "/your-path",
  name: "YourRoute",
  params: { id: "1" },
  meta: { title: "页面标题" }
});

// 删除标签页
useMultiTagsStoreHook().handleTags("splice", "/your-path");

// 获取所有标签
const tags = useMultiTagsStoreHook().multiTags;
```

### 标签页激活判断

| 条件 | 匹配方式 |
|------|----------|
| 无 `showLink: false` | 仅匹配路由 `name` |
| 有 `showLink: false` | 匹配 `path` + `query` + `params` |

### 右键菜单功能

- 刷新当前标签页
- 关闭当前标签页
- 关闭左侧标签页
- 关闭右侧标签页
- 关闭其他标签页
- 关闭所有标签页
- 内容区域全屏

## ⚠️ 注意事项

1. **params 类型匹配**: 添加标签页时，`params` 值必须转换为字符串类型
   ```typescript
   // ✅ 正确
   params: { alarmId: String(row.Id) }
   
   // ❌ 错误
   params: { alarmId: row.Id }  // 数字类型无法匹配
   ```

2. **动态路由标签**: 对于 `/alarm-data/:alarmId` 这类动态路由，每个参数值会创建独立标签

## 🔗 相关链接

- [返回布局目录](../../README.md)
- [返回源码目录](../../../README.md)
