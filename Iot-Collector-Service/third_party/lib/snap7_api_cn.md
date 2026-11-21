# Snap7 C 接口函数说明

> 本文档基于 `snap7.h`，说明所有 C 导入函数（`S7API` 导出）的传入参数、返回类型及含义。
>
> **通用约定**：
> - 几乎所有函数返回 `int`，返回 `0` 表示成功，非零为错误码（`longword` 类型）。
> - `S7Object` 本质是 `uintptr_t`（32 位平台为 32 位无符号整数，64 位平台为 64 位无符号整数），用作对象句柄。
> - `void *pUsrData` 为用户提供的数据缓冲区指针。
> - `void *pValue` 为通用参数值指针，实际类型取决于 `ParamNumber`。

---

## 一、基础类型定义

| 类型 | 原始定义 | 说明 |
|------|---------|------|
| `byte` | `uint8_t` | 8 位无符号整数 |
| `word` | `uint16_t` | 16 位无符号整数 |
| `longword` | `uint32_t` | 32 位无符号整数 |
| `S7Object` | `uintptr_t` | 对象句柄（整数） |
| `pbyte` | `byte *` | 字节指针 |
| `pword` | `word *` | 字指针 |

---

## 二、客户端（Client）函数

### 2.1 生命周期

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_Create()` | 无 | `S7Object` | 创建客户端实例，返回句柄 |
| `Cli_Destroy(S7Object *Client)` | `Client` [S7Object 指针] | `void` | 销毁客户端实例，释放资源 |

### 2.2 连接管理

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_ConnectTo(S7Object Client, const char *Address, int Rack, int Slot)` | `Client` [S7Object] 句柄；`Address` [const char *] PLC IP 地址；`Rack` [int] 机架号；`Slot` [int] 槽号 | `int` | 连接到指定 PLC，返回 0 成功 |
| `Cli_SetConnectionParams(S7Object Client, const char *Address, word LocalTSAP, word RemoteTSAP)` | `Client` [S7Object] 句柄；`Address` [const char *] PLC IP；`LocalTSAP` [word] 本地 TSAP；`RemoteTSAP` [word] 远程 TSAP | `int` | 设置连接参数（TSAP 模式） |
| `Cli_SetConnectionType(S7Object Client, word ConnectionType)` | `Client` [S7Object] 句柄；`ConnectionType` [word] 连接类型（`CONNTYPE_PG`=0x0001 / `CONNTYPE_OP`=0x0002 / `CONNTYPE_BASIC`=0x0003） | `int` | 设置连接类型 |
| `Cli_Connect(S7Object Client)` | `Client` [S7Object] 句柄 | `int` | 使用已设置的参数建立连接 |
| `Cli_Disconnect(S7Object Client)` | `Client` [S7Object] 句柄 | `int` | 断开连接 |
| `Cli_GetConnected(S7Object Client, int *Connected)` | `Client` [S7Object] 句柄；`Connected` [int *] 输出参数 | `int` | 查询连接状态，`*Connected` 返回 1=已连接 / 0=未连接 |

### 2.3 参数设置

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_GetParam(S7Object Client, int ParamNumber, void *pValue)` | `Client` [S7Object] 句柄；`ParamNumber` [int] 参数编号（见参数列表）；`pValue` [void *] 输出缓冲区 | `int` | 获取参数值 |
| `Cli_SetParam(S7Object Client, int ParamNumber, void *pValue)` | `Client` [S7Object] 句柄；`ParamNumber` [int] 参数编号；`pValue` [void *] 参数值指针 | `int` | 设置参数值 |
| `Cli_SetAsCallback(S7Object Client, pfn_CliCompletion pCompletion, void *usrPtr)` | `Client` [S7Object] 句柄；`pCompletion` [函数指针] 回调 `void(void *usrPtr, int opCode, int opResult)`；`usrPtr` [void *] 用户指针 | `int` | 设置异步操作完成回调 |

**参数编号列表**：

| 常量名 | 值 | 类型 | 说明 |
|--------|---|------|------|
| `p_u16_LocalPort` | 1 | `uint16_t` | 本地端口 |
| `p_u16_RemotePort` | 2 | `uint16_t` | 远程端口 |
| `p_i32_PingTimeout` | 3 | `int32_t` | Ping 超时（ms） |
| `p_i32_SendTimeout` | 4 | `int32_t` | 发送超时（ms） |
| `p_i32_RecvTimeout` | 5 | `int32_t` | 接收超时（ms） |
| `p_i32_WorkInterval` | 6 | `int32_t` | 工作间隔（ms） |
| `p_u16_SrcRef` | 7 | `uint16_t` | 源引用 |
| `p_u16_DstRef` | 8 | `uint16_t` | 目标引用 |
| `p_u16_SrcTSap` | 9 | `uint16_t` | 源 TSAP |
| `p_i32_PDURequest` | 10 | `int32_t` | 请求 PDU 大小 |
| `p_i32_MaxClients` | 11 | `int32_t` | 最大客户端数 |
| `p_i32_BSendTimeout` | 12 | `int32_t` | BSend 超时（ms） |
| `p_i32_BRecvTimeout` | 13 | `int32_t` | BRecv 超时（ms） |
| `p_u32_RecoveryTime` | 14 | `uint32_t` | 恢复时间（ms） |
| `p_u32_KeepAliveTime` | 15 | `uint32_t` | 保活时间（ms） |

### 2.4 数据读写 — 主函数

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_ReadArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData)` | `Client` [S7Object]；`Area` [int] 区域 ID（`S7AreaPE`=0x81 过程输入 / `S7AreaPA`=0x82 过程输出 / `S7AreaMK`=0x83 标志位 / `S7AreaDB`=0x84 数据块 / `S7AreaCT`=0x1C 计数器 / `S7AreaTM`=0x1D 定时器）；`DBNumber` [int] DB 编号（非 DB 区域时为 0）；`Start` [int] 起始地址；`Amount` [int] 读取数量；`WordLen` [int] 字长度（`S7WLBit`=0x01 / `S7WLByte`=0x02 / `S7WLWord`=0x04 / `S7WLDWord`=0x06 / `S7WLReal`=0x08 / `S7WLCounter`=0x1C / `S7WLTimer`=0x1D）；`pUsrData` [void *] 接收缓冲区 | `int` | 读取 PLC 指定区域数据 |
| `Cli_WriteArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData)` | 同 `Cli_ReadArea`，`pUsrData` [void *] 为待写入数据缓冲区 | `int` | 向 PLC 指定区域写入数据 |
| `Cli_ReadMultiVars(S7Object Client, PS7DataItem Item, int ItemsCount)` | `Client` [S7Object]；`Item` [PS7DataItem] `TS7DataItem` 数组指针（最多 `MaxVars`=20 项）；`ItemsCount` [int] 项数 | `int` | 批量读取多个变量 |
| `Cli_WriteMultiVars(S7Object Client, PS7DataItem Item, int ItemsCount)` | 同 `Cli_ReadMultiVars` | `int` | 批量写入多个变量 |

**TS7DataItem 结构体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `Area` | `int` | 区域 ID |
| `WordLen` | `int` | 字长度 |
| `Result` | `int` | 操作结果（读取后填充） |
| `DBNumber` | `int` | DB 编号 |
| `Start` | `int` | 起始地址 |
| `Amount` | `int` | 数量 |
| `pdata` | `void *` | 数据缓冲区指针 |

### 2.5 数据读写 — 快捷函数

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_DBRead(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData)` | `Client` [S7Object]；`DBNumber` [int] DB 编号；`Start` [int] 起始字节；`Size` [int] 读取字节数；`pUsrData` [void *] 接收缓冲区 | `int` | 读取 DB 数据 |
| `Cli_DBWrite(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData)` | 同上，`pUsrData` [void *] 待写入数据 | `int` | 写入 DB 数据 |
| `Cli_MBRead(S7Object Client, int Start, int Size, void *pUsrData)` | `Client` [S7Object]；`Start` [int] 起始地址；`Size` [int] 字节数；`pUsrData` [void *] 接收缓冲区 | `int` | 读取标志位（M 区） |
| `Cli_MBWrite(S7Object Client, int Start, int Size, void *pUsrData)` | 同上 | `int` | 写入标志位（M 区） |
| `Cli_EBRead(S7Object Client, int Start, int Size, void *pUsrData)` | `Client` [S7Object]；`Start` [int] 起始地址；`Size` [int] 字节数；`pUsrData` [void *] 接收缓冲区 | `int` | 读取过程输入（E/I 区） |
| `Cli_EBWrite(S7Object Client, int Start, int Size, void *pUsrData)` | 同上 | `int` | 写入过程输入（E/I 区） |
| `Cli_ABRead(S7Object Client, int Start, int Size, void *pUsrData)` | `Client` [S7Object]；`Start` [int] 起始地址；`Size` [int] 字节数；`pUsrData` [void *] 接收缓冲区 | `int` | 读取过程输出（A/Q 区） |
| `Cli_ABWrite(S7Object Client, int Start, int Size, void *pUsrData)` | 同上 | `int` | 写入过程输出（A/Q 区） |
| `Cli_TMRead(S7Object Client, int Start, int Amount, void *pUsrData)` | `Client` [S7Object]；`Start` [int] 起始编号；`Amount` [int] 数量；`pUsrData` [void *] 接收缓冲区 | `int` | 读取定时器 |
| `Cli_TMWrite(S7Object Client, int Start, int Amount, void *pUsrData)` | 同上 | `int` | 写入定时器 |
| `Cli_CTRead(S7Object Client, int Start, int Amount, void *pUsrData)` | `Client` [S7Object]；`Start` [int] 起始编号；`Amount` [int] 数量；`pUsrData` [void *] 接收缓冲区 | `int` | 读取计数器 |
| `Cli_CTWrite(S7Object Client, int Start, int Amount, void *pUsrData)` | 同上 | `int` | 写入计数器 |

### 2.6 目录/块信息

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_ListBlocks(S7Object Client, TS7BlocksList *pUsrData)` | `Client` [S7Object]；`pUsrData` [TS7BlocksList *] 输出结构体 | `int` | 列出 PLC 中所有块的数量 |
| `Cli_GetAgBlockInfo(S7Object Client, int BlockType, int BlockNum, TS7BlockInfo *pUsrData)` | `Client` [S7Object]；`BlockType` [int] 块类型（`Block_OB`=0x38 / `Block_DB`=0x41 / `Block_FC`=0x43 等）；`BlockNum` [int] 块编号；`pUsrData` [TS7BlockInfo *] 输出块信息 | `int` | 获取 AG 块信息 |
| `Cli_GetPgBlockInfo(S7Object Client, void *pBlock, TS7BlockInfo *pUsrData, int Size)` | `Client` [S7Object]；`pBlock` [void *] 块数据；`pUsrData` [TS7BlockInfo *] 输出信息；`Size` [int] 数据大小 | `int` | 获取 PG 块信息 |
| `Cli_ListBlocksOfType(S7Object Client, int BlockType, TS7BlocksOfType *pUsrData, int *ItemsCount)` | `Client` [S7Object]；`BlockType` [int] 块类型；`pUsrData` [TS7BlocksOfType *] 输出数组；`ItemsCount` [int *] 输入/输出：返回数量 | `int` | 列出指定类型的所有块编号 |

### 2.7 块操作

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_Upload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size)` | `Client` [S7Object]；`BlockType` [int] 块类型；`BlockNum` [int] 块编号；`pUsrData` [void *] 接收缓冲区；`Size` [int *] 输入缓冲区大小/输出实际大小 | `int` | 上传块（不含头信息） |
| `Cli_FullUpload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size)` | 同上 | `int` | 完整上传块（含头信息） |
| `Cli_Download(S7Object Client, int BlockNum, void *pUsrData, int Size)` | `Client` [S7Object]；`BlockNum` [int] 块编号；`pUsrData` [void *] 块数据；`Size` [int] 数据大小 | `int` | 下载块到 PLC |
| `Cli_Delete(S7Object Client, int BlockType, int BlockNum)` | `Client` [S7Object]；`BlockType` [int] 块类型；`BlockNum` [int] 块编号 | `int` | 删除指定块 |
| `Cli_DBGet(S7Object Client, int DBNumber, void *pUsrData, int *Size)` | `Client` [S7Object]；`DBNumber` [int] DB 编号；`pUsrData` [void *] 接收缓冲区；`Size` [int *] 返回实际大小 | `int` | 获取整个 DB 内容 |
| `Cli_DBFill(S7Object Client, int DBNumber, int FillChar)` | `Client` [S7Object]；`DBNumber` [int] DB 编号；`FillChar` [int] 填充字节值 | `int` | 用指定值填充 DB |

### 2.8 日期/时间

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_GetPlcDateTime(S7Object Client, tm *DateTime)` | `Client` [S7Object]；`DateTime` [tm *] 输出时间结构体 | `int` | 获取 PLC 日期时间 |
| `Cli_SetPlcDateTime(S7Object Client, tm *DateTime)` | `Client` [S7Object]；`DateTime` [tm *] 要设置的时间 | `int` | 设置 PLC 日期时间 |
| `Cli_SetPlcSystemDateTime(S7Object Client)` | `Client` [S7Object] | `int` | 将 PLC 时间设为系统时间 |

### 2.9 系统信息

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_GetOrderCode(S7Object Client, TS7OrderCode *pUsrData)` | `Client` [S7Object]；`pUsrData` [TS7OrderCode *] 输出订货号信息 | `int` | 获取 PLC 订货号 |
| `Cli_GetCpuInfo(S7Object Client, TS7CpuInfo *pUsrData)` | `Client` [S7Object]；`pUsrData` [TS7CpuInfo *] 输出 CPU 信息 | `int` | 获取 CPU 信息 |
| `Cli_GetCpInfo(S7Object Client, TS7CpInfo *pUsrData)` | `Client` [S7Object]；`pUsrData` [TS7CpInfo *] 输出 CP 信息 | `int` | 获取通信处理器信息 |
| `Cli_ReadSZL(S7Object Client, int ID, int Index, TS7SZL *pUsrData, int *Size)` | `Client` [S7Object]；`ID` [int] SZL ID；`Index` [int] SZL Index；`pUsrData` [TS7SZL *] 接收缓冲区；`Size` [int *] 返回数据大小 | `int` | 读取系统状态列表（SZL） |
| `Cli_ReadSZLList(S7Object Client, TS7SZLList *pUsrData, int *ItemsCount)` | `Client` [S7Object]；`pUsrData` [TS7SZLList *] 输出列表；`ItemsCount` [int *] 返回条目数 | `int` | 读取可用 SZL ID 列表 |

### 2.10 PLC 控制

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_PlcHotStart(S7Object Client)` | `Client` [S7Object] | `int` | PLC 热启动 |
| `Cli_PlcColdStart(S7Object Client)` | `Client` [S7Object] | `int` | PLC 冷启动 |
| `Cli_PlcStop(S7Object Client)` | `Client` [S7Object] | `int` | PLC 停止 |
| `Cli_CopyRamToRom(S7Object Client, int Timeout)` | `Client` [S7Object]；`Timeout` [int] 超时时间（ms） | `int` | 将 RAM 复制到 ROM |
| `Cli_Compress(S7Object Client, int Timeout)` | `Client` [S7Object]；`Timeout` [int] 超时时间（ms） | `int` | 压缩 PLC 内存 |
| `Cli_GetPlcStatus(S7Object Client, int *Status)` | `Client` [S7Object]；`Status` [int *] 输出状态值 | `int` | 获取 PLC 状态（`S7CpuStatusRun`=0x08 / `S7CpuStatusStop`=0x04 / `S7CpuStatusUnknown`=0x00） |

### 2.11 安全/密码

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_GetProtection(S7Object Client, TS7Protection *pUsrData)` | `Client` [S7Object]；`pUsrData` [TS7Protection *] 输出保护信息 | `int` | 获取 PLC 保护级别 |
| `Cli_SetSessionPassword(S7Object Client, char *Password)` | `Client` [S7Object]；`Password` [char *] 密码字符串 | `int` | 设置会话密码 |
| `Cli_ClearSessionPassword(S7Object Client)` | `Client` [S7Object] | `int` | 清除会话密码 |

### 2.12 底层 / 杂项

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_IsoExchangeBuffer(S7Object Client, void *pUsrData, int *Size)` | `Client` [S7Object]；`pUsrData` [void *] 发送/接收缓冲区；`Size` [int *] 输入发送大小/输出接收大小 | `int` | ISO 层缓冲区交换 |
| `Cli_GetExecTime(S7Object Client, int *Time)` | `Client` [S7Object]；`Time` [int *] 输出执行时间（ms） | `int` | 获取最后一次执行时间 |
| `Cli_GetLastError(S7Object Client, int *LastError)` | `Client` [S7Object]；`LastError` [int *] 输出最后错误码 | `int` | 获取最后一次错误码 |
| `Cli_GetPduLength(S7Object Client, int *Requested, int *Negotiated)` | `Client` [S7Object]；`Requested` [int *] 输出请求的 PDU 长度；`Negotiated` [int *] 输出协商后的 PDU 长度 | `int` | 获取 PDU 长度信息 |
| `Cli_ErrorText(int Error, char *Text, int TextLen)` | `Error` [int] 错误码；`Text` [char *] 输出文本缓冲区；`TextLen` [int] 缓冲区长度 | `int` | 将错误码转换为可读文本 |

### 2.13 异步函数

> 异步函数与同步函数参数一致，前缀为 `Cli_As`，立即返回，不等待 PLC 响应。
> 需配合 `Cli_CheckAsCompletion` 或 `Cli_WaitAsCompletion` 使用。

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Cli_AsReadArea(...)` | 同 `Cli_ReadArea` | `int` | 异步读取区域 |
| `Cli_AsWriteArea(...)` | 同 `Cli_WriteArea` | `int` | 异步写入区域 |
| `Cli_AsDBRead(...)` | 同 `Cli_DBRead` | `int` | 异步读 DB |
| `Cli_AsDBWrite(...)` | 同 `Cli_DBWrite` | `int` | 异步写 DB |
| `Cli_AsMBRead(...)` | 同 `Cli_MBRead` | `int` | 异步读标志位 |
| `Cli_AsMBWrite(...)` | 同 `Cli_MBWrite` | `int` | 异步写标志位 |
| `Cli_AsEBRead(...)` | 同 `Cli_EBRead` | `int` | 异步读过程输入 |
| `Cli_AsEBWrite(...)` | 同 `Cli_EBWrite` | `int` | 异步写过程输入 |
| `Cli_AsABRead(...)` | 同 `Cli_ABRead` | `int` | 异步读过程输出 |
| `Cli_AsABWrite(...)` | 同 `Cli_ABWrite` | `int` | 异步写过程输出 |
| `Cli_AsTMRead(...)` | 同 `Cli_TMRead` | `int` | 异步读定时器 |
| `Cli_AsTMWrite(...)` | 同 `Cli_TMWrite` | `int` | 异步写定时器 |
| `Cli_AsCTRead(...)` | 同 `Cli_CTRead` | `int` | 异步读计数器 |
| `Cli_AsCTWrite(...)` | 同 `Cli_CTWrite` | `int` | 异步写计数器 |
| `Cli_AsListBlocksOfType(...)` | 同 `Cli_ListBlocksOfType` | `int` | 异步列出块 |
| `Cli_AsReadSZL(...)` | 同 `Cli_ReadSZL` | `int` | 异步读 SZL |
| `Cli_AsReadSZLList(...)` | 同 `Cli_ReadSZLList` | `int` | 异步读 SZL 列表 |
| `Cli_AsUpload(...)` | 同 `Cli_Upload` | `int` | 异步上传 |
| `Cli_AsFullUpload(...)` | 同 `Cli_FullUpload` | `int` | 异步完整上传 |
| `Cli_AsDownload(...)` | 同 `Cli_Download` | `int` | 异步下载 |
| `Cli_AsCopyRamToRom(int Timeout)` | `Timeout` [int] 超时 ms | `int` | 异步 RAM 到 ROM |
| `Cli_AsCompress(int Timeout)` | `Timeout` [int] 超时 ms | `int` | 异步压缩 |
| `Cli_AsDBGet(...)` | 同 `Cli_DBGet` | `int` | 异步获取 DB |
| `Cli_AsDBFill(...)` | 同 `Cli_DBFill` | `int` | 异步填充 DB |
| `Cli_CheckAsCompletion(S7Object Client, int *opResult)` | `Client` [S7Object]；`opResult` [int *] 输出操作结果 | `int` | 检查异步操作是否完成（返回 `JobComplete`=0 或 `JobPending`=1） |
| `Cli_WaitAsCompletion(S7Object Client, int Timeout)` | `Client` [S7Object]；`Timeout` [int] 等待超时（ms） | `int` | 等待异步操作完成 |

---

## 三、服务端（Server）函数

### 3.1 生命周期

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Srv_Create()` | 无 | `S7Object` | 创建服务端实例 |
| `Srv_Destroy(S7Object *Server)` | `Server` [S7Object *] 句柄指针 | `void` | 销毁服务端实例 |

### 3.2 启停控制

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Srv_Start(S7Object Server)` | `Server` [S7Object] 句柄 | `int` | 启动服务端（监听所有接口） |
| `Srv_StartTo(S7Object Server, const char *Address)` | `Server` [S7Object]；`Address` [const char *] 绑定 IP 地址 | `int` | 绑定指定地址启动 |
| `Srv_Stop(S7Object Server)` | `Server` [S7Object] | `int` | 停止服务端 |

### 3.3 参数与状态

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Srv_GetParam(S7Object Server, int ParamNumber, void *pValue)` | `Server` [S7Object]；`ParamNumber` [int] 参数编号；`pValue` [void *] 输出缓冲区 | `int` | 获取参数 |
| `Srv_SetParam(S7Object Server, int ParamNumber, void *pValue)` | `Server` [S7Object]；`ParamNumber` [int]；`pValue` [void *] 值指针 | `int` | 设置参数 |
| `Srv_GetStatus(S7Object Server, int *ServerStatus, int *CpuStatus, int *ClientsCount)` | `Server` [S7Object]；`ServerStatus` [int *] 输出服务端状态；`CpuStatus` [int *] 输出 CPU 状态；`ClientsCount` [int *] 输出客户端数 | `int` | 获取服务端运行状态 |
| `Srv_SetCpuStatus(S7Object Server, int CpuStatus)` | `Server` [S7Object]；`CpuStatus` [int] CPU 状态值 | `int` | 设置模拟的 CPU 状态 |
| `Srv_ClearEvents(S7Object Server)` | `Server` [S7Object] | `int` | 清除事件队列 |

### 3.4 区域管理

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Srv_RegisterArea(S7Object Server, int AreaCode, word Index, void *pUsrData, int Size)` | `Server` [S7Object]；`AreaCode` [int] 区域代码（`srvAreaPE`=0 / `srvAreaPA`=1 / `srvAreaMK`=2 / `srvAreaCT`=3 / `srvAreaTM`=4 / `srvAreaDB`=5）；`Index` [word] 区域索引（DB 编号等）；`pUsrData` [void *] 数据缓冲区指针；`Size` [int] 缓冲区大小（字节） | `int` | 注册共享区域 |
| `Srv_UnregisterArea(S7Object Server, int AreaCode, word Index)` | `Server` [S7Object]；`AreaCode` [int]；`Index` [word] | `int` | 注销已注册的区域 |
| `Srv_LockArea(S7Object Server, int AreaCode, word Index)` | `Server` [S7Object]；`AreaCode` [int]；`Index` [word] | `int` | 锁定区域（客户端不可访问） |
| `Srv_UnlockArea(S7Object Server, int AreaCode, word Index)` | `Server` [S7Object]；`AreaCode` [int]；`Index` [word] | `int` | 解锁区域 |

### 3.5 事件处理

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Srv_PickEvent(S7Object Server, TSrvEvent *pEvent, int *EvtReady)` | `Server` [S7Object]；`pEvent` [TSrvEvent *] 输出事件结构体；`EvtReady` [int *] 输出是否有事件（1=有 / 0=无） | `int` | 从队列取出一个事件 |
| `Srv_GetMask(S7Object Server, int MaskKind, longword *Mask)` | `Server` [S7Object]；`MaskKind` [int] 掩码类型（`mkEvent`=0 / `mkLog`=1）；`Mask` [longword *] 输出掩码值 | `int` | 获取事件/日志掩码 |
| `Srv_SetMask(S7Object Server, int MaskKind, longword Mask)` | `Server` [S7Object]；`MaskKind` [int]；`Mask` [longword] 掩码值 | `int` | 设置事件/日志掩码 |
| `Srv_SetEventsCallback(S7Object Server, pfn_SrvCallBack pCallback, void *usrPtr)` | `Server` [S7Object]；`pCallback` [函数指针] `void(void *usrPtr, PSrvEvent PEvent, int Size)`；`usrPtr` [void *] | `int` | 设置事件回调 |
| `Srv_SetReadEventsCallback(S7Object Server, pfn_SrvCallBack pCallback, void *usrPtr)` | 同上 | `int` | 设置读取事件回调 |
| `Srv_SetRWAreaCallback(S7Object Server, pfn_RWAreaCallBack pCallback, void *usrPtr)` | `Server` [S7Object]；`pCallback` [函数指针] `int(void *usrPtr, int Sender, int Operation, PS7Tag PTag, void *pUsrData)`；`usrPtr` [void *] | `int` | 设置区域读写回调 |
| `Srv_EventText(TSrvEvent *Event, char *Text, int TextLen)` | `Event` [TSrvEvent *] 事件结构体；`Text` [char *] 输出文本；`TextLen` [int] 缓冲区长度 | `int` | 事件转文本 |
| `Srv_ErrorText(int Error, char *Text, int TextLen)` | `Error` [int] 错误码；`Text` [char *] 输出文本；`TextLen` [int] 缓冲区长度 | `int` | 错误码转文本 |

---

## 四、伙伴（Partner）函数

> Partner 用于两个 S7 节点之间的 S7 协议数据交换（BSend/BRecv）。

### 4.1 生命周期

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_Create(int Active)` | `Active` [int] 1=主动模式 / 0=被动模式 | `S7Object` | 创建 Partner 实例 |
| `Par_Destroy(S7Object *Partner)` | `Partner` [S7Object *] 句柄指针 | `void` | 销毁 Partner 实例 |

### 4.2 启停控制

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_Start(S7Object Partner)` | `Partner` [S7Object] | `int` | 启动 Partner |
| `Par_StartTo(S7Object Partner, const char *LocalAddress, const char *RemoteAddress, word LocTsap, word RemTsap)` | `Partner` [S7Object]；`LocalAddress` [const char *] 本地 IP；`RemoteAddress` [const char *] 远程 IP；`LocTsap` [word] 本地 TSAP；`RemTsap` [word] 远程 TSAP | `int` | 指定地址启动 |
| `Par_Stop(S7Object Partner)` | `Partner` [S7Object] | `int` | 停止 Partner |

### 4.3 参数

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_GetParam(S7Object Partner, int ParamNumber, void *pValue)` | `Partner` [S7Object]；`ParamNumber` [int]；`pValue` [void *] 输出缓冲区 | `int` | 获取参数 |
| `Par_SetParam(S7Object Partner, int ParamNumber, void *pValue)` | `Partner` [S7Object]；`ParamNumber` [int]；`pValue` [void *] 值指针 | `int` | 设置参数 |

### 4.4 BSend（发送数据）

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_BSend(S7Object Partner, longword R_ID, void *pUsrData, int Size)` | `Partner` [S7Object]；`R_ID` [longword] 远程 ID（标识接收方）；`pUsrData` [void *] 发送数据缓冲区；`Size` [int] 数据大小（字节） | `int` | 同步发送数据 |
| `Par_AsBSend(S7Object Partner, longword R_ID, void *pUsrData, int Size)` | 同上 | `int` | 异步发送数据 |
| `Par_CheckAsBSendCompletion(S7Object Partner, int *opResult)` | `Partner` [S7Object]；`opResult` [int *] 输出操作结果 | `int` | 检查异步发送是否完成 |
| `Par_WaitAsBSendCompletion(S7Object Partner, longword Timeout)` | `Partner` [S7Object]；`Timeout` [longword] 超时（ms） | `int` | 等待异步发送完成 |
| `Par_SetSendCallback(S7Object Partner, pfn_ParSendCompletion pCompletion, void *usrPtr)` | `Partner` [S7Object]；`pCompletion` [函数指针] `void(void *usrPtr, int opResult)`；`usrPtr` [void *] | `int` | 设置发送完成回调 |

### 4.5 BRecv（接收数据）

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_BRecv(S7Object Partner, longword *R_ID, void *pData, int *Size, longword Timeout)` | `Partner` [S7Object]；`R_ID` [longword *] 输出远程 ID；`pData` [void *] 接收缓冲区；`Size` [int *] 输出数据大小；`Timeout` [longword] 超时（ms） | `int` | 同步接收数据（阻塞） |
| `Par_CheckAsBRecvCompletion(S7Object Partner, int *opResult, longword *R_ID, void *pData, int *Size)` | `Partner` [S7Object]；`opResult` [int *] 输出结果；`R_ID` [longword *] 输出远程 ID；`pData` [void *] 接收缓冲区；`Size` [int *] 输出大小 | `int` | 检查异步接收是否完成 |
| `Par_SetRecvCallback(S7Object Partner, pfn_ParRecvCallBack pCompletion, void *usrPtr)` | `Partner` [S7Object]；`pCompletion` [函数指针] `void(void *usrPtr, int opResult, longword R_ID, void *pData, int Size)`；`usrPtr` [void *] | `int` | 设置数据接收回调 |

### 4.6 状态与统计

| 函数 | 传入参数 | 返回类型 | 说明 |
|------|---------|---------|------|
| `Par_GetTimes(S7Object Partner, longword *SendTime, longword *RecvTime)` | `Partner` [S7Object]；`SendTime` [longword *] 输出发送时间（ms）；`RecvTime` [longword *] 输出接收时间（ms） | `int` | 获取发送/接收耗时 |
| `Par_GetStats(S7Object Partner, longword *BytesSent, longword *BytesRecv, longword *SendErrors, longword *RecvErrors)` | `Partner` [S7Object]；`BytesSent` [longword *] 输出已发送字节数；`BytesRecv` [longword *] 输出已接收字节数；`SendErrors` [longword *] 输出发送错误数；`RecvErrors` [longword *] 输出接收错误数 | `int` | 获取传输统计 |
| `Par_GetLastError(S7Object Partner, int *LastError)` | `Partner` [S7Object]；`LastError` [int *] 输出最后错误码 | `int` | 获取最后错误 |
| `Par_GetStatus(S7Object Partner, int *Status)` | `Partner` [S7Object]；`Status` [int *] 输出状态值 | `int` | 获取 Partner 状态（`par_stopped`=0 / `par_connecting`=1 / `par_waiting`=2 / `par_linked`=3 / `par_sending`=4 / `par_receiving`=5 / `par_binderror`=6） |
| `Par_ErrorText(int Error, char *Text, int TextLen)` | `Error` [int] 错误码；`Text` [char *] 输出文本；`TextLen` [int] 缓冲区长度 | `int` | 错误码转文本 |

---

## 五、结构体速查

### TS7Tag（读写回调中的标签信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `Area` | `int` | 区域 |
| `DBNumber` | `int` | DB 编号 |
| `Start` | `int` | 起始地址 |
| `Size` | `int` | 数据大小 |
| `WordLen` | `int` | 字长度 |

### TSrvEvent（服务端事件）

| 字段 | 类型 | 说明 |
|------|------|------|
| `EvtTime` | `time_t` | 事件时间戳 |
| `EvtSender` | `int` | 发送者 |
| `EvtCode` | `longword` | 事件代码 |
| `EvtRetCode` | `word` | 事件结果代码 |
| `EvtParam1` ~ `EvtParam4` | `word` | 事件参数 1~4 |

### TS7CpuInfo（CPU 信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `ModuleTypeName` | `char[33]` | 模块类型名称 |
| `SerialNumber` | `char[25]` | 序列号 |
| `ASName` | `char[25]` | AS 名称 |
| `Copyright` | `char[27]` | 版权信息 |
| `ModuleName` | `char[25]` | 模块名称 |

### TS7CpInfo（通信处理器信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `MaxPduLengt` | `int` | 最大 PDU 长度 |
| `MaxConnections` | `int` | 最大连接数 |
| `MaxMpiRate` | `int` | 最大 MPI 速率 |
| `MaxBusRate` | `int` | 最大总线速率 |

### TS7Protection（保护信息）

| 字段 | 类型 | 说明 |
|------|------|------|
| `sch_schal` | `word` | 保护级别 |
| `sch_par` | `word` | 参数 |
| `sch_rel` | `word` | 相关 |
| `bart_sch` | `word` | 保护开关位置 |
| `anl_sch` | `word` | 启动保护 |
