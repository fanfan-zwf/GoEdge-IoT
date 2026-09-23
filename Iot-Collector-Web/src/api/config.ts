import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({
  baseURL: 'http://192.168.220.40:8103/api/v1.0',
  timeout: 10000
})

// 根据 Code 范围显示不同类型的消息提示
function showMessage(code: number, msg: string) {
  if (code >= 200 && code < 300) {
    ElMessage.success(msg)
  } else if (code >= 300 && code < 400) {
    ElMessage({ message: msg, type: 'info' })
  } else if (code >= 400 && code < 500) {
    ElMessage.warning(msg)
  } else {
    ElMessage.error(msg)
  }
}

// 响应拦截器 - 统一处理后端返回格式 { Code, Msg, Data, Timestamp }
http.interceptors.response.use(
  (response) => {
    const res = response.data
    const code = res.Code
    if (code >= 200 && code < 300) {
      return res
    }
    showMessage(code, res.Msg || '请求失败')
    return Promise.reject(new Error(res.Msg || '请求失败'))
  },
  (error) => {
    if (error.response?.data) {
      const res = error.response.data
      const code = res.Code || error.response.status
      showMessage(code, res.Msg || error.message)
    } else {
      ElMessage.error(error.message || '网络请求失败')
    }
    return Promise.reject(error)
  }
)

// ======== 配置管理 API ========

export const configApi = {
  // 驱动配置
  drive: {
    query: (params: { page: number; pageSize: number }) =>
      http.post('/config/drive/query', params),
    count: (params: { page: number; pageSize: number }) =>
      http.post('/config/drive/count', params),
    add: (data: any[]) => http.post('/config/drive/add', data),
    update: (data: any[]) => http.post('/config/drive/update', data),
    del: (ids: number[]) => http.post('/config/drive/del', { ids }),
    restart: (id: number) => http.post('/config/drive/restart', { id })
  },
  // 点位配置
  point: {
    query: (params: { driveId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/point/query', params),
    count: (params: { driveId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/point/count', params),
    add: (data: any[]) => http.post('/config/point/add', data),
    update: (data: any[]) => http.post('/config/point/update', data),
    del: (ids: number[]) => http.post('/config/point/del', { ids }),
    readValue: (pointIds: number[]) =>
      http.post('/point/read/value', pointIds),
    writeValue: (data: { PointId: number; Value: any; Type: string }[]) =>
      http.post('/point/write/value', data)
  },
  // 报警配置
  alarm: {
    query: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/alarm/query', params),
    count: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/alarm/count', params),
    add: (data: any[]) => http.post('/config/alarm/add', data),
    update: (data: any[]) => http.post('/config/alarm/update', data),
    del: (ids: number[]) => http.post('/config/alarm/del', { ids }),
    status: (pointIds: number[]) =>
      http.post('/alarm/status', pointIds)
  },
  // 历史配置
  history: {
    query: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/history/query', params),
    count: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post('/config/history/count', params),
    add: (data: any[]) => http.post('/config/history/add', data),
    update: (data: any[]) => http.post('/config/history/update', data),
    del: (ids: number[]) => http.post('/config/history/del', { ids })
  },
  // 历史数据查询（InfluxDB）
  historyData: {
    query: (params: {
      pointId: number
      startTime: string
      endTime: string
      page: number
      pageSize: number
    }) => http.post('/history/data/query', params)
  }
}

// 导出 Excel URL
export const exportUrl = 'http://192.168.220.40:8103/api/v1.0/config/export'
