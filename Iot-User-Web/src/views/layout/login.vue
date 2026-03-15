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

<script setup>
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock, Key } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { Api_Name_login_Refresh_Token_update, Api_Access_Token_update } from '@/api/token'
import { User__Get_Info } from '@/api/api'


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

// 登录处理
const handleLogin = async () => {
    try {
        loading.value = true
        Api_Name_login_Refresh_Token_update(loginForm.Name, loginForm.Passwd).then((response) => {
            Api_Access_Token_update().then(() => {
                User__Get_Info(0, true).then(() => {
                    ElMessage({
                        message: '登录成功',
                        type: 'success',
                    })
                    router.push("/")
                })
            })
        }).catch((error) => {
            console.log(error)
            ElMessage.error(error)
        })
    } catch (error) {
        console.log(error)
        ElMessage.error(error)
        showCaptcha.value = true
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