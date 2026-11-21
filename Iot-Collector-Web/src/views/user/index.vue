<template>
  <div class="account-page">
    <el-card class="box-card" shadow="never">
      <div class="page-toolbar">
        <div class="toolbar-left">
          <el-icon class="toolbar-icon"><component :is="UserIcon" /></el-icon>
          <span class="toolbar-title">{{ t("menus.pureUser") }}</span>
          <el-tag type="info" round size="small" class="ml-2">
            {{ t("account.total") }}: {{ rows.length }}
          </el-tag>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="keyword"
            :placeholder="t('account.searchPlaceholder')"
            :prefix-icon="SearchIcon"
            clearable
            class="search-input"
          />
          <el-button :icon="Refresh" circle @click="loadData" />
          <el-button type="primary" :icon="Plus" @click="handleAdd">{{ t("account.add") }}</el-button>
        </div>
      </div>

      <el-alert :title="t('account.userPageTip')" type="info" :closable="false" class="page-tip" show-icon />

      <el-table
        v-loading="loading"
        :data="pagedRows"
        border
        stripe
        highlight-current-row
        class="data-table"
      >
        <el-table-column prop="Id" :label="t('account.id')" width="72" align="center" />
        <el-table-column prop="Access_Key" :label="t('account.username')" min-width="160" show-overflow-tooltip />
        <el-table-column :label="t('account.enable')" width="100" align="center">
          <template #default="scope">
            <el-tag :type="scope.row.Enable == 1 ? 'success' : 'info'" size="small" effect="light">
              {{ scope.row.Enable == 1 ? t("account.enableOn") : t("account.enableOff") }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('account.expireAt')" width="180" align="center">
          <template #default="scope">
            <span :class="{ 'text-muted': isZeroTime(scope.row.Expire_At) }">{{ formatExpire(scope.row.Expire_At) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('account.accessTokenTTL')" width="130" align="center" prop="Access_Token_TTL" />
        <el-table-column :label="t('account.refreshTokenTTL')" width="150" align="center" prop="Refresh_Token_TTL" />
        <el-table-column :label="t('account.updateTime')" width="180" align="center" prop="Update_Time" />
        <el-table-column :label="t('common.operation')" width="190" align="center" fixed="right">
          <template #default="scope">
            <div class="op-btns">
              <el-button size="small" type="primary" plain :icon="EditIcon" @click="handleEdit(scope.row)">{{
                t("account.edit")
              }}</el-button>
              <el-button size="small" type="danger" plain :icon="DeleteIcon" @click="handleDel(scope.row)">{{
                t("account.del")
              }}</el-button>
            </div>
          </template>
        </el-table-column>
        <template #empty>
          <el-empty :description="t('account.noData')" />
        </template>
      </el-table>

      <el-pagination
        v-model:current-page="pg.page"
        v-model:page-size="pg.pageSize"
        :total="filteredRows.length"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        class="page-pagination"
      />
    </el-card>

    <!-- 新增/编辑 -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="560px" @close="handleDialogClose">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="130px" label-position="left">
        <el-form-item :label="t('account.username')" prop="Access_Key">
          <el-input v-model="form.Access_Key" :disabled="isEdit" :maxlength="64" :placeholder="t('account.accessKeyPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('account.password')" prop="Password">
          <el-input v-model="form.Password" type="password" show-password :placeholder="t('account.passwordPlaceholder')" />
          <div class="form-hint">{{ t("account.passwordTip") }}</div>
        </el-form-item>
        <el-form-item :label="t('account.enable')">
          <el-switch v-model="form.Enable" :active-value="1" :inactive-value="0" :active-text="t('account.enableOn')" :inactive-text="t('account.enableOff')" inline-prompt />
        </el-form-item>
        <el-form-item :label="t('account.expireAt')">
          <el-date-picker v-model="form.Expire_At" type="datetime" value-format="YYYY-MM-DDTHH:mm:ss" :placeholder="t('account.expireNever')" clearable />
        </el-form-item>
        <el-form-item :label="t('account.accessTokenTTL')">
          <el-input v-model="form.Access_Token_TTL" :maxlength="32" placeholder="15m0s" />
        </el-form-item>
        <el-form-item :label="t('account.refreshTokenTTL')">
          <el-input v-model="form.Refresh_Token_TTL" :maxlength="32" placeholder="168h0m0s" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("account.cancel") }}</el-button>
        <el-button type="primary" :disabled="disabled" @click="handleSubmit">{{ t("account.confirm") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { userApi } from "@/api/config";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Refresh, Plus, Search as SearchIcon, Edit as EditIcon, Delete as DeleteIcon, User as UserIcon } from "@element-plus/icons-vue";

const { t } = useI18n();
defineOptions({ name: "SystemUser" });

interface UserRow {
  Id: number;
  Access_Key: string;
  Enable: number;
  Expire_At: string;
  Access_Token_TTL: string;
  Refresh_Token_TTL: string;
  Create_Time: string;
  Update_Time: string;
}

const loading = ref(false);
const disabled = ref(false);
const rows = ref<UserRow[]>([]);
const keyword = ref("");

const filteredRows = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return rows.value;
  return rows.value.filter(r => r.Access_Key.toLowerCase().includes(kw));
});

const pg = reactive({ page: 1, pageSize: 10 });
const pagedRows = computed(() => {
  const start = (pg.page - 1) * pg.pageSize;
  return filteredRows.value.slice(start, start + pg.pageSize);
});

const dialogVisible = ref(false);
const isEdit = ref(false);
const formRef = ref<FormInstance>();
const form = reactive({
  Access_Key: "",
  Password: "",
  Enable: 1,
  Expire_At: "" as string | null,
  Access_Token_TTL: "15m0s",
  Refresh_Token_TTL: "168h0m0s"
});
const rules = computed<FormRules>(() => ({
  Access_Key: [{ required: true, message: t("account.accessKeyPlaceholder"), trigger: "blur" }],
  Password: isEdit.value
    ? []
    : [{ required: true, message: t("account.passwordRequired"), trigger: ["blur", "change"] }]
}));
const dialogTitle = computed(() => (isEdit.value ? t("account.editUserTitle") : t("account.addUserTitle")));

const loadData = async () => {
  loading.value = true;
  try {
    const res: any = await userApi.query({ page: 0, pageSize: 0 });
    rows.value = res?.Data || [];
  } finally {
    loading.value = false;
  }
};

const isZeroTime = (s?: string) => !s || s.startsWith("0001-01-01");
const formatExpire = (s?: string) => (isZeroTime(s) ? t("account.expireNever") : s);
const buildExpireAt = (v: string | null) => (v ? new Date(v).toISOString() : null);

const handleAdd = () => {
  isEdit.value = false;
  Object.assign(form, { Access_Key: "", Password: "", Enable: 1, Expire_At: "", Access_Token_TTL: "15m0s", Refresh_Token_TTL: "168h0m0s" });
  dialogVisible.value = true;
};

const handleEdit = (row: any) => {
  isEdit.value = true;
  Object.assign(form, {
    Access_Key: row.Access_Key,
    Password: "",
    Enable: row.Enable,
    Expire_At: isZeroTime(row.Expire_At) ? null : row.Expire_At,
    Access_Token_TTL: row.Access_Token_TTL || "15m0s",
    Refresh_Token_TTL: row.Refresh_Token_TTL || "168h0m0s"
  });
  dialogVisible.value = true;
};

const handleDialogClose = () => formRef.value?.clearValidate();

const handleSubmit = async () => {
  if (!formRef.value) return;
  await formRef.value.validate(async valid => {
    if (!valid) return;
    disabled.value = true;
    const expireAt = buildExpireAt(form.Expire_At);
    try {
      if (isEdit.value) {
        const payload: any = {
          access_key: form.Access_Key,
          enable: form.Enable,
          expire_at: expireAt,
          access_token_ttl: form.Access_Token_TTL,
          refresh_token_ttl: form.Refresh_Token_TTL
        };
        if (form.Password) payload.password = form.Password;
        await userApi.update(payload);
        ElMessage.success(t("account.editSuccess"));
      } else {
        await userApi.add({
          access_key: form.Access_Key,
          password: form.Password,
          enable: form.Enable,
          expire_at: expireAt,
          access_token_ttl: form.Access_Token_TTL,
          refresh_token_ttl: form.Refresh_Token_TTL
        });
        ElMessage.success(t("account.addSuccess"));
      }
      dialogVisible.value = false;
      await loadData();
    } catch {
      /* 拦截器已提示 */
    } finally {
      disabled.value = false;
    }
  });
};

const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("account.delConfirm"), t("account.title"), { type: "warning" });
  } catch {
    return;
  }
  try {
    await userApi.del([row.Id]);
    ElMessage.success(t("account.delSuccess"));
    await loadData();
  } catch {
    /* 拦截器已提示 */
  }
};

onMounted(loadData);
</script>

<style lang="scss" scoped>
.account-page {
  padding: 12px;
}

.page-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;

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

  .toolbar-right {
    display: flex;
    gap: 8px;
    align-items: center;

    .search-input {
      width: 200px;
    }
  }
}

.page-tip {
  margin-bottom: 12px;
}

.data-table {
  width: 100%;
}

.op-btns {
  display: flex;
  flex-wrap: nowrap;
  gap: 8px;
  align-items: center;
  justify-content: center;
  white-space: nowrap;

  :deep(.el-button + .el-button) {
    margin-left: 0;
  }
}

.page-pagination {
  justify-content: flex-end;
  margin-top: 14px;
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: #909399;
}

.text-muted {
  color: #a8abb2;
}

.ml-2 {
  margin-left: 8px;
}
</style>
