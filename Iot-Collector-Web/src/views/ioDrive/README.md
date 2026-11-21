# `src/views/ioDrive` — Modbus / Siemens 驱动 + 点位结构化表单

上级文档：[views 索引](../README.md) ｜ [前端总览](../../../README.md)

以**结构化字段表单**替代裸文本编辑，负责"字段 ⇄ `Config` 文本（JSON 或协议地址串）"的双向映射。后端消费见 [driver/modbus](../../../../Iot-Collector-Service/internal/driver/modbus/README.md)、[driver/siemens](../../../../Iot-Collector-Service/internal/driver/siemens/README.md)。

## 一、文件清单

| 文件 | 行数 | 职责 |
| --- | ---: | --- |
| `types.ts` | 209 | **集中式类型/默认值/选项表**：`ModbusConfig`/`SiemensConfig`/`ModbusPointConfig`/`SiemensPointConfig`、`*_DEFAULTS`、`S7_AREA_OPTIONS`/`S7_DATA_TYPE_OPTIONS`/`MODBUS_FUNCTION_OPTIONS`/`MODBUS_DATA_TYPE_OPTIONS`/`MODBUS_BYTE_ORDER_OPTIONS` + 通用 `parseDriveConfigJson(raw,defaults)`/`stringifyDriveConfig(cfg)` |
| `modbus/index.vue` | 574 | Modbus 完整页：驱动表 + 点位表 + 增删改 |
| `modbus/form.vue` | 88 | Modbus 驱动配置表单（可被 `drive/index.vue` 内嵌） |
| `modbus/point_form.vue` | 134 | Modbus 点位地址串表单（从机/功能码/寄存器[.子地址]/字节序/类型） |
| `siemens/index.vue` | 713 | Siemens 完整页 |
| `siemens/form.vue` | 121 | Siemens 驱动配置表单（Address/Rack/Slot/ConnectionType/TSAP…） |
| `siemens/point_form.vue` | 220 | Siemens 点位表单（Area/DB/Start/子地址/类型） |

## 二、⚠️ 最容易踩的坑

1. **字节序下拉含 5 个后端实现错误的值（高危）**：`MODBUS_BYTE_ORDER_OPTIONS` 覆盖 `Byte_Value` 全 14 项，但 `BADC / CDAB / BADCFEHG / CDABGHEF / HGFEDCBA` 这 5 个在后端**不能正确往返**，选中后 32/64 位值会静默错乱。能过入库校验、不报错。给用户加提示或直接禁用这 5 项。证据见 [pkg/conv README 坑1](../../../../Iot-Collector-Service/pkg/conv/README.md)。
2. **types.ts 只放类型/默认值/JSON 读写，映射逻辑在表单组件**：结构化字段 ⇄ `Config` 文本的转换内聚在各 `form.vue`/`point_form.vue`（靠第二个模型 `v-model:config`）。加驱动类型只扩 `types.ts`，别把解析逻辑塞进 types 或父页。
3. **`index.vue` 与 `form.vue` 是两种复用形态**：`index.vue` 是独立页（如 `/point-manage/modbus`），`form.vue` 只含驱动表单供 `drive/index.vue` 动态内嵌。别把内嵌场景指向 `index.vue`。
4. **点位地址串是冒号/空格文本，非 JSON**：Modbus 点位 `Config` 存 `1:3:100:ABCD:float32` 这类串，`point_form.vue` 负责拼/拆；后端 `modbus/new.go` 才真正解析。前端字段顺序/分隔符错了后端就报错。
5. **Siemens 有两种点位写法**：地址串（`DB1.DBD0 Real`）或 JSON（`{"Area":132,"DBNumber":1,...}`），`S7_AREA_OPTIONS` 等选项要与后端 Area 码一致。
6. **PDU / Packet_max 无需手填上限**：Siemens PDU 自动协商；Modbus `Packet_max` 有默认，乱填影响组包批量。
7. **字段名 PascalCase 对齐后端 Go 结构体 JSON 键**：改 TS 字段名要同步后端键名，否则 `parseDriveConfigJson` 取不到值回退默认。

## 三、核心函数

- `parseDriveConfigJson<T>(raw, defaults)`：安全解析 `Config` JSON，失败/缺字段回落 `defaults`。
- `stringifyDriveConfig(cfg)`：结构体 → JSON 文本入库。

## 四、相关文档

- 字节序/类型编码：[pkg/conv README](../../../../Iot-Collector-Service/pkg/conv/README.md)
- Modbus 地址解析与组包：[driver/modbus README](../../../../Iot-Collector-Service/internal/driver/modbus/README.md)
- Siemens Area/PDU：[driver/siemens README](../../../../Iot-Collector-Service/internal/driver/siemens/README.md)
- 姊妹页：[drive 通用驱动管理](../drive/README.md)、[point 点位管理](../point/README.md)
