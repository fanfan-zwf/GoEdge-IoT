<template>
  <div class="rbac-page">
    <el-card class="box-card" shadow="never">
      <div class="page-toolbar">
        <div class="toolbar-left">
          <el-icon class="toolbar-icon"><component :is="LockIcon" /></el-icon>
          <span class="toolbar-title">{{ t("rbac.title") }}</span>
        </div>
      </div>

      <el-alert type="info" :closable="false" class="page-tip" show-icon>
        <div class="page-tip-text">{{ t("rbac.pageTip") }}</div>
      </el-alert>

      <el-tabs v-model="activeTab" class="rbac-tabs">
        <!-- 三个子页各自独立成文件；lazy 保证默认只挂载/加载「角色管理」，切换时才加载其余 -->
        <el-tab-pane :label="t('rbac.tabRole')" name="role" lazy>
          <RoleManagement />
        </el-tab-pane>
        <el-tab-pane :label="t('rbac.tabPerm')" name="perm" lazy>
          <PermDictionary />
        </el-tab-pane>
        <el-tab-pane :label="t('rbac.tabGrant')" name="grant" lazy>
          <GrantBinding />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { Lock as LockIcon } from "@element-plus/icons-vue";
import RoleManagement from "./role.vue";
import PermDictionary from "./perm.vue";
import GrantBinding from "./grant.vue";

const { t } = useI18n();
defineOptions({ name: "SystemRbac" });

// 默认打开角色管理
const activeTab = ref("role");
</script>

<style lang="scss" scoped>
.rbac-page {
  padding: 12px;
}

.page-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;

  .toolbar-left {
    display: flex;
    align-items: center;

    .toolbar-icon {
      margin-right: 8px;
      font-size: 20px;
      color: var(--el-color-primary);
    }

    .toolbar-title {
      font-size: 16px;
      font-weight: 600;
    }
  }
}

.page-tip {
  margin-bottom: 8px;

  .page-tip-text {
    white-space: pre-line;
    line-height: 1.6;
  }
}
</style>
