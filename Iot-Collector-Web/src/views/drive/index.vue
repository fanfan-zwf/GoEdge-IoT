<template>
  <div class="main">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">{{ t("drive.title") }}</span>
          <div class="header-actions">
            <el-input v-model="searchName" :placeholder="t('common.searchPlaceholder')" clearable style="width: 200px"
              @keyup.enter="handleSearch" />
            <el-select v-model="searchType" :placeholder="t('drive.type')" clearable style="width: 160px">
              <el-option v-for="item in driveTypes" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button @click="handleSearch">{{ t("common.search") }}</el-button>
            <el-button @click="handleReset">{{ t("common.reset") }}</el-button>
            <el-button type="primary" @click="openAdd">{{ t("common.add") }}</el-button>
            <el-button @click="handleExport">{{ t("common.export") }}</el-button>
            <el-button @click="importDialogVisible = true">{{ t("common.import") }}</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" border max-height="calc(100vh - 360px)" style="width: 100%"
        :default-sort="{ prop: 'Id', order: 'ascending' }">
        <el-table-column prop="Id" :label="t('drive.id')" width="80" sortable />
        <el-table-column prop="Type" :label="t('drive.type')" width="150" />
        <el-table-column prop="Name" :label="t('drive.name')" width="150" />
        <el-table-column prop="Config" :label="t('drive.config')" min-width="220">
          <template #default="{ row }">
            <el-popover
              v-if="row.Config"
              trigger="click"
              placement="right"
              :width="440"
              popper-class="config-json-popper"
            >
              <template #reference>
                <span class="config-cell">{{ configPreview(row.Config) }}</span>
              </template>
              <pre class="config-json">{{ formatConfigJson(row.Config) }}</pre>
            </el-popover>
            <span v-else style="color: #999">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="Points_Length" :label="t('drive.pointsLength')" width="100" />
        <el-table-column prop="Creation_Time" :label="t('common.creationTime')" width="130">
          <template #default="{ row }">
            <div>{{ formatDate(row.Creation_Time) }}</div>
            <div>{{ formatTime(row.Creation_Time) }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="220" fixed="right" :expandable="false">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">{{ t("common.edit") }}</el-button>
            <el-button link type="warning" :loading="restartingId === row.Id" @click="handleRestart(row)">
              {{ t("common.restart") }}
            </el-button>
            <el-button link type="danger" @click="handleDel(row)">{{ t("common.del") }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 30, 50, 100, 150]" layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange" @current-change="handleCurrentChange" />
      </div>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('drive.editTitle') : t('drive.addTitle')" width="550px"
      destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item :label="t('drive.type')" prop="Type">
          <el-select v-model="form.Type" :placeholder="t('drive.typePlaceholder')" :disabled="isEdit"
            style="width: 100%" @change="handleTypeChange">
            <el-option v-for="item in driveTypes" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('drive.name')" prop="Name">
          <el-input v-model="form.Name" />
        </el-form-item>

        <!-- Modbus TCP 配置表单（字段与 form.Config 双向同步） -->
        <ModbusForm v-if="form.Type === 'Modbus_Tcp'" v-model="modbusForm" v-model:config="form.Config" />

        <!-- Siemens S7 配置表单（字段与 form.Config 双向同步） -->
        <SiemensForm v-else-if="form.Type === 'Siemens_S7'" v-model="s7Form" v-model:config="form.Config" />

        <!-- 其他驱动类型使用原始 textarea -->
        <el-form-item v-else prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t("common.config") }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a :href="docUrl" target="_blank" class="help-icon">?</a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Config" type="textarea" :rows="4" :placeholder="t('common.driveConfigPlaceholder')" />
          <div class="config-tip">{{ t("common.driveConfigTip") }}</div>
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
      <el-upload ref="uploadRef" :auto-upload="false" :limit="1" accept=".xlsx"
        :on-change="(file: any) => (importFile = file.raw)" :on-remove="() => (importFile = null)">
        <el-button type="primary">{{ t("common.importSelectFile") }}</el-button>
      </el-upload>
      <template #footer>
        <el-button @click="importDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="importLoading" @click="handleImport">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { message } from "@/utils/message";
import { ElMessageBox } from "element-plus";
import { configApi, exportConfig, importConfig } from "@/api/config";
import { CONFIG_MAX_LENGTH, ISSUE_MESSAGE_KEY, checkDriveConfig, type FormatIssue } from "@/utils/configFormat";
import type { FormInstance, FormRules } from "element-plus";
import ModbusForm from "@/views/ioDrive/modbus/form.vue";
import SiemensForm from "@/views/ioDrive/siemens/form.vue";
import type { ModbusConfig, SiemensConfig } from "@/views/ioDrive/types";
import { MODBUS_CONFIG_DEFAULTS, SIEMENS_CONFIG_DEFAULTS } from "@/views/ioDrive/types";

const { t } = useI18n();

const driveTypes = computed(() => [
  { value: "Modbus_Tcp", label: "Modbus TCP" },
  { value: "Siemens_S7", label: "Siemens S7" }
]);

const docUrl = computed(() => {
  const base = "https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/IO";
  if (form.Type === "Siemens_S7") return `${base}/Siemens_S7/README.md`;
  return `${base}/Modbus_Tcp/README.md`;
});

// Modbus TCP 配置
const modbusForm = reactive<ModbusConfig>({ ...MODBUS_CONFIG_DEFAULTS });

// Siemens S7 配置
const s7Form = reactive<SiemensConfig>({ ...SIEMENS_CONFIG_DEFAULTS });

// 重置两个可视化表单，避免上一条记录的字段残留
const resetDriveForms = () => {
  Object.assign(modbusForm, MODBUS_CONFIG_DEFAULTS);
  Object.assign(s7Form, SIEMENS_CONFIG_DEFAULTS);
};

// 驱动类型切换时重置表单：清空 Config 后由对应的可视化子组件按默认值重新生成
const handleTypeChange = () => {
  resetDriveForms();
  form.Config = "";
};

const tableData = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);

const searchName = ref("%%");
const searchType = ref("");
const isSearching = ref(false);

const dialogVisible = ref(false);
const isEdit = ref(false);
const submitting = ref(false);
const restartingId = ref(0);
const formRef = ref<FormInstance>();
const form = reactive({
  Id: 0,
  Type: "",
  Name: "",
  Config: ""
});

const rules = reactive<FormRules>({
  Type: [
    {
      required: true,
      message: () => t("common.required", { field: t("drive.type") }),
      trigger: "blur"
    }
  ],
  Name: [
    {
      required: true,
      message: () => t("common.required", { field: t("drive.name") }),
      trigger: "blur"
    }
  ],
  Config: [
    {
      required: true,
      message: () => t("common.required", { field: t("common.config") }),
      trigger: "blur"
    },
    // 与后端 mysql.Config__Validate_JSON 同源：驱动侧 Drive_Config_Switch 直接反序列化，非 JSON 只会拖到重启才爆
    {
      validator: (_rule: any, value: string, callback: (error?: Error) => void) => {
        const issue: FormatIssue = checkDriveConfig(value ?? "");
        callback(issue ? new Error(t(ISSUE_MESSAGE_KEY[issue], { max: CONFIG_MAX_LENGTH })) : undefined);
      },
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

// 把驱动 Config 里的 JSON 字符串美化成缩进 2 空格的多行文本；非 JSON 时原样返回
const formatConfigJson = (raw: string): string => {
  if (!raw) return "";
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
};

// 列表单元格内的紧凑预览：把 JSON 展平成 key=value, ... 一行，过长截断；非 JSON 直接截断原文
const configPreview = (raw: string): string => {
  if (!raw) return "-";
  let text: string;
  try {
    const obj = JSON.parse(raw);
    if (obj && typeof obj === "object") {
      text = Object.entries(obj)
        .map(([k, v]) => `${k}=${typeof v === "object" ? JSON.stringify(v) : v}`)
        .join(", ");
    } else {
      text = String(obj);
    }
  } catch {
    text = raw;
  }
  return text.length > 42 ? `${text.slice(0, 42)}…` : text;
};

const loadData = async () => {
  try {
    if (isSearching.value) {
      const res = (await configApi.drive.search({
        name: searchName.value,
        types: searchType.value ? [searchType.value] : undefined,
        quantity: 30
      })) as any;
      tableData.value = res.Data || [];
      total.value = tableData.value.length;
      return;
    }
    const countRes = (await configApi.drive.count({
      page: 0,
      pageSize: 0
    })) as any;
    total.value = countRes.Data || 0;
    if (total.value === 0) {
      tableData.value = [];
      return;
    }
    const res = (await configApi.drive.query({
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
  isSearching.value = false;
  loadData();
};

const handleCurrentChange = (val: number) => {
  page.value = val;
  isSearching.value = false;
  loadData();
};

const handleSearch = () => {
  isSearching.value = true;
  loadData();
};

const handleReset = () => {
  searchName.value = "%%";
  searchType.value = "";
  isSearching.value = false;
  loadData();
};

const resetForm = () => {
  form.Id = 0;
  form.Type = "";
  form.Name = "";
  form.Config = "";
  resetDriveForms();
};

const openAdd = () => {
  isEdit.value = false;
  resetForm();
  dialogVisible.value = true;
};

const openEdit = (row: any) => {
  isEdit.value = true;
  form.Id = row.Id;
  form.Type = row.Type;
  form.Name = row.Name;
  resetDriveForms();
  // 子组件挂载时会把 Config 解析回结构化字段，这里只需原样传入
  form.Config = row.Config;
  dialogVisible.value = true;
};

const handleSubmit = async () => {
  if (!formRef.value) return;
  await formRef.value.validate();
  submitting.value = true;
  // form.Config 已由子组件实时同步，无需再按驱动类型分支序列化
  const config = form.Config;
  try {
    if (isEdit.value) {
      await configApi.drive.update([{ Id: form.Id, Name: form.Name, Config: config }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.drive.add([{ Type: form.Type, Name: form.Name, Config: config }]);
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
    await configApi.drive.del([row.Id]);
    message(t("common.deleteSuccess"), { type: "success" });
    loadData();
  } catch (error: any) {
    if (error !== "cancel") {
      /* 拦截器已提示错误 */
    }
  }
};

const handleRestart = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      t("common.restartConfirm", {
        name: `[${row.Id}] ${row.Name || row.Type}`
      }),
      t("common.warning"),
      { type: "warning" }
    );
    restartingId.value = row.Id;
    await configApi.drive.restart(row.Id);
    message(t("common.restartSuccess"), { type: "success" });
  } catch (error: any) {
    if (error !== "cancel") {
      /* 拦截器已提示错误 */
    }
  } finally {
    restartingId.value = 0;
  }
};

const handleExport = () => {
  exportConfig("drive");
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
    const drive = data.drive_config || { added: 0, updated: 0, deleted: 0 };
    message(
      t("common.importSuccess", {
        added: drive.added,
        updated: drive.updated,
        deleted: drive.deleted
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

.config-cell {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--el-color-primary);
  white-space: nowrap;
  cursor: pointer;
}

.config-cell:hover {
  text-decoration: underline;
}
</style>

<style>
/* el-popover 内容 teleport 到 body，scoped 样式无法命中，故用 popper-class 限定作用域的全局样式 */
.config-json-popper .config-json {
  max-height: 360px;
  margin: 0;
  overflow: auto;
  font-family: Menlo, Consolas, "Courier New", monospace;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre;
  word-break: normal;
}
</style>
