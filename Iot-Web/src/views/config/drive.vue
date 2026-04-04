<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h3>用户管理</h3>
            <el-table :data="config_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" min-width="80" />
                <el-table-column prop="Name" label="名称" min-width="100" />
                <el-table-column prop="Permissions" label="权限" min-width="100" />
                <el-table-column prop="Refresh_Token_TTL" label="自动登出时间 (s)" min-width="140" />
                <el-table-column prop="Phone" label="电话" min-width="140" />
                <el-table-column prop="Email" label="邮箱" min-width="200" />
                <el-table-column prop="Discontinued" label="禁用" min-width="80" />
                <el-table-column label="操作" width="180" fixed="right">
                    <template #default="scope">
                        <el-button size="small" @click="editRow(scope)">编辑</el-button>
                        <el-button size="small" type="danger" @click="deleteRow(scope)">删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
            <div style="margin-top: 20px">
                <!-- <el-button type="primary" @click="addNewRow">新增数据</el-button> -->
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


    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { User, Lock, Key, Phone, Message } from '@element-plus/icons-vue'
import { reactive, onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Drive_Config__Count, Drive_Config__Query, Drive_Config__Add, Drive_Config__Update, Drive_Config__Del, type Drive_Config__table_interface } from '@/api/config_service'

const router = useRouter()

const config_data: Drive_Config__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
    add: false, // 增加中间确认值
})


// 分页查询 Page 页码
const Query = (Page: number) => {
    Drive_Config__Query({
        Page: Page,
        Page_Size: pagination.Page_length
    }).then((config_info) => {
        config_data.length = 0
        Object.assign(config_data, config_info)
    })
}

// 查询总条目
const Count = () => {
    // 查询总条目
    Drive_Config__Count().then((Count) => {
        pagination.total_length = Count
        Query(1)
    })
}

onMounted(() => {
    Count()
})


// 分页
// 每页显示条目个数 改变执行
const handleSizeChange = (value: number) => {
    pagination.Page_length = value
    Query(1)
}
// 页数 改变执行
const handleCurrentChange = (value: number) => {
    console.log(value)
    Query(value)
}

// 编辑行
const editRow = (scope: any) => {
    const id: number = scope.row.Id
    router.push({
        name: 'info',
        params: { User_Id: id }
    })
}

// 删除行
const deleteRow = (scope: any) => {
    const id: number = scope.row.Id ?? 0
    if (id === 0) {
        ElMessage.error('无效的用户ID')
        return
    }
    Drive_Config__Del(id).then(() => {
        ElMessage.success('删除成功')
        Count()
    }).catch((error) => {
        console.error('删除失败:', error)
        ElMessage.error('删除失败')
    })
}




// 新增用户
// const addNewRow = () => {
//     showAddDialog.value = true
// }
// 验证规则
const newItemRules = {
    Name: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        {
            pattern: /^.{0,23}$/,
            message: '太长了',
            trigger: 'blur'
        }
    ],
    Passwd: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        {
            pattern: /^(?=.*\d)(?=.*[!@#$%&])[A-Za-z\d!@#$%&]{8}$/,
            message: '请输入大于8位,包含大小写、数字和 ! @ # $ % & 特殊符号',
            trigger: 'blur'
        }
    ],
    Permissions: [
        { required: true, message: '请输入权限', trigger: 'blur' },
        {
            type: 'number',
            message: '权限必须为数字',
            trigger: 'blur'
        },
        {
            validator: (rule: any, value: number, callback: (arg0: Error | undefined) => void) => {
                if (value < 0) {
                    callback(new Error('权限不能为负数'))
                } else if (value > 5000) {
                    callback(new Error('权限不能超过5000'))
                } else {
                    callback(undefined)
                }
            },
            trigger: 'blur'
        }
    ],
    Refresh_Token_Time: [
        { required: true, message: '请输入过期时间(s)', trigger: 'blur' },
        {
            type: 'number',
            message: '过期时间必须为数字',
            trigger: 'blur'
        },
        {
            validator: (rule: any, value: number, callback: (arg0: Error | undefined) => void) => {
                if (value < 0) {
                    callback(new Error('过期时间不能为负数'))
                } else if (value > 604800) { // 1年
                    callback(new Error('过期时间不能超过1年'))
                } else {
                    callback(undefined)
                }
            },
            trigger: 'blur'
        }
    ],
    Phone: [
        // { required: true, message: '请输入电话', trigger: 'blur' },
        {
            pattern: /^1[3-9]\d{9}$/,
            message: '请输入有效电话',
            trigger: 'blur'
        }
    ],
    Email: [
        // { required: true, message: '请输入邮箱', trigger: 'blur' },
        {
            pattern: /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/,
            message: '请输入有效邮箱',
            trigger: 'blur'
        }
    ]
}

</script>

<style scoped></style>