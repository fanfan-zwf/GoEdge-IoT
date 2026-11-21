# `internal/repo/mysql` — MySQL 持久层：连接、表结构自校验、全部业务 SQL

上级文档：[服务总览 README](../../../README.md)

本包是服务唯一的配置/账号/日志存储入口，向上暴露两类能力：

1. **表结构自维护**（`schema.go`）：启动时对比"程序预期规则"与真实库，缺表自动建、缺列自动加、AUTO_INCREMENT 自动修，其余不一致直接 `panic`。
2. **业务 DAO**（`business*.go` / `rbac*.go`）：`Xxx__Query / Count / Search / Add / Update / Del / Sync` 七件套 + 账号、日志、标签树、RBAC 四组专用函数。

> 依赖方向：`handler` / `service/*` → `repo/mysql`。本包**不 import** 任何其他 `internal/*` 包，只依赖标准库与 `go-sql-driver/mysql`。

---

## 一、文件清单

| 文件 | 行数 | 职责 |
|---|---|---|
| `mysql.go` | 65 | `var DB *sql.DB`、`Init_sql(dsn)`、包级 `init()`（连不上即 panic）、`Close()`、`Example()` |
| `schema.go` | 693 | `ColumnRule`/`TableRule`、`expectTableRules`（10 张表）、`CheckAndFixTableStructure`、`CheckSqlStructure`、建表/加列/改自增 SQL 拼装 |
| `business.go` | 2830 | 采集侧四张配置表（Drive/Point/Alarm/History）+ `Alarm_Data` + `APP_Config` 的 CRUD/Sync/Count/Search；三条校验正则；`logErrorThrottled`；**尾部保留旧 `Account__*` 函数族** |
| `business_account.go` | 536 | 现役账号 DAO：`User__*`（`user_account`）与 `ApiKey__*`（`api_key`）两套，含 `ApiKey__Sync`、`ApiKey__AccessKeysByIds`、TTL 归一化 |
| `business_log.go` | 262 | 单表 `log_operate` 的读写与多维过滤：`Log_Source.Operator()` 生成操作人、`Log_Type`/`Log_Operate` 文本枚举、`Log_Filter` |
| `business_tagtree.go` | 240 | `tag_tree` 的重建与懒加载查询：`Tag_Tree__RebuildFromPointConfig`、`Query_Children_ByPath`、`Query_PointsUnderPath` |
| `rbac.go` | 247 | RBAC 规则装载：`LoadEffectiveRules`、`LoadRoleCodes`，主体常量 `Rbac_Subject_Biz=1` / `Rbac_Subject_Api=2` |
| `rbac_admin.go` | 394 | RBAC 后台管理 DAO：`RbacRole__*`、`RbacPerm__*`、`RolePerm__*`、`SubjectRole__*`、`SubjectPerm__*` 的 List/Replace |

---

## 二、⚠️ 最容易踩的坑

1. **`init()` 里连不上 MySQL 直接 `panic`，不是返回错误。**
   `mysql.go` 的包级 `init()` 依次执行 `Init_sql(config.GlobalConfig.SQL.Link)` → 失败 `panic(err)` → 成功后 `CheckSqlStructure()`。也就是说：**任何 import 了本包的二进制（包括 `go test`、临时脚本）在启动瞬间就要求 MySQL 可用**。想跑不依赖 DB 的单测必须用 build tag 或把逻辑抽到不 import 本包的层。

2. **表结构"部分不一致"会被静默修复，"另一部分不一致"会炸进程。**
   自动修复：缺表 → `CREATE TABLE`；缺列 → `ALTER TABLE ADD COLUMN`；`AUTO_INCREMENT` 属性不符 → `ALTER TABLE … MODIFY COLUMN`。
   直接 `panic`：列类型不同、非空约束不同、主键不同、默认值不同（且不满足豁免条件）、规则本身非法（自增列不是主键、自增列类型非整型等）。
   **手工作坊里"手动改过一列类型"是启动失败最常见的原因**，改前先确认与 `expectTableRules` 一致。

3. **RBAC 的 8 张表不在 `schema.go` 自维护列表里。**
   `t_biz_role`、`t_api_role`、`t_permission`、`t_role_permission`、`t_biz_user_role`、`t_api_client_role`、`t_direct_permission` 只存在于根目录的备份 SQL（`iot-collector-server_backup_20261120_235444.sql`）。
   后果：**在全新空库上启动，配置表会自动建好，但 RBAC 表不会**，`LoadEffectiveRules` 查询会报表不存在，而 RBAC 是 deny-by-default + fail-closed，于是所有业务接口返回 `520`。新库必须先导入备份 SQL 或手工建表。

4. **`idScope ...[]uint` 是可变形参，"不传"和"传空切片"含义相同且不等于"无权限"。**
   `applyIdScope` 里 `len(idScope)==0` 恒等价于"不加过滤 = 全量"。"授权集为空"这种真·无权场景由中间件提前判 `403`，**根本不会走到 DAO**。所以给 DAO 传 `[]uint{}` 千万别指望它返回空结果集——那会拖全表。调用方必须自己保证：要么有完整权限、要么有非空 id 集。

5. **`Xxx__Sync` 无事务、且 `Drive_Config__Sync` 的新增分支不写 `Name`。**
   `Drive_Config__Sync` 构造的是 `Drive_Config_Add_type{Id, Type, Config}`，**`Name` 字段被漏掉**；通过云端同步/导入新建的驱动名称会是空串（`Name` 列 `NOT NULL` 且无默认值，靠 Go 侧零值 `""` 落库）。UI 手工新增走 `Drive_Config__Add` 不受影响。改造建议：同步分支补 `Name: cfg.Name`。
   另外所有 `Sync` 都是"读全表 → 建 map → 分增/改/删 → 逐条 Exec"，中途失败会留下半同步状态。

6. **分页参数 `page=0` / `pageSize=0` 是"不分页"语义，不是第一页。**
   `Sync` 内部正是用 `Drive_Config__Query(0, 0)` 取全量的。上层若把「第 0 页」直接透传，会得到全表数据。

7. **`Regex_Config_Forbidden` 会拒绝含 SQL 关键字的正常配置。**
   配置值一律走 `?` 占位符，注入本就不可达，这条正则防的是"把 SQL 片段当配置存进来"的误操作。副作用：点位描述、驱动名称里出现 `create`/`select`/`test`（含 `sel`？不含，是整词边界 `\b`）等**英文整词**会被拒。文案里请避免这些单词，或改走 `Config__Validate_Basic` 之外的通道。
   它**不拦双引号和分号**——驱动配置是 JSON 必须带双引号，Modbus 点位配置是 `;`/`:` 分隔串。

8. **`Account__*` 是遗留函数族，与 `User__*` / `ApiKey__*` 并存。**
   `business.go` 尾部保留 `Account_type` / `Account_Add_type` / `Account__Query_By_Access_Key` / `Add` / `Update_Mutable` / `Sync` / `Query` / `Count` / `Del` + `normalizeTTL`。**新代码一律用 `business_account.go` 里的 `User__*`（网页账号）或 `ApiKey__*`（接口账号）**，两者按 `subjectType` 分流；`Account__*` 只操作 `api_key` 的旧字段视图，属待清理项，不要新增调用。

9. **同类错误日志 30s 内只打一条。**
   `logErrorThrottled(key, msg)` 按 `key` 分窗口（`Log_Error_Window = 30s`），窗口内后续错误只计数，下一条命中窗口的日志会追加"（上一窗口内同类错误已抑制 N 条）"。排障时**看不到报错不等于没报错**，注意数抑制提示。

---

## 三、连接与初始化

```go
// mysql.go
var DB *sql.DB

func Init_sql(dsn string) error {
    db, err := sql.Open("mysql", dsn)
    ...
    db.SetMaxOpenConns(20)      // 最大并发连接
    db.SetMaxIdleConns(10)      // 空闲连接池
    db.SetConnMaxLifetime(time.Hour) // 连接最长存活，防 MySQL wait_timeout 侧静默断开
    db.SetConnMaxIdleTime(10 * time.Minute)
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return db.PingContext(ctx)  // 5s 探活，配置错误立刻暴露
}

func init() {
    if err := Init_sql(config.GlobalConfig.SQL.Link); err != nil { panic(err) }
    CheckSqlStructure()
}
```

> 注意：`init()` 引用了 `internal/config`，因此本包**并非严格零依赖**——它只依赖配置，不依赖任何其他业务层。`Close()` 是 nil-safe 的，可安全重复调用。

DSN 来自 `configs/config.yaml` 的 `sql.link`，详见 [internal/config README](../../config/README.md)。

---

## 四、表结构规则（`expectTableRules`）

### 4.1 `ColumnRule` 字段语义

| 字段 | 含义 | 校验行为 |
|---|---|---|
| `ColumnName` | 列名 | 缺失 → 自动 `ADD COLUMN`；禁止用它新增自增列/主键列 |
| `ColumnType` | 如 `int unsigned`、`varchar(300)`、`json` | 不符 → **panic**（大小写/空白归一后比较） |
| `IsAutoInc` | 程序预期是否自增 | 不符 → 自动 `ALTER TABLE … MODIFY COLUMN` 修正 |
| `IsRequired` | 是否 `NOT NULL` | 不符 → **panic** |
| `IsPrimaryKey` | 是否主键（自增列必须为 `true`） | 不符 → **panic**；规则非法 → **panic** |
| `IsIndex` / `IsUnique` | 普通索引 / 唯一索引 | 用于建表，已存在表上不强制回改 |
| `DefaultValue` | `""`=无默认值；纯数字按字面量；其他字符串自动加单引号；`CURRENT_TIMESTAMP` 特殊处理 | 不符 → **panic**，但有豁免（见下） |

**默认值豁免规则**（避免"MySQL 8 可空列默认 NULL"这类无害差异炸启动）：
- 可空列：实际默认值为 `NULL` 视为匹配规则里的 `""`；
- 非空列：实际默认值 `""` 与 `"NULL"` 均视为匹配；
- 自增列：恒匹配，不比默认值。

表注释（`TableComment`）的校验代码已被整段注释掉，改表注释不会触发不一致。

### 4.2 10 张自维护表

| 表 | 用途 | 关键约束 |
|---|---|---|
| `APP_Config` | 键值型运行时配置 | `Ket` 索引；**注意列名就是 `Ket`（拼写沿用 Key 的笔误，已固化）** |
| `Drive_Config` | 驱动实例 | `Config` 列为 **`json` 类型**；`Type` 索引；`Points_Length` 默认 `0` |
| `Point_Config` | 点位 | `Tag varchar(300)` **UNIQUE**；`Drive_Id` 索引；`Conversion` 默认 `''` |
| `tag_tree` | Tag 路径派生的懒加载树 | `Path varchar(700)` **UNIQUE**（保证重建幂等）；`Node_Type` 1文件夹/2点位/3both；`Point_Id` NULL=纯文件夹 |
| `Alarm_Config` | 报警规则 | `Tag varchar(300)` **UNIQUE**（这里的 Tag 是规则名，不是点位标识） |
| `Alarm_Data` | 报警触发记录 | 仅 `Alarm_Id`/`Status`/`Time`，**无 idScope 过滤** |
| `History_Config` | 历史落库周期配置 | `Point_Id` 索引 |
| `user_account` | 网页登录账号（Bearer 通路） | `Access_Key` UNIQUE；`Login_Pwd_Hash` bcrypt；TTL 列是 `varchar(32)` 存 `time.Duration` 字符串（`15m0s` / `168h0m0s`） |
| `api_key` | AK/SK 接口账号（HMAC 通路） | `Access_Key` UNIQUE；`Secret_Key varchar(128)` 仅新增/重置时一次性返回 |
| `log_operate` | 全通路共用的一张操作日志 | `Type`/`Operate` 为 **`varchar(20)` 文本枚举**（存 `drive`/`write` 这种可读值，不用数字）；`Operator varchar(64)`（`user:1`/`aksk:1`/`internalmqtt`）索引；`Content text`；`Ip varchar(45)` 允许空串（MQTT 通路无来源 IP） |

> 全库混用两种命名风格：配置类是 `PascalCase`（`Drive_Config`），账号/日志/标签树是 `snake_case`（`user_account`、`log_operate`、`tag_tree`）。SQL 里统一用反引号包裹，改表名时注意 `business_tagtree.go` 等处直接写死的字符串。

---

## 五、DAO 函数族

### 5.1 采集配置四表（结构完全对称）

以 `Drive_Config` 为例，`Point` / `Alarm` / `History` 同构，只是多了个前置过滤数组：

```go
func Drive_Config__Query(page, pageSize uint, idScope ...[]uint) ([]Drive_Config_Query_type, error)
func Drive_Config__Query_Callback(page, pageSize uint, idScope []uint, callback func(Drive_Config_Query_type)) error
func Drive_Config__Count(page, pageSize uint, idScope ...[]uint) (uint, error)
func Drive_Config__Search(keyword string, page, pageSize uint, idScope ...[]uint) (...)
func Drive_Config__Add(configs ...Drive_Config_Add_type) error
func Drive_Config__Update(configs ...Drive_Config_Update_type) error
func Drive_Config__Del(ids ...uint) error
func Drive_Config__Sync(configs []Drive_Config_Query_type) error
func Drive_Config__AllIds() ([]uint, error)   // 走通用 selectAllIds(table)

// Point/Alarm/History 的版本多一个前置 id 列表（按所属驱动/点位筛）：
func Point_Config__Query(driveid []uint, page, pageSize uint, idScope ...[]uint) (...)
func Alarm_Config__Query(pointid []uint, page, pageSize uint, idScope ...[]uint) (...)
func History_Config__Query(pointid []uint, page, pageSize uint, idScope ...[]uint) (...)
```

Point 专用补充：

| 函数 | 用途 |
|---|---|
| `Point_Config__Query__DriveIds(driveIds []uint, ...)` | 按驱动批量取点位（驱动重启/删除联动用） |
| `Point_Config__Query__RW(...)` | 只取可写点位 |
| `Point_Config__Query__IdTag_ByTags(tags []string)` | Tag → id 反查（写值/订阅入口） |
| `Point_Config__Query__IdTag_ByIds(ids []uint)` | id → Tag 正查（返回值补 Tag） |
| `Point_Config__Query__Tag(id uint)` | 单点 id → Tag（`resolveTag` 的底层） |

Alarm/History 侧的 `Alarm_Data`：

```go
func Alarm_Data__Query(alarmIds []uint, page, pageSize uint) ([]Alarm_Data_type, error)
// ⚠️ 无 idScope 形参：报警记录不做行级权限过滤，只能按 alarmIds 收窄
```

`__Query_Callback` 是流式版本：`handler/export.go` 导出、`service/collector` 启动全量加载走它，避免一次性把几万点位堆在切片里。

### 5.2 账号（现役，`business_account.go`）

```go
User__Query_By_Access_Key(ak string) / User__Add / User__Update_Mutable / User__Query / User__Count / User__Del
ApiKey__Query_By_Access_Key / ApiKey__Add / ApiKey__Update_Mutable / ApiKey__Query / ApiKey__Count / ApiKey__Del
ApiKey__Sync(...)                       // 供云端配置同步
ApiKey__AccessKeysByIds(ids []uint)     // RBAC 列表回显主体名
```

`Update_Mutable` 的语义是"只改可变字段"：`Access_Key`、`Secret_Key`、密码哈希不在此路径，改 SK 走独立的 `ApiKey__ResetSecret`（handler 侧对应 `/config/apikey/reset-secret`）。`normalizeTTL` 把 `15m`、`168h` 之类的输入统一格式化成 `Go duration.String()`（`15m0s`）再入库，避免列值形态漂移。

### 5.3 操作日志（`business_log.go`，全通路一张表）

```go
type Log_Source string
const (
    Log_Source_AKSK Log_Source = "aksk"         // → Operator = aksk:<api_key.Id>
    Log_Source_User Log_Source = "user"         // → Operator = user:<user_account.Id>
    Log_Source_Mqtt Log_Source = "internalmqtt" // → Operator 固定为通道名，不归属账号
)
func (s Log_Source) Operator(accountId uint) string   // 拼 Operator 列，唯一入口

// 两个维度都是命名 string 类型：列值即文本枚举，与常量一字不差；
// 分开定义而非统一用 string，是为了让 typ/operate 传反时编译器能拦住
type Log_Type string    // 常量 Log_Type_All / Log_Type_Drive / …
type Log_Operate string // 常量 Log_Operate_Add / Log_Operate_Write / …

type Log_Row struct{ Id uint; Type Log_Type; Operate Log_Operate; Operator, Content, Ip string; Create_Time time.Time }

type Log_Filter struct { Types []Log_Type; Operates []Log_Operate; Operators []string; Keyword string; Start, End time.Time; Page, PageSize uint }
func inCond[T ~string](col string, vals []T) (string, []interface{})  // 空列表返回空串=不加条件
func (f Log_Filter) buildWhere() (string, []interface{})               // Keyword 在 Operator/Content 上 LIKE

func Log__Add(data ...Log_Row) error                        // 批量插入，Operator 为空直接报错
func Log__Query_Callback(f Log_Filter, cb func(Log_Row)) error
func Log__Query(f Log_Filter) ([]Log_Row, error)
func Log__Count(f Log_Filter) (uint, error)
func Log_Del(ids ...uint) error                             // Id=0 视为非法入参
```

两个维度定位事件、一个列区分操作人，**新增事件类型不加列也不加表，也不用改 DDL**（枚举只是新字符串值）：

| `Type`（操作对象） | 列值 | `Operate`（操作方式） | 列值 |
|---|---|---|---|
| All 全部配置/未细分 | `all` | Add 增 | `add` |
| Drive 驱动 | `drive` | Del 删 | `del` |
| Point 点位 | `point` | Update 改 | `update` |
| Alarm 报警 | `alarm` | Query 查 | `query` |
| Log 日志 | `log` | Write 写值 | `write` |
| History 历史 | `history` | Login 登陆 | `login` |
| Account 账号权限 | `account` | Logout 登出 | `logout` |
| System 系统配置 | `system` | Restart 重启 | `restart` |
| Service 服务进程 | `service` | Import 导入 | `import` |
| | | Export 导出 | `export` |

- 通路/请求方法/路径/UA/IP/目标 id·tag/变更内容/结果均序列化进 `Content`（`text`，允许 NULL）；失败事件用 `detail` 里的 `ok:false` + `reason` 表达，**不占独立列**。
- `Content` 扫描用 `sql.NullString`（旧数据可能是 NULL）；排序一律 `ORDER BY Id DESC`，`Count` 也沿用带 `LIMIT ?, ?` 的项目口径。
- handler 侧截断上限与之对应：`Log_Content_Max=8000`、`Log_Operator_Max=64`、`Log_Ip_Max=45`，见 [handler README](../../handler/README.md)。
- **旧的 6 张分表（`log_{aksk,user}_{config,login,write}`）已废弃**：历史数据已按 `Create_Time` 归并进 `log_operate`（操作人由 `CONCAT` 生成，登陆/写值行用 `JSON_OBJECT` 补 `Content`），旧表已 DROP。
- `Type`/`Operate` 曾为 `tinyint`，已改为 `varchar(20)` 文本枚举：因 `CheckSqlStructure` **只建不改**（列类型不一致直接 panic），
  那步变更靠手工 SQL 完成：先 `ALTER TABLE … MODIFY`（数字自动变字符串），再 `UPDATE … CASE` 映射成英文值，并补上 `idx_operate`。
  以后改这两列只改常量，不需再动 DDL。

### 5.4 标签树（`business_tagtree.go`）

```go
Tag_Tree__RebuildFromPointConfig()        // 全量重建：解析 Point_Config.Tag 的路径段，逐级 upsert
Tag_Tree__Query_Children_ByPath(host, path string) // 懒加载：只取直接子节点，Has_Children 决定箭头
Tag_Tree__Query_PointsUnderPath(path string)       // 子树内全部点位（前端"选整个目录"）
```

`Path` 的唯一键是幂等基础：重建 = 按 Path upsert，不删整表，因此**手工加的文件夹节点不会被重建清掉**（但它若不含任何点位，也不会被自动补充）。Tag 支持两种格式，`Host` 取自 `//host//xxx` 的第一段；纯路径 Tag（`/a/b`）的 `Host` 为空串。

### 5.5 RBAC（`rbac.go` / `rbac_admin.go`）

```go
const ( Rbac_Subject_Biz uint64 = 1; Rbac_Subject_Api uint64 = 2 )

func LoadEffectiveRules(subjectType uint64, subjectId uint) ([]Rule, error)  // 角色规则 ∪ 直授权规则
func LoadRoleCodes(subjectType uint64, subjectId uint) ([]string, error)

RbacRole__Query/Count/Add/Update/Del      // t_biz_role / t_api_role
RbacPerm__Query/Count/Add/Update/Del      // t_permission
RolePerm__List/Replace(roleId)            // t_role_permission
SubjectRole__List/Replace(subjectType, subjectId)
SubjectPerm__List/Replace(subjectType, subjectId)   // t_direct_permission
```

`Replace` 系列是"整组覆盖"语义（先删后插，走 `trans()` 事务），供 `/config/rbac/*/set` 使用。上层缓存与判定逻辑在 [service/rbac](../../service/rbac/README.md) 与 [pkg/rbac](../../../pkg/rbac/README.md)。

`rbac.go` 里还有一组**写入函数：`BizRole__Add`、`ApiRole__Add`、`Permission__Add`、`RolePermission__Grant/Revoke`、`BizUserRole__Bind`、`ApiClientRole__Bind`、`DirectPermission__Grant`、`Permission__SetStatus`**。它们是全库唯一带 `ON DUPLICATE KEY UPDATE is_enabled = 1` 幂等复启用逻辑的实现，注释写着"供示例/管理端调用"，但 **当前没有任何调用方**（管理端实际走上面 `rbac_admin.go` 的 `Rbac*__*` 一族）。两套并存，改 RBAC 写路径时不要只改一边。

两个要点：
- `t_permission` **没有 action 列**，`scanRules` 里硬编码 `ru.Action = "*"` → 动作维度已并入主题串，判定引擎的 `action` 形参只剩日志价值。
- 有效规则的过滤条件（`status=1`、`is_enabled=1`、`expire_at IS NULL OR expire_at > NOW()`）全部写在 SQL 里，Go 侧不再判；表名只由固定的 `subjectType` 分支决定，绝不拼接用户输入。

---

## 六、校验与配置格式约定

```go
Regex_Tag_Format          = `^(//[^/]+//[^/]+(/[^/]+)*|/[^/]+(/[^/]+)*)$`
Regex_Conversion_Format   = `^(|0|[0-9a-zA-Z\s\+\-\*\/\.\(\),]*x[0-9a-zA-Z\s\+\-\*\/\.\(\),]*)$`
Regex_Config_Forbidden    = "(?i)('|`|\\\\|--|/\\*|\\*/|\\b(select|insert|update|delete|drop|alter|create|truncate|union|exec|declare|sleep|benchmark|load_file|information_schema|into\\s+outfile)\\b)"
Config_Max_Length         = 4096
```

- `Regex_Tag_Format`：两种形态——`//设备id//标识符(/…)…` 或 `/标识符(/…)…`；设备 id 与标识符均非空、不能以 `/` 结尾、中间不能出现 `//`。
- `Regex_Conversion_Format`：空串或 `"0"` = 无换算；否则**必须含变量 `x`**，允许数字、`+ - * /`、括号、逗号、小数点、空白与函数名（如 `round2(x/3, 2)`）。这只是脏数据初筛，非法标识符/函数会在运行时 expr 编译阶段报错。**反向换算（写值时把用户值反推成设备值）仅支持一元一次方程**，非线性表达式无法反推。
- 三档校验入口：

| 目标列 | 用哪个 | 规则 |
|---|---|---|
| `Drive_Config.Config` | `Config__Validate_JSON(name, s)` | trim 后非空、≤4096、**必须以 `{` 开头**、能 `json.Unmarshal` 成 `map[string]any`（先验前缀，好让旧 `;` 分隔配置得到"必须是 JSON 对象"而不是费解的 JSON 报错） |
| `Point_Config.Config` | `Config__Validate_Basic(name, s)` | 非空、≤4096、无控制字符（放行 `\t\n\r`）、不命中 forbidden 正则；**格式正确性交给驱动层 `Point_Config_Switch`**（Modbus 是 `:` 分隔串、S7 是地址串或 JSON、历史数据里还有 `;` 分隔串） |
| `Alarm/History_Config.Config` | `Config__Validate_Basic` | 装的是 expr 表达式/采集周期，本层不解释 |

> 三条约定与前端 `src/utils/configFormat.ts` 一一对应，**改这里必须同步前端**。

---

## 七、启动流程与排障速查

```
package mysql 的 init()
  └─ Init_sql(config.SQL.Link)      失败 → panic（连接串/网络/权限）
       └─ sql.Open + 4 项池参数 + PingContext(5s)
  └─ CheckSqlStructure()            遍历 expectTableRules
       └─ CheckAndFixTableStructure(每表)
            ├─ 表不存在 → CREATE TABLE（含索引/唯一键/默认值）
            ├─ 缺列     → ALTER TABLE ADD COLUMN（拒绝补自增列/主键列 → panic）
            ├─ 自增不符 → ALTER TABLE MODIFY COLUMN
            └─ 类型/非空/主键/默认值不符 → panic
```

| 现象 | 定位 | 处理 |
|---|---|---|
| 启动即 `panic: … connection refused` / `Access denied` | `Init_sql` | 检查 `configs/config.yaml` 的 `sql.link`；注意是 `init()` 触发，报错栈里没有 main |
| `panic: 列 Xxx 类型不匹配 期望 varchar(300) 实际 varchar(255)` | `CheckAndFixTableStructure` | 手工 `ALTER TABLE … MODIFY COLUMN` 对齐规则，或改规则并同步备份 SQL |
| 所有业务接口 `520` | RBAC 表缺失（坑 3） | 导入 `iot-collector-server_backup_*.sql` 建 8 张 RBAC 表 |
| 云端同步后驱动名称空白 | `Drive_Config__Sync` 漏 `Name`（坑 5） | 补字段，或改用 `Drive_Config__Add` 路径 |
| 某条点位配置保存报"含非法字符" | `Regex_Config_Forbidden`（坑 7） | 去掉单引号/反引号/`--`/SQL 关键字整词 |
| 全量导出很慢/内存高 | 用了 `__Query` 而非 `__Query_Callback` | 大数据量一律走 Callback 流式版本 |
| 反复报错但日志只有一条 | `logErrorThrottled` 30s 窗口（坑 9） | 看"已抑制 N 条"提示；临时调小 `Log_Error_Window` 复现 |

---

## 八、扩展指引

**新增一张自维护表**：在 `expectTableRules` 追加 `TableRule`（第一列必须 `IsAutoInc + IsPrimaryKey + int unsigned`），然后写对应的 `Xxx__*` 七件套。**若该表要参与 RBAC 行级过滤，Query/Count/Search 必须留 `idScope ...[]uint` 形参并在拼 WHERE 之前调用 `applyIdScope`**，否则分页与 total 口径会不一致。

**新增一类日志来源**：在 `Log_Source` 加常量 + 在 `meta()` 加 case + 在 `expectTableRules` 加三张表规则。三处缺一会走到 `default` 分支报"未知日志来源"。

**校验规则变更**：`Regex_*` 三个常量是全服务的单一事实源，同时被 `internal/driver/*`、`internal/handler/*` 与前端引用，改动前先全局搜索。

---

## 九、相关文档

- [服务总览 README](../../../README.md)
- [internal/config](../../config/README.md) — `sql.link` 的来源
- [internal/handler](../../handler/README.md) — `idScope` / `Log_*_type` 的上游调用方
- [internal/service/collector](../../service/collector/README.md) — 启动时全量加载配置
- [internal/service/configsync](../../service/configsync/README.md) — `Xxx__Sync` 的调用方
- [internal/service/rbac](../../service/rbac/README.md) — `LoadEffectiveRules` 的缓存层
