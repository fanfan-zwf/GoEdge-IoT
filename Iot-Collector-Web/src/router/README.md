# 路由配置

Vue Router 路由定义和配置。

## 📁 目录结构

```
router/
├── modules/             # 路由模块
│   ├── home.ts          # 首页路由
│   └── remaining.ts     # 剩余路由（隐藏菜单）⭐
├── index.ts             # 路由入口 ⭐
└── utils.ts             # 路由工具函数
```

## 📁 文件说明

| 文件 | 说明 |
|------|------|
| `index.ts` | 路由实例创建、路由守卫、白名单配置 |
| `utils.ts` | 路由工具函数（初始化、格式化、权限处理等） |
| `modules/home.ts` | 首页路由定义 |
| `modules/remaining.ts` | 隐藏菜单路由（详情页、登录页等） |

## 🎯 核心功能

### 路由守卫

```typescript
// router/index.ts
router.beforeEach((to, from) => {
  // 1. 检查登录状态
  // 2. 初始化路由（首次加载）
  // 3. 添加标签页
  // 4. 设置页面标题
});
```

### 隐藏路由 (remaining.ts)

用于不在侧边栏菜单中显示，但需要独立标签页的页面：

```typescript
// 历史数据详情页
{
  path: "/history-data/:pointId",
  component: Layout,  // 必须使用 Layout 包裹
  children: [{
    name: "HistoryData",
    meta: { title: "历史数据", showLink: true }
  }]
}

// 报警数据详情页
{
  path: "/alarm-data/:alarmId",
  component: Layout,
  children: [{
    name: "AlarmData", 
    meta: { title: "报警数据", showLink: true }
  }]
}
```

## ⚠️ 注意事项

1. **Layout 包裹**: 需要侧边栏和标签页的页面必须用 `Layout` 组件包裹
2. **showLink 配置**: 设为 `true` 或省略，标签页激活只需匹配 `name`
3. **动态路由**: 使用 `:param` 格式定义动态参数

## 🔗 相关链接

- [返回源码目录](../README.md)
