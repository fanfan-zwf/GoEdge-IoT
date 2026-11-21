# 插件

第三方插件集成。

## 📁 文件列表

| 文件 | 说明 |
|------|------|
| `i18n.ts` | 国际化插件（vue-i18n） |
| `echarts.ts` | ECharts 图表插件 |

## 📝 使用示例

```typescript
// 国际化
import { useI18n } from "vue-i18n";
const { t } = useI18n();
console.log(t("common.query")); // 查询

// ECharts
import { useECharts } from "@/plugins/echarts";
const chart = useECharts(dom);
chart.setOption({ ... });
```

## 🔗 相关链接

- [返回源码目录](../README.md)
