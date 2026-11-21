import { ref, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { message } from "@/utils/message";
import { debounce } from "@pureadmin/utils";
import { useEventListener } from "@vueuse/core";
import type { FormInstance } from "element-plus";
import { useNav } from "@/layout/hooks/useNav";
import { useLayout } from "@/layout/hooks/useLayout";
import { useUserStoreHook } from "@/store/modules/user";
import { initRouter, getTopMenu } from "@/router/utils";
import { useTranslationLang } from "@/layout/hooks/useTranslationLang";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";

/**
 * 登录页共用逻辑：表单状态、登录动作、主题、多语言、布局 hooks
 * 现役登录页视觉为 VariantA（Deep Console）
 */
export function useLogin() {
  const router = useRouter();
  const loading = ref(false);
  const disabled = ref(false);
  const ruleFormRef = ref<FormInstance>();

  const { t } = useI18n();
  const { initStorage } = useLayout();
  initStorage();

  const { dataTheme, themeMode, dataThemeChange } = useDataThemeChange();
  dataThemeChange(themeMode.value);

  const { title, getDropdownItemStyle, getDropdownItemClass } = useNav();
  const { locale, currentLangLabel, translationLang } = useTranslationLang();

  const ruleForm = reactive({
    username: "admin",
    password: "admin123"
  });

  const onLogin = async (formEl: FormInstance | undefined) => {
    if (!formEl) return;
    await formEl.validate(valid => {
      if (valid) {
        loading.value = true;
        useUserStoreHook()
          .loginByUsername({
            username: ruleForm.username,
            password: ruleForm.password
          })
          .then(async () => {
            await initRouter();
            disabled.value = true;
            router.push(getTopMenu(true).path).then(() => {
              message(t("login.pureLoginSuccess"), { type: "success" });
            });
          })
          .catch(err => {
            message(err?.message || t("login.pureLoginFail"), {
              type: "error"
            });
          })
          .finally(() => {
            disabled.value = false;
            loading.value = false;
          });
      }
    });
  };

  const immediateDebounce: any = debounce(
    (formRef: FormInstance | undefined) => onLogin(formRef),
    1000,
    true
  );

  useEventListener(document, "keydown", ({ code }) => {
    if (
      ["Enter", "NumpadEnter"].includes(code) &&
      !disabled.value &&
      !loading.value
    )
      immediateDebounce(ruleFormRef.value);
  });

  return {
    // 表单
    ruleForm,
    ruleFormRef,
    loading,
    disabled,
    onLogin,
    // 主题
    dataTheme,
    themeMode,
    dataThemeChange,
    // 语言
    locale,
    currentLangLabel,
    translationLang,
    // 布局
    title,
    getDropdownItemStyle,
    getDropdownItemClass,
    // i18n
    t
  };
}
