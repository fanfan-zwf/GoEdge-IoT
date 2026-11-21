<script setup lang="ts">
import { useNav } from "@/layout/hooks/useNav";
import LayNavMix from "../lay-sidebar/NavMix.vue";
import { useTranslationLang, languageOptions } from "@/layout/hooks/useTranslationLang";
import LaySidebarFullScreen from "../lay-sidebar/components/SidebarFullScreen.vue";
import LaySidebarBreadCrumb from "../lay-sidebar/components/SidebarBreadCrumb.vue";
import LaySidebarTopCollapse from "../lay-sidebar/components/SidebarTopCollapse.vue";

import GlobalizationIcon from "@/assets/svg/globalization.svg?component";
import LogoutCircleRLine from "~icons/ri/logout-circle-r-line";
import Setting from "~icons/ri/settings-3-line";
import UserIcon from "~icons/ri/user-3-line";
import Check from "~icons/ep/check";

const {
  layout,
  device,
  logout,
  onPanel,
  pureApp,
  username,
  toggleSideBar,
  getDropdownItemStyle,
  getDropdownItemClass
} = useNav();

const { t, locale, currentLangLabel, translationLang } = useTranslationLang();
</script>

<template>
  <div class="navbar bg-white shadow-xs shadow-[rgba(0,21,41,0.08)]">
    <LaySidebarTopCollapse
      v-if="device === 'mobile'"
      class="hamburger-container"
      :is-active="pureApp.sidebar.opened"
      @toggleClick="toggleSideBar"
    />

    <LaySidebarBreadCrumb v-if="layout !== 'mix' && device !== 'mobile'" class="breadcrumb-container" />

    <LayNavMix v-if="layout === 'mix'" />

    <div v-if="layout === 'vertical'" class="vertical-header-right">
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
.navbar {
  width: 100%;
  height: 48px;
  overflow: hidden;

  .hamburger-container {
    float: left;
    height: 100%;
    line-height: 48px;
    cursor: pointer;
  }

  .vertical-header-right {
    display: flex;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
    min-width: 280px;
    height: 48px;
    padding-right: 15px;
    color: #000000d9;

    .el-dropdown-link {
      display: flex;
      align-items: center;
      justify-content: space-around;
      height: 48px;
      padding: 10px;
      color: #000000d9;
      cursor: pointer;

      p {
        font-size: 14px;
      }

      img {
        width: 22px;
        height: 22px;
        border-radius: 50%;
      }
    }
  }

  .breadcrumb-container {
    float: left;
    margin-left: 16px;
  }
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
