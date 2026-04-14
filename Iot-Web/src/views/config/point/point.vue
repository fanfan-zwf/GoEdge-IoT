<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <el-table :data="config_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" width="60" align="center" />
                <el-table-column prop="Collector_Name" label="采集器名称" min-width="120" show-overflow-tooltip />
                <el-table-column prop="Name" label="名称" max-width="120" show-overflow-tooltip />
                <el-table-column prop="Type" label="类型" min-width="60" align="center" />
                <el-table-column prop="Config" label="配置" min-width="200" show-overflow-tooltip />
                <el-table-column prop="Points_Length" label="点位数量" width="100" align="center" />
                <el-table-column prop="Creation_Time" label="创建时间" width="230" align="center" />
                <el-table-column label="操作" width="180" fixed="right">
                    <template #default="scope">
                        <el-button size="small" @click="editRow(scope)">编辑</el-button>
                        <el-button size="small" type="danger" @click="deleteRow(scope)">删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
            <div style="margin-top: 20px">
                <el-button type="primary" @click="addNewRow">新增数据</el-button>
            </div>
            <div class="demo-pagination-block input-group ">
                <!-- 分页查询 -->
                <el-form-item label="分页：">
                    <el-pagination v-model:page-size="pagination.Page_length" :page-sizes="[10, 50, 100, 150, 200]"
                        layout="total, sizes, prev, pager, next, jumper" :pager-count=10
                        :total="pagination.total_length" @size-change="handleSizeChange"
                        @current-change="handleCurrentChange" />
                </el-form-item>
            </div>
        </div>
        <!-- 新增/编辑数据对话框 -->
        <el-dialog v-model="showUpdateDialog" :title="UpdateItem.Id ? '编辑驱动' : '新增驱动'" :close-on-click-modal="false"
            destroy-on-close class="responsive-dialog">
            <!-- 动态宽度控制 -->
            <template #default>
                <el-form :model="UpdateItem" ref="addFormRef" :rules="newItemRules" label-width="120px">
                    <el-form-item prop="Id" label="驱动id" v-if="UpdateItem.Id !== 0">
                        <el-input v-model.number="UpdateItem.Id" placeholder="驱动 id" size="large" clearable readonly
                            disabled />
                    </el-form-item>

                    <el-form-item prop="Collector_Id" label="驱动" v-if="UpdateItem.Id === 0">
                        <search_drive
                            :result="(value: { Id: number, Type: string }) => { UpdateItem.Drive_Id = value.Id; UpdateItem.Drive_Type = value.Type; }" />
                    </el-form-item>

                    <el-form-item prop="Name" label="标识符">
                        <el-input v-model="UpdateItem.Tag" type="text" placeholder="请输入标识符" size="large" />
                    </el-form-item>

                    <el-form-item prop="Config" label="点位参数">
                        <el-input v-model="UpdateItem.Config" placeholder="请输入点位参数" size="large" autocomplete="off"
                            @input="UpdateItem.Config = filterInput(UpdateItem.Config)" clearable />
                        <div class="input-tip" v-html="typeOptions[UpdateItem.Drive_Type] || ''"></div>
                    </el-form-item>

                    <el-form-item prop="RW_Cancel" label="读写方式" v-if="UpdateItem.Id === 0">
                        <el-select v-model="UpdateItem.RW_Cancel" placeholder="请选择驱动类型" style="width: 100%">
                            <!-- <el-option label="禁用" value="N" /> -->
                            <el-option label="只读" value="R" />
                            <el-option label="读写" value="R/W" selected />
                            <el-option label="只写" value="W" />
                        </el-select>
                    </el-form-item>

                    <el-form-item prop="Description" label="描述">
                        <el-input v-model="UpdateItem.Description" type="textarea" />
                    </el-form-item>
                </el-form>
            </template>
            <template #footer>
                <el-button @click="showUpdateDialog = false">取消</el-button>
                <el-button type="primary" @click="UpdateNewRow">确定</el-button>
            </template>
        </el-dialog>
    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { reactive, onMounted, ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, ElMessageBox } from 'element-plus' // 引入 FormInstance 类型
// 修复点3: 移除未使用的 naive-ui 导入
// import { c } from 'naive-ui' 
import { Points_Config__Count, Points_Config__Query, Points_Config__Add, Points_Config__Update, Points_Config__Del, type Points_Config__table_interface } from '@/api/config_service'
import search_drive from '@/views/config/drive/search_drive.vue'

// const router = useRouter()

const config_data: Points_Config__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
})

// 分页查询 Page 页码
const Query = (Page: number) => {
    Points_Config__Query({
        Page: Page,
        Page_Size: pagination.Page_length
    }).then((config_info) => {
        config_data.length = 0
        Object.assign(config_data, config_info)
    })
}

// 查询总条目
const Count = () => {
    Points_Config__Count().then((Count) => {
        pagination.total_length = Count
        Query(1)
    })
}

onMounted(() => {
    Count()
})

// 分页事件
const handleSizeChange = (value: number) => {
    pagination.Page_length = value
    Query(1)
}

const handleCurrentChange = (value: number) => {
    Query(value)
}

// 编辑行
const editRow = (scope: any) => {
    // 注意：Object.assign 是浅拷贝，如果 Config 是对象可能需要深拷贝，这里假设是字符串
    Object.assign(UpdateItem, scope.row)
    showUpdateDialog.value = true
}

// 删除行
const deleteRow = (scope: any) => {
    const id: number = scope.row.Id ?? 0
    if (id === 0) {
        ElMessage.error('无效的ID')
        return
    }

    ElMessageBox.prompt(`确定要删除 <span style="color:#ff0000; font-size:14px">${scope.row.Tag ?? ''}</span> 点位吗？ 输入点位标识以确认删除。`,
        '警告', {
        confirmButtonText: '确定',
        confirmButtonType: 'danger',
        cancelButtonText: '取消',
        inputPattern: new RegExp(`^${scope.row.Tag ?? '未知'}$`),
        inputErrorMessage: '输入内容不正确',
        dangerouslyUseHTMLString: true,
    })
        .then(({ }) => {
            Points_Config__Del(id).then(() => {
                ElMessage.success('删除成功')
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        })
        .catch(() => {
            ElMessage.info('已取消输入')
        })

}

// 响应式数据 
const showUpdateDialog = ref(false)

// 修复点4: 定义表单 ref
const addFormRef = ref<FormInstance>()

// 新项目数据
const UpdateItem: Points_Config__table_interface & { Drive_Id: number } = reactive({
    Id: 0,   // 点位 id
    Tag: '', // 点位标识
    Description: '', // 点位描述
    RW_Cancel: '', // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
    Value_Type: '', // 输出类型
    Config: '',
    Creation_Time: '', // 创建时间
    Drive_Id: 0, // 驱动 id 唯一标识符
    Drive_Type: '', // 驱动类型
})

watch(() => UpdateItem.Config, (newValue, _) => {
    // 1. 按 : 分割成数组
    const parts = newValue.split(":");
    // 2. 取出最后一段（你要的内容）
    const lastStr = (parts.length > 0 ? parts[parts.length - 1] : '') ?? '';
    switch (lastStr) {
        case 'bool':
            UpdateItem.Value_Type = 'bool'
            break
        case 'int8':
        case 'uint8':
        case 'int16':
        case 'uint16':
        case 'int32':
        case 'uint32':
        case 'int64':
        case 'uint64':
            UpdateItem.Value_Type = 'int'
            break
        case 'float32':
        case 'float64':
            UpdateItem.Value_Type = 'float'
            break
        default:
            break
    }

})
const addNewRow = () => {
    // 重置表单验证状态
    addFormRef.value?.clearValidate()

    Object.assign(UpdateItem, {
        Id: 0,
        Drive_Id: 0,
        Drive_Type: '',
        Name: '',
        Config: '',
        Type: '',
        Points_Length: 0,
        Collector_Id: 0,
        Creation_Time: '',
        Tag: '',
        Description: '',
        RW_Cancel: '',
        Value_Type: '',
    })
    showUpdateDialog.value = true
}

// 新增或修改数据
const UpdateNewRow = () => {
    // 修复点5: 正确使用表单实例进行验证
    if (!addFormRef.value) return

    addFormRef.value.validate((valid) => {
        if (!valid) {
            ElMessage.error('请完善表单信息')
            return
        }

        if (UpdateItem.Id === 0) {
            Points_Config__Add(UpdateItem).then(() => {
                ElMessage.success('添加成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        } else {
            Points_Config__Update(UpdateItem).then(() => {
                ElMessage.success('修改成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        }
    })
}

// 验证规则
const newItemRules = {
    Collector_Id: [
        { required: true, message: '请输入采集器标识', trigger: 'blur' },
        {
            pattern: /^[1-9]\d*$/,
            message: '请选择采集服务',
            trigger: 'blur',
        },
    ],
    Name: [
        { required: true, message: '请输入驱动名称', trigger: 'blur' },
        {
            pattern: /^.{1,23}$/, // 修改为至少1个字符
            message: '长度应在1-23个字符之间',
            trigger: 'blur',
        },
    ],
    Config: [
        { required: true, message: '请输入设备连接参数', trigger: 'blur' },
        {
            pattern: /^[0-9a-zA-Z.:]*$/,
            message: '请输入正确的配置格式: ip:port:其他配置参数',
            trigger: 'blur',
        },
    ],
    Type: [
        { required: true, message: '请选择驱动类型', trigger: 'change' }, // 下拉框建议用 change
    ],
}

const filterInput = (val: string) => {
    return val.replace(/[^0-9a-zA-Z.:]/g, '')
}

// 定义提示文本
const typeOptions: { [key: string]: string } = {
    "Modbus_Tcp": '格式：从机地址:功能码&lt;01 02 03 04&gt;:寄存器地址.子地址[如果有]:数据类型&lt;bool uint16 int16 uint32 int32 float32 float64&gt; <br>示例：1:03:1.1:bool<br>示例：1:03:2:int16<br>示例：1:03:3:uint32<br>示例：1:01:1:bool',
    "Modbus_Rtu": '格式：从机地址:功能码&lt;01 02 03 04&gt;:寄存器地址.子地址[如果有]:数据类型&lt;bool uint16 int16 uint32 int32 float32 float64&gt; <br>示例：1:03:1.1:bool<br>示例：1:03:2:int16<br>示例：1:03:3:uint32<br>示例：1:01:1:bool',
    "Siemens_S7": '格式：寄存器类型:DB1[其他类型为0]:寄存器地址.子地址[如果有]:数据类型&lt;bool uint16 int16 uint32 int32 float32 float64&gt; <br>示例：I:0:0.1:bool <br>示例：M:0:0.1:bool <br>示例：DB:1:1.0:bool <br>示例：DB:1:2:int8 <br>示例：DB:1:3:int16<br> 示例：DB:1:5:float32',
}

</script>

<style scoped>
.input-tip {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
}
</style>