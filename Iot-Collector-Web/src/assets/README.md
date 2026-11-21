# 静态资源文件

存放图片、SVG、字体等静态资源。

## 📁 目录结构

| 目录 | 说明 |
|------|------|
| `iconfont/` | 图标字体文件（iconfont 图标库） |
| `login/` | 登录页面资源（背景图、头像 SVG） |
| `status/` | 状态页面图标（403、404、500 错误页） |
| `svg/` | 通用 SVG 图标文件 |
| `table-bar/` | 表格工具栏图标（折叠、拖拽、展开、刷新、设置） |

## 📝 使用方式

```vue
<!-- SVG 图标使用 -->
<img src="@/assets/login/bg.png" />

<!-- Iconfont 图标 -->
<IconifyIconOffline icon="ep:edit" />
```

## 🔗 相关链接

- [返回源码目录](../README.md)
