<template>
  <div class="main">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">{{ t("history.title") }}</span>
          <div class="header-actions">
            <el-button type="primary" @click="openAdd">{{ t("common.add") }}</el-button>
            <el-button @click="handleExport">{{ t("common.export") }}</el-button>
            <el-button @click="importDialogVisible = true">{{ t("common.import") }}</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" border max-height="calc(100vh - 360px)" style="width: 100%" :default-sort="{ prop: 'Id', order: 'ascending' }">
        <el-table-column prop="Id" :label="t('history.id')" width="80" sortable />
        <el-table-column prop="Point_Id" :label="t('history.pointId')" width="100" sortable />
        <el-table-column prop="Config" :label="t('history.config')" show-overflow-tooltip sortable />
        <el-table-column prop="Creation_Time" :label="t('common.creationTime')" width="130" sortable>
          <template #default="{ row }">
            <div>{{ formatDate(row.Creation_Time) }}</div>
            <div>{{ formatTime(row.Creation_Time) }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="220" fixed="right" :expandable="false">
          <template #default="{ row }">
            <el-button link type="primary" @click="viewHistoryData(row)">{{ t("history.viewData") }}</el-button>
            <el-button link type="primary" @click="openEdit(row)">{{ t("common.edit") }}</el-button>
            <el-button link type="danger" @click="handleDel(row)">{{ t("common.del") }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 30, 50, 100, 150]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? t('history.editTitle') : t('history.addTitle')"
      width="500px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('history.pointId')" prop="Point_Id">
          <div class="point-select-trigger">
            <span class="point-select-label">{{ selectedPointLabel || t('history.pointIdPlaceholder') }}</span>
            <el-button size="small" @click="pointSelectorVisible = true">{{ t("pointSelector.title") }}</el-button>
          </div>
        </el-form-item>
        <el-form-item prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t("history.config") }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a
                  href="https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/db/db_point/README.md"
                  target="_blank"
                  class="help-icon"
                >
                  ?
                </a>
              </el-tooltip>
            </span>
          </template>
          <el-select v-model="form.Config" :placeholder="t('common.historyConfigTip')" style="width: 100%">
            <el-option label="启用" value="record" />
            <el-option label="禁用" value="null" />
          </el-select>
          <div class="config-tip">{{ historyConfigFormatTip }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>

    <!-- 导入对话框 -->
    <el-dialog v-model="importDialogVisible" :title="t('common.importTitle')" width="480px" @close="importFile = null">
      <p style=" margin-bottom: 16px; font-size: 13px;color: #909399">{{ t("common.importTip") }}</p>
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :limit="1"
        accept=".xlsx"
        :on-change="(file: any) => (importFile = file.raw)"
        :on-remove="() => (importFile = null)"
      >
        <el-button type="primary">{{ t("common.importSelectFile") }}</el-button>
      </el-upload>
      <template #footer>
        <el-button @click="importDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="importLoading" @click="handleImport">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>

    <!-- 点位选择器 -->
    <PointSelector
      :visible="pointSelectorVisible"
      :multiple="false"
      :selected="form.Point_Id"
      @update:visible="pointSelectorVisible = $event"
      @confirm="onPointSelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import PointSelector from "@/components/RePointSelector/src/index.vue";
import { message } from "@/utils/message";
import { ElMessageBox, type FormInstance, type FormRules } from "element-plus";
import { configApi, exportConfig, importConfig } from "@/api/config";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";

const router = useRouter();
const { t } = useI18n();

const historyConfigFormatTip = computed(() => t("common.historyConfigFormat"));

const pointList = ref<any[]>([]);
const pointSelectorVisible = ref(false);

const selectedPointLabel = computed(() => {
  if (!form.Point_Id) return "";
  const p = pointList.value.find(x => x.Id === form.Point_Id);
  return p ? `[${p.Id}] ${p.Tag || p.Config || ""}` : `#${form.Point_Id}`;
});

const onPointSelected = (points: any[]) => {
  if (points.length > 0) {
    form.Point_Id = points[0].Id;
  }
};
const loadPointList = async () => {
  try {
    const res = (await configApi.point.query({
      page: 1,
      pageSize: 1000
    })) as any;
    pointList.value = res.Data || [];
  } catch {
    /* ignore */
  }
};

const tableData = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);

const dialogVisible = ref(false);
const isEdit = ref(false);
const submitting = ref(false);
const formRef = ref<FormInstance>();
const form = reactive({
  Id: 0,
  Point_Id: 1,
  Config: "1"
});

const rules = reactive<FormRules>({
  Point_Id: [
    {
      required: true,
      message: () => t("common.required", { field: t("history.pointId") }),
      trigger: "blur"
    }
  ],
  Config: [
    {
      required: true,
      message: () => t("common.required", { field: t("history.config") }),
      trigger: "blur"
    }
  ]
});

const formatDate = (time: string) => {
  if (!time) return "-";
  return new Date(time).toLocaleDateString();
};

const formatTime = (time: string) => {
  if (!time) return "-";
  return new Date(time).toLocaleTimeString() + "+08:00";
};

const loadData = async () => {
  try {
    const countRes = (await configApi.history.count({
      page: 0,
      pageSize: 0
    })) as any;
    total.value = countRes.Data || 0;
    if (total.value === 0) {
      tableData.value = [];
      return;
    }
    const res = (await configApi.history.query({
      page: page.value,
      pageSize: pageSize.value
    })) as any;
    tableData.value = res.Data || [];
  } catch {
    /* 拦截器已提示错误 */
  }
};

const handleSizeChange = (val: number) => {
  pageSize.value = val;
  page.value = 1;
  loadData();
};

const handleCurrentChange = (val: number) => {
  page.value = val;
  loadData();
};

const resetForm = () => {
  form.Id = 0;
  form.Point_Id = 1;
  form.Config = "1";
};

const openAdd = () => {
  isEdit.value = false;
  resetForm();
  loadPointList();
  dialogVisible.value = true;
};

const openEdit = (row: any) => {
  isEdit.value = true;
  loadPointList();
  form.Id = row.Id;
  form.Point_Id = row.Point_Id;
  form.Config = row.Config;
  dialogVisible.value = true;
};

const handleSubmit = async () => {
  if (!formRef.value) return;
  await formRef.value.validate();
  submitting.value = true;
  try {
    const data = { Point_Id: form.Point_Id, Config: form.Config };
    if (isEdit.value) {
      await configApi.history.update([{ Id: form.Id, ...data }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.history.add([data]);
      message(t("common.addSuccess"), { type: "success" });
    }
    dialogVisible.value = false;
    loadData();
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    submitting.value = false;
  }
};

const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("common.deleteConfirm", { name: `ID: ${row.Id}` }), t("common.warning"), {
      type: "warning"
    });
    await configApi.history.del([row.Id]);
    message(t("common.deleteSuccess"), { type: "success" });
    loadData();
  } catch (error: any) {
    if (error !== "cancel") {
      /* 拦截器已提示错误 */
    }
  }
};

const handleExport = () => {
  exportConfig("history");
};

const viewHistoryData = (row: any) => {
  const targetPath = `/history-data/${row.Point_Id}`;
  const pointIdStr = String(row.Point_Id);
  // 从 pointList 中获取点位 Tag 作为标题
  const point = pointList.value.find(p => p.Id === row.Point_Id);
  const tagTitle = point?.Tag || t("menus.pureHistoryData");
  // 检查标签页是否已存在
  const tags = useMultiTagsStoreHook().multiTags;
  const tagExists = tags.some((t: any) => t.path === targetPath);
  if (!tagExists) {
    // 添加新标签页
    useMultiTagsStoreHook().handleTags("push", {
      path: targetPath,
      name: "HistoryData",
      params: { pointId: pointIdStr },
      meta: {
        title: tagTitle
      }
    });
  }
  // 跳转到目标页面
  router.push({ name: "HistoryData", params: { pointId: pointIdStr } });
};

const importDialogVisible = ref(false);
const importFile = ref<File | null>(null);
const importLoading = ref(false);

const handleImport = async () => {
  if (!importFile.value) {
    message(t("common.importSelectFile"), { type: "warning" });
    return;
  }
  importLoading.value = true;
  try {
    const res = (await importConfig(importFile.value)) as any;
    const data = res.Data || {};
    const history = data.history_config || {
      added: 0,
      updated: 0,
      deleted: 0
    };
    message(
      t("common.importSuccess", {
        added: history.added,
        updated: history.updated,
        deleted: history.deleted
      }),
      { type: "success" }
    );
    importDialogVisible.value = false;
    loadData();
  } catch (e: any) {
    message(e.message || "Import failed", { type: "error" });
  } finally {
    importLoading.value = false;
  }
};

onMounted(() => {
  loadData();
});
</script>

<style scoped>
.main {
  padding: 16px;
}

.point-select-trigger {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;
}

.point-select-label {
  flex: 1;
  overflow: hidden;
  font-size: 13px;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-header {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.label-with-help {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.help-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  font-size: 12px;
  line-height: 1;
  color: #fff;
  text-decoration: none;
  background: var(--el-color-primary);
  border-radius: 50%;
}

.config-tip {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: pre-line;
}
</style>
