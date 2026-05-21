import { computed, reactive } from 'vue'
import { defineStore } from 'pinia'
import type { User__table_interface } from '@/api/api'

// 定义初始状态（方便重置）
const initUserState: User__table_interface = {
  Id: 0,
  Name: '',
  Avatar: '',
  Permissions: 0,
  Discontinued: false,
  Phone: '',
  Email: '',
  Refresh_Token_bits: 0,
  Access_Token_bits: 0,
  Refresh_Token_TTL: 0,
  Access_Token_TTL: 0,
}

export const useUserStore = defineStore('user', () => {
  // 统一响应式状态（替代大量 ref，代码极简）
  const userState = reactive<User__table_interface>({ ...initUserState })

  // 计算属性：获取纯对象（解除响应式，安全暴露给外部）
  const userInfo = computed((): User__table_interface => {
    return { ...userState }
  })

  // 设置用户信息（自动覆盖所有字段）
  const setUserInfo = (value: User__table_interface) => {
    Object.assign(userState, value)
  }

  // 重置为初始状态（登出必备）
  const resetUserState = () => {
    Object.assign(userState, initUserState)
  }

  return {
    // 状态
    ...userState,
    // 获取/设置/重置
    userInfo,
    setUserInfo,
    resetUserState,
  }
})