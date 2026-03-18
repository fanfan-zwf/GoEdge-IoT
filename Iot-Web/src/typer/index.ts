import { useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'


const router = useRouter()





// export type Persons = Array<PersonInter>
// export const ip = '192.168.31.32'
// export const ip = '192.168.31.123'
export const ip = '192.168.220.20'
export const port = 8078

export const http_Front_url = `http://${ip}:${port}`
export const ws_Front_url = `ws://${ip}:${port}`


// export const http_Front_url = `/api`
// export const ws_Front_url = `ws:/api`


// export const apiClient = axios.create({
//     baseURL: http_Front_url, // 设置基础URL
// });



