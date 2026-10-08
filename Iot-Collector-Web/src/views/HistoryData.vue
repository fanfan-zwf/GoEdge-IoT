<template>
  <div class="page-card">
    <div class="page-header">
      <h3>{{ t('historyData.title') }} <span v-if="routePointId" class="point-badge">Point ID: {{ routePointId }}</span></h3>
      <div>
        <el-button type="primary" @click="handleQuery">{{ t('common.query') }}</el-button>
        <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
      </div>
    </div>

    <!-- 查询条件 -->
    <el-form :inline="true" :model="queryForm" class="query-form">
      <el-form-item :label="t('historyData.timeRange')">
        <el-date-picker
          v-model="queryForm.timeRange"
          type="datetimerange"
          :start-placeholder="t('historyData.startTime')"
          :end-placeholder="t('historyData.endTime')"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          style="width: 380px"
        />
      </el-form-item>
    </el-form>

    <!-- 数据表格 -->
    <el-table :data="tableData" border style="width: 100%" v-loading="loading">
      <el-table-column prop="time" :label="t('historyData.time')" width="200" />
      <el-table-column prop="msg" :label="t('historyData.status')" width="120">
        <template #default="{ row }">
          <span v-if="row.msg" :class="['status-msg', row.msg === 'ok' || row.msg === '正常' ? 'status-ok' : 'status-err']">
            {{ row.msg }}
          </span>
          <span v-else class="status-msg">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="value" :label="t('historyData.value')" />
    </el-table>

    <!-- 分页 -->
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
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { configApi } from '@/api/config'

const route = useRoute()
const { t } = useI18n()

// 从路由参数获取点位 ID
const routePointId = Number(route.params.pointId) || undefined

// 查询表单
const queryForm = reactive({
  pointId: routePointId as number | undefined,
  timeRange: [] as string[]
})

// 表格数据
const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)

// 查询
const handleQuery = async () => {
  if (!queryForm.pointId) {
    ElMessage.warning('Point ID is required')
    return
  }
  if (!queryForm.timeRange || queryForm.timeRange.length !== 2) {
    ElMessage.warning(t('historyData.selectTimeRange'))
    return
  }

  loading.value = true
  try {
    const startTime = performance.now() // 使用高精度计时器
    const res = await configApi.historyData.query({
      pointId: queryForm.pointId,
      startTime: queryForm.timeRange[0],
      endTime: queryForm.timeRange[1],
      page: page.value,
      pageSize: pageSize.value
    }) as any
    const endTime = performance.now()
    const duration = Math.round(endTime - startTime) // 计算请求耗时（毫秒）
    
    // 后端返回格式：{ data: [...], total: 150 }
    const responseData = res.Data || {}
    console.log('后端返回数据:', responseData) // 调试日志
    
    tableData.value = (responseData.data || []).map((item: any) => {
      console.log('单条数据:', item) // 调试日志
      
      // 格式化为毫秒级时间：YYYY-MM-DD HH:mm:ss.SSS+08:00
      const date = new Date(item.Time)
      const dateStr = date.toLocaleDateString()
      const timeStr = date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
      })
      const ms = String(date.getMilliseconds()).padStart(3, '0')
      
      return {
        time: `${dateStr} ${timeStr}.${ms}+08:00`,
        value: item.Value,
        msg: item.Msg || '-'
      }
    })
    total.value = responseData.total || 0
    
    // 显示请求耗时
    ElMessage.success(`请求用时 ${duration}ms`)
  } catch (error: any) {
    ElMessage.error(error.message || t('common.queryFailed'))
  } finally {
    loading.value = false
  }
}

// 重置
const handleReset = () => {
  queryForm.pointId = routePointId // 重置时保留路由指定的点位
  queryForm.timeRange = []
  tableData.value = []
  total.value = 0
  page.value = 1
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  page.value = 1
  if (queryForm.pointId && queryForm.timeRange?.length === 2) {
    handleQuery()
  }
}

const handleCurrentChange = (val: number) => {
  page.value = val
  if (queryForm.pointId && queryForm.timeRange?.length === 2) {
    handleQuery()
  }
}

onMounted(() => {
  // 点位 ID 已从路由获取，无需加载点位列表
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
  padding: 16px;
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
</style>
