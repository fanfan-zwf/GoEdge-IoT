# 状态管理 (Pinia)

基于 Pinia 的全局状态管理。

## 📁 目录结构

```
store/
├── modules/
│   ├── app.ts           # 应用状态
│   ├── multiTags.ts     # 标签页状态 ⭐
│   ├── permission.ts    # 权限状态
│   ├── settings.ts      # 系统设置状态
│   └── user.ts          # 用户状态
├── index.ts             # Store 入口
└── utils.ts             # Store 工具函数
```

## 📁 模块说明

### app.ts - 应用状态

```typescript
import { useAppStoreHook } from "@/store/modules/app";

const appStore = useAppStoreHook();
// 设备类型
appStore.device; // "mobile" | "pc"
// 侧边栏状态
appStore.sidebarOpened; // boolean
```

### multiTags.ts - 标签页状态 ⭐

```typescript
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";

// 获取所有标签
const tags = useMultiTagsStoreHook().multiTags;

// 添加标签
useMultiTagsStoreHook().handleTags("push", {
  path: "/your-path",
  name: "YourRoute",
  params: { id: "1" },
  meta: { title: "标题" }
});

// 删除标签
useMultiTagsStoreHook().handleTags("splice", "/your-path");

// 清空标签
useMultiTagsStoreHook().handleTags("equal", []);
```

### permission.ts - 权限状态

```typescript
import { usePermissionStoreHook } from "@/store/modules/permission";

// 菜单列表
usePermissionStoreHook().wholeMenus;

// 路由列表
usePermissionStoreHook().flatteningRoutes;
```

### settings.ts - 系统设置

```typescript
import { useSettingStoreHook } from "@/store/modules/settings";

// 主题设置
useSettingStoreHook().theme;
```

### user.ts - 用户状态

```typescript
import { useUserStoreHook } from "@/store/modules/user";

// 用户信息
useUserStoreHook().userInfo;

// 登录
await useUserStoreHook().login(username, password);

// 登出
useUserStoreHook().loginOut();
```

## 🔗 相关链接

- [返回源码目录](../README.md)
