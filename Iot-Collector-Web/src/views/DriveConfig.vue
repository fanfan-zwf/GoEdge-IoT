<template>
  <div class="page-card">
    <div class="page-header">
      <h3>{{ t('drive.title') }}</h3>
      <div class="header-actions">
        <el-input v-model="searchName" :placeholder="t('common.searchPlaceholder')" clearable style="width: 200px" @keyup.enter="handleSearch" />
        <el-select v-model="searchType" :placeholder="t('drive.type')" clearable style="width: 160px">
          <el-option v-for="item in driveTypes" :key="item.value" :label="item.label" :value="item.value" />
        </el-select>
        <el-button @click="handleSearch">{{ t('common.search') }}</el-button>
        <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
        <el-button type="primary" @click="openAdd">{{ t('common.add') }}</el-button>
        <el-button @click="handleExport">{{ t('common.export') }}</el-button>
        <el-button @click="importDialogVisible = true">{{ t('common.import') }}</el-button>
      </div>
    </div>

    <el-table :data="tableData" border style="width: 100%">
      <el-table-column prop="Id" :label="t('drive.id')" width="80" />
      <el-table-column prop="Type" :label="t('drive.type')" width="150" />
      <el-table-column prop="Name" :label="t('drive.name')" width="150" />
      <el-table-column prop="Config" :label="t('drive.config')" />
      <el-table-column prop="Points_Length" :label="t('drive.pointsLength')" width="100" />
      <el-table-column prop="Creation_Time" :label="t('common.creationTime')" width="120">
        <template #default="{ row }">
          <div>{{ formatDate(row.Creation_Time) }}</div>
          <div>{{ formatTime(row.Creation_Time) }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="220">
        <template #default="{ row }">
          <el-button size="small" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" type="warning" :loading="restartingId === row.Id" @click="handleRestart(row)">{{ t('common.restart') }}</el-button>
          <el-button size="small" type="danger" @click="handleDel(row)">{{ t('common.del') }}</el-button>
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

    <!-- 新增/编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? t('drive.editTitle') : t('drive.addTitle')" width="500px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="t('drive.type')" prop="Type">
          <el-select v-model="form.Type" :placeholder="t('drive.typePlaceholder')" :disabled="isEdit" style="width: 100%">
            <el-option
              v-for="item in driveTypes"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('drive.name')" prop="Name">
          <el-input v-model="form.Name" />
        </el-form-item>
        <el-form-item prop="Config">
          <template #label>
            <span class="label-with-help">
              <span>{{ t('common.config') }}</span>
              <el-tooltip :content="t('common.viewDoc')" placement="top">
                <a :href="docUrl" target="_blank" class="help-icon">?</a>
              </el-tooltip>
            </span>
          </template>
          <el-input v-model="form.Config" type="textarea" :rows="4" :placeholder="t('common.configPlaceholder')" />
          <div class="config-tip">{{ currentConfigTip }}</div>
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
      <el-upload
        ref="uploadRef"
        :auto-upload="false"
        :limit="1"
        accept=".xlsx"
        :on-change="(file: any) => importFile = file.raw"
        :on-remove="() => importFile = null"
      >
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
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { configApi, exportUrl, importConfig } from '@/api/config'

const { t } = useI18n()

// 驱动类型下拉选项（对应 IO 文件夹中的驱动）
const driveTypes = computed(() => [
  { value: 'Modbus_Tcp', label: 'Modbus TCP' },
  { value: 'Siemens_S7', label: 'Siemens S7' }
])

// 根据驱动类型动态显示配置格式提示
const currentConfigTip = computed(() => {
  if (form.Type === 'Siemens_S7') return t('common.s7ConfigTip')
  return t('common.modbusConfigTip')
})

// 文档链接（指向 GitHub README）
const docUrl = computed(() => {
  const base = 'https://github.com/fanfan-zwf/GoEdge-IoT/blob/main/Iot-Collector-Service/IO'
  if (form.Type === 'Siemens_S7') return `${base}/Siemens_S7/README.md`
  return `${base}/Modbus_Tcp/README.md`
})

const tableData = ref<any[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 搜索相关
const searchName = ref('%%')
const searchType = ref('')
const isSearching = ref(false)

// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const restartingId = ref(0)
const formRef = ref<FormInstance>()
const form = reactive({
  Id: 0,
  Type: '',
  Name: '',
  Config: ''
})

const rules = reactive<FormRules>({
  Type: [{ required: true, message: () => t('common.required', { field: t('drive.type') }), trigger: 'blur' }],
  Name: [{ required: true, message: () => t('common.required', { field: t('drive.name') }), trigger: 'blur' }],
  Config: [{ required: true, message: () => t('common.required', { field: t('common.config') }), trigger: 'blur' }]
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
      const res = await configApi.drive.search({
        name: searchName.value,
        types: searchType.value ? [searchType.value] : undefined,
        quantity: 30
      }) as any
      tableData.value = res.Data || []
      total.value = tableData.value.length
      return
    }
    const countRes = await configApi.drive.count({ page: 0, pageSize: 0 }) as any
    total.value = countRes.Data || 0
    if (total.value === 0) {
      tableData.value = []
      return
    }
    const res = await configApi.drive.query({ page: page.value, pageSize: pageSize.value }) as any
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
  searchName.value = '%%'
  searchType.value = ''
  isSearching.value = false
  loadData()
}

const resetForm = () => {
  form.Id = 0
  form.Type = ''
  form.Name = ''
  form.Config = ''
}

const openAdd = () => {
  isEdit.value = false
  resetForm()
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  isEdit.value = true
  form.Id = row.Id
  form.Type = row.Type
  form.Name = row.Name
  form.Config = row.Config
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    if (isEdit.value) {
      await configApi.drive.update([{ Id: form.Id, Name: form.Name, Config: form.Config }])
      ElMessage.success(t('common.editSuccess'))
    } else {
      await configApi.drive.add([{ Type: form.Type, Name: form.Name, Config: form.Config }])
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
    await configApi.drive.del([row.Id])
    ElMessage.success(t('common.deleteSuccess'))
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') { /* 拦截器已提示错误 */ }
  }
}

const handleRestart = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      t('common.restartConfirm', { name: `[${row.Id}] ${row.Name || row.Type}` }),
      t('common.warning'),
      { type: 'warning' }
    )
    restartingId.value = row.Id
    await configApi.drive.restart(row.Id)
    ElMessage.success(t('common.restartSuccess'))
  } catch (error: any) {
    if (error !== 'cancel') { /* 拦截器已提示错误 */ }
  } finally {
    restartingId.value = 0
  }
}

const handleExport = () => {
  window.open(`${exportUrl}?type=drive`, '_blank')
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
    const drive = data.drive_config || { added: 0, updated: 0, deleted: 0 }
    ElMessage.success(t('common.importSuccess', { added: drive.added, updated: drive.updated, deleted: drive.deleted }))
    importDialogVisible.value = false
    loadData()
  } catch (e: any) {
    ElMessage.error(e.message || '导入失败')
  } finally {
    importLoading.value = false
  }
}

// 页面加载时初始化数据
onMounted(() => {
  loadData()
})
</script>
