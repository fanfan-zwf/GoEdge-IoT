<template>
  <div class="tab-toolbar">
    <el-input v-model="keyword" :placeholder="t('rbac.searchPerm')" clearable class="search-input" @keyup.enter="onSearch" @clear="onSearch" />
    <el-button :icon="SearchIcon" @click="onSearch">{{ t("common.query") }}</el-button>
    <el-button :icon="RefreshLeftIcon" @click="onReset">{{ t("common.reset") }}</el-button>
    <div class="spacer" />
    <el-button :icon="Refresh" circle @click="loadPerms" />
    <el-button v-perms="'config:rbac:perm:add'" type="primary" :icon="Plus" @click="openPermAdd">{{ t("rbac.addPerm") }}</el-button>
  </div>

  <el-table
    v-loading="permLoading"
    :data="perms"
    border
    stripe
    highlight-current-row
    max-height="calc(100vh - 460px)"
    :default-sort="{ prop: 'id', order: 'ascending' }"
    style="width: 100%"
  >
    <el-table-column prop="id" :label="t('account.id')" width="72" align="center" sortable />
    <el-table-column prop="perm_code" :label="t('rbac.permCode')" min-width="140" show-overflow-tooltip sortable />
    <el-table-column prop="topic_key" :label="t('rbac.topicKey')" min-width="220" show-overflow-tooltip sortable />
    <el-table-column :label="t('rbac.effect')" width="90" align="center">
      <template #default="s">
        <el-tag :type="s.row.effect == 1 ? 'success' : 'danger'" size="small" effect="light">
          {{ s.row.effect == 1 ? t("rbac.allow") : t("rbac.deny") }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="match_sort" :label="t('rbac.matchSort')" width="100" align="center" sortable />
    <el-table-column :label="t('account.enable')" width="90" align="center">
      <template #default="s">
        <el-tag :type="s.row.status == 1 ? 'success' : 'info'" size="small" effect="light">
          {{ s.row.status == 1 ? t("account.enableOn") : t("account.enableOff") }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="remark" :label="t('rbac.remark')" min-width="140" show-overflow-tooltip sortable />
    <el-table-column :label="t('common.operation')" width="180" align="center" fixed="right">
      <template #default="s">
        <div class="op-btns">
          <el-button v-perms="'config:rbac:perm:update'" size="small" type="primary" plain :icon="EditIcon" @click="openPermEdit(s.row)">{{
            t("account.edit")
          }}</el-button>
          <el-button v-perms="'config:rbac:perm:del'" size="small" type="danger" plain :icon="DeleteIcon" @click="delPerm(s.row)">{{
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
    @current-change="loadPerms"
  />

  <!-- 权限新增/编辑 -->
  <el-dialog v-model="permDialog" :title="permForm.id ? t('rbac.editPerm') : t('rbac.addPerm')" width="600px">
    <el-form ref="permFormRef" :model="permForm" :rules="permRules" label-width="120px" label-position="left">
      <el-form-item :label="t('rbac.permCode')" prop="perm_code">
        <el-input v-model="permForm.perm_code" :maxlength="64" />
      </el-form-item>
      <el-form-item :label="t('rbac.topicKey')" prop="topic_key">
        <el-input v-model="permForm.topic_key" :maxlength="255" placeholder="/config/drive/query" />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item :label="t('rbac.effect')">
            <el-radio-group v-model="permForm.effect">
              <el-radio-button :value="1">{{ t("rbac.allow") }}</el-radio-button>
              <el-radio-button :value="0">{{ t("rbac.deny") }}</el-radio-button>
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label="t('rbac.matchSort')">
            <el-input-number v-model="permForm.match_sort" :min="0" :max="9999" controls-position="right" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item :label="t('account.enable')">
        <el-switch v-model="permForm.status" :active-value="1" :inactive-value="0" inline-prompt :active-text="t('account.enableOn')" :inactive-text="t('account.enableOff')" />
      </el-form-item>
      <el-form-item :label="t('rbac.remark')">
        <el-input v-model="permForm.remark" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" :maxlength="255" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="permDialog = false">{{ t("account.cancel") }}</el-button>
      <el-button type="primary" :loading="submitting" @click="submitPerm">{{ t("account.confirm") }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { rbacApi } from "@/api/config";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Refresh, Plus, Edit as EditIcon, Delete as DeleteIcon, Search as SearchIcon, RefreshLeft as RefreshLeftIcon } from "@element-plus/icons-vue";

const { t } = useI18n();
defineOptions({ name: "RbacPermDictionary" });

const submitting = ref(false);

// ---------------- 权限字典 ----------------
const permLoading = ref(false);
const perms = ref<any[]>([]);
// 服务端分页 + 关键字查询
const keyword = ref("");
const total = ref(0);
const pg = reactive({ page: 1, pageSize: 10 });

async function loadPerms() {
  permLoading.value = true;
  try {
    const [cnt, list] = await Promise.all([
      rbacApi.perm.count({ keyword: keyword.value }),
      rbacApi.perm.query({ keyword: keyword.value, page: pg.page, pageSize: pg.pageSize })
    ]);
    total.value = (cnt as any)?.Data || 0;
    perms.value = (list as any)?.Data || [];
  } finally {
    permLoading.value = false;
  }
}
function onSearch() {
  pg.page = 1;
  loadPerms();
}
function onReset() {
  keyword.value = "";
  pg.page = 1;
  loadPerms();
}
function onSizeChange() {
  pg.page = 1;
  loadPerms();
}

const permDialog = ref(false);
const permFormRef = ref<FormInstance>();
const permForm = reactive({
  id: 0,
  perm_code: "",
  topic_key: "",
  match_sort: 0,
  effect: 1,
  status: 1,
  remark: ""
});
const permRules = reactive<FormRules>({
  perm_code: [{ required: true, message: t("rbac.permCode"), trigger: "blur" }],
  topic_key: [{ required: true, message: t("rbac.topicKey"), trigger: "blur" }]
});
function openPermAdd() {
  Object.assign(permForm, { id: 0, perm_code: "", topic_key: "", match_sort: 0, effect: 1, status: 1, remark: "" });
  permDialog.value = true;
}
function openPermEdit(row: any) {
  Object.assign(permForm, {
    id: row.id,
    perm_code: row.perm_code,
    topic_key: row.topic_key,
    match_sort: row.match_sort,
    effect: row.effect,
    status: row.status,
    remark: row.remark || ""
  });
  permDialog.value = true;
}
async function submitPerm() {
  if (!permFormRef.value) return;
  await permFormRef.value.validate(async valid => {
    if (!valid) return;
    submitting.value = true;
    try {
      const { id, ...rest } = permForm;
      if (id) {
        await rbacApi.perm.update({ id, ...rest });
      } else {
        await rbacApi.perm.add(rest);
      }
      ElMessage.success(t("rbac.saveSuccess"));
      permDialog.value = false;
      await loadPerms();
    } catch {
      /* 拦截器已提示 */
    } finally {
      submitting.value = false;
    }
  });
}
async function delPerm(row: any) {
  try {
    await ElMessageBox.confirm(t("rbac.delPermConfirm", { code: row.perm_code }), t("rbac.title"), { type: "warning" });
  } catch {
    return;
  }
  try {
    await rbacApi.perm.del({ id: row.id });
    ElMessage.success(t("account.delSuccess"));
    await loadPerms();
  } catch {
    /* 拦截器已提示 */
  }
}

onMounted(() => {
  loadPerms();
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

  .hint {
    font-size: 12px;
    color: #909399;
  }
}

.search-input {
  width: 220px;
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
</style>
