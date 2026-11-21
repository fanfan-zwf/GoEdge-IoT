# 工具函数

通用工具函数集合。

## 📁 目录结构

```
utils/
├── http/                # HTTP 请求封装
│   ├── index.ts         # axios 实例配置
│   └── types.ts         # 类型定义
├── localforage/         # 本地存储封装
│   └── index.ts
├── progress/            # 进度条
│   └── index.ts
├── auth.ts              # 认证工具（Token 管理）
├── message.ts           # 消息提示封装
├── mitt.ts              # 事件总线
├── sso.ts               # 单点登录
└── tree.ts              # 树形数据工具
```

## 📝 使用示例

### HTTP 请求

```typescript
import http from "@/utils/http";

// GET 请求
const res = await http.get("/api/users");

// POST 请求
const res = await http.post("/api/users", { name: "test" });
```

### 消息提示

```typescript
import { message } from "@/utils/message";

message("操作成功", { type: "success" });
message("操作失败", { type: "error" });
message("警告信息", { type: "warning" });
```

### 认证工具

```typescript
import { getAccessToken, setToken } from "@/utils/auth";

// 获取 Token
const token = getAccessToken();

// 设置 Token
setToken({ accessToken: "xxx", refreshToken: "xxx" });
```

### 事件总线

```typescript
import { emitter } from "@/utils/mitt";

// 监听事件
emitter.on("eventName", (data) => {
  console.log(data);
});

// 触发事件
emitter.emit("eventName", { id: 1 });

// 取消监听
emitter.off("eventName");
```

## 🔗 相关链接

- [返回源码目录](../README.md)
