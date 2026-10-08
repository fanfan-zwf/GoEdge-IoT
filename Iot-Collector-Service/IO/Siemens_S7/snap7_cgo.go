/*
* snap7 CGO 绑定层 — 封装 snap7 C 动态库，提供 Go 友好的 API
* 依赖: lib/libsnap7.so (编译自 https://github.com/davenardella/snap7)
 */
package siemenss7

/*
#cgo CFLAGS: -std=c11 -I${SRCDIR}/../../lib
#cgo LDFLAGS: -L${SRCDIR}/../../lib -lsnap7 -Wl,-rpath,$ORIGIN/lib
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
	"fmt"
	"unsafe"
)

// snap7 常量（对应 snap7.h 中的定义）
const (
	S7AreaPE = 0x81 // 输入
	S7AreaPA = 0x82 // 输出
	S7AreaMK = 0x83 // 标志位
	S7AreaDB = 0x84 // DB数据块
	S7AreaCT = 0x1C // 计数器
	S7AreaTM = 0x1D // 定时器

	S7WLBit     = 0x01 // 位
	S7WLByte    = 0x02 // 字节
	S7WLWord    = 0x04 // 字
	S7WLDWord   = 0x06 // 双字
	S7WLReal    = 0x08 // 浮点
	S7WLCounter = 0x1C // 计数器
	S7WLTimer   = 0x1D // 定时器

	ConnTypePG    = 1 // PG 编程设备
	ConnTypeOP    = 2 // OP 操作面板
	ConnTypeBasic = 3 // Basic 连接

	// snap7 参数编号
	ParamSendTimeout  = 4  // 发送超时 (ms)
	ParamRecvTimeout  = 5  // 接收超时 (ms)
	ParamWorkInterval = 6  // 工作间隔 (ms)
	ParamPDURequest   = 10 // 请求 PDU 长度
)

// S7DataItem Go 版读写项（内存布局与 C.TS7DataItem 一致）
type S7DataItem struct {
	Area     int
	WordLen  int
	Result   int
	DBNumber int
	Start    int
	Amount   int
	Data     []byte // Go 管理的缓冲区，调用前填充，调用后读取结果
}

// Snap7Client snap7 C 客户端的 Go 封装
type Snap7Client struct {
	client C.S7Object // snap7 客户端句柄
}

// NewSnap7Client 创建 snap7 客户端实例
func NewSnap7Client() *Snap7Client {
	return &Snap7Client{client: C.Cli_Create()}
}

// Destroy 释放客户端资源
func (c *Snap7Client) Destroy() {
	if c.client != 0 {
		C.Cli_Destroy(&c.client)
		c.client = 0
	}
}

// SetConnectionType 设置连接类型（必须在 Connect 前调用）
func (c *Snap7Client) SetConnectionType(connType int) int {
	return int(C.Cli_SetConnectionType(c.client, C.word(connType)))
}

// ConnectTo 一步连接到 PLC（自动计算 TSAP）
func (c *Snap7Client) ConnectTo(address string, rack, slot int) int {
	cAddr := C.CString(address)
	defer C.free(unsafe.Pointer(cAddr))
	return int(C.Cli_ConnectTo(c.client, cAddr, C.int(rack), C.int(slot)))
}

// SetConnectionParams 设置连接参数（自定义 TSAP 时使用）
func (c *Snap7Client) SetConnectionParams(address string, localTSAP, remoteTSAP uint16) int {
	cAddr := C.CString(address)
	defer C.free(unsafe.Pointer(cAddr))
	return int(C.Cli_SetConnectionParams(c.client, cAddr, C.word(localTSAP), C.word(remoteTSAP)))
}

// Connect 建立连接（需先调用 SetConnectionParams 或 SetConnectionType）
func (c *Snap7Client) Connect() int {
	return int(C.Cli_Connect(c.client))
}

// Disconnect 断开连接
func (c *Snap7Client) Disconnect() int {
	return int(C.Cli_Disconnect(c.client))
}

// SetParam 设置参数（int 类型）
func (c *Snap7Client) SetParam(paramNumber int, value int) int {
	v := C.int(value)
	return int(C.Cli_SetParam(c.client, C.int(paramNumber), unsafe.Pointer(&v)))
}

// GetPduLength 获取协商后的 PDU 长度
func (c *Snap7Client) GetPduLength() (requested int, negotiated int) {
	var req, neg C.int
	C.Cli_GetPduLength(c.client, &req, &neg)
	return int(req), int(neg)
}

// ReadArea 读取指定区域的数据
func (c *Snap7Client) ReadArea(area, dbNumber, start, amount, wordLen int, buf []byte) int {
	return int(C.Cli_ReadArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// WriteArea 向指定区域写入数据
func (c *Snap7Client) WriteArea(area, dbNumber, start, amount, wordLen int, buf []byte) int {
	return int(C.Cli_WriteArea(c.client, C.int(area), C.int(dbNumber), C.int(start),
		C.int(amount), C.int(wordLen), unsafe.Pointer(&buf[0])))
}

// ReadMultiVars 批量读取多个变量（对应 AGReadMulti）
func (c *Snap7Client) ReadMultiVars(items []S7DataItem) int {
	count := len(items)
	if count == 0 {
		return 0
	}

	// 分配 C 数组（使用 PS7DataItem 确保类型匹配）
	cItems := C.PS7DataItem(C.malloc(C.size_t(count) * C.sizeof_TS7DataItem))
	defer C.free(unsafe.Pointer(cItems))

	// Go → C 转换，为每个 item 分配数据缓冲区
	cBufs := make([]unsafe.Pointer, count)
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		ptr.Area = C.int(items[i].Area)
		ptr.WordLen = C.int(items[i].WordLen)
		ptr.DBNumber = C.int(items[i].DBNumber)
		ptr.Start = C.int(items[i].Start)
		ptr.Amount = C.int(items[i].Amount)
		// 分配 C 缓冲区：使用 Data 切片长度（而非 Amount），因为 Timer/Counter 的 Amount=1 但返回 2 字节
		bufLen := len(items[i].Data)
		if bufLen <= 0 {
			bufLen = items[i].Amount
		}
		buf := C.malloc(C.size_t(bufLen))
		cBufs[i] = buf
		ptr.pdata = buf
	}

	// 调用 snap7
	result := C._Cli_ReadMultiVars(c.client, unsafe.Pointer(cItems), C.int(count))

	// 回读结果到 Go 结构体
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		items[i].Result = int(ptr.Result)
		if items[i].Result == 0 && items[i].Data != nil {
			// 拷贝读取结果到 Go 缓冲区，取 C 缓冲区和 Go Data 的最小值防止溢出
			copyLen := len(items[i].Data)
			if C.int(copyLen) > ptr.Amount {
				// snap7 实际写入的数据量由 Amount 和 WordLen 决定
				// 对于 S7WLByte，Amount 就是字节数；对于 Timer/Counter，Amount=1 但返回 2 字节
				if items[i].WordLen == 0x1D || items[i].WordLen == 0x1C {
					// Timer/Counter: 固定 2 字节
					if copyLen > 2 {
						copyLen = 2
					}
				} else {
					// 普通区域: Amount 就是字节数
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

	return int(result)
}

// WriteMultiVars 批量写入多个变量（对应 AGWriteMulti）
func (c *Snap7Client) WriteMultiVars(items []S7DataItem) int {
	count := len(items)
	if count == 0 {
		return 0
	}

	// 分配 C 数组（使用 PS7DataItem 确保类型匹配）
	cItems := C.PS7DataItem(C.malloc(C.size_t(count) * C.sizeof_TS7DataItem))
	defer C.free(unsafe.Pointer(cItems))

	// Go → C 转换，pdata 直接指向 Go 切片的底层数组
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

	// 调用 snap7
	result := C._Cli_WriteMultiVars(c.client, unsafe.Pointer(cItems), C.int(count))

	// 回读每个 item 的结果
	for i := range items {
		ptr := (*C.TS7DataItem)(unsafe.Pointer(uintptr(unsafe.Pointer(cItems)) + uintptr(i)*C.sizeof_TS7DataItem))
		items[i].Result = int(ptr.Result)
	}

	return int(result)
}

// GetConnected 检查是否已连接
func (c *Snap7Client) GetConnected() bool {
	var connected C.int
	C.Cli_GetConnected(c.client, &connected)
	return connected != 0
}

// ErrorText 将 snap7 错误码转为可读字符串
func snap7ErrorText(errCode int) string {
	buf := make([]byte, 1024)
	C.Cli_ErrorText(C.int(errCode), (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	// 截取到第一个 \0
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

// snap7Error 将 snap7 返回码转为 Go error
func snap7Error(code int) error {
	if code == 0 {
		return nil
	}
	return fmt.Errorf("snap7: %s (0x%08X)", snap7ErrorText(code), code)
}
