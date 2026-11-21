import axios from "axios";
import { message } from "@/utils/message";
import { getToken, setToken } from "@/utils/auth";
import { getConfig } from "@/config";
import { normalizeServerAddr, getCurrentServerAddr, setCurrentServerAddr } from "@/utils/serverAddr";

// 服务器地址（IP:端口）优先级：localStorage 持久化的当前地址 > platform-config.json > 兜底。
// 登录页手输后写入 server-addr-current，刷新后自动恢复，不会丢失。
// 因本模块在应用启动前即被引入，platform-config.json 尚未加载，故用函数延迟取值。
//
// baseServerOverride：内存缓存，初值从 localStorage 恢复（localStorage 读取同步可用）。
let baseServerOverride = getCurrentServerAddr();

// setBaseServer 归一化并设置当前地址；persist=true 时同时写入 localStorage（刷新后保留）
export const setBaseServer = (value: string, persist = true): string => {
  baseServerOverride = normalizeServerAddr(value);
  if (persist && baseServerOverride) setCurrentServerAddr(baseServerOverride);
  return baseServerOverride;
};

// getBaseServer 优先级：当前地址（含持久化） > platform-config.json > 构建注入 VITE_BASE_SERVER > 同源回退。
// 不再写死内网 IP：未显式配置时回退到当前页面 host:port（边缘设备前端与后端通常同源），
// 跨域部署时由 platform-config.json 或构建期 VITE_BASE_SERVER 注入。
export const getBaseServer = (): string =>
  baseServerOverride ||
  (getConfig()?.BaseServer as string) ||
  (import.meta.env.VITE_BASE_SERVER as string) ||
  (typeof window !== "undefined" ? window.location.host : "");

// 解析最终 HTTP baseURL：与当前页面同源时使用相对路径 /api/v1.0（天然继承页面协议与端口，
// 避免混合内容并彻底摆脱写死地址）；配置了异机地址时才拼成绝对 http://。
export const resolveBaseURL = (): string => {
  const host = getBaseServer();
  if (typeof window !== "undefined" && host === window.location.host) return "/api/v1.0";
  return host ? `http://${host}/api/v1.0` : "/api/v1.0";
};

// 免携带 token 的鉴权接口路径（否则 401 时会陷入刷新死循环）
const AUTH_FREE_PATHS = ["/auth/login", "/auth/refresh-token"];

// 全局唯一 HTTP 实例：统一 baseURL 动态解析、Bearer 注入、401 单飞刷新+重放、
// {Code,Msg,Data} 解包、按 code 分类弹消息、blob 下载、刷新失败登出。
// 业务 API 均经本文件封装调用；如需新增接口请直接复用此 http，不要再建第二个实例。
export const http = axios.create({
  timeout: 10000
});

// 根据 Code 范围显示不同类型的消息提示
function showMessage(code: number, msg: string) {
  if (code >= 200 && code < 300) {
    message(msg, { type: "success" });
  } else if (code >= 300 && code < 400) {
    message(msg, { type: "info" });
  } else if (code >= 400 && code < 500) {
    message(msg, { type: "warning" });
  } else {
    message(msg, { type: "error" });
  }
}

// 判断是否为鉴权接口（登录/刷新），这类请求不注入 token、也不触发 401 自动刷新
function isAuthUrl(url?: string): boolean {
  return !!url && AUTH_FREE_PATHS.some(p => url.includes(p));
}

// 请求拦截器 - 除登录/刷新外，统一注入 Bearer access_token
http.interceptors.request.use(config => {
  // 每次请求时动态解析 baseURL，确保取到 platform-config.json 中的最新地址
  config.baseURL = resolveBaseURL();
  if (!isAuthUrl(config.url)) {
    const token = getToken();
    if (token?.accessToken) {
      config.headers["Authorization"] = `Bearer ${token.accessToken}`;
    }
  }
  return config;
});

/** 刷新令牌单飞：并发 401 时只发一次刷新请求，其余请求等待同一 Promise */
let refreshPromise: Promise<string> | null = null;
function refreshAccessToken(): Promise<string> {
  if (refreshPromise) return refreshPromise;
  const current = getToken();
  if (!current?.refreshToken) {
    return Promise.reject(new Error("缺少刷新令牌"));
  }
  refreshPromise = http
    .post("/auth/refresh-token", { refresh_token: current.refreshToken })
    .then((res: any) => {
      const d = res?.Data || {};
      const expiresMs = Date.now() + Number(d.expires_in || 900) * 1000;
      // 仅更新令牌本身（username/roles 等保持 localStorage 原值）
      setToken({
        accessToken: d.access_token,
        refreshToken: d.refresh_token,
        expires: new Date(expiresMs)
      });
      return d.access_token as string;
    })
    .finally(() => {
      refreshPromise = null;
    });
  return refreshPromise;
}

// 响应拦截器 - 统一处理后端返回格式 { Code, Msg, Data, Timestamp }
http.interceptors.response.use(
  response => {
    // 二进制下载（导出 Excel 等）：透传完整 response，不走 {Code} 解包
    if (response.config.responseType === "blob") {
      return response;
    }
    const res = response.data;
    const code = res.Code;
    if (code >= 200 && code < 300) {
      return res;
    }
    showMessage(code, res.Msg || "请求失败");
    return Promise.reject(new Error(res.Msg || "请求失败"));
  },
  async error => {
    const status = error.response?.status;
    const original = error.config;
    // 401（access_token 过期/无效）：非鉴权接口且未重放过 → 刷新令牌后重放一次
    if (status === 401 && original && !original._retry && !isAuthUrl(original.url)) {
      original._retry = true;
      try {
        const newToken = await refreshAccessToken();
        original.headers = original.headers || {};
        original.headers["Authorization"] = `Bearer ${newToken}`;
        return http(original);
      } catch (e: any) {
        // 刷新失败：真正登出（清 token + 跳登录页），而非仅弹提示
        // 若后端已返回具体原因（如“登录 IP 已变化，该账号已全部退出”），下方拦截器已按 Msg 弹过提示，不再重复弹通用文案
        if (!e?.response) {
          message("登录已过期，请重新登录", { type: "warning" });
        }
        try {
          const { useUserStoreHook } = await import("@/store/modules/user");
          await useUserStoreHook().logOut();
        } catch {
          /* 兼容登出异常，不阻断 */
        }
        return Promise.reject(error);
      }
    }
    if (error.response?.data) {
      let res: any = error.response.data;
      // blob 响应体中的错误（非 2xx）：先解析出 JSON 再取 Msg
      if (res instanceof Blob) {
        try {
          res = JSON.parse(await res.text());
        } catch {
          res = {};
        }
      }
      const code = res.Code || status;
      showMessage(code, res.Msg || error.message);
    } else {
      message(error.message || "网络请求失败", { type: "error" });
    }
    return Promise.reject(error);
  }
);

// ======== 系统信息 API ========

// formatUptime 将秒数格式化为运行时间字符串（与原后端语义一致）
function formatUptime(seconds: number): string {
  if (!seconds || seconds < 0) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时 ${mins}分钟`;
  if (hours > 0) return `${hours}小时 ${mins}分钟`;
  return `${mins}分钟`;
}

// pickData 从 Promise.allSettled 结果中安全取出 res.Data（单接口失败不影响整体）
function pickData(settled: PromiseSettledResult<any>): any {
  return settled.status === "fulfilled" ? settled.value?.Data : undefined;
}

// ======== 配置管理 API ========

export const configApi = {
  // 驱动配置
  drive: {
    query: (params: { page: number; pageSize: number }) => http.post("/config/drive/query", params),
    count: (params: { page: number; pageSize: number }) => http.post("/config/drive/count", params),
    add: (data: any[]) => http.post("/config/drive/add", data),
    update: (data: any[]) => http.post("/config/drive/update", data),
    del: (ids: number[]) => http.post("/config/drive/del", { ids }),
    restart: (id: number) => http.post("/config/drive/restart", { id }),
    search: (params: { name?: string; types?: string[]; quantity: number }) => http.post("/config/drive/search", params)
  },
  // 点位配置
  point: {
    query: (params: { driveId?: number[]; page: number; pageSize: number }) => http.post("/config/point/query", params),
    count: (params: { driveId?: number[]; page: number; pageSize: number }) => http.post("/config/point/count", params),
    add: (data: any[]) => http.post("/config/point/add", data),
    update: (data: any[]) => http.post("/config/point/update", data),
    del: (ids: number[]) => http.post("/config/point/del", { ids }),
    search: (params: { tag?: string; driveIds?: number[]; quantity: number }) =>
      http.post("/config/point/search", params),
    readValue: (pointIds: number[]) => http.post("/point/read/value", pointIds),
    writeValue: (data: { PointId: number; Value: any; Type: string }[]) => http.post("/point/write/value", data)
  },
  // 报警配置
  alarm: {
    query: (params: { pointId?: number[]; page: number; pageSize: number }) => http.post("/config/alarm/query", params),
    count: (params: { pointId?: number[]; page: number; pageSize: number }) => http.post("/config/alarm/count", params),
    add: (data: any[]) => http.post("/config/alarm/add", data),
    update: (data: any[]) => http.post("/config/alarm/update", data),
    del: (ids: number[]) => http.post("/config/alarm/del", { ids }),
    search: (params: { tag?: string; pointIds?: number[]; groups?: number[]; quantity: number }) =>
      http.post("/config/alarm/search", params),
    status: (pointIds: number[]) => http.post("/alarm/status", pointIds)
  },
  // 报警数据
  alarmData: {
    query: (params: { alarmId?: number[]; page: number; pageSize: number }) =>
      http.post("/config/alarm-data/query", params),
    count: (params: { alarmId?: number[]; page: number; pageSize: number }) =>
      http.post("/config/alarm-data/count", params)
  },
  // 历史配置
  history: {
    query: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post("/config/history/query", params),
    count: (params: { pointId?: number[]; page: number; pageSize: number }) =>
      http.post("/config/history/count", params),
    add: (data: any[]) => http.post("/config/history/add", data),
    update: (data: any[]) => http.post("/config/history/update", data),
    del: (ids: number[]) => http.post("/config/history/del", { ids })
  },
  // 历史数据查询（InfluxDB）
  historyData: {
    query: (params: { pointId: number; startTime: string; endTime: string; page: number; pageSize: number }) =>
      http.post("/history/data/query", params)
  },
  // 点位 Tag 路径树（懒加载）
  tagTree: {
    children: (params: { tag_path: string }) => http.post("/tagtree/children", params),
    points: (params: { tag_path: string }) => http.post("/tagtree/points", params),
    rebuild: () => http.post("/tagtree/rebuild", {})
  },
  // 系统配置（config.yaml）
  systemConfig: {
    read: () => http.post("/system/config/read", {}),
    write: (data: any) => http.post("/system/config/write", data),
    reload: () => http.post("/system/config/reload", {})
  },
  // 系统性能信息
  systemInfo: {
    // ---- 单功能原始接口（返回 gopsutil 原始结果，供监控平台/调试直接使用）----
    host: () => http.get("/system/info/host"),
    hostUsers: () => http.get("/system/info/host/users"),
    virtualization: () => http.get("/system/info/host/virtualization"),
    sensors: () => http.get("/system/info/sensors"),
    cpuInfo: () => http.get("/system/info/cpu"),
    cpuCores: (logical = true) =>
      http.get("/system/info/cpu/cores", { params: { logical } }),
    cpuPercent: (percpu = false) =>
      http.get("/system/info/cpu/percent", { params: { percpu } }),
    cpuTimes: (percpu = false) =>
      http.get("/system/info/cpu/times", { params: { percpu } }),
    memory: () => http.get("/system/info/mem"),
    swap: () => http.get("/system/info/mem/swap"),
    swapDevices: () => http.get("/system/info/mem/swapdevices"),
    diskPartitions: (all = false) =>
      http.get("/system/info/disk", { params: { all } }),
    diskUsage: (path = "/") =>
      http.get("/system/info/disk/usage", { params: { path } }),
    diskIO: () => http.get("/system/info/disk/io"),
    loadAvg: () => http.get("/system/info/load"),
    loadMisc: () => http.get("/system/info/load/misc"),
    netIO: (pernic = true) =>
      http.get("/system/info/net", { params: { pernic } }),
    netInterfaces: () => http.get("/system/info/net/interfaces"),
    netProtocols: (protocols?: string[]) =>
      http.get("/system/info/net/protocols", {
        params: protocols ? { protocols: protocols.join(",") } : {}
      }),
    netConnections: (kind = "tcp") =>
      http.get("/system/info/net/connections", { params: { kind } }),
    processes: () => http.get("/system/info/process"),
    pids: () => http.get("/system/info/process/pids"),

    // ---- 聚合接口：并行拉取页面所需模块，组装为旧版视图模型 ----
    // 返回值形如 { Data: { Timestamp, Host, OS, CPU, Memory, Disk } }，
    // 使 system-info / home 页面模板与读取代码无需改动（画面不动，仅重新绑定数据）。
    get: async () => {
      const results = await Promise.allSettled([
        http.get("/system/info/host"),
        http.get("/system/info/cpu"),
        http.get("/system/info/cpu/cores", { params: { logical: true } }),
        http.get("/system/info/cpu/cores", { params: { logical: false } }),
        http.get("/system/info/cpu/percent"),
        http.get("/system/info/mem"),
        http.get("/system/info/mem/swap"),
        http.get("/system/info/disk/usage", { params: { path: "/" } })
      ]);
      const [host, cpuInfo, logical, physical, percent, mem, swap, disk] =
        results.map(pickData);

      const cpu0 = Array.isArray(cpuInfo) ? cpuInfo[0] || {} : {};
      const cpuPercent = Array.isArray(percent) ? percent[0] : 0;

      return {
        Data: {
          Timestamp: new Date().toLocaleString("zh-CN", { hour12: false }),
          Host: { Hostname: host?.hostname || "-" },
          OS: {
            Name: host?.os || "-",
            Version: host?.platformVersion || "-",
            Platform: host?.platform || "-",
            Arch: host?.kernelArch || "-",
            Uptime: formatUptime(host?.uptime)
          },
          CPU: {
            ModelName: cpu0.modelName || "-",
            Frequency: cpu0.mhz ?? 0,
            Cores: logical ?? 0,
            PhysicalCores: physical ?? 0,
            UsagePercent: cpuPercent ?? 0
          },
          Memory: {
            Total: mem?.total ?? 0,
            Used: mem?.used ?? 0,
            Available: mem?.available ?? 0,
            UsagePercent: mem?.usedPercent ?? 0,
            SwapTotal: swap?.total ?? 0,
            SwapUsed: swap?.used ?? 0,
            SwapUsagePercent: swap?.usedPercent ?? 0
          },
          Disk: {
            Total: disk?.total ?? 0,
            Used: disk?.used ?? 0,
            Free: disk?.free ?? 0,
            UsagePercent: disk?.usedPercent ?? 0
          }
        }
      };
    }
  }
};

// exportConfig 以当前登录凭证（Bearer token）下载配置 Excel。
// 浏览器导航/新标签（window.open）无法携带 Authorization 头，会被统一鉴权 401，
// 故改用带鉴权的 axios(blob) 下载，自动复用拦截器的 token 注入与 401 刷新重放。
export async function exportConfig(type: string): Promise<void> {
  try {
    const resp: any = await http.get("/config/export", {
      params: { type },
      responseType: "blob"
    });
    const blob: Blob = resp.data;
    // 兼容：后端异常可能以 200 + JSON 返回，按 MIME 兜底识别并提示
    if (blob.type && blob.type.includes("application/json")) {
      const text = await blob.text();
      let msg = "导出失败";
      try {
        msg = JSON.parse(text).Msg || msg;
      } catch {
        /* 保留默认文案 */
      }
      message(msg, { type: "error" });
      return;
    }
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    const stamp = new Date().toISOString().slice(0, 19).replace(/[:T]/g, "");
    a.download = `config_export_${type}_${stamp}.xlsx`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  } catch {
    /* 拦截器已提示错误（含 401 自动刷新重放） */
  }
}

// 导入 Excel 配置
export function importConfig(file: File) {
  const formData = new FormData();
  formData.append("file", file);
  return http.post("/config/import", formData, {
    headers: { "Content-Type": "multipart/form-data" }
  });
}

// ======== 鉴权 API ========
// 返回体经响应拦截后为 { Code, Msg, Data }；login/refresh 的 Data =
// { access_token, refresh_token, token_type, expires_in, access_key }
export const authApi = {
  login: (data: { access_key: string; password: string }) => http.post("/auth/login", data),
  refresh: (data: { refresh_token: string }) => http.post("/auth/refresh-token", data),
  logout: (data?: { refresh_token?: string }) => http.post("/auth/logout", data ?? {})
};

// ======== 登录用户（user_account）管理 API ========
// 位于 /config/user/*，由后端鉴权中间件保护；密码仅新增/改密时提交，列表不回传
export const userApi = {
  query: (params: { page: number; pageSize: number }) => http.post("/config/user/query", params),
  count: (params: { page: number; pageSize: number }) => http.post("/config/user/count", params),
  add: (
    data: {
      access_key: string;
      password: string;
      enable?: number;
      expire_at?: string | null;
      access_token_ttl?: string;
      refresh_token_ttl?: string;
    }
  ) => http.post("/config/user/add", data),
  update: (
    data: {
      access_key: string;
      password?: string;
      enable?: number;
      expire_at?: string | null;
      access_token_ttl?: string;
      refresh_token_ttl?: string;
    }
  ) => http.post("/config/user/update", data),
  del: (ids: number[]) => http.post("/config/user/del", { ids })
};

// ======== AK/SK 接口账号（api_key）管理 API ========
// 位于 /config/apikey/*；secret_key 仅新增/重置时一次性返回
export const apiKeyApi = {
  query: (params: { page: number; pageSize: number }) => http.post("/config/apikey/query", params),
  count: (params: { page: number; pageSize: number }) => http.post("/config/apikey/count", params),
  add: (
    data: {
      access_key: string;
      name?: string;
      enable?: number;
      expire_at?: string | null;
      access_token_ttl?: string;
      refresh_token_ttl?: string;
    }
  ) => http.post("/config/apikey/add", data),
  update: (
    data: {
      access_key: string;
      name?: string;
      enable?: number;
      expire_at?: string | null;
      access_token_ttl?: string;
      refresh_token_ttl?: string;
    }
  ) => http.post("/config/apikey/update", data),
  resetSecret: (data: { access_key: string }) => http.post("/config/apikey/reset-secret", data),
  del: (ids: number[]) => http.post("/config/apikey/del", { ids })
};

// ======== RBAC 权限管理 API ========
// 位于 /config/rbac/*，由后端鉴权 + 权限中间件保护。角色/主体类型：1=业务 2=API。
// 绑定类接口统一用“集合替换”语义：get 返回 { ids:number[] }，set 提交 { ..., ids:number[] }。
export const rbacApi = {
  role: {
    query: (params: { role_type: number; keyword?: string; page?: number; pageSize?: number }) => http.post("/config/rbac/role/query", params),
    count: (params: { role_type: number; keyword?: string }) => http.post("/config/rbac/role/count", params),
    add: (data: {
      role_type: number;
      role_code: string;
      role_name?: string;
      sort?: number;
      status?: number;
      remark?: string;
    }) => http.post("/config/rbac/role/add", data),
    update: (data: {
      role_type: number;
      id: number;
      role_code: string;
      role_name?: string;
      sort?: number;
      status?: number;
      remark?: string;
    }) => http.post("/config/rbac/role/update", data),
    del: (data: { role_type: number; id: number }) => http.post("/config/rbac/role/del", data)
  },
  perm: {
    query: (params?: { keyword?: string; page?: number; pageSize?: number }) => http.post("/config/rbac/perm/query", params ?? {}),
    count: (params?: { keyword?: string }) => http.post("/config/rbac/perm/count", params ?? {}),
    add: (data: {
      perm_code: string;
      topic_key: string;
      action?: string;
      match_sort?: number;
      effect?: number;
      status?: number;
      remark?: string;
    }) => http.post("/config/rbac/perm/add", data),
    update: (data: {
      id: number;
      perm_code: string;
      topic_key: string;
      action?: string;
      match_sort?: number;
      effect?: number;
      status?: number;
      remark?: string;
    }) => http.post("/config/rbac/perm/update", data),
    del: (data: { id: number }) => http.post("/config/rbac/perm/del", data)
  },
  rolePerms: {
    get: (data: { role_type: number; role_id: number }) => http.post("/config/rbac/role/perms", data),
    set: (data: { role_type: number; role_id: number; ids: number[] }) => http.post("/config/rbac/role/perms/set", data)
  },
  subjectRoles: {
    get: (data: { subject_type: number; subject_id: number }) => http.post("/config/rbac/subject/roles", data),
    set: (data: { subject_type: number; subject_id: number; ids: number[] }) => http.post("/config/rbac/subject/roles/set", data)
  },
  subjectPerms: {
    get: (data: { subject_type: number; subject_id: number }) => http.post("/config/rbac/subject/perms", data),
    set: (data: { subject_type: number; subject_id: number; ids: number[] }) => http.post("/config/rbac/subject/perms/set", data)
  }
};
