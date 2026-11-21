# `src/views/system-info` — 系统运行信息监控

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

展示后端主机的 CPU / 内存 / Swap / 磁盘 等运行指标，支持定时刷新。对接 [api 文档](../../api/README.md) 的 `configApi.systemInfo.get()`（聚合接口）。后端数据来源见 [pkg/sysinfo README](../../../../Iot-Collector-Service/pkg/sysinfo/README.md)。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `index.vue` | 515 | `el-descriptions` 分组展示 + 刷新间隔选择 + 自动轮询 |

## 二、⚠️ 最容易踩的坑

1. **只调聚合接口 `systemInfo.get()`，不调裸接口**：`index.vue:317` 用的是前端聚合版（`config.ts` 里 `Promise.allSettled` 拼装的 `{Timestamp,Host,OS,CPU,Memory,Disk}`），页面模板按这套视图模型绑定。裸接口（`host/cpu/mem/disk/...`）仅供调试，页面没用。
2. **展示字段固定为 CPU/内存/Swap/磁盘四组**：`el-descriptions-item` 的 label 走 `systemInfo.*` i18n。后端 gopsutil 能采 24 类指标，但本页只渲染这四项，加字段要同时改聚合函数的返回结构。
3. **定时刷新有下限**：刷新选项含 1s/5s/10s（`index.vue:215-217`）。**后端 `CollectCPUPercent` 本身阻塞约 1s**（`DefaultCPUInterval`），选 1s 时几乎连续占用后端采集线程，别设更短。
4. **"err==nil 且全零"可能出现**：后端 sysinfo 对部分指标 `if err!=nil||v==nil {return T{},err}`，聚合版用 `?? 0` / `|| "-"` 兜底，看到 0 或 `-` 未必是故障，也可能该项确实取不到。
5. **组件卸载要清定时器**：自动轮询用 `setInterval`，离开页面务必 `clearInterval`，否则后台持续请求。
6. **失败仅弹 `loadFailed`**：`get()` 内部 `allSettled` 单项失败不影响整体返回，但网络级错误会被 axios 拦截器 reject 到 `catch`，只提示不阻断展示旧值。

## 三、渲染的数据结构（`get()` 返回）

```
Data: {
  Timestamp, Host:{Hostname}, OS:{Name,Version,Platform,Arch,Uptime},
  CPU:{ModelName,Frequency,Cores,PhysicalCores,UsagePercent},
  Memory:{Total,Used,Available,UsagePercent,SwapTotal,SwapUsed,SwapUsagePercent},
  Disk:{Total,Used,Free,UsagePercent}
}
```

## 四、相关文档

- 聚合与裸接口：[api README](../../api/README.md)
- 后端指标来源与坑：[pkg/sysinfo README](../../../../Iot-Collector-Service/pkg/sysinfo/README.md)
- 姊妹页：[system-config](../system-config/README.md)、[home 首页概览](../home/README.md)
