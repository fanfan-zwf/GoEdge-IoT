<script setup lang="ts">
import { ref, unref, watch, reactive, computed, nextTick, onUnmounted, onBeforeMount } from "vue";
import { useI18n } from "vue-i18n";
import { emitter } from "@/utils/mitt";
import LayPanel from "../lay-panel/index.vue";
import { useNav } from "@/layout/hooks/useNav";
import { useAppStoreHook } from "@/store/modules/app";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";
import Segmented, { type OptionsType } from "@/components/ReSegmented";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";
import { useIndustrialSkin } from "@/layout/hooks/useIndustrialSkin";
import { useDark, useGlobal, debounce, isNumber } from "@pureadmin/utils";

import Check from "~icons/ep/check";
import LeftArrow from "~icons/ri/arrow-left-s-line?width=20&height=20";
import RightArrow from "~icons/ri/arrow-right-s-line?width=20&height=20";
import DayIcon from "@/assets/svg/day.svg?component";
import DarkIcon from "@/assets/svg/dark.svg?component";
import SystemIcon from "@/assets/svg/system.svg?component";

const { t } = useI18n();
const { device } = useNav();
const { isDark } = useDark();
const { $storage } = useGlobal<GlobalPropertiesApi>();

const mixRef = ref();
const verticalRef = ref();
const horizontalRef = ref();

const { dataTheme, themeMode, layoutTheme, themeColors, toggleClass, dataThemeChange, setLayoutThemeColor } =
  useDataThemeChange();

/* body添加layout属性，作用于src/style/sidebar.scss */
if (unref(layoutTheme)) {
  const layout = unref(layoutTheme).layout;
  const theme = unref(layoutTheme).theme;
  document.documentElement.setAttribute("data-theme", theme);
  setMenuLayout(layout);
}

/** 页签风格默认为谷歌风格 */
const tagsStyleValue = ref($storage.configure?.tagsStyle ?? "chrome");

const logoVal = ref($storage.configure?.showLogo ?? true);

const watermarkConfigs = reactive({
  enable: $storage.configure.watermark,
  text: $storage.configure.watermarkText
});

const settings = reactive({
  greyVal: $storage.configure.grey,
  weakVal: $storage.configure.weak,
  tabsVal: $storage.configure.hideTabs,
  showLogo: $storage.configure.showLogo,
  tagsStyle: $storage.configure.tagsStyle,
  hideFooter: $storage.configure.hideFooter,
  multiTagsCache: $storage.configure.multiTagsCache,
  stretch: $storage.configure.stretch
});

const getThemeColorStyle = computed(() => {
  return color => {
    return { background: color };
  };
});

/** 当网页整体为暗色风格时不显示亮白色主题配色切换选项 */
const showThemeColors = computed(() => {
  return themeColor => {
    return themeColor === "light" && isDark.value ? false : true;
  };
});

function storageConfigureChange<T>(key: string, val: T): void {
  const storageConfigure = $storage.configure;
  storageConfigure[key] = val;
  $storage.configure = storageConfigure;
}

/** 灰色模式设置 */
const greyChange = (value): void => {
  const htmlEl = document.querySelector("html");
  toggleClass(settings.greyVal, "html-grey", htmlEl);
  storageConfigureChange("grey", value);
};

/** 色弱模式设置 */
const weekChange = (value): void => {
  const htmlEl = document.querySelector("html");
  toggleClass(settings.weakVal, "html-weakness", htmlEl);
  storageConfigureChange("weak", value);
};

/** 隐藏标签页设置 */
const tagsChange = () => {
  const showVal = settings.tabsVal;
  storageConfigureChange("hideTabs", showVal);
  emitter.emit("tagViewsChange", showVal as unknown as string);
};

/** 隐藏页脚设置 */
const hideFooterChange = () => {
  const hideFooter = settings.hideFooter;
  storageConfigureChange("hideFooter", hideFooter);
};

/** 标签页持久化设置 */
const multiTagsCacheChange = () => {
  const multiTagsCache = settings.multiTagsCache;
  storageConfigureChange("multiTagsCache", multiTagsCache);
  useMultiTagsStoreHook().multiTagsCacheChange(multiTagsCache);
};

function onChange({ option }) {
  const { value } = option;
  tagsStyleValue.value = value;
  storageConfigureChange("tagsStyle", value);
  emitter.emit("tagViewsTagsStyle", value);
}

function onWatermarkSwitchChange(value) {
  storageConfigureChange("watermark", value);
}

function onWatermarkInputChange(text) {
  storageConfigureChange("watermarkText", text);
}

/** 侧边栏Logo */
function logoChange() {
  unref(logoVal) ? storageConfigureChange("showLogo", true) : storageConfigureChange("showLogo", false);
  emitter.emit("logoChange", unref(logoVal));
}

function setFalse(Doms): any {
  Doms.forEach(v => {
    toggleClass(false, "is-select", unref(v));
  });
}

/** 页宽 */
const stretchTypeOptions = computed<Array<OptionsType>>(() => {
  return [
    {
      label: t("panel.pureStretchFixed"),
      tip: t("panel.pureStretchFixedTip"),
      value: "fixed"
    },
    {
      label: t("panel.pureStretchCustom"),
      tip: t("panel.pureStretchCustomTip"),
      value: "custom"
    }
  ];
});

const setStretch = value => {
  settings.stretch = value;
  storageConfigureChange("stretch", value);
};

const stretchTypeChange = ({ option }) => {
  const { value } = option;
  value === "custom" ? setStretch(1440) : setStretch(false);
};

const pClass = computed(() => {
  return ["mb-3!", "font-medium", "text-sm", "dark:text-white"];
});

/** 经典主题色块（其余） */
const classicColors = computed(() => themeColors.value.filter(v => v.group !== "industrial"));

/** 工业皮肤选择器（顶部独立区，已合并浅色/深色到皮肤里） */
const { skin: currentSkin, skins: industrialSkins, setSkin } = useIndustrialSkin();

/** 回退到 vue-pure-admin 经典主题，同时指定明暗 */
function pickClassicMode(mode: "light" | "dark") {
  setLayoutThemeColor(mode === "light" ? "light" : "default");
  dataTheme.value = mode === "dark";
  themeMode.value = mode;
  dataThemeChange(mode);
}

/** 主题色 激活选择项 */
const getThemeColor = computed(() => {
  return current => {
    if (current === layoutTheme.value.theme && layoutTheme.value.theme !== "light") {
      return "#fff";
    } else if (current === layoutTheme.value.theme && layoutTheme.value.theme === "light") {
      return "#1d2b45";
    } else {
      return "transparent";
    }
  };
});
const themeOptions = computed<Array<OptionsType>>(() => {
  return [
    {
      label: t("panel.pureThemeModeLight"),
      icon: DayIcon,
      theme: "light",
      tip: t("panel.pureThemeModeLightTip"),
      iconAttrs: { fill: isDark.value ? "#fff" : "#000" }
    },
    {
      label: t("panel.pureThemeModeDark"),
      icon: DarkIcon,
      theme: "dark",
      tip: t("panel.pureThemeModeDarkTip"),
      iconAttrs: { fill: isDark.value ? "#fff" : "#000" }
    },
    {
      label: t("panel.pureThemeModeSystem"),
      icon: SystemIcon,
      theme: "system",
      tip: t("panel.pureThemeModeSystemTip"),
      iconAttrs: { fill: isDark.value ? "#fff" : "#000" }
    }
  ];
});

const markOptions = computed<Array<OptionsType>>(() => {
  return [
    {
      label: t("panel.pureTagsStyleSmart"),
      tip: t("panel.pureTagsStyleSmartTip"),
      value: "smart"
    },
    {
      label: t("panel.pureTagsStyleCard"),
      tip: t("panel.pureTagsStyleCardTip"),
      value: "card"
    },
    {
      label: t("panel.pureTagsStyleChrome"),
      tip: t("panel.pureTagsStyleChromeTip"),
      value: "chrome"
    }
  ];
});

/** 设置菜单布局 */
function setMenuLayout(layout: string) {
  layoutTheme.value.layout = layout;
  window.document.body.setAttribute("layout", layout);
  $storage.layout = {
    layout,
    theme: layoutTheme.value.theme,
    darkMode: $storage.layout?.darkMode,
    sidebarStatus: $storage.layout?.sidebarStatus,
    epThemeColor: $storage.layout?.epThemeColor,
    themeColor: $storage.layout?.themeColor,
    themeMode: $storage.layout?.themeMode
  };
  useAppStoreHook().setLayout(layout);
}

watch($storage, ({ layout }) => {
  switch (layout["layout"]) {
    case "vertical":
      toggleClass(true, "is-select", unref(verticalRef));
      debounce(setFalse([horizontalRef]), 50);
      debounce(setFalse([mixRef]), 50);
      break;
    case "horizontal":
      toggleClass(true, "is-select", unref(horizontalRef));
      debounce(setFalse([verticalRef]), 50);
      debounce(setFalse([mixRef]), 50);
      break;
    case "mix":
      toggleClass(true, "is-select", unref(mixRef));
      debounce(setFalse([verticalRef]), 50);
      debounce(setFalse([horizontalRef]), 50);
      break;
  }
});

const mediaQueryList = window.matchMedia("(prefers-color-scheme: dark)");

/** 根据操作系统主题设置平台主题模式 */
function updateTheme() {
  if (themeMode.value !== "system") return;
  if (mediaQueryList.matches) {
    dataTheme.value = true;
  } else {
    dataTheme.value = false;
  }
  dataThemeChange(themeMode.value);
}

function removeMatchMedia() {
  mediaQueryList.removeEventListener("change", updateTheme);
}

/** 监听操作系统主题改变 */
function watchSystemThemeChange() {
  updateTheme();
  removeMatchMedia();
  mediaQueryList.addEventListener("change", updateTheme);
}

onBeforeMount(() => {
  /* 初始化系统配置 */
  nextTick(() => {
    watchSystemThemeChange();
    settings.greyVal && document.querySelector("html")?.classList.add("html-grey");
    settings.weakVal && document.querySelector("html")?.classList.add("html-weakness");
    settings.tabsVal && tagsChange();
    settings.hideFooter && hideFooterChange();
  });
});

onUnmounted(() => removeMatchMedia);
</script>

<template>
  <LayPanel>
    <div class="p-5">
      <p :class="pClass">{{ t("skin.groupIndustrial") }}</p>
      <div class="skin-grid">
        <button v-for="s in industrialSkins" :key="s.id" type="button" class="skin-cell"
          :class="{ 'is-active': currentSkin === s.id }" :style="{ '--skin-accent': s.color }" @click="setSkin(s.id)">
          <span class="skin-swatch" />
          <span class="skin-name">{{ t(s.labelKey) }}</span>
        </button>
        <button type="button" class="skin-cell" :class="{ 'is-active': currentSkin === 'classic' && !dataTheme }"
          :style="{ '--skin-accent': '#ffffff' }" @click="pickClassicMode('light')">
          <span class="skin-swatch skin-swatch--light" />
          <span class="skin-name">{{ t("panel.pureThemeModeLight") }}</span>
        </button>
        <button type="button" class="skin-cell" :class="{ 'is-active': currentSkin === 'classic' && dataTheme }"
          :style="{ '--skin-accent': '#1b2a47' }" @click="pickClassicMode('dark')">
          <span class="skin-swatch skin-swatch--dark" />
          <span class="skin-name">{{ t("panel.pureThemeModeDark") }}</span>
        </button>
      </div>

      <p :class="['mt-5!', pClass]">{{ t("panel.pureThemeColor") }}</p>
      <ul class="theme-color">
        <li v-for="(item, index) in classicColors" v-show="showThemeColors(item.themeColor)" :key="`cls-${index}`"
          :style="getThemeColorStyle(item.color)" @click="setLayoutThemeColor(item.themeColor)">
          <el-icon class="mt-px" :size="20" :color="getThemeColor(item.themeColor)">
            <IconifyIconOffline :icon="Check" />
          </el-icon>
        </li>
      </ul>

      <p :class="['mt-5!', pClass]">{{ t("panel.pureMenuLayout") }}</p>
      <ul class="pure-theme">
        <li ref="verticalRef" v-tippy="{
          content: t('panel.pureVerticalTip'),
          zIndex: 41000
        }" :class="layoutTheme.layout === 'vertical' ? 'is-select' : ''" @click="setMenuLayout('vertical')">
          <div />
          <div />
        </li>
        <li v-if="device !== 'mobile'" ref="horizontalRef" v-tippy="{
          content: t('panel.pureHorizontalTip'),
          zIndex: 41000
        }" :class="layoutTheme.layout === 'horizontal' ? 'is-select' : ''" @click="setMenuLayout('horizontal')">
          <div />
          <div />
        </li>
        <li v-if="device !== 'mobile'" ref="mixRef" v-tippy="{
          content: t('panel.pureMixTip'),
          zIndex: 41000
        }" :class="layoutTheme.layout === 'mix' ? 'is-select' : ''" @click="setMenuLayout('mix')">
          <div />
          <div />
        </li>
      </ul>

      <span v-if="useAppStoreHook().getViewportWidth > 1280">
        <p :class="['mt-5!', pClass]">{{ t("panel.pureStretch") }}</p>
        <Segmented resize class="mb-2 select-none" :modelValue="isNumber(settings.stretch) ? 1 : 0"
          :options="stretchTypeOptions" @change="stretchTypeChange" />
        <el-input-number v-if="isNumber(settings.stretch)" v-model="settings.stretch as number" :min="1280" :max="1600"
          controls-position="right" @change="value => setStretch(value)" />
        <button v-else v-ripple="{ class: 'text-gray-300' }"
          class="bg-transparent flex-c w-full h-20 rounded-md border border-(--pure-border-color)"
          @click="setStretch(!settings.stretch)">
          <div class="flex-bc transition-all duration-300" :class="[settings.stretch ? 'w-[24%]' : 'w-[50%]']"
            style="color: var(--el-color-primary)">
            <IconifyIconOffline :icon="settings.stretch ? RightArrow : LeftArrow" />
            <div class="grow border-0 border-b border-dashed" style="border-color: var(--el-color-primary)" />
            <IconifyIconOffline :icon="settings.stretch ? LeftArrow : RightArrow" />
          </div>
        </button>
      </span>

      <p :class="['mt-5!', pClass]">{{ t("panel.pureTagsStyle") }}</p>
      <Segmented resize class="select-none"
        :modelValue="tagsStyleValue === 'smart' ? 0 : tagsStyleValue === 'card' ? 1 : 2" :options="markOptions"
        @change="onChange" />

      <p class="mt-5! font-medium text-sm dark:text-white">
        {{ t("panel.pureFullScreenWatermark") }}
      </p>
      <ul class="setting">
        <li>
          <span class="dark:text-white">
            {{ t("panel.pureEnableWatermark") }}
          </span>
          <el-switch v-model="watermarkConfigs.enable" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="onWatermarkSwitchChange" />
        </li>
        <li v-if="watermarkConfigs.enable" v-motion-fade>
          <span class="dark:text-white">
            {{ t("panel.pureWatermarkText") }}
          </span>
          <el-input v-model="watermarkConfigs.text" class="w-32!" clearable
            :placeholder="t('panel.pureWatermarkTextPlaceholder')" @input="onWatermarkInputChange" />
        </li>
      </ul>

      <p class="mt-3! font-medium text-sm dark:text-white">
        {{ t("panel.pureInterfaceDisplay") }}
      </p>
      <ul class="setting">
        <li>
          <span class="dark:text-white">{{ t("panel.pureHiddenTags") }}</span>
          <el-switch v-model="settings.tabsVal" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="tagsChange" />
        </li>
        <li>
          <span class="dark:text-white">{{ t("panel.pureHiddenFooter") }}</span>
          <el-switch v-model="settings.hideFooter" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="hideFooterChange" />
        </li>
        <li>
          <span class="dark:text-white">
            {{ t("panel.pureMultiTagsCache") }}
          </span>
          <el-switch v-model="settings.multiTagsCache" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="multiTagsCacheChange" />
        </li>
        <li>
          <span class="dark:text-white">Logo</span>
          <el-switch v-model="logoVal" inline-prompt :active-value="true" :inactive-value="false"
            :active-text="t('buttons.pureOpenText')" :inactive-text="t('buttons.pureCloseText')" @change="logoChange" />
        </li>
        <li>
          <span class="dark:text-white">{{ t("panel.pureGreyModel") }}</span>
          <el-switch v-model="settings.greyVal" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="greyChange" />
        </li>
        <li>
          <span class="dark:text-white">{{ t("panel.pureWeakModel") }}</span>
          <el-switch v-model="settings.weakVal" inline-prompt :active-text="t('buttons.pureOpenText')"
            :inactive-text="t('buttons.pureCloseText')" @change="weekChange" />
        </li>
      </ul>
    </div>
  </LayPanel>
</template>

<style lang="scss" scoped>
:deep(.el-divider__text) {
  font-size: 16px;
  font-weight: 700;
}

:deep(.el-switch__core) {
  --el-switch-off-color: var(--pure-switch-off-color);

  min-width: 36px;
  height: 18px;
}

:deep(.el-switch__core .el-switch__action) {
  height: 14px;
}

.theme-color-group {
  margin-top: 6px;

  .theme-color-group-title {
    display: inline-block;
    margin-bottom: 2px;
    font-family: ui-monospace, Consolas, monospace;
    font-size: 10px;
    color: var(--ind-fg-dim, #909399);
    text-transform: uppercase;
    letter-spacing: 1.5px;
    opacity: 0.85;
  }
}

.theme-color {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-top: 4px;

  li {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 26px;
    cursor: pointer;
    border-radius: 4px;
    box-shadow: rgb(0 0 0 / 15%) 0 0 0 1px inset;
    transition: all 0.2s ease;

    &:hover {
      box-shadow:
        rgb(0 0 0 / 25%) 0 0 0 1px inset,
        0 2px 4px rgb(0 0 0 / 15%);
      transform: scale(1.05);
    }
  }
}

.skin-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-top: 4px;
}

.skin-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
  padding: 6px;
  cursor: pointer;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  transition: all 0.2s ease;

  .skin-swatch {
    display: block;
    width: 100%;
    height: 22px;
    background: var(--skin-accent);
    border-radius: 3px;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 12%);
  }

  .skin-swatch--light {
    background: #fff;
    box-shadow: inset 0 0 0 1px rgb(0 0 0 / 18%);
  }

  .skin-swatch--dark {
    background: #1b2a47;
  }

  .skin-name {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 12px;
    line-height: 1.2;
    color: var(--ind-fg, var(--el-text-color-primary));
    white-space: nowrap;
  }

  &:hover {
    border-color: var(--skin-accent);
    transform: translateY(-1px);

    .skin-name {
      color: var(--skin-accent);
    }
  }

  &.is-active {
    border-color: var(--skin-accent);
    box-shadow:
      0 0 0 2px var(--skin-accent),
      0 0 12px var(--skin-accent);

    .skin-name {
      font-weight: 700;
      color: var(--skin-accent);
      text-shadow:
        0 0 6px var(--skin-accent),
        0 0 1px currentcolor;
    }
  }
}

.pure-theme {
  display: flex;
  gap: 12px;

  li {
    position: relative;
    width: 46px;
    height: 36px;
    overflow: hidden;
    cursor: pointer;
    background: #f0f2f5;
    border-radius: 4px;
    box-shadow: 0 1px 2.5px 0 rgb(0 0 0 / 18%);

    &:nth-child(1) {
      div {
        &:nth-child(1) {
          width: 30%;
          height: 100%;
          background: #1b2a47;
        }

        &:nth-child(2) {
          position: absolute;
          top: 0;
          right: 0;
          width: 70%;
          height: 30%;
          background: #fff;
          box-shadow: 0 0 1px #888;
        }
      }
    }

    &:nth-child(2) {
      div {
        &:nth-child(1) {
          width: 100%;
          height: 30%;
          background: #1b2a47;
          box-shadow: 0 0 1px #888;
        }
      }
    }

    &:nth-child(3) {
      div {
        &:nth-child(1) {
          width: 100%;
          height: 30%;
          background: #1b2a47;
          box-shadow: 0 0 1px #888;
        }

        &:nth-child(2) {
          position: absolute;
          bottom: 0;
          left: 0;
          width: 30%;
          height: 70%;
          background: #fff;
          box-shadow: 0 0 1px #888;
        }
      }
    }
  }
}

.is-select {
  border: 2px solid var(--el-color-primary);
}

.setting {
  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 3px 0;
    font-size: 14px;
  }
}
</style>
