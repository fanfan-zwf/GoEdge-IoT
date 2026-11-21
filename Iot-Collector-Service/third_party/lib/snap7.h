/*=============================================================================|
|  PROJECT SNAP7                                                         1.4.0 |
|==============================================================================|
|  Copyright (C) 2013, 2014 Davide Nardella                                    |
|  All rights reserved.                                                        |
|==============================================================================|
|  SNAP7 is free software: you can redistribute it and/or modify               |
|  it under the terms of the Lesser GNU General Public License as published by |
|  the Free Software Foundation, either version 3 of the License, or           |
|  (at your option) any later version.                                         |
|                                                                              |
|  It means that you can distribute your commercial software linked with       |
|  SNAP7 without the requirement to distribute the source code of your         |
|  application and without the requirement that your application be itself     |
|  distributed under LGPL.                                                     |
|                                                                              |
|  SNAP7 is distributed in the hope that it will be useful,                    |
|  but WITHOUT ANY WARRANTY; without even the implied warranty of              |
|  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the               |
|  Lesser GNU General Public License for more details.                         |
|                                                                              |
|  You should have received a copy of the GNU General Public License and a     |
|  copy of Lesser GNU General Public License along with Snap7.                 |
|  If not, see  http://www.gnu.org/licenses/                                   |
|==============================================================================|
|                                                                              |
|  C/C++ Snap 7 classes/Imports definitions                                    |
|                                                                              |
|=============================================================================*/
#ifndef snap7_h
#define snap7_h
//---------------------------------------------------------------------------
// Platform detection
//---------------------------------------------------------------------------
#if defined (_WIN32)||defined(_WIN64)||defined(__WIN32__)||defined(__WINDOWS__)
# define OS_WINDOWS
#endif

// Visual Studio needs this to use the correct time_t size
#if defined (_WIN32) && !defined(_WIN64)
# define _USE_32BIT_TIME_T 
#endif

#if defined(unix) || defined(__unix__) || defined(__unix)
# define PLATFORM_UNIX
#endif

#if defined(__SVR4) || defined(__svr4__)
# define OS_SOLARIS
#endif

#if BSD>=0
# define OS_BSD
#endif

#if defined(__APPLE__)
# define OS_OSX
#endif

#if defined(PLATFORM_UNIX) || defined(OS_OSX)
# include <unistd.h>
# if defined(_POSIX_VERSION)
#   define POSIX
# endif
#endif

//---------------------------------------------------------------------------
// C++ Library
//---------------------------------------------------------------------------
#ifdef __cplusplus
#include <string>
#include <time.h>

// Visual C++ not C99 compliant (VS2008--)
#ifdef _MSC_VER
# if _MSC_VER >= 1600
#  include <stdint.h>  // VS2010++ have it 
# else
   typedef signed __int8     int8_t;
   typedef signed __int16    int16_t;
   typedef signed __int32    int32_t;
   typedef signed __int64    int64_t;
   typedef unsigned __int8   uint8_t;
   typedef unsigned __int16  uint16_t;
   typedef unsigned __int32  uint32_t;
   typedef unsigned __int64  uint64_t;
   #ifdef _WIN64
     typedef unsigned __int64  uintptr_t;
   #else
     typedef unsigned __int32  uintptr_t;
   #endif
# endif
#else
# include <stdint.h>
#endif

extern "C" {
#endif
//---------------------------------------------------------------------------
// C exact length types
//---------------------------------------------------------------------------
#ifndef __cplusplus

#ifdef OS_BSD
#  include <stdint.h>
#  include <time.h>
#endif

#ifdef OS_OSX
#  include <stdint.h>  
#  include <time.h>
#endif

#ifdef OS_SOLARIS
#  include <stdint.h>  
#  include <time.h>
#endif

#if defined(_UINTPTR_T_DEFINED)
#  include <stdint.h>
#  include <time.h>
#endif

#if !defined(_UINTPTR_T_DEFINED) && !defined(OS_SOLARIS) && !defined(OS_BSD) && !defined(OS_OSX)
  typedef unsigned char   uint8_t;  //  8 bit unsigned integer
  typedef unsigned short  uint16_t; // 16 bit unsigned integer
  typedef unsigned int    uint32_t; // 32 bit unsigned integer
  typedef unsigned long   uintptr_t;// 64 bit unsigned integer
#endif

#endif

#ifdef OS_WINDOWS
# define S7API __stdcall
#else
# define S7API
#endif

#pragma pack(1)
//******************************************************************************
//                                   COMMON
//******************************************************************************
// Exact length types regardless of platform/processor
typedef uint8_t    byte;
typedef uint16_t   word;
typedef uint32_t   longword;
typedef byte       *pbyte;
typedef word       *pword;
typedef uintptr_t  S7Object; // multi platform/processor object reference
                             // DON'T CONFUSE IT WITH AN OLE OBJECT, IT'S SIMPLY
                             // AN INTEGER VALUE (32 OR 64 BIT) USED AS HANDLE.

#ifndef __cplusplus
typedef struct
{
  int   tm_sec;
  int   tm_min;
  int   tm_hour;
  int   tm_mday;
  int   tm_mon;
  int   tm_year;
  int   tm_wday;
  int   tm_yday;
  int   tm_isdst;
}tm;

typedef int bool;
#define false 0;
#define true  1;
#endif

// 库级错误码
const int errLibInvalidParam  = -1;  // 无效参数
const int errLibInvalidObject = -2;  // 无效对象句柄

// CPU 状态
#define S7CpuStatusUnknown  0x00  // 未知
#define S7CpuStatusRun      0x08  // 运行中
#define S7CpuStatusStop     0x04  // 已停止

// ISO 层错误码
const longword errIsoConnect            = 0x00010000; // 连接错误
const longword errIsoDisconnect         = 0x00020000; // 断开错误
const longword errIsoInvalidPDU         = 0x00030000; // PDU 格式错误
const longword errIsoInvalidDataSize    = 0x00040000; // 传入发送/接收缓冲区的数据大小无效
const longword errIsoNullPointer    	= 0x00050000; // 传入了空指针
const longword errIsoShortPacket    	= 0x00060000; // 收到短包
const longword errIsoTooManyFragments   = 0x00070000; // 无 EoT 标志的分片过多
const longword errIsoPduOverflow    	= 0x00080000; // 分片数据总和超过最大包大小
const longword errIsoSendPacket         = 0x00090000; // 发送过程中出错
const longword errIsoRecvPacket         = 0x000A0000; // 接收过程中出错
const longword errIsoInvalidParams    	= 0x000B0000; // TSAP 参数无效
const longword errIsoResvd_1            = 0x000C0000; // 未分配
const longword errIsoResvd_2            = 0x000D0000; // 未分配
const longword errIsoResvd_3            = 0x000E0000; // 未分配
const longword errIsoResvd_4            = 0x000F0000; // 未分配

// Tag 结构体（读写回调中的标签信息）
typedef struct{
	int Area;      // 区域 ID
	int DBNumber;  // DB 编号
	int Start;     // 起始地址
	int Size;      // 数据大小(字节)
	int WordLen;   // 字长度
}TS7Tag, *PS7Tag;

//------------------------------------------------------------------------------
//                                  PARAMS LIST            
//------------------------------------------------------------------------------
const int p_u16_LocalPort  	    = 1;   // 本地端口 (uint16_t)
const int p_u16_RemotePort 	    = 2;   // 远程端口 (uint16_t)
const int p_i32_PingTimeout	    = 3;   // Ping 超时 ms (int32_t)
const int p_i32_SendTimeout     = 4;   // 发送超时 ms (int32_t)
const int p_i32_RecvTimeout     = 5;   // 接收超时 ms (int32_t)
const int p_i32_WorkInterval    = 6;   // 工作间隔 ms (int32_t)
const int p_u16_SrcRef          = 7;   // 源引用 (uint16_t)
const int p_u16_DstRef          = 8;   // 目标引用 (uint16_t)
const int p_u16_SrcTSap         = 9;   // 源 TSAP (uint16_t)
const int p_i32_PDURequest      = 10;  // 请求 PDU 大小 (int32_t)
const int p_i32_MaxClients      = 11;  // 最大客户端数 (int32_t)
const int p_i32_BSendTimeout    = 12;  // BSend 超时 ms (int32_t)
const int p_i32_BRecvTimeout    = 13;  // BRecv 超时 ms (int32_t)
const int p_u32_RecoveryTime    = 14;  // 恢复时间 ms (uint32_t)
const int p_u32_KeepAliveTime   = 15;  // 保活时间 ms (uint32_t)

// Client/Partner 异步任务状态
const int JobComplete           = 0;   // 已完成
const int JobPending            = 1;   // 进行中

//******************************************************************************
//                                   CLIENT
//******************************************************************************

// 客户端错误码
const longword errNegotiatingPDU            = 0x00100000; // PDU 协商失败
const longword errCliInvalidParams          = 0x00200000; // 参数无效
const longword errCliJobPending             = 0x00300000; // 任务进行中
const longword errCliTooManyItems           = 0x00400000; // 项目数过多
const longword errCliInvalidWordLen         = 0x00500000; // 字长度无效
const longword errCliPartialDataWritten     = 0x00600000; // 部分数据已写入
const longword errCliSizeOverPDU            = 0x00700000; // 数据大小超过 PDU
const longword errCliInvalidPlcAnswer       = 0x00800000; // PLC 响应无效
const longword errCliAddressOutOfRange      = 0x00900000; // 地址超出范围
const longword errCliInvalidTransportSize   = 0x00A00000; // 传输大小无效
const longword errCliWriteDataSizeMismatch  = 0x00B00000; // 写入数据大小不匹配
const longword errCliItemNotAvailable       = 0x00C00000; // 项目不可用
const longword errCliInvalidValue           = 0x00D00000; // 值无效
const longword errCliCannotStartPLC         = 0x00E00000; // 无法启动 PLC
const longword errCliAlreadyRun             = 0x00F00000; // PLC 已在运行
const longword errCliCannotStopPLC          = 0x01000000; // 无法停止 PLC
const longword errCliCannotCopyRamToRom     = 0x01100000; // 无法复制 RAM 到 ROM
const longword errCliCannotCompress         = 0x01200000; // 无法压缩
const longword errCliAlreadyStop            = 0x01300000; // PLC 已停止
const longword errCliFunNotAvailable        = 0x01400000; // 功能不可用
const longword errCliUploadSequenceFailed   = 0x01500000; // 上传序列失败
const longword errCliInvalidDataSizeRecvd   = 0x01600000; // 接收到的数据大小无效
const longword errCliInvalidBlockType       = 0x01700000; // 块类型无效
const longword errCliInvalidBlockNumber     = 0x01800000; // 块编号无效
const longword errCliInvalidBlockSize       = 0x01900000; // 块大小无效
const longword errCliDownloadSequenceFailed = 0x01A00000; // 下载序列失败
const longword errCliInsertRefused          = 0x01B00000; // 插入被拒绝
const longword errCliDeleteRefused          = 0x01C00000; // 删除被拒绝
const longword errCliNeedPassword           = 0x01D00000; // 需要密码
const longword errCliInvalidPassword        = 0x01E00000; // 密码无效
const longword errCliNoPasswordToSetOrClear = 0x01F00000; // 无需设置或清除密码
const longword errCliJobTimeout             = 0x02000000; // 任务超时
const longword errCliPartialDataRead        = 0x02100000; // 部分数据已读取
const longword errCliBufferTooSmall         = 0x02200000; // 缓冲区太小
const longword errCliFunctionRefused        = 0x02300000; // 功能被拒绝
const longword errCliDestroying             = 0x02400000; // 正在销毁
const longword errCliInvalidParamNumber     = 0x02500000; // 参数编号无效
const longword errCliCannotChangeParam      = 0x02600000; // 无法更改参数

const int MaxVars     = 20; // MultiRead/MultiWrite 最大变量数

// 客户端连接类型
const word CONNTYPE_PG                      = 0x0001;  // 以 PG 身份连接 PLC
const word CONNTYPE_OP                      = 0x0002;  // 以 OP 身份连接 PLC
const word CONNTYPE_BASIC                   = 0x0003;  // 基本连接

// 区域 ID
const byte S7AreaPE   =	0x81;  // 过程输入(I)
const byte S7AreaPA   =	0x82;  // 过程输出(Q)
const byte S7AreaMK   =	0x83;  // 标志位(M)
const byte S7AreaDB   =	0x84;  // 数据块(DB)
const byte S7AreaCT   =	0x1C;  // 计数器(C)
const byte S7AreaTM   =	0x1D;  // 定时器(T)

// 字长度
const int S7WLBit     = 0x01;  // 位
const int S7WLByte    = 0x02;  // 字节
const int S7WLWord    = 0x04;  // 字(16位)
const int S7WLDWord   = 0x06;  // 双字(32位)
const int S7WLReal    = 0x08;  // 浮点数(32位 IEEE 754)
const int S7WLCounter = 0x1C;  // 计数器
const int S7WLTimer   = 0x1D;  // 定时器

// 块类型
const byte Block_OB   = 0x38;  // 组织块
const byte Block_DB   = 0x41;  // 数据块
const byte Block_SDB  = 0x42;  // 系统数据块
const byte Block_FC   = 0x43;  // 功能
const byte Block_SFC  = 0x44;  // 系统功能
const byte Block_FB   = 0x45;  // 功能块
const byte Block_SFB  = 0x46;  // 系统功能块

// 子块类型
const byte SubBlk_OB  = 0x08;  // 组织块
const byte SubBlk_DB  = 0x0A;  // 数据块
const byte SubBlk_SDB = 0x0B;  // 系统数据块
const byte SubBlk_FC  = 0x0C;  // 功能
const byte SubBlk_SFC = 0x0D;  // 系统功能
const byte SubBlk_FB  = 0x0E;  // 功能块
const byte SubBlk_SFB = 0x0F;  // 系统功能块

// 块编程语言
const byte BlockLangAWL       = 0x01;  // 语句表(AWL/LAD)
const byte BlockLangKOP       = 0x02;  // 梯形图(KOP/FBD)
const byte BlockLangFUP       = 0x03;  // 功能图(FUP)
const byte BlockLangSCL       = 0x04;  // 结构化控制语言(SCL)
const byte BlockLangDB        = 0x05;  // 数据块(DB)
const byte BlockLangGRAPH     = 0x06;  // 顺序图(GRAPH)

// 读写多变量结构体
typedef struct{
   int   Area;      // 区域 ID
   int   WordLen;   // 字长度
   int   Result;    // 操作结果(读取后填充)
   int   DBNumber;  // DB 编号
   int   Start;     // 起始地址
   int   Amount;    // 数量
   void  *pdata;    // 数据缓冲区指针
} TS7DataItem, *PS7DataItem;

//typedef int TS7ResultItems[MaxVars];
//typedef TS7ResultItems *PS7ResultItems;

// 块列表结构体
typedef struct {
   int OBCount;   // 组织块数量
   int FBCount;   // 功能块数量
   int FCCount;   // 功能数量
   int SFBCount;  // 系统功能块数量
   int SFCCount;  // 系统功能数量
   int DBCount;   // 数据块数量
   int SDBCount;  // 系统数据块数量
} TS7BlocksList, *PS7BlocksList;

// 块信息结构体
typedef struct {
   int BlkType;    // 块类型(OB/DB等)
   int BlkNumber;  // 块编号
   int BlkLang;    // 块语言
   int BlkFlags;   // 块标志
   int MC7Size;    // 实际大小(字节)
   int LoadSize;   // 加载内存大小
   int LocalData;  // 本地数据
   int SBBLength;  // SBB 长度
   int CheckSum;   // 校验和
   int Version;    // 块版本
   // 字符信息
   char CodeDate[11]; // 代码日期
   char IntfDate[11]; // 接口日期
   char Author[9];    // 作者
   char Family[9];    // 族
   char Header[9];    // 标题
} TS7BlockInfo, *PS7BlockInfo ;

// 指定类型的块编号数组
typedef word TS7BlocksOfType[0x2000];
typedef TS7BlocksOfType *PS7BlocksOfType;

// 订货号结构体
typedef struct {
   char Code[21];  // 订货号代码
   byte V1;        // 主版本
   byte V2;        // 次版本
   byte V3;        // 修订版本
} TS7OrderCode, *PS7OrderCode;

// CPU 信息结构体
typedef struct {
   char ModuleTypeName[33];  // 模块类型名称
   char SerialNumber[25];    // 序列号
   char ASName[25];          // AS 名称
   char Copyright[27];       // 版权信息
   char ModuleName[25];      // 模块名称
} TS7CpuInfo, *PS7CpuInfo;

// 通信处理器(CP)信息结构体
typedef struct {
   int MaxPduLengt;   // 最大 PDU 长度
   int MaxConnections; // 最大连接数
   int MaxMpiRate;    // 最大 MPI 速率
   int MaxBusRate;    // 最大总线速率
} TS7CpInfo, *PS7CpInfo;

// SZL 头结构体 (参见 SFC51 系统功能说明)
typedef struct {
   word LENTHDR;  // 数据长度
   word N_DR;     // 数据记录编号
} SZL_HEADER, *PSZL_HEADER;

// SZL 系统状态列表结构体
typedef struct {
   SZL_HEADER Header;           // SZL 头
   byte Data[0x4000-4];         // SZL 数据
} TS7SZL, *PS7SZL;

// SZL ID 列表结构体(可用 SZL ID 列表，项目已调整为大端序)
typedef struct {
   SZL_HEADER Header;           // SZL 头
   word List[0x2000-2];         // ID 列表
} TS7SZLList, *PS7SZLList;

// PLC 保护信息结构体 (参见 §33.19)
typedef struct {
   word  sch_schal;  // 保护级别
   word  sch_par;    // 参数
   word  sch_rel;    // 相关
   word  bart_sch;   // 保护开关位置
   word  anl_sch;    // 启动保护
} TS7Protection, *PS7Protection;

// 客户端异步完成回调函数类型
typedef void (S7API *pfn_CliCompletion) (void *usrPtr, int opCode, int opResult);
//------------------------------------------------------------------------------
//  Import prototypes
//------------------------------------------------------------------------------
// 创建客户端实例 | 传入: 无 | 返回: [S7Object] 客户端句柄
S7Object S7API Cli_Create();
// 销毁客户端实例 | 传入: [S7Object*] 客户端句柄指针 | 返回: void
void S7API Cli_Destroy(S7Object *Client);
// 连接到 PLC | 传入: [S7Object] 句柄, [const char*] PLC IP地址, [int] 机架号, [int] 槽号 | 返回: [int] 0=成功
int S7API Cli_ConnectTo(S7Object Client, const char *Address, int Rack, int Slot);
// 设置连接参数(TSAP模式) | 传入: [S7Object] 句柄, [const char*] PLC IP, [word] 本地TSAP, [word] 远程TSAP | 返回: [int] 0=成功
int S7API Cli_SetConnectionParams(S7Object Client, const char *Address, word LocalTSAP, word RemoteTSAP);
// 设置连接类型 | 传入: [S7Object] 句柄, [word] 连接类型(CONNTYPE_PG=0x01/OP=0x02/BASIC=0x03) | 返回: [int] 0=成功
int S7API Cli_SetConnectionType(S7Object Client, word ConnectionType);
// 使用已设参数建立连接 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_Connect(S7Object Client);
// 断开连接 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_Disconnect(S7Object Client);
// 获取参数值 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 输出缓冲区 | 返回: [int] 0=成功
int S7API Cli_GetParam(S7Object Client, int ParamNumber, void *pValue);
// 设置参数值 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 参数值指针 | 返回: [int] 0=成功
int S7API Cli_SetParam(S7Object Client, int ParamNumber, void *pValue);
// 设置异步操作完成回调 | 传入: [S7Object] 句柄, [pfn_CliCompletion] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Cli_SetAsCallback(S7Object Client, pfn_CliCompletion pCompletion, void *usrPtr);
// Data I/O main functions
// 读取PLC指定区域数据 | 传入: [S7Object] 句柄, [int] 区域ID(PE=0x81/PA=0x82/MK=0x83/DB=0x84/CT=0x1C/TM=0x1D), [int] DB编号, [int] 起始地址, [int] 数量, [int] 字长度(Bit=0x01/Byte=0x02/Word=0x04/DWord=0x06/Real=0x08), [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_ReadArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);
// 向PLC指定区域写入数据 | 传入: 同Cli_ReadArea, [void*] 待写入数据缓冲区 | 返回: [int] 0=成功
int S7API Cli_WriteArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);
// 批量读取多个变量(最多20项) | 传入: [S7Object] 句柄, [PS7DataItem] TS7DataItem数组指针, [int] 项数 | 返回: [int] 0=成功
int S7API Cli_ReadMultiVars(S7Object Client, PS7DataItem Item, int ItemsCount);
// 批量写入多个变量(最多20项) | 传入: [S7Object] 句柄, [PS7DataItem] TS7DataItem数组指针, [int] 项数 | 返回: [int] 0=成功
int S7API Cli_WriteMultiVars(S7Object Client, PS7DataItem Item, int ItemsCount);
// Data I/O Lean functions
// 读取DB数据 | 传入: [S7Object] 句柄, [int] DB编号, [int] 起始字节, [int] 字节数, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_DBRead(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData);
// 写入DB数据 | 传入: [S7Object] 句柄, [int] DB编号, [int] 起始字节, [int] 字节数, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_DBWrite(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData);
// 读取标志位(M区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_MBRead(S7Object Client, int Start, int Size, void *pUsrData);
// 写入标志位(M区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_MBWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 读取过程输入(E/I区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_EBRead(S7Object Client, int Start, int Size, void *pUsrData);
// 写入过程输入(E/I区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_EBWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 读取过程输出(A/Q区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_ABRead(S7Object Client, int Start, int Size, void *pUsrData);
// 写入过程输出(A/Q区) | 传入: [S7Object] 句柄, [int] 起始地址, [int] 字节数, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_ABWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 读取定时器 | 传入: [S7Object] 句柄, [int] 起始编号, [int] 数量, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_TMRead(S7Object Client, int Start, int Amount, void *pUsrData);
// 写入定时器 | 传入: [S7Object] 句柄, [int] 起始编号, [int] 数量, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_TMWrite(S7Object Client, int Start, int Amount, void *pUsrData);
// 读取计数器 | 传入: [S7Object] 句柄, [int] 起始编号, [int] 数量, [void*] 接收缓冲区 | 返回: [int] 0=成功
int S7API Cli_CTRead(S7Object Client, int Start, int Amount, void *pUsrData);
// 写入计数器 | 传入: [S7Object] 句柄, [int] 起始编号, [int] 数量, [void*] 待写入数据 | 返回: [int] 0=成功
int S7API Cli_CTWrite(S7Object Client, int Start, int Amount, void *pUsrData);
// Directory functions
// 列出PLC中所有块的数量 | 传入: [S7Object] 句柄, [TS7BlocksList*] 输出结构体 | 返回: [int] 0=成功
int S7API Cli_ListBlocks(S7Object Client, TS7BlocksList *pUsrData);
// 获取AG块信息 | 传入: [S7Object] 句柄, [int] 块类型(OB=0x38/DB=0x41/FC=0x43等), [int] 块编号, [TS7BlockInfo*] 输出块信息 | 返回: [int] 0=成功
int S7API Cli_GetAgBlockInfo(S7Object Client, int BlockType, int BlockNum, TS7BlockInfo *pUsrData);
// 获取PG块信息 | 传入: [S7Object] 句柄, [void*] 块数据, [TS7BlockInfo*] 输出信息, [int] 数据大小 | 返回: [int] 0=成功
int S7API Cli_GetPgBlockInfo(S7Object Client, void *pBlock, TS7BlockInfo *pUsrData, int Size);
// 列出指定类型的所有块编号 | 传入: [S7Object] 句柄, [int] 块类型, [TS7BlocksOfType*] 输出数组, [int*] 输入/输出数量 | 返回: [int] 0=成功
int S7API Cli_ListBlocksOfType(S7Object Client, int BlockType, TS7BlocksOfType *pUsrData, int *ItemsCount);
// Blocks functions
// 上传块(不含头信息) | 传入: [S7Object] 句柄, [int] 块类型, [int] 块编号, [void*] 接收缓冲区, [int*] 输入缓冲区大小/输出实际大小 | 返回: [int] 0=成功
int S7API Cli_Upload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size);
// 完整上传块(含头信息) | 传入: 同Cli_Upload | 返回: [int] 0=成功
int S7API Cli_FullUpload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size);
// 下载块到PLC | 传入: [S7Object] 句柄, [int] 块编号, [void*] 块数据, [int] 数据大小 | 返回: [int] 0=成功
int S7API Cli_Download(S7Object Client, int BlockNum, void *pUsrData, int Size);
// 删除指定块 | 传入: [S7Object] 句柄, [int] 块类型, [int] 块编号 | 返回: [int] 0=成功
int S7API Cli_Delete(S7Object Client, int BlockType, int BlockNum);
// 获取整个DB内容 | 传入: [S7Object] 句柄, [int] DB编号, [void*] 接收缓冲区, [int*] 返回实际大小 | 返回: [int] 0=成功
int S7API Cli_DBGet(S7Object Client, int DBNumber, void *pUsrData, int *Size);
// 用指定值填充DB | 传入: [S7Object] 句柄, [int] DB编号, [int] 填充字节值 | 返回: [int] 0=成功
int S7API Cli_DBFill(S7Object Client, int DBNumber, int FillChar);
// Date/Time functions
// 获取PLC日期时间 | 传入: [S7Object] 句柄, [tm*] 输出时间结构体 | 返回: [int] 0=成功
int S7API Cli_GetPlcDateTime(S7Object Client, tm *DateTime);
// 设置PLC日期时间 | 传入: [S7Object] 句柄, [tm*] 要设置的时间 | 返回: [int] 0=成功
int S7API Cli_SetPlcDateTime(S7Object Client, tm *DateTime);
// 将PLC时间设为系统时间 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_SetPlcSystemDateTime(S7Object Client);
// System Info functions
// 获取PLC订货号 | 传入: [S7Object] 句柄, [TS7OrderCode*] 输出订货号信息 | 返回: [int] 0=成功
int S7API Cli_GetOrderCode(S7Object Client, TS7OrderCode *pUsrData);
// 获取CPU信息 | 传入: [S7Object] 句柄, [TS7CpuInfo*] 输出CPU信息 | 返回: [int] 0=成功
int S7API Cli_GetCpuInfo(S7Object Client, TS7CpuInfo *pUsrData);
// 获取通信处理器(CP)信息 | 传入: [S7Object] 句柄, [TS7CpInfo*] 输出CP信息 | 返回: [int] 0=成功
int S7API Cli_GetCpInfo(S7Object Client, TS7CpInfo *pUsrData);
// 读取系统状态列表(SZL) | 传入: [S7Object] 句柄, [int] SZL ID, [int] SZL Index, [TS7SZL*] 接收缓冲区, [int*] 返回数据大小 | 返回: [int] 0=成功
int S7API Cli_ReadSZL(S7Object Client, int ID, int Index, TS7SZL *pUsrData, int *Size);
// 读取可用SZL ID列表 | 传入: [S7Object] 句柄, [TS7SZLList*] 输出列表, [int*] 返回条目数 | 返回: [int] 0=成功
int S7API Cli_ReadSZLList(S7Object Client, TS7SZLList *pUsrData, int *ItemsCount);
// Control functions
// PLC热启动 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_PlcHotStart(S7Object Client);
// PLC冷启动 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_PlcColdStart(S7Object Client);
// PLC停止 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_PlcStop(S7Object Client);
// 将RAM复制到ROM | 传入: [S7Object] 句柄, [int] 超时时间(ms) | 返回: [int] 0=成功
int S7API Cli_CopyRamToRom(S7Object Client, int Timeout);
// 压缩PLC内存 | 传入: [S7Object] 句柄, [int] 超时时间(ms) | 返回: [int] 0=成功
int S7API Cli_Compress(S7Object Client, int Timeout);
// 获取PLC状态 | 传入: [S7Object] 句柄, [int*] 输出状态值(Run=0x08/Stop=0x04/Unknown=0x00) | 返回: [int] 0=成功
int S7API Cli_GetPlcStatus(S7Object Client, int *Status);
// Security functions
// 获取PLC保护级别 | 传入: [S7Object] 句柄, [TS7Protection*] 输出保护信息 | 返回: [int] 0=成功
int S7API Cli_GetProtection(S7Object Client, TS7Protection *pUsrData);
// 设置会话密码 | 传入: [S7Object] 句柄, [char*] 密码字符串 | 返回: [int] 0=成功
int S7API Cli_SetSessionPassword(S7Object Client, char *Password);
// 清除会话密码 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Cli_ClearSessionPassword(S7Object Client);
// Low level
// ISO层缓冲区交换 | 传入: [S7Object] 句柄, [void*] 发送/接收缓冲区, [int*] 输入发送大小/输出接收大小 | 返回: [int] 0=成功
int S7API Cli_IsoExchangeBuffer(S7Object Client, void *pUsrData, int *Size);
// Misc
// 获取最后一次执行时间 | 传入: [S7Object] 句柄, [int*] 输出执行时间(ms) | 返回: [int] 0=成功
int S7API Cli_GetExecTime(S7Object Client, int *Time);
// 获取最后一次错误码 | 传入: [S7Object] 句柄, [int*] 输出最后错误码 | 返回: [int] 0=成功
int S7API Cli_GetLastError(S7Object Client, int *LastError);
// 获取PDU长度信息 | 传入: [S7Object] 句柄, [int*] 输出请求的PDU长度, [int*] 输出协商后的PDU长度 | 返回: [int] 0=成功
int S7API Cli_GetPduLength(S7Object Client, int *Requested, int *Negotiated);
// 错误码转可读文本 | 传入: [int] 错误码, [char*] 输出文本缓冲区, [int] 缓冲区长度 | 返回: [int] 0=成功
int S7API Cli_ErrorText(int Error, char *Text, int TextLen);
// 1.1.0
// 查询连接状态 | 传入: [S7Object] 句柄, [int*] 输出连接状态(1=已连接/0=未连接) | 返回: [int] 0=成功
int S7API Cli_GetConnected(S7Object Client, int *Connected);
//------------------------------------------------------------------------------
//  Async functions (异步函数，参数与同步版本一致，立即返回不等待PLC响应)
//------------------------------------------------------------------------------
// 异步读取区域 | 传入: 同Cli_ReadArea | 返回: [int] 0=已提交
int S7API Cli_AsReadArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);
// 异步写入区域 | 传入: 同Cli_WriteArea | 返回: [int] 0=已提交
int S7API Cli_AsWriteArea(S7Object Client, int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);
// 异步读DB | 传入: 同Cli_DBRead | 返回: [int] 0=已提交
int S7API Cli_AsDBRead(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData);
// 异步写DB | 传入: 同Cli_DBWrite | 返回: [int] 0=已提交
int S7API Cli_AsDBWrite(S7Object Client, int DBNumber, int Start, int Size, void *pUsrData);
// 异步读标志位(M区) | 传入: 同Cli_MBRead | 返回: [int] 0=已提交
int S7API Cli_AsMBRead(S7Object Client, int Start, int Size, void *pUsrData);
// 异步写标志位(M区) | 传入: 同Cli_MBWrite | 返回: [int] 0=已提交
int S7API Cli_AsMBWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 异步读过程输入(E/I区) | 传入: 同Cli_EBRead | 返回: [int] 0=已提交
int S7API Cli_AsEBRead(S7Object Client, int Start, int Size, void *pUsrData);
// 异步写过程输入(E/I区) | 传入: 同Cli_EBWrite | 返回: [int] 0=已提交
int S7API Cli_AsEBWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 异步读过程输出(A/Q区) | 传入: 同Cli_ABRead | 返回: [int] 0=已提交
int S7API Cli_AsABRead(S7Object Client, int Start, int Size, void *pUsrData);
// 异步写过程输出(A/Q区) | 传入: 同Cli_ABWrite | 返回: [int] 0=已提交
int S7API Cli_AsABWrite(S7Object Client, int Start, int Size, void *pUsrData);
// 异步读定时器 | 传入: 同Cli_TMRead | 返回: [int] 0=已提交
int S7API Cli_AsTMRead(S7Object Client, int Start, int Amount, void *pUsrData);
// 异步写定时器 | 传入: 同Cli_TMWrite | 返回: [int] 0=已提交
int S7API Cli_AsTMWrite(S7Object Client, int Start, int Amount, void *pUsrData);
// 异步读计数器 | 传入: 同Cli_CTRead | 返回: [int] 0=已提交
int S7API Cli_AsCTRead(S7Object Client, int Start, int Amount, void *pUsrData);
// 异步写计数器 | 传入: 同Cli_CTWrite | 返回: [int] 0=已提交
int S7API Cli_AsCTWrite(S7Object Client, int Start, int Amount, void *pUsrData);
// 异步列出指定类型的所有块 | 传入: 同Cli_ListBlocksOfType | 返回: [int] 0=已提交
int S7API Cli_AsListBlocksOfType(S7Object Client, int BlockType, TS7BlocksOfType *pUsrData, int *ItemsCount);
// 异步读取SZL | 传入: 同Cli_ReadSZL | 返回: [int] 0=已提交
int S7API Cli_AsReadSZL(S7Object Client, int ID, int Index, TS7SZL *pUsrData, int *Size);
// 异步读取SZL列表 | 传入: 同Cli_ReadSZLList | 返回: [int] 0=已提交
int S7API Cli_AsReadSZLList(S7Object Client, TS7SZLList *pUsrData, int *ItemsCount);
// 异步上传块 | 传入: 同Cli_Upload | 返回: [int] 0=已提交
int S7API Cli_AsUpload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size);
// 异步完整上传块 | 传入: 同Cli_FullUpload | 返回: [int] 0=已提交
int S7API Cli_AsFullUpload(S7Object Client, int BlockType, int BlockNum, void *pUsrData, int *Size);
// 异步下载块 | 传入: 同Cli_Download | 返回: [int] 0=已提交
int S7API Cli_AsDownload(S7Object Client, int BlockNum, void *pUsrData, int Size);
// 异步RAM复制到ROM | 传入: [S7Object] 句柄, [int] 超时(ms) | 返回: [int] 0=已提交
int S7API Cli_AsCopyRamToRom(S7Object Client, int Timeout);
// 异步压缩PLC内存 | 传入: [S7Object] 句柄, [int] 超时(ms) | 返回: [int] 0=已提交
int S7API Cli_AsCompress(S7Object Client, int Timeout);
// 异步获取整个DB | 传入: 同Cli_DBGet | 返回: [int] 0=已提交
int S7API Cli_AsDBGet(S7Object Client, int DBNumber, void *pUsrData, int *Size);
// 异步填充DB | 传入: 同Cli_DBFill | 返回: [int] 0=已提交
int S7API Cli_AsDBFill(S7Object Client, int DBNumber, int FillChar);
// 检查异步操作是否完成 | 传入: [S7Object] 句柄, [int*] 输出操作结果 | 返回: [int] JobComplete=0 / JobPending=1
int S7API Cli_CheckAsCompletion(S7Object Client, int *opResult);
// 等待异步操作完成 | 传入: [S7Object] 句柄, [int] 等待超时(ms) | 返回: [int] 0=成功
int S7API Cli_WaitAsCompletion(S7Object Client, int Timeout);

//******************************************************************************
//                                   SERVER
//******************************************************************************
// 操作类型(用于读写回调)
const int OperationRead  = 0;   // 读操作
const int OperationWrite = 1;   // 写操作

// 掩码类型
const int mkEvent = 0;   // 事件掩码
const int mkLog   = 1;   // 日志掩码

// 服务端区域 ID (用于 Register/Unregister/Lock/Unlock Area)
const int srvAreaPE = 0;   // 过程输入(I)
const int srvAreaPA = 1;   // 过程输出(Q)
const int srvAreaMK = 2;   // 标志位(M)
const int srvAreaCT = 3;   // 计数器(C)
const int srvAreaTM = 4;   // 定时器(T)
const int srvAreaDB = 5;   // 数据块(DB)

// 服务端错误码
const longword errSrvCannotStart        = 0x00100000; // 服务端无法启动
const longword errSrvDBNullPointer      = 0x00200000; // 传入了空指针作为 PData
const longword errSrvAreaAlreadyExists  = 0x00300000; // 区域重复注册
const longword errSrvUnknownArea        = 0x00400000; // 未知区域
const longword errSrvInvalidParams      = 0x00500000; // 提供的参数无效
const longword errSrvTooManyDB          = 0x00600000; // 无法注册 DB
const longword errSrvInvalidParamNumber = 0x00700000; // 参数编号无效(srv_get/set_param)
const longword errSrvCannotChangeParam  = 0x00800000; // 运行中无法更改参数

// TCP 服务端事件码
const longword evcServerStarted       = 0x00000001; // 服务端已启动
const longword evcServerStopped       = 0x00000002; // 服务端已停止
const longword evcListenerCannotStart = 0x00000004; // 监听器无法启动
const longword evcClientAdded         = 0x00000008; // 客户端已添加
const longword evcClientRejected      = 0x00000010; // 客户端被拒绝
const longword evcClientNoRoom        = 0x00000020; // 客户端无空间
const longword evcClientException     = 0x00000040; // 客户端异常
const longword evcClientDisconnected  = 0x00000080; // 客户端已断开
const longword evcClientTerminated    = 0x00000100; // 客户端已终止
const longword evcClientsDropped      = 0x00000200; // 客户端已丢弃
const longword evcReserved_00000400   = 0x00000400; // 未使用
const longword evcReserved_00000800   = 0x00000800; // 未使用
const longword evcReserved_00001000   = 0x00001000; // 未使用
const longword evcReserved_00002000   = 0x00002000; // 未使用
const longword evcReserved_00004000   = 0x00004000; // 未使用
const longword evcReserved_00008000   = 0x00008000; // 未使用
// S7 服务端事件码
const longword evcPDUincoming  	      = 0x00010000; // PDU 进入
const longword evcDataRead            = 0x00020000; // 数据读取
const longword evcDataWrite    	      = 0x00040000; // 数据写入
const longword evcNegotiatePDU        = 0x00080000; // PDU 协商
const longword evcReadSZL             = 0x00100000; // 读取 SZL
const longword evcClock               = 0x00200000; // 时钟操作
const longword evcUpload              = 0x00400000; // 上传
const longword evcDownload            = 0x00800000; // 下载
const longword evcDirectory           = 0x01000000; // 目录操作
const longword evcSecurity            = 0x02000000; // 安全操作
const longword evcControl             = 0x04000000; // 控制操作
const longword evcReserved_08000000   = 0x08000000; // 未使用
const longword evcReserved_10000000   = 0x10000000; // 未使用
const longword evcReserved_20000000   = 0x20000000; // 未使用
const longword evcReserved_40000000   = 0x40000000; // 未使用
const longword evcReserved_80000000   = 0x80000000; // 未使用
// 事件掩码开关
const longword evcAll                 = 0xFFFFFFFF; // 启用所有事件
const longword evcNone                = 0x00000000; // 禁用所有事件
// 事件子代码
const word evsUnknown                 = 0x0000; // 未知
const word evsStartUpload             = 0x0001; // 开始上传
const word evsStartDownload           = 0x0001; // 开始下载
const word evsGetBlockList            = 0x0001; // 获取块列表
const word evsStartListBoT            = 0x0002; // 开始列出块类型
const word evsListBoT                 = 0x0003; // 列出块类型
const word evsGetBlockInfo            = 0x0004; // 获取块信息
const word evsGetClock                = 0x0001; // 获取时钟
const word evsSetClock                = 0x0002; // 设置时钟
const word evsSetPassword             = 0x0001; // 设置密码
const word evsClrPassword             = 0x0002; // 清除密码
// 事件参数: 功能组
const word grProgrammer               = 0x0041; // 编程器
const word grCyclicData               = 0x0042; // 循环数据
const word grBlocksInfo               = 0x0043; // 块信息
const word grSZL                      = 0x0044; // SZL
const word grPassword                 = 0x0045; // 密码
const word grBSend                    = 0x0046; // BSend
const word grClock                    = 0x0047; // 时钟
const word grSecurity                 = 0x0045; // 安全
// 事件参数: 控制代码
const word CodeControlUnknown         = 0x0000; // 未知控制
const word CodeControlColdStart       = 0x0001; // 冷启动
const word CodeControlWarmStart       = 0x0002; // 热启动
const word CodeControlStop            = 0x0003; // 停止
const word CodeControlCompress        = 0x0004; // 压缩
const word CodeControlCpyRamRom       = 0x0005; // 复制 RAM 到 ROM
const word CodeControlInsDel          = 0x0006; // 插入/删除
// 事件结果
const word evrNoError                 = 0x0000; // 无错误
const word evrFragmentRejected        = 0x0001; // 分片被拒绝
const word evrMalformedPDU            = 0x0002; // PDU 格式错误
const word evrSparseBytes             = 0x0003; // 稀疏字节
const word evrCannotHandlePDU         = 0x0004; // 无法处理 PDU
const word evrNotImplemented          = 0x0005; // 未实现
const word evrErrException            = 0x0006; // 异常错误
const word evrErrAreaNotFound         = 0x0007; // 区域未找到
const word evrErrOutOfRange           = 0x0008; // 超出范围
const word evrErrOverPDU              = 0x0009; // 超过 PDU
const word evrErrTransportSize        = 0x000A; // 传输大小错误
const word evrInvalidGroupUData       = 0x000B; // 用户数据组无效
const word evrInvalidSZL              = 0x000C; // SZL 无效
const word evrDataSizeMismatch        = 0x000D; // 数据大小不匹配
const word evrCannotUpload            = 0x000E; // 无法上传
const word evrCannotDownload          = 0x000F; // 无法下载
const word evrUploadInvalidID         = 0x0010; // 上传 ID 无效
const word evrResNotFound             = 0x0011; // 资源未找到

// 服务端事件结构体
typedef struct{
	time_t EvtTime;    // 时间戳
	int EvtSender;     // 发送者
	longword EvtCode;  // 事件代码
	word EvtRetCode;   // 事件结果
	word EvtParam1;    // 参数 1(如可用)
	word EvtParam2;    // 参数 2(如可用)
	word EvtParam3;    // 参数 3(如可用)
	word EvtParam4;    // 参数 4(如可用)
}TSrvEvent, *PSrvEvent;

// 服务端事件回调函数类型
typedef void (S7API *pfn_SrvCallBack)(void *usrPtr, PSrvEvent PEvent, int Size);
// 服务端读写回调函数类型
typedef int(S7API *pfn_RWAreaCallBack)(void *usrPtr, int Sender, int Operation, PS7Tag PTag, void *pUsrData);

// 创建服务端实例 | 传入: 无 | 返回: [S7Object] 服务端句柄
S7Object S7API Srv_Create();
// 销毁服务端实例 | 传入: [S7Object*] 服务端句柄指针 | 返回: void
void S7API Srv_Destroy(S7Object *Server);
// 获取服务端参数 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 输出缓冲区 | 返回: [int] 0=成功
int S7API Srv_GetParam(S7Object Server, int ParamNumber, void *pValue);
// 设置服务端参数 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 参数值指针 | 返回: [int] 0=成功
int S7API Srv_SetParam(S7Object Server, int ParamNumber, void *pValue);
// 绑定指定地址启动服务端 | 传入: [S7Object] 句柄, [const char*] 绑定IP地址 | 返回: [int] 0=成功
int S7API Srv_StartTo(S7Object Server, const char *Address);
// 启动服务端(监听所有接口) | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Srv_Start(S7Object Server);
// 停止服务端 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Srv_Stop(S7Object Server);
// 注册共享区域 | 传入: [S7Object] 句柄, [int] 区域代码(PE=0/PA=1/MK=2/CT=3/TM=4/DB=5), [word] 区域索引(DB编号), [void*] 数据缓冲区指针, [int] 缓冲区大小(字节) | 返回: [int] 0=成功
int S7API Srv_RegisterArea(S7Object Server, int AreaCode, word Index, void *pUsrData, int Size);
// 注销已注册的区域 | 传入: [S7Object] 句柄, [int] 区域代码, [word] 区域索引 | 返回: [int] 0=成功
int S7API Srv_UnregisterArea(S7Object Server, int AreaCode, word Index);
// 锁定区域(客户端不可访问) | 传入: [S7Object] 句柄, [int] 区域代码, [word] 区域索引 | 返回: [int] 0=成功
int S7API Srv_LockArea(S7Object Server, int AreaCode, word Index);
// 解锁区域 | 传入: [S7Object] 句柄, [int] 区域代码, [word] 区域索引 | 返回: [int] 0=成功
int S7API Srv_UnlockArea(S7Object Server, int AreaCode, word Index);
// 获取服务端运行状态 | 传入: [S7Object] 句柄, [int*] 输出服务端状态, [int*] 输出CPU状态, [int*] 输出客户端数 | 返回: [int] 0=成功
int S7API Srv_GetStatus(S7Object Server, int *ServerStatus, int *CpuStatus, int *ClientsCount);
// 设置模拟的CPU状态 | 传入: [S7Object] 句柄, [int] CPU状态值 | 返回: [int] 0=成功
int S7API Srv_SetCpuStatus(S7Object Server, int CpuStatus);
// 清除事件队列 | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Srv_ClearEvents(S7Object Server);
// 从队列取出一个事件 | 传入: [S7Object] 句柄, [TSrvEvent*] 输出事件结构体, [int*] 输出是否有事件(1=有/0=无) | 返回: [int] 0=成功
int S7API Srv_PickEvent(S7Object Server, TSrvEvent *pEvent, int *EvtReady);
// 获取事件/日志掩码 | 传入: [S7Object] 句柄, [int] 掩码类型(mkEvent=0/mkLog=1), [longword*] 输出掩码值 | 返回: [int] 0=成功
int S7API Srv_GetMask(S7Object Server, int MaskKind, longword *Mask);
// 设置事件/日志掩码 | 传入: [S7Object] 句柄, [int] 掩码类型, [longword] 掩码值 | 返回: [int] 0=成功
int S7API Srv_SetMask(S7Object Server, int MaskKind, longword Mask);
// 设置事件回调 | 传入: [S7Object] 句柄, [pfn_SrvCallBack] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Srv_SetEventsCallback(S7Object Server, pfn_SrvCallBack pCallback, void *usrPtr);
// 设置读取事件回调 | 传入: [S7Object] 句柄, [pfn_SrvCallBack] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Srv_SetReadEventsCallback(S7Object Server, pfn_SrvCallBack pCallback, void *usrPtr);
// 设置区域读写回调 | 传入: [S7Object] 句柄, [pfn_RWAreaCallBack] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Srv_SetRWAreaCallback(S7Object Server, pfn_RWAreaCallBack pCallback, void *usrPtr);
// 事件转文本 | 传入: [TSrvEvent*] 事件结构体, [char*] 输出文本, [int] 缓冲区长度 | 返回: [int] 0=成功
int S7API Srv_EventText(TSrvEvent *Event, char *Text, int TextLen);
// 错误码转文本 | 传入: [int] 错误码, [char*] 输出文本, [int] 缓冲区长度 | 返回: [int] 0=成功
int S7API Srv_ErrorText(int Error, char *Text, int TextLen);

//******************************************************************************
//                                   PARTNER
//******************************************************************************

// Partner 状态
const int par_stopped         = 0;   // 已停止
const int par_connecting      = 1;   // 运行中并主动连接
const int par_waiting         = 2;   // 运行中并等待连接
const int par_linked          = 3;   // 运行中并已连接
const int par_sending         = 4;   // 发送数据中
const int par_receiving       = 5;   // 接收数据中
const int par_binderror       = 6;   // 被动服务端启动错误

// Partner 错误码
const longword errParAddressInUse       = 0x00200000; // 地址已被使用
const longword errParNoRoom             = 0x00300000; // 无空间
const longword errServerNoRoom          = 0x00400000; // 服务端无空间
const longword errParInvalidParams      = 0x00500000; // 参数无效
const longword errParNotLinked          = 0x00600000; // 未连接
const longword errParBusy               = 0x00700000; // 忙
const longword errParFrameTimeout       = 0x00800000; // 帧超时
const longword errParInvalidPDU         = 0x00900000; // PDU 无效
const longword errParSendTimeout        = 0x00A00000; // 发送超时
const longword errParRecvTimeout        = 0x00B00000; // 接收超时
const longword errParSendRefused        = 0x00C00000; // 发送被拒绝
const longword errParNegotiatingPDU     = 0x00D00000; // PDU 协商中
const longword errParSendingBlock       = 0x00E00000; // 发送块中
const longword errParRecvingBlock       = 0x00F00000; // 接收块中
const longword errParBindError          = 0x01000000; // 绑定错误
const longword errParDestroying         = 0x01100000; // 正在销毁
const longword errParInvalidParamNumber = 0x01200000; // 参数编号无效(par_get/set_param)
const longword errParCannotChangeParam  = 0x01300000; // 运行中无法更改参数
const longword errParBufferTooSmall     = 0x01400000; // 缓冲区太小(LabVIEW 包装器引发)

// BRecv 数据接收回调函数类型
typedef void (S7API *pfn_ParRecvCallBack)(void * usrPtr, int opResult, longword R_ID, void *pData, int Size);
// BSend 发送完成回调函数类型
typedef void (S7API *pfn_ParSendCompletion)(void * usrPtr, int opResult);

// 创建Partner实例 | 传入: [int] 1=主动模式/0=被动模式 | 返回: [S7Object] Partner句柄
S7Object S7API Par_Create(int Active);
// 销毁Partner实例 | 传入: [S7Object*] Partner句柄指针 | 返回: void
void S7API Par_Destroy(S7Object *Partner);
// 获取Partner参数 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 输出缓冲区 | 返回: [int] 0=成功
int S7API Par_GetParam(S7Object Partner, int ParamNumber, void *pValue);
// 设置Partner参数 | 传入: [S7Object] 句柄, [int] 参数编号, [void*] 参数值指针 | 返回: [int] 0=成功
int S7API Par_SetParam(S7Object Partner, int ParamNumber, void *pValue);
// 指定地址启动Partner | 传入: [S7Object] 句柄, [const char*] 本地IP, [const char*] 远程IP, [word] 本地TSAP, [word] 远程TSAP | 返回: [int] 0=成功
int S7API Par_StartTo(S7Object Partner, const char *LocalAddress, const char *RemoteAddress,
    word LocTsap, word RemTsap);
// 启动Partner | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Par_Start(S7Object Partner);
// 停止Partner | 传入: [S7Object] 句柄 | 返回: [int] 0=成功
int S7API Par_Stop(S7Object Partner);
// BSend
// 同步发送数据 | 传入: [S7Object] 句柄, [longword] 远程ID(R_ID), [void*] 发送数据缓冲区, [int] 数据大小(字节) | 返回: [int] 0=成功
int S7API Par_BSend(S7Object Partner, longword R_ID, void *pUsrData, int Size);
// 异步发送数据 | 传入: 同Par_BSend | 返回: [int] 0=已提交
int S7API Par_AsBSend(S7Object Partner, longword R_ID, void *pUsrData, int Size);
// 检查异步发送是否完成 | 传入: [S7Object] 句柄, [int*] 输出操作结果 | 返回: [int] 0=成功
int S7API Par_CheckAsBSendCompletion(S7Object Partner, int *opResult);
// 等待异步发送完成 | 传入: [S7Object] 句柄, [longword] 超时(ms) | 返回: [int] 0=成功
int S7API Par_WaitAsBSendCompletion(S7Object Partner, longword Timeout);
// 设置发送完成回调 | 传入: [S7Object] 句柄, [pfn_ParSendCompletion] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Par_SetSendCallback(S7Object Partner, pfn_ParSendCompletion pCompletion, void *usrPtr);
// BRecv
// 同步接收数据(阻塞) | 传入: [S7Object] 句柄, [longword*] 输出远程ID, [void*] 接收缓冲区, [int*] 输出数据大小, [longword] 超时(ms) | 返回: [int] 0=成功
int S7API Par_BRecv(S7Object Partner, longword *R_ID, void *pData, int *Size, longword Timeout);
// 检查异步接收是否完成 | 传入: [S7Object] 句柄, [int*] 输出结果, [longword*] 输出远程ID, [void*] 接收缓冲区, [int*] 输出大小 | 返回: [int] 0=成功
int S7API Par_CheckAsBRecvCompletion(S7Object Partner, int *opResult, longword *R_ID,
	void *pData, int *Size);
// 设置数据接收回调 | 传入: [S7Object] 句柄, [pfn_ParRecvCallBack] 回调函数, [void*] 用户指针 | 返回: [int] 0=成功
int S7API Par_SetRecvCallback(S7Object Partner, pfn_ParRecvCallBack pCompletion, void *usrPtr);
// Stat
// 获取发送/接收耗时 | 传入: [S7Object] 句柄, [longword*] 输出发送时间(ms), [longword*] 输出接收时间(ms) | 返回: [int] 0=成功
int S7API Par_GetTimes(S7Object Partner, longword *SendTime, longword *RecvTime);
// 获取传输统计 | 传入: [S7Object] 句柄, [longword*] 输出已发送字节, [longword*] 输出已接收字节, [longword*] 输出发送错误数, [longword*] 输出接收错误数 | 返回: [int] 0=成功
int S7API Par_GetStats(S7Object Partner, longword *BytesSent, longword *BytesRecv,
	longword *SendErrors, longword *RecvErrors);
// 获取最后错误 | 传入: [S7Object] 句柄, [int*] 输出最后错误码 | 返回: [int] 0=成功
int S7API Par_GetLastError(S7Object Partner, int *LastError);
// 获取Partner状态 | 传入: [S7Object] 句柄, [int*] 输出状态值(stopped=0/connecting=1/waiting=2/linked=3/sending=4/receiving=5/binderror=6) | 返回: [int] 0=成功
int S7API Par_GetStatus(S7Object Partner, int *Status);
// 错误码转文本 | 传入: [int] 错误码, [char*] 输出文本, [int] 缓冲区长度 | 返回: [int] 0=成功
int S7API Par_ErrorText(int Error, char *Text, int TextLen);


#pragma pack()
#ifdef __cplusplus
 }
#endif // __cplusplus

#ifdef __cplusplus

//******************************************************************************
//                           TS7Client C++ 客户端类定义
//******************************************************************************
class TS7Client
{
private:
    S7Object Client;  // 客户端句柄
public:
	TS7Client();   // 构造函数
	~TS7Client();  // 析构函数
    // 控制函数
    int Connect();                                                    // 使用已设参数建立连接
    int ConnectTo(const char *RemAddress, int Rack, int Slot);        // 连接到指定 PLC
    int SetConnectionParams(const char *RemAddress, word LocalTSAP, word RemoteTSAP); // 设置连接参数(TSAP模式)
    int SetConnectionType(word ConnectionType);                       // 设置连接类型
    int Disconnect();                                                 // 断开连接
    int GetParam(int ParamNumber, void *pValue);                      // 获取参数值
    int SetParam(int ParamNumber, void *pValue);                      // 设置参数值
    // 数据读写主函数
    int ReadArea(int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);   // 读取PLC指定区域
    int WriteArea(int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);  // 写入PLC指定区域
    int ReadMultiVars(PS7DataItem Item, int ItemsCount);              // 批量读取多个变量(最多20项)
    int WriteMultiVars(PS7DataItem Item, int ItemsCount);             // 批量写入多个变量(最多20项)
    // 数据读写快捷函数
    int DBRead(int DBNumber, int Start, int Size, void *pUsrData);    // 读取DB数据
    int DBWrite(int DBNumber, int Start, int Size, void *pUsrData);   // 写入DB数据
    int MBRead(int Start, int Size, void *pUsrData);                  // 读取标志位(M区)
    int MBWrite(int Start, int Size, void *pUsrData);                 // 写入标志位(M区)
    int EBRead(int Start, int Size, void *pUsrData);                  // 读取过程输入(E/I区)
    int EBWrite(int Start, int Size, void *pUsrData);                 // 写入过程输入(E/I区)
    int ABRead(int Start, int Size, void *pUsrData);                  // 读取过程输出(A/Q区)
    int ABWrite(int Start, int Size, void *pUsrData);                 // 写入过程输出(A/Q区)
    int TMRead(int Start, int Amount, void *pUsrData);                // 读取定时器
    int TMWrite(int Start, int Amount, void *pUsrData);               // 写入定时器
    int CTRead(int Start, int Amount, void *pUsrData);                // 读取计数器
    int CTWrite(int Start, int Amount, void *pUsrData);               // 写入计数器
    // 目录函数
    int ListBlocks(PS7BlocksList pUsrData);                           // 列出所有块数量
    int GetAgBlockInfo(int BlockType, int BlockNum, PS7BlockInfo pUsrData); // 获取AG块信息
    int GetPgBlockInfo(void *pBlock, PS7BlockInfo pUsrData, int Size);// 获取PG块信息
    int ListBlocksOfType(int BlockType, TS7BlocksOfType *pUsrData, int *ItemsCount); // 列出指定类型所有块
    // 块操作函数
    int Upload(int BlockType, int BlockNum, void *pUsrData, int *Size);   // 上传块(不含头)
    int FullUpload(int BlockType, int BlockNum, void *pUsrData, int *Size);// 完整上传块(含头)
    int Download(int BlockNum, void *pUsrData, int Size);             // 下载块到PLC
    int Delete(int BlockType, int BlockNum); 出水液位画面                         // 删除指定块
    int DBGet(int DBNumber, void *pUsrData, int *Size);               // 获取整个DB内容
    int DBFill(int DBNumber, int FillChar);                           // 填充DB
    // 日期/时间函数
    int GetPlcDateTime(tm *DateTime);                                 // 获取PLC日期时间
    int SetPlcDateTime(tm *DateTime);                                 // 设置PLC日期时间
    int SetPlcSystemDateTime();                                       // 将PLC时间设为系统时间
    // 系统信息函数
    int GetOrderCode(PS7OrderCode pUsrData);                          // 获取PLC订货号
    int GetCpuInfo(PS7CpuInfo pUsrData);                              // 获取CPU信息
    int GetCpInfo(PS7CpInfo pUsrData);                                // 获取CP信息
	int ReadSZL(int ID, int Index, PS7SZL pUsrData, int *Size);       // 读取SZL
	int ReadSZLList(PS7SZLList pUsrData, int *ItemsCount);            // 读取SZL列表
	// PLC 控制函数
	int PlcHotStart();                                                // PLC热启动
	int PlcColdStart();                                               // PLC冷启动
	int PlcStop();                                                    // PLC停止
	int CopyRamToRom(int Timeout);                                    // RAM复制到ROM
	int Compress(int Timeout);                                        // 压缩PLC内存
	// 安全函数
	int GetProtection(PS7Protection pUsrData);                        // 获取保护级别
	int SetSessionPassword(char *Password);                           // 设置会话密码
	int ClearSessionPassword();                                       // 清除会话密码
	// 属性
	int ExecTime();                                                   // 获取最后执行时间(ms)
	int LastError();                                                  // 获取最后错误码
	int PDURequested();                                               // 获取请求的PDU长度
	int PDULength();                                                  // 获取协商后的PDU长度
	int PlcStatus();                                                  // 获取PLC状态
	bool Connected();                                                 // 查询是否已连接
	// 异步函数
	int SetAsCallback(pfn_CliCompletion pCompletion, void *usrPtr);   // 设置异步完成回调
	bool CheckAsCompletion(int *opResult);                            // 检查异步操作是否完成
	int WaitAsCompletion(longword Timeout);                           // 等待异步操作完成
	int AsReadArea(int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData);  // 异步读取区域
	int AsWriteArea(int Area, int DBNumber, int Start, int Amount, int WordLen, void *pUsrData); // 异步写入区域
	int AsListBlocksOfType(int BlockType, PS7BlocksOfType pUsrData, int *ItemsCount); // 异步列出块
	int AsReadSZL(int ID, int Index, PS7SZL pUsrData, int *Size);     // 异步读取SZL
	int AsReadSZLList(PS7SZLList pUsrData, int *ItemsCount);          // 异步读取SZL列表
	int AsUpload(int BlockType, int BlockNum, void *pUsrData, int *Size);  // 异步上传
	int AsFullUpload(int BlockType, int BlockNum, void *pUsrData, int *Size);// 异步完整上传
	int AsDownload(int BlockNum, void *pUsrData,  int Size);          // 异步下载
	int AsCopyRamToRom(int Timeout);                                  // 异步RAM到ROM
	int AsCompress(int Timeout);                                      // 异步压缩
	int AsDBRead(int DBNumber, int Start, int Size, void *pUsrData);  // 异步读DB
	int AsDBWrite(int DBNumber, int Start, int Size, void *pUsrData); // 异步写DB
	int AsMBRead(int Start, int Size, void *pUsrData);                // 异步读标志位
	int AsMBWrite(int Start, int Size, void *pUsrData);               // 异步写标志位
	int AsEBRead(int Start, int Size, void *pUsrData);                // 异步读过程输入
	int AsEBWrite(int Start, int Size, void *pUsrData);               // 异步写过程输入
	int AsABRead(int Start, int Size, void *pUsrData);                // 异步读过程输出
	int AsABWrite(int Start, int Size, void *pUsrData);               // 异步写过程输出
    int AsTMRead(int Start, int Amount, void *pUsrData);              // 异步读定时器
    int AsTMWrite(int Start, int Amount, void *pUsrData);             // 异步写定时器
    int AsCTRead(int Start, int Amount, void *pUsrData);              // 异步读计数器
	int AsCTWrite(int Start, int Amount, void *pUsrData);             // 异步写计数器
    int AsDBGet(int DBNumber, void *pUsrData, int *Size);             // 异步获取DB
	int AsDBFill(int DBNumber, int FillChar);                         // 异步填充DB
};
typedef TS7Client *PS7Client;
//******************************************************************************
//                           TS7Server C++ 服务端类定义
//******************************************************************************
class TS7Server
{
private:
    S7Object Server;  // 服务端句柄
public:
    TS7Server();   // 构造函数
    ~TS7Server();  // 析构函数
    // 控制
    int Start();                                   // 启动服务端(监听所有接口)
    int StartTo(const char *Address);              // 绑定指定地址启动
    int Stop();                                    // 停止服务端
    int GetParam(int ParamNumber, void *pValue);   // 获取参数
    int SetParam(int ParamNumber, void *pValue);   // 设置参数
    // 事件
    int SetEventsCallback(pfn_SrvCallBack PCallBack, void *UsrPtr);   // 设置事件回调
	int SetReadEventsCallback(pfn_SrvCallBack PCallBack, void *UsrPtr); // 设置读取事件回调
	int SetRWAreaCallback(pfn_RWAreaCallBack PCallBack, void *UsrPtr);  // 设置区域读写回调
    bool PickEvent(TSrvEvent *pEvent);             // 从队列取出一个事件
    void ClearEvents();                            // 清除事件队列
    longword GetEventsMask();                      // 获取事件掩码
    longword GetLogMask();                         // 获取日志掩码
    void SetEventsMask(longword Mask);             // 设置事件掩码
    void SetLogMask(longword Mask);                // 设置日志掩码
    // 资源
    int RegisterArea(int AreaCode, word Index, void *pUsrData, word Size); // 注册共享区域
    int UnregisterArea(int AreaCode, word Index);  // 注销区域
    int LockArea(int AreaCode, word Index);        // 锁定区域
    int UnlockArea(int AreaCode, word Index);      // 解锁区域
    // 属性
    int ServerStatus();                            // 获取服务端状态
    int GetCpuStatus();                            // 获取CPU状态
    int SetCpuStatus(int Status);                  // 设置CPU状态
	int ClientsCount();                            // 获取客户端数量
};
typedef TS7Server *PS7Server;

//******************************************************************************
//                          TS7Partner C++ Partner 类定义
//******************************************************************************
class TS7Partner
{
private:
	S7Object Partner; // Partner 句柄
public:
	TS7Partner(bool Active);   // 构造函数, Active: 1=主动/0=被动
	~TS7Partner();             // 析构函数
	// 控制
	int GetParam(int ParamNumber, void *pValue);   // 获取参数
	int SetParam(int ParamNumber, void *pValue);   // 设置参数
	int Start();                                   // 启动Partner
	int StartTo(const char *LocalAddress,          // 指定地址启动
				const char *RemoteAddress,
				int LocalTSAP,
				int RemoteTSAP);
	int Stop();                                    // 停止Partner
	// 数据读写: BSend
	int BSend(longword R_ID, void *pUsrData, int Size);              // 同步发送数据
	int AsBSend(longword R_ID, void *pUsrData, int Size);            // 异步发送数据
	bool CheckAsBSendCompletion(int *opResult);                      // 检查异步发送是否完成
	int WaitAsBSendCompletion(longword Timeout);                     // 等待异步发送完成
	int SetSendCallback(pfn_ParSendCompletion pCompletion, void *usrPtr); // 设置发送完成回调
	// 数据读写: BRecv
	int BRecv(longword *R_ID, void *pUsrData, int *Size, longword Timeout); // 同步接收数据(阻塞)
	bool CheckAsBRecvCompletion(int *opResult, longword *R_ID, void *pUsrData, int *Size); // 检查异步接收
	int SetRecvCallback(pfn_ParRecvCallBack pCallback, void *usrPtr); // 设置数据接收回调
	// 属性
	int Status();                                    // 获取Partner状态
	int LastError();                                 // 获取最后错误
	int GetTimes(longword *SendTime, longword *RecvTime); // 获取发送/接收耗时
	int GetStats(longword *BytesSent,                // 获取传输统计
				 longword *BytesRecv,
				 longword *ErrSend,
				 longword *ErrRecv);
	bool Linked();                                   // 查询是否已连接
};
typedef TS7Partner *PS7Partner;
//******************************************************************************
//                               文本函数
// 仅用于 C++, 纯 C 请使用 xxx_ErrorText() 函数(使用 *char)
//******************************************************************************
#define TextLen 1024

// 字符串类型(默认映射到 std::string，可按需替换为 Unicodestring/Ansistring 等)

typedef std::string TextString;

TextString CliErrorText(int Error);    // 客户端错误码转文本
TextString SrvErrorText(int Error);    // 服务端错误码转文本
TextString ParErrorText(int Error);    // Partner错误码转文本
TextString SrvEventText(TSrvEvent *Event); // 服务端事件转文本


#endif // __cplusplus
#endif // snap7_h
