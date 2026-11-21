/*
 * snap7_cgo.go — snap7 C 动态库完整 CGO 绑定
 * 覆盖 snap7.h 全部 C 函数: Client(同步+异步) / Server / Partner
 * 依赖: lib/libsnap7.so
 */
package siemens

/*
#cgo CFLAGS: -std=c11 -I${SRCDIR}/../../../third_party/lib
#cgo LDFLAGS: -L${SRCDIR}/../../../third_party/lib -lsnap7 -Wl,-rpath,$ORIGIN/third_party/lib
#include <snap7.h>
#include <stdlib.h>
#include <string.h>

// C 包装函数：用 void* 绕过 CGO 对匿名结构体指针的类型检查
static int _Cli_ReadMultiVars(S7Object client, void *items, int count) {
	return Cli_ReadMultiVars(client, (PS7DataItem)items, count);
}
static int _Cli_WriteMultiVars(S7Object client, void *items, int count) {
	return Cli_WriteMultiVars(client, (PS7DataItem)items, count);
}
*/
import "C"
import (
	"time"
	"unsafe"
)

//******************************************************************************
//                        公共常量 (COMMON)
//******************************************************************************

const (
	ErrLibInvalidParam  = -1 // 无效参数
	ErrLibInvalidObject = -2 // 无效对象句柄
)

const (
	CpuStatusUnknown = 0x00 // 未知
	CpuStatusRun     = 0x08 // 运行中
	CpuStatusStop    = 0x04 // 已停止
)

// ISO 层错误码
const (
	ErrIsoConnect          = 0x00010000 // 连接错误
	ErrIsoDisconnect       = 0x00020000 // 断开错误
	ErrIsoInvalidPDU       = 0x00030000 // PDU 格式错误
	ErrIsoInvalidDataSize  = 0x00040000 // 数据大小无效
	ErrIsoNullPointer      = 0x00050000 // 空指针
	ErrIsoShortPacket      = 0x00060000 // 短包
	ErrIsoTooManyFragments = 0x00070000 // 分片过多
	ErrIsoPduOverflow      = 0x00080000 // PDU 溢出
	ErrIsoSendPacket       = 0x00090000 // 发送错误
	ErrIsoRecvPacket       = 0x000A0000 // 接收错误
	ErrIsoInvalidParams    = 0x000B0000 // 参数无效
)

// 客户端错误码
const (
	ErrNegotiatingPDU            = 0x00100000 // PDU 协商失败
	ErrCliInvalidParams          = 0x00200000 // 参数无效
	ErrCliJobPending             = 0x00300000 // 任务进行中
	ErrCliTooManyItems           = 0x00400000 // 项目数过多
	ErrCliInvalidWordLen         = 0x00500000 // 字长度无效
	ErrCliPartialDataWritten     = 0x00600000 // 部分数据已写入
	ErrCliSizeOverPDU            = 0x00700000 // 数据超过 PDU
	ErrCliInvalidPlcAnswer       = 0x00800000 // PLC 响应无效
	ErrCliAddressOutOfRange      = 0x00900000 // 地址超出范围
	ErrCliInvalidTransportSize   = 0x00A00000 // 传输大小无效
	ErrCliWriteDataSizeMismatch  = 0x00B00000 // 写入大小不匹配
	ErrCliItemNotAvailable       = 0x00C00000 // 项目不可用
	ErrCliInvalidValue           = 0x00D00000 // 值无效
	ErrCliCannotStartPLC         = 0x00E00000 // 无法启动 PLC
	ErrCliAlreadyRun             = 0x00F00000 // PLC 已在运行
	ErrCliCannotStopPLC          = 0x01000000 // 无法停止 PLC
	ErrCliCannotCopyRamToRom     = 0x01100000 // 无法复制 RAM 到 ROM
	ErrCliCannotCompress         = 0x01200000 // 无法压缩
	ErrCliAlreadyStop            = 0x01300000 // PLC 已停止
	ErrCliFunNotAvailable        = 0x01400000 // 功能不可用
	ErrCliUploadSequenceFailed   = 0x01500000 // 上传序列失败
	ErrCliInvalidDataSizeRecvd   = 0x01600000 // 接收数据大小无效
	ErrCliInvalidBlockType       = 0x01700000 // 块类型无效
	ErrCliInvalidBlockNumber     = 0x01800000 // 块编号无效
	ErrCliInvalidBlockSize       = 0x01900000 // 块大小无效
	ErrCliDownloadSequenceFailed = 0x01A00000 // 下载序列失败
	ErrCliInsertRefused          = 0x01B00000 // 插入被拒绝
	ErrCliDeleteRefused          = 0x01C00000 // 删除被拒绝
	ErrCliNeedPassword           = 0x01D00000 // 需要密码
	ErrCliInvalidPassword        = 0x01E00000 // 密码无效
	ErrCliNoPasswordToSetOrClear = 0x01F00000 // 无需设置/清除密码
	ErrCliJobTimeout             = 0x02000000 // 任务超时
	ErrCliPartialDataRead        = 0x02100000 // 部分数据已读取
	ErrCliBufferTooSmall         = 0x02200000 // 缓冲区太小
	ErrCliFunctionRefused        = 0x02300000 // 功能被拒绝
	ErrCliDestroying             = 0x02400000 // 正在销毁
	ErrCliInvalidParamNumber     = 0x02500000 // 参数编号无效
	ErrCliCannotChangeParam      = 0x02600000 // 无法更改参数
)

const (
	S7AreaPE = 0x81 // 过程输入(I)
	S7AreaPA = 0x82 // 过程输出(Q)
	S7AreaMK = 0x83 // 标志位(M)
	S7AreaDB = 0x84 // 数据块(DB)
	S7AreaCT = 0x1C // 计数器(C)
	S7AreaTM = 0x1D // 定时器(T)
)

const (
	S7WLBit     = 0x01 // 位
	S7WLByte    = 0x02 // 字节
	S7WLWord    = 0x04 // 字(16位)
	S7WLDWord   = 0x06 // 双字(32位)
	S7WLReal    = 0x08 // 浮点数(32位)
	S7WLCounter = 0x1C // 计数器
	S7WLTimer   = 0x1D // 定时器
)

const (
	ConnTypePG    = 1 // PG 编程设备
	ConnTypeOP    = 2 // OP 操作面板
	ConnTypeBasic = 3 // 基本连接
)

const (
	ParamLocalPort     = 1  // 本地端口 (uint16)
	ParamRemotePort    = 2  // 远程端口 (uint16)
	ParamPingTimeout   = 3  // Ping 超时 ms
	ParamSendTimeout   = 4  // 发送超时 ms
	ParamRecvTimeout   = 5  // 接收超时 ms
	ParamWorkInterval  = 6  // 工作间隔 ms
	ParamSrcRef        = 7  // 源引用
	ParamDstRef        = 8  // 目标引用
	ParamSrcTSap       = 9  // 源 TSAP
	ParamPDURequest    = 10 // 请求 PDU 长度
	ParamMaxClients    = 11 // 最大客户端数
	ParamBSendTimeout  = 12 // BSend 超时 ms
	ParamBRecvTimeout  = 13 // BRecv 超时 ms
	ParamRecoveryTime  = 14 // 恢复时间 ms
	ParamKeepAliveTime = 15 // 保活时间 ms
)

const (
	JobComplete = 0 // 已完成
	JobPending  = 1 // 进行中
)

const (
	BlockOB  = 0x38 // 组织块
	BlockDB  = 0x41 // 数据块
	BlockSDB = 0x42 // 系统数据块
	BlockFC  = 0x43 // 功能
	BlockSFC = 0x44 // 系统功能
	BlockFB  = 0x45 // 功能块
	BlockSFB = 0x46 // 系统功能块
)

const (
	SubBlkOB  = 0x08 // 组织块
	SubBlkDB  = 0x0A // 数据块
	SubBlkSDB = 0x0B // 系统数据块
	SubBlkFC  = 0x0C // 功能
	SubBlkSFC = 0x0D // 系统功能
	SubBlkFB  = 0x0E // 功能块
	SubBlkSFB = 0x0F // 系统功能块
)

const (
	BlockLangAWL   = 0x01 // 语句表(AWL/LAD)
	BlockLangKOP   = 0x02 // 梯形图(KOP/FBD)
	BlockLangFUP   = 0x03 // 功能图(FUP)
	BlockLangSCL   = 0x04 // 结构化控制语言(SCL)
	BlockLangDB    = 0x05 // 数据块(DB)
	BlockLangGRAPH = 0x06 // 顺序图(GRAPH)
)

//******************************************************************************
//                        服务端常量 (SERVER)
//******************************************************************************

const (
	OperationRead  = 0 // 读操作
	OperationWrite = 1 // 写操作
	MkEvent        = 0 // 事件掩码
	MkLog          = 1 // 日志掩码
	SrvAreaPE      = 0 // 过程输入
	SrvAreaPA      = 1 // 过程输出
	SrvAreaMK      = 2 // 标志位
	SrvAreaCT      = 3 // 计数器
	SrvAreaTM      = 4 // 定时器
	SrvAreaDB      = 5 // 数据块
)

const (
	ErrSrvCannotStart        = 0x00100000 // 服务端无法启动
	ErrSrvDBNullPointer      = 0x00200000 // 空指针
	ErrSrvAreaAlreadyExists  = 0x00300000 // 区域重复注册
	ErrSrvUnknownArea        = 0x00400000 // 未知区域
	ErrSrvInvalidParams      = 0x00500000 // 参数无效
	ErrSrvTooManyDB          = 0x00600000 // DB 过多
	ErrSrvInvalidParamNumber = 0x00700000 // 参数编号无效
	ErrSrvCannotChangeParam  = 0x00800000 // 无法更改参数
)

const (
	EvcServerStarted       = 0x00000001 // 服务端已启动
	EvcServerStopped       = 0x00000002 // 服务端已停止
	EvcListenerCannotStart = 0x00000004 // 监听器无法启动
	EvcClientAdded         = 0x00000008 // 客户端已添加
	EvcClientRejected      = 0x00000010 // 客户端被拒绝
	EvcClientNoRoom        = 0x00000020 // 客户端无空间
	EvcClientException     = 0x00000040 // 客户端异常
	EvcClientDisconnected  = 0x00000080 // 客户端已断开
	EvcClientTerminated    = 0x00000100 // 客户端已终止
	EvcClientsDropped      = 0x00000200 // 客户端已丢弃
	EvcPDUincoming         = 0x00010000 // PDU 进入
	EvcDataRead            = 0x00020000 // 数据读取
	EvcDataWrite           = 0x00040000 // 数据写入
	EvcNegotiatePDU        = 0x00080000 // PDU 协商
	EvcReadSZL             = 0x00100000 // 读取 SZL
	EvcClock               = 0x00200000 // 时钟操作
	EvcUpload              = 0x00400000 // 上传
	EvcDownload            = 0x00800000 // 下载
	EvcDirectory           = 0x01000000 // 目录操作
	EvcSecurity            = 0x02000000 // 安全操作
	EvcControl             = 0x04000000 // 控制操作
	EvcAll                 = 0xFFFFFFFF // 启用所有事件
	EvcNone                = 0x00000000 // 禁用所有事件
)

const (
	EvsUnknown       = 0x0000 // 未知
	EvsStartUpload   = 0x0001 // 开始上传
	EvsStartDownload = 0x0001 // 开始下载
	EvsGetBlockList  = 0x0001 // 获取块列表
	EvsStartListBoT  = 0x0002 // 开始列出块类型
	EvsListBoT       = 0x0003 // 列出块类型
	EvsGetBlockInfo  = 0x0004 // 获取块信息
	EvsGetClock      = 0x0001 // 获取时钟
	EvsSetClock      = 0x0002 // 设置时钟
	EvsSetPassword   = 0x0001 // 设置密码
	EvsClrPassword   = 0x0002 // 清除密码
)

const (
	GrProgrammer = 0x0041 // 编程器
	GrCyclicData = 0x0042 // 循环数据
	GrBlocksInfo = 0x0043 // 块信息
	GrSZL        = 0x0044 // SZL
	GrPassword   = 0x0045 // 密码
	GrBSend      = 0x0046 // BSend
	GrClock      = 0x0047 // 时钟
	GrSecurity   = 0x0045 // 安全
)

const (
	CodeControlUnknown   = 0x0000 // 未知
	CodeControlColdStart = 0x0001 // 冷启动
	CodeControlWarmStart = 0x0002 // 热启动
	CodeControlStop      = 0x0003 // 停止
	CodeControlCompress  = 0x0004 // 压缩
	CodeControlCpyRamRom = 0x0005 // RAM 到 ROM
	CodeControlInsDel    = 0x0006 // 插入/删除
)

const (
	EvrNoError           = 0x0000 // 无错误
	EvrFragmentRejected  = 0x0001 // 分片被拒绝
	EvrMalformedPDU      = 0x0002 // PDU 格式错误
	EvrSparseBytes       = 0x0003 // 稀疏字节
	EvrCannotHandlePDU   = 0x0004 // 无法处理 PDU
	EvrNotImplemented    = 0x0005 // 未实现
	EvrErrException      = 0x0006 // 异常错误
	EvrErrAreaNotFound   = 0x0007 // 区域未找到
	EvrErrOutOfRange     = 0x0008 // 超出范围
	EvrErrOverPDU        = 0x0009 // 超过 PDU
	EvrErrTransportSize  = 0x000A // 传输大小错误
	EvrInvalidGroupUData = 0x000B // 用户数据组无效
	EvrInvalidSZL        = 0x000C // SZL 无效
	EvrDataSizeMismatch  = 0x000D // 数据大小不匹配
	EvrCannotUpload      = 0x000E // 无法上传
	EvrCannotDownload    = 0x000F // 无法下载
	EvrUploadInvalidID   = 0x0010 // 上传 ID 无效
	EvrResNotFound       = 0x0011 // 资源未找到
)

//******************************************************************************
//                        Partner 常量
//******************************************************************************

const (
	ParStopped    = 0 // 已停止
	ParConnecting = 1 // 连接中
	ParWaiting    = 2 // 等待连接
	ParLinked     = 3 // 已连接
	ParSending    = 4 // 发送中
	ParReceiving  = 5 // 接收中
	ParBindError  = 6 // 绑定错误
)

const (
	ErrParAddressInUse       = 0x00200000 // 地址已被使用
	ErrParNoRoom             = 0x00300000 // 无空间
	ErrServerNoRoom          = 0x00400000 // 服务端无空间
	ErrParInvalidParams      = 0x00500000 // 参数无效
	ErrParNotLinked          = 0x00600000 // 未连接
	ErrParBusy               = 0x00700000 // 忙
	ErrParFrameTimeout       = 0x00800000 // 帧超时
	ErrParInvalidPDU         = 0x00900000 // PDU 无效
	ErrParSendTimeout        = 0x00A00000 // 发送超时
	ErrParRecvTimeout        = 0x00B00000 // 接收超时
	ErrParSendRefused        = 0x00C00000 // 发送被拒绝
	ErrParNegotiatingPDU     = 0x00D00000 // PDU 协商中
	ErrParSendingBlock       = 0x00E00000 // 发送块中
	ErrParRecvingBlock       = 0x00F00000 // 接收块中
	ErrParBindError          = 0x01000000 // 绑定错误
	ErrParDestroying         = 0x01100000 // 正在销毁
	ErrParInvalidParamNumber = 0x01200000 // 参数编号无效
	ErrParCannotChangeParam  = 0x01300000 // 无法更改参数
	ErrParBufferTooSmall     = 0x01400000 // 缓冲区太小
)

//******************************************************************************
//                        错误类型
//******************************************************************************

// S7Error snap7 错误码，实现 Go error 接口
type S7Error int32

func (e S7Error) Error() string {
	buf := make([]byte, 1024)
	C.Cli_ErrorText(C.int(e), (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	return cStringToGo(&buf[0])
}

// s7err 将 C 返回的错误码转为 Go error (0 → nil)
func s7err(code C.int) error {
	if code == 0 {
		return nil
	}
	return S7Error(code)
}

//******************************************************************************
//                        结构体
//******************************************************************************

// S7DataItem 读写数据项（内存布局与 C.TS7DataItem 一致）
type S7DataItem struct {
	Area     int32  // 区域 ID (C int)
	WordLen  int32  // 字长度 (C int)
	Result   int32  // 操作结果 (C int)
	DBNumber int32  // DB 编号 (C int)
	Start    int32  // 起始地址 (C int)
	Amount   int32  // 数量 (C int)
	Data     []byte // 数据缓冲区
}

type S7BlocksList struct {
	OBCount, FBCount, FCCount int32 // C int
	SFBCount, SFCCount        int32 // C int
	DBCount, SDBCount         int32 // C int
}

type S7BlockInfo struct {
	BlkType   int32  // 块类型 (C int)
	BlkNumber int32  // 块编号 (C int)
	BlkLang   int32  // 块语言 (C int)
	BlkFlags  int32  // 块标志 (C int)
	MC7Size   int32  // 实际大小(字节) (C int)
	LoadSize  int32  // 加载内存大小 (C int)
	LocalData int32  // 本地数据 (C int)
	SBBLength int32  // SBB 长度 (C int)
	CheckSum  int32  // 校验和 (C int)
	Version   int32  // 块版本 (C int)
	CodeDate  string // 代码日期
	IntfDate  string // 接口日期
	Author    string // 作者
	Family    string // 族
	Header    string // 标题
}

type S7OrderCode struct {
	Code string // 订货号代码
	V1   byte   // 主版本
	V2   byte   // 次版本
	V3   byte   // 修订版本
}

type S7CpuInfo struct {
	ModuleTypeName string // 模块类型名称
	SerialNumber   string // 序列号
	ASName         string // AS 名称
	Copyright      string // 版权信息
	ModuleName     string // 模块名称
}

type S7CpInfo struct {
	MaxPduLength   int32  // 最大 PDU 长度 (C int)
	MaxConnections int32  // 最大连接数 (C int)
	MaxMpiRate     uint32 // 最大 MPI 速率 (C longword)
	MaxBusRate     uint32 // 最大总线速率 (C longword)
}

type S7SZL struct {
	LENTHDR uint16 // 数据长度
	N_DR    uint16 // 数据记录编号
	Data    []byte // SZL 数据
}

type S7SZLList struct {
	LENTHDR uint16   // 数据长度
	N_DR    uint16   // 数据记录编号
	IDs     []uint16 // SZL ID 列表
}

type S7Protection struct {
	SchSchal int32 // 保护级别 (C int)
	SchPar   int32 // 参数 (C int)
	SchRel   int32 // 相关 (C int)
	BartSch  int32 // 保护开关位置 (C int)
	AnlSch   int32 // 启动保护 (C int)
}

type SrvEvent struct {
	EvtTime    time.Time // 时间戳
	EvtSender  int32     // 发送者 (C int)
	EvtCode    uint32    // 事件代码
	EvtRetCode uint16    // 事件结果
	EvtParam1  uint16    // 参数1
	EvtParam2  uint16    // 参数2
	EvtParam3  uint16    // 参数3
	EvtParam4  uint16    // 参数4
}

// Snap7Server snap7 C 服务端的 Go 封装
type Snap7Server struct{ handle C.S7Object }

// Snap7Partner snap7 C Partner 的 Go 封装
type Snap7Partner struct{ handle C.S7Object }

//******************************************************************************
//                        Snap7Client 客户端函数
//******************************************************************************

// Snap7Client snap7 C 客户端的 Go 封装
type Snap7Client struct {
	client C.S7Object // snap7 客户端句柄
}

// NewSnap7Client 创建客户端实例 (向后兼容)
// 作用: 分配并返回一个新的 snap7 客户端对象
// 传入: 无
// 返回: [*Snap7Client] 客户端指针
func NewSnap7Client() *Snap7Client { return &Snap7Client{client: C.Cli_Create()} }

// CliCreate 创建客户端实例
// 作用: 分配并返回一个新的 snap7 客户端对象
// 传入: 无
// 返回: [*Snap7Client] 客户端指针
func CliCreate() *Snap7Client { return &Snap7Client{client: C.Cli_Create()} }

// Destroy 销毁客户端实例
// 作用: 释放客户端占用的 C 资源，句柄置零
// 传入: 无 (接收者方法)
// 返回: 无
func (c *Snap7Client) Destroy() {
	if c.client != 0 {
		C.Cli_Destroy(&c.client)
		c.client = 0
	}
}

// CliDestroy 销毁客户端实例
// 作用: 释放客户端占用的 C 资源，句柄置零
// 传入: [c *Snap7Client] 待销毁的客户端指针
// 返回: 无
func CliDestroy(c *Snap7Client) {
	if c != nil && c.client != 0 {
		C.Cli_Destroy(&c.client)
		c.client = 0
	}
}

// --- 连接管理 ---

// ConnectTo 一步连接到 PLC（自动计算 TSAP）
// 作用: 根据 IP+机架+槽位自动计算 TSAP 并建立连接
// 传入: [address string] PLC IP 地址, [rack int] 机架号(S7-300=0, S7-400=0~1), [slot int] 槽位号(S7-300=2, S7-400=1~31)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) ConnectTo(address string, rack, slot int32) error {
	cAddr := C.CString(address)
	defer C.free(unsafe.Pointer(cAddr))
	return s7err(C.Cli_ConnectTo(c.client, cAddr, C.int(rack), C.int(slot)))
}

// SetConnectionParams 设置连接参数（自定义 TSAP 模式）
// 作用: 使用自定义本地/远程 TSAP 设置连接参数，需在 Connect() 前调用
// 传入: [address string] PLC IP 地址, [localTSAP uint16] 本地 TSAP, [remoteTSAP uint16] 远程 TSAP
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetConnectionParams(address string, localTSAP, remoteTSAP uint16) error {
	cAddr := C.CString(address)
	defer C.free(unsafe.Pointer(cAddr))
	return s7err(C.Cli_SetConnectionParams(c.client, cAddr, C.word(localTSAP), C.word(remoteTSAP)))
}

// SetConnectionType 设置连接类型
// 作用: 设置连接身份类型，必须在 Connect() 前调用
// 传入: [connType int32] 连接类型: 1=PG(编程设备), 2=OP(操作面板), 3=Basic(基本连接)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetConnectionType(connType int32) error {
	return s7err(C.Cli_SetConnectionType(c.client, C.word(connType)))
}

// Connect 使用已设参数建立连接
// 作用: 使用 SetConnectionParams/SetConnectionType 设置的参数建立连接
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) Connect() error { return s7err(C.Cli_Connect(c.client)) }

// Disconnect 断开与 PLC 的连接
// 作用: 断开当前连接，但不释放客户端资源
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) Disconnect() error { return s7err(C.Cli_Disconnect(c.client)) }

// GetConnected 查询连接状态
// 作用: 检查客户端是否已与 PLC 建立连接
// 传入: 无
// 返回: [bool] true=已连接, false=未连接
func (c *Snap7Client) GetConnected() bool {
	var connected C.int
	C.Cli_GetConnected(c.client, &connected)
	return connected != 0
}

// --- 参数 ---

// SetParam 设置客户端参数
// 作用: 设置连接或通信参数（如超时、PDU 等）
// 传入: [paramNumber int] 参数编号: 1=本地端口, 2=远程端口, 3=Ping超时, 4=发送超时, 5=接收超时, 6=工作间隔, 7=源引用, 8=目标引用, 9=源TSAP, 10=请求PDU长度, 11=最大客户端数, 12=BSend超时, 13=BRecv超时, 14=恢复时间, 15=保活时间, [value int] 参数值
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetParam(paramNumber int32, value int32) error {
	v := C.int(value)
	return s7err(C.Cli_SetParam(c.client, C.int(paramNumber), unsafe.Pointer(&v)))
}

// GetParam 获取客户端参数
// 作用: 读取当前参数值
// 传入: [paramNumber int] 参数编号: 1~15, 含义同 SetParam
// 返回: [int] 参数值, [err error] nil=成功, 非nil=错误
func (c *Snap7Client) GetParam(paramNumber int32) (int32, error) {
	var v C.int
	err := s7err(C.Cli_GetParam(c.client, C.int(paramNumber), unsafe.Pointer(&v)))
	return int32(v), err
}

// GetPduLength 获取 PDU 长度信息
// 作用: 获取请求的和协商后的 PDU 长度
// 传入: 无
// 返回: [requested int] 请求的 PDU 长度, [negotiated int] 协商后的实际 PDU 长度
func (c *Snap7Client) GetPduLength() (requested int32, negotiated int32) {
	var req, neg C.int
	C.Cli_GetPduLength(c.client, &req, &neg)
	return int32(req), int32(neg)
}

// --- 数据读写：单项 ---

// ReadArea 读取 PLC 指定区域数据
// 作用: 从 PLC 的指定存储区读取数据到缓冲区
// 传入: [area int] 区域ID: 0x81=过程输入(I), 0x82=过程输出(Q), 0x83=标志位(M), 0x84=数据块(DB), 0x1C=计数器(C), 0x1D=定时器(T), [dbNumber int] DB编号(非DB区填0), [start int] 起始地址(字节偏移或位地址), [amount int] 读取数量, [wordLen int] 字长度: 0x01=位, 0x02=字节, 0x04=字, 0x06=双字, 0x08=浮点, 0x1C=计数器, 0x1D=定时器, [buf []byte] 接收缓冲区
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) ReadArea(area, dbNumber, start, amount, wordLen int32, buf []byte) error {
	return s7err(C.Cli_ReadArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// WriteArea 向 PLC 指定区域写入数据
// 作用: 将缓冲区数据写入 PLC 的指定存储区
// 传入: [area int] 区域ID: 同 ReadArea, [dbNumber int] DB编号, [start int] 起始地址, [amount int] 数量, [wordLen int] 字长度: 同 ReadArea, [buf []byte] 待写入数据
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) WriteArea(area, dbNumber, start, amount, wordLen int32, buf []byte) error {
	return s7err(C.Cli_WriteArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// --- 数据读写：快捷函数 ---

// DBRead 读取 DB 数据块
// 作用: 从指定 DB 编号读取指定长度数据
// 传入: [dbNumber int] DB编号, [start int] 起始字节, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) DBRead(dbNumber, start, size int32, buf []byte) error {
	return s7err(C.Cli_DBRead(c.client, C.int(dbNumber), C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// DBWrite 写入 DB 数据块
// 作用: 向指定 DB 编号写入数据
// 传入: [dbNumber int] DB编号, [start int] 起始字节, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) DBWrite(dbNumber, start, size int32, buf []byte) error {
	return s7err(C.Cli_DBWrite(c.client, C.int(dbNumber), C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// MBRead 读取标志位(M区)
// 作用: 从 PLC 的 M 区读取数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) MBRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_MBRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// MBWrite 写入标志位(M区)
// 作用: 向 PLC 的 M 区写入数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) MBWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_MBWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// EBRead 读取过程输入(I区)
// 作用: 从 PLC 的 I 区(过程输入)读取数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) EBRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_EBRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// EBWrite 写入过程输入(I区)
// 作用: 向 PLC 的 I 区(过程输入)写入数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) EBWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_EBWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// ABRead 读取过程输出(Q区)
// 作用: 从 PLC 的 Q 区(过程输出)读取数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) ABRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_ABRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// ABWrite 写入过程输出(Q区)
// 作用: 向 PLC 的 Q 区(过程输出)写入数据
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) ABWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_ABWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// TMRead 读取定时器
// 作用: 从 PLC 读取定时器当前值
// 传入: [start int] 起始定时器编号, [amount int] 数量, [buf []byte] 接收缓冲区(每个定时器2字节)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) TMRead(start, amount int32, buf []byte) error {
	return s7err(C.Cli_TMRead(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// TMWrite 写入定时器
// 作用: 向 PLC 定时器写入设定值
// 传入: [start int] 起始定时器编号, [amount int] 数量, [buf []byte] 待写入数据(每个定时器2字节)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) TMWrite(start, amount int32, buf []byte) error {
	return s7err(C.Cli_TMWrite(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// CTRead 读取计数器
// 作用: 从 PLC 读取计数器当前值
// 传入: [start int] 起始计数器编号, [amount int] 数量, [buf []byte] 接收缓冲区(每个计数器2字节)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) CTRead(start, amount int32, buf []byte) error {
	return s7err(C.Cli_CTRead(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// CTWrite 写入计数器
// 作用: 向 PLC 计数器写入设定值
// 传入: [start int] 起始计数器编号, [amount int] 数量, [buf []byte] 待写入数据(每个计数器2字节)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) CTWrite(start, amount int32, buf []byte) error {
	return s7err(C.Cli_CTWrite(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// --- 数据读写：批量 ---

// ReadMultiVars 批量读取多个变量（最多 20 项）
// 作用: 一次请求读取多个不同区域的数据，减少通信次数
// 传入: [items []S7DataItem] 数据项数组, 每项包含 Area/WordLen/DBNumber/Start/Amount/Data, Data 需预分配缓冲区
// 返回: [error] nil=成功, 非零=错误码; 每项的 Result 字段单独表示该次读取结果
func (c *Snap7Client) ReadMultiVars(items []S7DataItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}
	cItems := C.PS7DataItem(C.malloc(C.size_t(count) * C.sizeof_TS7DataItem))
	defer C.free(unsafe.Pointer(cItems))
	cBufs := make([]unsafe.Pointer, count)
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		ptr.Area = C.int(items[i].Area)
		ptr.WordLen = C.int(items[i].WordLen)
		ptr.DBNumber = C.int(items[i].DBNumber)
		ptr.Start = C.int(items[i].Start)
		ptr.Amount = C.int(items[i].Amount)
		bufLen := len(items[i].Data)
		if bufLen <= 0 {
			bufLen = int(items[i].Amount)
		}
		buf := C.malloc(C.size_t(bufLen))
		cBufs[i] = buf
		ptr.pdata = buf
	}
	result := C._Cli_ReadMultiVars(c.client, unsafe.Pointer(cItems), C.int(count))
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		items[i].Result = int32(ptr.Result)
		if items[i].Result == 0 && items[i].Data != nil {
			copyLen := len(items[i].Data)
			if C.int(copyLen) > ptr.Amount {
				if items[i].WordLen == S7WLTimer || items[i].WordLen == S7WLCounter {
					if copyLen > 2 {
						copyLen = 2
					}
				} else {
					amt := int(ptr.Amount)
					if amt < copyLen {
						copyLen = amt
					}
				}
			}
			if copyLen > 0 {
				C.memcpy(unsafe.Pointer(&items[i].Data[0]), cBufs[i], C.size_t(copyLen))
			}
		}
		C.free(cBufs[i])
	}
	return s7err(result)
}

// WriteMultiVars 批量写入多个变量（最多 20 项）
// 作用: 一次请求写入多个不同区域的数据，减少通信次数
// 传入: [items []S7DataItem] 数据项数组, 每项包含 Area/WordLen/DBNumber/Start/Amount/Data
// 返回: [error] nil=成功, 非零=错误码; 每项的 Result 字段单独表示该次写入结果
func (c *Snap7Client) WriteMultiVars(items []S7DataItem) error {
	count := len(items)
	if count == 0 {
		return nil
	}
	cItems := C.PS7DataItem(C.malloc(C.size_t(count) * C.sizeof_TS7DataItem))
	defer C.free(unsafe.Pointer(cItems))
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		ptr.Area = C.int(items[i].Area)
		ptr.WordLen = C.int(items[i].WordLen)
		ptr.DBNumber = C.int(items[i].DBNumber)
		ptr.Start = C.int(items[i].Start)
		ptr.Amount = C.int(items[i].Amount)
		if len(items[i].Data) > 0 {
			ptr.pdata = unsafe.Pointer(&items[i].Data[0])
		}
	}
	result := C._Cli_WriteMultiVars(c.client, unsafe.Pointer(cItems), C.int(count))
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		items[i].Result = int32(ptr.Result)
	}
	return s7err(result)
}

// --- 目录/块信息 ---

// ListBlocks 列出 PLC 中所有块的数量统计
// 作用: 获取 PLC 中各类型块的总数
// 传入: 无
// 返回: [S7BlocksList] 各类型块数量(OB/FB/FC/SFB/SFC/DB/SDB), [error] nil=成功, 非nil=错误
func (c *Snap7Client) ListBlocks() (S7BlocksList, error) {
	var cList C.TS7BlocksList
	err := s7err(C.Cli_ListBlocks(c.client, &cList))
	return S7BlocksList{
		OBCount: int32(cList.OBCount), FBCount: int32(cList.FBCount), FCCount: int32(cList.FCCount),
		SFBCount: int32(cList.SFBCount), SFCCount: int32(cList.SFCCount),
		DBCount: int32(cList.DBCount), SDBCount: int32(cList.SDBCount),
	}, err
}

// GetAgBlockInfo 获取 AG 侧块信息
// 作用: 从 PLC 获取指定块的详细信息（大小、语言、作者等）
// 传入: [blockType int] 块类型: 0x38=OB, 0x41=DB, 0x42=SDB, 0x43=FC, 0x44=SFC, 0x45=FB, 0x46=SFB, [blockNum int] 块编号
// 返回: [S7BlockInfo] 块详细信息, [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetAgBlockInfo(blockType, blockNum int32) (S7BlockInfo, error) {
	var cInfo C.TS7BlockInfo
	err := s7err(C.Cli_GetAgBlockInfo(c.client, C.int(blockType), C.int(blockNum), &cInfo))
	return cBlockInfoToGo(&cInfo), err
}

// GetPgBlockInfo 获取 PG 侧块信息
// 作用: 从已上传的块数据中解析块详细信息
// 传入: [blockData []byte] 已上传的块原始数据, [size int] 数据长度
// 返回: [S7BlockInfo] 块详细信息, [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetPgBlockInfo(blockData []byte, size int32) (S7BlockInfo, error) {
	var cInfo C.TS7BlockInfo
	err := s7err(C.Cli_GetPgBlockInfo(c.client, unsafe.Pointer(&blockData[0]), &cInfo, C.int(size)))
	return cBlockInfoToGo(&cInfo), err
}

// ListBlocksOfType 列出指定类型的所有块编号
// 作用: 获取 PLC 中某一类型块的全部编号列表
// 传入: [blockType int] 块类型: 0x38=OB, 0x41=DB, 0x43=FC, 0x45=FB 等
// 返回: [[]uint16] 块编号列表, [error] nil=成功, 非nil=错误
func (c *Snap7Client) ListBlocksOfType(blockType int32) ([]uint16, error) {
	var cBlocks C.TS7BlocksOfType
	var count C.int
	err := s7err(C.Cli_ListBlocksOfType(c.client, C.int(blockType), &cBlocks, &count))
	blocks := make([]uint16, int(count))
	for i := 0; i < int(count); i++ {
		blocks[i] = uint16(cBlocks[i])
	}
	return blocks, err
}

// --- 块操作 ---

// Upload 上传块（不含头信息）
// 作用: 从 PLC 上传指定块的 MC7 代码（不含块头）
// 传入: [blockType int] 块类型: 0x38=OB, 0x41=DB, 0x43=FC 等, [blockNum int] 块编号
// 返回: [[]byte] 块原始数据, [error] nil=成功, 非nil=错误
func (c *Snap7Client) Upload(blockType, blockNum int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_Upload(c.client, C.int(blockType), C.int(blockNum), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// FullUpload 完整上传块（含头信息）
// 作用: 从 PLC 上传指定块的完整数据（包含块头）
// 传入: [blockType int] 块类型: 同 Upload, [blockNum int] 块编号
// 返回: [[]byte] 块完整原始数据, [error] nil=成功, 非nil=错误
func (c *Snap7Client) FullUpload(blockType, blockNum int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_FullUpload(c.client, C.int(blockType), C.int(blockNum), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// Download 下载块到 PLC
// 作用: 将块数据下载到 PLC
// 传入: [blockNum int] 块编号, [data []byte] 块原始数据(通常由 FullUpload 获取)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) Download(blockNum int32, data []byte) error {
	return s7err(C.Cli_Download(c.client, C.int(blockNum), unsafe.Pointer(&data[0]), C.int(len(data))))
}

// Delete 删除 PLC 中的块
// 作用: 从 PLC 中删除指定块
// 传入: [blockType int] 块类型: 同 Upload, [blockNum int] 块编号
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) Delete(blockType, blockNum int32) error {
	return s7err(C.Cli_Delete(c.client, C.int(blockType), C.int(blockNum)))
}

// DBGet 获取 DB 完整内容
// 作用: 从 PLC 读取指定 DB 的完整数据
// 传入: [dbNumber int] DB编号
// 返回: [[]byte] DB 数据内容, [error] nil=成功, 非nil=错误
func (c *Snap7Client) DBGet(dbNumber int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_DBGet(c.client, C.int(dbNumber), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// DBFill 填充 DB 数据块
// 作用: 用指定字节值填充整个 DB 块
// 传入: [dbNumber int] DB编号, [fillChar int] 填充字节值(0~255)
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) DBFill(dbNumber, fillChar int32) error {
	return s7err(C.Cli_DBFill(c.client, C.int(dbNumber), C.int(fillChar)))
}

// --- 日期/时间 ---

// GetPlcDateTime 获取 PLC 日期时间
// 作用: 读取 PLC 内部的实时时钟
// 传入: 无
// 返回: [time.Time] PLC 当前时间(UTC), [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetPlcDateTime() (time.Time, error) {
	var tm C.tm
	err := s7err(C.Cli_GetPlcDateTime(c.client, &tm))
	t := time.Date(int(tm.tm_year)+1900, time.Month(int(tm.tm_mon)+1), int(tm.tm_mday),
		int(tm.tm_hour), int(tm.tm_min), int(tm.tm_sec), 0, time.UTC)
	return t, err
}

// SetPlcDateTime 设置 PLC 日期时间
// 作用: 将指定时间写入 PLC 实时时钟
// 传入: [t time.Time] 要设置的日期时间
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetPlcDateTime(t time.Time) error {
	tm := C.tm{
		tm_sec: C.int(t.Second()), tm_min: C.int(t.Minute()), tm_hour: C.int(t.Hour()),
		tm_mday: C.int(t.Day()), tm_mon: C.int(t.Month() - 1), tm_year: C.int(t.Year() - 1900),
	}
	return s7err(C.Cli_SetPlcDateTime(c.client, &tm))
}

// SetPlcSystemDateTime 同步 PLC 与系统时间
// 作用: 将 PLC 时钟设置为上位机的当前系统时间
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetPlcSystemDateTime() error {
	return s7err(C.Cli_SetPlcSystemDateTime(c.client))
}

// --- 系统信息 ---

// GetOrderCode 获取 PLC 订货号
// 作用: 读取 PLC 的订货号代码和固件版本
// 传入: 无
// 返回: [S7OrderCode] 订货号信息(Code=订货号字符串, V1/V2/V3=版本号), [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetOrderCode() (S7OrderCode, error) {
	var cCode C.TS7OrderCode
	err := s7err(C.Cli_GetOrderCode(c.client, &cCode))
	return S7OrderCode{
		Code: C.GoString(&cCode.Code[0]),
		V1:   byte(cCode.V1), V2: byte(cCode.V2), V3: byte(cCode.V3),
	}, err
}

// GetCpuInfo 获取 CPU 信息
// 作用: 读取 PLC CPU 的型号、序列号、名称等信息
// 传入: 无
// 返回: [S7CpuInfo] CPU 信息(ModuleTypeName=模块类型, SerialNumber=序列号, ASName=AS名称, ModuleName=模块名称), [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetCpuInfo() (S7CpuInfo, error) {
	var cInfo C.TS7CpuInfo
	err := s7err(C.Cli_GetCpuInfo(c.client, &cInfo))
	return S7CpuInfo{
		ModuleTypeName: C.GoString(&cInfo.ModuleTypeName[0]),
		SerialNumber:   C.GoString(&cInfo.SerialNumber[0]),
		ASName:         C.GoString(&cInfo.ASName[0]),
		Copyright:      C.GoString(&cInfo.Copyright[0]),
		ModuleName:     C.GoString(&cInfo.ModuleName[0]),
	}, err
}

// GetCpInfo 获取 CP 通信处理器信息
// 作用: 读取 PLC 通信处理器的能力参数
// 传入: 无
// 返回: [S7CpInfo] CP 信息(MaxPduLength=最大PDU长度, MaxConnections=最大连接数, MaxMpiRate=最大MPI速率, MaxBusRate=最大总线速率), [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetCpInfo() (S7CpInfo, error) {
	var cInfo C.TS7CpInfo
	err := s7err(C.Cli_GetCpInfo(c.client, &cInfo))
	return S7CpInfo{
		MaxPduLength:   int32(cInfo.MaxPduLengt),
		MaxConnections: int32(cInfo.MaxConnections),
		MaxMpiRate:     uint32(cInfo.MaxMpiRate),
		MaxBusRate:     uint32(cInfo.MaxBusRate),
	}, err
}

// ReadSZL 读取系统状态列表(SZL)
// 作用: 读取 PLC 的指定 SZL 数据记录，用于获取硬件、诊断等系统信息
// 传入: [szlID int] SZL ID(如 0x0011=模块标识, 0x001C=CPU特性等), [index int] 数据记录索引
// 返回: [S7SZL] SZL 数据(LENTHDR=数据长度, N_DR=记录编号, Data=原始数据), [error] nil=成功, 非nil=错误
func (c *Snap7Client) ReadSZL(szlID, index int32) (S7SZL, error) {
	var cSZL C.TS7SZL
	var cSize C.int = C.sizeof_TS7SZL
	err := s7err(C.Cli_ReadSZL(c.client, C.int(szlID), C.int(index), &cSZL, &cSize))
	dataLen := int(cSZL.Header.LENTHDR)
	maxLen := int(C.sizeof_TS7SZL) - 4
	if dataLen <= 0 || dataLen > maxLen {
		dataLen = maxLen
	}
	data := make([]byte, dataLen)
	if dataLen > 0 {
		C.memcpy(unsafe.Pointer(&data[0]), unsafe.Pointer(&cSZL.Data[0]), C.size_t(dataLen))
	}
	return S7SZL{LENTHDR: uint16(cSZL.Header.LENTHDR), N_DR: uint16(cSZL.Header.N_DR), Data: data}, err
}

// ReadSZLList 读取可用 SZL ID 列表
// 作用: 获取 PLC 支持的所有 SZL ID 列表
// 传入: 无
// 返回: [S7SZLList] SZL ID 列表(IDs=可用的 SZL ID 数组), [error] nil=成功, 非nil=错误
func (c *Snap7Client) ReadSZLList() (S7SZLList, error) {
	var cList C.TS7SZLList
	var cCount C.int
	err := s7err(C.Cli_ReadSZLList(c.client, &cList, &cCount))
	n := int(cCount)
	if n < 0 {
		n = 0
	}
	ids := make([]uint16, n)
	for i := 0; i < n; i++ {
		ids[i] = uint16(cList.List[i])
	}
	return S7SZLList{
		LENTHDR: uint16(cList.Header.LENTHDR), N_DR: uint16(cList.Header.N_DR), IDs: ids,
	}, err
}

// GetProtection 获取 PLC 保护等级
// 作用: 读取 PLC 的安全保护状态（密码、保护开关等）
// 传入: 无
// 返回: [S7Protection] 保护信息(SchSchal=保护级别, BartSch=保护开关位置, AnlSch=启动保护), [error] nil=成功, 非nil=错误
func (c *Snap7Client) GetProtection() (S7Protection, error) {
	var cProt C.TS7Protection
	err := s7err(C.Cli_GetProtection(c.client, &cProt))
	return S7Protection{
		SchSchal: int32(cProt.sch_schal), SchPar: int32(cProt.sch_par),
		SchRel: int32(cProt.sch_rel), BartSch: int32(cProt.bart_sch), AnlSch: int32(cProt.anl_sch),
	}, err
}

// --- PLC 控制 ---

// PlcHotStart 热启动 PLC
// 作用: 将 PLC 切换到 RUN 模式(保持当前数据)
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) PlcHotStart() error { return s7err(C.Cli_PlcHotStart(c.client)) }

// PlcColdStart 冷启动 PLC
// 作用: 将 PLC 切换到 RUN 模式(清除所有数据)
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) PlcColdStart() error { return s7err(C.Cli_PlcColdStart(c.client)) }

// PlcStop 停止 PLC
// 作用: 将 PLC 切换到 STOP 模式
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) PlcStop() error { return s7err(C.Cli_PlcStop(c.client)) }

// CopyRamToRom 将 RAM 复制到 ROM
// 作用: 将 PLC 当前 RAM 数据保存到 ROM(断电保持)
// 传入: [timeout int] 超时时间(毫秒), 0=使用默认值
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) CopyRamToRom(timeout int32) error {
	return s7err(C.Cli_CopyRamToRom(c.client, C.int(timeout)))
}

// Compress 压缩 PLC 内存
// 作用: 清理 PLC 内存中的碎片
// 传入: [timeout int] 超时时间(毫秒), 0=使用默认值
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) Compress(timeout int32) error {
	return s7err(C.Cli_Compress(c.client, C.int(timeout)))
}

// GetPlcStatus 获取 PLC 运行状态
// 作用: 查询 PLC 当前运行状态
// 传入: 无
// 返回: [int32] 0x00=未知, 0x04=停止(Stop), 0x08=运行(Run), [err error] nil=成功, 非nil=错误
func (c *Snap7Client) GetPlcStatus() (int32, error) {
	var status C.int
	C.Cli_GetPlcStatus(c.client, &status)
	return int32(status), s7err(status)
}

// --- 安全/密码 ---

// SetSessionPassword 设置会话密码
// 作用: 设置与 PLC 通信的会话密码，用于访问受保护的 PLC
// 传入: [password string] 密码字符串
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) SetSessionPassword(password string) error {
	cPwd := C.CString(password)
	defer C.free(unsafe.Pointer(cPwd))
	return s7err(C.Cli_SetSessionPassword(c.client, cPwd))
}

// ClearSessionPassword 清除会话密码
// 作用: 清除当前会话密码，解除 PLC 的访问保护
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (c *Snap7Client) ClearSessionPassword() error {
	return s7err(C.Cli_ClearSessionPassword(c.client))
}

// --- 底层 ---

// IsoExchangeBuffer ISO 层缓冲区交换
// 作用: 发送原始 ISO 数据并接收 PLC 响应，用于底层协议调试
// 传入: [buf []byte] 待发送的 ISO 数据帧
// 返回: [[]byte] PLC 响应数据, [error] nil=成功, 非nil=错误
func (c *Snap7Client) IsoExchangeBuffer(buf []byte) ([]byte, error) {
	size := C.int(len(buf))
	err := s7err(C.Cli_IsoExchangeBuffer(c.client, unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// --- 杂项 ---

// GetExecTime 获取执行时间
// 作用: 获取上一次操作的执行耗时
// 传入: 无
// 返回: [int] 执行时间(毫秒)
func (c *Snap7Client) GetExecTime() int32 {
	var t C.int
	C.Cli_GetExecTime(c.client, &t)
	return int32(t)
}

// GetLastError 获取最后错误码
// 作用: 获取最近一次操作的错误码
// 传入: 无
// 返回: [int] 错误码: 0=无错误, 非零=具体错误码(参见 ErrXxx 常量)
func (c *Snap7Client) GetLastError() int32 {
	var e C.int
	C.Cli_GetLastError(c.client, &e)
	return int32(e)
}

//******************************************************************************
//                     Snap7Client 异步函数
//******************************************************************************

// CheckAsCompletion 检查异步操作是否完成
// 作用: 非阻塞检查当前异步操作是否已完成
// 传入: 无
// 返回: [opResult int32] 操作结果: 0=已完成, 1=进行中, [err error] nil=成功, 非nil=错误码
func (c *Snap7Client) CheckAsCompletion() (opResult int32, err error) {
	var result C.int
	err = s7err(C.Cli_CheckAsCompletion(c.client, &result))
	return int32(result), err
}

// WaitAsCompletion 等待异步操作完成
// 作用: 阻塞等待当前异步操作完成或超时
// 传入: [timeout time.Duration] 最大等待时间
// 返回: [error] nil=成功, 非零=错误码(超时返回 ErrCliJobTimeout)
func (c *Snap7Client) WaitAsCompletion(timeout time.Duration) error {
	return s7err(C.Cli_WaitAsCompletion(c.client, C.int(timeout.Milliseconds())))
}

// AsReadArea 异步读取 PLC 指定区域数据
// 作用: 非阻塞读取 PLC 数据，需用 CheckAsCompletion/WaitAsCompletion 等待结果
// 传入: [area int] 区域ID: 同 ReadArea, [dbNumber int] DB编号, [start int] 起始地址, [amount int] 数量, [wordLen int] 字长度, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsReadArea(area, dbNumber, start, amount, wordLen int32, buf []byte) error {
	return s7err(C.Cli_AsReadArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// AsWriteArea 异步写入 PLC 指定区域数据
// 作用: 非阻塞写入 PLC 数据，需用 CheckAsCompletion/WaitAsCompletion 等待结果
// 传入: [area int] 区域ID: 同 WriteArea, [dbNumber int] DB编号, [start int] 起始地址, [amount int] 数量, [wordLen int] 字长度, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsWriteArea(area, dbNumber, start, amount, wordLen int32, buf []byte) error {
	return s7err(C.Cli_AsWriteArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// AsDBRead 异步读取 DB 数据块
// 作用: 非阻塞读取 DB 数据，参数含义同 DBRead
// 传入: [dbNumber int] DB编号, [start int] 起始字节, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsDBRead(dbNumber, start, size int32, buf []byte) error {
	return s7err(C.Cli_AsDBRead(c.client, C.int(dbNumber), C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsDBWrite 异步写入 DB 数据块
// 作用: 非阻塞写入 DB 数据，参数含义同 DBWrite
// 传入: [dbNumber int] DB编号, [start int] 起始字节, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsDBWrite(dbNumber, start, size int32, buf []byte) error {
	return s7err(C.Cli_AsDBWrite(c.client, C.int(dbNumber), C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsMBRead 异步读取标志位(M区)
// 作用: 非阻塞读取 M 区数据，参数含义同 MBRead
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsMBRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsMBRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsMBWrite 异步写入标志位(M区)
// 作用: 非阻塞写入 M 区数据，参数含义同 MBWrite
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsMBWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsMBWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsEBRead 异步读取过程输入(I区)
// 作用: 非阻塞读取 I 区数据，参数含义同 EBRead
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsEBRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsEBRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsEBWrite 异步写入过程输入(I区)
// 作用: 非阻塞写入 I 区数据，参数含义同 EBWrite
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsEBWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsEBWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsABRead 异步读取过程输出(Q区)
// 作用: 非阻塞读取 Q 区数据，参数含义同 ABRead
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsABRead(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsABRead(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsABWrite 异步写入过程输出(Q区)
// 作用: 非阻塞写入 Q 区数据，参数含义同 ABWrite
// 传入: [start int] 起始地址, [size int] 字节数, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsABWrite(start, size int32, buf []byte) error {
	return s7err(C.Cli_AsABWrite(c.client, C.int(start), C.int(size), unsafe.Pointer(&buf[0])))
}

// AsTMRead 异步读取定时器
// 作用: 非阻塞读取定时器，参数含义同 TMRead
// 传入: [start int] 起始定时器编号, [amount int] 数量, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsTMRead(start, amount int32, buf []byte) error {
	return s7err(C.Cli_AsTMRead(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// AsTMWrite 异步写入定时器
// 作用: 非阻塞写入定时器，参数含义同 TMWrite
// 传入: [start int] 起始定时器编号, [amount int] 数量, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsTMWrite(start, amount int32, buf []byte) error {
	return s7err(C.Cli_AsTMWrite(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// AsCTRead 异步读取计数器
// 作用: 非阻塞读取计数器，参数含义同 CTRead
// 传入: [start int] 起始计数器编号, [amount int] 数量, [buf []byte] 接收缓冲区
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsCTRead(start, amount int32, buf []byte) error {
	return s7err(C.Cli_AsCTRead(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// AsCTWrite 异步写入计数器
// 作用: 非阻塞写入计数器，参数含义同 CTWrite
// 传入: [start int] 起始计数器编号, [amount int] 数量, [buf []byte] 待写入数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsCTWrite(start, amount int32, buf []byte) error {
	return s7err(C.Cli_AsCTWrite(c.client, C.int(start), C.int(amount), unsafe.Pointer(&buf[0])))
}

// AsListBlocksOfType 异步列出指定类型的所有块编号
// 作用: 非阻塞获取某一类型块的全部编号列表
// 传入: [blockType int] 块类型: 0x38=OB, 0x41=DB, 0x43=FC, 0x45=FB 等
// 返回: [[]uint16] 块编号列表, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsListBlocksOfType(blockType int32) ([]uint16, error) {
	var cBlocks C.TS7BlocksOfType
	var count C.int
	err := s7err(C.Cli_AsListBlocksOfType(c.client, C.int(blockType), &cBlocks, &count))
	blocks := make([]uint16, int(count))
	for i := 0; i < int(count); i++ {
		blocks[i] = uint16(cBlocks[i])
	}
	return blocks, err
}

// AsReadSZL 异步读取系统状态列表(SZL)
// 作用: 非阻塞读取 SZL 数据，参数含义同 ReadSZL
// 传入: [szlID int] SZL ID, [index int] 数据记录索引
// 返回: [S7SZL] SZL 数据, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsReadSZL(szlID, index int32) (S7SZL, error) {
	var cSZL C.TS7SZL
	var cSize C.int = C.sizeof_TS7SZL
	err := s7err(C.Cli_AsReadSZL(c.client, C.int(szlID), C.int(index), &cSZL, &cSize))
	dataLen := int(cSZL.Header.LENTHDR)
	maxLen := int(C.sizeof_TS7SZL) - 4
	if dataLen <= 0 || dataLen > maxLen {
		dataLen = maxLen
	}
	data := make([]byte, dataLen)
	if dataLen > 0 {
		C.memcpy(unsafe.Pointer(&data[0]), unsafe.Pointer(&cSZL.Data[0]), C.size_t(dataLen))
	}
	return S7SZL{LENTHDR: uint16(cSZL.Header.LENTHDR), N_DR: uint16(cSZL.Header.N_DR), Data: data}, err
}

// AsReadSZLList 异步读取可用 SZL ID 列表
// 作用: 非阻塞获取 PLC 支持的所有 SZL ID 列表
// 传入: 无
// 返回: [S7SZLList] SZL ID 列表, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsReadSZLList() (S7SZLList, error) {
	var cList C.TS7SZLList
	var cCount C.int
	err := s7err(C.Cli_AsReadSZLList(c.client, &cList, &cCount))
	n := int(cCount)
	if n < 0 {
		n = 0
	}
	ids := make([]uint16, n)
	for i := 0; i < n; i++ {
		ids[i] = uint16(cList.List[i])
	}
	return S7SZLList{
		LENTHDR: uint16(cList.Header.LENTHDR), N_DR: uint16(cList.Header.N_DR), IDs: ids,
	}, err
}

// AsUpload 异步上传块（不含头信息）
// 作用: 非阻塞上传块的 MC7 代码，参数含义同 Upload
// 传入: [blockType int] 块类型, [blockNum int] 块编号
// 返回: [[]byte] 块原始数据, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsUpload(blockType, blockNum int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_AsUpload(c.client, C.int(blockType), C.int(blockNum), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// AsFullUpload 异步完整上传块（含头信息）
// 作用: 非阻塞上传块的完整数据，参数含义同 FullUpload
// 传入: [blockType int] 块类型, [blockNum int] 块编号
// 返回: [[]byte] 块完整原始数据, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsFullUpload(blockType, blockNum int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_AsFullUpload(c.client, C.int(blockType), C.int(blockNum), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// AsDownload 异步下载块到 PLC
// 作用: 非阻塞下载块数据，参数含义同 Download
// 传入: [blockNum int] 块编号, [data []byte] 块原始数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsDownload(blockNum int32, data []byte) error {
	return s7err(C.Cli_AsDownload(c.client, C.int(blockNum), unsafe.Pointer(&data[0]), C.int(len(data))))
}

// AsCopyRamToRom 异步将 RAM 复制到 ROM
// 作用: 非阻塞将 RAM 数据保存到 ROM，参数含义同 CopyRamToRom
// 传入: [timeout int] 超时时间(毫秒), 0=使用默认值
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsCopyRamToRom(timeout int32) error {
	return s7err(C.Cli_AsCopyRamToRom(c.client, C.int(timeout)))
}

// AsCompress 异步压缩 PLC 内存
// 作用: 非阻塞清理内存碎片，参数含义同 Compress
// 传入: [timeout int] 超时时间(毫秒), 0=使用默认值
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsCompress(timeout int32) error {
	return s7err(C.Cli_AsCompress(c.client, C.int(timeout)))
}

// AsDBGet 异步获取 DB 完整内容
// 作用: 非阻塞读取指定 DB 的完整数据，参数含义同 DBGet
// 传入: [dbNumber int] DB编号
// 返回: [[]byte] DB 数据内容, [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsDBGet(dbNumber int32) ([]byte, error) {
	buf := make([]byte, 65536)
	var size C.int = C.int(len(buf))
	err := s7err(C.Cli_AsDBGet(c.client, C.int(dbNumber), unsafe.Pointer(&buf[0]), &size))
	return buf[:int(size)], err
}

// AsDBFill 异步填充 DB 数据块
// 作用: 非阻塞用指定字节填充整个 DB 块，参数含义同 DBFill
// 传入: [dbNumber int] DB编号, [fillChar int] 填充字节值(0~255)
// 返回: [error] nil=任务已提交, 非nil=错误
func (c *Snap7Client) AsDBFill(dbNumber, fillChar int32) error {
	return s7err(C.Cli_AsDBFill(c.client, C.int(dbNumber), C.int(fillChar)))
}

//******************************************************************************
//                        Snap7Server 服务端函数
//******************************************************************************

// SrvCreate 创建服务端实例
// 作用: 分配并返回一个新的 snap7 服务端对象，用于模拟 PLC
// 传入: 无
// 返回: [*Snap7Server] 服务端指针
func SrvCreate() *Snap7Server { return &Snap7Server{handle: C.Srv_Create()} }

// SrvDestroy 销毁服务端实例
// 作用: 释放服务端占用的 C 资源
// 传入: [s *Snap7Server] 待销毁的服务端指针
// 返回: 无
func SrvDestroy(s *Snap7Server) {
	if s != nil && s.handle != 0 {
		C.Srv_Destroy(&s.handle)
		s.handle = 0
	}
}

// Start 启动服务端（监听所有接口）
// 作用: 启动服务端监听，接受客户端连接
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) Start() error { return s7err(C.Srv_Start(s.handle)) }

// Stop 停止服务端
// 作用: 停止服务端监听，断开所有客户端
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) Stop() error { return s7err(C.Srv_Stop(s.handle)) }

// ClearEvents 清除事件队列
// 作用: 清空服务端的事件缓冲区
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) ClearEvents() error { return s7err(C.Srv_ClearEvents(s.handle)) }

// StartTo 启动服务端（指定监听地址）
// 作用: 在指定 IP 地址上启动服务端监听
// 传入: [address string] 监听 IP 地址(如 "0.0.0.0")
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) StartTo(address string) error {
	cAddr := C.CString(address)
	defer C.free(unsafe.Pointer(cAddr))
	return s7err(C.Srv_StartTo(s.handle, cAddr))
}

// SetParam 设置服务端参数
// 作用: 设置服务端通信参数
// 传入: [paramNumber int] 参数编号: 11=最大客户端数, 12=BSend超时, 13=BRecv超时, 14=恢复时间, 15=保活时间, [value int] 参数值
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) SetParam(paramNumber int32, value int32) error {
	v := C.int(value)
	return s7err(C.Srv_SetParam(s.handle, C.int(paramNumber), unsafe.Pointer(&v)))
}

// GetParam 获取服务端参数
// 作用: 读取服务端当前参数值
// 传入: [paramNumber int] 参数编号: 同 SetParam
// 返回: [int] 参数值, [err error] nil=成功, 非nil=错误
func (s *Snap7Server) GetParam(paramNumber int32) (int32, error) {
	var v C.int
	err := s7err(C.Srv_GetParam(s.handle, C.int(paramNumber), unsafe.Pointer(&v)))
	return int32(v), err
}

// RegisterArea 注册内存区域
// 作用: 向服务端注册一块内存区域，供客户端读写
// 传入: [areaCode int] 区域代码: 0=PE, 1=PA, 2=MK, 3=CT, 4=TM, 5=DB, [index uint16] DB编号(DB区有效), [buf unsafe.Pointer] 数据缓冲区指针, [size int] 缓冲区大小(字节)
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) RegisterArea(areaCode int32, index uint16, buf unsafe.Pointer, size int32) error {
	return s7err(C.Srv_RegisterArea(s.handle, C.int(areaCode), C.word(index), buf, C.int(size)))
}

// UnregisterArea 注销内存区域
// 作用: 从服务端注销已注册的内存区域
// 传入: [areaCode int] 区域代码: 同 RegisterArea, [index uint16] DB编号
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) UnregisterArea(areaCode int32, index uint16) error {
	return s7err(C.Srv_UnregisterArea(s.handle, C.int(areaCode), C.word(index)))
}

// LockArea 锁定内存区域
// 作用: 锁定指定区域，阻止客户端访问
// 传入: [areaCode int] 区域代码: 同 RegisterArea, [index uint16] DB编号
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) LockArea(areaCode int32, index uint16) error {
	return s7err(C.Srv_LockArea(s.handle, C.int(areaCode), C.word(index)))
}

// UnlockArea 解锁内存区域
// 作用: 解锁指定区域，恢复客户端访问
// 传入: [areaCode int] 区域代码: 同 RegisterArea, [index uint16] DB编号
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) UnlockArea(areaCode int32, index uint16) error {
	return s7err(C.Srv_UnlockArea(s.handle, C.int(areaCode), C.word(index)))
}

// GetStatus 获取服务端状态
// 作用: 查询服务端运行状态、CPU 状态和客户端数量
// 传入: 无
// 返回: [serverStatus int] 服务端状态: 0=停止, 1=运行, [cpuStatus int] CPU状态: 0x00=未知, 0x04=Stop, 0x08=Run, [clientsCount int] 当前连接的客户端数, [err error] nil=成功, 非nil=错误
func (s *Snap7Server) GetStatus() (serverStatus, cpuStatus, clientsCount int32, err error) {
	var ss, cs, cc C.int
	C.Srv_GetStatus(s.handle, &ss, &cs, &cc)
	return int32(ss), int32(cs), int32(cc), nil
}

// SetCpuStatus 设置 CPU 状态
// 作用: 设置服务端模拟的 PLC CPU 状态
// 传入: [status int32] CPU状态: 0x00=未知, 0x04=Stop, 0x08=Run
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) SetCpuStatus(status int32) error {
	return s7err(C.Srv_SetCpuStatus(s.handle, C.int(status)))
}

// PickEvent 取出一个事件
// 作用: 从事件队列中取出一个事件（非阻塞）
// 传入: 无
// 返回: [SrvEvent] 事件详情(EvtTime=时间, EvtCode=事件码, EvtRetCode=结果码), [bool] true=有事件, false=队列为空
func (s *Snap7Server) PickEvent() (SrvEvent, bool) {
	var cEvt C.TSrvEvent
	var ready C.int
	C.Srv_PickEvent(s.handle, &cEvt, &ready)
	if ready == 0 {
		return SrvEvent{}, false
	}
	return SrvEvent{
		EvtTime: time.Unix(int64(cEvt.EvtTime), 0), EvtSender: int32(cEvt.EvtSender),
		EvtCode: uint32(cEvt.EvtCode), EvtRetCode: uint16(cEvt.EvtRetCode),
		EvtParam1: uint16(cEvt.EvtParam1), EvtParam2: uint16(cEvt.EvtParam2),
		EvtParam3: uint16(cEvt.EvtParam3), EvtParam4: uint16(cEvt.EvtParam4),
	}, true
}

// GetMask 获取事件掩码
// 作用: 获取指定类型的事件掩码
// 传入: [maskKind int32] 掩码类型: 0=事件掩码(MkEvent), 1=日志掩码(MkLog)
// 返回: [uint32] 掩码值(EvcXxx 常量组合)
func (s *Snap7Server) GetMask(maskKind int32) uint32 {
	var mask C.longword
	C.Srv_GetMask(s.handle, C.int(maskKind), &mask)
	return uint32(mask)
}

// SetMask 设置事件掩码
// 作用: 设置指定类型的事件掩码，过滤哪些事件被记录
// 传入: [maskKind int32] 掩码类型: 0=事件掩码, 1=日志掩码, [mask uint32] 掩码值(EvcXxx 常量组合)
// 返回: [error] nil=成功, 非nil=错误
func (s *Snap7Server) SetMask(maskKind int32, mask uint32) error {
	return s7err(C.Srv_SetMask(s.handle, C.int(maskKind), C.longword(mask)))
}

// EventText 获取事件文本描述
// 作用: 将事件对象转换为可读的文本描述
// 传入: [evt SrvEvent] 事件对象
// 返回: [string] 事件的可读文本描述
func (s *Snap7Server) EventText(evt SrvEvent) string {
	cEvt := C.TSrvEvent{
		EvtTime: C.time_t(evt.EvtTime.Unix()), EvtSender: C.int(evt.EvtSender),
		EvtCode: C.longword(evt.EvtCode), EvtRetCode: C.word(evt.EvtRetCode),
		EvtParam1: C.word(evt.EvtParam1), EvtParam2: C.word(evt.EvtParam2),
		EvtParam3: C.word(evt.EvtParam3), EvtParam4: C.word(evt.EvtParam4),
	}
	buf := make([]byte, 1024)
	C.Srv_EventText(&cEvt, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	return cStringToGo(&buf[0])
}

// SrvErrorText 服务端错误文本
// 作用: 将服务端错误码转换为可读文本
// 传入: [errCode int] 错误码(ErrSrvXxx 常量)
// 返回: [error] 错误对象(包含错误码和描述文本)
func SrvErrorText(errCode int32) error {
	return s7err(C.int(errCode))
}

//******************************************************************************
//                        Snap7Partner 伙伴函数
//******************************************************************************

// ParCreate 创建 Partner 实例
// 作用: 分配并返回一个新的 snap7 Partner 对象，用于 S7 点对点通信
// 传入: [active bool] true=主动连接方, false=被动等待方
// 返回: [*Snap7Partner] Partner 指针
func ParCreate(active bool) *Snap7Partner {
	cA := C.int(0)
	if active {
		cA = 1
	}
	return &Snap7Partner{handle: C.Par_Create(cA)}
}

// ParDestroy 销毁 Partner 实例
// 作用: 释放 Partner 占用的 C 资源
// 传入: [p *Snap7Partner] 待销毁的 Partner 指针
// 返回: 无
func ParDestroy(p *Snap7Partner) {
	if p != nil && p.handle != 0 {
		C.Par_Destroy(&p.handle)
		p.handle = 0
	}
}

// SetParam 设置 Partner 参数
// 作用: 设置 Partner 通信参数
// 传入: [paramNumber int] 参数编号: 12=BSend超时, 13=BRecv超时, 14=恢复时间, 15=保活时间, [value int] 参数值
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) SetParam(paramNumber int32, value int32) error {
	v := C.int(value)
	return s7err(C.Par_SetParam(p.handle, C.int(paramNumber), unsafe.Pointer(&v)))
}

// GetParam 获取 Partner 参数
// 作用: 读取 Partner 当前参数值
// 传入: [paramNumber int] 参数编号: 同 SetParam
// 返回: [int] 参数值, [err error] nil=成功, 非nil=错误
func (p *Snap7Partner) GetParam(paramNumber int32) (int32, error) {
	var v C.int
	err := s7err(C.Par_GetParam(p.handle, C.int(paramNumber), unsafe.Pointer(&v)))
	return int32(v), err
}

// Start 启动 Partner
// 作用: 启动 Partner 连接（使用已设参数）
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) Start() error { return s7err(C.Par_Start(p.handle)) }

// Stop 停止 Partner
// 作用: 停止 Partner 连接
// 传入: 无
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) Stop() error { return s7err(C.Par_Stop(p.handle)) }

// StartTo 启动 Partner（指定地址和 TSAP）
// 作用: 使用指定地址和 TSAP 建立 Partner 连接
// 传入: [localAddr string] 本地 IP, [remoteAddr string] 远程 IP, [localTSAP uint16] 本地 TSAP, [remoteTSAP uint16] 远程 TSAP
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) StartTo(localAddr, remoteAddr string, localTSAP, remoteTSAP uint16) error {
	cL := C.CString(localAddr)
	defer C.free(unsafe.Pointer(cL))
	cR := C.CString(remoteAddr)
	defer C.free(unsafe.Pointer(cR))
	return s7err(C.Par_StartTo(p.handle, cL, cR, C.word(localTSAP), C.word(remoteTSAP)))
}

// BSend 同步发送数据
// 作用: 向 Partner 对方同步发送数据块
// 传入: [rID uint32] 接收方 ID(对方标识), [data []byte] 待发送数据
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) BSend(rID uint32, data []byte) error {
	return s7err(C.Par_BSend(p.handle, C.longword(rID), unsafe.Pointer(&data[0]), C.int(len(data))))
}

// AsBSend 异步发送数据
// 作用: 向 Partner 对方异步发送数据块，需用 CheckAsBSendCompletion 等待完成
// 传入: [rID uint32] 接收方 ID, [data []byte] 待发送数据
// 返回: [error] nil=任务已提交, 非nil=错误
func (p *Snap7Partner) AsBSend(rID uint32, data []byte) error {
	return s7err(C.Par_AsBSend(p.handle, C.longword(rID), unsafe.Pointer(&data[0]), C.int(len(data))))
}

// CheckAsBSendCompletion 检查异步发送是否完成
// 作用: 非阻塞检查异步发送操作是否已完成
// 传入: 无
// 返回: [opResult int32] 操作结果: 0=已完成, 1=进行中, [err error] nil=成功, 非nil=错误码
func (p *Snap7Partner) CheckAsBSendCompletion() (opResult int32, err error) {
	var result C.int
	err = s7err(C.Par_CheckAsBSendCompletion(p.handle, &result))
	return int32(result), err
}

// WaitAsBSendCompletion 等待异步发送完成
// 作用: 阻塞等待异步发送操作完成或超时
// 传入: [timeout time.Duration] 最大等待时间
// 返回: [error] nil=成功, 非nil=错误
func (p *Snap7Partner) WaitAsBSendCompletion(timeout time.Duration) error {
	return s7err(C.Par_WaitAsBSendCompletion(p.handle, C.longword(timeout.Milliseconds())))
}

// BRecv 同步接收数据
// 作用: 阻塞等待并接收 Partner 对方发来的数据块
// 传入: [timeout time.Duration] 最大等待时间
// 返回: [rID uint32] 发送方 ID, [data []byte] 接收到的数据, [err error] nil=成功, 非nil=错误
func (p *Snap7Partner) BRecv(timeout time.Duration) (rID uint32, data []byte, err error) {
	var cRID C.longword
	buf := make([]byte, 65536)
	var cSize C.int
	err = s7err(C.Par_BRecv(p.handle, &cRID, unsafe.Pointer(&buf[0]), &cSize, C.longword(timeout.Milliseconds())))
	return uint32(cRID), buf[:int(cSize)], err
}

// CheckAsBRecvCompletion 检查异步接收是否完成
// 作用: 非阻塞检查异步接收操作是否已完成
// 传入: 无
// 返回: [opResult int32] 操作结果: 0=已完成, 1=进行中, [rID uint32] 发送方 ID, [data []byte] 接收到的数据, [err error] nil=成功, 非nil=错误码
func (p *Snap7Partner) CheckAsBRecvCompletion() (opResult int32, rID uint32, data []byte, err error) {
	var result C.int
	var cRID C.longword
	buf := make([]byte, 65536)
	var cSize C.int
	err = s7err(C.Par_CheckAsBRecvCompletion(p.handle, &result, &cRID, unsafe.Pointer(&buf[0]), &cSize))
	return int32(result), uint32(cRID), buf[:int(cSize)], err
}

// GetStatus 获取 Partner 状态
// 作用: 查询 Partner 当前运行状态
// 传入: 无
// 返回: [int32] 状态: 0=已停止, 1=连接中, 2=等待连接, 3=已连接, 4=发送中, 5=接收中, 6=绑定错误, [err error] nil=成功, 非nil=错误
func (p *Snap7Partner) GetStatus() (int32, error) {
	var status C.int
	C.Par_GetStatus(p.handle, &status)
	return int32(status), s7err(status)
}

// GetLastError 获取最后错误码
// 作用: 获取 Partner 最近一次操作的错误码
// 传入: 无
// 返回: [int] 错误码: 0=无错误, 非零=具体错误码(参见 ErrParXxx 常量)
func (p *Snap7Partner) GetLastError() int32 {
	var e C.int
	C.Par_GetLastError(p.handle, &e)
	return int32(e)
}

// GetTimes 获取通信时间
// 作用: 获取最近一次发送和接收的耗时
// 传入: 无
// 返回: [sendTime uint32] 发送耗时(毫秒), [recvTime uint32] 接收耗时(毫秒)
func (p *Snap7Partner) GetTimes() (sendTime, recvTime uint32) {
	var st, rt C.longword
	C.Par_GetTimes(p.handle, &st, &rt)
	return uint32(st), uint32(rt)
}

// GetStats 获取通信统计
// 作用: 获取 Partner 的发送/接收字节数和错误数统计
// 传入: 无
// 返回: [bytesSent uint32] 已发送字节数, [bytesRecv uint32] 已接收字节数, [sendErrors uint32] 发送错误数, [recvErrors uint32] 接收错误数
func (p *Snap7Partner) GetStats() (bytesSent, bytesRecv, sendErrors, recvErrors uint32) {
	var bs, br, se, re C.longword
	C.Par_GetStats(p.handle, &bs, &br, &se, &re)
	return uint32(bs), uint32(br), uint32(se), uint32(re)
}

// ParErrorText Partner 错误文本
// 作用: 将 Partner 错误码转换为可读文本
// 传入: [errCode int] 错误码(ErrParXxx 常量)
// 返回: [error] 错误对象(包含错误码和描述文本)
func ParErrorText(errCode int32) error {
	return s7err(C.int(errCode))
}

//******************************************************************************
//                        错误文本 & 辅助
//******************************************************************************

// CliErrorText 客户端错误文本
// 作用: 将客户端错误码转换为可读文本
// 传入: [errCode int] 错误码(ErrCliXxx/ErrIsoXxx 常量)
// 返回: [error] 错误对象(包含错误码和描述文本)
func CliErrorText(errCode int32) error {
	return s7err(C.int(errCode))
}

// cBlockInfoToGo C 块信息转 Go 结构体
// 作用: 将 C 层的 TS7BlockInfo 转换为 Go 层的 S7BlockInfo
// 传入: [cInfo *C.TS7BlockInfo] C 层块信息指针
// 返回: [S7BlockInfo] Go 层块信息结构体
func cBlockInfoToGo(cInfo *C.TS7BlockInfo) S7BlockInfo {
	return S7BlockInfo{
		BlkType: int32(cInfo.BlkType), BlkNumber: int32(cInfo.BlkNumber),
		BlkLang: int32(cInfo.BlkLang), BlkFlags: int32(cInfo.BlkFlags),
		MC7Size: int32(cInfo.MC7Size), LoadSize: int32(cInfo.LoadSize),
		LocalData: int32(cInfo.LocalData), SBBLength: int32(cInfo.SBBLength),
		CheckSum: int32(cInfo.CheckSum), Version: int32(cInfo.Version),
		CodeDate: C.GoString(&cInfo.CodeDate[0]), IntfDate: C.GoString(&cInfo.IntfDate[0]),
		Author: C.GoString(&cInfo.Author[0]), Family: C.GoString(&cInfo.Family[0]),
		Header: C.GoString(&cInfo.Header[0]),
	}
}

// ErrorText 将 snap7 错误码转为可读字符串
// 作用: 将客户端错误码转换为可读文本（CliErrorText 的导出别名）
// 传入: [errCode int] 错误码(ErrCliXxx/ErrIsoXxx 常量)
// 返回: [error] 错误对象(包含错误码和描述文本)
func ErrorText(errCode int32) error {
	return CliErrorText(errCode)
}

// cStringToGo C 字符串转 Go 字符串
// 作用: 将 C 层的以 0 结尾的字节数组转换为 Go 字符串
// 传入: [buf *byte] C 层字符串首字节指针
// 返回: [string] Go 字符串
func cStringToGo(buf *byte) string {
	b := (*[1024]byte)(unsafe.Pointer(buf))
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b[:])
}
