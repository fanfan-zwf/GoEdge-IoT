<template>
    <div class="search-add-table">
        <!-- 搜索和操作区域 -->
        <div class="table-header">
            <div class="search-section">
                <el-input v-model="searchText" placeholder="搜索姓名、部门、职位..." style="width: 300px" clearable
                    @input="handleSearch">
                    <template #prefix>
                        <el-icon>
                            <Search />
                        </el-icon>
                    </template>
                </el-input>

                <el-select v-model="filterDepartment" placeholder="部门筛选" clearable
                    style="width: 150px; margin-left: 10px;">
                    <el-option label="技术部" value="技术部" />
                    <el-option label="销售部" value="销售部" />
                    <el-option label="人事部" value="人事部" />
                    <el-option label="财务部" value="财务部" />
                </el-select>

                <el-button type="primary" @click="showAddDialog = true" style="margin-left: 10px;">
                    <el-icon>
                        <Plus />
                    </el-icon>
                    新增员工
                </el-button>
            </div>

            <!-- 搜索统计 -->
            <div class="search-stats" v-if="searchText || filterDepartment">
                <el-tag type="info">
                    搜索到 {{ filteredData.length }} 条结果
                    <el-button type="text" @click="clearSearch" style="margin-left: 5px;">清除筛选</el-button>
                </el-tag>
            </div>
        </div>

        <!-- 数据表格 -->
        <el-table :data="filteredData" style="width: 100%" v-loading="loading"
            :empty-text="searchText ? '未找到匹配的数据' : '暂无数据'">
            <el-table-column type="index" label="序号" width="60" />
            <el-table-column prop="name" label="姓名" width="120">
                <template #default="scope">
                    <span class="name-cell" :class="{ 'highlight': shouldHighlight(scope.row, 'name') }">
                        {{ scope.row.name }}
                    </span>
                </template>
            </el-table-column>
            <el-table-column prop="age" label="年龄" width="80" />
            <el-table-column prop="gender" label="性别" width="80">
                <template #default="scope">
                    <el-tag :type="scope.row.gender === '男' ? 'primary' : 'danger'">
                        {{ scope.row.gender }}
                    </el-tag>
                </template>
            </el-table-column>
            <el-table-column prop="department" label="部门">
                <template #default="scope">
                    <span :class="{ 'highlight': shouldHighlight(scope.row, 'department') }">
                        {{ scope.row.department }}
                    </span>
                </template>
            </el-table-column>
            <el-table-column prop="position" label="职位">
                <template #default="scope">
                    <span :class="{ 'highlight': shouldHighlight(scope.row, 'department') }">
                        {{ scope.row.position }}
                    </span>
                </template>
            </el-table-column>
            <el-table-column prop="email" label="邮箱" width="200" />
            <el-table-column prop="phone" label="电话" width="130" />
            <el-table-column prop="joinDate" label="入职日期" width="120" />
            <el-table-column label="操作" width="150" fixed="right">
                <template #default="scope">
                    <el-button size="small" @click="editItem(scope.row)">编辑</el-button>
                    <el-button size="small" type="danger" @click="deleteItem(scope.row.id)">删除</el-button>
                </template>
            </el-table-column>
        </el-table>

        <!-- 新增数据对话框 -->
        <el-dialog v-model="showAddDialog" title="新增员工" width="700px">
            <el-form :model="newItem" label-width="100px" :rules="formRules" ref="addFormRef">
                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item label="姓名" prop="name">
                            <el-input v-model="newItem.name" placeholder="请输入姓名" />
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item label="年龄" prop="age">
                            <el-input-number v-model="newItem.age" :min="18" :max="65" style="width: 100%" />
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item label="性别" prop="gender">
                            <el-radio-group v-model="newItem.gender">
                                <el-radio label="男">男</el-radio>
                                <el-radio label="女">女</el-radio>
                            </el-radio-group>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item label="部门" prop="department">
                            <el-select v-model="newItem.department" placeholder="请选择部门" style="width: 100%">
                                <el-option label="技术部" value="技术部" />
                                <el-option label="销售部" value="销售部" />
                                <el-option label="人事部" value="人事部" />
                                <el-option label="财务部" value="财务部" />
                                <el-option label="市场部" value="市场部" />
                            </el-select>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-row :gutter="20">
                    <el-col :span="12">
                        <el-form-item label="职位" prop="position">
                            <el-input v-model="newItem.position" placeholder="请输入职位" />
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item label="电话" prop="phone">
                            <el-input v-model="newItem.phone" placeholder="请输入电话" />
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-form-item label="邮箱" prop="email">
                    <el-input v-model="newItem.email" placeholder="请输入邮箱" />
                </el-form-item>

                <el-form-item label="入职日期" prop="joinDate">
                    <el-date-picker v-model="newItem.joinDate" type="date" placeholder="选择日期" style="width: 100%" />
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="showAddDialog = false">取消</el-button>
                <el-button type="primary" @click="addNewItem">确定添加</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Plus } from '@element-plus/icons-vue'

// 模拟初始数据
const initialData = [
    {
        id: 1,
        name: '张三',
        age: 25,
        gender: '男',
        department: '技术部',
        position: '前端工程师',
        email: 'zhangsan@company.com',
        phone: '138-0013-8000',
        joinDate: '2023-01-15'
    },
    {
        id: 2,
        name: '李四',
        age: 28,
        gender: '女',
        department: '销售部',
        position: '销售经理',
        email: 'lisi@company.com',
        phone: '138-0013-8001',
        joinDate: '2022-08-20'
    },
    {
        id: 3,
        name: '王五',
        age: 32,
        gender: '男',
        department: '人事部',
        position: 'HR主管',
        email: 'wangwu@company.com',
        phone: '138-0013-8002',
        joinDate: '2021-03-10'
    },
    {
        id: 4,
        name: '赵六',
        age: 29,
        gender: '女',
        department: '财务部',
        position: '财务专员',
        email: 'zhaoliu@company.com',
        phone: '138-0013-8003',
        joinDate: '2022-05-12'
    },
    {
        id: 5,
        name: '钱七',
        age: 26,
        gender: '男',
        department: '技术部',
        position: '后端工程师',
        email: 'qianqi@company.com',
        phone: '138-0013-8004',
        joinDate: '2023-03-01'
    }
]

// 响应式数据
const tableData = ref([])
const searchText = ref('')
const filterDepartment = ref('')
const showAddDialog = ref(false)
const loading = ref(false)
const addFormRef = ref()

// 新项目数据
const newItem = reactive({
    name: '',
    age: 25,
    gender: '男',
    department: '',
    position: '',
    email: '',
    phone: '',
    joinDate: ''
})

// 表单验证规则
const formRules = {
    name: [
        { required: true, message: '请输入姓名', trigger: 'blur' },
        { min: 2, max: 10, message: '姓名长度在 2 到 10 个字符', trigger: 'blur' }
    ],
    age: [
        { required: true, message: '请输入年龄', trigger: 'blur' }
    ],
    department: [
        { required: true, message: '请选择部门', trigger: 'change' }
    ],
    email: [
        { required: true, message: '请输入邮箱', trigger: 'blur' },
        { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
    ],
    phone: [
        { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号码', trigger: 'blur' }
    ]
}

// 计算属性 - 过滤数据
const filteredData = computed(() => {
    let result = tableData.value

    // 部门筛选
    if (filterDepartment.value) {
        result = result.filter(item => item.department === filterDepartment.value)
    }

    // 关键词搜索
    if (searchText.value) {
        const keyword = searchText.value.toLowerCase()
        result = result.filter(item =>
            Object.values(item).some(value =>
                String(value).toLowerCase().includes(keyword)
            )
        )
    }

    return result
})

// 高亮显示匹配的文本
const shouldHighlight = (row, field) => {
    if (!searchText.value) return false
    const value = String(row[field]).toLowerCase()
    return value.includes(searchText.value.toLowerCase())
}

// 搜索处理
const handleSearch = () => {
    loading.value = true
    // 模拟搜索延迟
    setTimeout(() => {
        loading.value = false
    }, 300)
}

// 清除搜索
const clearSearch = () => {
    searchText.value = ''
    filterDepartment.value = ''
}

// 添加新项目
const addNewItem = async () => {
    try {
        // 表单验证
        await addFormRef.value.validate()

        // 生成新ID
        const newId = Math.max(...tableData.value.map(item => item.id), 0) + 1

        // 格式化日期
        const formattedDate = newItem.joinDate ?
            new Date(newItem.joinDate).toISOString().split('T')[0] :
            new Date().toISOString().split('T')[0]

        // 添加到表格数据
        const newEmployee = {
            id: newId,
            ...newItem,
            joinDate: formattedDate
        }

        tableData.value.unshift(newEmployee)

        // 重置表单
        resetForm()

        // 关闭对话框
        showAddDialog.value = false

        ElMessage.success('员工添加成功')

        // 如果当前有搜索条件，自动清除以便显示新添加的数据
        if (searchText.value || filterDepartment.value) {
            setTimeout(() => {
                clearSearch()
            }, 1000)
        }
    } catch (error) {
        ElMessage.error('请完善表单信息')
    }
}

// 重置表单
const resetForm = () => {
    Object.assign(newItem, {
        name: '',
        age: 25,
        gender: '男',
        department: '',
        position: '',
        email: '',
        phone: '',
        joinDate: ''
    })
}

// 编辑项目
const editItem = (item) => {
    ElMessage.info(`编辑 ${item.name}`)
    // 编辑逻辑...
}

// 删除项目
const deleteItem = async (id) => {
    try {
        await ElMessageBox.confirm('确定要删除这条数据吗？', '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        })

        const index = tableData.value.findIndex(item => item.id === id)
        if (index !== -1) {
            tableData.value.splice(index, 1)
            ElMessage.success('删除成功')
        }
    } catch {
        ElMessage.info('已取消删除')
    }
}

// 初始化数据
onMounted(() => {
    loading.value = true
    setTimeout(() => {
        tableData.value = [...initialData]
        loading.value = false
    }, 500)
})
</script>

<style scoped>
.search-add-table {
    padding: 20px;
}

.table-header {
    margin-bottom: 20px;
}

.search-section {
    display: flex;
    align-items: center;
    margin-bottom: 10px;
}

.search-stats {
    margin-top: 10px;
}

.name-cell {
    font-weight: 600;
    color: #409EFF;
}

.highlight {
    background-color: #fff566;
    padding: 2px 4px;
    border-radius: 2px;
}

:deep(.el-table .cell) {
    padding: 8px 12px;
}

:deep(.el-table .highlight .cell) {
    background-color: #fff566;
}
</style>