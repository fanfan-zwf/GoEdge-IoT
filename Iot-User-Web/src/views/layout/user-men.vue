<template>
 <template>
  <div class="tabs-router-container" style="padding: 20px;">
    <!-- Element Plus 标签页 + 路由联动 -->
    <el-tabs 
      v-model="activeRoute" 
      class="custom-tabs"
      type="card"  
      @tab-click="handleTabClick"
    >
      <!-- 遍历路由生成标签页 -->
      <el-tab-pane 
        v-for="route in routerRoutes" 
        :key="route.name"
        :label="route.meta?.title ?? ''"
        :name="route.name"
        :disabled="route.meta?.disabled ?? false"
      >
        <!-- 路由视图：渲染当前选中标签对应的页面 -->
        <router-view />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

 
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, reactive, watch } from 'vue' 
import { Menu, Fold, Expand, Key } from '@element-plus/icons-vue'
import { User__Get_Info } from '@/api/api'
import type { User__table_interface } from '@/api/api' 
import { useRouter, useRoute, RouterView } from 'vue-router'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()

// 过滤出需要显示在标签页的路由（排除默认重定向的根路由）
const routerRoutes = computed(() => {
  return router.options.routes.filter(item => item.path !== '/')
})

// 绑定当前激活的标签（与路由名称联动）
const activeRoute = ref(route.name)

// 监听路由变化，同步更新激活的标签
watch(
  () => route.name,
  (newName) => {
    activeRoute.value = newName
  },
  { immediate: true }
)

// 点击标签页时跳转对应路由
const handleTabClick = (tab: { props: { disabled: any; name: any } }) => {
  // 如果标签禁用，提示并返回
  if (tab.props.disabled) {
    ElMessage.warning('该标签页暂不可用')
    return
  }
  // 跳转到对应路由
  router.push({ name: tab.props.name })
}
const User_info: User__table_interface = reactive({
    Id: 0, // 用户ID
    Name: '', // 用户名
    Permissions: 0,   // 权限
    Refresh_Token_Time: 0,   // 过期时间设定（s）
    Discontinued: false,    // 停用
    Phone: '',  // 电话
    Email: '',  // 邮箱 
    Refresh_Token_bits: 0,    // 刷新令牌RSA密钥长度 
    Access_Token_bits: 0,     // 访问令牌RSA密钥长度 
    Refresh_Token_TTL: 0,     // 刷新令牌过期时间（s）
    Access_Token_TTL: 0,     // 访问令牌过期时间（s）
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



</script>

<style scoped>

/* 自定义标签页样式（覆盖 Element Plus 默认样式） */
.custom-tabs :deep(.el-tabs__header) {
  margin-bottom: 20px;
}

/* 激活标签的样式 */
.custom-tabs :deep(.el-tabs__item.is-active) {
  color: #1989fa; /* 自定义激活文字颜色 */
  font-weight: bold;
}

/* 卡片式标签的激活样式 */
.custom-tabs :deep(.el-tabs--card .el-tabs__item.is-active) {
  border-bottom-color: #1989fa; /* 自定义激活下划线颜色 */
}

/* 禁用标签的样式 */
.custom-tabs :deep(.el-tabs__item.is-disabled) {
  color: #ccc !important;
  cursor: not-allowed;
}

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