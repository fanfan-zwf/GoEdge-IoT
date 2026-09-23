<template>
  <div class="page-card">
    <div class="page-header">
      <h3>{{ t('point.title') }}</h3>
      <div class="header-actions">
        <el-button :type="showTrend ? 'primary' : 'default'" @click="showTrend = !showTrend">{{ t('point.toggleTrend') }}</el-button>
        <el-button type="primary" @click="openAdd">{{ t('common.add') }}</el-button>
        <el-button @click="handleExport">{{ t('common.export') }}</el-button>
      </div>
    </div>

    <el-table :data="tableData" border :row-style="showTrend ? { height: '200px' } : {}" style="width: 100%">
      <el-table-column prop="Id" :label="t('point.id')" width="80" />
      <el-table-column prop="Drive_Id" :label="t('point.driveId')" width="100" />
      <el-table-column prop="Name" :label="t('point.name')" width="150" />
      <el-table-column prop="Description" :label="t('point.description')" width="150" />
      <el-table-column v-if="!showTrend" prop="Config" :label="t('point.config')" />
      <el-table-column v-if="!showTrend" prop="RW_Cancel" :label="t('point.rwCancel')" width="110">
        <template #default="{ row }">{{ rwCancelMap[row.RW_Cancel] || row.RW_Cancel }}</template>
      </el-table-column>
      <el-table-column v-if="!showTrend" prop="Value_Type" :label="t('point.valueType')" width="110">
        <template #default="{ row }">{{ valueTypeMap[row.Value_Type] || row.Value_Type }}</template>
      </el-table-column>
      <el-table-column v-if="!showTrend" prop="Creation_Time" :label="t('common.creationTime')" width="170" />
      <el-table-column :label="t('point.status')" width="100">
        <template #default="{ row }">
          <span v-if="pointData[row.Id]?.msg" :class="['status-msg', pointData[row.Id].msg === 'ok' ? 'status-ok' : 'status-err']">
            {{ pointData[row.Id].msg }}
          </span>
          <span v-else class="status-msg">-</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('point.currentValue')" width="150">
        <template #default="{ row }">
          <span class="current-value" :class="{ 'has-value': pointData[row.Id]?.value !== undefined }">
            {{ pointData[row.Id]?.value !== undefined ? pointData[row.Id].value : '-' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column :label="t('point.timeDelay')" width="120">
        <template #default="{ row }">
          <span v-if="pointData[row.Id]?.timeDelay?.text" :class="pointData[row.Id].timeDelay.cls">
            {{ pointData[row.Id].timeDelay.text }}
          </span>
          <span v-else class="time-delay-none">-</span>
        </template>
      </el-table-column>
      <el-table-column v-if="showTrend" :label="t('point.trend')" min-width="300">
        <template #default="{ row }">
          <div :ref="el => setSparklineRef(el, row.Id)" class="sparkline"></div>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="210">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" type="warning" :disabled="row.RW_Cancel !== 3 && row.RW_Cancel !== 4" @click="openWrite(row)">{{ t('point.write') }}</el-button>
          <el-button size="small" type="danger" @click="handleDel(row)">{{ t('common.del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('point.editTitle') : t('point.addTitle')" width="500px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('point.driveId')" prop="Drive_Id">
          <el-select v-model="form.Drive_Id" :placeholder="t('point.driveIdPlaceholder')" filterable style="width: 100%">
            <el-option v-for="d in driveList" :key="d.Id" :label="`[${d.Id}] ${d.Name || d.Type}`" :value="d.Id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('point.name')" prop="Name">
          <el-input v-model="form.Name" />
        </el-form-item>
        <el-form-item :label="t('point.description')">
          <el-input v-model="form.Description" />
        </el-form-item>
        <el-form-item prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t('common.config') }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a href="https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/IO/Modbus_Tcp/README.md" target="_blank" class="help-icon">?</a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Config" type="textarea" :rows="3" :placeholder="t('common.configPlaceholder')" />
          <div class="config-tip">{{ pointConfigTipText }}</div>
        </el-form-item>
        <el-form-item :label="t('point.rwCancel')" prop="RW_Cancel">
          <el-select v-model="form.RW_Cancel" style="width: 100%">
            <el-option v-for="item in rwCancelOptions" :key="item.value" :label="t(item.labelKey)" :value="item.value" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('point.valueType')" prop="Value_Type">
          <el-select v-model="form.Value_Type" style="width: 100%">
            <el-option v-for="item in valueTypeOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 写入值对话框 -->
    <el-dialog v-model="writeDialogVisible" :title="t('point.writeTitle')" width="420px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item :label="t('point.writePointName')">
          <span>{{ writePoint.Name || `ID: ${writePoint.Id}` }} (ID: {{ writePoint.Id }})</span>
        </el-form-item>
        <el-form-item :label="t('point.writeType')">
          <span>{{ valueTypeMap[writePoint.Value_Type] || writePoint.Value_Type }}</span>
        </el-form-item>
        <el-form-item :label="t('point.writeCurrentValue')">
          <span class="current-value has-value">{{ pointData[writePoint.Id]?.value ?? '-' }}</span>
        </el-form-item>
        <el-form-item :label="t('point.writeValueLabel')">
          <el-input v-model="writeValue" :placeholder="t('point.writeValuePlaceholder')" style="width: 200px" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="writeDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="writeLoading" @click="handleWrite">{{ t('point.write') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import * as echarts from 'echarts'
import { configApi, exportUrl } from '@/api/config'

const { t } = useI18n()

// 读写方式选项（对应后端 RW_Cancel_map）
const rwCancelOptions = [
  { value: 1, labelKey: 'point.rwN' },
  { value: 2, labelKey: 'point.rwR' },
  { value: 3, labelKey: 'point.rwW' },
  { value: 4, labelKey: 'point.rwRW' }
]
const rwCancelMap = computed(() =>
  Object.fromEntries(rwCancelOptions.map(o => [o.value, t(o.labelKey)]))
)

// 值类型选项（对应后端 Value_Type_map）
const valueTypeOptions = [
  { value: 1, label: 'bool' },
  { value: 2, label: 'int8' },
  { value: 3, label: 'uint8' },
  { value: 4, label: 'int16' },
  { value: 5, label: 'uint16' },
  { value: 6, label: 'int32' },
  { value: 7, label: 'uint32' },
  { value: 8, label: 'int64' },
  { value: 9, label: 'uint64' },
  { value: 10, label: 'int' },
  { value: 11, label: 'uint' },
  { value: 12, label: 'float32' },
  { value: 13, label: 'float64' },
  { value: 14, label: 'float' },
  { value: 15, label: 'string' }
]
const valueTypeMap: Record<number, string> = Object.fromEntries(valueTypeOptions.map(o => [o.value, o.label]))

// 驱动列表（下拉选择用）
const driveList = ref<any[]>([])
const loadDriveList = async () => {
  try {
    const res = await configApi.drive.query({ page: 1, pageSize: 1000 }) as any
    driveList.value = res.Data || []
  } catch { /* ignore */ }
}

// 根据选中驱动类型动态显示点位配置提示
const currentDriveType = computed(() => {
  const drive = driveList.value.find(d => d.Id === form.Drive_Id)
  return drive?.Type || ''
})
const pointConfigTipText = computed(() => {
  if (currentDriveType.value === 'Siemens_S7') return t('common.s7PointConfigTip')
  return t('common.pointConfigTip')
})

const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// WebSocket 实时值（合并：当前值、状态消息、时间延迟、时间戳）
const pointData = ref<Record<number, { value: any; msg: string; time: string; timeDelay: { text: string; cls: string } }>>({})
const ensurePointData = (id: number) => {
  if (!pointData.value[id]) {
    pointData.value[id] = { value: undefined, msg: '', time: '', timeDelay: { text: '', cls: '' } }
  }
  return pointData.value[id]
}
let ws: WebSocket | null = null

const WS_URL = 'ws://192.168.220.40:8103/api/v1.0/point/ws'

// 视图 & 曲线图
const showTrend = ref(false) // 趋势列显示开关
const chartSeriesData: Record<number, { timestamps: number[]; values: number[] }> = {}
const MAX_CHART_POINTS = 1000 // 每个点位最多保留的数据点数（安全上限）

// Sparkline 小曲线
const sparklineInstances = new Map<number, echarts.ECharts>()
const sparklineRefs = new Map<number, HTMLElement>()

const setSparklineRef = (el: any, pointId: number) => {
  if (el) {
    sparklineRefs.set(pointId, el)
    initSparkline(pointId)
    // 如果已有数据（API 预加载），立即更新图表
    const sd = chartSeriesData[pointId]
    if (sd && sd.timestamps.length > 0) {
      updateSparkline(pointId)
    }
  } else {
    sparklineRefs.delete(pointId)
    const inst = sparklineInstances.get(pointId)
    if (inst) { inst.dispose(); sparklineInstances.delete(pointId) }
  }
}

const initSparkline = (pid: number) => {
  const el = sparklineRefs.get(pid)
  if (!el) return
  let inst = sparklineInstances.get(pid)
  if (inst) { inst.dispose() }
  inst = echarts.init(el)
  sparklineInstances.set(pid, inst)
  // 初始化图表骨架（坐标轴 + 空系列），后续通过 setOption 增量更新数据
  const nowMs = Date.now()
  inst.setOption({
    animation: true,
    animationDurationUpdate: 1000,
    animationEasingUpdate: 'linear',
    grid: { left: 30, right: 15, top: 10, bottom: 25 },
    xAxis: {
      type: 'value',
      show: true,
      min: nowMs - 40000,
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
      type: 'value',
      show: true,
      splitNumber: 2,
      min: -1,
      max: 1,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { fontSize: 10, formatter: (val: number) => Math.round(val) },
      splitLine: { show: false }
    },
    series: [{
      type: 'line',
      data: [],
      showSymbol: false,
      lineStyle: { width: 1.5, color: '#5470c6' }
    }]
  })
}

// 批量更新所有 sparkline 图表（WebSocket 数据到来时调用）
const updateAllSparklines = () => {
  for (const pid of sparklineInstances.keys()) {
    updateSparkline(pid)
  }
}

// 完整更新：仅在新数据到来时调用，追加 series + 重算 y 轴 + 同步 x 轴窗口
const updateSparkline = (pid: number) => {
  const inst = sparklineInstances.get(pid)
  if (!inst) return
  const sd = chartSeriesData[pid]
  if (!sd || sd.timestamps.length === 0) return

  // 值为 0 的数据点直接剔除，曲线不显示 0 值部分
  const seriesData: [number, number][] = []
  let yMin = Infinity
  let yMax = -Infinity
  for (let i = 0; i < sd.timestamps.length; i++) {
    const v = sd.values[i]
    if (v === 0) continue
    seriesData.push([sd.timestamps[i], v])
    if (v < yMin) yMin = v
    if (v > yMax) yMax = v
  }

  // 计算 Y 轴范围（加 10% 边距），无非零值时使用默认范围
  let axisMin: number
  let axisMax: number
  if (yMax > yMin) {
    const pad = (yMax - yMin) * 0.1
    axisMin = yMin - pad
    axisMax = yMax + pad
  } else {
    axisMin = -1
    axisMax = 1
  }

  // 完整更新：series + y 轴 + x 轴窗口
  const nowMs = Date.now()
  inst.setOption({
    xAxis: {
      min: nowMs - 40000,
      max: nowMs,
      axisLabel: {
        formatter: (val: number) => `${Math.round((nowMs - val) / 1000)}s`
      }
    },
    yAxis: { min: axisMin, max: axisMax },
    series: [{ data: seriesData }]
  })
}

// 监听 pointData 变化，从 pointData 读取值和时间戳追加到曲线
watch(pointData, () => {
  if (!showTrend.value) return
  for (const pid of sparklineInstances.keys()) {
    const pd = pointData.value[pid]
    if (!pd || pd.value === undefined) continue
    if (!chartSeriesData[pid]) {
      chartSeriesData[pid] = { timestamps: [], values: [] }
    }
    const sd = chartSeriesData[pid]
    const val = typeof pd.value === 'number' ? pd.value : (Number(pd.value) || 0)
    // 直接用 pointData 里存的时间戳，没有则用当前时间
    const ts = pd.time ? new Date(pd.time).getTime() : Date.now()
    sd.timestamps.push(ts)
    sd.values.push(val)
    // 安全上限
    if (sd.timestamps.length > MAX_CHART_POINTS) {
      const removeCount = sd.timestamps.length - MAX_CHART_POINTS
      sd.timestamps.splice(0, removeCount)
      sd.values.splice(0, removeCount)
    }
    updateSparkline(pid)
  }
}, { deep: true })

const connectWebSocket = () => {
  disconnectWebSocket()
  const pointIds = tableData.value.map(r => r.Id).filter(id => id > 0)
  if (pointIds.length === 0) return

  const url = `${WS_URL}?points=${JSON.stringify(pointIds)}`
  ws = new WebSocket(url)

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      // 后端批量推送，msg 是数组
      const items = Array.isArray(msg) ? msg : [msg]
      for (const data of items) {
        if (data.PointId === undefined) continue
        ensurePointData(data.PointId).value = data.Value
        if (data.Msg !== undefined) {
          ensurePointData(data.PointId).msg = data.Msg
        }
        // 直接存 WS 原始时间字符串
        ensurePointData(data.PointId).time = data.Time || ''
        // 时间延迟直接算
        const ts = data.Time ? new Date(data.Time).getTime() : Date.now()
        const delayMs = Date.now() - ts
        const delaySec = Math.floor(delayMs / 1000)
        let text: string
        let cls: string
        if (delayMs < 1000) { text = `${delayMs}ms`; cls = 'time-delay-green' }
        else if (delaySec < 60) { text = `${delaySec}s`; cls = delayMs < 3000 ? 'time-delay-yellow' : 'time-delay-red' }
        else { const min = Math.floor(delaySec / 60); text = `${min}m${delaySec % 60}s`; cls = 'time-delay-red' }
        ensurePointData(data.PointId).timeDelay = { text, cls }
        // 更新曲线图数据（只更新 pointData，由 watch 统一追加到曲线）
        ensurePointData(data.PointId)
      }
      // 更新每行 sparkline
      updateAllSparklines()
    } catch { /* ignore */ }
  }

  ws.onclose = () => {
    ws = null
  }

  ws.onerror = () => {
    ws?.close()
    ws = null
  }
}

const disconnectWebSocket = () => {
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
}



// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  Id: 0,
  Drive_Id: 1,
  Name: '',
  Description: '',
  Config: '',
  RW_Cancel: 0,
  Value_Type: 0
})

const rules = reactive<FormRules>({
  Drive_Id: [{ required: true, message: () => t('common.required', { field: t('point.driveId') }), trigger: 'blur' }],
  Name: [{ required: true, message: () => t('common.required', { field: t('point.name') }), trigger: 'blur' }],
  Config: [{ required: true, message: () => t('common.required', { field: t('common.config') }), trigger: 'blur' }],
  RW_Cancel: [{ required: true, message: () => t('common.required', { field: t('point.rwCancel') }), trigger: 'blur' }],
  Value_Type: [{ required: true, message: () => t('common.required', { field: t('point.valueType') }), trigger: 'blur' }]
})

const loadData = async () => {
  try {
    const countRes = await configApi.point.count({ page: 0, pageSize: 0 }) as any
    total.value = countRes.Data || 0
    if (total.value === 0) {
      tableData.value = []
      return
    }
    // 加载全部点位
    const res = await configApi.point.query({ page: 1, pageSize: total.value }) as any
    tableData.value = res.Data || []
    
    // 先通过API获取当前值
    const pointIds = tableData.value.map(r => r.Id).filter(id => id > 0)
    try {
      const currentValueRes: any = await configApi.point.readValue(pointIds)
      if (currentValueRes && currentValueRes.Code === 200) {
        // 直接使用响应中的 data 数组
        const values = Array.isArray(currentValueRes.Data) ? currentValueRes.Data : []
        
        for (const item of values) {
          const pointId = item.PointId
          ensurePointData(pointId).value = item.Value
          if (item.Msg !== undefined) {
            ensurePointData(pointId).msg = item.Msg
          }
          // 存 API 原始时间字符串
          ensurePointData(pointId).time = item.Time || ''
          // 时间延迟直接算
          const ts = item.Time ? new Date(item.Time).getTime() : Date.now()
          if (ts > 0) {
            const delayMs = Date.now() - ts
            const delaySec = Math.floor(delayMs / 1000)
            let text: string
            let cls: string
            if (delayMs < 1000) { text = `${delayMs}ms`; cls = 'time-delay-green' }
            else if (delaySec < 60) { text = `${delaySec}s`; cls = delayMs < 3000 ? 'time-delay-yellow' : 'time-delay-red' }
            else { const min = Math.floor(delaySec / 60); text = `${min}m${delaySec % 60}s`; cls = 'time-delay-red' }
            ensurePointData(pointId).timeDelay = { text, cls }
          }
        }
        
        // 计算API请求延迟，类似于WebSocket的延迟计算
        if (currentValueRes.Timestamp) {
          const serverTime = new Date(currentValueRes.Timestamp).getTime()
          if (serverTime > 0) {
            const diffMs = Date.now() - serverTime
            const sec = Math.floor(diffMs / 1000)
            let text: string
            let cls: string
            if (diffMs < 1000) { text = `${diffMs}ms`; cls = 'time-delay-green' }
            else if (sec < 60) { text = `${sec}s`; cls = diffMs < 3000 ? 'time-delay-yellow' : 'time-delay-red' }
            else { const min = Math.floor(sec / 60); text = `${min}m${sec % 60}s`; cls = 'time-delay-red' }
            
            // 为所有点位设置相同的API请求延迟
            for (const item of values) {
              ensurePointData(item.PointId).timeDelay = { text, cls }
            }
          }
        }
      }
    } catch (err) {
      console.error('获取初始值失败:', err)
    }
    
    // 然后连接 WebSocket
    connectWebSocket()
    
    // 强制更新 sparkline 图表以显示初始值
    if (showTrend.value) {
      nextTick(() => {
        updateAllSparklines()
      })
    }
  } catch { /* 拦截器已提示错误 */ }
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

const resetForm = () => {
  form.Id = 0
  form.Drive_Id = 1
  form.Name = ''
  form.Description = ''
  form.Config = ''
  form.RW_Cancel = 0
  form.Value_Type = 0
}

const openAdd = () => {
  isEdit.value = false
  resetForm()
  loadDriveList()
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  isEdit.value = true
  loadDriveList()
  form.Id = row.Id
  form.Drive_Id = row.Drive_Id
  form.Name = row.Name
  form.Description = row.Description || ''
  form.Config = row.Config
  form.RW_Cancel = row.RW_Cancel
  form.Value_Type = row.Value_Type
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    const data = {
      Drive_Id: form.Drive_Id,
      Name: form.Name,
      Description: form.Description,
      Config: form.Config,
      RW_Cancel: form.RW_Cancel,
      Value_Type: form.Value_Type
    }
    if (isEdit.value) {
      await configApi.point.update([{ Id: form.Id, ...data }])
      ElMessage.success(t('common.editSuccess'))
    } else {
      await configApi.point.add([data])
      ElMessage.success(t('common.addSuccess'))
    }
    dialogVisible.value = false
    loadData()
  } catch { /* 拦截器已提示错误 */ } finally {
    submitting.value = false
  }
}

const handleDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      t('common.deleteConfirm', { name: `ID: ${row.Id}` }),
      t('common.warning'),
      { type: 'warning' }
    )
    await configApi.point.del([row.Id])
    ElMessage.success(t('common.deleteSuccess'))
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') { /* 拦截器已提示错误 */ }
  }
}

const handleExport = () => {
  window.open(`${exportUrl}?type=point`, '_blank')
}

// ======== 写入功能 ========
const valueTypeNameMap: Record<number, string> = {
  1: 'bool', 2: 'int8', 3: 'uint8', 4: 'int16', 5: 'uint16',
  6: 'int32', 7: 'uint32', 8: 'int64', 9: 'uint64', 10: 'int',
  11: 'uint', 12: 'float32', 13: 'float64', 14: 'float', 15: 'string'
}

const writeDialogVisible = ref(false)
const writeLoading = ref(false)
const writeValue = ref('')
const writePoint = reactive({ Id: 0, Name: '', Value_Type: 0 })

const openWrite = (row: any) => {
  writePoint.Id = row.Id
  writePoint.Name = row.Name
  writePoint.Value_Type = row.Value_Type
  writeValue.value = ''
  writeDialogVisible.value = true
}

const handleWrite = async () => {
  if (writeValue.value === '') {
    ElMessage.warning(t('point.writeValueRequired'))
    return
  }
  writeLoading.value = true
  try {
    const typeName = valueTypeNameMap[writePoint.Value_Type] || 'float64'
    let parsedValue: any = writeValue.value
    // 尝试解析为数字
    if (typeName !== 'string' && typeName !== 'bool') {
      const num = Number(writeValue.value)
      if (!isNaN(num)) parsedValue = num
    } else if (typeName === 'bool') {
      parsedValue = writeValue.value === 'true' || writeValue.value === '1'
    }
    await configApi.point.writeValue([{
      PointId: writePoint.Id,
      Value: parsedValue,
      Type: typeName
    }])
    ElMessage.success(t('point.writeSuccess'))
    writeDialogVisible.value = false
  } catch { /* 拦截器已提示错误 */ } finally {
    writeLoading.value = false
  }
}

const handleResize = () => {
  for (const inst of sparklineInstances.values()) {
    inst.resize()
  }
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})

// 监听趋势列切换，确保切换后图表立即拿到已有数据
watch(showTrend, (val) => {
  if (val) {
    // 双 nextTick：第一次等 Vue 响应式更新，第二次等 el-table 渲染新列的 DOM
    nextTick(() => {
      nextTick(() => {
        // 先用 pointData 当前值填充曲线，让图表立即显示
        const now = Date.now()
        for (const pid of sparklineInstances.keys()) {
          const pd = pointData.value[pid]
          if (!pd || pd.value === undefined) continue
          if (!chartSeriesData[pid]) {
            chartSeriesData[pid] = { timestamps: [], values: [] }
          }
          const sd = chartSeriesData[pid]
          const v = typeof pd.value === 'number' ? pd.value : (Number(pd.value) || 0)
          const ts = pd.time ? new Date(pd.time).getTime() : now
          sd.timestamps.push(ts)
          sd.values.push(v)
        }
        // 然后正常更新图表
        updateAllSparklines()
      })
    })
  }
})

onBeforeUnmount(() => {
  disconnectWebSocket()
  window.removeEventListener('resize', handleResize)
  for (const inst of sparklineInstances.values()) {
    inst.dispose()
  }
  sparklineInstances.clear()
})
</script>

<style scoped>
.current-value {
  color: #909399;
}
.current-value.has-value {
  color: #67c23a;
  font-weight: 600;
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
  color: #67c23a;
  font-weight: 600;
}
.time-delay-yellow {
  color: #e6a23c;
  font-weight: 600;
}
.time-delay-red {
  color: #f56c6c;
  font-weight: 600;
}
.time-delay-none {
  color: #909399;
}
.header-actions {
  display: flex;
  align-items: center;
}
.sparkline {
  width: 100%;
  height: 200px;
  box-sizing: border-box;
}
</style>
