<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h6 v-if="drive_name && drive_name !== 'undefined'">驱动名称：{{ drive_name }}</h6>
            <el-table :data="config_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" width="60" align="center" />
                <el-table-column prop="Tag" label="点位标签" min-width="200" max-width="300" show-overflow-tooltip />
                <el-table-column prop="Config" label="配置" min-width="200" max-width="300" show-overflow-tooltip />
                <el-table-column prop="Drive.Type" label="驱动类型" width="130" align="center" show-overflow-tooltip />
                <el-table-column prop="Creation_Time" label="创建时间" width="230" align="center" />
                <el-table-column prop="Drive.Name" label="驱动名称" min-width="120" show-overflow-tooltip />
                <el-table-column prop="Collector.Name" label="采集器名称" min-width="120" show-overflow-tooltip />
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
                        layout="total, sizes, prev, pager, next, jumper" :pager-count=7 :total="pagination.total_length"
                        @size-change="handleSizeChange" @current-change="handleCurrentChange" />
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

                    <el-form-item prop="Drive.Id" label="驱动" v-if="UpdateItem.Id === 0 && drive_name === ''">
                        <search_drive :result="(value: Drive_Config__table_interface) => {
                            UpdateItem.Drive.Id = value.Id;
                            UpdateItem.Drive.Type = value.Type;
                            UpdateItem.Collector.Uuid = value.Collector.Uuid;
                            UpdateItem.Tag = `//${value.Collector.Uuid}//${value.Name}/`;
                        }" />
                    </el-form-item>

                    <el-form-item prop="Tag" label="标识符">
                        <el-input v-model="UpdateItem.Tag" type="text" placeholder="请输入标识符" size="large" />
                    </el-form-item>

                    <el-form-item prop="Config" label="点位参数">
                        <el-input v-model="UpdateItem.Config" placeholder="请输入点位参数" size="large" autocomplete="off"
                            clearable disabled />
                    </el-form-item>

                    <el-form-item prop="RW_Cancel" label="读写方式">
                        <el-select v-model="UpdateItem.RW_Cancel" placeholder="请选择驱动类型" style="width: 100%">
                            <!-- <el-option label="禁用" value="N" /> -->
                            <el-option label="只读" value="R" />
                            <el-option label="读写" value="R/W" selected />
                            <el-option label="只写" value="W" />
                        </el-select>
                    </el-form-item>

                    <el-form-item prop="Description" label="描述">
                        <el-input v-model="UpdateItem.Description" type="textarea" clearable
                            @clear="handleCustomClear" />
                        <div class="input-tip">
                            <!-- <el-button @clear="handleCustomClear" key="primary" type="primary" >
                                清空
                            </el-button> 请输入:null -->
                        </div>
                    </el-form-item>

                    <el-divider />

                    <DynamicConfigForm v-model="UpdateItem.Config" :field-rules="myRules[UpdateItem.Drive.Type] ?? []"
                        :UpdateItem="UpdateItem" />
                </el-form>
            </template>
            <template #footer>
                <el-button @click="UpdateItem.Drive.Id = 0;

                showUpdateDialog = false">取消</el-button>
                <el-button type="primary" @click="UpdateNewRow">确定</el-button>
            </template>
        </el-dialog>
    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { reactive, onMounted, ref, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance, ElMessageBox } from 'element-plus'
import {
    Points_Config__Count,
    Points_Config__Query,
    Points_Config__Add,
    Points_Config__Update,
    Points_Config__Del,
    type Points_Config__table_interface,
    type Drive_Config__table_interface,
    type Points_Config__add_interface
} from '@/api/config_service'
import search_drive from '@/views/config/drive/search_drive.vue'
import DynamicConfigForm, { type DynamicFieldItem } from '@/components/Custom_Form.vue'

const route = useRoute()
// const router = useRouter()

const drive_name = ref<string>(String(route.params.name) ?? '')
const drive_id = ref<number>(Number(route.query.drive_id ?? 0) ?? 0)

const config_data: Points_Config__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
})

// 分页查询 Page 页码
const Query = (Page: number) => {
    const params: any = {
        Page: Page,
        Page_Size: pagination.Page_length
    };

    if (drive_id.value !== 0) {
        params.Drive_Id = [drive_id.value];
    }
    Points_Config__Query(params).then((config_info) => {
        config_data.length = 0
        Object.assign(config_data, config_info)
    }).catch((error) => {
        ElMessage.error(error)
    })
}

// 查询总条目
const Count = () => {
    // 构建请求参数，如果 drive_id 为 0 则不传 Drive_Id
    const params: any = {};

    if (drive_id.value !== 0) {
        params.Drive_Id = [drive_id.value];
    }
    Points_Config__Count(params).then((Count) => {
        pagination.total_length = Count
        Query(1)
    }).catch((error) => {
        ElMessage.error(error)
    })
}

onMounted(() => {
    Count()
})

watch(
    () => route, // 监听整个路由
    () => {
        // 路由一变，就重新赋值
        drive_name.value = String(route.params.name ?? '')
        drive_id.value = Number(route.query.drive_id ?? 0) || 0
        Count() // 重新查询数据
    },
    { immediate: true, deep: true } // 进入页面立即执行一次
)
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
    Object.assign(UpdateItem, scope.row);
    nextTick(() => {
        addFormRef.value?.clearValidate(); // ✅ 编辑时清除旧校验
    });
    showUpdateDialog.value = true;
};
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
const UpdateItem: Points_Config__table_interface = reactive({
    Id: 0,   // 点位 id
    Tag: '', // 点位标识
    Description: '', // 点位描述
    RW_Cancel: 'R', // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
    Value_Type: '', // 输出类型
    Config: '',
    Creation_Time: '', // 创建时间 
    // 修复点：补充 Drive 对象中缺失的 Type 属性
    Drive: {
        Id: 0,
        Name: '',
        Type: '', // <--- 添加此行以匹配 Drive__Carry_interface
    },
    Collector: {
        Id: 0,
        Name: '',
        Uuid: '',
        // 注意：如果 Collector 对应的接口也有必填字段缺失，请在此处一并补充
    },
})

watch(() => UpdateItem.Config, (newValue, _) => {
    // 1. 按 ; 分割成数组
    const parts = newValue.split(";");
    // 2. 取出最后一段（你要的内容） 
    const lastStr = (parts[parts.length - 1]) ?? '';
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
            console.error('未知类型:', lastStr, parts)
            break
    }

}, { deep: true })
const addNewRow = () => {
    addFormRef.value?.clearValidate();
    Object.assign(UpdateItem, {
        Id: 0,   // 点位 id
        Tag: `//${String(route.query.collector_uuid)}//${drive_name.value}/`, // 点位标识
        Description: '', // 点位描述
        RW_Cancel: 'R', // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
        Value_Type: '', // 输出类型
        Config: '',
        Creation_Time: '', // 创建时间 
        // 修复点：补充 Drive 对象中缺失的 Type 属性
        Drive: {
            Id: drive_id.value ?? 0,
            Name: drive_name.value ?? '',
            Type: String(route.query.drive_type) ?? '', // <--- 添加此行以匹配 Drive__Carry_interface
        },
        Collector: {
            Id: 0,
            Name: '',
            Uuid: '',
            // 注意：如果 Collector 对应的接口也有必填字段缺失，请在此处一并补充
        },
    });
    showUpdateDialog.value = true;
};

const UpdateNewRow = () => {
    if (!addFormRef.value) return

    addFormRef.value.validate((valid) => {
        if (!valid) {
            ElMessage.error('请完善表单信息')
            return
        }
        if (UpdateItem.Id === 0) {
            // 此时 payload符合 Points_Config__add_interface
            Points_Config__Add({
                Drive_Id: UpdateItem.Drive.Id,   // 点位 id 唯一标识符 
                Drive_Type: UpdateItem.Drive.Type, // 驱动类型
                Id: UpdateItem.Id,   // 点位 id
                Tag: UpdateItem.Tag, // 点位标识
                Description: UpdateItem.Description, // 点位描述
                RW_Cancel: UpdateItem.RW_Cancel, // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
                Value_Type: UpdateItem.Value_Type, // 输出类型
                Config: UpdateItem.Config
            }).then(() => {
                ElMessage.success('添加成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                ElMessage.error(error)
            })
        } else {
            Points_Config__Update({
                Id: UpdateItem.Id,
                Tag: UpdateItem.Tag,
                Description: UpdateItem.Description,
                RW_Cancel: UpdateItem.RW_Cancel,
                Value_Type: UpdateItem.Value_Type,
                Config: UpdateItem.Config,
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

// 自定义清空逻辑
const handleCustomClear = () => {
    UpdateItem.Description = "null"
}



const formRules = {
    Child_Address_Exist: [
        { required: true, message: '请输入点位参数', trigger: 'blur' },
        {
            // 自定义校验器
            validator: (rule: any, value: any, callback: any) => {

                // 1. 按 ; 分割成数组
                const parts = UpdateItem.Config.split(";");
                // 2. 取出最后一段（你要的内容） 
                const lastStr = (parts[parts.length - 1]) ?? '';

                const isFloatEnable = lastStr === 'bool';
                const val = String(value).trim();

                // 1. 不能是负数
                if (val.startsWith('-')) {
                    return callback(new Error('不能输入负数'));
                }

                if (isFloatEnable) {
                    // ------------------------------
                    // 允许小数：格式 数字.0~7
                    // ------------------------------
                    const reg = /^\d+\.[0-7]$/;
                    if (reg.test(val)) {
                        callback();
                    } else {
                        callback(new Error('格式不正确：类型是bool时，必须带子地址，并且子地址只能0-7'));
                    }
                } else {
                    // ------------------------------
                    // 不允许小数：必须纯整数
                    // ------------------------------
                    const reg = /^\d+$/;
                    if (reg.test(val)) {
                        callback();
                    } else {
                        callback(new Error('格式不正确：不能带小数，必须是纯整数'));
                    }
                }
            },
            trigger: 'blur'
        },
    ],
}

// 验证规则
// 优化后：无BUG、稳定、适配你真实业务
const newItemRules = reactive({
    // Drive_Id: [
    //     { required: true, message: '请选择驱动', trigger: 'blur' },
    //     {
    //         pattern: /^[1-9]\d*$/,
    //         message: '请选择有效的驱动',
    //         trigger: 'blur'
    //     },
    // ],
    'Drive.Id': [
        { required: true, message: '请选择驱动', trigger: 'blur' },
        {
            validator: (rule: any, value: any, callback: any) => {
                // 必须是数字，且必须 > 0
                if (typeof UpdateItem.Drive.Id === 'number' && UpdateItem.Drive.Id > 0) {
                    callback()
                } else {
                    callback(new Error('请选择有效的驱动'))
                }
            },
            trigger: 'blur'
        }
    ],
    Tag: [
        { required: true, message: '请输入标识符', trigger: 'blur' },
        {
            pattern: /^\/\/[a-zA-Z0-9\u4e00-\u9fa5_-]+\/\/[a-zA-Z0-9\u4e00-\u9fa5_-]+(\/[a-zA-Z0-9\u4e00-\u9fa5_-]+)*$/,
            message: '格式不正确',
            trigger: 'blur'
        },
    ],
    Config: [
        { required: true, message: '请输入点位参数', trigger: 'blur' },
        {
            pattern: /^[0-9a-zA-Z.;]*$/,
            message: '格式不正确',
            trigger: 'blur'
        },
    ],
    RW_Cancel: [
        { required: true, message: '请选择读写方式', trigger: 'change' },
    ],
    Modbus__SlaveID: [
        { required: true, message: '请输入从机地址', trigger: 'change' },
        {
            pattern: /^(?:[1-9]\d?|1\d{2}|2[0-3]\d|24[0-7])$/,
            message: '格式不正确',
            trigger: 'blur'
        },
    ],
    Function: [
        { required: true, message: '请选择功能码', trigger: 'change' }, // 下拉框建议用 change
    ],
    Type: [
        { required: true, message: '请选择值类型', tr高哦igigger: 'change' },
    ],
    Byte_Order: [
        { required: true, message: '请选择值类型', trigger: 'change' },
    ],
    Siemens_S7__Register_Type: [
        { required: true, message: '请选择寄存器类型', trigger: 'change' },
    ],
    Siemens_S7__DB_ID: [
        { required: true, message: '请输入DB编号', trigger: 'change' },
    ],
    Address: formRules.Child_Address_Exist,
    Siemens_S7__Value_Type: [
        { required: true, message: '请选择值类型', trigger: 'change' },
    ]
});

const myRules: { [key: string]: DynamicFieldItem[] } = {
    "Modbus_Tcp": [
        { prop: 'Modbus__SlaveID', label: '从机地址', type: 'unit', placeholder: '请输入从机地址' },
        {
            prop: 'Function', label: '功能码', type: 'select',
            options: [
                { label: '01', value: '01' },
                { label: '02', value: '02' },
                { label: '03', value: '03' },
                { label: '04', value: '04' }
            ]
        },
        { prop: 'Address', label: '寄存器地址', type: 'string', placeholder: '请输入寄存器地址' },
        {
            prop: 'Byte_Order', label: '字节序', type: 'select',
            hidden: () => {
                const parts = UpdateItem.Config.split(';');
                const funcCode = parts[1] || '';
                return funcCode === '01' || funcCode === '02';
            },
            options: [
                { label: 'AB', value: 'AB' },
                { label: 'BA', value: 'BA' },
                { label: 'ABCD', value: 'ABCD' },
                { label: 'ABDC', value: 'ABDC' },
                { label: 'BACD', value: 'BACD' },
                { label: 'DCBA', value: 'DCBA' }
            ]
        },
        {
            prop: 'Type', label: '数据类型', type: 'select',
            options: [
                { label: 'bool', value: 'bool' },
                { label: 'uint16', value: 'uint16' },
                { label: 'int16', value: 'int16' },
                { label: 'uint32', value: 'uint32' },
                { label: 'int32', value: 'int32' },
                { label: 'float32', value: 'float32' }
            ]
        },

    ],
    "Modbus_Rtu": [
        { prop: 'Modbus__SlaveID', label: '从机地址', type: 'unit', placeholder: '请输入从机地址' },
        {
            prop: 'Function', label: '功能码', type: 'select',
            options: [
                { label: '01', value: '01' },
                { label: '02', value: '02' },
                { label: '03', value: '03' },
                { label: '04', value: '04' }
            ]
        },
        { prop: 'Address', label: '寄存器地址', type: 'string', placeholder: '请输入寄存器地址' },
        {
            prop: 'Byte_Order',
            label: '字节序',
            type: 'select',
            hidden: () => {
                const parts = UpdateItem.Config.split(';');
                const funcCode = parts[1] || '';
                return funcCode === '01' || funcCode === '02';
            },
            options: [
                { label: 'AB', value: 'AB' },
                { label: 'BA', value: 'BA' },
                { label: 'ABCD', value: 'ABCD' },
                { label: 'ABDC', value: 'ABDC' },
                { label: 'BACD', value: 'BACD' },
                { label: 'DCBA', value: 'DCBA' }
            ]
        },
        {
            prop: 'Type', label: '数据类型', type: 'select',
            options: [
                { label: 'bool', value: 'bool' },
                { label: 'uint16', value: 'uint16' },
                { label: 'int16', value: 'int16' },
                { label: 'uint32', value: 'uint32' },
                { label: 'int32', value: 'int32' },
                { label: 'float32', value: 'float32' }
            ]
        },
    ],
    "Siemens_S7": [
        {
            prop: 'Siemens_S7__Register_Type', label: '寄存器类型', type: 'select',
            options: [
                { label: 'I', value: 'I' },
                { label: 'Q', value: 'Q' },
                { label: 'M', value: 'M' },
                { label: 'DB', value: 'DB' }
            ]
        },
        { prop: 'Siemens_S7__DB_ID', label: 'DB编号', type: 'number', placeholder: '请输入DB编号', hidden: () => UpdateItem.Config.split(';')[0] !== 'DB' },
        { prop: 'Address', label: '寄存器地址', type: 'string', placeholder: '请输入寄存器地址' },
        {
            prop: 'Siemens_S7__Value_Type', label: '值类型', type: 'select',
            options: [
                { label: 'bool', value: 'bool' },
                { label: 'uint8', value: 'uint8' },
                { label: 'int8', value: 'int8' },
                { label: 'uint16', value: 'uint16' },
                { label: 'int16', value: 'int16' },
                { label: 'uint32', value: 'uint32' },
                { label: 'int32', value: 'int32' },
                { label: 'float32', value: 'float32' }
            ]
        },
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