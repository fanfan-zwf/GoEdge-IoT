<template>
    <demo />
    <!-- <demo v-if="route.path != '/login' && Loading_Completed"></demo> -->
    <!-- <login v-if="route.path == '/login'"></login> -->
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Access_Token_Query } from "@/api/token"
import { User__Get_Info } from '@/api/api'
import { useUserStore } from '@/stores/user'
import { ElMessage, ElMessageBox } from 'element-plus'

import demo from "@/views/layout/body.vue"
import login from '@/views/layout/login.vue'

const userStore = useUserStore()

const route = useRoute()
const router = useRouter()

var Loading_Completed = ref(false)

// 修复：应用启动时先从 sessionStorage 恢复用户信息
const cachedUserInfo = sessionStorage.getItem('F_User_Info')
if (cachedUserInfo) {
    try {
        const userInfo = JSON.parse(cachedUserInfo)
        userStore.setUserInfo(userInfo)
    } catch (e) {
        ElMessage.error(`sessionStorage 恢复用户信息 解析缓存用户信息失败:${e}`)
    }
} else {
    console.log("⚠️ sessionStorage 中没有缓存的用户信息")
}

// // 获取刷新令牌
// Access_Token_Query().then((Access_Token_value) => {
//     console.log("已登录", Access_Token_value)
//     router.push("/")
// }).catch((error) => {
//     console.log("未登录", error)
//     ElMessage({
//         message: '未登录',
//         type: 'warning',
//     })
//     router.push("/login")
// }).finally(() => {
//     Loading_Completed.value = true
// })

Access_Token_Query().then((Access_Token_value) => {
    User__Get_Info().then((User_info) => {
        userStore.setUserInfo(User_info)
    }).catch((error) => {
        ElMessage.error(`User__Get_Info 调用失败:${error}`)
    })
}).catch((error) => {
    ElMessage.error(`Access_Token_Query 调用失败:${error}`)
}).finally(() => {
    Loading_Completed.value = true
})


</script>