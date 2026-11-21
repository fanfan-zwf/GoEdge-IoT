<script setup lang="ts">
import { emitter } from "@/utils/mitt";
import { useNav } from "@/layout/hooks/useNav";
import { responsiveStorageNameSpace } from "@/config";
import { ref, nextTick, computed, onMounted } from "vue";
import { storageLocal, isAllEmpty } from "@pureadmin/utils";
import { useTranslationLang, languageOptions } from "../../hooks/useTranslationLang";
import { usePermissionStoreHook } from "@/store/modules/permission";
import LaySidebarItem from "../lay-sidebar/components/SidebarItem.vue";
import LaySidebarFullScreen from "../lay-sidebar/components/SidebarFullScreen.vue";

import GlobalizationIcon from "@/assets/svg/globalization.svg?component";
import LogoutCircleRLine from "~icons/ri/logout-circle-r-line";
import Setting from "~icons/ri/settings-3-line";
import UserIcon from "~icons/ri/user-3-line";
import Check from "~icons/ep/check";

const menuRef = ref();
const showLogo = ref(
  storageLocal().getItem<StorageConfigs>(`${responsiveStorageNameSpace()}configure`)?.showLogo ?? true
);

const { t, route, locale, currentLangLabel, translationLang } = useTranslationLang(menuRef);
const {
  title,
  logout,
  onPanel,
  getLogo,
  username,
  backTopMenu,
  getDropdownItemStyle,
  getDropdownItemClass
} = useNav();

const defaultActive = computed(() => (!isAllEmpty(route.meta?.activePath) ? route.meta.activePath : route.path));

nextTick(() => {
  menuRef.value?.handleResize();
});

onMounted(() => {
  emitter.on("logoChange", key => {
    showLogo.value = key;
  });
});
</script>

<template>
  <div v-loading="usePermissionStoreHook().wholeMenus.length === 0" class="horizontal-header">
    <div v-if="showLogo" class="horizontal-header-left" @click="backTopMenu">
      <img :src="getLogo()" alt="logo" />
      <span>{{ title }}</span>
    </div>
    <el-menu
      ref="menuRef"
      mode="horizontal"
      popper-class="pure-scrollbar"
      class="horizontal-header-menu"
      :default-active="defaultActive"
    >
      <LaySidebarItem
        v-for="route in usePermissionStoreHook().wholeMenus"
        :key="route.path"
        :item="route"
        :base-path="route.path"
      />
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
        <span class="el-dropdown-link navbar-bg-hover">
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
