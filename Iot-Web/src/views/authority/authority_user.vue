<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h3>权限管理</h3>
            <el-table :data="authority_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="用户权限Id" max-width="30" />
                <el-table-column fixed label="操作" min-width="65">
                    <template #default="scope">
                        <el-switch v-model.boolean="scope.row.Enable" class="ml-2"
                            style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                            @click="click_switch(scope)" />
                    </template>
                </el-table-column>
                <el-table-column prop="User_Id" label="用户Id" max-width="30" />
                <el-table-column prop="User.Name" label="用户名称" max-width="70" />
                <el-table-column prop="Authority_Id" label="权限Id" max-width="30" />
                <el-table-column prop="Authority.Name" label="权限名称" max-width="70" />
                <el-table-column prop="Authority.Theme" label="权限主题" min-width="120" max-width="170" />
                <el-table-column label="操作" width="170">
                    <template #default="scope">
                        <el-popconfirm title="确认删除这个用户权限吗？" @confirm="function () { deleteRow(scope) }">
                            <template #reference>
                                <el-button size="small" type="danger">删除</el-button>
                            </template>
                        </el-popconfirm>
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


    <!-- 新增数据对话框 -->
    <el-dialog v-model="showAddDialog" title="新增用户权限" width="700px">
        <el-form :model="newItem" label-width="100px" ref="addFormRef">
            <!-- <el-form-item label="Id" prop="Id">
                <el-input v-model.number="newItem.Id" placeholder="请输入Id" /> 
            </el-form-item> -->
            <el-form-item label="搜索用户名称" prop="User_Id">
                <span> {{ `用户Id: ${newItem.User_Id}; 用户名称: ${newItem.User_Name}` }}</span>
                <user_search :choice="user_choice" />
            </el-form-item>
            <el-form-item label="搜索权限名称" prop="Authority_Id">
                <span> {{ `权限Id: ${newItem.Authority_Id}; 权限名称: ${newItem.Authority_Name}` }}</span>
                <authority_search :choice="authority_choice" />
            </el-form-item>
            <el-form-item label="使能" prop="Enable">
                <el-switch v-model.boolean="newItem.Enable" class="ml-2"
                    style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949" />
            </el-form-item>
        </el-form>

        <template #footer>
            <el-button @click="showAddDialog = false">取消</el-button>
            <el-button type="primary" @click="addNewItem">确定添加</el-button>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
    Authority_User__All_Count, Authority_User__All_Query, Authority_User__Del,
    Authority_User__Add, User__Get_Info_Array,
    Authority__Id_Array,
    Authority_User__Enable,
    type Authority_User__table_interface,
    type Authority__table_interface,
    type User__table_interface,
} from '@/typer/api'
import authority_search from '@/views/authority/authority_search.vue'
import user_search from '@/views/user/user_search.vue'

export interface Authority_User_interface extends Authority_User__table_interface {
    User: User__table_interface
    Authority: Authority__table_interface
}

const authority_data: Authority_User_interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
    add: false, // 增加中间确认值
})


// 分页查询 Page 页码
const Query = (Page: number) => {
    Authority_User__All_Query(((Page - 1) * pagination.Page_length) + 1, pagination.Page_length).then((Authority_table) => {
        authority_data.length = 0
        Object.assign(authority_data, Authority_table)
        authority_data.reverse()

        if (authority_data.length == 0) {
            return
        }

        let AuthorityId_Array: number[] = []
        let User_Id_Array: number[] = []
        for (let Authority of Authority_table) {
            User_Id_Array.push(Authority.User_Id)
            AuthorityId_Array.push(Authority.Authority_Id)
        }

        User__Get_Info_Array(User_Id_Array).then((User_table) => {
            for (let User of User_table) {
                for (let authority of authority_data) {
                    if (authority.User_Id == User.Id) {
                        authority.User = User
                        continue
                    }
                }
            }
        })

        Authority__Id_Array(AuthorityId_Array).then((Authority_table) => {
            for (let Authority of Authority_table) {
                for (let authority of authority_data) {
                    if (authority.Authority_Id == Authority.Id) {
                        authority.Authority = Authority
                        continue
                    }
                }
            }
        })
    })
}

// 查询总条目
const Count = () => {
    // 查询总条目
    Authority_User__All_Count().then((Count) => {
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
    Query(value)
}


// 删除行
const deleteRow = (scope: any) => {
    // 调用接口删除
    console.log(scope.row.Id)
    Authority_User__Del(scope.row.Id).then(() => {
        // 删除成功重新加载
        authority_data.splice(scope.$index, 1)
        pagination.total_length -= 1
        // Count()
    })
}

// 使能
const click_switch = (scope: any) => {
    Authority_User__Enable(scope.row.Id, scope.row.Enable).catch((error: any) => {
        scope.row.Enable != scope.row.Enable
    })
}


// 增加数据

// 响应式数据 
const showAddDialog = ref(false)
// 新项目数据
const newItem = reactive({
    Id: 0,
    User_Id: 0,
    User_Name: '',
    Authority_Id: 0,
    Authority_Name: '',
    Enable: true,
})

// 新增行
const addNewRow = () => {
    showAddDialog.value = true
}

// 用户搜索返回
const user_choice = (User: User__table_interface) => {
    newItem.User_Id = User.Id
    newItem.User_Name = User.Name
}

// 选择搜索返回
const authority_choice = (Authority: Authority__table_interface) => {
    newItem.Authority_Id = Authority.Id
    newItem.Authority_Name = Authority.Name
}


// 添加新项目
const addNewItem = async () => {
    try {
        const Authority_data: Authority_User__table_interface = {
            Id: 0,
            User_Id: newItem.User_Id,
            Authority_Id: newItem.Authority_Id,
            Enable: newItem.Enable,
        }
        Authority_User__Add(Authority_data).then(() => {
            Count()
        })

    } catch (error) {
        ElMessage.error('请完善表单信息')
    }
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