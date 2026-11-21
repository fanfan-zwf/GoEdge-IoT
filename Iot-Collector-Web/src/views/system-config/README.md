# `src/views/system-config` — 在线编辑后端 config.yaml

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

直接读写后端进程的 `configs/config.yaml` 文本，并提供"重载 / 下载 / 上传 / 重置"。对接 [api 文档](../../api/README.md) 的 `configApi.systemConfig`。后端实现见 [handler 文档](../../../../Iot-Collector-Service/internal/handler/README.md)，配置字段语义见 [config 文档](../../../../Iot-Collector-Service/internal/config/README.md)。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `index.vue` | 479 | YAML 文本编辑区 + 保存/重载/下载/上传/重置 + 变更确认 |

## 二、⚠️ 最容易踩的坑

1. **改完必须"重载"且未必全量生效**：`systemConfig.write({content})` 落盘后需 `systemConfig.reload()`；但**部分段（如 RBAC `super_biz_admin_ids`、`enable/enforce`）后端只在包 `init()` 的 `loadConfig()` 读取，运行期 reload 不一定重建**，需重启后端才生效。别把 reload 当万能热更。
2. **写的是整份原始 YAML 文本**：编辑区是全文本，任何缩进/语法错误会直接落盘，`reload` 时后端解析失败可能让服务回退默认或异常。上传前务必本地校验 YAML。
3. **`read` 返回的是文本不是结构化对象**：`loadConfig` 把后端原文塞进编辑区（`index.vue:128`），保存也是整段 `content` 提交，前端不做字段级 schema 校验。
4. **空内容保护**：`contentEmpty` 时拦截保存/上传，避免把配置清成空串。
5. **下载/重置语义**：下载导出当前文本，重置走确认弹窗恢复（`resetSuccess`）。重置/上传都会覆盖现网配置，属高危操作。
6. **权限门控**：该页受后端 `/system/config/*` 的 RBAC 主题保护，无权限时接口 403（前端 `enforce:true`），菜单也可能不显示。

## 三、调用的 API（`configApi.systemConfig`）

`read()` · `write({content})` · `reload()`；下载/上传复用导出导入式交互。

## 四、相关文档

- 配置结构与默认值：[internal/config README](../../../../Iot-Collector-Service/internal/config/README.md)
- 接口与鉴权：[handler README](../../../../Iot-Collector-Service/internal/handler/README.md)
- 姊妹页：[system-info 系统信息](../system-info/README.md)
