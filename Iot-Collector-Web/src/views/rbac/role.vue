<template>
  <div class="tab-toolbar">
    <el-radio-group v-model="roleType" @change="onRoleTypeChange">
      <el-radio-button :value="1">{{ t("rbac.bizRole") }}</el-radio-button>
      <el-radio-button :value="2">{{ t("rbac.apiRole") }}</el-radio-button>
    </el-radio-group>
    <el-input v-model="keyword" :placeholder="t('rbac.searchRole')" clearable class="search-input" @keyup.enter="onSearch" @clear="onSearch" />
    <el-button :icon="SearchIcon" @click="onSearch">{{ t("common.query") }}</el-button>
    <el-button :icon="RefreshLeftIcon" @click="onReset">{{ t("common.reset") }}</el-button>
    <div class="spacer" />
    <el-button :icon="Refresh" circle @click="loadRoles" />
    <el-button v-perms="'config:rbac:role:add'" type="primary" :icon="Plus" @click="openRoleAdd">{{ t("rbac.addRole") }}</el-button>
  </div>

  <el-table
    v-loading="roleLoading"
    :data="roles"
    border
    stripe
    highlight-current-row
    max-height="calc(100vh - 460px)"
    :default-sort="{ prop: 'id', order: 'ascending' }"
    style="width: 100%"
  >
    <el-table-column prop="id" :label="t('account.id')" width="72" align="center" sortable />
    <el-table-column prop="role_code" :label="t('rbac.roleCode')" min-width="160" show-overflow-tooltip sortable />
    <el-table-column prop="role_name" :label="t('rbac.roleName')" min-width="140" show-overflow-tooltip sortable />
    <el-table-column prop="sort" :label="t('rbac.sort')" width="80" align="center" sortable />
    <el-table-column :label="t('account.enable')" width="100" align="center">
      <template #default="s">
        <el-tag :type="s.row.status == 1 ? 'success' : 'info'" size="small" effect="light">
          {{ s.row.status == 1 ? t("account.enableOn") : t("account.enableOff") }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="remark" :label="t('rbac.remark')" min-width="160" show-overflow-tooltip sortable />
    <el-table-column :label="t('common.operation')" width="280" align="center" fixed="right">
      <template #default="s">
        <div class="op-btns">
          <el-button v-perms="'config:rbac:role:perms:set'" size="small" type="success" plain @click="openRolePerms(s.row)">{{
            t("rbac.configPerms")
          }}</el-button>
          <el-button v-perms="'config:rbac:role:update'" size="small" type="primary" plain :icon="EditIcon" @click="openRoleEdit(s.row)">{{
            t("account.edit")
          }}</el-button>
          <el-button v-perms="'config:rbac:role:del'" size="small" type="danger" plain :icon="DeleteIcon" @click="delRole(s.row)">{{
            t("account.del")
          }}</el-button>
        </div>
      </template>
    </el-table-column>
    <template #empty><el-empty :description="t('account.noData')" /></template>
  </el-table>

  <el-pagination
    v-model:current-page="pg.page"
    v-model:page-size="pg.pageSize"
    class="page-pagination"
    :total="total"
    :page-sizes="[10, 20, 50, 100]"
    layout="total, sizes, prev, pager, next, jumper"
    @size-change="onSizeChange"
    @current-change="loadRoles"
  />

  <!-- 角色新增/编辑 -->
  <el-dialog v-model="roleDialog" :title="roleForm.id ? t('rbac.editRole') : t('rbac.addRole')" width="520px">
    <el-form ref="roleFormRef" :model="roleForm" :rules="roleRules" label-width="110px" label-position="left">
      <el-form-item :label="t('rbac.roleCode')" prop="role_code">
        <el-input v-model="roleForm.role_code" :maxlength="64" />
      </el-form-item>
      <el-form-item :label="t('rbac.roleName')">
        <el-input v-model="roleForm.role_name" :maxlength="128" />
      </el-form-item>
      <el-form-item :label="t('rbac.sort')">
        <el-input-number v-model="roleForm.sort" :min="0" :max="9999" controls-position="right" />
      </el-form-item>
      <el-form-item :label="t('account.enable')">
        <el-switch v-model="roleForm.status" :active-value="1" :inactive-value="0" inline-prompt :active-text="t('account.enableOn')" :inactive-text="t('account.enableOff')" />
      </el-form-item>
      <el-form-item :label="t('rbac.remark')">
        <el-input v-model="roleForm.remark" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" :maxlength="255" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="roleDialog = false">{{ t("account.cancel") }}</el-button>
      <el-button type="primary" :loading="submitting" @click="submitRole">{{ t("account.confirm") }}</el-button>
    </template>
  </el-dialog>

  <!-- 角色配置权限 -->
  <el-dialog v-model="rolePermDialog" :title="t('rbac.configPerms')" width="720px">
    <div class="perm-transfer">
      <el-transfer v-model="rolePermIds" :data="permTransferData" :titles="[t('rbac.available'), t('rbac.selected')]" filterable />
    </div>
    <template #footer>
      <el-button @click="rolePermDialog = false">{{ t("account.cancel") }}</el-button>
      <el-button type="primary" :loading="submitting" @click="submitRolePerms">{{ t("account.confirm") }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { rbacApi } from "@/api/config";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Refresh, Plus, Edit as EditIcon, Delete as DeleteIcon, Search as SearchIcon, RefreshLeft as RefreshLeftIcon } from "@element-plus/icons-vue";

const { t } = useI18n();
defineOptions({ name: "RbacRoleManagement" });

interface TransferItem {
  key: number;
  label: string;
}

const submitting = ref(false);

// ---------------- 角色 ----------------
const roleType = ref(1);
const roleLoading = ref(false);
const roles = ref<any[]>([]);
// 服务端分页 + 关键字查询
const keyword = ref("");
const total = ref(0);
const pg = reactive({ page: 1, pageSize: 10 });
async function loadRoles() {
  roleLoading.value = true;
  try {
    const [cnt, list] = await Promise.all([
      rbacApi.role.count({ role_type: roleType.value, keyword: keyword.value }),
      rbacApi.role.query({ role_type: roleType.value, keyword: keyword.value, page: pg.page, pageSize: pg.pageSize })
    ]);
    total.value = (cnt as any)?.Data || 0;
    roles.value = (list as any)?.Data || [];
  } finally {
    roleLoading.value = false;
  }
}
// 切换角色类型/搜索/重置均回到首页；重置同时清空关键字
function onRoleTypeChange() {
  pg.page = 1;
  loadRoles();
}
function onSearch() {
  pg.page = 1;
  loadRoles();
}
function onReset() {
  keyword.value = "";
  pg.page = 1;
  loadRoles();
}
function onSizeChange() {
  pg.page = 1;
  loadRoles();
}

const roleDialog = ref(false);
const roleFormRef = ref<FormInstance>();
const roleForm = reactive({ id: 0, role_code: "", role_name: "", sort: 0, status: 1, remark: "" });
const roleRules = reactive<FormRules>({
  role_code: [{ required: true, message: t("rbac.roleCode"), trigger: "blur" }]
});
function openRoleAdd() {
  Object.assign(roleForm, { id: 0, role_code: "", role_name: "", sort: 0, status: 1, remark: "" });
  roleDialog.value = true;
}
function openRoleEdit(row: any) {
  Object.assign(roleForm, {
    id: row.id,
    role_code: row.role_code,
    role_name: row.role_name || "",
    sort: row.sort || 0,
    status: row.status,
    remark: row.remark || ""
  });
  roleDialog.value = true;
}
async function submitRole() {
  if (!roleFormRef.value) return;
  await roleFormRef.value.validate(async valid => {
    if (!valid) return;
    submitting.value = true;
    try {
      const { id, ...rest } = roleForm;
      const payload = { role_type: roleType.value, ...rest };
      if (id) {
        await rbacApi.role.update({ id, ...payload });
      } else {
        await rbacApi.role.add(payload);
      }
      ElMessage.success(t("rbac.saveSuccess"));
      roleDialog.value = false;
      await loadRoles();
    } catch {
      /* 拦截器已提示 */
    } finally {
      submitting.value = false;
    }
  });
}
async function delRole(row: any) {
  try {
    await ElMessageBox.confirm(t("rbac.delRoleConfirm", { code: row.role_code }), t("rbac.title"), { type: "warning" });
  } catch {
    return;
  }
  try {
    await rbacApi.role.del({ role_type: roleType.value, id: row.id });
    ElMessage.success(t("account.delSuccess"));
    await loadRoles();
  } catch {
    /* 拦截器已提示 */
  }
}

// 权限字典（配置权限对话框用，首次打开时懒加载一次）
const perms = ref<any[]>([]);
const permsLoaded = ref(false);
const permTransferData = computed<TransferItem[]>(() =>
  perms.value.map(p => ({ key: p.id, label: `${p.perm_code} · ${p.topic_key} ${p.effect == 1 ? "" : "(deny)"}` }))
);
async function ensurePerms() {
  if (permsLoaded.value) return;
  const res: any = await rbacApi.perm.query();
  perms.value = res?.Data || [];
  permsLoaded.value = true;
}

// 角色 → 权限
const rolePermDialog = ref(false);
const rolePermIds = ref<number[]>([]);
const currentRoleId = ref(0);
async function openRolePerms(row: any) {
  currentRoleId.value = row.id;
  await ensurePerms();
  try {
    const res: any = await rbacApi.rolePerms.get({ role_type: roleType.value, role_id: row.id });
    rolePermIds.value = res?.Data?.ids || [];
  } catch {
    rolePermIds.value = [];
  }
  rolePermDialog.value = true;
}
async function submitRolePerms() {
  submitting.value = true;
  try {
    await rbacApi.rolePerms.set({ role_type: roleType.value, role_id: currentRoleId.value, ids: rolePermIds.value });
    ElMessage.success(t("rbac.saveSuccess"));
    rolePermDialog.value = false;
  } catch {
    /* 拦截器已提示 */
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  loadRoles();
});
</script>

<style lang="scss" scoped>
.tab-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;

  .spacer {
    flex: 1;
  }
}

.search-input {
  width: 200px;
}

.page-pagination {
  justify-content: flex-end;
  margin-top: 14px;
}

.op-btns {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  justify-content: center;

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }
}

.perm-transfer :deep(.el-transfer) {
  display: flex;
  align-items: center;
  justify-content: center;
}

.perm-transfer :deep(.el-transfer-panel) {
  flex-shrink: 0;
  width: 280px;
}

.perm-transfer :deep(.el-transfer__buttons) {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: center;
  padding: 0 16px;
}

.perm-transfer :deep(.el-transfer__button:nth-child(2)) {
  margin: 0;
}
</style>
