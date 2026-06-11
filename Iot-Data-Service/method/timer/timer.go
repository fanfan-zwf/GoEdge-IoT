package timer

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// TimerTask 定时任务管理器
type TimerTask struct {
	ticker    *time.Ticker
	stopCh    chan struct{}
	running   bool
	mu        sync.Mutex
	lastRunAt time.Time // 记录上一次执行时间
}

// NewTimerTask 创建定时任务实例
func NewTimerTask() *TimerTask {
	return &TimerTask{
		stopCh: make(chan struct{}),
	}
}

// Start 启动/重启定时任务
// duration: 执行间隔
// task: 业务回调，参数为本次触发的时间
// 多次调用会终止旧任务，以最后一次调用为准
func (t *TimerTask) Start(duration time.Duration, task func(callTime time.Time)) {
	t.mu.Lock()

	// 正在运行则先停止旧任务
	if t.running {
		close(t.stopCh)
		t.running = false
		t.ticker.Stop()
		log.Println("重复调用 Start，终止旧任务，准备启动新任务")
	}

	// 重置资源
	t.stopCh = make(chan struct{})
	t.running = true
	t.ticker = time.NewTicker(duration)
	t.mu.Unlock()

	log.Println("定时任务已启动")

	go func() {
		defer func() {
			t.mu.Lock()
			t.running = false
			t.ticker.Stop()
			t.mu.Unlock()
			log.Println("定时任务协程已退出")
		}()

		for {
			select {
			case now := <-t.ticker.C:
				// 执行回调，传入本次调用时间
				task(now)
				// 更新最后执行时间
				t.mu.Lock()
				t.lastRunAt = now
				t.mu.Unlock()

			case <-t.stopCh:
				fmt.Println("收到停止信号，任务结束")
				return
			}
		}
	}()
}

// Stop 停止任务，支持重复调用，幂等安全
func (t *TimerTask) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		log.Println("任务未运行，重复 Stop 已忽略")
		return
	}
	close(t.stopCh)
}

// IsRunning 返回任务当前运行状态
func (t *TimerTask) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// GetLastRunTime 获取上一次任务执行时间
func (t *TimerTask) GetLastRunTime() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastRunAt
}

// 测试示例
func main() {
	task := NewTimerTask()

	// 第一次启动：间隔 1s，回调接收调用时间
	fmt.Println("=== 第一次 Start ===")
	task.Start(1*time.Second, func(callTime time.Time) {
		fmt.Printf("任务执行，调用时间: %s\n", callTime.Format("2006-01-02 15:04:05"))
	})
	time.Sleep(2 * time.Second)

	// 连续第二次 Start，覆盖旧任务（以最后一次为准）
	fmt.Println("\n=== 第二次 Start（覆盖旧任务）===")
	task.Start(2*time.Second, func(callTime time.Time) {
		fmt.Printf("新任务执行，调用时间: %s\n", callTime.Format("2006-01-02 15:04:05"))
	})
	time.Sleep(4 * time.Second)

	// 连续多次 Stop
	fmt.Println("\n=== 第一次 Stop ===")
	task.Stop()
	fmt.Println("=== 第二次 Stop ===")
	task.Stop()

	time.Sleep(1 * time.Second)
	fmt.Println("程序结束")
}
