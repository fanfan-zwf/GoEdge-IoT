import { http_Front_url } from '@/typer/index'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { sha3_256_sync } from '@/utils/function'


/**
 * 通用的Axios错误处理函数（提取catch中的逻辑）
 * @param error 捕获的错误对象
 * @returns 格式化后的错误信息
 */
function handleAxiosError(error: unknown): string {
    // 类型守卫：更安全的类型断言
    const axiosError = error as {
        code?: string;
        response?: {
            data?: { Msg?: string },
            status: number
        };
        message?: string;
    };

    // 细分网络错误类型，提示更精准
    let errorMsg = '';
    if (axiosError.code === "ERR_NETWORK") {
        errorMsg = '网络异常，请检查网络连接';
        ElMessage({ message: errorMsg, type: 'error' });
    } else if (axiosError.code === "ECONNABORTED") {
        errorMsg = '请求超时，请稍后重试';
        ElMessage({ message: errorMsg, type: 'error' });
    } else {
        // 后端返回的错误信息
        errorMsg = axiosError.response?.data?.Msg || '请求失败';
        ElMessage({
            message: errorMsg,
            type: 'error',
        });
    }

    return errorMsg;
}

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
        const response = axios.post(http_Front_url + '/gui/v1.0/user/get/info', {
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
        handleAxiosError(error);
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
        const response = axios.post(http_Front_url + '/gui/v1.0/user/get/info_array', {
            User_Id_Array: User_Id_Array
        })

        const status = (await response).status
        if (status == 200) {
            const User_info: User__table_interface[] = (await response).data.Data
            return User_info
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        handleAxiosError(error);
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
        const response = axios.post(http_Front_url + '/gui/v1.0/user/get/search', {
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
        handleAxiosError(error);
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
        const response = axios.post(http_Front_url + '/gui/v1.0/user/set/add', value)
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
        handleAxiosError(error);
    }

}

/**
 * 设置用户名
 * Param Name 新用户名, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Name(Name: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/gui/v1.0/user/set/name', {
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
        handleAxiosError(error);
    }

}

/**
 * 设置密码
 * Param Passwd 新密码, User_Id 用户ID(默认0-当前用户)
 */
export async function User__Set_Passwd(Passwd: string, User_Id: number = 0): Promise<void> {
    try {
        const response = axios.post(http_Front_url + '/gui/v1.0/user/set/passwd', {
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
        handleAxiosError(error);
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
        const response = axios.post(http_Front_url + '/gui/v1.0/user/set/del', {
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
        handleAxiosError(error);
    }
}

/**
 * 获取用户条数
 */
export async function User__All_Count(): Promise<number> {
    try {
        const response = axios.post(http_Front_url + '/gui/v1.0/user/get/count')

        const status = (await response).status
        if (status == 200) {
            const Count: number = (await response).data.Data
            return Count
        }
        throw (await response).data.Msg || '未知错误';
    } catch (error: unknown) {
        handleAxiosError(error);
    }
}

/**
 * 分页查询用户
 */
export async function User__All_Query(Page: number, Page_Size: number): Promise<User__table_interface> {
    try {
        const response = axios.post(http_Front_url + '/gui/v1.0/user/get/query', {
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
        handleAxiosError(error);
    }
}



/**
*******************驱动*******************
*/

export interface Drive_Config_type {
    Id: number,   // 驱动id
    Type: string, // 驱动类型
    Name: string, // 驱动名称
    Points_Length: number, // 点位数量
    Config: string  // json配置参数
}

/**
 * 驱动-》查询数量
 * 传入参数：驱动类型，页数，每页数量
 * 返回：数量
 */
export async function Drive_Config__Count(Drive_Type: string,
    Page: number,
    Page_Size: number): Promise<number> {
    try {
        // 修复：补充缺失的await，避免重复await response
        const response = await axios.post(http_Front_url + '/gui/v1.0/config/drive/count', {
            Drive_Type: Drive_Type,
            Page: Page,
            Page_Size: Page_Size
        });

        // 简化：只获取一次response属性，避免重复await和取值
        const { status, data } = response;

        if (status === 200) {
            // 增加类型安全：确保返回值是数字类型
            const count = Number(data.Data);
            if (isNaN(count)) {
                throw new Error('返回的总数不是有效数字');
            }
            return count;
        }

        // 非200状态码抛出错误
        throw new Error(data.Msg || '未知错误');

    } catch (error: unknown) {
        handleAxiosError(error);
    }
}

/**
 * 驱动-》查询配置
 * 传入参数：驱动类型，页数，每页数量
 * 返回：配置数组
 */
export async function Drive_Config__Query(Drive_Type: string,
    Page: number,
    Page_Size: number): Promise<Drive_Config_type[]> {
    try {
        // 修复：补充缺失的await，避免重复await response
        const response = await axios.post(http_Front_url + '/gui/v1.0/config/drive/query', {
            Drive_Type: Drive_Type,
            Page: Page,
            Page_Size: Page_Size
        });

        // 简化：只获取一次response属性，避免重复await和取值
        const { status, data } = response;

        if (status === 200) {
            // 增加类型安全：确保返回值是数字类型 
            return data.Data as Drive_Config_type[];
        }

        // 非200状态码抛出错误
        throw new Error(data.Msg || '未知错误');

    } catch (error: unknown) {
        handleAxiosError(error);
    }
}

/**
 * 驱动-》增加
 * 传入参数： 驱动配置
 * 返回：无
 */
export async function Drive_Config__Add(cfg: Drive_Config_type): Promise<void> {
    if (cfg.Id != 0) {
        throw new Error('新增驱动配置时Id必须为0');
    }

    if (cfg.Points_Length == 0) {
        throw new Error('新增驱动配置时Points_Length必须为0');
    }

    try {
        // 修复：补充缺失的await，避免重复await response
        const response = await axios.post(http_Front_url + '/gui/v1.0/config/drive/add', cfg);

        // 简化：只获取一次response属性，避免重复await和取值
        const { status, data } = response;

        if (status === 200) {
            // 增加类型安全：确保返回值是数字类型 
            return;
        }

        // 非200状态码抛出错误
        throw new Error(data.Msg || '未知错误');

    } catch (error: unknown) {
        handleAxiosError(error);
    }
}

/**
 * 驱动-》更新
 * 传入参数： 驱动配置
 * 返回：无
 */
export async function Drive_Config__Update(cfg: Drive_Config_type): Promise<void> {
    if (cfg.Id == 0) {
        throw new Error('更新驱动配置时Id不能为0');
    }

    if (cfg.Points_Length == 0) {
        throw new Error('更新驱动配置时Points_Length必须为0');
    }

    try {
        // 修复：补充缺失的await，避免重复await response
        const response = await axios.post(http_Front_url + '/gui/v1.0/config/drive/update', cfg);

        // 简化：只获取一次response属性，避免重复await和取值
        const { status, data } = response;

        if (status === 200) {
            // 增加类型安全：确保返回值是数字类型 
            return;
        }

        // 非200状态码抛出错误
        throw new Error(data.Msg || '未知错误');

    } catch (error: unknown) {
        handleAxiosError(error);
    }
}

/**
 * 驱动-》删除
 * 传入参数： 驱动配置
 * 返回：无
 */
export async function Drive_Config__Del(id: number): Promise<void> {
    if (id == 0) {
        throw new Error('删除驱动配置时Id不能为0');
    }

    try {
        const response = await axios.post(http_Front_url + '/gui/v1.0/config/drive/del', { Id: id });
        const { status, data } = response;

        if (status === 200) {
            return;
        }

        throw new Error(data.Msg || '未知错误');
    } catch (error: unknown) {
        handleAxiosError(error);
    }
 
}

/**
*******************点位*******************
*/

export interface Drive_Config_type {
    Id: number,   // 驱动id
    Type: string, // 驱动类型
    Name: string, // 驱动名称
    Points_Length: number, // 点位数量
    Config: string  // json配置参数
}
