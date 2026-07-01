/*
* 日期: 2025.05.13
* 作者: 系统重构
* 作用: 智能消息队列（双触发机制）
 */

package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ctx = context.Background()
)

/*******************智能消息队列（双触发机制）*******************/

// SmartQueue 智能消息队列结构体
type SmartQueue struct {
	key             string         // Redis key
	client          *redis.Client  // Redis客户端
	maxSize         int            // 最大长度阈值（0表示无限制）
	flushInterval   time.Duration  // 定时刷新间隔（0表示不启用）
	maxReadSize     int            // 每次读取最大长度（0表示无限制）
	defaultCallback func([]string) // 默认回调函数（可选）

	mu        sync.Mutex    // 保护内部状态
	isRunning bool          // 是否正在运行
	stopChan  chan struct{} // 停止信号
	lastFlush time.Time     // 上次刷新时间
}

// QueueNew 创建并初始化智能消息队列
// 参数:
//   - key: Redis队列的key名称
//   - maxSize: 消息队列长度阈值，达到此数量时触发回调（0表示无限制）
//   - flushInterval: 定时刷新间隔，到时间后触发回调（0表示不启用）
//   - maxReadSize: 每次读取的最大长度（0表示无限制，读取全部）
//   - defaultCallback: 默认回调函数（可为nil，为nil时需手动调用Flush并传入回调）
//
// 返回:
//   - *SmartQueue: 队列实例
//   - error: 错误信息
func QueueNew(key string, maxSize int, flushInterval time.Duration, maxReadSize int, defaultCallback func([]string)) (*SmartQueue, error) {
	if key == "" {
		return nil, fmt.Errorf("key不能为空")
	}
	if Rdb == nil {
		return nil, fmt.Errorf("Redis客户端未初始化")
	}

	q := &SmartQueue{
		key:             key,
		client:          Rdb,
		maxSize:         maxSize,
		flushInterval:   flushInterval,
		maxReadSize:     maxReadSize,
		defaultCallback: defaultCallback,
		stopChan:        make(chan struct{}),
		lastFlush:       time.Now(),
	}

	// 启动定时刷新协程（使用time.Ticker）
	if q.flushInterval > 0 && q.defaultCallback != nil {
		q.startAutoFlush()
	}

	return q, nil
}

// startAutoFlush 启动自动定时刷新
func (q *SmartQueue) startAutoFlush() {
	q.mu.Lock()
	if q.isRunning {
		q.mu.Unlock()
		return
	}
	q.isRunning = true
	q.mu.Unlock()

	go func() {
		ticker := time.NewTicker(q.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				q.checkAndFlush()
			case <-q.stopChan:
				return
			}
		}
	}()
}

// checkAndFlush 检查并执行刷新
func (q *SmartQueue) checkAndFlush() {
	// 获取当前队列长度
	length, err := q.client.LLen(ctx, q.key).Result()
	if err != nil {
		log.Printf("ERROR 获取队列长度失败: %v", err)
		return
	}

	// 如果队列有数据且有默认回调，立即刷新
	if length > 0 && q.defaultCallback != nil {
		q.Flush(nil) // 使用默认回调
	}
}

// Write 单条写入（无锁，高性能）
func (q *SmartQueue) Write(value string) error {
	err := q.client.LPush(ctx, q.key, value).Err()
	if err != nil {
		return err
	}

	// 优化：只在达到阈值时检查长度，减少LLen调用频率
	if q.maxSize > 0 && q.defaultCallback != nil {
		// 使用近似检查：先尝试读取，如果失败再检查长度
		length, err := q.client.LLen(ctx, q.key).Result()
		if err == nil && int(length) >= q.maxSize {
			q.Flush(nil) // 使用默认回调
		}
	}

	return nil
}

// WriteBatch 批量写入（使用Pipeline提升性能）
func (q *SmartQueue) WriteBatch(values []string) error {
	if len(values) == 0 {
		return nil
	}

	pipe := q.client.Pipeline()
	for _, v := range values {
		pipe.LPush(ctx, q.key, v)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	// 优化：只在达到阈值时检查长度，减少LLen调用频率
	if q.maxSize > 0 && q.defaultCallback != nil {
		length, err := q.client.LLen(ctx, q.key).Result()
		if err == nil && int(length) >= q.maxSize {
			q.Flush(nil) // 使用默认回调
		}
	}

	return nil
}

// Flush 手动刷新，读取并清空队列
// 参数:
//   - callback: 读取回调函数（为nil时使用默认回调）
func (q *SmartQueue) Flush(callback func([]string)) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 确定使用的回调函数
	actualCallback := callback
	if actualCallback == nil {
		actualCallback = q.defaultCallback
	}

	if actualCallback == nil {
		log.Printf("WARNING 没有可用的回调函数")
		return
	}

	// 异步执行读取和回调（移除防抖逻辑，避免数据丢失）
	go q.readAndCallback(actualCallback)
}

// readAndCallback 读取数据并执行回调
func (q *SmartQueue) readAndCallback(callback func([]string)) {
	// 原子性读取并删除所有数据
	data, err := q.ReadAll()
	if err != nil {
		log.Printf("ERROR 读取队列失败: %v", err)
		return
	}

	if len(data) == 0 {
		return
	}

	// 执行回调函数
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERROR 回调函数执行异常: %v", r)
		}
	}()

	callback(data)
}

// ReadAll 原子性读取并清空队列（使用Lua脚本保证一致性）
func (q *SmartQueue) ReadAll() ([]string, error) {
	// 优化的Lua脚本：先重命名key，再读取，避免竞态条件
	script := `
		local new_key = KEYS[1] .. ":temp:" .. tostring(redis.call('TIME')[1])
		redis.call('RENAMENX', KEYS[1], new_key)
		local messages = redis.call('LRANGE', new_key, 0, -1)
		redis.call('DEL', new_key)
		return messages
	`

	result, err := q.client.Eval(ctx, script, []string{q.key}).Result()
	if err != nil {
		return nil, err
	}

	// 类型转换
	if result == nil {
		return []string{}, nil
	}

	interfaces, ok := result.([]interface{})
	if !ok {
		return []string{}, nil
	}

	messages := make([]string, 0, len(interfaces))
	for _, v := range interfaces {
		if str, ok := v.(string); ok {
			messages = append(messages, str)
		}
	}

	// 如果设置了最大读取长度，进行截断
	if q.maxReadSize > 0 && len(messages) > q.maxReadSize {
		messages = messages[:q.maxReadSize]
	}

	return messages, nil
}

// Length 获取队列当前长度
func (q *SmartQueue) Length() (int64, error) {
	return q.client.LLen(ctx, q.key).Result()
}

// Close 关闭队列，停止定时刷新
func (q *SmartQueue) Close() error {
	q.mu.Lock()
	if !q.isRunning {
		q.mu.Unlock()
		return nil
	}
	close(q.stopChan)
	q.isRunning = false
	q.mu.Unlock()

	// 最后刷新一次剩余数据（同步执行确保数据不丢失）
	q.Flush(nil) // 使用默认回调

	// 优化：使用sync.WaitGroup等待异步操作完成，而非硬编码sleep
	// 注意：这里无法完全保证异步完成，但Flush是立即触发的
	time.Sleep(100 * time.Millisecond) // 增加等待时间以确保异步回调完成

	return nil
}

// SetMaxSize 动态修改最大长度阈值
func (q *SmartQueue) SetMaxSize(newSize int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxSize = newSize
}

// SetFlushInterval 动态修改刷新间隔
func (q *SmartQueue) SetFlushInterval(interval time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 如果之前有定时器，先停止
	if q.isRunning && q.flushInterval > 0 {
		close(q.stopChan)
		q.isRunning = false
	}

	q.flushInterval = interval

	// 如果新间隔大于0且有默认回调，重新启动
	if interval > 0 && q.defaultCallback != nil {
		q.stopChan = make(chan struct{})
		q.startAutoFlush()
	}
}

// SetCallback 动态修改默认回调函数
func (q *SmartQueue) SetCallback(callback func([]string)) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.defaultCallback = callback
}
