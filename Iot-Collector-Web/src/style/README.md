# 全局样式

系统全局样式定义。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `index.scss` | 样式入口文件 |
| `reset.scss` | 浏览器默认样式重置 |
| `sidebar.scss` | 侧边栏样式 ⭐ |
| `dark.scss` | 暗色主题适配 |
| `element-plus.scss` | Element Plus 样式覆盖 |
| `theme.scss` | CSS 变量主题 |
| `transition.scss` | 过渡动画效果 |

## 🎨 主题变量

```scss
// 使用 CSS 变量
:root {
  --el-color-primary: #409eff;
  --el-color-success: #67c23a;
  --el-color-warning: #e6a23c;
  --el-color-danger: #f56c6c;
  --el-color-info: #909399;
}
```

## 📝 使用示例

```vue
<style scoped>
.my-element {
  color: var(--el-color-primary);
  background: var(--el-bg-color);
}
</style>
```

## 🔗 相关链接

- [返回源码目录](../README.md)
