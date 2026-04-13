/*
* 日期：2026.3.31 PM9:59
* 作者：范范 zwf
* 作用：配置服务接口
 */


import axios from 'axios'
import { ElMessage } from 'element-plus'
import { config_service_url } from '@/api/index'



/**
*******************用户*******************
*/
/**
 * 采集配置配置增加表接口
 */
export interface Collector_Info__Add_interface {
    Label: string // 标识
    Uuid: string // Uuid
    Name: string // 设备名称
    User_Id: number   // 用户 id
}

/**
 * 采集配置配置更新表接口
 */
export interface Collector_Info__Update_interface {
    Id: number // 采集 Id 
    Name: string // 设备名称 
}
/**
 * 采集配置配置表接口
 */
export interface Collector_Info__table_interface {
    Id: number      // 采集 Id
    Label: string    // 标识
    Creation_Time: string// 创建时间
    Uuid: string    // Uuid (修正为 string)
    Sn: string    // 设备 sn
    User_Id: number        // 创建用户 id
    User_Name: string        // 创建用户名
    Version: string    // 版本
    Last_Activity_Time: string // 最后活动时间
    Equipment_Id: number        // 设备 id
    Name: string    // 设备名称
}


/**
 * 采集 -》查询数量
 * 传递：page 页码，pageSize 每页数量 返回：Count 数量
 */
export async function Collector_Info__Count(params?: {
    Page?: number; Page_Size?: number;
}): Promise<number> {
    try {
        // 修改：直接 await axios.post，移除多余的二次 await
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/count', {
            Page: params?.Page,
            Page_Size: params?.Page_Size,
        })

        if (response.status == 200) {
            return response.data.Data as number
        }
        throw response.data.Msg || '未知错误';
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
 * 采集 -》查询配置 
 * 传递：传递：page 页码，pageSize 每页数量 返回：configs 配置列表
 */
export async function Collector_Info__Query(params?: {
    Page?: number; Page_Size?: number;
}): Promise<Collector_Info__table_interface[]> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/query', {
            Page: params?.Page,
            Page_Size: params?.Page_Size,
        })

        if (response.status == 200) {
            return response.data.Data as Collector_Info__table_interface[]
        }
        throw response.data.Msg || '未知错误';
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
 * 采集 -》增加配置
 * 传递：config 配置数组形式
 */
export async function Collector_Info__Add(add: Collector_Info__Add_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/add', add)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 采集 -》增加配置
 * 传递：config 配置数组形式
 */
export async function Collector_Info__Update(add: Collector_Info__Update_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/update', add)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 采集 -》增加删除
 * 传递：Id 需要删除的 id
 */
export async function Collector_Info__Del(Id: number): Promise<void> {
    if (Id == 0) {
        throw '请选择需要删除的配置'
    }

    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/del', {
            Id: Id
        })

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 采集-》搜索
 * 传递：field quantity 数量，vague 模糊搜索字符串 返回：configs 配置，err 错误
 */
export async function Collector_Info__Search_Name(params?: {
    Field: string; Quantity: number; Vague: string;
}): Promise<Collector_Info__table_interface[]> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/collector_info/search', {
            Field: params?.Field,
            Quantity: params?.Quantity,
            Vague: params?.Vague
        })

        if (response.status == 200) {
            return response.data.Data as Collector_Info__table_interface[]
        }
        throw response.data.Msg || '未知错误';
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
*******************驱动*******************
*/

/**
 * 驱动配置增加表接口
 */
export interface Drive_Config__add_interface {
    Name: string // 驱动名称
    Config: string // json 配置参数
    Type: string // 驱动类型
    Collector_Id: number   // 采集器标识
}

/**
 * 驱动配置更新表接口
 */
export interface Drive_Config__Update_interface {
    Id: number   // 驱动 id
    Name: string // 驱动名称
    Config: string // json 配置参数
}


/**
 * 驱动配置配置表接口
 */
export interface Drive_Config__table_interface {
    Id: number   // 驱动 id
    Name: string // 驱动名称
    Config: string // json 配置参数
    Type: string    // 驱动类型
    Points_Length: number      // 点位数量
    Collector_Id: number      // 采集器标识
    Creation_Time: string// 创建时间
    Collector_Name: string // 采集器名称
}

/**
 * 驱动配置 -》查询数量
 * 传递：Page 页码，Page_Size 每页数量，Collector_Id 采集器标识，Drive_Type 驱动类型 返回：Count 数量
 */

export async function Drive_Config__Count(params?: {
    Page?: number; Page_Size?: number; Collector_Id?: number;
    Drive_Type?: string;
}): Promise<number> {
    // 默认值
    const {
        Page = 0,
        Page_Size = 0,
        Collector_Id = 0,
        Drive_Type = ""
    } = params || {};
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/drive/count', {
            Page: Page,
            Page_Size: Page_Size,
            Collector_Id: Collector_Id,
            Drive_Type: Drive_Type,
        })

        if (response.status == 200) {
            return response.data.Data as number
        }
        throw response.data.Msg || '未知错误';
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
 * 驱动配置 -》查询配置
 * 传递：Page 页码，Page_Size 每页数量，Collector_Id 采集器标识，Drive_Type 驱动类型 返回：配置列表
 */
export async function Drive_Config__Query(params?: {
    Page?: number; Page_Size?: number; Collector_Id?: number;
    Drive_Type?: string;
}): Promise<Drive_Config__table_interface[]> {
    // 默认值
    const {
        Page = 0,
        Page_Size = 0,
        Collector_Id = 0,
        Drive_Type = ""
    } = params || {}

    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/drive/query', {
            Page: Page,
            Page_Size: Page_Size,
            Collector_Id: Collector_Id,
            Drive_Type: Drive_Type,
        })

        if (response.status == 200) {
            return response.data.Data as Drive_Config__table_interface[]
        }
        throw response.data.Msg || '未知错误';
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
 * 驱动配置 -》增加配置
 * 传递：config 配置对象，包含 Name 驱动名称，Config json 配置参数，Type 驱动类型，Collector_Id 采集器标识
 */
export async function Drive_Config__Add(config: Drive_Config__add_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/drive/add', config)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 驱动配置 -》更新配置
 * 传递：config 配置对象，包含 Id 驱动 id, Name 驱动名称，Config json 配置参数，Type 驱动类型，Collector_Id 采集器标识
 */
export async function Drive_Config__Update(config: Drive_Config__Update_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/drive/update', config)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 驱动配置 -》更新配置
 * 传递：config 配置对象，包含 Id 驱动 id, Name 驱动名称，Config json 配置参数，Type 驱动类型，Collector_Id 采集器标识
 */
export async function Drive_Config__Del(Id: number): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/drive/del', {
            Id: Id
        })

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 采集-》搜索
 * 传递：field quantity 数量，vague 模糊搜索字符串 返回：configs 配置，err 错误
 */
export async function Drive_Config__Search_Name(params?: {
    Field: string; Quantity: number; Vague: string;
}): Promise< Drive_Config__table_interface[]> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/drive/search', {
            Field: params?.Field,
            Quantity: params?.Quantity,
            Vague: params?.Vague
        })

        if (response.status == 200) {
            return response.data.Data as Drive_Config__table_interface[]
        }
        throw response.data.Msg || '未知错误';
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
*******************点位*******************
*/


/**
 * 点位配置更新表接口
 */
export interface Points_Config__Update_interface {
    Id: number   // 点位 id
    Tag: string // 点位标识
    Description: string // 点位描述
    RW_Cancel: string // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
    Value_Type: string // 输出类型
    Config: string
}

/**
 * 点位配置增加表接口
 */
export interface Points_Config__add_interface extends Points_Config__Update_interface {
    Drive_Id: number   // 点位 id 唯一标识符 
    Drive_Type: string // 驱动类型
}

/**
 * 点位配置配置表接口
 */
export interface Points_Config__table_interface extends Points_Config__Update_interface {
    Creation_Time: string // 创建时间  
    Drive_Type: string // 驱动类型 
}


/**
 * 点位配置 -》查询数量
 * 传递：Page 页码，Page_Size 每页数量，Drive_Id 驱动 id 返回：Count 数量
 */

export async function Points_Config__Count(params?: {
    Page?: number; Page_Size?: number; Drive_Id?: number;
}): Promise<number> {
    // 默认值
    const {
        Page = 0,
        Page_Size = 0,
        Drive_Id = 0,
    } = params || {}
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/points/count', {
            Page: Page,
            Page_Size: Page_Size,
            Drive_Id: Drive_Id,
        })

        if (response.status == 200) {
            return response.data.Data as number
        }
        throw response.data.Msg || '未知错误';
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
 * 点位配置 -》查询配置
 * 传递：Page 页码，Page_Size 每页数量，Drive_Id 驱动 id 返回：配置列表
 */
export async function Points_Config__Query(params?: {
    Page?: number; Page_Size?: number; Drive_Id?: number;
}): Promise<Points_Config__table_interface[]> {
    // 默认值
    const {
        Page = 0,
        Page_Size = 0,
        Drive_Id = 0,
    } = params || {}
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/points/query', {
            Page: Page,
            Page_Size: Page_Size,
            Drive_Id: Drive_Id,
        })

        if (response.status == 200) {
            return response.data.Data as Points_Config__table_interface[]
        }
        throw response.data.Msg || '未知错误';
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
 * 点位配置 -》增加配置
 * 传递：config 配置对象，包含 Name 点位名称，Config json 配置参数，Type 点位类型，Drive_Id 驱动 id
 */
export async function Points_Config__Add(config: Points_Config__add_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/points/add', config)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 点位配置 -》更新配置
 * 传递：config 配置对象，包含 Id 点位 id, Name 点位名称，Config json 配置参数，Type 点位类型，Drive_Id 驱动 id
 */
export async function Points_Config__Update(config: Points_Config__Update_interface): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/points/update', config)

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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
 * 点位配置 -》删除配置
 * 传递：Id 点位 id
 */
export async function Points_Config__Del(Id: number): Promise<void> {
    try {
        // 修改：直接 await axios.post
        const response = await axios.post(config_service_url + '/api/gui/v1.0/config/points/del', {
            Id: Id
        })

        if (response.status == 200) {
            return
        }
        throw response.data.Msg || '未知错误';
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