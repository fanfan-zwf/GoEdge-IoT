<template>
    <el-config-provider :locale="zhCn">
        <div class="user-info-card">
            <h3>用户分组</h3>
            <el-table :data="Group_User_data" style="width: 100%" max-height="800px">
                <el-table-column fixed prop="Id" label="用户分组Id" max-width="30" />
                <el-table-column prop="User.Id" label="用户Id" max-width="30" />
                <el-table-column prop="User.Name" label="用户名称" max-width="70" />
                <el-table-column label="管理员" min-width="65">
                    <template #default="scope">
                        <el-switch v-model.boolean="scope.row.Administrator" class="ml-2"
                            style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949"
                            @click="click_switch(scope)" />
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="170">
                    <template #default="scope">
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
    </el-config-provider>


    <!-- 新增数据对话框 -->
    <el-dialog v-model="showAddDialog" title="新增用户权限" width="700px">
        <el-form :model="newItem" label-width="100px" ref="addFormRef">
            <el-form-item label="搜索用户名称" prop="User_Id">
                <span> {{ `用户Id: ${newItem.User_Id}; 用户名称: ${newItem.User.Name}` }}</span>
                <user_search :choice="user_choice" />
            </el-form-item>
        </el-form>
        <el-form-item label="管理员" prop="Enable">
            <el-switch v-model.boolean="newItem.Administrator" class="ml-2"
                style="--el-switch-on-color: #13ce66; --el-switch-off-color: #ff4949" />
        </el-form-item>
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
import { useRoute, useRouter } from 'vue-router'
import {
    Group_User__Count, Group_User__Query, Group_User__Add, Group_User__Del, Group_User__Administrator,
    User__Get_Info_Array,
    type Group_User__table_interface,
    type User__table_interface,
} from '@/typer/api'
import user_search from '@/views/user/user_search.vue'


const route = useRoute()
const group_user__id = ref<number>(0)
group_user__id.value = Number(route.params.group_user__id)


export interface Group_User__interface extends Group_User__table_interface {
    User: User__table_interface
}

const Group_User_data: Group_User__interface[] = reactive([])
const pagination = reactive({
    Page_length: 10, // 每页数量
    total_length: 0, // 总数量
    add: false, // 增加中间确认值
})


// 分页查询 Page 页码
const Query = (Page: number) => {
    Group_User__Query(group_user__id.value, ((Page - 1) * pagination.Page_length) + 1, pagination.Page_length).then((Authority_table) => {
        Group_User_data.length = 0
        Object.assign(Group_User_data, Authority_table)
        Group_User_data.reverse()

        if (Group_User_data.length == 0) {
            return
        }

        let User_Id_Array: number[] = []
        for (let Group_User of Group_User_data) {
            User_Id_Array.push(Group_User.User_Id)
        }

        User__Get_Info_Array(User_Id_Array).then((User_table) => {
            for (let User of User_table) {
                for (let Group_User of Group_User_data) {
                    if (Group_User.User_Id == User.Id) {
                        Group_User.User = User
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
    Group_User__Count(group_user__id.value).then((Count) => {
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
    Group_User__Del(scope.row.Id).then(() => {
        // 删除成功重新加载
        Group_User_data.splice(scope.$index, 1)
        pagination.total_length -= 1
        // Count()
    })
}

const click_switch = (scope: any) => {
    Group_User__Administrator(scope.row.Id, scope.row.Administrator).catch((error: any) => {
        scope.row.Enable != scope.row.Enable
    })
}


// 增加数据

// 响应式数据 
const showAddDialog = ref(false)
// 新项目数据
const newItem = reactive({
    Id: 0,               // 用户分组ID 
    User_Id: 0,           // 用户id 
    Group_Id: 0,          // 用户组id
    Administrator: false,    // 是否是管理员
    User: {
        Id: 0,  // 用户ID
        Name: '', // 用户名
        Permissions: 0,    // 权限
        Refresh_Token_Time: 0,    // 过期时间设定（s）
        Discontinued: false,    // 停用
        Phone: '',  // 电话
        Email: '',  // 邮箱
    }
})

// 新增行
const addNewRow = () => {
    showAddDialog.value = true
}

// 用户搜索返回
const user_choice = (User: User__table_interface) => {
    newItem.User_Id = User.Id
    newItem.User = User
}



// 添加新项目
const addNewItem = async () => {
    try {
        const Authority_data: Group_User__table_interface = {
            Id: 0,                                  // 用户分组ID 
            User_Id: newItem.User_Id,               // 用户id
            Group_Id: group_user__id.value,             // 用户组id
            Administrator: newItem.Administrator,   // 是否是管理员
        }

        Group_User__Add(Authority_data).then(() => {
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