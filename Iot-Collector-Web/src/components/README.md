# 全局公共组件

可复用的全局组件，以 `Re` 前缀命名。

## 📁 组件列表

| 组件 | 说明 | 使用场景 |
|------|------|----------|
| `ReAnimateSelector/` | 动画选择器 | 选择 CSS 动画效果 |
| `ReAuth/` | 权限验证组件 | 按钮级权限控制 |
| `ReBarcode/` | 条形码组件 | 生成条形码 |
| `ReCol/` | 列布局组件 | 响应式列布局 |
| `ReCountTo/` | 数字动画组件 | 数字滚动效果 |
| `ReCropper/` | 图片裁剪组件 | 图片上传前裁剪 |
| `ReCropperPreview/` | 裁剪预览组件 | 预览裁剪结果 |
| `ReDialog/` | 对话框组件 | 全局弹窗 |
| `ReDrawer/` | 抽屉组件 | 侧边抽屉 |
| `ReFlicker/` | 闪烁效果组件 | 点闪烁动画 |
| `ReFlop/` | 翻牌器组件 | 数字翻牌效果 |
| `ReFlowChart/` | 流程图组件 | 流程图画布 |
| `ReIcon/` | 图标组件 | 图标选择器 |
| `ReImageVerify/` | 图形验证码组件 | 登录验证码 |
| `ReMap/` | 地图组件 | 高德地图集成 |
| `RePerms/` | 权限指令组件 | 权限判断 |
| `RePureTableBar/` | 表格工具栏组件 | 表格列设置 |
| `ReQrcode/` | 二维码组件 | 生成二维码 |
| `ReSeamlessScroll/` | 无缝滚动组件 | 列表滚动 |
| `ReSegmented/` | 分段控制器组件 | 选项切换 |
| `ReSelector/` | 选择器组件 | 通用选择 |
| `ReSplitPane/` | 分割面板组件 | 可拖拽分割面板 |
| `ReText/` | 文本组件 | 文本展示 |
| `ReTreeLine/` | 树形连线组件 | 树形结构连线 |
| `ReTypeit/` | 打字机效果组件 | 文字打字动画 |
| `ReVxeTableBar/` | VxeTable 工具栏组件 | VxeTable 表格工具 |

## 📝 使用示例

```vue
<!-- 图标组件 -->
<ReIcon icon="ep:edit" />

<!-- 数字动画 -->
<ReCountTo :endVal="1000" :duration="2000" />

<!-- 二维码 -->
<ReQrcode :text="'https://example.com'" />
```

## 🔗 相关链接

- [返回源码目录](../README.md)
