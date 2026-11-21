# Iot-Collector-Web

工业物联网边缘采集服务的**前端管理台**：基于 `vue-pure-admin`（Vue 3 + Vite + Element Plus + TypeScript）精简改造，对接后端 [`Iot-Collector-Service`](../Iot-Collector-Service/README.md) 的 HTTP/WebSocket 接口，提供驱动、点位、报警、历史、权限（RBAC）、账号 / APIKey、系统信息与系统配置等管理页面。

> 🌐 **在线 Demo**：<http://collector.demo.zwf.fan/> —— **前后端一体化部署**的完整系统演示实例，可直接打开体验驱动/点位/报警/历史/RBAC 等全部功能；数据为演示环境，请勿用于生产。

> 本文件是**前端总览入口**。每个模块都有独立 README，请从下方 [🧭 模块导航](#-模块导航) 点进去。

## 📋 目录

- [技术栈](#-技术栈)
- [🧭 模块导航](#-模块导航)
- [项目结构](#-项目结构)
- [快速开始](#-快速开始)
- [运行时服务器地址配置](#-运行时服务器地址配置)
- [与后端的约定](#-与后端的约定)
- [许可证](#-许可证)

## 🔧 技术栈

| 类别 | 技术 | 说明 |
|------|------|------|
| 框架 | Vue 3（组合式 API + `<script setup>`） | 主框架 |
| 构建 | Vite | 开发/打包；`vite.config.ts` |
| 语言 | TypeScript（严格模式） | `tsconfig.json`，类型检查 `vue-tsc` |
| UI | Element Plus + vxe-table | 组件库与高性能表格 |
| 状态 | Pinia | `src/store` |
| 路由 | vue-router（hash 模式） | `src/router`，`VITE_ROUTER_HISTORY="hash"` |
| 国际化 | vue-i18n | `src/plugins/i18n.ts`，多语言见 `public/locales` |
| HTTP | axios（全局单例） | 封装在 `src/api/config.ts` |
| 图表 | echarts | `src/plugins/echarts.ts` |
| 包管理 | **pnpm**（`only-allow pnpm` 钩子） | `package.json` name=`iot-collector-web`，version=`7.0.0` |

## 🧭 模块导航

点击"详细文档"进入各模块 README。

### 应用入口与配置

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `src/main.ts` / `src/App.vue` | 应用引导：`getPlatformConfig` 拉取平台配置、挂载 router/store/i18n | [src 总览](src/README.md) |
| `src/config` | 平台配置 `getConfig()` 与默认项 | [README](src/config/README.md) |

### 数据与网络

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `src/api` | 全局 `http` 实例 + 拦截器 + 各业务 API 分组 + 服务器地址解析 | [README](src/api/README.md) |
| `src/store` | Pinia：app / user / permission / multiTags / settings / epTheme | [README](src/store/README.md) |

### 路由与布局

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `src/router` | 路由实例、静态/动态路由、`modules`（home/point/remaining/system） | [README](src/router/README.md) |
| `src/layout` | 主布局：navbar / sidebar / tag / hooks | [README](src/layout/README.md) |

### 视图页面

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `src/views` | 所有业务页面索引 | [README](src/views/README.md) |
| ├ `views/login` | 登录（AK/密码 + 服务器地址） | [README](src/views/login/README.md) |
| ├ `views/home` | 首页概览 | [README](src/views/home/README.md) |
| ├ `views/drive` | 驱动配置管理 | [README](src/views/drive/README.md) |
| ├ `views/ioDrive` | Modbus / Siemens 驱动表单（index+form+point_form） | [README](src/views/ioDrive/README.md) |
| ├ `views/point` | 点位配置管理 | [README](src/views/point/README.md) |
| ├ `views/alarm` | 报警规则配置 | [README](src/views/alarm/README.md) |
| ├ `views/alarm-data` | 报警数据查询 | [README](src/views/alarm-data/README.md) |
| ├ `views/history` | 历史采集配置 | [README](src/views/history/README.md) |
| ├ `views/history-data` | 历史数据查询（InfluxDB） | [README](src/views/history-data/README.md) |
| ├ `views/rbac` | RBAC：角色 / 权限 / 授权 | [README](src/views/rbac/README.md) |
| ├ `views/user` | 登录账号管理 | [README](src/views/user/README.md) |
| ├ `views/apikey` | APIKey（AK/SK）管理 | [README](src/views/apikey/README.md) |
| ├ `views/system-info` | 系统信息监控 | [README](src/views/system-info/README.md) |
| ├ `views/system-config` | 在线编辑 config.yaml | [README](src/views/system-config/README.md) |
| └ `views/error` / `empty` / `account` | 错误页 / 空白页 / 账号页 | [README](src/views/error/README.md) |

### 公共能力

| 模块 | 职责 | 详细文档 |
|------|------|----------|
| `src/components` | 通用组件（ReXxx 系列，含自研 `RePointSelector` 等） | [README](src/components/README.md) |
| `src/utils` | 工具：`auth`(令牌) / `serverAddr` / `configFormat` / `http`(空) 等 | [README](src/utils/README.md) |
| `src/directives` | 指令：`auth` / `perms` / `copy` / `ripple` / `longpress` 等 | [README](src/directives/README.md) |
| `src/plugins` | 插件装配：elementPlus / echarts / i18n / vxeTable | [README](src/plugins/README.md) |
| `src/style` | 全局样式与主题 | [README](src/style/README.md) |
| `src/assets` | 静态资源 | [README](src/assets/README.md) |

### 后端联动（跨项目）

| 关注点 | 后端文档 |
|------|----------|
| 接口 / 鉴权 / 响应信封 `{Code,Msg,Data}` | [handler README](../Iot-Collector-Service/internal/handler/README.md) |
| RBAC 主题匹配规则（前端 topic_key 需对齐） | [pkg/rbac README](../Iot-Collector-Service/pkg/rbac/README.md) |
| 字段格式正则的前端镜像 `configFormat.ts` ↔ 后端 | [conv README](../Iot-Collector-Service/pkg/conv/README.md) |

## 📁 项目结构

```
Iot-Collector-Web/
├── src/
│   ├── api/          # 全局 http + 拦截器 + 业务 API（config.ts 为核心）
│   ├── assets/       # 静态资源
│   ├── components/   # 通用组件 ReXxx（27 个）
│   ├── config/       # 平台配置 getConfig
│   ├── directives/   # 自定义指令（auth/perms/copy/...）
│   ├── layout/       # 主布局（navbar/sidebar/tag/hooks）
│   ├── plugins/      # elementPlus/echarts/i18n/vxeTable 装配
│   ├── router/       # 路由 + modules（home/point/remaining/system）
│   ├── store/        # Pinia（app/user/permission/multiTags/settings/epTheme）
│   ├── style/        # 全局样式
│   ├── utils/        # auth/serverAddr/configFormat/... （http 目录为空）
│   ├── views/        # 业务页面（17 个子目录）
│   ├── App.vue       # 根组件
│   └── main.ts       # 入口
├── public/
│   └── platform-config.json  # 运行时可改配置（BaseServer 等）
├── types/            # 全局 .d.ts
├── build/            # Vite 构建脚本
├── .env / .env.development / .env.production
├── vite.config.ts / tsconfig.json / package.json
└── Dockerfile
```

## 🚀 快速开始

```bash
# 安装依赖（必须用 pnpm，only-allow 钩子会拦截 npm install）
pnpm install

# 开发（VITE_PORT=8113，后端默认 8103）
npm run dev          # 或 pnpm dev

# 类型检查
npm run typecheck    # vue-tsc --noEmit --skipLibCheck

# 生产构建（先 rimraf dist → vite build → generate-version-file 产出 dist/version）
npm run build

# 预览构建产物
npm run preview
```

> `only-allow pnpm` 钩子只拦 `npm install`，不拦 `npm run <script>`，故 `npm run dev/build/typecheck` 均可直接执行；装依赖须用 `pnpm install`。
> 构建内存不足时脚本已内置 `NODE_OPTIONS=--max-old-space-size=8192`。
> `vite build` 打印 `[INEFFECTIVE_DYNAMIC_IMPORT]`（user.ts 被动态+静态同时引入）是可忽略的 rollup 提示。

## 🔌 运行时服务器地址配置

前端连接后端的地址**构建后可改**，优先级（`src/api/config.ts` 的 `getBaseServer`）：

1. 登录页手输并持久化到 localStorage 的当前地址
2. `public/platform-config.json` 的 `BaseServer`（`"IP:端口"`，**默认应留空**，勿写死内网 IP）
3. 构建期注入的 `VITE_BASE_SERVER`（dev 在 `.env.development`，prod 留空）
4. `window.location.host` 同源兜底

`resolveBaseURL()`：host 与页面同源时返回相对 `/api/v1.0`（继承协议/端口，避免混合内容），异机时拼 `http://{host}/api/v1.0`。修改 `platform-config.json` 无需重新构建，刷新浏览器即生效。详见 [api 详细文档](src/api/README.md)。

## 🤝 与后端的约定

- **响应信封**：后端统一返回 `{Code,Msg,Data,Timestamp}`，`Code` 2xx 成功；全局响应拦截器已按 code 弹消息并 `reject` 非 2xx，业务层直接取 `.Data`。
- **鉴权**：Bearer `access_token` 自动注入，401 单飞刷新 + 重放一次，刷新失败登出（`src/api/config.ts`）。
- **字段格式校验**：`src/utils/configFormat.ts` 是后端 `Point/Drive_Config` 校验正则的前端镜像，改后端正则必须同步这里，否则"前端能提交、后端回 417"。
- **RBAC topic_key**：权限规则的主题式匹配串需与后端 [pkg/rbac](../Iot-Collector-Service/pkg/rbac/README.md) 的分段/通配语义一致。

## 📄 许可证

本项目基于 vue-pure-admin（[MIT](./LICENSE)）改造。上游版权：© 2020-present, pure-admin。
