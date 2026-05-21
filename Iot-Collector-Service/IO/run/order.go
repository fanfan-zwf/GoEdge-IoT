package run

import (
	"sync"
)

type MsgBroker struct {
	mu          sync.RWMutex
	subscribers []Subscriber
	statusMap   map[uint]string // id -> status
}

type Subscriber struct {
	ID       uint
	Msg      string
	Callback func(id uint, msg string) // 回调带回 id+msg
}

func NewMsgBroker() *MsgBroker {
	return &MsgBroker{
		statusMap: make(map[uint]string),
	}
}

// OnMsg 订阅：id=0 全局，否则只收对应id
func (b *MsgBroker) OnMsg(id uint, expectMsg string, callback func(id uint, msg string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = append(b.subscribers, Subscriber{
		ID:       id,
		Msg:      expectMsg,
		Callback: callback,
	})
}

// Send(0,msg)全局广播；Send(id,msg)发给全局+指定id
// 【关键】发送时自动更新状态：id→msg
func (b *MsgBroker) Send(sendID uint, msg string) {
	// 1. 更新状态：sendID 的状态设为 msg
	b.RegisterStatus(sendID, msg)

	b.mu.RLock()
	defer b.mu.RUnlock()

	var matches []func(id uint, msg string)
	for _, sub := range b.subscribers {
		if sub.Msg != msg {
			continue
		}
		if sendID == 0 || sub.ID == 0 || sub.ID == sendID {
			matches = append(matches, sub.Callback)
		}
	}

	// 阻塞等所有回调跑完
	var wg sync.WaitGroup
	for _, cb := range matches {
		wg.Add(1)
		go func(f func(uint, string)) {
			defer wg.Done()
			f(sendID, msg)
		}(cb)
	}
	wg.Wait()
}

// RegisterStatus 手动注册/覆盖状态
func (b *MsgBroker) RegisterStatus(id uint, status string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.statusMap[id] = status
}

// GetStatus 查单个id状态
func (b *MsgBroker) GetStatus(id uint) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.statusMap[id]
}

// GetAllStatus 查全部状态副本
func (b *MsgBroker) GetAllStatus() map[uint]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	res := make(map[uint]string, len(b.statusMap))
	for k, v := range b.statusMap {
		res[k] = v
	}
	return res
}

// ===================== 使用示例 =====================
// func main() {
// 	broker := NewMsgBroker()

// 	// 全局订阅：所有 Send 都能收到
// 	broker.OnMsg(0, "启动", func(id uint, msg string) {
// 		fmt.Printf("全局收到：id=%d, msg=%s\n", id, msg)
// 	})

// 	// 只订阅 id=100 的"启动"
// 	broker.OnMsg(100, "启动", func(id uint, msg string) {
// 		fmt.Printf("订阅100收到：id=%d, msg=%s\n", id, msg)
// 	})

// 	fmt.Println("--- Send(0, 启动) ---")
// 	broker.Send(0, "启动") // 全局触发，状态：0→启动

// 	fmt.Println("\n--- Send(100, 启动) ---")
// 	broker.Send(100, "启动") // 全局+100触发，状态：100→启动

// 	fmt.Println("\n--- 所有状态 ---")
// 	for id, s := range broker.GetAllStatus() {
// 		fmt.Println(id, "→", s)
// 	}
// }

// 运行效果
// === Send(0, 启动) ===
// 【全局0】收到启动
// 【ID=100】收到启动
// 【ID=200】收到启动

// === 所有状态 ===
// ID:100 → 状态:运行中
// ID:200 → 状态:待机
// ID:300 → 状态:故障

// === 单个状态 ===
// ID:100 → 运行中

// 初始化 重启 重新加载 启动 停止
