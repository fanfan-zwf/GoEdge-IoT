import { http_Front_url } from '@/api/index'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { sha3_256_sync } from '@/api/index'
import { useUserStore } from '@/stores/user'



/**
*******************用户*******************
*/

/**
 * 用户接口
 */
export interface User__table_interface {
    Id: number // 用户ID
    Name: string // 用户名
    Avatar: string // 头像
    Permissions: number   // 权限 
    Discontinued: boolean    // 停用
    Phone: string  // 电话
    Email: string  // 邮箱 

    Refresh_Token_bits: number    // 刷新令牌RSA密钥长度 
    Access_Token_bits: number    // 访问令牌RSA密钥长度 
    Refresh_Token_TTL: number    // 刷新令牌过期时间（s）
    Access_Token_TTL: number    // 访问令牌过期时间（s）
}

/**
 * 用户接口
 */
export interface User__all_table_type extends User__table_interface {
    Passwd: string // 密码
}

/**
 * 获取用户信息
 * Param User_Id 用户ID(默认0-当前用户)
 */
export async function User__Get_Info(User_Id: number = 0): Promise<User__table_interface> {

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/get/info', {
            User_Id: User_Id
        })

        const status = (await response).status
        if (status == 200) {
            const User_info: User__table_interface = (await response).data.Data
            if (User_Id == 0) {
                sessionStorage.removeItem('F_User_Info');
                sessionStorage.setItem('F_User_Info', JSON.stringify(User_info))
                const userStore = useUserStore()
                userStore.set(User_info)
            }
            return User_info
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        // ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 查询多个用户信息
 * Param User_Id_Array 用户ID传递是数组
 */
export async function User__Get_Info_Array(User_Id_Array: number[]): Promise<User__table_interface[]> {
    if (User_Id_Array.length == 0) {
        throw "User_Id_Array长度是0"
    }
    if (User_Id_Array.length > 1000) {
        throw "User_Id_Array过长"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/get/info_array', {
            User_Id_Array: User_Id_Array
        })

        const status = (await response).status
        if (status == 200) {
            const User_info: User__table_interface[] = (await response).data.Data
            return User_info
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 查询多个用户信息
 * Param User_Id_Array 用户ID传递是数组
 */
export async function User__Get_Info_Search(Search: string, Type: string, Number: number): Promise<User__table_interface[]> {
    const User_Info_Search_Type: string[] = ["Name", "Phone", "Email"]

    if (Type != "" && !User_Info_Search_Type.includes(Type)) {
        throw "Type不存在的方法"
    } else if (Type == "") (
        Type = "Name"
    )

    if (Number = 0) {
        Number = 10
    } else if (Number > 1000) {
        throw "搜索数量过长"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/get/search', {
            Search: Search,
            Type: Type,
            Number: Number
        })

        const status = (await response).status
        if (status == 200) {
            const User_info: User__table_interface[] = (await response).data.Data
            return User_info
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 增加用户
 * Param Name 新用户名, User_Id 用户ID(默认0-当前用户)
 */
export async function User_Set_Add(value: User__all_table_type): Promise<void> {
    if (value.Id) {
        throw 'Id不能对于0'
    }

    value.Passwd = sha3_256_sync(0, value.Passwd)

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/add', value)
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 设置用户名
 * Param Name 新用户名, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Name(Name: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/name', {
            User_Id: User_Id,
            Name: Name
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 设置密码
 * Param Passwd 新密码, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Passwd(Passwd: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/passwd', {
            User_Id: User_Id,
            Passwd: sha3_256_sync(0, Passwd)
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 设置密码
 * Param Url 头像地址, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Avatar(Url: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/Url', {
            User_Id: User_Id,
            Url: Url,
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}


/**
 * 设置电话
 * Param Phone 新电话, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Phone(Phone: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/phone', {
            User_Id: User_Id,
            Phone: Phone
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 设置邮箱
 * Param Email 新邮箱, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Email(Email: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/email', {
            User_Id: User_Id,
            Email: Email
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 设置邮箱
 * Param Email 新邮箱, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Del(User_Id: number): Promise<void> {
    if (User_Id == 0) {
        throw 'User_Id不能是0'
    }
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/set/del', {
            User_Id: User_Id
        })
        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}


/**
 * 获取权限条数
 */
export async function User__All_Count(): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/get/count')

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 分页查询权限
 */
export async function User__All_Query(Page: number, Page_Size: number): Promise<User__table_interface> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/user/get/query', {
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            const Authority_table: User__table_interface = (await response).data.Data
            return Authority_table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
*******************权限*******************
*/

/**
 * 权限接口
 */
export interface Authority__table_interface {
    Id: number        // 权限ID
    Name: string      // 权限名称
    Theme: string     // 权限主题
    Explain: string   // 说明
}

/**
 * 获取权限条数
 */
export async function Authority__Count(): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/count')

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 分页查询权限
 */
export async function Authority__Query(Page: number, Page_Size: number): Promise<Authority__table_interface> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/query', {
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            const Authority_table: Authority__table_interface = (await response).data.Data
            return Authority_table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 查询指定权限Id
 */
export async function Authority__Id_Array(Authority_Id: number[]): Promise<Authority__table_interface[]> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/id', {
            Authority_Id: Authority_Id
        })

        const status = (await response).status
        if (status == 200) {
            const Authority_table: Authority__table_interface[] = (await response).data.Data
            return Authority_table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}


/**
 * 查询多个用户信息
 * Param User_Id_Array 用户ID传递是数组
 */
export async function Authority__Search(Search: string, Type: string, Number: number): Promise<Authority__table_interface[]> {
    const Authority_Search_Type: string[] = ["Name", "Theme", "Explain"]

    if (Type != "" && !Authority_Search_Type.includes(Type)) {
        throw "Type不存在的方法"
    } else if (Type == "") (
        Type = "Name"
    )

    if (Number = 0) {
        Number = 10
    } else if (Number > 1000) {
        throw "搜索数量过长"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/search', {
            Search: Search,
            Type: Type,
            Number: Number
        })

        const status = (await response).status
        if (status == 200) {
            const Authority_info: Authority__table_interface[] = (await response).data.Data
            return Authority_info
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
        throw axiosError.response?.data?.Msg || '请求失败';
    }

}

/**
 * 增加权限
 */
export async function Authority__Add(Authority: Authority__table_interface): Promise<void> {
    if (Authority.Id != 0) {
        throw "id应该等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/add', Authority)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 修改权限
 */
export async function Authority__Update(Authority: Authority__table_interface): Promise<void> {
    if (Authority.Id == 0) {
        throw "id应该不等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/update', Authority)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 删除权限
 */
export async function Authority__Del(Authority_Id: number): Promise<void> {
    if (Authority_Id == 0) {
        throw "id应该不等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority/del', {
            Authority_Id: Authority_Id
        })

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}



/**
 * 用户对应的权限
 */
export interface Authority_User__table_interface {
    Id: number               // Id
    User_Id: number          // 用户id
    Authority_Id: number     // 权限id
    Enable: boolean          // 使能
}

/**
 * 分页查询全部用户权限条数
 */
export async function Authority_User__All_Count(User_Id: number = 0): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority_user/count', {
            User_Id: User_Id,
        })

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 分页查询全部用户权限 Page页码(0代表全部) Page_Size每页条数
 */
export async function Authority_User__All_Query(Page: number, Page_Size: number, User_Id: number = 0): Promise<Authority_User__table_interface[]> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority_user/query', {
            User_Id: User_Id,
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            const Authority_User_table: Authority_User__table_interface[] = (await response).data.Data
            return Authority_User_table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 权限使能设定
 */
export async function Authority_User__Enable(User_Id: number, Enable: boolean): Promise<void> {
    if (User_Id == 0) {
        throw 'User_Id不能等于0';
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority_user/enable', {
            User_Id: User_Id,
            Enable: Enable,
        })

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 权限增加
 */
export async function Authority_User__Add(Authority_User: Authority_User__table_interface): Promise<void> {
    if (Authority_User.Id != 0) {
        throw "参数不正确Id应该是0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority_user/add', Authority_User)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 权限删除
 */
export async function Authority_User__Del(Authority_User__Id: number): Promise<void> {
    if (Authority_User__Id == 0) {
        throw "参数不正确Id应该不是0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/authority_user/del', {
            Id: Authority_User__Id
        })

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response).data.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}


/**
*******************分组*******************
*/

/**
 * 分组接口
 */
export interface Group__table_interface {
    Id: number        // 分组ID
    Name: string      // 分组名称 
    Explain: string   // 说明
}

/**
 * 获取分组条数
 */
export async function Group__Count(): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group/count')

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 分页查询分组
 */
export async function Group__Query(Page: number, Page_Size: number): Promise<Group__table_interface> {
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group/query', {
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            const Group__table: Group__table_interface = (await response).data.Data
            return Group__table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 增加分组
 */
export async function Group__Add(Group: Group__table_interface): Promise<void> {
    if (Group.Id != 0) {
        throw "id应该等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group/add', Group)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 修改分组
 */
export async function Group__Update(Group: Group__table_interface): Promise<void> {
    if (Group.Id == 0) {
        throw "id应该不等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group/update', Group)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 删除分组
 */
export async function Group__Del(Group__Id: number): Promise<void> {
    if (Group__Id == 0) {
        throw "id应该不等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group/del', {
            Group_Id: Group__Id
        })

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}


/**
*******************用户分组*******************
*/

/**
 * 用户分组接口
 */
export interface Group_User__table_interface {
    Id: number               // 用户分组ID 
    User_Id: number          // 用户id
    Group_Id: number         // 用户组id
    Administrator: boolean   // 是否是管理员
}

/**
 * 获取分组条数
 */
export async function Group_User__Count(Group_User_Id: number): Promise<number> {
    if (Group_User_Id == 0) {
        throw 'Group_User_Id不应该等于0'
    }
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group_user/count', {
            Group_User_Id: Group_User_Id
        })

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 分页查询分组
 */
export async function Group_User__Query(Group_User_Id: number, Page: number, Page_Size: number): Promise<Group_User__table_interface[]> {
    if (Group_User_Id == 0) {
        throw 'Group_User_Id不应该是0'
    }
    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group_user/query', {
            Group_User_Id: Group_User_Id,
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            const Group_User__table: Group_User__table_interface[] = (await response).data.Data
            return Group_User__table
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 组管理员设定
 */
export async function Group_User__Administrator(Group_User__Id: number, Administrator: boolean): Promise<void> {
    if (Group_User__Id == 0) {
        throw "Group_User__Id不应该是0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group_user/administrator',
            {
                Group_User__Id: Group_User__Id,
                Administrator: Administrator
            }
        )

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 用户分组增加
 */
export async function Group_User__Add(Group_User: Group_User__table_interface): Promise<void> {
    if (Group_User.Id != 0) {
        throw "id应该等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group_user/add', Group_User)

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 用户分组删除
 */
export async function Group_User__Del(Group_User__Id: number): Promise<void> {
    if (Group_User__Id == 0) {
        throw "id应该不等于0"
    }

    try {
        const response = axios.post(http_Front_url + '/api/gui/v1.0/group_user/del', {
            Group_Id: Group_User__Id
        })

        const status = (await response).status
        if (status == 200) {
            ElMessage({
                message: (await response)?.data?.Msg || 'ok',
                type: 'success',
            })
            return
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        const axiosError = error as { code?: string; response?: { data?: { Msg?: string }, status: number } }
        if (axiosError.code == "ERR_NETWORK") {
            ElMessage({ message: '请求超时', type: 'error' })
            throw '请求超时'
        }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

