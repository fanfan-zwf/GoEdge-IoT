<template>
  <div class="main">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span class="card-title">
            {{ t("historyData.title") }}
            <span v-if="routePointId" class="point-badge">Point ID: {{ routePointId }}</span>
          </span>
          <div class="header-actions">
            <el-button :type="showChart ? 'success' : 'default'" @click="toggleChart">
              {{ showChart ? t('historyData.showTable') : t('historyData.showChart') }}
            </el-button>
            <el-button type="primary" @click="handleQuery">{{ t("common.query") }}</el-button>
            <el-button @click="handleReset">{{ t("common.reset") }}</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="queryForm" class="query-form">
        <el-form-item :label="t('historyData.timeRange')">
          <el-date-picker v-model="queryForm.timeRange" type="datetimerange"
            :start-placeholder="t('historyData.startTime')" :end-placeholder="t('historyData.endTime')"
            value-format="YYYY-MM-DDTHH:mm:ssZ" style="width: 380px" />
        </el-form-item>
      </el-form>

      <!-- 表格视图 -->
      <el-table
        v-if="!showChart"
        v-loading="loading"
        :data="tableData"
        border
        max-height="calc(100vh - 360px)"
        style="width: 100%"
        :default-sort="{ prop: 'time', order: 'descending' }"
      >
        <el-table-column prop="time" :label="t('historyData.time')" width="200" sortable />
        <el-table-column prop="msg" :label="t('historyData.status')" width="120" sortable>
          <template #default="{ row }">
            <span v-if="row.msg"
              :class="['status-msg', row.msg === 'ok' || row.msg === '正常' ? 'status-ok' : 'status-err']">
              {{ row.msg }}
            </span>
            <span v-else class="status-msg">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="value" :label="t('historyData.value')" sortable />
      </el-table>

      <!-- 曲线视图 -->
      <div v-show="showChart" ref="chartContainerRef" class="history-chart" />

      <div class="pagination-wrapper">
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
          :page-sizes="[10, 20, 30, 50, 100, 150]" layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange" @current-change="handleCurrentChange" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount, nextTick, watch } from "vue";
import { useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import * as echarts from "echarts";
import { message } from "@/utils/message";
import { configApi } from "@/api/config";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";

const route = useRoute();
const { t } = useI18n();

const routePointId = Number(route.params.pointId) || undefined;

const updateTabTitle = async () => {
  if (!routePointId) return;
  try {
    const res = (await configApi.point.query({
      page: 1,
      pageSize: 1000
    })) as any;
    const points = res.Data || [];
    const point = points.find((p: any) => p.Id === routePointId);
    if (point?.Tag) {
      const title = `${point.Tag} 历史数据`;
      route.meta.title = title;
      document.title = `${title} | Iot-Collector`;
      // 更新 LayTag 标签页标题
      const tags = useMultiTagsStoreHook().multiTags;
      const tag = tags.find(t => t.path === route.path && t.params?.pointId == String(routePointId));
      if (tag) {
        tag.meta.title = title;
        useMultiTagsStoreHook().multiTags = [...tags];
      }
    }
  } catch {
    // 忽略错误
  }
};

const queryForm = reactive({
  pointId: routePointId as number | undefined,
  timeRange: [] as string[]
});

const tableData = ref<any[]>([]);
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);
const loading = ref(false);

// ---- 曲线图 ----
const showChart = ref(false);
const chartContainerRef = ref<HTMLElement | null>(null);
let chartInstance: echarts.ECharts | null = null;

const toggleChart = () => {
  showChart.value = !showChart.value;
  if (showChart.value) {
    nextTick(() => renderChart());
  }
};

const renderChart = () => {
  if (!chartContainerRef.value) return;
  if (!chartInstance) {
    chartInstance = echarts.init(chartContainerRef.value);
  }
  // 按时间升序排列（图表从左到右时间递增）
  const sorted = [...tableData.value].sort(
    (a, b) => new Date(a.time).getTime() - new Date(b.time).getTime()
  );
  const seriesData: [number, number][] = sorted
    .map(row => {
      const ts = new Date(row.time).getTime();
      const val = typeof row.value === "number" ? row.value : Number(row.value);
      if (isNaN(ts) || isNaN(val)) return null;
      return [ts, val] as [number, number];
    })
    .filter((d): d is [number, number] => d !== null);

  chartInstance.setOption({
    animation: false,
    tooltip: {
      trigger: "axis",
      formatter: (params: any) => {
        const p = params[0];
        if (!p) return "";
        const time = new Date(p.value[0]).toLocaleString("zh-CN", { hour12: false });
        return `${time}<br/>值: <b>${p.value[1]}</b>`;
      }
    },
    grid: { left: 60, right: 30, top: 20, bottom: 40 },
    xAxis: {
      type: "time",
      axisLabel: { hideOverlap: true }
    },
    yAxis: {
      type: "value",
      scale: true,
      splitLine: { lineStyle: { type: "dashed" } }
    },
    series: [
      {
        type: "line",
        data: seriesData,
        showSymbol: true,
        symbolSize: 6,
        lineStyle: { width: 1.5, color: "#5470c6" },
        itemStyle: { color: "#5470c6" }
      }
    ]
  });
  chartInstance.resize();
};

const handleResize = () => {
  chartInstance?.resize();
};

watch(tableData, () => {
  if (showChart.value) {
    nextTick(() => renderChart());
  }
});

const handleQuery = async () => {
  if (!queryForm.pointId) {
    message("Point ID is required", { type: "warning" });
    return;
  }
  if (!queryForm.timeRange || queryForm.timeRange.length !== 2) {
    message(t("historyData.selectTimeRange"), { type: "warning" });
    return;
  }
  loading.value = true;
  try {
    const startTime = performance.now();
    const res = (await configApi.historyData.query({
      pointId: queryForm.pointId,
      startTime: queryForm.timeRange[0],
      endTime: queryForm.timeRange[1],
      page: page.value,
      pageSize: pageSize.value
    })) as any;
    const endTime = performance.now();
    const duration = Math.round(endTime - startTime);
    const responseData = res.Data || {};
    tableData.value = (responseData.data || []).map((item: any) => {
      const date = new Date(item.Time);
      const dateStr = date.toLocaleDateString();
      const timeStr = date.toLocaleTimeString("zh-CN", {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
        hour12: false
      });
      const ms = String(date.getMilliseconds()).padStart(3, "0");
      return {
        time: `${dateStr} ${timeStr}.${ms}+08:00`,
        value: item.Value,
        msg: item.Msg || "-"
      };
    });
    total.value = responseData.total || 0;
    message(`请求用时 ${duration}ms`, { type: "success" });
  } catch (error: any) {
    message(error.message || t("common.queryFailed"), {
      type: "error"
    });
  } finally {
    loading.value = false;
  }
};

const handleReset = () => {
  queryForm.pointId = routePointId;
  queryForm.timeRange = [];
  tableData.value = [];
  total.value = 0;
  page.value = 1;
};

const handleSizeChange = (val: number) => {
  pageSize.value = val;
  page.value = 1;
  if (queryForm.pointId && queryForm.timeRange?.length === 2) {
    handleQuery();
  }
};

const handleCurrentChange = (val: number) => {
  page.value = val;
  if (queryForm.pointId && queryForm.timeRange?.length === 2) {
    handleQuery();
  }
};

onMounted(() => {
  if (!routePointId) {
    message("Point ID is required", { type: "warning" });
    return;
  }
  updateTabTitle();
  window.addEventListener("resize", handleResize);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", handleResize);
  chartInstance?.dispose();
  chartInstance = null;
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

.point-badge {
  display: inline-block;
  padding: 2px 10px;
  margin-left: 8px;
  font-size: 13px;
  font-weight: 500;
  vertical-align: middle;
  color: #409eff;
  background: #ecf5ff;
  border: 1px solid #b3d8ff;
  border-radius: 12px;
}

.query-form {
  padding: 16px;
  margin-bottom: 16px;
  background: #f5f7fa;
  border-radius: 4px;
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

.history-chart {
  width: 100%;
  height: 400px;
  margin-top: 8px;
}
</style>
