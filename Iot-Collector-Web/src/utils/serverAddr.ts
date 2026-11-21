import { storageLocal } from "@pureadmin/utils";

/**
 * 服务器地址管理：全部基于 localStorage，分两个独立 key
 * - HISTORY_KEY：JSON 数组，保存多个历史地址（去重、置顶、限量）
 * - CURRENT_KEY：单独保存当前选中的地址，刷新后据此恢复，不再丢失
 * 供 api/config.ts 与登录页共用，故放在中立的 utils 层。
 */
const HISTORY_KEY = "server-addr-history";
const CURRENT_KEY = "server-addr-current";
const MAX_HISTORY = 10;

/** 归一化输入：去掉协议前缀、首尾空白、末尾斜杠，得到 host:port */
export function normalizeServerAddr(value: string): string {
  return (value || "").trim().replace(/^https?:\/\//i, "").replace(/\/+$/, "");
}

/** 读取历史记录（异常/格式不符一律返回空数组） */
export function getServerHistory(): string[] {
  const raw = storageLocal().getItem<string[]>(HISTORY_KEY);
  return Array.isArray(raw) ? raw.filter(v => typeof v === "string" && v) : [];
}

/** 新增一条历史：归一化后去重置顶，超过上限截断 */
export function addServerHistory(value: string): string[] {
  const addr = normalizeServerAddr(value);
  if (!addr) return getServerHistory();
  const list = [addr, ...getServerHistory().filter(v => v !== addr)].slice(0, MAX_HISTORY);
  storageLocal().setItem(HISTORY_KEY, list);
  return list;
}

/** 删除单条历史 */
export function removeServerHistory(value: string): string[] {
  const list = getServerHistory().filter(v => v !== value);
  storageLocal().setItem(HISTORY_KEY, list);
  return list;
}

/** 清空全部历史 */
export function clearServerHistory(): string[] {
  storageLocal().removeItem(HISTORY_KEY);
  return [];
}

/** 读取当前持久化地址（未设置返回空串） */
export function getCurrentServerAddr(): string {
  const raw = storageLocal().getItem<string>(CURRENT_KEY);
  return typeof raw === "string" ? raw : "";
}

/** 持久化当前地址（归一化后写入），返回最终值 */
export function setCurrentServerAddr(value: string): string {
  const addr = normalizeServerAddr(value);
  if (!addr) return getCurrentServerAddr();
  storageLocal().setItem(CURRENT_KEY, addr);
  return addr;
}

/** 清除当前持久化地址（回退到 platform-config.json 默认） */
export function clearCurrentServerAddr(): void {
  storageLocal().removeItem(CURRENT_KEY);
}
