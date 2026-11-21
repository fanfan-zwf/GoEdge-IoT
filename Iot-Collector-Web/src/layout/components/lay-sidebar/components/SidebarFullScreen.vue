<script setup lang="ts">
import { ref, watch } from "vue";
import { useNav } from "@/layout/hooks/useNav";
import { useI18n } from "vue-i18n";

const screenIcon = ref();
const { toggle, isFullscreen, Fullscreen, ExitFullscreen } = useNav();
const { t } = useI18n();

isFullscreen.value = !!(
  document.fullscreenElement ||
  document.webkitFullscreenElement ||
  document.mozFullScreenElement ||
  document.msFullscreenElement
);

watch(
  isFullscreen,
  full => {
    screenIcon.value = full ? ExitFullscreen : Fullscreen;
  },
  {
    immediate: true
  }
);
</script>

<template>
  <span class="fullscreen-icon navbar-bg-hover" @click="toggle">
    <IconifyIconOffline :icon="screenIcon" />
    <span class="fullscreen-label">{{ t("common.fullScreen") }}</span>
  </span>
</template>

<style lang="scss" scoped>
.fullscreen-label {
  margin-left: 4px;
  font-size: 14px;
  white-space: nowrap;
}
</style>
