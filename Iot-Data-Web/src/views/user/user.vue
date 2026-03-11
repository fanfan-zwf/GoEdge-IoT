<template>
    <div class="user-info-card">
        <!-- 头像区域 -->
        <div class="avatar-section">
            <el-avatar :size="80" :src="User_info.Name" class="user-avatar">
                {{ User_info.Name.charAt(0) }}
            </el-avatar>
            <br>
            <el-tag :type="getRoleType(User_info.Permissions)" class="role-tag">
                {{ getRole(User_info.Permissions) }}
            </el-tag>
        </div>

        <!-- 基本信息 -->
        <div class="info-section">
            <p class="user-info">
                <el-icon>
                    <img src="@/assets/icons/id card.svg" alt="用户Id" />
                </el-icon>
                <span class="info-label">Id </span>
                <span class="info-value">{{ User_info.Id || '错误' }}</span>
            </p>
            <p class="user-info">
                <el-icon>
                    <img src="@/assets/icons/用户名 (1).svg" alt="用户名" />
                </el-icon>
                <span class="info-label">用户名 </span>
                <span class="info-value">{{ User_info.Name || '错误' }}</span>
                <el-button class="info-set" plain @click="Set_Name">编辑</el-button>

            </p>
            <p class="user-info">
                <el-icon>
                    <img src="@/assets/icons/密码.svg" alt="密码" />
                </el-icon>
                <span class="info-label">密码</span>
                <span class="info-value">********</span>
                <el-button class="info-set" plain @click="Set_Passwd">编辑</el-button>
            </p>
            <p class="user-info">
                <el-icon>
                    <Phone />
                </el-icon>
                <span class="info-label">电话</span>
                <span class="info-value">{{ User_info.Phone || '未设置' }}</span>
                <el-button class="info-set" plain @click="Set_Phone">编辑</el-button>
            </p>
            <p class="user-info">
                <el-icon>
                    <Message />
                </el-icon>
                <span class="info-label">邮箱</span>
                <span class="info-value">{{ User_info.Email || '未设置' }}</span>
                <el-button class="info-set" plain @click="Set_Email">编辑</el-button>
            </p>
        </div>

        <!-- 状态信息 -->
        <!-- <div class="status-section">s
            <div class="status-item">
                <span class="label">状态</span>
                <el-tag :type="userInfo.status === 'active' ? 'success' : 'info'" size="small">
                    {{ userInfo.status === 'active' ? '在线' : '离线' }}
                </el-tag>
            </div>
            <div class="status-item">
                <span class="label">最后登录</span>
                <span class="value">{{ userInfo.lastLogin }}</span>
            </div>
        </div> -->
    </div>
</template>

<script setup lang="ts">
import { reactive, onMounted, watch, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Message, OfficeBuilding, Phone } from '@element-plus/icons-vue'
import { User__Get_Info, User__Set_Phone, User__Set_Email, User__Set_Passwd, User__Set_Name, type User__table_interface } from '@/utils/api'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const User_Id = ref<number>(0)
User_Id.value = Number(route.params.User_Id)

watch(User_Id, (_, oldValue) => {
    User__Get_Info(oldValue, true).then((User) => {
        console.log("获取用户信息", User)
        Object.assign(User_info, User)
    })
})

const getRoleType = (role: number) => {
    const roleMap: Record<number, string> = {
        0: 'danger',
        2: 'warning',
        3: 'success',
        4: 'info'
    }

    return roleMap[role] || 4
}

const getRole = (role: number) => {
    const roleMap: Record<number, string> = {
        0: '超级管理员',
        1: '管理员',
        2: '用户',
    }
    return roleMap[role] || '用户'
}

const User_info: User__table_interface = reactive({
    Id: 0, // 用户ID
    Name: '', // 用户名
    Permissions: 0,   // 权限
    Refresh_Token_Time: 0,   // 过期时间设定（s）
    Discontinued: false,    // 停用
    Phone: '',  // 电话
    Email: ''  // 邮箱
})

const Set_get = () => {
    User__Get_Info(User_Id.value, true).then((User) => {
        console.log("获取用户信息", User)
        Object.assign(User_info, User)
    })

}
onMounted(() => {
    User__Get_Info(User_Id.value).then((User) => {
        console.log("获取用户信息", User)
        Object.assign(User_info, User)
    })
})


// 设置用户名
const Set_Name = () => {
    ElMessageBox.prompt('请输入您的用户名', '设置用户名', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPattern:
            /^.{0,23}$/,
        inputErrorMessage: '无效用户名',
    })
        .then(({ value }) => {
            User__Set_Name(value, User_Id.value).then(() => {
                Set_get()
            })
        })
        .catch(() => {
            ElMessage({
                type: 'info',
                message: '取消设置用户名',
            })
        })
}


// 设置密码
const Set_Passwd = () => {
    ElMessageBox.prompt('请输入您的新密码', '设置密码', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValidator: (value) => {
            // 允许为空
            if (!value || value.trim() === '') {
                return true
            }
            // 有值时使用原来的正则验证
            if (! /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[!@#$%^&*])[A-Za-z\d!@#$%^&*]{8,}$/.test(value)) {
                return '无效电话号码'
            }
            return true
        },
        inputErrorMessage: '请输入大于8位,包含大小写、数字和特殊符号',
    })
        .then(({ value }) => {
            User__Set_Passwd(value, User_Id.value).then(() => {
                Set_get()
            })
        })
        .catch(() => {
            ElMessage({
                type: 'info',
                message: '取消设置密码',
            })
        })
}


// 设置电话
const Set_Phone = () => {
    ElMessageBox.prompt('请输入您的电话号码', '设置电话', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValidator: (value) => {
            // 允许为空
            if (!value || value.trim() === '') {
                return true
            }
            // 有值时使用原来的正则验证
            if (!/^1[3-9]\d{9}$/.test(value)) {
                return '无效电话号码'
            }
            return true
        },
        inputErrorMessage: '无效电话号码',
    })
        .then(({ value }) => {
            User__Set_Phone(value, User_Id.value).then(() => {
                Set_get()
            })
        })
        .catch(() => {
            ElMessage({
                type: 'info',
                message: '取消设置手机号码',
            })
        })
}

// 设置邮箱
const Set_Email = () => {
    ElMessageBox.prompt('请输入您的邮箱', '设置邮箱', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValidator: (value) => {
            // 允许为空
            if (!value || value.trim() === '') {
                return true
            }
            // 有值时使用原来的正则验证
            if (!/^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/.test(value)) {
                return '无效电话号码'
            }
            return true
        },
        inputErrorMessage: '无效邮箱地址',
    })
        .then(({ value }) => {
            User__Set_Email(value, User_Id.value).then(() => {
                Set_get()
            })
        })
        .catch(() => {
            ElMessage({
                type: 'info',
                message: '取消设置手机号码',
            })
        })
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
    max-width: 1200px;
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