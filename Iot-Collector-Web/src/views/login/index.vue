<script setup lang="ts">
import { computed, defineAsyncComponent } from "vue";

defineOptions({
  name: "Login"
});

// 定稿：使用 Deep Console（VariantA）作为唯一登录页视觉。
// 预览期的 A/B/C 切换浮标与 URL ?v= 参数已移除。
const VariantA = defineAsyncComponent(() => import("./variants/VariantA.vue"));
const chosen = computed(() => VariantA);
</script>

<template>
  <div class="login-dispatch">
    <component :is="chosen" />
  </div>
</template>

<style scoped>
.login-dispatch {
  position: relative;
  width: 100%;
  height: 100%;
}
</style>
