<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h3>用户管理</h3>
            <el-table :data="authority_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="Id" max-width="70" />
                <el-table-column prop="Name" label="名称" max-width="50" />
                <el-table-column prop="Permissions" label="权限" max-width="70" />
                <el-table-column prop="Refresh_Token_Time" label="自动登出时间(s)" max-width="100" />
                <el-table-column prop="Phone" label="电话" max-width="100" />
                <el-table-column prop="Email" label="邮箱" max-width="100" />
                <el-table-column prop="Discontinued" label="禁用" max-width="50" />
                <el-table-column label="操作" width="170">
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

        <!-- 新增数据对话框 -->
        <el-dialog v-model="showAddDialog" title="新增用户" width="700px">
            <el-form :model="newItem" label-width="100px" ref="addFormRef" :rules="newItemRules">
                <!-- 用户名输入框 -->
                <el-form-item prop="Name" label="用户名">
                    <el-input v-model="newItem.Name" placeholder="请输入用户名" size="large" :prefix-icon="User" clearable />
                </el-form-item>

                <!-- 密码输入框 -->
                <el-form-item prop="Passwd" label="密码">
                    <el-input v-model="newItem.Passwd" type="password" placeholder="请输入密码" size="large"
                        :prefix-icon="Lock" show-password />
                </el-form-item>

                <el-form-item prop="Permissions" label="权限">
                    <el-input v-model.number="newItem.Permissions" type="Permissions" placeholder="请输入权限" size="large"
                        :prefix-icon="Lock" show-Permissions />
                </el-form-item>

                <el-form-item prop="Refresh_Token_Time" label="过期时间">
                    <el-input v-model.number="newItem.Refresh_Token_Time" type="Refresh_Token_Time"
                        placeholder="请输入过期时间设定（s）" size="large" :prefix-icon="Lock" show-Refresh_Token_Time />
                </el-form-item>

                <el-form-item prop="Phone" label="电话">
                    <el-input v-model="newItem.Phone" type="Phone" placeholder="请输入电话" size="large" :prefix-icon="Phone"
                        show-Phone />
                </el-form-item>

                <el-form-item prop="Email" label="邮箱">
                    <el-input v-model="newItem.Email" type="Email" placeholder="请输入邮箱" size="large"
                        :prefix-icon="Message" show-Message />
                </el-form-item>

                <el-form-item prop="Discontinued" label="停用">
                    <el-switch v-model="newItem.Discontinued" />
                </el-form-item>
            </el-form>
            <template #footer>
                <el-button @click="showAddDialog = false">取消</el-button>
                <el-button type="primary" @click="addNewRow">确定添加</el-button>
            </template>
        </el-dialog>
    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { User, Lock, Key, Phone, Message } from '@element-plus/icons-vue'
import { reactive, onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User__All_Count, User__All_Query, User__Set_Del, type User__all_table_type, type User__table_interface } from '@/utils/api'

const router = useRouter()

const authority_data: User__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
    add: false, // 增加中间确认值
})


// 分页查询 Page 页码
const Query = (Page: number) => {
    User__All_Query(((Page - 1) * pagination.Page_length) + 1, pagination.Page_length).then((Authority_table) => {
        authority_data.length = 0
        Object.assign(authority_data, Authority_table)
        authority_data.reverse()
    })
}

// 查询总条目
const Count = () => {
    // 查询总条目
    User__All_Count().then((Count) => {
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
        name: 'user',
        params: { User_Id: id }
    })
}

// 删除行
const deleteRow = (scope: any) => {
    const id: number = scope.row.Id ?? 0
    User__Set_Del(id)
}




// 增加用户

// 响应式数据 
const showAddDialog = ref(false)
// 新项目数据
const newItem: User__all_table_type = reactive({
    Id: 0, // 用户ID
    Name: '', // 用户名
    Passwd: '', // 密码
    Permissions: 0,   // 权限
    Refresh_Token_Time: 604800,  // 过期时间设定（s）
    Discontinued: true,    // 停用
    Phone: '',  // 电话
    Email: '', // 邮箱
})

// 新增用户
const addNewRow = () => {
    showAddDialog.value = true
}
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

<style scoped>
img {
    width: 100%;
    height: 100%;
}

.user-info-card {
    background: white;
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
    max-width: 100%;
    margin: 20px auto;
}

.avatar-section {
    text-align: center;
    margin-bottom: 20px;
}

.user-avatar {
    border: 3px solid #409EFF;
    margin-bottom: 10px;
}

.role-tag {
    margin-top: 8px;
}

.info-section {
    margin-bottom: 20px;
}

.user-name {
    font-size: 20px;
    font-weight: 600;
    color: #303133;
    margin: 0 0 16px 0;
    text-align: center;
}

.user-info {
    display: flex;
    align-items: center;
    margin-bottom: 15px;
    padding: 10px 0;
}

.user-info .info-set {
    color: #409EFF;
    cursor: pointer;
    /* margin-left: 12px; */
    border: none;
}

.user-info:last-child {
    border-bottom: none;
    margin-bottom: 0;
}

.info-label {
    min-width: 60px;
    color: #606266;
    font-size: 0.9rem;
    margin-right: 8px;
    flex-shrink: 0;
}

.info-value {
    color: #303133;
    /* font-size: 0.95rem; */
    font-weight: 30;
    min-width: 180px;
    word-break: break-word;
    flex-grow: 0;
    margin-right: 20px;
}


.status-section {
    border-top: 1px solid #e4e7ed;
    padding-top: 16px;
}

.status-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
}

.status-item .label {
    color: #909399;
    font-size: 13px;
}

.status-item .value {
    color: #303133;
    font-size: 13px;
}

.user-info .el-icon {
    margin-right: 8px;
}
</style>