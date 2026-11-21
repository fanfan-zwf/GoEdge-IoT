# 布局 Hooks

布局相关的组合式函数。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `useBoolean.ts` | 布尔值状态管理 |
| `useDataThemeChange.ts` | 主题切换逻辑 |
| `useLayout.ts` | 布局初始化逻辑 |
| `useMultiFrame.ts` | 多 iframe 管理 |
| `useNav.ts` | 导航相关逻辑 |
| `useTag.ts` | 标签页操作逻辑 ⭐ |
| `useTranslationLang.ts` | 语言切换逻辑 |

## 🎯 核心 Hook

### useTag

标签页相关操作：

```typescript
import { useTags } from "@/layout/hooks/useTag";

const {
  route,           // 当前路由
  router,          // 路由实例
  multiTags,       // 标签页列表
  activeIndex,     // 当前激活索引
  linkIsActive,    // 标签激活样式
  iconIsActive,    // 图标激活样式
  onContentFullScreen  // 内容全屏
} = useTags();
```

### useNav

导航相关：

```typescript
import { useNav } from "@/layout/hooks/useNav";

const {
  device,          // 设备类型（mobile/pc）
  collapse,        // 侧边栏折叠状态
  toggleCollapse   // 切换折叠
} = useNav();
```

## 🔗 相关链接

- [返回布局目录](../README.md)
- [返回源码目录](../../README.md)
