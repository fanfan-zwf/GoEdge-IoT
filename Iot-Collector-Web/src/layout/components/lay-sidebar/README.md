# 侧边栏组件

侧边栏导航菜单相关组件。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `NavVertical.vue` | 垂直导航菜单（默认） |
| `NavHorizontal.vue` | 水平导航菜单（顶部） |
| `NavMix.vue` | 混合导航菜单 |

## 📁 components/ 子组件

| 组件 | 说明 |
|------|------|
| `SidebarItem.vue` | 菜单项组件（递归渲染） |
| `SidebarLinkItem.vue` | 菜单链接项 |
| `SidebarBreadCrumb.vue` | 面包屑导航 |
| `SidebarLogo.vue` | Logo 组件 |
| `SidebarLeftCollapse.vue` | 左侧折叠按钮 |
| `SidebarCenterCollapse.vue` | 中间折叠按钮 |
| `SidebarTopCollapse.vue` | 顶部折叠按钮 |
| `SidebarExtraIcon.vue` | 菜单额外图标 |
| `SidebarFullScreen.vue` | 全屏按钮 |

## 🎯 核心功能

### 菜单渲染流程
```
路由配置 (router/modules/)
    ↓
权限过滤 (store/modules/permission.ts)
    ↓
菜单树构建 (utils/tree.ts)
    ↓
SidebarItem 递归渲染
    ↓
SidebarLinkItem 生成链接
```

### 导航模式
- **垂直模式**: 侧边栏展示所有菜单
- **水平模式**: 顶部展示一级菜单，子菜单下拉
- **混合模式**: 顶部展示一级，侧边栏展示二级及以下

## 🔗 相关链接

- [返回布局目录](../../README.md)
- [返回源码目录](../../../README.md)
