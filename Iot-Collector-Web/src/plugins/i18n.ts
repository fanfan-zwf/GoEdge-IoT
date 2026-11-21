// 多组件库的国际化和本地项目国际化兼容（运行时 fetch 加载）
import { type I18n, createI18n } from "vue-i18n";
import type { App, WritableComputedRef } from "vue";
import { responsiveStorageNameSpace } from "@/config";
import { storageLocal } from "@pureadmin/utils";

// element-plus 国际化（静态导入，体积小）
import enLocale from "element-plus/es/locale/lang/en";
import zhLocale from "element-plus/es/locale/lang/zh-cn";
import frLocale from "element-plus/es/locale/lang/fr";
import ruLocale from "element-plus/es/locale/lang/ru";
import esLocale from "element-plus/es/locale/lang/es";
import arLocale from "element-plus/es/locale/lang/ar";

/** Element Plus locale 映射 */
const elementLocales: Record<string, any> = {
  zh: zhLocale,
  en: enLocale,
  fr: frLocale,
  ru: ruLocale,
  es: esLocale,
  ar: arLocale
};

/** locale 编码 → JSON 文件名映射 */
const localeFileMap: Record<string, string> = {
  zh: "zh-CN",
  en: "en",
  fr: "fr",
  ru: "ru",
  es: "es",
  ar: "ar"
};

/** 已加载的语言集合（防止重复请求） */
const loadedLocales = new Set<string>();

/**
 * 动态加载指定语言的 JSON 翻译文件（浏览器原生 fetch）
 * 文件位于 public/locales/{name}.json，经 Vite 静态服务直接提供给浏览器
 */
export async function loadLocale(locale: string): Promise<void> {
  if (loadedLocales.has(locale)) return;

  const filename = localeFileMap[locale] || locale;
  const base = import.meta.env.VITE_PUBLIC_PATH || "/";
  const url = `${base}locales/${filename}.json`;

  try {
    const response = await fetch(url);
    if (!response.ok) throw new Error(`HTTP ${response.status} for ${url}`);
    const messages = await response.json();

    // 合并 Element Plus locale（组件库内置翻译）
    const elLocale = elementLocales[locale];
    if (elLocale) {
      Object.assign(messages, elLocale);
    }

    (i18n.global as any).setLocaleMessage(locale, messages);
    loadedLocales.add(locale);
  } catch (err) {
    console.error(`[i18n] Failed to load locale "${locale}":`, err);
  }
}

/**
 * 国际化转换工具函数
 * - 对象格式 {zh:"", en:""} → 取当前 locale 对应值
 * - 带点的字符串（如 "menus.pureHome"）→ 尝试翻译
 * - 其他 → 原样返回
 */
export function transformI18n(message: any = ""): string {
  if (!message) return "";

  // 处理存储动态路由的 title，格式 {zh:"",en:""}
  if (typeof message === "object" && message !== null) {
    const locale: string | WritableComputedRef<string> | any = i18n.global.locale;
    return message[locale?.value ?? locale] || "";
  }

  if (typeof message === "string" && message.includes(".")) {
    // vue-i18n 在找不到 key 时返回 key 本身，我们可以用它判断是否真的翻译成功
    const rt = i18n.global as any;
    const translated = rt.t(message);
    return translated !== message ? translated : message;
  }
  return String(message);
}

/** 此函数只是配合 i18n Ally 插件来进行国际化智能提示，并无实际意义 */
export const $t = (key: string) => key;

export const i18n: I18n = createI18n({
  legacy: false,
  locale: storageLocal().getItem<StorageConfigs>(`${responsiveStorageNameSpace()}locale`)?.locale ?? "zh",
  fallbackLocale: "en",
  messages: {} // 运行时动态加载
});

/** 初始化时加载当前语言 + fallback 语言 */
export async function setupI18n(app: App): Promise<void> {
  const currentLocale = i18n.global.locale as any;
  const locale = typeof currentLocale === "object" ? currentLocale.value : currentLocale;

  // 并行加载当前语言和回退语言
  await Promise.all([loadLocale(locale), loadLocale("en")]);

  app.use(i18n);
}

/** 兼容旧用法：单独导出 useI18n 安装函数（不含异步加载） */
export function useI18n(app: App) {
  app.use(i18n);
}
