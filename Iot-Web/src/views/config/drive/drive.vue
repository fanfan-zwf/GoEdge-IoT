<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <el-table :data="config_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" width="60" align="center" />
                <el-table-column prop="Collector.Name" label="采集器名称" min-width="120" show-overflow-tooltip />
                <el-table-column prop="Name" label="名称" max-width="120" show-overflow-tooltip />
                <el-table-column prop="Type" label="类型" min-width="60" align="center" />
                <el-table-column prop="Config" label="配置" min-width="200" show-overflow-tooltip />
                <el-table-column prop="Points_Length" label="点位数量" width="100" align="center" />
                <el-table-column prop="Creation_Time" label="创建时间" width="230" align="center" />
                <el-table-column label="操作" width="200" fixed="right">
                    <template #default="scope">
                        <el-button size="small" @click="viewPoints(scope)">点位</el-button>
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
                        layout="total, sizes, prev, pager, next, jumper" :pager-count=7
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

                    <el-form-item prop="Collector.Id" label="采集服务" v-if="UpdateItem.Id === 0">
                        <Search_Collector :result="(value) => { UpdateItem.Collector.Id = value.Id; }" />
                    </el-form-item>

                    <!-- <el-form-item prop="Collector_Id" label="采集器标识" v-if="UpdateItem.Id === 0">
                            <el-input v-model.number="UpdateItem.Collector_Id" type="number" placeholder="请输入采集器标识"
                                size="large" />
                        </el-form-item> -->

                    <el-form-item prop="Type" label="驱动类型">
                        <el-select v-model="UpdateItem.Type" placeholder="请选择驱动类型" :disabled="UpdateItem.Id !== 0"
                            style="width: 100%">
                            <el-option label="Modbus_Tcp" value="Modbus_Tcp" />
                            <el-option label="Modbus_Rtu" value="Modbus_Rtu" />
                            <el-option label="西门子s7" value="Siemens_S7" />
                        </el-select>
                    </el-form-item>

                    <el-form-item prop="Name" label="驱动名称">
                        <el-input v-model="UpdateItem.Name" type="text" placeholder="请输入驱动名称" size="large" clearable />
                    </el-form-item>

                    <el-form-item prop="Config" label="连接配置">
                        <el-input v-model="UpdateItem.Config" placeholder="请输入设备连接参数" size="large" autocomplete="off"
                            clearable disabled />
                    </el-form-item>

                    <DynamicConfigForm v-model="UpdateItem.Config" :field-rules="myRules[UpdateItem.Type] ?? []"
                        :UpdateItem="UpdateItem" />
                </el-form>
            </template>
            <template #footer>
                <el-button @click="UpdateItem.Collector.Id = 0; showUpdateDialog = false">取消</el-button>
                <el-button type="primary" @click="UpdateNewRow">确定</el-button>
            </template>
        </el-dialog>
    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { reactive, onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, ElMessageBox } from 'element-plus'
import {
    Drive_Config__Count,
    Drive_Config__Query,
    Drive_Config__Add,
    Drive_Config__Update,
    Drive_Config__Del,
    type Drive_Config__table_interface,
    type Drive_Config__add_interface
} from '@/api/config_service'
import Search_Collector from '@/views/config/collector/search_collector.vue'
import DynamicConfigForm, { type DynamicFieldItem } from '@/components/Custom_Form.vue'
import { c } from 'naive-ui'


const router = useRouter()

const config_data: Drive_Config__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
})

// 分页查询 Page 页码
const Query = (Page: number) => {
    Drive_Config__Query({
        Page: Page,
        Page_Size: pagination.Page_length
    }).then((config_info) => {
        config_data.length = 0
        Object.assign(config_data, config_info)
    }).catch((error) => {
        ElMessage.error(error)
    })
}

// 查询总条目
const Count = () => {
    Drive_Config__Count().then((Count) => {
        pagination.total_length = Count
        Query(1)
    }).catch((error) => {
        ElMessage.error(error)
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

    ElMessageBox.prompt(`确定要删除 <span style="color:#ff0000; font-size:14px">${scope.row.Name ?? ''}</span> 驱动吗？ 输入驱动名称以确认删除。`,
        '警告', {
        confirmButtonText: '确定',
        confirmButtonType: 'danger',
        cancelButtonText: '取消',
        inputPattern: new RegExp(`^${scope.row.Name}$`),
        inputErrorMessage: '输入内容不正确',
        dangerouslyUseHTMLString: true,
    })
        .then(({ }) => {
            Drive_Config__Del(id).then(() => {
                ElMessage.success('删除成功')
                Count()
            }).catch((error) => {
                console.error('删除失败:', error)
                ElMessage.error('删除失败')
            })
        })
        .catch(() => {
            ElMessage.info('已取消输入')
        })
}

// 跳转当前点位配置页面
const viewPoints = (scope: any) => {
    const id: number = scope.row.Id ?? 0
    if (id === 0) {
        ElMessage.error('无效的ID')
        return
    }
    router.push({
        name: 'point_config',
        params: { name: scope.row.Name ?? '' },
        query: {
            drive_id: id,
            drive_type: scope.row.Type ?? '',
            collector_uuid: scope.row.Collector.Uuid ?? ''
        }
    }
    )
}

// 响应式数据 
const showUpdateDialog = ref(false)

// 修复点4: 定义表单 ref
// const addFormRef = ref<FormInstance>()
const addFormRef = ref<FormInstance | null>(null)

// 新项目数据
const UpdateItem: Drive_Config__table_interface = reactive({
    Id: 0,
    Name: '',
    Config: '',
    Type: '',
    Points_Length: 0,
    Creation_Time: '',
    Collector: {
        Id: 0,
        Name: '',
        Uuid: '',
    }
})

const addNewRow = () => {
    // 重置表单验证状态
    addFormRef.value?.clearValidate()

    Object.assign(UpdateItem, {
        Id: 0,
        Name: '',
        Config: '',
        Type: '',
        Points_Length: 0,
        Creation_Time: '',
        Collector: {
            Id: 0,
            Name: '',
            Uuid: '',
        }
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
            Drive_Config__Add({
                Name: UpdateItem.Name,                 // 驱动名称
                Config: UpdateItem.Config,             // json 配置参数
                Type: UpdateItem.Type,                 // 驱动类型
                Collector_Id: UpdateItem.Collector.Id, // 采集器标识
            }).then(() => {
                ElMessage.success('添加成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        } else {
            Drive_Config__Update({
                Id: UpdateItem.Id,        // 驱动 id
                Name: UpdateItem.Name,    // 驱动名称
                Config: UpdateItem.Config // json 配置参数
            }).then(() => {
                ElMessage.success('修改成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        }
    })
}


const formRules = {
    number: [
        { required: true, message: '请输入数字', trigger: 'blur' },
        {
            pattern: /^[1-9]\d*$/,
            message: '请输入有效的数字',
            trigger: 'blur',
        },
    ],
}
// 验证规则
const newItemRules = {

    'Collector.Id': [
        { required: true, message: '请选择驱动', trigger: 'blur' },
        {
            validator: (rule: any, value: any, callback: any) => {
                // 必须是数字，且必须 > 0
                if (typeof UpdateItem.Collector.Id === 'number' && UpdateItem.Collector.Id > 0) {
                    callback()
                } else {
                    callback(new Error('请选择有效的驱动'))
                }
            },
            trigger: 'blur'
        }
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
            pattern: /^[0-9a-zA-Z.;]*$/,
            message: '请输入正确的配置格式: ip;port;其他配置参数',
            trigger: 'blur',
        },
    ],
    Type: [
        { required: true, message: '请选择驱动类型', trigger: 'change' }, // 下拉框建议用 change
    ],
    Ip: [
        { required: true, message: '请输入设备ip', trigger: 'blur' },
        {
            pattern: /^(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)$/,
            message: '请输入正确ip',
            trigger: 'blur',
        },
    ],
    Port: [
        { required: true, message: '请输入设备端口', trigger: 'blur' },
        {
            pattern: /^([1-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])$/,
            message: '请输入正确的端口号',
            trigger: 'blur',
        },
    ],
    ConnectionType: [
        { required: true, message: '请输入设备连接类型', trigger: 'blur' },
        {
            pattern: /^(PG|OP|Basic)$/,
            message: '请输入正确的连接类型,默认是OP',
            trigger: 'blur',
        },
    ],
    PortName: [
        { required: true, message: '请输入设备串口号', trigger: 'blur' },
        {
            pattern: /^(COM\d+|\/dev\/(ttyS\d+|ttyUSB\d+|ttyACM\d+|ttyAMA\d+))$/,
            message: '请输入正确的串口号',
            trigger: 'blur',
        },
    ],
    Timeout: formRules.number,
    Interval: formRules.number,
    Response: formRules.number,
    RetryTime: formRules.number,
    PollTime: formRules.number,
    Rack: formRules.number,
    Slot: formRules.number,
    Packet_max: formRules.number,
}

const myRules: { [key: string]: DynamicFieldItem[] } = {
    "Modbus_Tcp": [
        { prop: 'Ip', label: 'IP地址', type: 'string', placeholder: '请输入设备IP地址' },
        { prop: 'Port', label: '端口', type: 'number', placeholder: '请输入设备端口，默认502' },
        { prop: 'RetryTime', label: '重试间隔', type: 'unit', unitType: 's', placeholder: '请输入重试间隔，默认12s' },
        { prop: 'Timeout', label: '连接超时', type: 'unit', unitType: 's', placeholder: '请输入超时时间，默认3s' },
        { prop: 'Interval', label: '响应超时', type: 'unit', unitType: 'ms', placeholder: '请输入间隔时间，默认20ms' },
        { prop: 'Response', label: '轮询间隔', type: 'unit', unitType: 'ms', placeholder: '请输入响应时间，默认200ms' },
        { prop: 'Packet_max', label: '组包字节个数', type: 'unit', placeholder: '请输入组包字节个数，默认64' },
   {
            prop: 'concurrency', label: '并发', type: 'select',
            options: [
                { label: 'true', value: 'true' },
                { label: 'false', value: 'false' }
            ]
        },
    ],
    "Modbus_Rtu": [
        { prop: 'PortName', label: '串口号', type: 'string', placeholder: '请输入串口号' },
        { prop: 'Timeout', label: '超时时间', type: 'unit', unitType: 's', placeholder: '请输入超时时间，默认3s' },
        { prop: 'Interval', label: '间隔时间', type: 'unit', unitType: 'ms', placeholder: '请输入间隔时间，默认20ms' },
        { prop: 'Response', label: '响应时间', type: 'unit', unitType: 'ms', placeholder: '请输入响应时间，默认200ms' },
        { prop: 'Packet_max', label: '组包字节个数', type: 'unit', placeholder: '请输入组包字节个数，默认64' }
    ],
    "Siemens_S7": [
        { prop: 'Ip', label: 'IP地址', type: 'string', placeholder: '请输入设备IP地址' },
        { prop: 'Port', label: '端口', type: 'number', placeholder: '请输入设备端口，默认502' },
        {
            prop: 'ConnectionType', label: '连接类型', type: 'select',
            options: [
                { label: 'PG', value: 'PG' },
                { label: 'OP', value: 'OP' },
                { label: 'Basic', value: 'Basic' }
            ]
        },
        { prop: 'Rack', label: '机架号', type: 'number', placeholder: '请输入机架号' },
        { prop: 'Slot', label: '槽号', type: 'number', placeholder: '请输入槽号' },
        { prop: 'Timeout', label: '超时时间', type: 'unit', unitType: 's', placeholder: '请输入超时时间，默认3s' },
        { prop: 'RetryTime', label: '重试时间', type: 'unit', unitType: 's', placeholder: '请输入重试时间，默认10s' },
        { prop: 'PollTime', label: '轮询时间', type: 'unit', unitType: 'ms', placeholder: '请输入轮询时间，默认100ms' }
    ]
}
</script>

<style scoped>
.input-tip {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
}
</style>

<!-- 非 scoped 样式，用于处理 Teleport 到 body 的 el-dialog -->
<style></style>