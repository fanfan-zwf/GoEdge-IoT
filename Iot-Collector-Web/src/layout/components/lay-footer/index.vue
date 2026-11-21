<script setup lang="ts">
const REPO_URL = "https://github.com/fanfan-zwf/GoEdge-IoT";
</script>

<template>
  <footer class="layout-footer text-[rgba(0,0,0,0.6)] dark:text-[rgba(220,220,242,0.8)]">
    Copyright © 2026
    <a class="hover:text-primary!" :href="REPO_URL" target="_blank">&nbsp;GoEdge-IoT</a>
  </footer>
</template>

<style lang="scss" scoped>
.layout-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 0 0 8px;
  font-size: 14px;
}
</style>
