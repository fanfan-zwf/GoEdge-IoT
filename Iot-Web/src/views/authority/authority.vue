<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h3>权限创建</h3>
            <el-table :data="authority_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="权限Id" max-width="70" />
                <el-table-column label="名称" max-width="50">
                    <template #default="scope">
                        <div v-if="scope.row.editing">
                            <el-input v-model="scope.row.Name" size="small" />
                        </div>
                        <div v-else>
                            {{ scope.row.Name }}
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="主题" max-width="70">
                    <template #default="scope">
                        <div v-if="scope.row.editing">
                            <el-input v-model="scope.row.Theme" size="small" />
                        </div>
                        <div v-else>
                            {{ scope.row.Theme }}
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="说明" max-width="300">
                    <template #default="scope">
                        <div v-if="scope.row.editing">
                            <el-input v-model="scope.row.Explain" size="small" />
                        </div>
                        <div v-else>
                            {{ scope.row.Explain }}
                        </div>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="170">
                    <template #default="scope">
                        <div v-if="scope.row.editing">
                            <el-button size="small" @click="saveRow(scope)">保存</el-button>
                            <el-button size="small" @click="cancelEdit(scope)">取消</el-button>
                        </div>
                        <div v-else>
                            <el-button size="small" @click="editRow(scope)">编辑</el-button>
                            <el-popconfirm title="确认删除这个权限吗？" @confirm="function () { deleteRow(scope) }">
                                <template #reference>
                                    <el-button size="small" type="danger">删除</el-button>
                                </template>
                            </el-popconfirm>
                        </div>
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
    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Authority__Count, Authority__Query, Authority__Add, Authority__Update, Authority__Del, type Authority__table_interface } from '@/typer/api'


const authority_data: Authority__table_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
    add: false, // 增加中间确认值
})


// 分页查询 Page 页码
const Query = (Page: number) => {
    Authority__Query(((Page - 1) * pagination.Page_length) + 1, pagination.Page_length).then((Authority_table) => {
        authority_data.length = 0
        Object.assign(authority_data, Authority_table)
    })
}

// 查询总条目
const Count = () => {
    // 查询总条目
    Authority__Count().then((Count) => {
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
    scope.row.editing = true
    // 保存原始数据用于取消编辑时恢复
    scope.row.originalData = { ...scope.row }
}

// 保存行
const saveRow = async (scope: any) => {
    pagination.add = false
    scope.row.editing = false
    scope.row.originalData = null
    const Authority: Authority__table_interface = {
        Id: scope.row.Id, // 权限ID
        Name: scope.row.Name, // 权限名称
        Theme: scope.row.Theme,   // 权限主题
        Explain: scope.row.Explain,    // 说明
    }

    const a = authority_data[scope.$index]
    if (a == undefined) {
        console.log(" authority_data[scope.$index] == undefined", authority_data[scope.$index])
        return
    }
    if (scope.row.Id == 0) {
        Authority__Add(Authority).then(() => {
            pagination.total_length += 1
        })
    } else {
        Authority__Update(Authority)
    }
    // Count()

    // 这里可以添加保存到后端的逻辑
}

// 取消编辑
const cancelEdit = (scope: any) => {
    if (scope.row.originalData) {
        Object.assign(scope.row, scope.row.originalData)
    }
    scope.row.editing = false
    scope.row.originalData = null

    if (scope.row.Id == 0) {
        authority_data.splice(0, 1)
        pagination.add = false
    }
}

// 删除行
const deleteRow = (scope: any) => {
    const id = authority_data[scope.$index]?.Id ?? -1
    if (id == -1) {
        ElMessage.error('找不到下标')
        return
    }
    // 调用接口删除
    Authority__Del(id).then(() => {
        // 删除成功重新加载
        authority_data.splice(scope.$index, 1)
        pagination.total_length = -1
        // Count()
    })
}

// 新增行
const addNewRow = () => {
    if (pagination.add) {
        ElMessage.warning('已增加，请输入')
        return
    }
    pagination.add = true
    const newRow = {
        Id: 0,
        Name: '', // 权限名称
        Theme: '',    // 权限主题
        Explain: '',    // 说明
        editing: true,
    }
    authority_data.unshift(newRow)

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