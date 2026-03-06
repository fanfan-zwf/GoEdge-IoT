import { http_Front_url } from '@/typer/index'
import axios from 'axios'
import router from '@/router/index'
import { DualMutex } from '@/typer/function'
import { sha3_256_sync } from '@/typer/function'


const mutex = new DualMutex();
// const router = useRouter()
const MAX_ACCESS_ATTEMPTS = 2;
const QUEUE_TIMEOUT = 30000; // 队列请求超时时间 30秒


// 存储状态  
let failedQueue: { isCancelled: boolean; config: any; resolve: (token: any) => void; reject: (err: any) => void }[] = []; // 失败请求队列

// 队列处理器
const processQueue = (error: unknown, token: any = null) => {
    failedQueue.forEach(prom => {
        if (prom.isCancelled) {
            console.log('跳过已取消的请求');
            return;
        }

        if (error) {
            prom.reject(error);
        } else {
            prom.resolve(token);
        }

    });

    failedQueue = [];
};
// 取消所有队列请求
const cancelAllQueuedRequests = (message = '请求被取消') => {
    console.log(`取消 ${failedQueue.length} 个队列中的请求`);

    const error = new Error(message);

    // 1. 先标记所有请求为已取消
    failedQueue.forEach(prom => {
        prom.isCancelled = true;
    });

    // 2. 调用 processQueue 处理这些请求（传递错误）
    processQueue(error, null);

    // 注意：processQueue 内部已经清空了 failedQueue
    // 所以这里不需要再次清空
};

// 添加请求拦截器
axios.interceptors.request.use(
    function (config) {
        // 在发送请求之前添加令牌
        const tokenString = localStorage.getItem('F_Access_Token') || null
        if (tokenString == null) {
            console.log('请求拦截器未找到令牌', config.url);
            return config;
        }
        const token = JSON.parse(tokenString) as localStorage_Access_Token_interface
        if (token && !config.url?.includes('/Gui/v1.0/Login')) {
            config.headers['F_Access_Token'] = token.F_Access_Token;
            console.log('请求拦截器添加令牌', config.url);
        }
        return config;
    },
    function (error) {
        console.log('请求拦截器错误', error);
        return Promise.reject(error);
    }
);

// 添加响应拦截器
axios.interceptors.response.use(
    function (response) {
        // 响应拦截器成功 
        return response;
    },
    async function (error) {
        const originalRequest = error.config;
        // 排除登录接口 
        if (originalRequest?.url?.includes('/Gui/v1.0/Login') || error.status != 401) {
            console.log('排除登录接口', originalRequest?.url);
            return Promise.reject(error);
        }

        // 修改这里 ↓↓↓
        if (!mutex.tryLock()) {
            console.log('令牌正在刷新中，当前请求加入等待队列');
            return new Promise((resolve, reject) => {
                // 设置30秒超时
                const timer = setTimeout(() => {
                    reject(new Error('请求队列等待超时'));
                }, 6 * 1000); // 6秒

                failedQueue.push({
                    isCancelled: false,
                    config: originalRequest,
                    resolve: (token) => {
                        clearTimeout(timer);
                        originalRequest.headers['Access_Token'] = token; // ✅ 修复：设置令牌
                        resolve(axios(originalRequest)); // ✅ 修复：发起请求
                    },
                    reject: (err) => {
                        clearTimeout(timer);
                        reject(err);
                    }
                });
            });
        }


        // 如果已经在刷新令牌，将当前请求加入等待队列

        // 处理401未授权错误
        if (error.response && error.response.status === 401) {
            console.log('检测到401错误，开始处理');


            // 标记为正在刷新令牌 
            originalRequest._retry = true;

            try {
                console.log('开始刷新令牌...');
                const newTokenData = await Api_Access_Token_update();
                const newToken = newTokenData.F_Access_Token;

                console.log('令牌刷新成功:', newToken);

                // 保存新令牌
                localStorage.setItem('F_Access_Token', newTokenData ? JSON.stringify(newTokenData) : '');
                // 使用新令牌重新发送原始请求
                originalRequest.headers['F_Access_Token'] = newToken;
                const retryResponse = await axios(originalRequest);

                // 处理队列中的所有等待请求
                console.log(`处理 ${failedQueue.length} 个等待中的请求`);
                processQueue(null, newTokenData);
                return retryResponse;

            } catch (refreshError) {
                console.error('令牌刷新失败:', refreshError);

                cancelAllQueuedRequests('令牌刷新失败，取消请求');  // 取消所有队列中的请求

                // 清除本地令牌
                localStorage.removeItem('F_Access_Token');
                localStorage.removeItem('F_Refresh_Token');

                sessionStorage.removeItem('F_User_Info');

                // 跳转到登录页面
                router.push({ name: 'login' });

                return Promise.reject(refreshError);
            } finally {
                mutex.unlock();
            }
        }

        // 其他错误
        console.log('响应拦截器错误', error);
        return Promise.reject(error);
    }
);

/*
*******************刷新令牌*******************
*/

/**
 * 判断刷新令牌时间是否过期
 * param time 过期时间
 * return boolean 未过期true，已过期false
 */
export function Expires_in_judgment(time: string): boolean {
    const targetDate = new Date(time);
    const now = new Date();

    if (targetDate.getTime() < now.getTime()) {
        return false
    }
    //目标时间还未到 
    return true
}

/**
 * 本地存储刷新令牌
 */
export interface localStorage_Refresh_Token_interface {
    User_Id: number
    F_Refresh_Token: string
    F_Expires_in: string
}

/**
 * 更新刷新令牌-用户登录
 * param Name 用户名, Passwd 密码
 */
export async function Api_Name_login_Refresh_Token_update(Name: string, Passwd: string): Promise<localStorage_Refresh_Token_interface> {
    try {
        // 修改
        const response = axios.post(http_Front_url + '/Gui/v1.0/Login/Name', {
            Name: Name,
            Passwd: sha3_256_sync(0, Passwd)
        })

        const status = (await response).status
        if (status == 200) {
            const Refresh_Token: localStorage_Refresh_Token_interface = (await response).data.Data
            console.log(Refresh_Token)
            localStorage.setItem('F_Refresh_Token', JSON.stringify(Refresh_Token)) // 写入本地存储
            // router.push("/")
            return Refresh_Token
        }
        throw (await response).data.Msg ?? '请求失败';
    } catch (error: unknown) {
        const axiosError = error as { response?: { data?: { Msg?: string }, status: number } }
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 获取刷新令牌
 */
export function Refresh_Token_Query(): localStorage_Refresh_Token_interface {
    const Cloud_configure_token = localStorage.getItem('F_Refresh_Token') || null
    if (Cloud_configure_token == null) {
        throw '未找到Refresh_Token'
    }
    const token = JSON.parse(Cloud_configure_token) as localStorage_Refresh_Token_interface
    if (!Expires_in_judgment(token.F_Expires_in)) {
        router.push({ name: 'login' })
        throw 'Refresh_Token已过期'
    }
    return token
}

/**
 * 本地存储刷新令牌
 */
export interface localStorage_Access_Token_interface {
    F_Access_Token: string
    F_Expires_in: string;
}

/**
 * 更新访问令牌
 */
export async function Api_Access_Token_update(): Promise<localStorage_Access_Token_interface> {
    console.log('正在刷新Access_Token')
    const Refresh_Token_value = Refresh_Token_Query()
    if (!Refresh_Token_value) {
        router.push({ name: 'login' })
        throw 'Refresh_Token获取失败'
    }
    try {
        const response = axios.post(http_Front_url + '/Gui/v1.0/Login/Access_Token', {
            User_Id: Refresh_Token_value?.User_Id,
            F_Refresh_Token: Refresh_Token_value?.F_Refresh_Token
        })

        const status = (await response).status
        if (status == 200) {
            const Access_Token: localStorage_Access_Token_interface = (await response).data.Data
            localStorage.setItem('F_Access_Token', JSON.stringify(Access_Token)) // 写入本地存储
            axios.defaults.headers.common['F_Access_Token'] = Access_Token.F_Access_Token
            return Access_Token
        }
        throw (await response).data.Msg || '请求失败'
    } catch (error: unknown) {
        const axiosError = error as { response?: { data?: { Msg?: string }, status: number } }
        throw axiosError.response?.data?.Msg || '请求失败';
    }
}

/**
 * 获取访问令牌
 */
export async function Access_Token_Query(): Promise<localStorage_Access_Token_interface> {

    const Cloud_configure_token = localStorage.getItem('F_Access_Token') || null
    if (Cloud_configure_token == null) {
        const Access_Token = await Api_Access_Token_update()
        if (!Access_Token) {
            throw 'Access_Token获取失败'
        }
        axios.defaults.headers.common['F_Access_Token'] = Access_Token.F_Access_Token
        return Access_Token
    } else {
        let token = JSON.parse(Cloud_configure_token) as localStorage_Access_Token_interface
        if (!Expires_in_judgment(token.F_Expires_in)) {
            console.log('Access_Token过期，正在刷新')
            const Access_Token = await Api_Access_Token_update()
            if (!Access_Token) {

            }
            axios.defaults.headers.common['F_Access_Token'] = token.F_Access_Token
            return Access_Token
        }
        return token
    }

}