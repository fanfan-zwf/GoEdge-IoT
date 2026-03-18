<template>
    <div class="common-layout">
        <el-container class="layout-container">
            <!-- 左侧侧边栏 -->
            <el-aside :width="asideWidth" class="left-sidebar" :class="{ 'mobile-visible': isMobileVisible }">
                <div class="sidebar-content" :class="{ 'collapsed': isCollapsed }">
                    <!-- Logo 区域 - 修改：分为左右两部分 -->
                    <div class="sidebar-logo">
                        <!-- 左侧：Logo 图标 -->
                        <div class="logo-icon-wrapper">
                            <img src="@/assets/icons/log.svg" alt="Logo" class="logo-image" />
                        </div>
                        <!-- 右侧：文字标题 -->
                        <div class="logo-text-wrapper" v-show="!isCollapsed || isMobile">
                            <span class="logo-text">管理系统</span>
                        </div>
                    </div>

                    <!-- 菜单区域 -->
                    <el-menu default-active="1" class="sidebar-menu" :collapse="isCollapsed" @select="handleMenuSelect">
                        <!-- 修改：路由名称应为 'info' 而非 'user' -->
                        <router-link active-class="active" :to="{ name: 'info', params: { User_Id: User_info.Id } }">
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

                    <!-- 折叠按钮 -->
                    <div class="sidebar-footer">
                        <el-button :icon="isCollapsed ? Expand : Fold" @click="toggleCollapse" circle size="small"
                            class="collapse-button" />
                    </div>
                </div>

                <!-- 修改：优化遮罩层，确保仅在移动端且可见时显示，调整类名控制动画 -->
                <div v-if="isMobile && isMobileVisible" class="mobile-overlay" :class="{ 'fade-in': isMobileVisible }"
                    @click="closeSidebar"></div>
            </el-aside>

            <!-- 右侧主内容区域 -->
            <el-container class="right-content">
                <!-- 上方 header - 头像在右，路径在左 -->
                <el-header class="top-header">
                    <div class="header-left">
                        <!-- 修改：移动端添加汉堡菜单按钮 -->
                        <el-icon v-if="isMobile" class="mobile-menu-btn" @click="toggleCollapse">
                            <Fold v-if="!isCollapsed" />
                            <Expand v-else />
                        </el-icon>
                        <el-breadcrumb separator="/">
                            <el-breadcrumb-item :to="{ path: '/' }">Home</el-breadcrumb-item>
                            <el-breadcrumb-item>Dashboard</el-breadcrumb-item>
                        </el-breadcrumb>
                    </div>
                    <div class="header-right">
                        <UserMenu :user="User_info" />
                    </div>
                </el-header>

                <el-container class="main-content">
                    <el-main class="content-main">
                        <RouterView />
                    </el-main>
                    <el-footer class="content-footer">
                        <div class="footer-content">
                            <p>Footer 内容 - 会随着 Main 一起滚动</p>
                            <p>© 2024 版权所有</p>
                        </div>
                    </el-footer>
                </el-container>
            </el-container>
        </el-container>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { Fold, Expand } from '@element-plus/icons-vue'
import UserMenu from '@/components/UserMenu.vue'
import { User__Get_Info } from '@/typer/api'
import type { User__table_interface } from '@/typer/api'

const router = useRouter()

const User_info: User__table_interface = reactive({
    Id: 0,
    Name: '',
    Permissions: 0,
    Refresh_Token_Time: 0,
    Discontinued: false,
    Phone: '',
    Email: '',
})

User__Get_Info().then((User) => {
    Object.assign(User_info, User)
})

const isCollapsed = ref(false)
// 新增：移动端状态控制
const isMobile = ref(false)
const isMobileVisible = ref(false)

const toggleCollapse = () => {
    if (isMobile.value) {
        // 移动端：切换显示/隐藏
        const willBeVisible = !isMobileVisible.value
        isMobileVisible.value = willBeVisible

        // IO 注意：移动端打开侧边栏时，强制确保菜单处于展开状态，防止内容被折叠隐藏
        if (willBeVisible) {
            isCollapsed.value = false
        }
    } else {
        // 桌面端：切换折叠/展开
        isCollapsed.value = !isCollapsed.value
    }
}

// 新增：关闭侧边栏（用于遮罩点击或菜单选择后）
const closeSidebar = () => {
    if (isMobile.value) {
        isMobileVisible.value = false
    }
}

// 检测屏幕尺寸
const checkMobile = () => {
    const wasMobile = isMobile.value
    isMobile.value = window.innerWidth < 768

    // IO 注意：当从移动端切换到桌面端时，必须关闭移动端特有的遮罩/显示状态
    if (wasMobile && !isMobile.value) {
        isMobileVisible.value = false
        // 可选：桌面端恢复默认逻辑，如果需要可以在此处重置 isCollapsed
    } else if (!wasMobile && isMobile.value) {
        // 从桌面切到移动，默认隐藏，保持折叠状态不影响，因为宽度由 isMobileVisible 控制
        isMobileVisible.value = false
    }
}

onMounted(() => {
    checkMobile()
    window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
    window.removeEventListener('resize', checkMobile)
})

const asideWidth = computed(() => {
    if (isMobile.value) {
        return isMobileVisible.value ? '200px' : '0px'
    }
    // 桌面端逻辑
    return isCollapsed.value ? '64px' : '200px'
})

const handleMenuSelect = (index: any) => {
    // 移动端选择菜单后自动关闭侧边栏
    if (isMobile.value) {
        closeSidebar()
    }
    // 菜单选择逻辑
}

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

/* ===== 侧边栏样式 - 强化白色背景防止变黑 ===== */
.left-sidebar {
    height: 100vh !important;
    box-sizing: border-box !important;
    /* 强制背景为白色，优先级最高 */
    background: #ffffff !important;
    border-right: 1px solid #f0f0f0 !important;
    box-shadow: none;
    transition: width 0.3s ease, background-color 0.3s ease;
    position: relative;
    z-index: 1001;
    overflow: hidden;
}

/* 移动端特殊样式 */
@media (max-width: 767px) {
    .left-sidebar {
        position: fixed;
        top: 0;
        left: 0;
        height: 100%;
        /* 默认宽度为 0 */
        width: 0 !important;
        /* 再次确保移动端侧边栏背景也是白色，防止动画期间变黑 */
        background: #ffffff !important;
        box-shadow: 2px 0 8px rgba(0, 0, 0, 0.15);
        transition: width 0.3s ease, background-color 0.3s ease;
        z-index: 1002;
    }

    .left-sidebar.mobile-visible {
        width: 200px !important;
        /* 展开时确保持续白色 */
        background: #ffffff !important;
    }

    /* 添加淡入动画类 */
    .mobile-overlay.fade-in {
        opacity: 1;
        visibility: visible;
    }

    .sidebar-content {
        overflow-x: hidden;
        /* 确保内容区域背景透明，完全依赖父级白色背景 */
        background: transparent !important;
    }
}

.sidebar-content {
    position: relative;
    padding-bottom: 60px;
    overflow-y: auto !important;
    overflow-x: hidden !important;
    height: 100%;
    scrollbar-width: thin;
    /* 显式声明背景，防止继承问题 */
    background-color: #ffffff !important;
}

.sidebar-content::-webkit-scrollbar {
    width: 6px;
}

.sidebar-content::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 3px;
}

.sidebar-content::-webkit-scrollbar-thumb:hover {
    background: rgba(0, 0, 0, 0.4);
}

.sidebar-content.collapsed {
    overflow-y: hidden !important;
}

.sidebar-logo {
    height: 60px !important; /* 稍微增加高度以容纳放大的图标 */
    display: flex !important;
    justify-content: flex-start !important; /* 左对齐开始 */
    align-items: center !important;
    padding: 0 10px; /* 调整内边距 */
    background: #ffffff !important;
    border-bottom: 1px solid #f5f5f5;
    overflow: hidden; /* 防止放大时溢出 */
    transition: all 0.3s ease;
}

/* 左侧图标容器 */
.logo-icon-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    width: 44px; /* 固定宽度占位，防止文字消失时抖动 */
    flex-shrink: 0;
    transition: transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275); /* 添加弹性过渡 */
}

/* 核心需求：折叠时图标顶比例缩放放大 */
.sidebar-content.collapsed .logo-icon-wrapper {
    transform: scale(1.6); /* 放大显示 */
    width: 100%; /* 占满整个折叠后的宽度 */
}

/* 移动端不应用放大效果，保持正常 */
@media (max-width: 767px) {
    .sidebar-content.collapsed .logo-icon-wrapper {
        transform: scale(1); 
        width: 44px;
    }
}

.logo-image {
    max-width: 32px !important;
    max-height: 32px !important;
    object-fit: contain !important;
    display: block !important;
}

/* 右侧文字容器 */
.logo-text-wrapper {
    margin-left: 12px;
    white-space: nowrap;
    opacity: 1;
    transition: opacity 0.3s ease, margin 0.3s ease;
    overflow: hidden;
}

/* 折叠时隐藏文字（桌面端） */
.sidebar-content.collapsed .logo-text-wrapper {
    opacity: 0;
    margin-left: 0;
    width: 0;
}

.logo-text {
    font-size: 18px;
    font-weight: bold;
    color: #333;
    display: block;
}

.sidebar-menu {
    border-right: none !important;
    background-color: transparent !important;
    /* 修改：移除菜单项之间的边框 */
}

.el-menu,
.el-menu--vertical,
.el-menu--collapse {
    border-right: none !important;
    background-color: transparent !important;
}

.el-menu-item {
    /* 修改：默认文字颜色为黑色 */
    color: #333333 !important;
    /* 修改：添加圆角倒角效果 */
    border-radius: 8px !important;
    margin: 4px 8px !important;
    width: auto !important;
    transition: all 0.3s ease;

    /* 新增：强制左对齐，解决不对齐问题 */
    display: flex !important;
    justify-content: flex-start !important;
    align-items: center !important;
    padding-left: 15px !important;
    /* 统一左侧内边距 */
}

/* 新增：修复子菜单标题的对齐 */
:deep(.el-sub-menu__title) {
    display: flex !important;
    justify-content: flex-start !important;
    align-items: center !important;
    padding-left: 15px !important;
    border-radius: 8px !important;
    margin: 4px 8px !important;
    width: auto !important;
}

:deep(.el-sub-menu__title:hover) {
    background-color: rgba(64, 158, 255, 0.1) !important;
    color: #409EFF !important;
}

/* 新增：修复折叠状态下的图标对齐 */
:deep(.el-menu--collapse .el-menu-item),
:deep(.el-menu--collapse .el-sub-menu__title) {
    justify-content: center !important;
    /* 折叠时居中 */
    padding-left: 0 !important;
    /* 折叠时移除左侧内边距，让图标完全居中 */
    padding-right: 0 !important;
    /* 确保右侧也无内边距干扰 */
    width: 100% !important;
    box-sizing: border-box;
}

:deep(.el-menu--collapse .el-menu-item .el-icon),
:deep(.el-menu--collapse .el-sub-menu__title .el-icon) {
    margin-right: 0 !important;
    /* 折叠时移除图标右边距 */
    display: flex;
    justify-content: center;
    align-items: center;
    width: 100%;
    /* 确保图标容器占满宽度以便居中 */
}

/* 新增：确保折叠时隐藏文字，只留图标居中 */
:deep(.el-menu--collapse .el-menu-item span),
:deep(.el-menu--collapse .el-sub-menu__title span) {
    display: none !important;
}

.el-menu-item:hover {
    /* 修改：悬停背景为浅蓝色 */
    background-color: rgba(64, 158, 255, 0.1) !important;
    /* 修改：悬停文字颜色保持深色或微蓝 */
    color: #409EFF !important;
}

.el-menu-item.is-active {
    /* 修改：选中背景为更明显的浅蓝色 */
    background-color: rgba(64, 158, 255, 0.15) !important;
    /* 修改：选中文字颜色为蓝色 */
    color: #409EFF !important;
    font-weight: 600;
}

/* 修复子菜单项的样式继承 */
.el-menu--inline .el-menu-item {
    margin: 4px 8px 4px 20px !important;
    background-color: transparent !important;
}

.el-menu--inline .el-menu-item:hover {
    background-color: rgba(64, 158, 255, 0.1) !important;
}

.sidebar-footer {
    position: absolute;
    bottom: 0;
    left: 0;
    width: 100%;
    padding: 12px 0;
    /* 强制白色背景 */
    background: #ffffff !important;
    border-top: 1px solid #f0f0f0;
    display: flex;
    justify-content: center;
}

.collapse-button {
    /* 修改：按钮样式改为浅色风格 */
    background-color: #f5f7fa;
    color: #606266;
    border: 1px solid #e6e6e6;
    outline: none;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    transition: transform 0.3s ease, background-color 0.3s ease, color 0.3s ease;
}

.collapse-button:hover {
    background-color: #409EFF;
    color: #ffffff;
    border-color: #409EFF;
    transform: rotate(180deg);
}

/* ===== 主内容区域 ===== */
.right-content {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
    /* 确保主内容背景始终为浅色 */
    background-color: #f5f7fa !important;
}

.main-content {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
}

.content-main {
    background-color: #f0f2f5;
    padding: 20px;
    overflow-y: auto;
    flex: 1;
    /* 确保内容区域不会因父级问题变黑 */
    color: #333;
}

.content-footer {
    background: #ffffff;
    border-top: 1px solid #e6e6e6;
    padding: 15px 20px;
    flex-shrink: 0;
}

.footer-content {
    text-align: center;
    color: #666;
    line-height: 1.6;
}

.footer-content p {
    margin: 5px 0;
    font-size: 14px;
}

/* ===== Header 样式 - 优化左右布局 ===== */
.top-header {
    background: #ffffff;
    color: #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0 20px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    z-index: 1000;
    height: 50px;
    border-bottom: 1px solid #e6e6e6;
    position: relative;
}

.header-left {
    display: flex;
    align-items: center;
    flex: 0 0 auto;
    gap: 10px;
    /* 增加按钮和面包屑间距 */
}

.mobile-menu-btn {
    font-size: 20px;
    cursor: pointer;
    color: #606266;
    display: flex;
    align-items: center;
    padding: 4px;
    border-radius: 4px;
}

.mobile-menu-btn:hover {
    background-color: #f5f7fa;
}

/* 桌面端隐藏移动端按钮 */
@media (min-width: 768px) {
    .mobile-menu-btn {
        display: none;
    }
}

.header-right {
    display: flex;
    align-items: center;
    flex: 0 0 auto;
    /* 移除多余的背景和边框，让 UserMenu 内部控制样式 */
    cursor: pointer;
}

/* 移除之前针对 header-right 的奇怪圆形边框样式 */
/* 面包屑样式 */
.el-breadcrumb {
    color: #606266;
}

.el-breadcrumb-item a {
    color: #606266;
    transition: color 0.3s ease;
}

.el-breadcrumb-item a:hover {
    color: #409EFF;
}

.el-breadcrumb-item.is-link {
    color: #606266;
}

.el-breadcrumb-item:last-child {
    color: #409EFF;
    font-weight: 500;
}

/* ===== 响应式调整 ===== */
@media (max-width: 767px) {
    .top-header {
        padding: 0 10px;
        height: 50px;
    }

    .header-right {
        /* 移动端保留头像，但可能需要调整大小 */
        display: flex;
    }

    .user-avatar-img {
        --el-avatar-size: 32px;
        /* 稍微缩小头像 */
    }

    .el-main {
        padding: 10px;
    }

    .sidebar-logo {
        height: 50px !important;
        /* 移动端 Logo 可能需要调整 */
        justify-content: flex-start !important;
        padding-left: 15px;
    }

    /* 移动端面包屑简化 */
    .el-breadcrumb-item:not(:last-child) {
        display: none;
    }

    /* 强制主内容区域在移动端保持亮色背景 */
    .right-content,
    .content-main,
    .common-layout {
        background-color: #f5f7fa !important;
    }
}

/* 确保 UserMenu 内部的头像也是圆形的 */
.header-right :deep(img),
.header-right :deep(.el-avatar) {
    border-radius: 50% !important;
    display: block;
}
</style>