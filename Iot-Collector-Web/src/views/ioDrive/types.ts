/**
 * 驱动配置 TypeScript 类型定义
 * 与后端 Go 结构体 Config_type 字段名一一对应
 */

// ==================== 驱动 JSON 配置读写 ====================

/**
 * JSON 文本 → 结构化字段：以 defaults 的键为准，只挑出已知字段，避免脏键写回配置。
 * 空串、非法 JSON、非对象一律返回 null，由调用方决定回落默认值。
 */
export const parseDriveConfigJson = <T extends object>(raw: string, defaults: T): Partial<T> | null => {
  const text = (raw ?? "").trim();
  if (!text) return null;
  let obj: unknown;
  try {
    obj = JSON.parse(text);
  } catch {
    return null;
  }
  if (!obj || typeof obj !== "object" || Array.isArray(obj)) return null;
  const source = obj as Record<string, unknown>;
  const picked: Record<string, unknown> = {};
  for (const key of Object.keys(defaults)) {
    if (source[key] !== undefined) picked[key] = source[key];
  }
  return picked as Partial<T>;
};

/** 结构化字段 → JSON 文本（后端 Drive_Config_Switch 直接反序列化该结构） */
export const stringifyDriveConfig = <T extends object>(cfg: T): string => JSON.stringify(cfg);

// ==================== Modbus TCP ====================

/** Modbus TCP 驱动 JSON 配置（对应后端 Config_type） */
export interface ModbusConfig {
  Address: string; // IP:端口，如 "192.168.1.1:502"
  Retry_timeout: string; // 重试间隔，如 "3s"
  Connect_timeout: string; // 连接超时，如 "10s"
  Response_timeout: string; // 响应超时，如 "10s"
  Delay_between_polls: string; // 轮询间隔，如 "0s"
  Packet_max: number; // 组包字节数，默认 120
}

/** Modbus 配置默认值 */
export const MODBUS_CONFIG_DEFAULTS: ModbusConfig = {
  Address: ":502",
  Retry_timeout: "3s",
  Connect_timeout: "10s",
  Response_timeout: "10s",
  Delay_between_polls: "0s",
  Packet_max: 120
};

// ==================== Siemens S7 ====================

/** Siemens S7 驱动 JSON 配置（对应后端 Config_type） */
export interface SiemensConfig {
  Address: string; // IP 地址
  Rack: number; // 机架号，默认 0
  Slot: number; // 槽位号，默认 2
  Retry_timeout: string; // 重试间隔，如 "3s"
  Connect_timeout: string; // 连接超时，如 "10s"
  Response_timeout: string; // 响应超时，如 "10s"
  Delay_between_polls: string; // 轮询间隔，如 "0s"
  ConnectionType: number; // 连接类型，1=PG 2=OP 3=Basic，默认 2
  Max_reconnect_interval: string; // 重连最大间隔（指数退避封顶），如 "30s"
  Local_TSAP: number; // 本地 TSAP 覆盖（snap7 原生 uint16，十进制，如 258=0x0102），0=按 Rack/Slot 自动推导
  Remote_TSAP: number; // 远端 TSAP 覆盖（snap7 原生 uint16，十进制，如 CP343-1 Advanced 自定义值），0=自动推导
}

/** Siemens S7 配置默认值 */
export const SIEMENS_CONFIG_DEFAULTS: SiemensConfig = {
  Address: "",
  Rack: 0,
  Slot: 2,
  Retry_timeout: "3s",
  Connect_timeout: "10s",
  Response_timeout: "10s",
  Delay_between_polls: "0s",
  ConnectionType: 2,
  Max_reconnect_interval: "30s",
  Local_TSAP: 0,
  Remote_TSAP: 0
};

// ==================== Siemens S7 点位配置 ====================

/** Siemens S7 点位配置（对应后端 S7Points_type） */
export interface SiemensPointConfig {
  Area: number; // 存储区 0x81=I 0x82=Q 0x83=M 0x84=DB
  DBNumber: number; // DB号
  Start: number; // 字节偏移
  Child_Address: number; // 子地址（位偏移）
  Type: number; // 采集类型（PLC 侧原始类型，对齐 TIA Portal）
}

/** 存储区选项 */
export const S7_AREA_OPTIONS = [
  { value: 0x81, label: "I (输入)" },
  { value: 0x82, label: "Q (输出)" },
  { value: 0x83, label: "M (标志)" },
  { value: 0x84, label: "DB (数据块)" }
];

/**
 * S7 采集类型选项（TIA Portal 声明类型 ↔ Go 类型 ↔ 代码值 ↔ 地址后缀）
 * 代码值与后端 conv.TYPE_* 一一对应；驱动会依据本项决定缓冲区长度与字节流解析方式
 * suffix 用于拼接 S7 地址字符串（如 DB1.DBD0 → Real）
 */
export const S7_DATA_TYPE_OPTIONS = [
  { value: 1, label: "Bool (位)", tia: "Bool", suffix: "X" },
  { value: 2, label: "SInt (int8)", tia: "SInt", suffix: "B" },
  { value: 3, label: "USInt (uint8)", tia: "USInt", suffix: "B" },
  { value: 4, label: "Int (int16)", tia: "Int", suffix: "W" },
  { value: 5, label: "UInt (uint16)", tia: "UInt", suffix: "W" },
  { value: 6, label: "DInt (int32)", tia: "DInt", suffix: "D" },
  { value: 7, label: "UDInt (uint32)", tia: "UDInt", suffix: "D" },
  { value: 8, label: "LInt (int64)", tia: "LInt", suffix: "L" },
  { value: 9, label: "ULInt (uint64)", tia: "ULInt", suffix: "L" },
  { value: 12, label: "Real (float32)", tia: "Real", suffix: "D" },
  { value: 13, label: "LReal (float64)", tia: "LReal", suffix: "L" },
  { value: 15, label: "String", tia: "String", suffix: "B" }
];

/**
 * S7 输出类型选项（写入数据库/上报的最终类型，由用户自定义）
 * 与采集类型可以不同，中间由 Conversion 表达式换算衔接
 */
export const S7_VALUE_TYPE_OPTIONS = [
  { value: 1, label: "bool" },
  { value: 2, label: "int8" },
  { value: 3, label: "uint8" },
  { value: 4, label: "int16" },
  { value: 5, label: "uint16" },
  { value: 6, label: "int32" },
  { value: 7, label: "uint32" },
  { value: 8, label: "int64" },
  { value: 9, label: "uint64" },
  { value: 12, label: "float32" },
  { value: 13, label: "float64" },
  { value: 15, label: "string" }
];

/** S7 点位配置默认值 */
export const SIEMENS_POINT_DEFAULTS: SiemensPointConfig = {
  Area: 0x84, // DB
  DBNumber: 1,
  Start: 0,
  Child_Address: 0,
  Type: 12 // Real
};

// ==================== Modbus TCP 点位配置 ====================

/**
 * Modbus TCP 点位配置（对应后端 Point_Config_Switch 冒号分隔格式）
 * 字符串格式: 从机地址:功能码:寄存器地址[.位偏移]:字节序:数据类型
 * 示例: 1:3:100:ABCD:float32 / 1:1:100.5:ABCD:bool
 */
export interface ModbusPointConfig {
  SlaveID: number; // 从机地址 0-255
  Function: number; // 功能码 1-4
  Address: number; // 寄存器地址 0-65535
  Child_Address: number; // 位偏移 0-15（仅 bool）
  ByteOrder: string; // 字节序，如 ABCD
  DataType: string; // 数据类型，如 float32
}

/** 功能码选项（后端正则要求单个数字 1-4） */
export const MODBUS_FUNCTION_OPTIONS = [
  { value: 1, label: "01 - 读线圈 (Coils)" },
  { value: 2, label: "02 - 读离散输入 (Discrete Inputs)" },
  { value: 3, label: "03 - 读保持寄存器 (Holding Registers)" },
  { value: 4, label: "04 - 读输入寄存器 (Input Registers)" }
];

/** 数据类型选项（对应后端 conv.ValueTypeCode） */
export const MODBUS_DATA_TYPE_OPTIONS = [
  { value: "bool", label: "bool" },
  { value: "int16", label: "int16" },
  { value: "uint16", label: "uint16" },
  { value: "int32", label: "int32" },
  { value: "uint32", label: "uint32" },
  { value: "float32", label: "float32" },
  { value: "float64", label: "float64" }
];

/** 字节序选项（对应后端 conv.Byte_Value） */
export const MODBUS_BYTE_ORDER_OPTIONS = [
  { value: "AB", label: "AB" },
  { value: "BA", label: "BA" },
  { value: "ABCD", label: "ABCD" },
  { value: "BADC", label: "BADC" },
  { value: "CDAB", label: "CDAB" },
  { value: "DCBA", label: "DCBA" },
  { value: "ABCDEFGH", label: "ABCDEFGH" },
  { value: "DCBAHGFE", label: "DCBAHGFE" }
];

/** Modbus 点位配置默认值 */
export const MODBUS_POINT_DEFAULTS: ModbusPointConfig = {
  SlaveID: 1,
  Function: 3,
  Address: 0,
  Child_Address: 0,
  ByteOrder: "ABCD",
  DataType: "float32"
};
