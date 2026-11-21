import { useNav } from "./useNav";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { watch, computed, onBeforeMount, type Ref } from "vue";
import { loadLocale } from "@/plugins/i18n";

/** 支持的语言列表 */
export const languageOptions = [
  { locale: "zh", label: "简体中文" },
  { locale: "en", label: "English" },
  { locale: "fr", label: "Français" },
  { locale: "ru", label: "Русский" },
  { locale: "es", label: "Español" },
  { locale: "ar", label: "العربية" }
];

export function useTranslationLang(ref?: Ref) {
  const { $storage, changeTitle, handleResize } = useNav();
  const { locale, t } = useI18n();
  const route = useRoute();

  async function translationLang(lang: string) {
    await loadLocale(lang);
    $storage.locale = { locale: lang };
    locale.value = lang;
    ref && handleResize(ref.value);
  }

  function translationCh() {
    translationLang("zh");
  }

  function translationEn() {
    translationLang("en");
  }

  watch(
    () => locale.value,
    () => {
      changeTitle(route.meta);
    }
  );

  onBeforeMount(() => {
    locale.value = $storage.locale?.locale ?? "zh";
  });

  /** 当前语言的全名标签（用于触发器文字显示） */
  const currentLangLabel = computed(
    () => languageOptions.find(l => l.locale === locale.value)?.label ?? locale.value
  );

  return {
    t,
    route,
    locale,
    currentLangLabel,
    translationCh,
    translationEn,
    translationLang
  };
}
