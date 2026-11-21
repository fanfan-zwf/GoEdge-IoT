/*
 * 作用: 配置字段格式校验，与后端 internal/repo/mysql/business.go 的正则一一对应
 *
 * 前后端各持一份正则，改任何一边都必须同步另一边，否则会出现「前端放行、后端 417」
 * 或反过来的割裂。每条常量都注明了对应的后端标识符，便于对照：
 *   - TAG_FORMAT          <- Regex_Tag_Format
 *   - CONVERSION_FORMAT   <- Regex_Conversion_Format
 *   - CONFIG_FORBIDDEN    <- Regex_Config_Forbidden
 *   - CONFIG_MAX_LENGTH   <- Config_Max_Length
 */

/** 点位标识：//设备id//标识符 或 /标识符（后端 Regex_Tag_Format 原文） */
export const TAG_FORMAT = "^(//[^/]+//[^/]+(/[^/]+)*|/[^/]+(/[^/]+)*)$";

/** 换算表达式：空、"0" 或含变量 x 的表达式（可用 round2(x,2) 保留小数；后端 Regex_Conversion_Format 原文） */
export const CONVERSION_FORMAT = "^(|0|[0-9a-zA-Z\\s\\+\\-\\*\\/\\.\\(\\),]*x[0-9a-zA-Z\\s\\+\\-\\*\\/\\.\\(\\),]*)$";

/**
 * 配置内容禁止片段（后端 Regex_Config_Forbidden 原文，去掉行首 (?i) 改用 RegExp 的 i 标志）
 * 拦：单引号、反引号、反斜杠、SQL 行注释（两个减号）与块注释标记、常见 SQL 关键字
 * 不拦双引号与分号：驱动配置是 JSON，Modbus 点位配置是 ;/: 分隔串
 * 反引号写成 u0060 转义，直接落在字符串里会与定界符冲突
 */
export const CONFIG_FORBIDDEN =
  "('|\u0060|\\\\|--|/\\*|\\*/|\\b(select|insert|update|delete|drop|alter|create|truncate|union|exec|declare|sleep|benchmark|load_file|information_schema|into\\s+outfile)\\b)";

/** 配置长度上限（后端 Config_Max_Length；后端按字节计，这里按字符计，纯 ASCII 配置两者一致） */
export const CONFIG_MAX_LENGTH = 4096;

const tagRe = new RegExp(TAG_FORMAT);
const conversionRe = new RegExp(CONVERSION_FORMAT);
const configForbiddenRe = new RegExp(CONFIG_FORBIDDEN, "i");
// 控制字符：允许 \t \n \r，其余 0x00-0x1f 一律拒绝（与后端 IndexFunc 的 r < 0x20 判断等价）
const controlCharRe = /[\x00-\x08\x0b\x0c\x0e-\x1f]/;

/** 校验结果：空串表示通过，其余为错误类别，由页面映射成对应语言的提示文案 */
export type FormatIssue =
  "" | "tooLong" | "controlChar" | "illegalChars" | "notJsonObject" | "invalidJson" | "tagFormat" | "conversionFormat";

/** issue -> i18n 文案键（六个语言文件里都得有对应条目）；空值不在此列，由表单的 required 规则负责 */
export const ISSUE_MESSAGE_KEY: Record<Exclude<FormatIssue, "">, string> = {
  tooLong: "common.configTooLong",
  controlChar: "common.configControlChar",
  illegalChars: "common.configIllegalChars",
  notJsonObject: "common.configNotJson",
  invalidJson: "common.configInvalidJson",
  tagFormat: "common.tagFormat",
  conversionFormat: "common.conversionFormat"
};

/**
 * 驱动配置校验（后端 Config__Validate_JSON）：必须是 JSON 对象
 * 驱动侧 Drive_Config_Switch 直接反序列化到结构体，非 JSON 只会拖到重启时炸
 */
export function checkDriveConfig(value: string): FormatIssue {
  const trimmed = (value ?? "").trim();
  if (!trimmed) return "";
  if (trimmed.length > CONFIG_MAX_LENGTH) return "tooLong";
  if (!trimmed.startsWith("{")) return "notJsonObject";
  try {
    const parsed = JSON.parse(trimmed);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
      return "notJsonObject";
    }
  } catch {
    return "invalidJson";
  }
  return "";
}

/**
 * 点位配置基础校验（后端 Config__Validate_Basic）：只挡脏数据，不校验格式
 * Modbus 的 : 分隔串、S7 的地址串、JSON、含空格等合法形态一律放行，格式交给驱动层解释
 */
export function checkPointConfig(value: string): FormatIssue {
  if (value === undefined || value === null || value === "") return "";
  if (value.length > CONFIG_MAX_LENGTH) return "tooLong";
  if (controlCharRe.test(value)) return "controlChar";
  if (configForbiddenRe.test(value)) return "illegalChars";
  return "";
}

/** 点位标识校验（后端 Regex_Tag_Format） */
export function checkTag(value: string): FormatIssue {
  if (!value) return "";
  return tagRe.test(value) ? "" : "tagFormat";
}

/** 换算表达式校验（后端 Regex_Conversion_Format：空、0、或含 x 的一元一次方程） */
export function checkConversion(value: string): FormatIssue {
  if (!value) return "";
  return conversionRe.test(value) ? "" : "conversionFormat";
}
