<template>
    <demo v-if="route.path != '/login' && Loading_Completed"></demo>
    <login v-if="route.path == '/login'"></login>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Access_Token_Query } from "@/typer/token"
import { ElMessage } from 'element-plus'

import demo from "@/views/layout/body.vue"
import login from '@/views/layout/login.vue'

const route = useRoute()
const router = useRouter()

var Loading_Completed = ref(false)

// 获取刷新令牌
Access_Token_Query().then((Access_Token_value) => {
    console.log("已登录", Access_Token_value)
    router.push("/")
}).catch((error) => {
    console.log("未登录", error)
    ElMessage({
        message: '未登录',
        type: 'warning',
    })
    router.push("/login")
}).finally(() => {
    Loading_Completed.value = true
})



</script>

<style></style>
