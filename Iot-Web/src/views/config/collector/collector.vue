<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <el-table :data="config_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" width="60" align="center" />
                <el-table-column prop="Name" label="名称" min-width="120" show-overflow-tooltip />
                <el-table-column prop="Label" label="标识" min-width="120" show-overflow-tooltip />
                <el-table-column prop="Sn" label="Sn" width="160" align="center" />
                <el-table-column prop="User_Id" label="创建用户Id" width="100" align="center" />
                <el-table-column prop="Version" label="版本" width="230" align="center" />
                <el-table-column prop="Creation_Time" label="创建时间" min-width="200" show-overflow-tooltip />
                <el-table-column prop="Last_Activity_Time" label="最后活动时间" width="230" align="center" />
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
            destroy-on-close>
            <!-- 动态宽度控制 -->
            <template #default>
                <div style="width: 100%; max-width: 95%; margin: 0 auto;">
                    <el-form :model="UpdateItem" label-width="100px" ref="addFormRef" :rules="newItemRules">
                        <el-form-item prop="Label" label="标识" v-if="UpdateItem.Id === 0">
                            <el-input v-model="UpdateItem.Label" placeholder="请输入采集器标识" size="large" />
                        </el-form-item>

                        <el-form-item prop="Uuid" label="Uuid" v-if="UpdateItem.Id === 0">
                            <el-input v-model="UpdateItem.Uuid" placeholder="请输入采集器Uuid" size="large" />
                        </el-form-item>

                        <el-form-item prop="Name" label="名称">
                            <el-input v-model="UpdateItem.Name" placeholder="请输入采集器名称" size="large" />
                        </el-form-item>

                    </el-form>
                </div>
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
import { reactive, onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus' // 引入 FormInstance 类型
// 修复点3: 移除未使用的 naive-ui 导入
// import { c } from 'naive-ui' 
import {
    Collector_Info__Count,
    Collector_Info__Query,
    Collector_Info__Add,
    Collector_Info__Del,
    Collector_Info__Update,
    type Collector_Info__table_interface,
    type Collector_Info__Add_interface,
} from '@/api/config_service'
import { useUserStore } from '@/stores/user'

const UserStore = useUserStore() // 获取用户信息
const router = useRouter()

const config_data: Collector_Info__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
})

// 分页查询 Page 页码
const Query = (Page: number) => {
    Collector_Info__Query({
        Page: Page,
        Page_Size: pagination.Page_length
    }).then((config_info) => {
        config_data.length = 0
        Object.assign(config_data, config_info)
    })
}

// 查询总条目
const Count = () => {
    Collector_Info__Count().then((Count) => {
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
    Collector_Info__Del(id).then(() => {
        ElMessage.success('删除成功')
        Count()
    }).catch((error) => {
        console.error('删除失败:', error)
        ElMessage.error('删除失败')
    })
}

// 响应式数据 
const showUpdateDialog = ref(false)

// 修复点4: 定义表单 ref
const addFormRef = ref<FormInstance>()

// 新项目数据
const UpdateItem: Collector_Info__table_interface = reactive({
    Id: 0,      // 采集 Id
    Label: '',    // 标识
    Creation_Time: '',// 创建时间
    Uuid: '',    // Uuid (修正为 string)
    Sn: '',    // 设备 sn
    User_Id: 0,        // 用户 id
    User_Name: '',        // 用户 id
    Version: '',    // 版本
    Last_Activity_Time: '', // 最后活动时间
    Equipment_Id: 0,        // 设备 id
    Name: ''    // 设备名称
})

const addNewRow = () => {
    // 重置表单验证状态
    addFormRef.value?.clearValidate()

    Object.assign(UpdateItem, {
        Label: '', // 标识
        Uuid: '', // Uuid
        User_Id: UserStore.Id,  // 用户 id
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
            Collector_Info__Add(UpdateItem).then(() => {
                ElMessage.success('添加成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                console.error('添加失败:', error)
                ElMessage.error('添加失败')
            })
        } else {
            Collector_Info__Update(UpdateItem).then(() => {
                ElMessage.success('修改成功')
                showUpdateDialog.value = false
                Count()
            }).catch((error) => {
                console.error('修改失败:', error)
                ElMessage.error('修改失败')
            })
        }
    })
}

// 验证规则
const newItemRules = {
    Label: [
        { required: true, message: '请输入标识', trigger: 'blur' },
        {
            pattern: /^.{1,37}$/, // 修改为至少1个字符
            message: '长度应在1-37个字符之间',
            trigger: 'blur',
        },
    ],
    Name: [
        { required: true, message: '请输入名称', trigger: 'blur' },
        {
            pattern: /^.{1,23}$/, // 修改为至少1个字符
            message: '长度应在1-23个字符之间',
            trigger: 'blur',
        },
    ],
    Uuid: [
        { required: true, message: '请输入采集器Uuid', trigger: 'blur' },
        {
            pattern: /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/,
            message: 'Uuid格式不正确',
            trigger: 'blur',
        },
    ],

}



const filterInput = (val: string) => {
    return val.replace(/[^0-9a-zA-Z.:]/g, '')
}

// const tipText = computed(() => {
//     const type = UpdateItem.Type
//     if (type === "Modbus_Tcp") {
//         return '格式：IP:端口:连接超时:响应超时:间隔时间:字节长度，例如 192.168.1.1:502:3s:200ms:1s:8'
//     }
//     if (type === "Modbus_Rtu") {
//         return '格式：串口号:连接超时:响应超时:间隔时间:字节长度，例如 com1:3s:200ms:1s:8'
//     }
//     if (type === "Siemens_S7Comm") {
//         return '格式：IP:端口:连接类型<PG OP[默认] Basic>:机架号:槽号:超时时间:重试时间:轮询时间 192.168.1.1:502:OP:0:1:3s:10s:100ms'
//     }
//     return ''
// })
</script>

<style scoped>
.input-tip {
    margin-top: 4px;
    font-size: 12px;
    color: #909399;
}

/* 手机 < 768px 时自动变小 */
@media (max-width: 768px) {

    /* 优化对话框在移动端的显示 */
    :deep(.el-dialog) {
        width: 95% !important;
        max-width: 100% !important;
        margin-top: 5vh !important;
    }

    :deep(.el-dialog__body) {
        padding: 15px 10px;
    }

    /* 关键修改：移动端改为垂直布局，防止溢出 */
    :deep(.el-form-item) {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
    }

    :deep(.el-form-item__label) {
        width: 100% !important;
        text-align: left;
        padding-bottom: 4px;
        line-height: 1.5;
        font-size: 14px;
    }

    :deep(.el-form-item__content) {
        margin-left: 0 !important;
        width: 100%;
    }

    /* 优化输入框和选择框大小 */
    :deep(.el-input),
    :deep(.el-select) {
        width: 100%;
    }

    :deep(.el-input__inner),
    :deep(.el-textarea__inner) {
        font-size: 14px;
        padding: 8px 12px;
    }

    /* 优化底部按钮 */
    :deep(.el-dialog__footer) {
        padding: 10px;
        display: flex;
        justify-content: space-between;
    }

    :deep(.el-button) {
        padding: 8px 15px;
        font-size: 14px;
        flex: 1;
        margin: 0 5px;
    }

    /* 提示文字适配 */
    .input-tip {
        font-size: 11px;
        line-height: 1.4;
        word-break: break-all;
    }

}


/* 屏幕宽度小于800px时，调整.el-dialog宽度为80% */
@media (max-width: 800px) {
    :deep(.el-dialog) {
        width: 80% !important;
        max-width: 95% !important;
        margin-top: 5vh !important;
    }

    :deep(.el-dialog__body) {
        padding: 15px 10px;
    }

    /* 优化输入框宽度 */
    :deep(.el-input),
    :deep(.el-select) {
        width: 100%;
    }

    :deep(.el-input__inner),
    :deep(.el-textarea__inner) {
        font-size: 14px;
        padding: 8px 12px;
    }
}

/* 更小的手机 < 480px */
@media (max-width: 480px) {
    :deep(.el-dialog) {
        /* --el-dialog-width: 50% !important; */
        margin-top: 0 !important;
        height: 50vh;
        max-height: 100vh;
        display: flex;
        flex-direction: column;
        border-radius: 0;
    }

    :deep(.el-dialog__header) {
        padding: 15px;
        margin-right: 0;
        border-bottom: 1px solid #ebeef5;
    }

    :deep(.el-dialog__body) {
        flex: 1;
        overflow-y: auto;
        padding: 15px;
    }

    :deep(.el-dialog__footer) {
        border-top: 1px solid #ebeef5;
        padding: 15px;
    }

    :deep(.el-form-item__label) {
        font-size: 13px;
    }
}
</style>