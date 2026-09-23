<template>
  <el-config-provider :locale="elementLocale">
    <div class="common-layout">
      <el-container class="layout-container">
        <!-- 左侧侧边栏 -->
        <el-aside :width="asideWidth" class="left-sidebar" :class="{ 'mobile-visible': isMobileVisible }">
          <div class="sidebar-content" :class="{ collapsed: isCollapsed }">
            <!-- Logo 区域 -->
            <div class="sidebar-logo">
              <div class="logo-icon-wrapper">
                <span style="font-size: 24px">⚙</span>
              </div>
              <div class="logo-text-wrapper" v-show="!isCollapsed || isMobile">
                <span class="logo-text">{{ t('menu.title') }}</span>
              </div>
            </div>

            <!-- 菜单区域 -->
            <el-menu
              :default-active="activePath"
              class="sidebar-menu"
              :collapse="isCollapsed"
              router
              @select="handleMenuSelect"
            >
              <el-menu-item index="/drive">
                <el-icon><Monitor /></el-icon>
                <span>{{ t('menu.drive') }}</span>
              </el-menu-item>
              <el-menu-item index="/point">
                <el-icon><Pointer /></el-icon>
                <span>{{ t('menu.point') }}</span>
              </el-menu-item>
              <el-menu-item index="/alarm">
                <el-icon><Bell /></el-icon>
                <span>{{ t('menu.alarm') }}</span>
              </el-menu-item>
              <el-menu-item index="/history">
                <el-icon><Clock /></el-icon>
                <span>{{ t('menu.history') }}</span>
              </el-menu-item>
              <el-menu-item index="/history-data">
                <el-icon><TrendCharts /></el-icon>
                <span>{{ t('menu.historyData') }}</span>
              </el-menu-item>
            </el-menu>

            <!-- 折叠按钮 -->
            <div class="sidebar-footer">
              <el-button
                :icon="isCollapsed ? Expand : Fold"
                @click="toggleCollapse"
                circle
                size="small"
                class="collapse-button"
              />
            </div>
          </div>

        </el-aside>

        <!-- 移动端遮罩层（放在 aside 外面，避免阻挡侧边栏点击） -->
        <div
          v-if="isMobile && isMobileVisible"
          class="mobile-overlay"
          @click="closeSidebar"
        />

        <!-- 右侧主内容 -->
        <el-container class="right-content">
          <!-- 顶部 Header -->
          <el-header class="top-header">
            <div class="header-left">
              <el-icon v-if="isMobile" class="mobile-menu-btn" @click="toggleCollapse">
                <Fold v-if="!isCollapsed" />
                <Expand v-else />
              </el-icon>
              <el-breadcrumb separator="/">
                <el-breadcrumb-item :to="{ path: '/' }">{{ t('common.home') }}</el-breadcrumb-item>
                <el-breadcrumb-item>{{ currentTitle }}</el-breadcrumb-item>
              </el-breadcrumb>
            </div>
            <div class="header-right">
              <!-- 语言切换 -->
              <el-dropdown @command="handleLangChange" trigger="click" style="margin-right: 16px">
                <span class="lang-switch">
                  {{ currentLangLabel }}
                </span>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="zh">中文</el-dropdown-item>
                    <el-dropdown-item command="en">English</el-dropdown-item>
                    <el-dropdown-item command="fr">Français</el-dropdown-item>
                    <el-dropdown-item command="ru">Русский</el-dropdown-item>
                    <el-dropdown-item command="es">Español</el-dropdown-item>
                    <el-dropdown-item command="ar">العربية</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <span>Iot-Collector Service</span>
            </div>
          </el-header>

          <!-- 主内容 -->
          <el-container class="main-content">
            <el-main class="content-main">
              <router-view />
            </el-main>
          </el-container>
        </el-container>
      </el-container>
    </div>
  </el-config-provider>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Fold, Expand, Monitor, Pointer, Bell, Clock, TrendCharts } from '@element-plus/icons-vue'
import { setLocale } from '@/i18n'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import enUs from 'element-plus/es/locale/lang/en'
import frFr from 'element-plus/es/locale/lang/fr'
import ruRu from 'element-plus/es/locale/lang/ru'
import esEs from 'element-plus/es/locale/lang/es'
import arDz from 'element-plus/es/locale/lang/ar'

const { t, locale } = useI18n()
const route = useRoute()

// Element Plus 语言包动态切换
const elementLocaleMap: Record<string, any> = {
  zh: zhCn,
  en: enUs,
  fr: frFr,
  ru: ruRu,
  es: esEs,
  ar: arDz
}
const elementLocale = computed(() => {
  return elementLocaleMap[locale.value] || enUs
})

// 当前语言显示标签
const langLabels: Record<string, string> = {
  zh: '中文',
  en: 'EN',
  fr: 'FR',
  ru: 'RU',
  es: 'ES',
  ar: 'AR'
}
const currentLangLabel = computed(() => {
  return langLabels[locale.value] || locale.value
})

// 语言切换
const handleLangChange = (lang: 'zh' | 'en' | 'fr' | 'ru' | 'es' | 'ar') => {
  setLocale(lang)
  locale.value = lang
}

// 侧边栏折叠状态
const isCollapsed = ref(false)
const isMobile = ref(false)
const isMobileVisible = ref(false)

// 路由匹配当前路径
const activePath = computed(() => route.path)

// 面包屑当前标题 - 使用 i18n
const titleMap: Record<string, string> = {
  '/drive': t('menu.drive'),
  '/point': t('menu.point'),
  '/alarm': t('menu.alarm'),
  '/history': t('menu.history'),
  '/history-data': t('menu.historyData')
}
const currentTitle = computed(() => titleMap[route.path] || t('menu.title'))

// 侧边栏宽度
const asideWidth = computed(() => {
  if (isMobile.value) {
    return isMobileVisible.value ? '200px' : '0px'
  }
  return isCollapsed.value ? '64px' : '200px'
})

// 折叠切换
const toggleCollapse = () => {
  if (isMobile.value) {
    const willBeVisible = !isMobileVisible.value
    isMobileVisible.value = willBeVisible
    if (willBeVisible) {
      isCollapsed.value = false
    }
  } else {
    isCollapsed.value = !isCollapsed.value
  }
}

// 关闭侧边栏
const closeSidebar = () => {
  if (isMobile.value) {
    isMobileVisible.value = false
  }
}

// 菜单选择
const handleMenuSelect = () => {
  if (isMobile.value) {
    closeSidebar()
  }
}

// 检测屏幕尺寸
const checkMobile = () => {
  const wasMobile = isMobile.value
  isMobile.value = window.innerWidth < 768
  if (wasMobile && !isMobile.value) {
    isMobileVisible.value = false
  } else if (!wasMobile && isMobile.value) {
    isMobileVisible.value = false
  }
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style>
@import '@/assets/main.css';
</style>

<style scoped>
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  z-index: 1001;
}

.mobile-menu-btn {
  font-size: 20px;
  cursor: pointer;
  color: #606266;
  display: flex;
  align-items: center;
  padding: 4px;
  border-radius: 4px;
}

.mobile-menu-btn:hover {
  background-color: #f5f7fa;
}

.lang-switch {
  cursor: pointer;
  font-size: 14px;
  color: #606266;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.3s;
}

.lang-switch:hover {
  color: #409eff;
  background-color: rgba(64, 158, 255, 0.1);
}

@media (min-width: 768px) {
  .mobile-menu-btn {
    display: none;
  }
}
</style>
