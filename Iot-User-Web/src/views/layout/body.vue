<template>
    <div class="common-layout">
        <el-container>
            <!-- Header -->
            <el-header>
                <div v-if="isMobile" class="header-content">
                    <el-button :icon="Menu" @click="toggleAside" circle plain class="menu-button" />
                    <span class="header-title"> </span>
                </div>
            </el-header>

            <!-- 主要内容区域 -->
            <el-container class="main-container">
                <!-- 统一侧边栏 -->
                <transition name="aside-slide">
                    <el-aside v-if="asideVisible" :width="asideWidth"
                        :class="['aside-wrapper', { 'mobile-aside': isMobile }]">
                        <div class="aside-content">
                            <div class="aside-header">
                                <h3>导航菜单</h3>
                                <!-- <el-button v-if="!isMobile" :icon="Fold" @click="toggleAside" circle size="small"
                                    class="close-button" /> -->
                            </div>

                            <el-menu default-active="1" class="aside-menu" :collapse="isCollapsed"
                                @select="handleMenuSelect">
                                <router-link active-class="active"
                                    :to="{ name: 'user', params: { User_Id: User_info.Id } }">
                                    <el-menu-item index="1">
                                        <el-icon>
                                            <img src="@/assets/icons/账号信息.svg" alt="账号信息" />
                                        </el-icon>
                                        <template #title>账号信息</template>
                                    </el-menu-item>
                                </router-link>

                                <el-sub-menu index="2" v-if="User_info.Permissions == 0">
                                    <template #title>
                                        <el-icon>
                                            <img src="@/assets/icons/log.svg" alt="系统日志" />
                                        </el-icon>
                                        <span>权限管理</span>
                                    </template>
                                    <router-link active-class="active" :to="{ name: 'authority_user' }">
                                        <el-menu-item index="2-1">用户权限</el-menu-item>
                                    </router-link>
                                    <router-link active-class="active" :to="{ name: 'authority' }">
                                        <el-menu-item index="2-2">权限创建</el-menu-item>
                                    </router-link>
                                </el-sub-menu>


                                <router-link active-class="active" :to="{ name: 'group' }">
                                    <el-menu-item index="3" v-if="User_info.Permissions == 0">
                                        <el-icon>
                                            <img src="@/assets/icons/20gl-userGroup.svg" alt="分组管理" />
                                        </el-icon>
                                        <template #title>分组管理</template>
                                    </el-menu-item>
                                </router-link>
                                <router-link active-class="active" :to="{ name: 'user_account' }">
                                    <el-menu-item index="4" v-if="User_info.Permissions == 0">
                                        <el-icon>
                                            <img src="@/assets/icons/用户管理.svg" alt="用户管理" />
                                        </el-icon>
                                        <template #title>用户管理</template>
                                    </el-menu-item>
                                </router-link>
                                <el-sub-menu index="5" v-if="User_info.Permissions == 0">
                                    <template #title>
                                        <el-icon>
                                            <img src="@/assets/icons/log.svg" alt="系统日志" />
                                        </el-icon>
                                        <span>系统日志</span>
                                    </template>
                                    <el-menu-item index="5-1">我的</el-menu-item>
                                    <el-menu-item index="5-2"></el-menu-item>
                                </el-sub-menu>
                            </el-menu>

                            <div class="aside-footer">
                                <el-button v-if="!isMobile" :icon="isCollapsed ? Expand : Fold" @click="toggleCollapse"
                                    circle size="small" class="collapse-button" />
                            </div>
                        </div>
                    </el-aside>
                </transition>

                <!-- 移动端遮罩层 -->
                <div v-if="isMobile && asideVisible" class="aside-mask" @click="toggleAside"></div>

                <!-- Main 和 Footer 滚动区域 -->
                <div class="scrollable-area" ref="scrollContainer">
                    <div class="content-area">
                        <el-main>
                            <!-- <div class="main-content">
                                <div class="breadcrumb">
                                <el-breadcrumb separator="/">
                                    <el-breadcrumb-item>Home</el-breadcrumb-item>
                                    <el-breadcrumb-item>Dashboard</el-breadcrumb-item>
                                </el-breadcrumb>
                            </div>  </div> -->
                            <RouterView />



                        </el-main>

                        <el-footer>
                            <div class="footer-content">
                                <p>Footer 内容 - 会随着 Main 一起滚动</p>
                                <p>© 2024 版权所有</p>
                            </div>
                        </el-footer>
                    </div>
                </div>

            </el-container>
        </el-container>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive } from 'vue'
import { RouterView } from 'vue-router'
import { Menu, Fold, Expand, Key } from '@element-plus/icons-vue'
import { User__Get_Info } from '@/typer/api'
import type { User__table_interface } from '@/typer/api'

const User_info: User__table_interface = reactive({
    Id: 0, // 用户ID
    Name: '', // 用户名
    Permissions: 0,   // 权限
    Refresh_Token_Time: 0,   // 过期时间设定（s）
    Discontinued: false,    // 停用
    Phone: '',  // 电话
    Email: '',  // 邮箱 
})

User__Get_Info().then((User) => {
    Object.assign(User_info, User)
})



// 响应式判断
const isMobile = ref(false)
const asideVisible = ref(true)
const isCollapsed = ref(false)

// 计算侧边栏宽度
const asideWidth = computed(() => {
    if (isMobile.value) {
        return '280px' // 移动端抽屉宽度
    }
    return isCollapsed.value ? '64px' : '200px'
})

// 切换侧边栏显示/隐藏
const toggleAside = () => {
    asideVisible.value = !asideVisible.value
}

// 切换折叠状态（仅桌面端）
const toggleCollapse = () => {
    isCollapsed.value = !isCollapsed.value
}

// 菜单选择处理
const handleMenuSelect = (index: any) => {
    // 移动端选择菜单后自动隐藏侧边栏
    if (isMobile.value) {
        toggleAside()
    }
}

// 检查是否为移动端
const checkMobile = () => {
    const mobile = window.innerWidth < 768
    isMobile.value = mobile

    // 移动端默认隐藏侧边栏
    if (mobile && asideVisible.value) {
        asideVisible.value = false
    }
    // 桌面端默认显示侧边栏
    if (!mobile && !asideVisible.value) {
        asideVisible.value = true
    }
}

// 监听窗口大小变化
onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
    window.removeEventListener('resize', checkMobile)
})

// 监听键盘ESC键关闭侧边栏
onMounted(() => {
    const handleEscKey = (event: { key: string }) => {
        if (event.key === 'Escape' && isMobile.value && asideVisible.value) {
            toggleAside()
        }
    }
    window.addEventListener('keydown', handleEscKey)
    onUnmounted(() => window.removeEventListener('keydown', handleEscKey))
})
</script>

<style scoped>
a {
    text-decoration: none;
}

img {
    width: 100%;
    height: 100%;
}

.common-layout {
    height: 100vh;
    overflow: hidden;
}

.el-container {
    height: 100%;
}

/* Header 样式 */
.el-header {
    background: linear-gradient(135deg, #409EFF, #337ecc);
    color: white;
    display: flex;
    align-items: center;
    padding: 0 20px;
    box-shadow: 0 2px 8px rgba(64, 158, 255, 0.2);
    z-index: 1000;
}

.header-content {
    display: flex;
    align-items: center;
    gap: 15px;
    width: 100%;
}

.menu {
    margin: 20px;
}

.menu-button {
    background-color: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.3);
    color: white;
    transition: all 0.3s ease;
}

.menu-button:hover {
    background-color: rgba(255, 255, 255, 0.2);
    transform: scale(1.05);
}

.header-title {
    font-size: 20px;
    font-weight: bold;
    letter-spacing: 1px;
}

/* 主容器 */
.main-container {
    position: relative;
    flex: 1;
    overflow: hidden;
}

/* 侧边栏容器 */
.aside-wrapper {
    background-color: white;
    border-right: 1px solid #e4e7ed;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
    z-index: 100;
    overflow: hidden;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

/* 移动端侧边栏样式 */
.aside-wrapper.mobile-aside {
    position: fixed !important;
    top: 0;
    left: 0;
    bottom: 0;
    height: 100vh;
    z-index: 2000;
    box-shadow: 4px 0 16px rgba(0, 0, 0, 0.15);
    transform: translateX(0);
}

.aside-wrapper:not(.mobile-aside) {
    position: relative;
}

/* 侧边栏内容 */
.aside-content {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 20px 0;
}

.aside-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 20px 20px;
    border-bottom: 1px solid #e4e7ed;
    margin-bottom: 10px;
}

.aside-header h3 {
    color: #409EFF;
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.close-button {
    background-color: #f5f7fa;
    border-color: #e4e7ed;
    color: #606266;
}

.close-button:hover {
    background-color: #ecf5ff;
    border-color: #409EFF;
    color: #409EFF;
}

/* 侧边栏菜单 */
.aside-menu {
    border-right: none;
    flex: 1;
    padding: 0 10px;
    transition: all 0.3s ease;
}

.aside-menu:not(.el-menu--collapse) {
    width: 100%;
}

/* 侧边栏底部 */
.aside-footer {
    padding: 20px 20px 0;
    border-top: 1px solid #e4e7ed;
    margin-top: 20px;
    display: flex;
    justify-content: center;
}

.collapse-button {
    background-color: #409EFF;
    color: white;
    border: none;
    box-shadow: 0 2px 4px rgba(64, 158, 255, 0.3);
    transition: all 0.3s ease;
}

.collapse-button:hover {
    background-color: #337ecc;
    transform: rotate(180deg);
}

/* 移动端遮罩层 */
.aside-mask {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 1999;
    animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
    from {
        opacity: 0;
    }

    to {
        opacity: 1;
    }
}

/* 侧边栏过渡动画 */
.aside-slide-enter-active,
.aside-slide-leave-active {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.aside-slide-enter-from {
    transform: translateX(-100%);
    opacity: 0;
}

.aside-slide-leave-to {
    transform: translateX(-100%);
    opacity: 0;
}

/* 主内容区域 */
.el-main {
    background-color: #f5f7fa;
    padding: 20px;
    overflow-y: auto;
    transition: padding-left 0.3s ease;
}

.main-content {
    max-width: 1200px;
    margin: 0 auto;
}


.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: bold;
    color: #409EFF;
}


/* 响应式调整 */
@media (max-width: 767px) {
    .el-header {
        padding: 0 15px;
    }

    .header-title {
        font-size: 18px;
    }

    .el-main {
        padding: 15px;
    }

    .content-body {
        grid-template-columns: 1fr;
    }

    .content-header {
        padding: 15px;
    }

    .content-header h2 {
        font-size: 20px;
    }

    .aside-content {
        padding: 15px 0;
    }

    .aside-header {
        padding: 0 15px 15px;
    }
}


/* 可滚动区域 */
.scrollable-area {
    flex: 1;
    overflow-y: auto;
    background: #f5f7fa;
    -webkit-overflow-scrolling: touch;
    /* 移动端流畅滚动 */
}

.content-area {
    min-height: 100%;
    display: flex;
    flex-direction: column;
}

/* 主内容 */
.main {
    flex: 1;
    padding: 20px;
    background: white;
    border-radius: 8px;
    margin: 20px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}


/* Footer */
footer {
    background: white;
    border-top: 1px solid #e4e7ed;
    padding: 20px;
    margin-top: auto;
    flex-shrink: 0;
    box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.05);
}

.footer-content {
    max-width: 1200px;
    margin: 0 auto;
    text-align: center;
    color: #666;
    line-height: 1.6;
}

/* 移动端遮罩 */
.mobile-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1000;
    animation: fadeIn 0.3s ease;
}
</style>