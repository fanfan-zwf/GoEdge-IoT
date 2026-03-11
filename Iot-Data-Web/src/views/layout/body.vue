<template>
    <div class="common-layout">
        <el-container>
            <!-- 头部导航 -->
            <el-header class="layout-header">
                <el-menu :default-active="activeIndex" class="el-menu-demo" mode="horizontal" @select="handleSelect"
                    :router="true" active-text-color="#409EFF" background-color="#ffffff" text-color="#333333">
                    <el-menu-item index="/monitor">
                        <template #title>数据监控</template>
                    </el-menu-item>
                    <!-- 注释的菜单保持原有结构，优化格式 -->
                    <!-- <el-menu-item index="/alarm">
            <template #title>事件报警</template>
          </el-menu-item>
          <el-menu-item index="/history">
            <template #title>历史记录</template>
          </el-menu-item>
          <el-menu-item index="/drive">
            <template #title>驱动配置</template>
          </el-menu-item> -->
                    <el-menu-item index="/about">
                        <template #title>关于系统</template>
                    </el-menu-item>
                    <el-menu-item index="/about">
                        <template #title>关于系统</template>
                    </el-menu-item>
                </el-menu>
            </el-header>

            <!-- 主内容区 -->
            <el-main class="layout-main">
                <RouterView />
            </el-main>
        </el-container>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue"
import { useRoute } from "vue-router"

// 1. 获取路由实例
const route = useRoute()

// 2. 激活的菜单索引（绑定路由路径，更直观）
const activeIndex = ref(route.path)

// 3. 菜单选择事件（可扩展业务逻辑）
const handleSelect = (key: string) => {
    activeIndex.value = key
    // 可添加额外逻辑：比如埋点、权限校验、页面跳转前提示等
    console.log(`切换到菜单：${key}`)
}

// 4. 监听路由变化，同步菜单激活状态（解决手动输入URL/浏览器返回时菜单不高亮问题）
watch(
    () => route.path,
    (newPath) => {
        activeIndex.value = newPath
    },
    { immediate: true }
)

// 5. 组件挂载时初始化菜单状态
onMounted(() => {
    activeIndex.value = route.path
})
</script>

<style scoped>
/* 样式作用域隔离，避免全局污染 */
.layout-header {
    border-bottom: 1px solid #e6e6e6;
}

.layout-main {
    padding: 20px;
    min-height: calc(100vh - 60px);
    /* 适配页面高度，避免内容被遮挡 */
}

/* 激活态样式增强（配合Element Plus的active-class） */
:deep(.el-menu-item.is-active) {
    border-bottom: 2px solid #409EFF;
    font-weight: 600;
}

/* 移除a标签默认样式，优化hover效果 */
:deep(.el-menu-item a) {
    text-decoration: none;
    color: inherit;
    display: block;
    width: 100%;
    height: 100%;
}

:deep(.el-menu-item:hover) {
    background-color: #f5f7fa;
}
</style>