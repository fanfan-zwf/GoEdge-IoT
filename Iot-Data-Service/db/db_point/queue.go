package db_point

import (
	"sync"
	"time"
)

// MessageQueue 泛型消息队列
type MessageQueue[T any] struct {
	mu          sync.Mutex
	data        []T           // 存储数据的切片
	maxSize     int           // 最大长度，达到后触发回调
	callback    func([]T)     // 回调函数
	autoFlush   bool          // 是否自动刷新
	flushTicker *time.Ticker  // 定时刷新器
	stopChan    chan struct{} // 停止信号
}

// NewMessageQueue 初始化消息队列
// maxSize: 队列最大长度，达到后自动触发回调 『5000个占内存~1.5 MB』
// callback: 处理数据的回调函数
// autoFlushInterval: 自动刷新间隔（0表示不自动刷新）
func NewMessageQueue[T any](maxSize int, callback func([]T), autoFlushInterval time.Duration) *MessageQueue[T] {
	if maxSize <= 0 {
		maxSize = 100 // 默认100
	}

	q := &MessageQueue[T]{
		data:      make([]T, 0, maxSize),
		maxSize:   maxSize,
		callback:  callback,
		autoFlush: autoFlushInterval > 0,
		stopChan:  make(chan struct{}),
	}

	// 启动定时刷新
	if q.autoFlush {
		q.flushTicker = time.NewTicker(autoFlushInterval)
		go q.autoFlushLoop()
	}

	return q
}

// Push 写入数据（支持单个或多个）
func (q *MessageQueue[T]) Push(values ...T) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 添加数据
	q.data = append(q.data, values...)

	// 检查是否达到最大长度
	if len(q.data) >= q.maxSize {
		q.flushLocked()
	}
}

// Flush 手动刷新，立即执行回调
func (q *MessageQueue[T]) Flush() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.flushLocked()
}

// flushLocked 内部刷新（需要持有锁）
func (q *MessageQueue[T]) flushLocked() {
	if len(q.data) == 0 {
		return
	}

	// 复制数据
	batch := make([]T, len(q.data))
	copy(batch, q.data)

	// 清空队列
	q.data = q.data[:0]

	// 执行回调（在锁外执行，避免死锁）
	go q.safeCallback(batch)
}

// safeCallback 安全执行回调
func (q *MessageQueue[T]) safeCallback(batch []T) {
	defer func() {
		if r := recover(); r != nil {
			// 防止回调 panic 导致程序崩溃
			// 可以在这里记录日志
			// log.Printf("回调执行异常: %v", r)
		}
	}()
	q.callback(batch)
}

// autoFlushLoop 自动刷新循环
func (q *MessageQueue[T]) autoFlushLoop() {
	for {
		select {
		case <-q.flushTicker.C:
			q.Flush()
		case <-q.stopChan:
			return
		}
	}
}

// Close 关闭队列，释放资源
func (q *MessageQueue[T]) Close() {
	if q.flushTicker != nil {
		q.flushTicker.Stop()
	}
	close(q.stopChan)

	// 最后刷新一次
	q.Flush()
}

// Len 获取当前队列长度
func (q *MessageQueue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.data)
}

// IsEmpty 判断队列是否为空
func (q *MessageQueue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.data) == 0
}

// SetCallback 动态修改回调函数
func (q *MessageQueue[T]) SetCallback(callback func([]T)) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.callback = callback
}

// SetMaxSize 动态修改最大长度（立即生效）
func (q *MessageQueue[T]) SetMaxSize(newSize int) {
	if newSize <= 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxSize = newSize

	// 如果新长度小于当前数据长度，立即刷新
	if len(q.data) >= q.maxSize {
		q.flushLocked()
	}
}

// func main() {
// 	// 创建消息队列，最大长度3，回调打印数据
// 	queue := NewMessageQueue[int](3, func(batch []int) {
// 		fmt.Printf("处理批量数据: %v\n", batch)
// 	}, 0) // 0表示不自动刷新

// 	// 逐个推送数据
// 	for i := 1; i <= 10; i++ {
// 		queue.Push(i)
// 		fmt.Printf("推送: %d, 当前长度: %d\n", i, queue.Len())
// 		time.Sleep(100 * time.Millisecond)
// 	}

// 	// 手动刷新剩余数据
// 	queue.Flush()
// 	queue.Close()
// }
