<template>
    <el-dropdown trigger="click" @command="handleCommand" class="user-menu-dropdown">
        <span class="user-dropdown-trigger">
            <!-- 恢复使用标准的 el-avatar 组件 -->
            <el-avatar :size="36" :icon="user.Name ? undefined : 'User'" :name="user.Name" class="user-avatar-img">
                <!-- 如果有头像 URL 可以在这里通过 src 属性传入，目前使用名字首字母或默认图标 -->
                {{ user.Name ? user.Name.charAt(0).toUpperCase() : '' }}
            </el-avatar>
        </span>

        <template #dropdown>
            <!-- 修改：确保下拉卡片在移动端也有正确的背景色，防止继承黑色 -->
            <div class="dropdown-card" :class="{ 'mobile-card': isMobile }">
                <div class="dropdown-header">
                    <el-avatar size="48" icon="User" class="header-avatar" />
                    <div class="header-info">
                        <div class="header-main">
                            <span class="header-name">{{ user.Name || '未登录' }}</span>
                            <span class="header-badge">Pro</span>
                        </div>
                        <div class="header-email">{{ user.Email || '未设置邮箱' }}</div>
                    </div>
                </div>

                <el-dropdown-menu class="menu-list">
                    <el-dropdown-item command="profile" icon="User">个人中心</el-dropdown-item>
                    <el-dropdown-item command="docs" icon="Document">文档</el-dropdown-item>
                    <el-dropdown-item command="github" icon="Document">GitHub</el-dropdown-item>
                    <el-dropdown-item command="help" icon="Help">问题 & 帮助</el-dropdown-item>
                    <el-dropdown-item command="lock" icon="Lock">锁定屏幕</el-dropdown-item>
                    <el-dropdown-item divided command="logout" icon="SwitchButton">退出登录</el-dropdown-item>
                </el-dropdown-menu>
            </div>
        </template>
    </el-dropdown>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { User__Get_Info } from '@/typer/api'
import type { User__table_interface } from '@/typer/api'

const props = defineProps<{
    user: User__table_interface
}>()

const isMobile = ref(false)

const checkMobile = () => {
    isMobile.value = window.innerWidth < 768
}

onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
    window.removeEventListener('resize', checkMobile)
})

const router = useRouter()

const handleCommand = (command: string) => {
    switch (command) {
        case 'profile':
            router.push({ name: 'info', params: { User_Id: props.user.Id } })
            break
        case 'docs':
            window.open('https://www.your-docs-domain.com', '_blank')
            break
        case 'github':
            window.open('https://github.com', '_blank')
            break
        case 'help':
            ElMessage({ message: '请联系管理员或查看文档获取帮助', type: 'info' })
            break
        case 'lock':
            ElMessage({ message: '锁屏功能暂未实现', type: 'warning' })
            break
        case 'logout':
            localStorage.removeItem('F_Access_Token')
            localStorage.removeItem('F_Refresh_Token')
            sessionStorage.removeItem('F_User_Info')
            router.push({ name: 'login' })
            break
        default:
            break
    }
}
</script>

<style scoped>
.user-menu-dropdown {
    display: inline-flex;
    align-items: center;
}

.user-dropdown-trigger {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    /* 移除白色文字颜色，适应浅色背景 */
    color: #606266;
    transition: opacity 0.3s;
}

.user-dropdown-trigger:hover {
    opacity: 0.8;
}

.user-avatar-img {
    background-color: #409EFF;
    color: #fff;
    font-weight: bold;
    border: 1px solid #e6e6e6;
}

.dropdown-card {
    width: 220px;
    padding: 12px 0;
    /* 显式设置背景色为白色，防止在某些主题下变黑 */
    background-color: #ffffff;
    border-radius: 4px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 移动端下拉卡片适配 */
@media (max-width: 767px) {
    .dropdown-card.mobile-card {
        width: 280px;
        max-width: 90vw;
        position: fixed;
        right: 10px;
        max-height: 80vh;
        overflow-y: auto;
        /* 确保移动端下拉框背景为白色 */
        background-color: #ffffff !important;
    }
    
    .header-info {
        margin-left: 8px;
    }
    
    .header-name {
        font-size: 14px;
    }
    
    .header-email {
        font-size: 11px;
    }
}

.dropdown-header {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.08);
}

.header-avatar {
    background: rgba(64, 158, 255, 0.2);
}

.header-info {
    margin-left: 12px;
}

.header-name {
    font-weight: 700;
    color: rgba(0, 0, 0, 0.85);
}

.header-badge {
    background: rgba(64, 158, 255, 0.12);
    color: #409EFF;
    font-size: 10px;
    font-weight: 700;
    border-radius: 10px;
    padding: 2px 8px;
    margin-left: 8px;
}

.header-main {
    display: flex;
    align-items: center;
}

.header-email {
    font-size: 12px;
    color: rgba(0, 0, 0, 0.55);
}

.menu-list {
    padding: 4px 0;
}

.el-dropdown-menu__item {
    padding: 10px 16px;
}

.el-dropdown-menu__item:hover {
    background: rgba(64, 158, 255, 0.08);
}
</style>