import { ref, computed, reactive, toRaw } from 'vue'
import { defineStore } from 'pinia'
import { type User__table_interface } from '@/api/api'


export const useUserStore = defineStore('user', () => {
    const Id = ref(0)// 用户 ID
    const Name = ref('') // 用户名
    const Avatar = ref('')// 头像
    const Permissions = ref(0) // 权限
    const Discontinued = ref(false) // 停用
    const Phone = ref('') // 电话
    const Email = ref('') // 邮箱
    const Refresh_Token_bits = ref(0) // 刷新令牌 RSA 密钥长度
    const Access_Token_bits = ref(0)// 访问令牌 RSA 密钥长度
    const Refresh_Token_TTL = ref(0) // 刷新令牌过期时间（s）
    const Access_Token_TTL = ref(0)// 访问令牌过期时间（s）

    const get = computed(() => {
        // 修复：使用 toRaw 确保返回的是纯对象，彻底解除响应式依赖，防止外部直接修改触发不必要的更新或类型问题
        return {
            Id: Id.value,
            Name: Name.value,
            Avatar: Avatar.value,
            Permissions: Permissions.value,
            Discontinued: Discontinued.value,
            Phone: Phone.value,
            Email: Email.value,
            Refresh_Token_bits: Refresh_Token_bits.value,
            Access_Token_bits: Access_Token_bits.value,
            Refresh_Token_TTL: Refresh_Token_TTL.value,
            Access_Token_TTL: Access_Token_TTL.value
        } as User__table_interface
    })

    const set = (value: User__table_interface) => {
        Id.value = value.Id
        Name.value = value.Name
        Avatar.value = value.Avatar
        Permissions.value = value.Permissions
        Discontinued.value = value.Discontinued
        Phone.value = value.Phone
        Email.value = value.Email
        Refresh_Token_bits.value = value.Refresh_Token_bits
        Access_Token_bits.value = value.Access_Token_bits
        Refresh_Token_TTL.value = value.Refresh_Token_TTL
        Access_Token_TTL.value = value.Access_Token_TTL
    }

    return {
        Id,
        Name,
        Avatar,
        Permissions,
        Discontinued,
        Phone,
        Email,
        Refresh_Token_bits,
        Access_Token_bits,
        Refresh_Token_TTL,
        Access_Token_TTL,
        get,
        set
    }
})

