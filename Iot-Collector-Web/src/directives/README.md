# 全局指令

自定义 Vue 指令。

## 📁 指令列表

| 目录 | 指令 | 说明 |
|------|------|------|
| `auth/` | `v-auth` | 权限验证指令 |
| `copy/` | `v-copy` | 复制指令 |
| `longpress/` | `v-longpress` | 长按指令 |
| `optimize/` | `v-optimize` | 防抖节流指令 |
| `perms/` | `v-perms` | 权限指令 |
| `ripple/` | `v-ripple` | 水波纹效果指令 |

## 📝 使用示例

```vue
<!-- 权限指令 -->
<el-button v-auth="'admin'">管理员按钮</el-button>

<!-- 复制指令 -->
<el-button v-copy="text">复制文本</el-button>

<!-- 长按指令 -->
<el-button v-longpress="handleLongPress">长按我</el-button>

<!-- 水波纹效果 -->
<el-button v-ripple>水波纹按钮</el-button>
```

## 🔗 相关链接

- [返回源码目录](../README.md)
