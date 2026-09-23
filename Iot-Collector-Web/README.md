# Iot-Collector-Web

Iot-Collector-Service 的前端配置管理界面，用于管理采集器的驱动、点位、报警和历史记录配置。

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Vue | 3.5 | 前端框架 |
| TypeScript | 5.6 | 类型安全 |
| Vite | 6.0 | 构建工具 |
| Element Plus | 2.9 | UI 组件库 |
| vue-router | 4.5 | 路由（Hash 模式） |
| vue-i18n | 10.0 | 国际化 |
| ECharts | 6.1 | 趋势曲线图 |
| Axios | 1.7 | HTTP 请求 |

## 构建与运行

```bash
npm install        # 安装依赖
npm run dev        # 开发模式（http://0.0.0.0:8113）
npm run build      # 生产构建（vue-tsc + vite build）
npm run preview    # 预览生产构建
```

## 项目结构

```
src/
├── api/
│   └── config.ts          # HTTP 客户端 + 全部 API 封装
├── assets/
│   └── main.css           # 全局样式
├── i18n/
│   ├── index.ts           # i18n 入口（默认 zh）
│   ├── zh.ts              # 中文
│   ├── en.ts              # English
│   ├── fr.ts              # Français
│   ├── ru.ts              # Русский
│   ├── es.ts              # Español
│   └── ar.ts              # العربية
├── router/
│   └── index.ts           # 路由配置（4 个页面）
├── views/
│   ├── DriveConfig.vue    # 驱动配置页
│   ├── PointConfig.vue    # 点位配置页
│   ├── AlarmConfig.vue    # 报警配置页
│   └── HistoryConfig.vue  # 历史配置页
├── App.vue                # 布局（侧边栏 + 顶部栏 + 主内容）
└── main.ts                # 入口（挂载 ElementPlus / Router / i18n）
```

## 页面功能说明

### 1. 驱动配置（/drive）— DriveConfig.vue

管理 IO 通信驱动，每个驱动代表一个与物理设备的连接通道。

**表格列：**

| 列 | 字段 | 说明 |
|----|------|------|
| ID | `Id` | 驱动自增 id |
| 驱动类型 | `Type` | `Modbus_Tcp` 或 `Siemens_S7` |
| 驱动名称 | `Name` | 用户自定义名称 |
| 配置 | `Config` | 分号分隔的连接参数字符串 |
| 点位数 | `Points_Length` | 该驱动下的点位数量 |
| 创建时间 | `Creation_Time` | - |

**操作：**

| 操作 | 说明 |
|------|------|
| 新增 | 选择驱动类型 → 填写名称和配置参数 → 创建 |
| 编辑 | 修改名称和配置参数（驱动类型不可改） |
| 重启 | 向采集器发送重启指令，重新初始化该驱动连接 |
| 删除 | 确认后删除驱动及其关联数据 |
| 导出 | 导出全部驱动配置为 Excel 文件 |

**配置格式提示（随驱动类型动态切换）：**

| 驱动类型 | 配置格式 | 示例 |
|----------|----------|------|
| Modbus TCP | `IP;重试间隔;连接超时;响应超时;轮询间隔;组包最大长度` | `192.168.1.1;3s;3s;180s;1s;256` |
| Siemens S7 | `IP;机架号;槽位号;重试间隔;轮询间隔;连接超时;响应超时;组包大小;连接类型` | `192.168.1.1;0;2;3s;0;10s;10s;480;2` |

> 配置输入框旁有 `?` 链接，跳转到对应驱动的 GitHub README 文档。

---

### 2. 点位配置（/point）— PointConfig.vue

管理每个驱动下的数据采集点位，支持实时值监控、趋势曲线和写入操作。

**表格列：**

| 列 | 字段 | 说明 |
|----|------|------|
| ID | `Id` | 点位自增 id |
| 驱动 ID | `Drive_Id` | 所属驱动 |
| 点位名称 | `Name` | 用户自定义名称 |
| 描述 | `Description` | 可选说明 |
| 配置 | `Config` | 分号分隔的点位采集参数（趋势模式时隐藏） |
| 读写方式 | `RW_Cancel` | N/R/W/R/W（趋势模式时隐藏） |
| 值类型 | `Value_Type` | bool/int16/float32 等（趋势模式时隐藏） |
| 创建时间 | `Creation_Time` | - |
| **状态** | 实时 | WebSocket 推送，`ok` 绿色 / 错误红色 |
| **当前值** | 实时 | WebSocket 推送的最新采集值 |
| **时间延迟** | 实时 | 采集时间到当前的延迟（绿/黄/红三色） |
| **趋势** | 实时 | ECharts Sparkline 小曲线（需手动开启） |

**操作：**

| 操作 | 说明 |
|------|------|
| 显示趋势 | 切换趋势曲线列的显示/隐藏 |
| 新增 | 选择驱动 → 填写名称、配置、读写方式、值类型 → 创建 |
| 编辑 | 修改点位属性 |
| 写入 | 仅 `RW_Cancel` 为 W(3) 或 R/W(4) 时可用，弹出写入对话框 |
| 删除 | 确认后删除点位 |
| 导出 | 导出全部点位配置为 Excel 文件 |

**读写方式（RW_Cancel）：**

| 值 | 显示 | 说明 |
|----|------|------|
| 1 | N | 不采集 |
| 2 | R | 只读 |
| 3 | W | 只写 |
| 4 | R/W | 读写 |

**值类型（Value_Type）：**

| 值 | 类型 | 值 | 类型 |
|----|------|----|------|
| 1 | bool | 9 | uint64 |
| 2 | int8 | 10 | int |
| 3 | uint8 | 11 | uint |
| 4 | int16 | 12 | float32 |
| 5 | uint16 | 13 | float64 |
| 6 | int32 | 14 | float |
| 7 | uint32 | 15 | string |
| 8 | int64 | | |

**实时数据获取（双通道）：**

```
页面加载
    │
    ├── 1. HTTP API 获取初始值
    │   └── POST /point/read/value → 读取内存中的当前值
    │
    └── 2. WebSocket 持续推送实时值
        └── ws://HOST:PORT/api/v1.0/point/ws?points=[id1,id2,...]
            ├── 推送内容：PointId / Value / Msg / Time
            ├── 时间延迟计算：Date.now() - new Date(Time).getTime()
            │   ├── < 1s → 绿色（ms 级）
            │   ├── < 3s → 黄色（s 级）
            │   ├── < 60s → 红色（s 级）
            │   └── ≥ 60s → 红色（m 分 s 秒）
            └── 趋势曲线：每次数据追加到 ECharts Sparkline
```

**点位配置格式提示（随所属驱动类型动态切换）：**

| 驱动类型 | 配置格式 | 示例 |
|----------|----------|------|
| Modbus TCP | `从机地址;功能码;寄存器地址;字节序;数据类型` | `1;3;100;ABCD;float32` |
| Siemens S7 | `存储区;DB块号;字节偏移;数据类型;子地址` | `132;1;0;12;0` |

---

### 3. 报警配置（/alarm）— AlarmConfig.vue

为点位配置报警规则，当采集值满足条件时触发报警。

**表格列：**

| 列 | 字段 | 说明 |
|----|------|------|
| ID | `Id` | 报警配置自增 id |
| 点位 ID | `Point_Id` | 关联的点位 |
| 报警名称 | `Name` | 用户自定义名称 |
| 报警规则 | `Config` | 报警条件表达式 |
| 报警组 | `Group` | 报警分组编号（≥1） |
| 创建时间 | `Creation_Time` | - |

**操作：**

| 操作 | 说明 |
|------|------|
| 新增 | 选择点位（可搜索下拉）→ 填写名称、规则、组号 → 创建 |
| 编辑 | 修改报警规则属性 |
| 删除 | 确认后删除 |
| 导出 | 导出全部报警配置为 Excel 文件 |

**报警规则（Config）支持的值：**

| 规则 | 含义 | 触发条件 |
|------|------|----------|
| `==0` | 值等于 false/0 | 值为 false → 报警开始；值为 true → 报警恢复 |
| `==1` | 值等于 true/非0 | 值为 true → 报警开始；值为 false → 报警恢复 |
| `0<>1` | 值变化触发 | 任何值变化都触发报警 |

> 一个点位可以配置多条报警规则，每条规则独立判断。

---

### 4. 历史配置（/history）— HistoryConfig.vue

为点位启用历史数据记录，配置生效后该点位的每次采集值都会被记录到历史存储。

**表格列：**

| 列 | 字段 | 说明 |
|----|------|------|
| ID | `Id` | 历史配置自增 id |
| 点位 ID | `Point_Id` | 关联的点位 |
| 配置 | `Config` | 历史采集配置（非空即启用） |
| 创建时间 | `Creation_Time` | - |

**操作：**

| 操作 | 说明 |
|------|------|
| 新增 | 选择点位（可搜索下拉）→ 填写配置 → 创建 |
| 编辑 | 修改历史配置 |
| 删除 | 确认后删除 |
| 导出 | 导出全部历史配置为 Excel 文件 |

> 配置值不能为空或 `"null"`，否则该点位不会被记录历史数据。

---

## API 接口

后端基础地址：`http://192.168.220.40:8103/api/v1.0`

### 响应格式

所有接口返回统一格式：

```json
{
  "Code": 200,
  "Msg": "ok",
  "Data": [...],
  "Timestamp": "2026-09-23T10:00:00Z"
}
```

前端根据 `Code` 范围显示不同消息类型：
- `2xx` → 成功（绿色）
- `3xx` → 信息（蓝色）
- `4xx` → 警告（黄色）
- `5xx` → 错误（红色）

### 接口列表

| 模块 | 方法 | 路径 | 参数 |
|------|------|------|------|
| **驱动** | POST | `/config/drive/query` | `{ page, pageSize }` |
| | POST | `/config/drive/count` | `{ page, pageSize }` |
| | POST | `/config/drive/add` | `[{ Type, Name, Config }]` |
| | POST | `/config/drive/update` | `[{ Id, Name, Config }]` |
| | POST | `/config/drive/del` | `{ ids: [] }` |
| | POST | `/config/drive/restart` | `{ id }` |
| **点位** | POST | `/config/point/query` | `{ driveId?, page, pageSize }` |
| | POST | `/config/point/count` | `{ driveId?, page, pageSize }` |
| | POST | `/config/point/add` | `[{ Drive_Id, Name, Description, Config, RW_Cancel, Value_Type }]` |
| | POST | `/config/point/update` | `[{ Id, ... }]` |
| | POST | `/config/point/del` | `{ ids: [] }` |
| | POST | `/point/read/value` | `[pointId1, pointId2, ...]` |
| | POST | `/point/write/value` | `[{ PointId, Value, Type }]` |
| **报警** | POST | `/config/alarm/query` | `{ pointId?, page, pageSize }` |
| | POST | `/config/alarm/count` | `{ pointId?, page, pageSize }` |
| | POST | `/config/alarm/add` | `[{ Point_Id, Name, Config, Group }]` |
| | POST | `/config/alarm/update` | `[{ Id, ... }]` |
| | POST | `/config/alarm/del` | `{ ids: [] }` |
| **历史** | POST | `/config/history/query` | `{ pointId?, page, pageSize }` |
| | POST | `/config/history/count` | `{ pointId?, page, pageSize }` |
| | POST | `/config/history/add` | `[{ Point_Id, Config }]` |
| | POST | `/config/history/update` | `[{ Id, ... }]` |
| | POST | `/config/history/del` | `{ ids: [] }` |
| **导出** | GET | `/config/export?type=drive\|point\|alarm\|history` | - |
| **WebSocket** | WS | `/point/ws?points=[id1,id2,...]` | 实时推送点位值 |

## 布局与交互

### 整体布局

```
┌──────────────────────────────────────────────┐
│  ⚙ 采集配置           语言切换  Iot-Collector │  ← 顶部 Header
├────────┬─────────────────────────────────────┤
│        │  首页 / 当前页面                      │  ← 面包屑
│  驱动  │─────────────────────────────────────│
│  点位  │                                     │
│  报警  │  页面标题          [新增] [导出]      │
│  历史  │  ┌─────────────────────────────┐    │
│        │  │ 表格数据                      │    │
│  [≡]   │  └─────────────────────────────┘    │
│        │  共 X 条  < 1 2 3 ... >             │  ← 分页
└────────┴─────────────────────────────────────┘
```

### 响应式适配

| 屏幕宽度 | 行为 |
|----------|------|
| ≥ 768px | 侧边栏常驻，支持折叠/展开 |
| < 768px | 侧边栏隐藏，点击汉堡菜单弹出（带遮罩层） |

### 国际化支持

6 种语言实时切换（中文 / English / Français / Русский / Español / العربية），同时切换 Element Plus 组件库语言包。

## 四页面配置关系

```
驱动配置（DriveConfig）
    │ 一个驱动包含多个点位
    ▼
点位配置（PointConfig）
    │ 一个点位可关联多条报警 + 一条历史
    ├──▶ 报警配置（AlarmConfig）  ——  按 Point_Id 关联
    └──▶ 历史配置（HistoryConfig） ——  按 Point_Id 关联
```

## 依赖

- `Iot-Collector-Service` — 后端 API 服务（端口 8103）
