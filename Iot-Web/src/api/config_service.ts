/*
* 日期: 2026.3.31 PM9:59
* 作者: 范范zwf
* 作用: 配置服务接口
 */


import axios from 'axios'
import { ElMessage } from 'element-plus'
import { config_service_url } from '@/api/index'



/**
*******************用户*******************
*/

/**
 * 用户接口
 */
export interface Collector_Info__table_interface {
    Id: number //  ID
    Device_Id: string // 设备ID
    Label: string // 标识
    Creation_Time: number   // 创建时间
    Uuid: boolean    // Uuid
    Sn: string  // 序列号
    User_Id: string  // 用户id  
    Version: number    // 版本号 
    Last_Activity_Time: number    // 最后活动时间
    Name: number    // 刷新令牌过期时间（s）

}


/**
 * 采集-》查询数量
 * 传递: page 页码, pageSize 每页数量 返回: Count 数量
 */
export async function Collector_Info__Count(Page: number = 0, Page_Size: number = 0): Promise<number> {
    try {
        const response = axios.post(config_service_url + '/api/gui/v1.0/Collector_info/count', {
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            return (await response).data.Data as number
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
 * 采集-》查询配置 
 * 传递: 传递: page 页码, pageSize 每页数量 返回: configs 配置列表
 */
export async function Collector_Info__Query(Page: number = 0, Page_Size: number = 0): Promise<Collector_Info__table_interface> {
    try {
        const response = axios.post(config_service_url + '/api/gui/v1.0/Collector_info/query', {
            Page: Page,
            Page_Size: Page_Size,
        })

        const status = (await response).status
        if (status == 200) {
            return (await response).data.Data as Collector_Info__table_interface
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
