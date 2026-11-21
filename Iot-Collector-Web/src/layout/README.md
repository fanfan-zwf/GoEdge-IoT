# 布局组件

系统整体布局相关组件，基于 vue-pure-admin 框架。

## 📁 目录结构

```
layout/
├── components/          # 布局子组件
│   ├── lay-content/     # 内容区域
│   ├── lay-footer/      # 页脚
│   ├── lay-frame/       # iframe 容器
│   ├── lay-navbar/      # 顶部导航栏
│   ├── lay-notice/      # 通知中心
│   ├── lay-panel/       # 设置面板
│   ├── lay-search/      # 搜索功能
│   ├── lay-setting/     # 系统设置
│   ├── lay-sidebar/     # 侧边栏 ⭐
│   └── lay-tag/         # 标签页 ⭐
├── hooks/               # 布局相关 hooks
├── index.vue            # 布局主组件
├── frame.vue            # iframe 页面
├── redirect.vue         # 重定向页面
└── types.ts             # 类型定义
```

## 📁 子组件文档

| 目录 | 说明 | 文档 |
|------|------|------|
| [components/lay-sidebar/](./components/lay-sidebar/README.md) | 侧边栏组件 | [查看](./components/lay-sidebar/README.md) |
| [components/lay-tag/](./components/lay-tag/README.md) | 标签页组件 | [查看](./components/lay-tag/README.md) |
| [components/lay-navbar/](./components/lay-navbar/README.md) | 导航栏组件 | [查看](./components/lay-navbar/README.md) |
| [hooks/](./hooks/README.md) | 布局 Hooks | [查看](./hooks/README.md) |

## 📁 其他组件

| 组件 | 说明 |
|------|------|
| `lay-content/` | 主内容区域容器 |
| `lay-footer/` | 页脚组件 |
| `lay-frame/` | iframe 容器（用于嵌入外部页面） |
| `lay-notice/` | 通知中心组件 |
| `lay-panel/` | 设置面板（右侧抽屉） |
| `lay-search/` | 全局搜索组件（Ctrl+K） |
| `lay-setting/` | 系统设置组件（主题、布局等） |

## 🔗 相关链接

- [返回源码目录](../README.md)
