<template>
  <div class="tab-toolbar">
    <el-radio-group v-model="subjectType" @change="onSubjectTypeChange">
      <el-radio-button :value="1">{{ t("rbac.bizUser") }}</el-radio-button>
      <el-radio-button :value="2">{{ t("rbac.apiClient") }}</el-radio-button>
    </el-radio-group>
    <el-select v-model="subjectId" filterable class="subject-select" :placeholder="t('rbac.selectSubject')" @change="loadBindings">
      <el-option v-for="s in subjects" :key="s.Id" :label="`${s.Access_Key}${s.Name ? ' · ' + s.Name : ''}`" :value="s.Id" />
    </el-select>
  </div>

  <el-empty v-if="!subjectId" :description="t('rbac.noSubject')" />
  <div v-else class="grant-body">
    <div class="grant-block">
      <div class="grant-head">
        <span>{{ t("rbac.boundRoles") }}</span>
        <el-button v-perms="'config:rbac:subject:roles:set'" type="primary" size="small" :loading="savingRoles" @click="saveRoles">{{
          t("rbac.save")
        }}</el-button>
      </div>
      <el-transfer v-model="boundRoleIds" :data="roleTransferData" :titles="[t('rbac.available'), t('rbac.selected')]" filterable />
    </div>
    <el-divider direction="vertical" class="grant-divider" />
    <div class="grant-block">
      <div class="grant-head">
        <span>{{ t("rbac.directPerms") }}</span>
        <el-button v-perms="'config:rbac:subject:perms:set'" type="primary" size="small" :loading="savingPerms" @click="saveDirectPerms">{{
          t("rbac.save")
        }}</el-button>
      </div>
      <el-transfer v-model="boundPermIds" :data="permTransferData" :titles="[t('rbac.available'), t('rbac.selected')]" filterable />
    </div>
  </div>
</template>

<script setup lang="ts">
import { rbacApi, userApi, apiKeyApi } from "@/api/config";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
defineOptions({ name: "RbacGrantBinding" });

interface TransferItem {
  key: number;
  label: string;
}

// ---------------- 授权绑定 ----------------
const subjectType = ref(1);
const subjects = ref<any[]>([]);
const subjectId = ref<number | undefined>(undefined);
const grantRoles = ref<any[]>([]);
const perms = ref<any[]>([]);
const boundRoleIds = ref<number[]>([]);
const boundPermIds = ref<number[]>([]);
const savingRoles = ref(false);
const savingPerms = ref(false);

const roleTransferData = computed<TransferItem[]>(() =>
  grantRoles.value.map(r => ({ key: r.id, label: `${r.role_name || r.role_code} (${r.role_code})` }))
);
const permTransferData = computed<TransferItem[]>(() =>
  perms.value.map(p => ({ key: p.id, label: `${p.perm_code} · ${p.topic_key} ${p.effect == 1 ? "" : "(deny)"}` }))
);

async function loadSubjects() {
  const res: any = subjectType.value === 1 ? await userApi.query({ page: 0, pageSize: 0 }) : await apiKeyApi.query({ page: 0, pageSize: 0 });
  subjects.value = res?.Data || [];
}
async function loadGrantRoles() {
  const res: any = await rbacApi.role.query({ role_type: subjectType.value });
  grantRoles.value = res?.Data || [];
}
async function loadPerms() {
  const res: any = await rbacApi.perm.query();
  perms.value = res?.Data || [];
}
async function onSubjectTypeChange() {
  subjectId.value = undefined;
  boundRoleIds.value = [];
  boundPermIds.value = [];
  await Promise.all([loadSubjects(), loadGrantRoles()]);
}
async function loadBindings() {
  if (!subjectId.value) return;
  try {
    const [r, p] = await Promise.all([
      rbacApi.subjectRoles.get({ subject_type: subjectType.value, subject_id: subjectId.value }),
      rbacApi.subjectPerms.get({ subject_type: subjectType.value, subject_id: subjectId.value })
    ]);
    boundRoleIds.value = (r as any)?.Data?.ids || [];
    boundPermIds.value = (p as any)?.Data?.ids || [];
  } catch {
    boundRoleIds.value = [];
    boundPermIds.value = [];
  }
}
async function saveRoles() {
  if (!subjectId.value) return;
  savingRoles.value = true;
  try {
    await rbacApi.subjectRoles.set({ subject_type: subjectType.value, subject_id: subjectId.value, ids: boundRoleIds.value });
    ElMessage.success(t("rbac.saveSuccess"));
  } catch {
    /* 拦截器已提示 */
  } finally {
    savingRoles.value = false;
  }
}
async function saveDirectPerms() {
  if (!subjectId.value) return;
  savingPerms.value = true;
  try {
    await rbacApi.subjectPerms.set({ subject_type: subjectType.value, subject_id: subjectId.value, ids: boundPermIds.value });
    ElMessage.success(t("rbac.saveSuccess"));
  } catch {
    /* 拦截器已提示 */
  } finally {
    savingPerms.value = false;
  }
}

onMounted(async () => {
  await loadPerms();
  await onSubjectTypeChange();
});
</script>

<style lang="scss" scoped>
.tab-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;

  .subject-select {
    width: 260px;
  }
}

.grant-body {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-around;

  .grant-block {
    flex: 1;
  }

  .grant-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
    font-weight: 600;
  }

  .grant-divider {
    height: auto;
  }
}

:deep(.el-transfer-panel) {
  width: 280px;
}
</style>
