<template>
  <div class="main">
    <!-- 驱动配置卡片 -->
    <el-card shadow="never" class="config-card">
      <template #header>
        <div class="card-header">
          <div class="header-title">
            <el-icon :size="20" color="#409eff"><Monitor /></el-icon>
            <span>{{ t("s7.driveTitle") }}</span>
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
          <el-table-column prop="ip" :label="t('s7.ip')" width="140" />
          <el-table-column prop="rack" :label="t('s7.rack')" width="70" />
          <el-table-column prop="slot" :label="t('s7.slot')" width="70" />
          <el-table-column prop="retryTimeout" :label="t('s7.retryTimeout')" width="90" />
          <el-table-column prop="delayBetweenPolls" :label="t('s7.delayBetweenPolls')" width="90" />
          <el-table-column prop="connectTimeout" :label="t('s7.connectTimeout')" width="90" />
          <el-table-column prop="responseTimeout" :label="t('s7.responseTimeout')" width="100" />
          <el-table-column prop="packetMax" :label="t('s7.packetMax')" width="90" />
          <el-table-column prop="connectionType" :label="t('s7.connectionType')" width="100" />
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
            <span>{{ t("s7.pointTitle") }}</span>
          </div>
          <div class="header-actions">
            <el-select v-model="selectedDriveId" :placeholder="t('s7.selectDrive')" clearable style="width: 200px" @change="loadPointData">
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
          <el-table-column prop="area" :label="t('s7.area')" width="80" />
          <el-table-column prop="dbNumber" :label="t('s7.dbNumber')" width="90" />
          <el-table-column prop="start" :label="t('s7.start')" width="80" />
          <el-table-column prop="dataType" :label="t('s7.dataType')" width="110" />
          <el-table-column prop="childAddress" :label="t('s7.childAddress')" width="80" />
        </el-table-column>
        <el-table-column prop="valueTypeLabel" :label="t('s7.valueType')" width="90" />
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
    <el-dialog v-model="driveDialogVisible" :title="isDriveEdit ? t('drive.editTitle') : t('drive.addTitle')" width="700px" destroy-on-close>
      <el-form ref="driveFormRef" :model="driveForm" :rules="driveRules" label-width="120px">
        <el-form-item :label="t('drive.name')" prop="Name">
          <el-input v-model="driveForm.Name" />
        </el-form-item>
        <el-divider content-position="left">{{ t("s7.driveConfig") }}</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.ip')" prop="ip">
              <el-input v-model="driveForm.ip" placeholder="192.168.1.1" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item :label="t('s7.rack')" prop="rack">
              <el-input v-model="driveForm.rack" placeholder="0" />
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item :label="t('s7.slot')" prop="slot">
              <el-input v-model="driveForm.slot" placeholder="2" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.retryTimeout')">
              <el-input v-model="driveForm.retryTimeout" placeholder="3s" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.delayBetweenPolls')">
              <el-input v-model="driveForm.delayBetweenPolls" placeholder="0" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.connectTimeout')">
              <el-input v-model="driveForm.connectTimeout" placeholder="10s" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.responseTimeout')">
              <el-input v-model="driveForm.responseTimeout" placeholder="10s" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.packetMax')">
              <el-input v-model="driveForm.packetMax" placeholder="480" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.connectionType')">
              <el-select v-model="driveForm.connectionType" style="width: 100%">
                <el-option label="PG (编程设备)" :value="1" />
                <el-option label="OP (操作面板)" :value="2" />
                <el-option label="Basic (基本连接)" :value="3" />
              </el-select>
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
        <el-divider content-position="left">{{ t("s7.pointConfig") }}</el-divider>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.area')" prop="area">
              <el-select v-model="pointForm.area" style="width: 100%">
                <el-option label="I (过程输入)" value="I" />
                <el-option label="Q (过程输出)" value="Q" />
                <el-option label="M (标志位)" value="M" />
                <el-option label="DB (数据块)" value="DB" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.dbNumber')" prop="dbNumber">
              <el-input v-model="pointForm.dbNumber" placeholder="1" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.start')" prop="start">
              <el-input v-model="pointForm.start" placeholder="0" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.dataType')" prop="dataType">
              <el-select v-model="pointForm.dataType" style="width: 100%">
                <el-option
                  v-for="item in S7_DATA_TYPE_OPTIONS"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label="t('s7.childAddress')">
              <el-input v-model="pointForm.childAddress" placeholder="0 (Bool 位偏移)" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('s7.valueType')" prop="Value_Type">
              <el-select v-model="pointForm.Value_Type" style="width: 100%">
                <el-option
                  v-for="item in S7_VALUE_TYPE_OPTIONS"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
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
import { S7_DATA_TYPE_OPTIONS, S7_VALUE_TYPE_OPTIONS } from "@/views/ioDrive/types";

const { t } = useI18n();

// 读写方式映射
const rwCancelMap: Record<number, string> = {
  1: t("point.rwCancelForbidden"),
  2: t("point.rwCancelRead"),
  3: t("point.rwCancelWrite"),
  4: t("point.rwCancelReadWrite")
};

// 存储区映射
const areaMap: Record<string, number> = { "I": 0x81, "Q": 0x82, "M": 0x83, "DB": 0x84 };
const areaMapReverse: Record<number, string> = { 0x81: "I", 0x82: "Q", 0x83: "M", 0x84: "DB" };

// 采集类型代码 ↔ TIA Portal 类型名（key 统一小写，与后端 new.go s7TypeMap 对齐，代码值↔conv.TYPE_*）
const dataTypeMap: Record<string, number> = {
  bool: 1, sint: 2, usint: 3, byte: 3,
  int: 4, uint: 5, word: 5,
  dint: 6, udint: 7, dword: 7,
  lint: 8, ulint: 9, lword: 9,
  real: 12, lreal: 13, string: 15
};
// 代码值 → 展示 label（仅用于表格列展示，取首选名，与 S7_DATA_TYPE_OPTIONS 一致）
const dataTypeMapReverse: Record<number, string> = Object.fromEntries(
  S7_DATA_TYPE_OPTIONS.map(o => [o.value, o.label])
);

// 输出类型代码 ↔ 展示 label（直接复用 S7_VALUE_TYPE_OPTIONS）
const valueTypeMapReverse: Record<number, string> = Object.fromEntries(
  S7_VALUE_TYPE_OPTIONS.map(o => [o.value, o.label])
);

// 驱动表格数据
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
  rack: "0",
  slot: "2",
  retryTimeout: "3s",
  delayBetweenPolls: "0",
  connectTimeout: "10s",
  responseTimeout: "10s",
  packetMax: "480",
  connectionType: 2
});

const driveRules = reactive<FormRules>({
  Name: [{ required: true, message: () => t("common.required", { field: t("drive.name") }), trigger: "blur" }],
  ip: [{ required: true, message: () => t("common.required", { field: t("s7.ip") }), trigger: "blur" }],
  rack: [{ required: true, message: () => t("common.required", { field: t("s7.rack") }), trigger: "blur" }],
  slot: [{ required: true, message: () => t("common.required", { field: t("s7.slot") }), trigger: "blur" }]
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
  area: "DB",
  dbNumber: "",
  start: "",
  dataType: 12, // 采集类型代码（TIA），默认 Real
  childAddress: "0",
  Value_Type: 12, // 输出类型代码（DB 列），默认 float32
  Conversion: "",
  RW_Cancel: 2
});

const pointRules = reactive<FormRules>({
  Tag: [{ required: true, message: () => t("common.required", { field: t("point.tag") }), trigger: "blur" }],
  area: [{ required: true, message: () => t("common.required", { field: t("s7.area") }), trigger: "change" }],
  dbNumber: [{ required: true, message: () => t("common.required", { field: t("s7.dbNumber") }), trigger: "blur" }],
  start: [{ required: true, message: () => t("common.required", { field: t("s7.start") }), trigger: "blur" }],
  dataType: [{ required: true, message: () => t("common.required", { field: t("s7.dataType") }), trigger: "change" }]
});

// 解析 S7 驱动配置字符串: IP;机架号;槽位号;重试间隔;轮询间隔;连接超时;响应超时;组包大小;连接类型
const parseDriveConfig = (configStr: string) => {
  const parts = configStr.split(";");
  return {
    ip: parts[0] || "",
    rack: parts[1] || "0",
    slot: parts[2] || "2",
    retryTimeout: parts[3] || "3s",
    delayBetweenPolls: parts[4] || "0",
    connectTimeout: parts[5] || "10s",
    responseTimeout: parts[6] || "10s",
    packetMax: parts[7] || "480",
    connectionType: parseInt(parts[8]) || 2
  };
};

// 序列化驱动配置
const serializeDriveConfig = () => {
  return `${driveForm.ip};${driveForm.rack};${driveForm.slot};${driveForm.retryTimeout};${driveForm.delayBetweenPolls};${driveForm.connectTimeout};${driveForm.responseTimeout};${driveForm.packetMax};${driveForm.connectionType}`;
};

// S7 地址正则（与后端 new.go Regex_S7_Point_Config 一致）
const S7_ADDR_RE = /^(?:(DB)(\d+)\.(DB[XBWDL])|([IQM])([XBWDL]?))(\d+)(?:\.(\d+))?$/i;

// 解析 S7 点位配置：主格式 "S7地址 [TIA类型名]"（如 "DB1.DBD0 Real"），兼容 JSON 与旧分号格式
const parsePointConfig = (configStr: string) => {
  const fallback = { area: "DB", dbNumber: "", start: "", dataType: 12, childAddress: "0" };
  if (!configStr) return fallback;
  const trimmed = configStr.trim();

  // 兼容旧 JSON 格式
  if (trimmed.startsWith("{")) {
    try {
      const obj = JSON.parse(trimmed);
      const areaCode = obj.Area ?? 0x84;
      return {
        area: areaMapReverse[areaCode] || "DB",
        dbNumber: String(obj.DBNumber ?? ""),
        start: String(obj.Start ?? ""),
        dataType: obj.Type ?? 12,
        childAddress: String(obj.Child_Address ?? "0")
      };
    } catch {
      return fallback;
    }
  }

  // 主格式：S7 地址字符串 + 可选 TIA 类型名后缀
  if (!trimmed.includes(";")) {
    let addr = trimmed;
    let typeName = "";
    const spIdx = trimmed.search(/\s/);
    if (spIdx >= 0) {
      addr = trimmed.slice(0, spIdx).trim();
      typeName = trimmed.slice(spIdx + 1).trim();
    }
    const m = addr.match(S7_ADDR_RE);
    if (m) {
      const isDB = m[1]?.toUpperCase() === "DB";
      const areaChar = isDB ? "DB" : m[4].toUpperCase();
      const dbNumber = isDB ? m[2] || "" : "";
      const suffix = (isDB ? (m[3] || "").slice(3) : m[5] || "").toUpperCase();
      const startStr = m[6] || "0";
      const childAddr = suffix === "X" && m[7] ? m[7] : "0";
      // Type 代码：优先显式 TIA 名（大小写不敏感），其次按后缀推断
      let dataType = typeName ? dataTypeMap[typeName.toLowerCase()] ?? 0 : 0;
      if (!dataType) {
        switch (suffix) {
          case "X": dataType = 1; break;
          case "B": dataType = 3; break;
          case "W": dataType = 5; break;
          case "D": dataType = 7; break;
          case "L": dataType = 9; break;
          default: dataType = 12;
        }
      }
      return { area: areaChar, dbNumber, start: startStr, dataType, childAddress: childAddr };
    }
  }

  // 旧分号格式（历史数据兼容）
  const parts = trimmed.split(";");
  const areaCode = parseInt(parts[0]) || 0x84;
  const typeCode = parseInt(parts[3]) || 12;
  return {
    area: areaMapReverse[areaCode] || "DB",
    dbNumber: parts[1] || "",
    start: parts[2] || "",
    dataType: typeCode,
    childAddress: parts[4] || "0"
  };
};

// 序列化点位配置为 "S7地址 TIA类型名" 字符串（与后端 Point_Config_Switch S7 地址分支对齐）
const serializePointConfig = () => {
  const dbNum = parseInt(pointForm.dbNumber) || 0;
  const start = parseInt(pointForm.start) || 0;
  const childAddr = parseInt(pointForm.childAddress) || 0;
  const typeMeta =
    S7_DATA_TYPE_OPTIONS.find(o => o.value === pointForm.dataType) ??
    S7_DATA_TYPE_OPTIONS.find(o => o.value === 12)!;
  const suffix = typeMeta.suffix;
  const tia = typeMeta.tia;

  let addr: string;
  if (pointForm.area === "DB") {
    addr = pointForm.dataType === 1
      ? `DB${dbNum}.DBX${start}.${childAddr}`
      : `DB${dbNum}.DB${suffix}${start}`;
  } else {
    // I/Q/M
    addr = pointForm.dataType === 1
      ? `${pointForm.area}${start}.${childAddr}`
      : `${pointForm.area}${suffix}${start}`;
  }
  return `${addr} ${tia}`;
};

// 加载驱动数据
const loadDriveData = async () => {
  try {
    const res = (await configApi.drive.search({
      name: "%%",
      types: ["Siemens_S7"],
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
      ...parsePointConfig(p.Config),
      valueTypeLabel: valueTypeMapReverse[p.Value_Type] || String(p.Value_Type ?? "")
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
  driveForm.rack = "0";
  driveForm.slot = "2";
  driveForm.retryTimeout = "3s";
  driveForm.delayBetweenPolls = "0";
  driveForm.connectTimeout = "10s";
  driveForm.responseTimeout = "10s";
  driveForm.packetMax = "480";
  driveForm.connectionType = 2;
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
  driveForm.rack = config.rack;
  driveForm.slot = config.slot;
  driveForm.retryTimeout = config.retryTimeout;
  driveForm.delayBetweenPolls = config.delayBetweenPolls;
  driveForm.connectTimeout = config.connectTimeout;
  driveForm.responseTimeout = config.responseTimeout;
  driveForm.packetMax = config.packetMax;
  driveForm.connectionType = config.connectionType;
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
      await configApi.drive.add([{ Type: "Siemens_S7", Name: driveForm.Name, Config: configStr }]);
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
  pointForm.area = "DB";
  pointForm.dbNumber = "";
  pointForm.start = "";
  pointForm.dataType = 12; // Real
  pointForm.childAddress = "0";
  pointForm.Value_Type = 12; // float32
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
  pointForm.area = config.area;
  pointForm.dbNumber = config.dbNumber;
  pointForm.start = config.start;
  pointForm.dataType = config.dataType;
  pointForm.childAddress = config.childAddress;
  pointForm.Value_Type = row.Value_Type ?? 12;
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
        RW_Cancel: pointForm.RW_Cancel,
        Value_Type: pointForm.Value_Type
      }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.point.add([{
        Drive_Id: pointForm.Drive_Id,
        Tag: pointForm.Tag,
        Description: pointForm.Description,
        Config: configStr,
        Conversion: pointForm.Conversion,
        RW_Cancel: pointForm.RW_Cancel,
        Value_Type: pointForm.Value_Type
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
