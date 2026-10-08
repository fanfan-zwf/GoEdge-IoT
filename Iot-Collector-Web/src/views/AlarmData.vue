<template>
  <div class="page-card">
    <div class="page-header">
      <h3>{{ t('alarmData.title') }} <span v-if="routeAlarmId" class="point-badge">Alarm ID: {{ routeAlarmId }}</span></h3>
      <div>
        <el-button type="primary" @click="handleQuery">{{ t('common.query') }}</el-button>
        <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
      </div>
    </div>

    <!-- 查询条件 -->
    <el-form :inline="true" :model="queryForm" class="query-form">
      <el-form-item :label="t('alarmData.timeRange')">
        <el-date-picker
          v-model="queryForm.timeRange"
          type="datetimerange"
          :start-placeholder="t('alarmData.startTime')"
          :end-placeholder="t('alarmData.endTime')"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          style="width: 380px"
        />
      </el-form-item>
    </el-form>

    <el-table :data="filteredData" border style="width: 100%" v-loading="loading">
      <el-table-column prop="Id" :label="t('alarmData.id')" width="80" />
      <el-table-column prop="Alarm_Id" :label="t('alarmData.alarmId')" width="100" />
      <el-table-column :label="t('alarmData.status')" width="120">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.Status)" size="small">{{ row.Status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="Time" :label="t('alarmData.time')" width="120">
        <template #default="{ row }">
          <div>{{ formatDate(row.Time) }}</div>
          <div>{{ formatTime(row.Time) }}</div>
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { configApi } from '@/api/config'

const route = useRoute()
const { t } = useI18n()

const routeAlarmId = computed(() => route.params.alarmId ? Number(route.params.alarmId) : 0)

const queryForm = ref({
  timeRange: null as string[] | null
})

const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)

// 按时间范围过滤（后端暂不支持时间过滤，前端先过滤）
const filteredData = computed(() => {
  if (!queryForm.value.timeRange || queryForm.value.timeRange.length !== 2) {
    return tableData.value
  }
  const start = new Date(queryForm.value.timeRange[0]).getTime()
  const end = new Date(queryForm.value.timeRange[1]).getTime()
  return tableData.value.filter(row => {
    const t = new Date(row.Time).getTime()
    return t >= start && t <= end
  })
})

const statusTagType = (status: string) => {
  if (status === '报警') return 'danger'
  if (status === '恢复') return 'success'
  return 'warning'
}

const formatDate = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleDateString()
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleTimeString() + '+08:00'
}

const loadData = async () => {
  loading.value = true
  try {
    const alarmIds = routeAlarmId.value ? [routeAlarmId.value] : undefined
    const countRes = await configApi.alarmData.count({ alarmId: alarmIds, page: 0, pageSize: 0 }) as any
    total.value = countRes.Data || 0
    if (total.value === 0) {
      tableData.value = []
      return
    }
    const res = await configApi.alarmData.query({ alarmId: alarmIds, page: page.value, pageSize: pageSize.value }) as any
    tableData.value = res.Data || []
  } catch { /* 拦截器已提示错误 */ } finally {
    loading.value = false
  }
}

const handleQuery = () => {
  page.value = 1
  loadData()
}

const handleReset = () => {
  queryForm.value.timeRange = null
  page.value = 1
  loadData()
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  page.value = 1
  loadData()
}

const handleCurrentChange = (val: number) => {
  page.value = val
  loadData()
}

onMounted(() => {
  if (!routeAlarmId.value) {
    ElMessage.warning('Alarm ID is required')
    return
  }
  loadData()
})
</script>

<style scoped>
.point-badge {
  display: inline-block;
  padding: 2px 10px;
  margin-left: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #409eff;
  background: #ecf5ff;
  border: 1px solid #b3d8ff;
  border-radius: 12px;
  vertical-align: middle;
}

.query-form {
  margin-bottom: 16px;
  padding: 16px 0;
  border-bottom: 1px solid #ebeef5;
}
</style>
