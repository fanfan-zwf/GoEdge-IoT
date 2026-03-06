import { http_Front_url } from '@/typer/index'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { sha3_256_sync } from '@/typer/function'


/**
*******************用户*******************
*/

/**
 * 用户接口
 */
export interface User__table_interface {
    Id: number // 用户ID
    Name: string // 用户名
    Permissions: number   // 权限
    Refresh_Token_Time: number   // 过期时间设定（s）
    Discontinued: boolean    // 停用 
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
export async function User__Get_Info(User_Id: number = 0, Again: boolean = false): Promise<User__table_interface> {
    if (User_Id == 0) {
        const data = sessionStorage.getItem('F_User_Info') || null;
        if (data != null && !Again) {
            return JSON.parse(data) as User__table_interface;
        }
    }


    try {
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Get/Info', {
            User_Id: User_Id
        })

        const status = (await response).status
        if (status == 200) {
            const User_info: User__table_interface = (await response).data.Data
            if (User_Id == 0) {
                sessionStorage.removeItem('F_User_Info');
                sessionStorage.setItem('F_User_Info', JSON.stringify(User_info))
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
        ElMessage({ message: axiosError?.response?.data?.Msg || '请求失败', type: 'error' })
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
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Get/Info_Array', {
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
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Get/Search', {
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
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Set/Add', value)
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
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Set/Name', {
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
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Set/Passwd', {
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
 * 删除用户
 * Param Email 新邮箱, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Del(User_Id: number): Promise<void> {
    if (User_Id == 0) {
        throw 'User_Id不能是0'
    }
    try {
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Set/Del', {
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
 * 获取用户条数
 */
export async function User__All_Count(): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Get/Count')

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
 * 分页查询用户
 */
export async function User__All_Query(Page: number, Page_Size: number): Promise<User__table_interface> {
    try {
        const response = axios.post(http_Front_url + '/Gui/v1.0/User/Get/Query', {
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



// 获取点位数量 传递驱动id，0代表全部
export async function Points_length(id: number): Promise<number> {
    try {
        // 修改
        const response = axios.post(http_Front_url + '/api/v1.0/IO/points/length', {
            Id: id
        })

        const status = (await response).status
        if (status == 200) {
            const a: number = (await response).data.Data
            return a
        }
        return 0
    } catch (error: unknown) {
        const axiosError = error as { response?: { data?: { Msg?: string } } }
        ElMessage({
            message: axiosError.response?.data?.Msg || '请求失败',
            type: 'error',
        })
        return 0
    }
}