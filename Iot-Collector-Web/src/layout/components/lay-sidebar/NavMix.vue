<script setup lang="ts">
import { isAllEmpty } from "@pureadmin/utils";
import { useNav } from "@/layout/hooks/useNav";
import { transformI18n } from "@/plugins/i18n";
import { ref, toRaw, watch, onMounted, nextTick } from "vue";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { getParentPaths, findRouteByPath } from "@/router/utils";
import { useTranslationLang, languageOptions } from "../../hooks/useTranslationLang";
import { usePermissionStoreHook } from "@/store/modules/permission";
import LaySidebarExtraIcon from "../lay-sidebar/components/SidebarExtraIcon.vue";
import LaySidebarFullScreen from "../lay-sidebar/components/SidebarFullScreen.vue";

import GlobalizationIcon from "@/assets/svg/globalization.svg?component";
import LogoutCircleRLine from "~icons/ri/logout-circle-r-line";
import Setting from "~icons/ri/settings-3-line";
import UserIcon from "~icons/ri/user-3-line";
import Check from "~icons/ep/check";

const menuRef = ref();
const defaultActive = ref(null);

const { t, route, locale, currentLangLabel, translationLang } = useTranslationLang(menuRef);
const {
  device,
  logout,
  onPanel,
  resolvePath,
  username,
  getDivStyle,
  getDropdownItemStyle,
  getDropdownItemClass
} = useNav();

function getDefaultActive(routePath) {
  const wholeMenus = usePermissionStoreHook().wholeMenus;
  /** 当前路由的父级路径 */
  const parentRoutes = getParentPaths(routePath, wholeMenus)[0];
  defaultActive.value = !isAllEmpty(route.meta?.activePath)
    ? route.meta.activePath
    : findRouteByPath(parentRoutes, wholeMenus)?.children[0]?.path;
}

onMounted(() => {
  getDefaultActive(route.path);
});

nextTick(() => {
  menuRef.value?.handleResize();
});

watch(
  () => [route.path, usePermissionStoreHook().wholeMenus],
  () => {
    getDefaultActive(route.path);
  }
);
</script>

<template>
  <div
    v-if="device !== 'mobile'"
    v-loading="usePermissionStoreHook().wholeMenus.length === 0"
    class="horizontal-header"
  >
    <el-menu
      ref="menuRef"
      router
      mode="horizontal"
      popper-class="pure-scrollbar"
      class="horizontal-header-menu"
      :default-active="defaultActive"
    >
      <el-menu-item
        v-for="route in usePermissionStoreHook().wholeMenus"
        :key="route.path"
        :index="resolvePath(route) || route.redirect"
      >
        <template #title>
          <div v-if="toRaw(route.meta.icon)" :class="['sub-menu-icon', route.meta.icon]">
            <component :is="useRenderIcon(route.meta && toRaw(route.meta.icon))" />
          </div>
          <div :style="getDivStyle">
            <span class="select-none">
              {{ transformI18n(route.meta.title) }}
            </span>
            <LaySidebarExtraIcon :extraIcon="route.meta.extraIcon" />
          </div>
        </template>
      </el-menu-item>
    </el-menu>
    <div class="horizontal-header-right">
      <!-- 国际化 -->
      <el-dropdown id="header-translation" trigger="click">
        <div class="globalization-icon navbar-bg-hover">
          <IconifyIconOffline :icon="GlobalizationIcon" />
          <span>{{ currentLangLabel }}</span>
        </div>
        <template #dropdown>
          <el-dropdown-menu class="translation">
            <el-dropdown-item
              v-for="lang in languageOptions"
              :key="lang.locale"
              :style="getDropdownItemStyle(locale, lang.locale)"
              :class="['dark:text-white!', getDropdownItemClass(locale, lang.locale)]"
              @click="translationLang(lang.locale)"
            >
              <IconifyIconOffline v-show="locale === lang.locale" :icon="Check" style="position: absolute; left: 20px" />
              {{ lang.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <!-- 全屏 -->
      <LaySidebarFullScreen id="full-screen" />
      <!-- 退出登录 -->
      <el-dropdown trigger="click">
        <span class="el-dropdown-link navbar-bg-hover select-none">
          <IconifyIconOffline :icon="UserIcon" style="margin-right: 4px" />
          <p v-if="username" class="dark:text-white">{{ username }}</p>
        </span>
        <template #dropdown>
          <el-dropdown-menu class="logout">
            <el-dropdown-item @click="logout">
              <IconifyIconOffline :icon="LogoutCircleRLine" style="margin: 5px" />
              {{ t("buttons.pureLoginOut") }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <span class="set-icon navbar-bg-hover" :title="t('buttons.pureOpenSystemSet')" @click="onPanel">
        <IconifyIconOffline :icon="Setting" />
        <span class="set-label">{{ t("common.systemSet") }}</span>
      </span>
    </div>
  </div>
</template>

<style lang="scss" scoped>
:deep(.el-loading-mask) {
  opacity: 0.45;
}

.translation {
  :deep(.el-dropdown-menu__item) {
    padding: 5px 16px 5px 40px;
  }
}

.logout {
  width: 120px;

  :deep(.el-dropdown-menu__item) {
    display: inline-flex;
    flex-wrap: wrap;
    min-width: 100%;
  }
}
</style>
