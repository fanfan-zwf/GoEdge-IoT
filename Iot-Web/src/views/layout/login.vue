<template>
    <div class="login-container">
        <el-card class="login-card">
            <div class="login-header">
                <h2>系统登录</h2>
                <p>请输入您的账号和密码</p>
            </div>

            <el-form ref="loginFormRef" :model="loginForm" :rules="loginRules" class="login-form">
                <!-- 用户名输入框 -->
                <el-form-item prop="Name">
                    <el-input v-model="loginForm.Name" placeholder="请输入用户名" size="large" :prefix-icon="User"
                        clearable />
                </el-form-item>

                <!-- 密码输入框 -->
                <el-form-item prop="Passwd">
                    <el-input v-model="loginForm.Passwd" type="password" placeholder="请输入密码" size="large"
                        :prefix-icon="Lock" show-password @keyup.enter="handleLogin" />
                </el-form-item>

                <!-- 验证码（可选） -->
                <!-- <el-form-item prop="captcha" v-if="showCaptcha">
                    <div class="captcha-container">
                        <el-input v-model="loginForm.captcha" placeholder="请输入验证码" size="large" :prefix-icon="Key"
                            class="captcha-input" @keyup.enter="handleLogin" />
                        <img :src="captchaImage" alt="验证码" class="captcha-image" @click="refreshCaptcha" />
                    </div>
                </el-form-item> -->

                <!-- 记住我选项 -->
                <!-- <el-form-item>
                    <el-checkbox v-model="loginForm.remember">记住密码</el-checkbox>
                    <el-link type="primary" class="forgot-password">忘记密码？</el-link>
                </el-form-item> -->

                <!-- 登录按钮 -->
                <el-form-item>
                    <el-button type="primary" size="large" class="login-button" :loading="loading" @click="handleLogin">
                        登录
                    </el-button>
                </el-form-item>

                <!-- 其他登录方式 -->
                <!-- <div class="other-login">
                    <el-divider>其他登录方式</el-divider>
                    <div class="oauth-buttons">
                        <el-button circle>
                            <el-icon>
                                <IconWechat />
                            </el-icon>
                        </el-button>
                        <el-button circle>
                            <el-icon>
                                <IconQq />
                            </el-icon>
                        </el-button>
                        <el-button circle>
                            <el-icon>
                                <IconPhone />
                            </el-icon>
                        </el-button>
                    </div>
                </div> -->
            </el-form>
        </el-card>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock, Key } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { Api_Name_login_Refresh_Token_update, Api_Access_Token_update } from '@/api/token'
import { User__Get_Info } from '@/api/api'
import { useUserStore } from '@/stores/user'

const router = useRouter()

// 表单数据
const loginForm = reactive({
    Name: '',
    Passwd: ''
})

// 验证规则
const loginRules = {
    Name: [
        { required: true, message: '请输入用户名', trigger: 'blur' },
        {
            pattern: /^.{0,23}$/,
            message: '太长了',
            trigger: 'blur'
        }
    ],
    Passwd: [
        { required: true, message: '请输入密码', trigger: 'blur' },
        {
            pattern: /^(?=.*\d)(?=.*[!@#$%&])[A-Za-z\d!@#$%&]{8}$/,
            message: '请输入大于8位,包含大小写、数字和 ! @ # $ % & 特殊符号',
            trigger: 'blur'
        }
    ]
}

const loading = ref(false)
const showCaptcha = ref(false)


// 登录处理 - 重构为 async/await 确保时序
const handleLogin = async () => {
    // 在此处获取 store，确保 Pinia 已初始化
    const userStore = useUserStore()

    // 手动触发校验（如果使用了 ref 绑定 form）
    // await loginFormRef.value?.validate() 

    try {
        loading.value = true

        // 1. 第一步：获取 Refresh Token
        console.log('正在获取 Refresh Token...')
        const refreshData = await Api_Name_login_Refresh_Token_update(loginForm.Name, loginForm.Passwd)
        console.log('Refresh Token 获取成功:', refreshData)

        // 2. 第二步：获取 Access Token (此步骤会将 token 写入 localStorage)
        console.log('正在获取 Access Token...')
        const accessData = await Api_Access_Token_update()
        console.log('Access Token 获取成功:', accessData)

        // 3. 第三步：获取用户信息 (此时 localStorage 中一定有 token，拦截器会正常添加)
        console.log('正在获取用户信息...')
        const userInfo = await User__Get_Info(0)

        // 4. 登录成功后的处理
        ElMessage({
            message: '登录成功',
            type: 'success',
        })

        userStore.set(userInfo)
        router.push("/")

    } catch (error: any) {
        console.error('登录过程失败:', error)
        const msg = typeof error === 'string' ? error : (error.message || '登录失败，请稍后重试')
        ElMessage.error(msg)
    } finally {
        loading.value = false
    }
}

</script>

<style scoped>
.login-container {
    min-height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    padding: 20px;
}

.login-card {
    width: 100%;
    max-width: 420px;
    border-radius: 12px;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
}

.login-header {
    text-align: center;
    margin-bottom: 30px;
}

.login-header h2 {
    color: #303133;
    margin-bottom: 8px;
    font-size: 24px;
}

.login-header p {
    color: #909399;
    font-size: 14px;
}

.login-form {
    padding: 0 20px;
}

:deep(.el-input__wrapper) {
    border-radius: 8px;
}

.login-button {
    width: 100%;
    border-radius: 8px;
    height: 48px;
    font-size: 16px;
    margin-top: 10px;
}

.forgot-password {
    float: right;
    font-size: 14px;
}

.captcha-container {
    display: flex;
    gap: 10px;
    align-items: center;
}

.captcha-input {
    flex: 1;
}

.captcha-image {
    height: 40px;
    cursor: pointer;
    border-radius: 4px;
    border: 1px solid #dcdfe6;
}

.other-login {
    margin-top: 30px;
    text-align: center;
}

.oauth-buttons {
    display: flex;
    justify-content: center;
    gap: 20px;
    margin-top: 20px;
}

:deep(.el-divider__text) {
    background-color: transparent;
    color: #909399;
    font-size: 12px;
}
</style>