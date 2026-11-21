# 全局配置

系统全局配置导出。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.ts` | 全局配置导出（获取配置、存储命名空间等） |

## 📝 使用示例

```typescript
import { getConfig } from "@/config";

const config = getConfig();
console.log(config.Title); // 获取网站标题
```

## 🔗 相关链接

- [返回源码目录](../README.md)
