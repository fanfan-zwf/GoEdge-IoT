<template>
  <div class="page-card">
    <div class="page-header">
      <h3>{{ t('alarm.title') }}</h3>
      <div class="header-actions">
        <el-input v-model="searchTag" :placeholder="t('common.searchPlaceholder')" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-button @click="handleSearch">{{ t('common.search') }}</el-button>
        <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
        <el-select v-model="refreshInterval" size="small" style="width: 120px">
          <el-option :label="t('common.off')" :value="0" />
          <el-option label="1s" :value="1000" />
          <el-option label="5s" :value="5000" />
          <el-option label="10s" :value="10000" />
        </el-select>
        <el-button size="small" @click="loadStatus">{{ t('common.manualRefresh') }}</el-button>
        <el-button type="primary" @click="openAdd">{{ t('common.add') }}</el-button>
        <el-button @click="handleExport">{{ t('common.export') }}</el-button>
        <el-button @click="importDialogVisible = true">{{ t('common.import') }}</el-button>
      </div>
    </div>

    <el-table :data="tableData" border style="width: 100%">
      <el-table-column prop="Id" :label="t('alarm.id')" width="80" />
      <el-table-column prop="Point_Id" :label="t('alarm.pointId')" width="100" />
      <el-table-column prop="Tag" :label="t('alarm.tag')" width="150" />
      <el-table-column prop="Config" :label="t('alarm.config')" />
      <el-table-column prop="Group" :label="t('alarm.group')" width="100" />
      <el-table-column :label="t('alarm.status')" width="120">
        <template #default="{ row }">
          <el-tag v-if="row._status" :type="statusTagType(row._status)" size="small">{{ row._status }}</el-tag>
          <span v-else style="color: #999">-</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('alarm.lastAlarmTime')" width="120">
        <template #default="{ row }">
          <div v-if="row._time">{{ formatDate(row._time) }}</div>
          <div v-if="row._time">{{ formatTime(row._time) }}</div>
          <span v-else style="color: #999">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="Creation_Time" :label="t('common.creationTime')" width="120">
        <template #default="{ row }">
          <div>{{ formatDate(row.Creation_Time) }}</div>
          <div>{{ formatTime(row.Creation_Time) }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="viewAlarmData(row)">{{ t('alarm.viewData') }}</el-button>
          <el-button size="small" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" type="danger" @click="handleDel(row)">{{ t('common.del') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total"
        :page-sizes="[10, 20, 30, 50, 100, 150]" layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange" @current-change="handleCurrentChange" />
    </div>

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('alarm.editTitle') : t('alarm.addTitle')" width="500px"
      destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('alarm.pointId')" prop="Point_Id">
          <el-select v-model="form.Point_Id" :placeholder="t('alarm.pointIdPlaceholder')" filterable
            style="width: 100%">
            <el-option v-for="p in pointList" :key="p.Id" :label="`[${p.Id}] ${p.Tag || p.Config}`" :value="p.Id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('alarm.tag')" prop="Tag">
          <el-input v-model="form.Tag" />
        </el-form-item>
        <el-form-item prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t('alarm.config') }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a href="https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/db/db_point/README.md"
                  target="_blank" class="help-icon">?</a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Config" type="textarea" :rows="3" :placeholder="t('common.alarmConfigTip')" />
          <div class="config-tip">{{ alarmConfigFormatTip }}</div>
        </el-form-item>
        <el-form-item :label="t('alarm.group')" prop="Group">
          <el-input-number v-model="form.Group" :min="1" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
    <!-- 导入对话框 -->
    <el-dialog v-model="importDialogVisible" :title="t('common.importTitle')" width="480px" @close="importFile = null">
      <p style="color: #909399; font-size: 13px; margin-bottom: 16px;">{{ t('common.importTip') }}</p>
      <el-upload ref="uploadRef" :auto-upload="false" :limit="1" accept=".xlsx"
        :on-change="(file: any) => importFile = file.raw" :on-remove="() => importFile = null">
        <el-button type="primary">{{ t('common.importSelectFile') }}</el-button>
      </el-upload>
      <template #footer>
        <el-button @click="importDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="importLoading" @click="handleImport">{{ t('common.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { configApi, exportUrl, importConfig } from '@/api/config'

const router = useRouter()
const { t } = useI18n()

// 报警配置格式提示
const alarmConfigFormatTip = computed(() => t('common.alarmConfigFormat'))

// 点位列表（下拉选择用）
const pointList = ref<any[]>([])
const loadPointList = async () => {
  try {
    const res = await configApi.point.query({ page: 1, pageSize: 1000 }) as any
    pointList.value = res.Data || []
  } catch { /* ignore */ }
}

const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 搜索相关
const searchTag = ref('%%')
const isSearching = ref(false)

// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive({
  Id: 0,
  Point_Id: 1,
  Tag: '',
  Config: '',
  Group: 1
})

const rules = reactive<FormRules>({
  Point_Id: [{ required: true, message: () => t('common.required', { field: t('alarm.pointId') }), trigger: 'blur' }],
  Tag: [{ required: true, message: () => t('common.required', { field: t('alarm.tag') }), trigger: 'blur' }],
  Config: [{ required: true, message: () => t('common.required', { field: t('alarm.config') }), trigger: 'blur' }],
  Group: [{ required: true, message: () => t('common.required', { field: t('alarm.group') }), trigger: 'blur' }]
})

const formatDate = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleDateString()
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleTimeString() + '+08:00'
}

const loadData = async () => {
  try {
    if (isSearching.value) {
      const res = await configApi.alarm.search({
        tag: searchTag.value,
        quantity: 30
      }) as any
      tableData.value = res.Data || []
      total.value = tableData.value.length
      return
    }
    const countRes = await configApi.alarm.count({ page: 0, pageSize: 0 }) as any
    total.value = countRes.Data || 0
    if (total.value === 0) {
      tableData.value = []
      return
    }
    const res = await configApi.alarm.query({ page: page.value, pageSize: pageSize.value }) as any
    tableData.value = res.Data || []
  } catch { /* 拦截器已提示错误 */ }
}

const handleSizeChange = (val: number) => {
  pageSize.value = val
  page.value = 1
  isSearching.value = false
  loadData()
}

const handleCurrentChange = (val: number) => {
  page.value = val
  isSearching.value = false
  loadData()
}

const handleSearch = () => {
  isSearching.value = true
  loadData()
}

const handleReset = () => {
  searchTag.value = '%%'
  isSearching.value = false
  loadData()
}

const resetForm = () => {
  form.Id = 0
  form.Point_Id = 1
  form.Tag = ''
  form.Config = ''
  form.Group = 1
}

const openAdd = () => {
  isEdit.value = false
  resetForm()
  loadPointList()
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  isEdit.value = true
  loadPointList()
  form.Id = row.Id
  form.Point_Id = row.Point_Id
  form.Tag = row.Tag
  form.Config = row.Config
  form.Group = row.Group
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    const data = { Point_Id: form.Point_Id, Tag: form.Tag, Config: form.Config, Group: form.Group }
    if (isEdit.value) {
      await configApi.alarm.update([{ Id: form.Id, ...data }])
      ElMessage.success(t('common.editSuccess'))
    } else {
      await configApi.alarm.add([data])
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
    await configApi.alarm.del([row.Id])
    ElMessage.success(t('common.deleteSuccess'))
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') { /* 拦截器已提示错误 */ }
  }
}

const handleExport = () => {
  window.open(`${exportUrl}?type=alarm`, '_blank')
}

const viewAlarmData = (row: any) => {
  router.push({ name: 'alarmData', params: { alarmId: row.Id } })
}

// ======== 导入功能 ========
const importDialogVisible = ref(false)
const importFile = ref<File | null>(null)
const importLoading = ref(false)

const handleImport = async () => {
  if (!importFile.value) {
    ElMessage.warning(t('common.importSelectFile'))
    return
  }
  importLoading.value = true
  try {
    const res = await importConfig(importFile.value) as any
    const data = res.Data || {}
    const alarm = data.alarm_config || { added: 0, updated: 0, deleted: 0 }
    ElMessage.success(t('common.importSuccess', { added: alarm.added, updated: alarm.updated, deleted: alarm.deleted }))
    importDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '导入失败')
  } finally {
    importLoading.value = false
  }
}

// ======== 报警状态实时显示 ========
const statusTimer = ref<ReturnType<typeof setInterval> | null>(null)
const refreshInterval = ref<number>(5000) // 默认 5s

const restartTimer = () => {
  if (statusTimer.value) {
    clearInterval(statusTimer.value)
    statusTimer.value = null
  }
  if (refreshInterval.value > 0) {
    statusTimer.value = setInterval(() => loadStatus(), refreshInterval.value)
  }
}

const statusTagType = (status: string) => {
  if (status === '报警') return 'danger'
  if (status === '恢复') return 'success'
  return 'warning' // 错误信息等其他情况
}

const loadStatus = async () => {
  try {
    // 收集当前页所有点位的 Point_Id（去重）
    const pointIdSet = new Set<number>()
    tableData.value.forEach(row => {
      if (row.Point_Id) pointIdSet.add(row.Point_Id)
    })
    if (pointIdSet.size === 0) return

    const pointIds = Array.from(pointIdSet)
    const res = await configApi.alarm.status(pointIds) as any
    const statusList: any[] = res.Data || []

    // 按 AlarmId 匹配，合并状态到 tableData
    const statusMap = new Map<number, any>()
    for (const s of statusList) {
      // 同一个点位可能有多条报警，取最新的（Time 最大的）
      const existing = statusMap.get(s.AlarmId)
      if (!existing || new Date(s.Time) > new Date(existing.Time)) {
        statusMap.set(s.AlarmId, s)
      }
    }

    tableData.value.forEach(row => {
      const s = statusMap.get(row.Id)
      if (s) {
        row._status = s.Status || ''
        row._time = s.Time || ''
      } else {
        row._status = ''
        row._time = ''
      }
    })
  } catch { /* 静默失败，不影响主流程 */ }
}

onBeforeUnmount(() => {
  if (statusTimer.value) {
    clearInterval(statusTimer.value)
  }
})

// 监听刷新间隔变化，重启定时器
watch(refreshInterval, () => {
  restartTimer()
})

onMounted(() => {
  loadData().then(() => loadStatus())
  // 启动定时器（默认 5s）
  restartTimer()
})
</script>
