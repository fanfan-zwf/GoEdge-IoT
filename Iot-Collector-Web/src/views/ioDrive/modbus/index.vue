<template>
  <div class="main">
    <!-- 驱动配置卡片 -->
    <el-card shadow="never" class="config-card">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon :size="20" color="#409eff"><Monitor /></el-icon>
            <span>{{ t("modbus.driveTitle") }}</span>
          </div>
          <div class="header-actions">
            <el-button type="primary" @click="openDriveAdd">{{ t("common.add") }}</el-button>
          </div>
        </div>
      </template>

      <el-table :data="driveTableData" border style="width: 100%">
        <el-table-column prop="Id" :label="t('drive.id')" width="80" />
        <el-table-column prop="Name" :label="t('drive.name')" width="150" />
        <el-table-column :label="t('drive.config')">
          <el-table-column prop="ip" :label="t('modbus.ip')" width="140" />
          <el-table-column prop="port" :label="t('modbus.port')" width="80" />
          <el-table-column prop="retryTimeout" :label="t('modbus.retryTimeout')" width="100" />
          <el-table-column prop="connectTimeout" :label="t('modbus.connectTimeout')" width="100" />
          <el-table-column prop="responseTimeout" :label="t('modbus.responseTimeout')" width="110" />
          <el-table-column prop="delayBetweenPolls" :label="t('modbus.delayBetweenPolls')" width="100" />
          <el-table-column prop="packetMax" :label="t('modbus.packetMax')" width="100" />
        </el-table-column>
        <el-table-column prop="Points_Length" :label="t('drive.pointsLength')" width="100" />
        <el-table-column :label="t('common.operation')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDriveEdit(row)">{{ t("common.edit") }}</el-button>
            <el-button link type="warning" @click="handleDriveRestart(row)">{{ t("common.restart") }}</el-button>
            <el-button link type="danger" @click="handleDriveDel(row)">{{ t("common.del") }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 点位配置卡片 -->
    <el-card shadow="never" class="config-card">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon :size="20" color="#67C23A"><Connection /></el-icon>
            <span>{{ t("modbus.pointTitle") }}</span>
          </div>
          <div class="header-actions">
            <el-select v-model="selectedDriveId" :placeholder="t('modbus.selectDrive')" clearable style="width: 200px" @change="loadPointData">
              <el-option v-for="d in driveTableData" :key="d.Id" :label="`[${d.Id}] ${d.Name}`" :value="d.Id" />
            </el-select>
            <el-button type="primary" @click="openPointAdd" :disabled="!selectedDriveId">{{ t("common.add") }}</el-button>
          </div>
        </div>
      </template>

      <el-table :data="pointTableData" border style="width: 100%">
        <el-table-column prop="Id" :label="t('point.id')" width="80" />
        <el-table-column prop="Tag" :label="t('point.tag')" width="150" />
        <el-table-column :label="t('point.config')">
          <el-table-column prop="slaveId" :label="t('modbus.slaveId')" width="100" />
          <el-table-column prop="function" :label="t('modbus.function')" width="100" />
          <el-table-column prop="address" :label="t('modbus.address')" width="100" />
          <el-table-column prop="byteOrder" :label="t('modbus.byteOrder')" width="100" />
          <el-table-column prop="dataType" :label="t('modbus.dataType')" width="100" />
        </el-table-column>
        <el-table-column prop="Conversion" :label="t('point.conversion')" width="120" />
        <el-table-column prop="RW_Cancel" :label="t('point.rwCancel')" width="100">
          <template #default="{ row }">
            {{ rwCancelMap[row.RW_Cancel] || row.RW_Cancel }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openPointEdit(row)">{{ t("common.edit") }}</el-button>
            <el-button link type="danger" @click="handlePointDel(row)">{{ t("common.del") }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 驱动配置对话框 -->
    <el-dialog v-model="driveDialogVisible" :title="isDriveEdit ? t('drive.editTitle') : t('drive.addTitle')" width="600px" destroy-on-close>
      <el-form ref="driveFormRef" :model="driveForm" :rules="driveRules" label-width="120px">
        <el-form-item :label="t('drive.name')" prop="Name">
          <el-input v-model="driveForm.Name" />
        </el-form-item>
        <el-divider content-position="left">{{ t("modbus.driveConfig") }}</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.ip')" prop="ip">
              <el-input v-model="driveForm.ip" placeholder="192.168.1.1" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('modbus.port')" prop="port">
              <el-input v-model="driveForm.port" placeholder="502" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.retryTimeout')">
              <el-input v-model="driveForm.retryTimeout" placeholder="3s" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('modbus.connectTimeout')">
              <el-input v-model="driveForm.connectTimeout" placeholder="3s" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.responseTimeout')">
              <el-input v-model="driveForm.responseTimeout" placeholder="10s" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('modbus.delayBetweenPolls')">
              <el-input v-model="driveForm.delayBetweenPolls" placeholder="1s" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.packetMax')">
              <el-input v-model="driveForm.packetMax" placeholder="64" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="driveDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="driveSubmitting" @click="handleDriveSubmit">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>

    <!-- 点位配置对话框 -->
    <el-dialog v-model="pointDialogVisible" :title="isPointEdit ? t('point.editTitle') : t('point.addTitle')" width="600px" destroy-on-close>
      <el-form ref="pointFormRef" :model="pointForm" :rules="pointRules" label-width="120px">
        <el-form-item :label="t('point.tag')" prop="Tag">
          <el-input v-model="pointForm.Tag" />
        </el-form-item>
        <el-form-item :label="t('point.description')">
          <el-input v-model="pointForm.Description" />
        </el-form-item>
        <el-divider content-position="left">{{ t("modbus.pointConfig") }}</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.slaveId')" prop="slaveId">
              <el-input v-model="pointForm.slaveId" placeholder="1" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('modbus.function')" prop="function">
              <el-select v-model="pointForm.function" style="width: 100%">
                <el-option label="01 - Read Coils" value="01" />
                <el-option label="02 - Read Discrete Inputs" value="02" />
                <el-option label="03 - Read Holding Registers" value="03" />
                <el-option label="04 - Read Input Registers" value="04" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.address')" prop="address">
              <el-input v-model="pointForm.address" placeholder="100" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('modbus.byteOrder')" prop="byteOrder">
              <el-select v-model="pointForm.byteOrder" style="width: 100%">
                <el-option label="AB" value="AB" />
                <el-option label="BA" value="BA" />
                <el-option label="ABCD" value="ABCD" />
                <el-option label="ABDC" value="ABDC" />
                <el-option label="BACD" value="BACD" />
                <el-option label="DCBA" value="DCBA" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('modbus.dataType')" prop="dataType">
              <el-select v-model="pointForm.dataType" style="width: 100%">
                <el-option label="bool" value="bool" />
                <el-option label="uint16" value="uint16" />
                <el-option label="int16" value="int16" />
                <el-option label="uint32" value="uint32" />
                <el-option label="int32" value="int32" />
                <el-option label="float32" value="float32" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-divider content-position="left">{{ t("point.otherConfig") }}</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('point.conversion')">
              <el-input v-model="pointForm.Conversion" placeholder="x*1.5+10" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('point.rwCancel')">
              <el-select v-model="pointForm.RW_Cancel" style="width: 100%">
                <el-option :label="t('point.rwCancelForbidden')" :value="1" />
                <el-option :label="t('point.rwCancelRead')" :value="2" />
                <el-option :label="t('point.rwCancelWrite')" :value="3" />
                <el-option :label="t('point.rwCancelReadWrite')" :value="4" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="pointDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="pointSubmitting" @click="handlePointSubmit">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { message } from "@/utils/message";
import { ElMessageBox } from "element-plus";
import { Monitor, Connection } from "@element-plus/icons-vue";
import { configApi } from "@/api/config";
import type { FormInstance, FormRules } from "element-plus";

const { t } = useI18n();

// 读写方式映射
const rwCancelMap: Record<number, string> = {
  1: t("point.rwCancelForbidden"),
  2: t("point.rwCancelRead"),
  3: t("point.rwCancelWrite"),
  4: t("point.rwCancelReadWrite")
};

// 驱动表格数据（带解析后的配置）
const driveTableData = ref<any[]>([]);
// 点位表格数据
const pointTableData = ref<any[]>([]);
const selectedDriveId = ref<number | null>(null);

// 驱动对话框
const driveDialogVisible = ref(false);
const isDriveEdit = ref(false);
const driveSubmitting = ref(false);
const driveFormRef = ref<FormInstance>();
const driveForm = reactive({
  Id: 0,
  Name: "",
  ip: "",
  port: "502",
  retryTimeout: "3s",
  connectTimeout: "3s",
  responseTimeout: "10s",
  delayBetweenPolls: "1s",
  packetMax: "64"
});

const driveRules = reactive<FormRules>({
  Name: [{ required: true, message: () => t("common.required", { field: t("drive.name") }), trigger: "blur" }],
  ip: [{ required: true, message: () => t("common.required", { field: t("modbus.ip") }), trigger: "blur" }]
});

// 点位对话框
const pointDialogVisible = ref(false);
const isPointEdit = ref(false);
const pointSubmitting = ref(false);
const pointFormRef = ref<FormInstance>();
const pointForm = reactive({
  Id: 0,
  Drive_Id: 0,
  Tag: "",
  Description: "",
  slaveId: "",
  function: "03",
  address: "",
  byteOrder: "ABCD",
  dataType: "float32",
  Conversion: "",
  RW_Cancel: 2
});

const pointRules = reactive<FormRules>({
  Tag: [{ required: true, message: () => t("common.required", { field: t("point.tag") }), trigger: "blur" }],
  slaveId: [{ required: true, message: () => t("common.required", { field: t("modbus.slaveId") }), trigger: "blur" }],
  function: [{ required: true, message: () => t("common.required", { field: t("modbus.function") }), trigger: "change" }],
  address: [{ required: true, message: () => t("common.required", { field: t("modbus.address") }), trigger: "blur" }],
  byteOrder: [{ required: true, message: () => t("common.required", { field: t("modbus.byteOrder") }), trigger: "change" }],
  dataType: [{ required: true, message: () => t("common.required", { field: t("modbus.dataType") }), trigger: "change" }]
});

// 解析 Modbus 驱动配置字符串
const parseDriveConfig = (configStr: string) => {
  const parts = configStr.split(";");
  const addrParts = (parts[0] || "").split(":");
  return {
    ip: addrParts[0] || "",
    port: addrParts[1] || "502",
    retryTimeout: parts[1] || "3s",
    connectTimeout: parts[2] || "3s",
    responseTimeout: parts[3] || "10s",
    delayBetweenPolls: parts[4] || "1s",
    packetMax: parts[5] || "64"
  };
};

// 序列化驱动配置
const serializeDriveConfig = () => {
  return `${driveForm.ip}:${driveForm.port};${driveForm.retryTimeout};${driveForm.connectTimeout};${driveForm.responseTimeout};${driveForm.delayBetweenPolls};${driveForm.packetMax}`;
};

// 解析 Modbus 点位配置字符串
const parsePointConfig = (configStr: string) => {
  const parts = configStr.split(";");
  return {
    slaveId: parts[0] || "",
    function: parts[1] || "03",
    address: parts[2] || "",
    byteOrder: parts[3] || "ABCD",
    dataType: parts[4] || "float32"
  };
};

// 序列化点位配置
const serializePointConfig = () => {
  return `${pointForm.slaveId};${pointForm.function};${pointForm.address};${pointForm.byteOrder};${pointForm.dataType}`;
};

// 加载驱动数据
const loadDriveData = async () => {
  try {
    const res = (await configApi.drive.search({
      name: "%%",
      types: ["Modbus_Tcp"],
      quantity: 100
    })) as any;
    const data = res.Data || [];
    driveTableData.value = data.map((d: any) => ({
      ...d,
      ...parseDriveConfig(d.Config)
    }));
  } catch {
    /* 拦截器已提示错误 */
  }
};

// 加载点位数据
const loadPointData = async () => {
  if (!selectedDriveId.value) {
    pointTableData.value = [];
    return;
  }
  try {
    const res = (await configApi.point.search({
      tag: "%%",
      driveIds: [selectedDriveId.value],
      quantity: 100
    })) as any;
    const data = res.Data || [];
    pointTableData.value = data.map((p: any) => ({
      ...p,
      ...parsePointConfig(p.Config)
    }));
  } catch {
    /* 拦截器已提示错误 */
  }
};

// 驱动操作
const resetDriveForm = () => {
  driveForm.Id = 0;
  driveForm.Name = "";
  driveForm.ip = "";
  driveForm.port = "502";
  driveForm.retryTimeout = "3s";
  driveForm.connectTimeout = "3s";
  driveForm.responseTimeout = "10s";
  driveForm.delayBetweenPolls = "1s";
  driveForm.packetMax = "64";
};

const openDriveAdd = () => {
  isDriveEdit.value = false;
  resetDriveForm();
  driveDialogVisible.value = true;
};

const openDriveEdit = (row: any) => {
  isDriveEdit.value = true;
  driveForm.Id = row.Id;
  driveForm.Name = row.Name;
  const config = parseDriveConfig(row.Config);
  driveForm.ip = config.ip;
  driveForm.port = config.port;
  driveForm.retryTimeout = config.retryTimeout;
  driveForm.connectTimeout = config.connectTimeout;
  driveForm.responseTimeout = config.responseTimeout;
  driveForm.delayBetweenPolls = config.delayBetweenPolls;
  driveForm.packetMax = config.packetMax;
  driveDialogVisible.value = true;
};

const handleDriveSubmit = async () => {
  if (!driveFormRef.value) return;
  await driveFormRef.value.validate();
  driveSubmitting.value = true;
  try {
    const configStr = serializeDriveConfig();
    if (isDriveEdit.value) {
      await configApi.drive.update([{ Id: driveForm.Id, Name: driveForm.Name, Config: configStr }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.drive.add([{ Type: "Modbus_Tcp", Name: driveForm.Name, Config: configStr }]);
      message(t("common.addSuccess"), { type: "success" });
    }
    driveDialogVisible.value = false;
    loadDriveData();
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    driveSubmitting.value = false;
  }
};

const handleDriveDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("common.deleteConfirm", { name: `[${row.Id}] ${row.Name}` }), t("common.warning"), { type: "warning" });
    await configApi.drive.del([row.Id]);
    message(t("common.deleteSuccess"), { type: "success" });
    loadDriveData();
  } catch {
    /* 取消 */
  }
};

const handleDriveRestart = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("common.restartConfirm", { name: `[${row.Id}] ${row.Name}` }), t("common.warning"), { type: "warning" });
    await configApi.drive.restart(row.Id);
    message(t("common.restartSuccess"), { type: "success" });
  } catch {
    /* 取消 */
  }
};

// 点位操作
const resetPointForm = () => {
  pointForm.Id = 0;
  pointForm.Drive_Id = selectedDriveId.value || 0;
  pointForm.Tag = "";
  pointForm.Description = "";
  pointForm.slaveId = "";
  pointForm.function = "03";
  pointForm.address = "";
  pointForm.byteOrder = "ABCD";
  pointForm.dataType = "float32";
  pointForm.Conversion = "";
  pointForm.RW_Cancel = 2;
};

const openPointAdd = () => {
  isPointEdit.value = false;
  resetPointForm();
  pointDialogVisible.value = true;
};

const openPointEdit = (row: any) => {
  isPointEdit.value = true;
  pointForm.Id = row.Id;
  pointForm.Drive_Id = row.Drive_Id;
  pointForm.Tag = row.Tag;
  pointForm.Description = row.Description;
  const config = parsePointConfig(row.Config);
  pointForm.slaveId = config.slaveId;
  pointForm.function = config.function;
  pointForm.address = config.address;
  pointForm.byteOrder = config.byteOrder;
  pointForm.dataType = config.dataType;
  pointForm.Conversion = row.Conversion || "";
  pointForm.RW_Cancel = row.RW_Cancel || 2;
  pointDialogVisible.value = true;
};

const handlePointSubmit = async () => {
  if (!pointFormRef.value) return;
  await pointFormRef.value.validate();
  pointSubmitting.value = true;
  try {
    const configStr = serializePointConfig();
    if (isPointEdit.value) {
      await configApi.point.update([{
        Id: pointForm.Id,
        Drive_Id: pointForm.Drive_Id,
        Tag: pointForm.Tag,
        Description: pointForm.Description,
        Config: configStr,
        Conversion: pointForm.Conversion,
        RW_Cancel: pointForm.RW_Cancel
      }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.point.add([{
        Drive_Id: pointForm.Drive_Id,
        Tag: pointForm.Tag,
        Description: pointForm.Description,
        Config: configStr,
        Conversion: pointForm.Conversion,
        RW_Cancel: pointForm.RW_Cancel
      }]);
      message(t("common.addSuccess"), { type: "success" });
    }
    pointDialogVisible.value = false;
    loadPointData();
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    pointSubmitting.value = false;
  }
};

const handlePointDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("common.deleteConfirm", { name: row.Tag }), t("common.warning"), { type: "warning" });
    await configApi.point.del([row.Id]);
    message(t("common.deleteSuccess"), { type: "success" });
    loadPointData();
  } catch {
    /* 取消 */
  }
};

onMounted(() => {
  loadDriveData();
});
</script>

<style scoped>
.main {
  padding: 16px;
}

.config-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header-title {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 16px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
