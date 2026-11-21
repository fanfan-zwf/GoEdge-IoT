<template>
  <div class="main point-layout">
    <!-- 左侧：点位树（懒加载；点击节点过滤右侧表格） -->
    <div class="tree-col">
      <el-card shadow="never" class="tree-card">
        <template #header>
          <div class="card-header">
            <span class="card-title">{{ t("tree.title") }}</span>
            <div class="header-actions">
              <el-button link @click="handleTreeRefresh">{{ t("tree.refresh") }}</el-button>
              <el-button link type="primary" :loading="rebuilding" @click="handleRebuild">
                {{ t("tree.rebuild") }}
              </el-button>
            </div>
          </div>
        </template>
        <el-scrollbar class="tree-box">
          <el-tree ref="treeRef" :key="treeKey" :load="loadNode" lazy node-key="id" :props="treeProps" highlight-current
            :expand-on-click-node="false" @node-click="handleTreeNodeClick">
            <template #default="{ data }">
              <span class="tree-node">
                <el-icon class="node-icon">
                  <Folder v-if="data.nodeType !== 2" />
                  <Document v-else />
                </el-icon>
                <span class="node-label">{{ data.name }}</span>
                <el-tag v-if="data.pointId" size="small" type="success" class="node-tag">
                  #{{ data.pointId }}
                </el-tag>
              </span>
            </template>
          </el-tree>
        </el-scrollbar>
      </el-card>
    </div>

    <!-- 右侧：点位表格 -->
    <div class="table-col">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">{{ t("point.title") }}<code v-if="tagCommonPrefix" class="tag-prefix-inline">{{ tagCommonPrefix }}</code></span>
          <div class="header-actions">
            <el-tag v-if="treeFilterPath" closable type="info" class="filter-tag" @close="clearTreeFilter">
              {{ treeFilterName || treeFilterPath }}
            </el-tag>
            <el-input v-model="searchTag" :placeholder="t('common.searchPlaceholder')" clearable
              :style="{ width: isMobile ? '100%' : '200px' }" class="search-input"
              @keyup.enter="handleSearch" />
            <el-button @click="handleSearch">{{ t("common.search") }}</el-button>
            <el-button @click="handleReset">{{ t("common.reset") }}</el-button>
            <el-button :type="showTrend ? 'primary' : 'default'" @click="showTrend = !showTrend">
              {{ t("point.toggleTrend") }}
            </el-button>
            <el-button type="primary" @click="openAdd">{{ t("common.add") }}</el-button>
            <el-button @click="handleExport">{{ t("common.export") }}</el-button>
            <el-button @click="importDialogVisible = true">{{ t("common.import") }}</el-button>
          </div>
        </div>
      </template>

      <el-table :data="tableData" border scrollbar-always-on :max-height="isMobile ? 'calc(100vh - 300px)' : 'calc(100vh - 360px)'"
        :row-style="showTrend ? { height: isMobile ? '140px' : '200px' } : {}" style="width: 100%"
        :default-sort="{ prop: 'Id', order: 'ascending' }">
        <!-- 左侧钉住：ID / 驱动 ID / 点位标识（手机端不钉，避免无滞动区） -->
        <el-table-column prop="Id" :label="t('point.id')" :width="isMobile ? 60 : 80" sortable
          :fixed="isMobile ? false : 'left'" :expandable="false" />
        <el-table-column v-if="!isMobile" prop="Drive_Id" :label="t('point.driveId')" width="100" sortable fixed="left"
          :expandable="false" />
        <el-table-column prop="Tag" :label="t('point.tag')" :width="isMobile ? 120 : 150" sortable
          :fixed="isMobile ? false : 'left'" :expandable="false">
          <template #default="{ row }">
            {{ stripTagPrefix(row.Tag) }}
          </template>
        </el-table-column>
        <!-- 中间可滞动区 -->
        <el-table-column v-if="!isMobile" prop="Description" :label="t('point.description')" width="150" />
        <el-table-column v-if="!showTrend && !isMobile" prop="Config" :label="t('point.config')" min-width="200" />
        <el-table-column v-if="!showTrend && !isMobile" prop="RW_Cancel" :label="t('point.rwCancel')" width="110">
          <template #default="{ row }">
            {{ rwCancelMap[row.RW_Cancel] || row.RW_Cancel }}
          </template>
        </el-table-column>
        <el-table-column v-if="!showTrend && !isMobile" prop="Value_Type" :label="t('point.valueType')" width="110">
          <template #default="{ row }">
            {{ valueTypeMap[row.Value_Type] || row.Value_Type }}
          </template>
        </el-table-column>
        <el-table-column v-if="!isMobile" prop="Creation_Time" :label="t('common.creationTime')" width="130" sortable>
          <template #default="{ row }">
            <div>{{ formatDate(row.Creation_Time) }}</div>
            <div>{{ formatTime(row.Creation_Time) }}</div>
          </template>
        </el-table-column>
        <el-table-column v-if="showTrend" :label="t('point.trend')" :min-width="isMobile ? 180 : 300">
          <template #default="{ row }">
            <div :ref="el => setSparklineRef(el, row.Id)" class="sparkline" />
          </template>
        </el-table-column>
        <!-- 右侧钉住：状态 / 当前值 / 操作 -->
        <el-table-column :label="t('point.status')" :width="isMobile ? 80 : 100" :fixed="isMobile ? false : 'right'"
          :expandable="false">
          <template #default="{ row }">
            <span v-if="pointData[row.Id]?.msg"
              :class="['status-msg', pointData[row.Id].msg === 'ok' ? 'status-ok' : 'status-err']">
              {{ pointData[row.Id].msg }}
            </span>
            <span v-else class="status-msg">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('point.currentValue')" :width="isMobile ? 110 : 150"
          :fixed="isMobile ? false : 'right'" :expandable="false">
          <template #default="{ row }">
            <span class="current-value" :class="{ 'has-value': pointData[row.Id]?.value !== undefined }">
              {{ pointData[row.Id]?.value !== undefined ? pointData[row.Id].value : "-" }}
            </span>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.operation')" :width="isMobile ? 120 : 250" :fixed="isMobile ? false : 'right'"
          :expandable="false">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">{{ t("common.edit") }}</el-button>
            <el-button v-if="!isMobile" link type="primary" @click="openCopy(row)">{{ t("common.copy") }}</el-button>
            <el-button v-if="!isMobile" link type="warning" :disabled="row.RW_Cancel !== 3 && row.RW_Cancel !== 4"
              @click="openWrite(row)">
              {{ t("point.write") }}
            </el-button>
            <el-button link type="danger" @click="handleDel(row)">{{ t("common.del") }}</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="!treeFilterPath" class="pagination-wrapper">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 30, 50, 100, 150]"
          :layout="isMobile ? 'prev, pager, next' : 'total, sizes, prev, pager, next, jumper'"
          :pager-count="isMobile ? 5 : 7" @size-change="handleSizeChange" @current-change="handleCurrentChange" />
      </div>
    </el-card>
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('point.editTitle') : t('point.addTitle')" width="500px"
      destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px" @submit.prevent>
        <el-form-item :label="t('point.driveId')" prop="Drive_Id">
          <el-select v-model="form.Drive_Id" :placeholder="t('point.driveIdPlaceholder')" filterable
            style="width: 100%" @change="handleDriveChange">
            <el-option v-for="d in driveList" :key="d.Id" :label="`[${d.Id}] ${d.Name || d.Type}`" :value="d.Id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('point.tag')" prop="Tag">
          <el-input v-model="form.Tag" />
        </el-form-item>
        <!-- 通用字段：换算表达式、读写方式、数据类型 -->
        <el-form-item prop="Conversion">
          <template #label>
            <span class="label-with-help">
              <span>{{ t("point.conversion") }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a href="https://expr-lang.org/docs/language-definition" target="_blank" class="help-icon">?</a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Conversion" :placeholder="t('point.conversionPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('point.rwCancel')" prop="RW_Cancel">
          <el-select v-model="form.RW_Cancel" style="width: 100%">
            <el-option v-for="item in rwCancelOptions" :key="item.value" :label="t(item.labelKey)"
              :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('point.valueType')" prop="Value_Type">
          <el-select v-model="form.Value_Type" style="width: 100%">
            <el-option v-for="item in valueTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
        <!-- Config 输入框始终显示 -->
        <el-form-item prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t("common.config") }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a href="https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/IO/Modbus_Tcp/README.md"
                  target="_blank" class="help-icon">
                  ?
                </a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Config" type="textarea" :rows="2" :placeholder="t('common.pointConfigPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('point.description')">
          <!-- data-enter-newline：标记该框保留回车换行，不参与全局「回车=确认」 -->
          <el-input v-model="form.Description" type="textarea" :rows="3" data-enter-newline />
        </el-form-item>

        <!-- Siemens S7 驱动点位参数（可视化辅助，字段与 Config 双向同步） -->
        <SiemensPointForm v-if="isSiemensS7" v-model="s7PointForm" v-model:config="form.Config" />
        <!-- Modbus TCP 驱动点位参数（可视化辅助，字段与 Config 双向同步） -->
        <ModbusPointForm v-if="isModbusTcp" v-model="modbusPointForm" v-model:config="form.Config" />
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t("common.confirm") }}</el-button>
      </template>
    </el-dialog>

    <!-- 写入值对话框 -->
    <el-dialog v-model="writeDialogVisible" :title="t('point.writeTitle')" width="420px" destroy-on-close>
      <el-form label-width="100px" @submit.prevent>
        <el-form-item :label="t('point.writePointName')">
          <span>{{ writePoint.Tag || `ID: ${writePoint.Id}` }} (ID: {{ writePoint.Id }})</span>
        </el-form-item>
        <el-form-item :label="t('point.writeType')">
          <span>{{ valueTypeMap[writePoint.Value_Type] || writePoint.Value_Type }}</span>
        </el-form-item>
        <el-form-item :label="t('point.writeCurrentValue')">
          <span class="current-value has-value">{{ pointData[writePoint.Id]?.value ?? "-" }}</span>
        </el-form-item>
        <el-form-item :label="t('point.writeValueLabel')">
          <el-switch v-if="writePoint.Value_Type === 1" v-model="writeValue" :active-text="t('point.writeBoolTrue')"
            :inactive-text="t('point.writeBoolFalse')" />
          <el-input v-else :model-value="String(writeValue)" :placeholder="t('point.writeValuePlaceholder')"
            style="width: 200px" @update:model-value="writeValue = $event" @keyup.enter="handleWrite" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="writeDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="writeLoading" @click="handleWrite">{{ t("point.write") }}</el-button>
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
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick, watch } from "vue";
import { useI18n } from "vue-i18n";
import { message } from "@/utils/message";
import { ElMessageBox, type FormInstance, type FormRules } from "element-plus";
import { Folder, Document } from "@element-plus/icons-vue";
import * as echarts from "echarts";
import { configApi, exportConfig, importConfig, getBaseServer } from "@/api/config";
import {
  CONFIG_MAX_LENGTH,
  ISSUE_MESSAGE_KEY,
  checkConversion,
  checkPointConfig,
  checkTag,
  type FormatIssue
} from "@/utils/configFormat";
import { getToken } from "@/utils/auth";
import SiemensPointForm from "@/views/ioDrive/siemens/point_form.vue";
import ModbusPointForm from "@/views/ioDrive/modbus/point_form.vue";
import type { SiemensPointConfig, ModbusPointConfig } from "@/views/ioDrive/types";
import { SIEMENS_POINT_DEFAULTS, MODBUS_POINT_DEFAULTS } from "@/views/ioDrive/types";

const { t } = useI18n();

const rwCancelOptions = [
  { value: 1, labelKey: "point.rwN" },
  { value: 2, labelKey: "point.rwR" },
  { value: 3, labelKey: "point.rwW" },
  { value: 4, labelKey: "point.rwRW" }
];
const rwCancelMap = computed(() => Object.fromEntries(rwCancelOptions.map(o => [o.value, t(o.labelKey)])));

const valueTypeOptions = [
  { value: 1, label: "bool" },
  { value: 2, label: "int8" },
  { value: 3, label: "uint8" },
  { value: 4, label: "int16" },
  { value: 5, label: "uint16" },
  { value: 6, label: "int32" },
  { value: 7, label: "uint32" },
  { value: 8, label: "int64" },
  { value: 9, label: "uint64" },
  { value: 10, label: "int" },
  { value: 11, label: "uint" },
  { value: 12, label: "float32" },
  { value: 13, label: "float64" },
  { value: 14, label: "float" },
  { value: 15, label: "string" }
];
const valueTypeMap: Record<number, string> = Object.fromEntries(valueTypeOptions.map(o => [o.value, o.label]));

const driveList = ref<any[]>([]);
const loadDriveList = async () => {
  try {
    const res = (await configApi.drive.query({ page: 1, pageSize: 1000 })) as any;
    driveList.value = res.Data || [];
  } catch {
    /* ignore */
  }
};

const currentDriveType = computed(() => {
  const drive = driveList.value.find(d => d.Id === form.Drive_Id);
  return drive?.Type || "";
});
const tableData = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);

/** 计算当前表格所有 Tag 的公共前缀（按 "/" 分段） */
const tagCommonPrefix = computed(() => {
  const tags = tableData.value.map(r => r.Tag).filter(Boolean) as string[];
  if (tags.length < 2) return "";
  // 按 "/" 分割，保留空串（因为 Tag 可能以 / 开头）
  const parts = tags.map(t => t.split("/"));
  const first = parts[0];
  let prefixLen = 0;
  // 至少留最后一段作为独特部分
  for (let i = 0; i < first.length - 1; i++) {
    if (parts.every(p => p.length > i && p[i] === first[i])) {
      prefixLen = i + 1;
    } else {
      break;
    }
  }
  if (prefixLen === 0) return "";
  const prefix = first.slice(0, prefixLen).join("/");
  // 确保以 / 结尾显示
  return prefix.endsWith("/") ? prefix : prefix + "/";
});

/** 去掉公共前缀后的短 Tag */
const stripTagPrefix = (tag: string) => {
  if (!tagCommonPrefix.value) return tag;
  const raw = tagCommonPrefix.value.endsWith("/")
    ? tagCommonPrefix.value.slice(0, -1)
    : tagCommonPrefix.value;
  return tag.startsWith(raw)
    ? tag.slice(raw.length).replace(/^\//, "")
    : tag;
};

const searchTag = ref("%%");
const isSearching = ref(false);
// 移动端适配：宽度 < 768px 时启用精简列与堆叠头
const isMobile = ref(false);

const pointData = ref<
  Record<
    number,
    {
      value: any;
      msg: string;
      time: string;
      timeDelay: { text: string; cls: string };
    }
  >
>({});
const ensurePointData = (id: number) => {
  if (!pointData.value[id]) {
    pointData.value[id] = {
      value: undefined,
      msg: "",
      time: "",
      timeDelay: { text: "", cls: "" }
    };
  }
  return pointData.value[id];
};
const calcDelay = (delayMs: number): { text: string; cls: string } => {
  const ms = delayMs % 1000;
  const sec = Math.floor(delayMs / 1000);
  let text: string;
  let cls: string;
  if (delayMs < 1000) {
    text = `${delayMs}ms`;
    cls = "time-delay-green";
  } else if (sec < 60) {
    text = `${sec}s${ms}ms`;
    cls = delayMs < 3000 ? "time-delay-yellow" : "time-delay-red";
  } else {
    const min = Math.floor(sec / 60);
    text = `${min}m${sec % 60}s`;
    cls = "time-delay-red";
  }
  return { text, cls };
};
let ws: WebSocket | null = null;
const wsUrl = () => `ws://${getBaseServer()}/api/v1.0/point/ws`;
// 断线重连：指数退避 5s → 10s → 20s → 封顶 30s，连上后复位
const WS_RETRY_BASE = 5000;
const WS_RETRY_MAX = 30000;
let wsRetryTimer: ReturnType<typeof setTimeout> | null = null;
let wsRetryDelay = WS_RETRY_BASE;
let wsManualClose = false; // 主动断开（页面卸载）不触发重连
let wsWarned = false; // 同一次断线只弹一次提示，避免退避重连刷屏

const showTrend = ref(false);
// 固定 4000 格数组：每 10ms 推进一次，总覆盖 40s 窗口；X 轴标签仍显示 0s/5s/.../40s
// last 保存该点位最后一次拿到的有效数值：后端在读取失败/换算失败时会把 Value 置空推过来，
// 曲线不能因此断成空白，要用 last 沿用上一有效值继续右延
const chartSeriesData: Record<number, { values: (number | null)[]; last: number | null }> = {};
const TREND_TICK_MS = 10;
const TREND_WINDOW_MS = 40000;
const TREND_SAMPLES = TREND_WINDOW_MS / TREND_TICK_MS; // 4000

/** 点位原始值 → 曲线可用的数值；null/undefined/空串/非数值字符串（如文本点位）都算「无值」 */
const toTrendNumber = (raw: any): number | null => {
  if (raw === undefined || raw === null || raw === "") return null;
  if (typeof raw === "boolean") return raw ? 1 : 0;
  const n = typeof raw === "number" ? raw : Number(raw);
  return Number.isNaN(n) ? null : n;
};

const sparklineInstances = new Map<number, echarts.ECharts>();
const sparklineRefs = new Map<number, HTMLElement>();

const setSparklineRef = (el: any, pointId: number) => {
  if (el) {
    sparklineRefs.set(pointId, el);
    initSparkline(pointId);
    const sd = chartSeriesData[pointId];
    if (sd && sd.values.some(v => v !== null)) {
      updateSparkline(pointId);
    }
  } else {
    sparklineRefs.delete(pointId);
    const inst = sparklineInstances.get(pointId);
    if (inst) {
      inst.dispose();
      sparklineInstances.delete(pointId);
    }
  }
};

const initSparkline = (pid: number) => {
  const el = sparklineRefs.get(pid);
  if (!el) return;
  let inst = sparklineInstances.get(pid);
  if (inst) inst.dispose();
  inst = echarts.init(el);
  sparklineInstances.set(pid, inst);
  const nowMs = Date.now();
  inst.setOption({
    animation: false,
    grid: { left: 30, right: 15, top: 10, bottom: 25 },
    xAxis: {
      type: "value",
      show: true,
      min: nowMs - TREND_WINDOW_MS,
      max: nowMs,
      interval: 5000,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        fontSize: 10,
        formatter: (val: number) => `${Math.round((nowMs - val) / 1000)}s`
      },
      splitLine: { show: false }
    },
    yAxis: {
      type: "value",
      show: true,
      splitNumber: 2,
      min: -1,
      max: 1,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        fontSize: 10,
        formatter: (val: number) => Math.round(val)
      },
      splitLine: { show: false }
    },
    series: [
      {
        type: "line",
        data: [],
        step: "end",
        showSymbol: false,
        lineStyle: { width: 1.5, color: "#5470c6" }
      }
    ]
  });
};

const updateAllSparklines = () => {
  for (const pid of sparklineInstances.keys()) {
    updateSparkline(pid);
  }
};

const updateSparkline = (pid: number) => {
  const inst = sparklineInstances.get(pid);
  if (!inst) return;
  const sd = chartSeriesData[pid];
  const nowMs = Date.now();
  if (!sd || !sd.values.some(v => v !== null)) {
    // 无数据时也刷 X 轴，让窗口随时间前移
    inst.setOption({
      xAxis: {
        min: nowMs - TREND_WINDOW_MS,
        max: nowMs,
        axisLabel: { formatter: (val: number) => `${Math.round((nowMs - val) / 1000)}s` }
      }
    });
    return;
  }
  const len = sd.values.length;
  const seriesData: [number, number][] = [];
  let yMin = Infinity;
  let yMax = -Infinity;
  for (let i = 0; i < len; i++) {
    const v = sd.values[i];
    if (v === null) continue;
    const x = nowMs - (len - 1 - i) * TREND_TICK_MS;
    seriesData.push([x, v]);
    if (v < yMin) yMin = v;
    if (v > yMax) yMax = v;
  }
  let axisMin: number;
  let axisMax: number;
  if (yMax > yMin) {
    const pad = (yMax - yMin) * 0.1;
    axisMin = yMin - pad;
    axisMax = yMax + pad;
  } else {
    // 全程只有一个值（初始值铺满整窗、或值一直没变）：yMax === yMin，
    // 此时沿用默认的 -1~1 会把 6010、43 这类值画到坐标范围外，看着就像“没有曲线”
    // （初值为 0 时刚好落在 -1~1 里所以能看见，就是这个 bug 的伪装面）。
    // 改成以该值为中心开一个窗口：小值固定 ±1，大值按 10% 取幅。
    const span = Math.max(Math.abs(yMin) * 0.1, 1);
    axisMin = yMin - span;
    axisMax = yMin + span;
  }
  inst.setOption({
    xAxis: {
      min: nowMs - TREND_WINDOW_MS,
      max: nowMs,
      axisLabel: { formatter: (val: number) => `${Math.round((nowMs - val) / 1000)}s` }
    },
    yAxis: { min: axisMin, max: axisMax },
    series: [{ data: seriesData }]
  });
};

/** 每 TREND_TICK_MS 推进一次：直接遍历 tableData，从 pointData 读当前值推入曲线数组 */
let trendTicker: ReturnType<typeof setInterval> | null = null;
const tickTrend = () => {
  for (const row of tableData.value) {
    const pid = row.Id;
    if (!pid) continue;
    // 从 pointData 读实时值（hydrateRealtime 给初始值，WS 持续更新）
    const val = toTrendNumber(pointData.value[pid]?.value);
    let sd = chartSeriesData[pid];
    if (!sd) {
      // 新出现的行（翻页、切树节点等）用当前初始值填满整窗
      const initVal = val ?? null;
      sd = {
        values: new Array<number | null>(TREND_SAMPLES).fill(initVal),
        last: initVal
      };
      chartSeriesData[pid] = sd;
    } else {
      // 建数组时还没拿到值（整窗 null）、之后才拿到首个有效值：
      // 把开头的空位补成该值，曲线立刻铺满整窗，而不是从右往左慢慢长
      if (val !== null && sd.last === null) {
        for (let i = 0; i < sd.values.length; i++) {
          if (sd.values[i] === null) sd.values[i] = val;
        }
      }
      // 本轮无值时沿用上一有效值，曲线保持水平右延；从未取到过值才真的留空
      sd.values.push(val ?? sd.last);
      if (sd.values.length > TREND_SAMPLES) sd.values.shift();
    }
    if (val !== null) sd.last = val;
    // 只有 sparkline 实例已渲染时才重绘
    if (sparklineInstances.has(pid)) {
      updateSparkline(pid);
    }
  }
};
const startTrendTicker = () => {
  stopTrendTicker();
  trendTicker = setInterval(tickTrend, TREND_TICK_MS);
};
const stopTrendTicker = () => {
  if (trendTicker) {
    clearInterval(trendTicker);
    trendTicker = null;
  }
};

// tickTrend 每 50ms 直接从 pointData 读取最新值，不再需要 watch(pointData) 推送

/** 关闭当前连接并清掉待重连定时器（纯收尾，不管是否主动断开） */
const closeSocket = () => {
  if (wsRetryTimer) {
    clearTimeout(wsRetryTimer);
    wsRetryTimer = null;
  }
  if (ws) {
    ws.onopen = null;
    ws.onmessage = null;
    ws.onclose = null;
    ws.onerror = null;
    try {
      ws.close();
    } catch {
      /* 已关闭，忽略 */
    }
    ws = null;
  }
};

/** 异常断开（含握手被 401 拒掉）：提示一次 + 安排退避重连，重连时会重新取最新令牌 */
const handleWsClosed = () => {
  if (wsManualClose) return;
  if (!wsWarned) {
    wsWarned = true;
    message(t("point.wsDisconnected"), { type: "warning" });
  }
  if (wsRetryTimer) return;
  wsRetryTimer = setTimeout(() => {
    wsRetryTimer = null;
    connectWebSocket();
  }, wsRetryDelay);
  wsRetryDelay = Math.min(wsRetryDelay * 2, WS_RETRY_MAX);
};

const connectWebSocket = () => {
  wsManualClose = false;
  closeSocket();
  const pointIds = tableData.value.map(r => r.Id).filter(id => id > 0);
  if (pointIds.length === 0) return;
  const token = getToken()?.accessToken ?? "";
  // 无令牌（未登录/已登出）不建连，页面随后会被路由守卫带回登录页
  if (!token) return;
  const url = `${wsUrl()}?points=${JSON.stringify(pointIds)}`;
  // 浏览器 WebSocket 握手无法自定义 Authorization 头，令牌借子协议位传递（不进 URL，避免被访问日志明文记录）
  try {
    ws = new WebSocket(url, [token]);
  } catch {
    // 令牌不是合法子协议 token 时构造会抛错，统一走重连处理
    ws = null;
    handleWsClosed();
    return;
  }
  ws.onopen = () => {
    wsRetryDelay = WS_RETRY_BASE;
    if (wsWarned) message(t("point.wsRestored"), { type: "success" });
    wsWarned = false;
  };
  ws.onmessage = event => {
    try {
      const msg = JSON.parse(event.data);
      const items = Array.isArray(msg) ? msg : [msg];
      for (const data of items) {
        if (data.PointId === undefined) continue;
        const pd = ensurePointData(data.PointId);
        // 采集失败/换算失败时后端只带 Msg 而不带 Value（Value 为 null），
        // 这种情况要保留上一次的有效值，否则已显示的当前值会被推成空白
        if (data.Value !== undefined && data.Value !== null) pd.value = data.Value;
        if (data.Msg !== undefined) {
          pd.msg = data.Msg;
        }
        pd.time = data.Time || "";
        const ts = data.Time ? new Date(data.Time).getTime() : Date.now();
        pd.timeDelay = calcDelay(Date.now() - ts);
      }
      updateAllSparklines();
    } catch {
      /* ignore */
    }
  };
  ws.onclose = () => {
    ws = null;
    handleWsClosed();
  };
  ws.onerror = () => {
    // 仅关闭，后续统一由 onclose 处理提示与重连，避免两边重复触发
    ws?.close();
  };
};

/** 主动断开：页面卸载时调用，不再重连 */
const disconnectWebSocket = () => {
  wsManualClose = true;
  closeSocket();
};

const dialogVisible = ref(false);
const isEdit = ref(false);
const submitting = ref(false);
const formRef = ref<FormInstance>();
const form = reactive({
  Id: 0,
  Drive_Id: 1,
  Tag: "",
  Description: "",
  Config: "",
  Conversion: "",
  RW_Cancel: 0,
  Value_Type: 0
});

// S7 点位配置表单
const s7PointForm = reactive<SiemensPointConfig>({ ...SIEMENS_POINT_DEFAULTS });

// Modbus 点位配置表单
const modbusPointForm = reactive<ModbusPointConfig>({ ...MODBUS_POINT_DEFAULTS });

// 判断当前驱动是否需要可视化参数表单
const isSiemensS7 = computed(() => currentDriveType.value === "Siemens_S7");
const isModbusTcp = computed(() => currentDriveType.value === "Modbus_Tcp");

// 重置可视化表单，避免上一条记录（或另一种驱动）的字段残留
const resetPointForms = () => {
  Object.assign(s7PointForm, SIEMENS_POINT_DEFAULTS);
  Object.assign(modbusPointForm, MODBUS_POINT_DEFAULTS);
};

// 用户在对话框内切换驱动：旧 Config 对新驱动没有意义，清空后由子组件按默认值重新生成
const handleDriveChange = () => {
  resetPointForms();
  form.Config = "";
};

// 地址字符串 ⇄ 结构化字段的解析与回写均由子组件通过 v-model:config 完成，父组件只维护 form.Config

// formatValidator 把 configFormat 的格式校验包成 Element Plus 规则。
// 与后端 internal/repo/mysql/business.go 同源：前端先拦一道，免得提交后才收到 417；
// 空值不在这里报，交给同一字段上的 required 规则，保证提示仍是统一的「不能为空」
const formatValidator =
  (check: (value: string) => FormatIssue) => (_rule: any, value: string, callback: (error?: Error) => void) => {
    const issue = check(value ?? "");
    callback(issue ? new Error(t(ISSUE_MESSAGE_KEY[issue], { max: CONFIG_MAX_LENGTH })) : undefined);
  };

const rules = reactive<FormRules>({
  Drive_Id: [
    {
      required: true,
      message: () => t("common.required", { field: t("point.driveId") }),
      trigger: "blur"
    }
  ],
  Tag: [
    {
      required: true,
      message: () => t("common.required", { field: t("point.tag") }),
      trigger: "blur"
    },
    { validator: formatValidator(checkTag), trigger: "blur" }
  ],
  Config: [
    {
      required: true,
      message: () => t("common.required", { field: t("common.config") }),
      trigger: "blur"
    },
    { validator: formatValidator(checkPointConfig), trigger: "blur" }
  ],
  Conversion: [{ validator: formatValidator(checkConversion), trigger: "blur" }],
  RW_Cancel: [
    {
      required: true,
      message: () => t("common.required", { field: t("point.rwCancel") }),
      trigger: "blur"
    }
  ],
  Value_Type: [
    {
      required: true,
      message: () => t("common.required", { field: t("point.valueType") }),
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

// ==================== 左侧点位树：懒加载 + 节点过滤联动 ====================
interface TreeNode {
  id: number;
  host: string;
  parentId: number;
  name: string;
  path: string;
  level: number;
  nodeType: number; // 1=文件夹 2=点位 3=既是点位又是文件夹
  pointId: number; // 0=非点位
  hasChildren: number; // 1=可展开
}

const treeRef = ref();
const treeKey = ref(0); // 变更以强制重挂载、触发根节点重新懒加载
const rebuilding = ref(false);
const treeFilterPath = ref(""); // 非空=按该树节点过滤右侧表格
const treeFilterName = ref("");

// isLeaf：后端 hasChildren=1 才可展开；否则视为叶子
const treeProps = {
  label: "name",
  isLeaf: (data: TreeNode) => data.hasChildren !== 1
};

// el-tree 懒加载回调：根层传空 tag_path，其余传当前节点 path；文件夹/文件两列表按 id 去重合并
const loadNode = async (node: any, resolve: (data: TreeNode[]) => void) => {
  const tagPath = node.level === 0 ? "" : node.data.path;
  try {
    const res: any = await configApi.tagTree.children({ tag_path: tagPath });
    const data = res?.Data || {};
    const folders: TreeNode[] = data["子路径"] || [];
    const files: TreeNode[] = data["文件"] || [];
    const merged = new Map<number, TreeNode>();
    for (const n of [...folders, ...files]) merged.set(n.id, n);
    resolve([...merged.values()]);
  } catch {
    resolve([]);
  }
};

// 清除树过滤：恢复常规分页查询
const clearTreeFilter = () => {
  treeFilterPath.value = "";
  treeFilterName.value = "";
  treeRef.value?.setCurrentKey?.(null);
  isSearching.value = false;
  page.value = 1;
  loadData();
};

// 点击节点 -> 过滤右侧表格为该节点下的点位；再次点击同一节点取消过滤
const handleTreeNodeClick = (data: TreeNode) => {
  if (treeFilterPath.value === data.path) {
    clearTreeFilter();
    return;
  }
  treeFilterPath.value = data.path;
  treeFilterName.value = data.name;
  isSearching.value = false;
  page.value = 1;
  loadData();
};

const handleTreeRefresh = () => {
  treeKey.value += 1;
};

// 从 Point_Config 全量重建树（点位增删改后调用一次）
const handleRebuild = async () => {
  try {
    await ElMessageBox.confirm(t("tree.rebuildTip"), t("common.warning"), { type: "warning" });
  } catch {
    return;
  }
  rebuilding.value = true;
  try {
    await configApi.tagTree.rebuild();
    message(t("tree.rebuildSuccess"), { type: "success" });
    treeKey.value += 1;
    loadData();
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    rebuilding.value = false;
  }
};

// 拉取当前表内点位的初始值并建立实时订阅（分页 / 树过滤模式共用）
const hydrateRealtime = async () => {
  const pointIds = tableData.value.map(r => r.Id).filter(id => id > 0);
  try {
    const currentValueRes: any = await configApi.point.readValue(pointIds);
    if (currentValueRes && currentValueRes.Code === 200) {
      const values = Array.isArray(currentValueRes.Data) ? currentValueRes.Data : [];
      for (const item of values) {
        const pointId = item.PointId;
        const pd = ensurePointData(pointId);
        // 同上：后端内存里没有有效值的点位不覆盖已有值
        if (item.Value !== undefined && item.Value !== null) pd.value = item.Value;
        if (item.Msg !== undefined) {
          pd.msg = item.Msg;
        }
        pd.time = item.Time || "";
        const ts = item.Time ? new Date(item.Time).getTime() : Date.now();
        if (ts > 0) {
          pd.timeDelay = calcDelay(Date.now() - ts);
        }
      }
      if (currentValueRes.Timestamp) {
        const serverTime = new Date(currentValueRes.Timestamp).getTime();
        if (serverTime > 0) {
          const { text, cls } = calcDelay(Date.now() - serverTime);
          for (const item of values) {
            ensurePointData(item.PointId).timeDelay = {
              text,
              cls
            };
          }
        }
      }
    }
  } catch (err) {
    console.error("获取初始值失败:", err);
  }
  connectWebSocket();
  if (showTrend.value) {
    nextTick(() => {
      updateAllSparklines();
    });
  }
};

const loadData = async () => {
  try {
    // 树节点过滤模式：取该节点子树下的全部点位（不分页）
    if (treeFilterPath.value) {
      const res = (await configApi.tagTree.points({
        tag_path: treeFilterPath.value
      })) as any;
      tableData.value = res.Data || [];
      total.value = tableData.value.length;
      await hydrateRealtime();
      return;
    }
    if (isSearching.value) {
      const res = (await configApi.point.search({
        tag: searchTag.value,
        quantity: 30
      })) as any;
      tableData.value = res.Data || [];
      total.value = tableData.value.length;
      // 搜索结果同样要拉初始值 + 重连 WS 订阅，否则新点位既无当前值也无曲线
      await hydrateRealtime();
      return;
    }
    const countRes = (await configApi.point.count({
      page: 0,
      pageSize: 0
    })) as any;
    total.value = countRes.Data || 0;
    if (total.value === 0) {
      tableData.value = [];
      return;
    }
    const res = (await configApi.point.query({
      page: page.value,
      pageSize: pageSize.value
    })) as any;
    tableData.value = res.Data || [];
    await hydrateRealtime();
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
  searchTag.value = "%%";
  isSearching.value = false;
  treeFilterPath.value = "";
  treeFilterName.value = "";
  treeRef.value?.setCurrentKey?.(null);
  loadData();
};

const resetForm = () => {
  form.Id = 0;
  form.Drive_Id = 1;
  form.Tag = "";
  form.Description = "";
  form.Config = "";
  form.Conversion = "";
  form.RW_Cancel = 0;
  form.Value_Type = 0;
  resetPointForms();
};

const openAdd = () => {
  isEdit.value = false;
  resetForm();
  loadDriveList();
  dialogVisible.value = true;
};

const openEdit = (row: any) => {
  isEdit.value = true;
  loadDriveList();
  form.Id = row.Id;
  form.Drive_Id = row.Drive_Id;
  form.Tag = row.Tag;
  form.Description = row.Description || "";
  form.Config = row.Config;
  form.Conversion = row.Conversion || "";
  form.RW_Cancel = row.RW_Cancel;
  form.Value_Type = row.Value_Type;
  resetPointForms();
  // 子组件挂载时会把 Config 解析回结构化字段，这里只需原样传入
  dialogVisible.value = true;
};

// 复制：源点位的标签、驱动、配置、换算、读写方式、数据类型全部带入新增对话框，只改末级标识符即可
// 注意 Tag 在库里是全库唯一，直接原样提交会报重复，需改掉最后一段
const openCopy = (row: any) => {
  isEdit.value = false;
  loadDriveList();
  form.Id = 0;
  form.Drive_Id = row.Drive_Id;
  form.Tag = row.Tag || "";
  form.Description = row.Description || "";
  form.Config = row.Config;
  form.Conversion = row.Conversion || "";
  form.RW_Cancel = row.RW_Cancel;
  form.Value_Type = row.Value_Type;
  resetPointForms();
  dialogVisible.value = true;
};

const handleSubmit = async () => {
  if (submitting.value) return;
  if (!formRef.value) return;
  try {
    await formRef.value.validate();
  } catch (fields: any) {
    // 回车提交时焦点不在出错字段上，内联红字容易被忽略，再弹一条气泡提示
    const first: any = Object.values(fields ?? {}).flat()[0];
    if (first?.message) message(String(first.message), { type: "warning" });
    return;
  }
  submitting.value = true;
  try {
    // 直接使用 form.Config（已由子组件与可视化字段实时双向同步）
    const data = {
      Drive_Id: form.Drive_Id,
      Tag: form.Tag,
      Description: form.Description,
      Config: form.Config,
      Conversion: form.Conversion,
      RW_Cancel: form.RW_Cancel,
      Value_Type: form.Value_Type
    };
    if (isEdit.value) {
      await configApi.point.update([{ Id: form.Id, ...data }]);
      message(t("common.editSuccess"), { type: "success" });
    } else {
      await configApi.point.add([data]);
      message(t("common.addSuccess"), { type: "success" });
    }
    dialogVisible.value = false;
    loadData();
    treeKey.value += 1; // 后端已随点位变更重建树，刷新左树展示最新层级
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    submitting.value = false;
  }
};

// 对话框打开期间全局接管回车：刚点「复制/编辑」时焦点还在表格按钮上（弹框并不抢焦点），
// 只给输入框绑 @keyup.enter 会出现「按回车没反应」，所以统一在 window 的 keydown 阶段处理
const onDialogKeydown = (e: KeyboardEvent) => {
  if (!dialogVisible.value) return;
  if (e.key !== "Enter" || e.isComposing) return; // isComposing：中文输入法上屏的回车不算提交
  if (e.shiftKey || e.ctrlKey || e.altKey || e.metaKey) return;
  const el = e.target as HTMLElement | null;
  if (el?.closest(".el-select, [data-enter-newline]")) return; // 下拉选中选项，描述框插入换行
  if (el?.tagName === "BUTTON" && el.closest(".el-dialog")) return; // 焦点在弹内按钮上，交给按钮自身的 click
  e.preventDefault(); // 同时阻止表单外那个「复制」按钮被回车重复激活
  handleSubmit();
};

const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(t("common.deleteConfirm", { name: `ID: ${row.Id}` }), t("common.warning"), {
      type: "warning"
    });
    await configApi.point.del([row.Id]);
    message(t("common.deleteSuccess"), { type: "success" });
    loadData();
    treeKey.value += 1; // 删除后刷新左树（后端已重建树）
  } catch (error: any) {
    if (error !== "cancel") {
      /* 拦截器已提示错误 */
    }
  }
};

const handleExport = () => {
  exportConfig("point");
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
    const point = data.point_config || {
      added: 0,
      updated: 0,
      deleted: 0
    };
    message(
      t("common.importSuccess", {
        added: point.added,
        updated: point.updated,
        deleted: point.deleted
      }),
      { type: "success" }
    );
    importDialogVisible.value = false;
    loadData();
    treeKey.value += 1; // 导入后刷新左树（后端已重建树）
  } catch (e: any) {
    message(e.message || "Import failed", { type: "error" });
  } finally {
    importLoading.value = false;
  }
};

const valueTypeNameMap: Record<number, string> = {
  1: "bool",
  2: "int8",
  3: "uint8",
  4: "int16",
  5: "uint16",
  6: "int32",
  7: "uint32",
  8: "int64",
  9: "uint64",
  10: "int",
  11: "uint",
  12: "float32",
  13: "float64",
  14: "float",
  15: "string"
};

const writeDialogVisible = ref(false);
const writeLoading = ref(false);
const writeValue = ref<string | boolean>("");
const writePoint = reactive({ Id: 0, Tag: "", Value_Type: 0 });

const openWrite = (row: any) => {
  writePoint.Id = row.Id;
  writePoint.Tag = row.Tag;
  writePoint.Value_Type = row.Value_Type;
  writeValue.value = row.Value_Type === 1 ? !pointData.value[row.Id]?.value : "";
  writeDialogVisible.value = true;
};

const handleWrite = async () => {
  if (writeLoading.value) return;
  const isBool = writePoint.Value_Type === 1;
  if (!isBool && writeValue.value === "") {
    message(t("point.writeValueRequired"), { type: "warning" });
    return;
  }
  writeLoading.value = true;
  try {
    const typeName = valueTypeNameMap[writePoint.Value_Type] || "float64";
    let parsedValue: any = writeValue.value;
    if (typeName !== "string" && typeName !== "bool") {
      const num = Number(writeValue.value);
      if (!isNaN(num)) parsedValue = num;
    } else if (typeName === "bool") {
      parsedValue = !!writeValue.value;
    }
    await configApi.point.writeValue([
      {
        PointId: writePoint.Id,
        Value: parsedValue,
        Type: typeName
      }
    ]);
    message(t("point.writeSuccess"), { type: "success" });
    writeDialogVisible.value = false;
  } catch {
    /* 拦截器已提示错误 */
  } finally {
    writeLoading.value = false;
  }
};

const handleResize = () => {
  isMobile.value = window.innerWidth < 768;
  for (const inst of sparklineInstances.values()) {
    inst.resize();
  }
};

watch(showTrend, val => {
  if (val) {
    // 初始化各 sparkline 的固定数组：直接遍历 tableData，不依赖 sparkline 是否已渲染完
    startTrendTicker();
    nextTick(() => {
      updateAllSparklines();
    });
  } else {
    stopTrendTicker();
  }
});

onMounted(() => {
  loadData();
  isMobile.value = window.innerWidth < 768;
  window.addEventListener("resize", handleResize);
  window.addEventListener("keydown", onDialogKeydown);
});

onBeforeUnmount(() => {
  disconnectWebSocket();
  stopTrendTicker();
  window.removeEventListener("resize", handleResize);
  window.removeEventListener("keydown", onDialogKeydown);
  for (const inst of sparklineInstances.values()) {
    inst.dispose();
  }
  sparklineInstances.clear();
});
</script>

<style scoped>
.main {
  padding: 16px;
}

.tag-prefix-inline {
  margin-left: 10px;
  padding: 2px 8px;
  font-size: 13px;
  font-family: monospace;
  font-weight: normal;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: 3px;
  vertical-align: middle;
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

.current-value {
  color: #909399;
}

.current-value.has-value {
  font-weight: 600;
  color: #67c23a;
}

.status-msg {
  font-weight: 600;
}

.status-ok {
  color: #67c23a;
}

.status-err {
  color: #f56c6c;
}

.time-delay-green {
  font-weight: 600;
  color: #67c23a;
}

.time-delay-yellow {
  font-weight: 600;
  color: #e6a23c;
}

.time-delay-red {
  font-weight: 600;
  color: #f56c6c;
}

.time-delay-none {
  color: #909399;
}

.sparkline {
  box-sizing: border-box;
  width: 100%;
  height: 200px;
}

/* 左树右表布局 */
.point-layout {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

.tree-col {
  flex: 0 0 300px;
  width: 300px;
}

.table-col {
  flex: 1 1 auto;
  min-width: 0;
}

.tree-card :deep(.el-card__body) {
  padding-top: 8px;
}

.tree-box {
  height: calc(100vh - 360px);
  min-height: 200px;
}

.tree-box :deep(.el-scrollbar__wrap) {
  height: 100%;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
}

.node-icon {
  color: var(--el-color-primary);
}

.node-label {
  font-size: 14px;
}

.node-tag {
  margin-left: 4px;
}

.filter-tag {
  margin-right: 8px;
}

@media (max-width: 768px) {
  .main {
    padding: 8px;
  }

  .point-layout {
    flex-direction: column;
    gap: 8px;
  }

  .tree-col {
    flex-basis: auto;
    width: 100%;
  }

  .tree-box {
    height: auto;
    max-height: 220px;
    min-height: 120px;
  }

  .tree-box :deep(.el-scrollbar__wrap) {
    height: auto;
    max-height: 220px;
  }

  /* 卡片头部堆叠：标题占一行，工具区占一行，搜索框全宽 */
  .card-header {
    flex-direction: column;
    gap: 6px;
    align-items: stretch;
  }

  .header-actions {
    flex-wrap: wrap;
    justify-content: flex-start;
    gap: 6px;
  }

  .header-actions .search-input {
    flex: 1 1 100%;
    width: 100%;
  }

  /* 卡片内边距收紧，给窄屏多留内容区 */
  .tree-card :deep(.el-card__header),
  .table-col :deep(.el-card__header) {
    padding: 8px 12px;
  }

  .tree-card :deep(.el-card__body),
  .table-col :deep(.el-card__body) {
    padding: 8px 10px;
  }

  /* 表格字号与行高略减 */
  .table-col :deep(.el-table) {
    font-size: 13px;
  }

  .table-col :deep(.el-table th),
  .table-col :deep(.el-table td) {
    padding: 6px 4px;
  }

  .pagination-wrapper {
    justify-content: center;
    margin-top: 8px;
  }
}

@media (max-width: 480px) {
  .header-actions .el-button {
    padding: 4px 8px;
    font-size: 12px;
  }

  .card-title {
    font-size: 14px;
  }

  .table-col :deep(.el-table) {
    font-size: 12px;
  }

  .sparkline {
    height: 120px;
  }
}
</style>
