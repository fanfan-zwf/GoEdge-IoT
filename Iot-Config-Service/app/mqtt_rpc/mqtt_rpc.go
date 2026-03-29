package mqtt_rpc

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ====================== 【正确】消息结构 ======================
type RequestID string

type RpcMessage struct {
	ReqID      RequestID `json:"req_id"`      // 唯一ID，防冲突
	ReplyTopic string    `json:"reply_topic"` // ✅ 回复主题（必须）
	BizTopic   string    `json:"biz_topic"`   // 业务接口名
	Payload    []byte    `json:"payload"`     // 业务数据
}

func NewReqID() RequestID {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return RequestID(fmt.Sprintf("%x", b))
}

// ====================== MQTT 配置 ======================
type MqttConfig struct {
	Broker      string
	Username    string
	Password    string
	ClientID    string
	ListenTopic string // 我订阅的主题（别人发给我）
}

// ====================== MQTT 实例 ======================
type BizHandler func(req []byte) ([]byte, error)

type mqttInstance struct {
	client      mqtt.Client
	listenTopic string
	handlers    map[string]BizHandler
	hMutex      sync.RWMutex
	waiters     map[RequestID]chan []byte
	wMutex      sync.Mutex
}

// ====================== MQTT 管理器（map 按名称管理）======================
type MqttManager struct {
	inst map[string]*mqttInstance
	mu   sync.RWMutex
}

func NewMqttManager() *MqttManager {
	return &MqttManager{
		inst: make(map[string]*mqttInstance),
	}
}

// 添加MQTT连接（指定名称）
func (m *MqttManager) Add(name string, cfg MqttConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	inst := &mqttInstance{
		client:      client,
		listenTopic: cfg.ListenTopic,
		handlers:    make(map[string]BizHandler),
		waiters:     make(map[RequestID]chan []byte),
	}

	// 订阅自己的主题
	client.Subscribe(cfg.ListenTopic, 1, inst.onMsg)
	m.inst[name] = inst
	fmt.Println("已连接MQTT:", name, " 监听主题:", cfg.ListenTopic)
	return nil
}

// 收到消息
func (inst *mqttInstance) onMsg(c mqtt.Client, msg mqtt.Message) {
	var m RpcMessage
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		return
	}

	// 1. 若是响应 → 交给等待的Call
	inst.wMutex.Lock()
	ch, ok := inst.waiters[m.ReqID]
	if ok {
		delete(inst.waiters, m.ReqID)
	}
	inst.wMutex.Unlock()
	if ok {
		ch <- m.Payload
		return
	}

	// 2. 若是请求 → 执行业务
	inst.hMutex.RLock()
	fn, exists := inst.handlers[m.BizTopic]
	inst.hMutex.RUnlock()
	if !exists {
		return
	}

	// 处理并回复
	respData, _ := fn(m.Payload)
	respMsg := RpcMessage{
		ReqID:      m.ReqID,
		ReplyTopic: m.ReplyTopic,
		Payload:    respData,
	}
	respBytes, _ := json.Marshal(respMsg)
	inst.client.Publish(m.ReplyTopic, 1, false, respBytes).Wait()
}

// ====================== 注册业务（指定MQTT名称）======================
func (m *MqttManager) Register(mqttName string, bizTopic string, fn BizHandler) error {
	m.mu.RLock()
	inst, ok := m.inst[mqttName]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("mqtt %s not found", mqttName)
	}
	inst.hMutex.Lock()
	defer inst.hMutex.Unlock()
	inst.handlers[bizTopic] = fn
	return nil
}

// ====================== 调用（指定MQTT + 目标主题）======================
func (m *MqttManager) Call(
	mqttName string,
	targetTopic string, // 对方订阅的主题
	bizTopic string,
	req []byte,
	timeout time.Duration,
) ([]byte, error) {

	m.mu.RLock()
	inst, ok := m.inst[mqttName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mqtt %s not found", mqttName)
	}

	reqID := NewReqID()

	// ✅ 关键：消息里带上自己的监听主题（回复主题）
	msg := RpcMessage{
		ReqID:      reqID,
		ReplyTopic: inst.listenTopic, // ✅ 我在哪收响应
		BizTopic:   bizTopic,
		Payload:    req,
	}
	reqBytes, _ := json.Marshal(msg)

	ch := make(chan []byte, 1)

	inst.wMutex.Lock()
	inst.waiters[reqID] = ch
	inst.wMutex.Unlock()

	// 超时
	go func() {
		time.Sleep(timeout)
		inst.wMutex.Lock()
		delete(inst.waiters, reqID)
		inst.wMutex.Unlock()
		ch <- nil
	}()

	// 发给对方主题
	inst.client.Publish(targetTopic, 1, false, reqBytes).Wait()

	resp := <-ch
	if resp == nil {
		return nil, fmt.Errorf("timeout")
	}
	return resp, nil
}

func (m *MqttManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.inst {
		inst.client.Disconnect(200)
	}
}

// ====================== 测试 ======================
// func main() {
// 	m := NewMqttManager()
// 	defer m.Close()

// 	// 连接 MQTT1
// 	m.Add("mqtt1", MqttConfig{
// 		Broker:      "tcp://127.0.0.1:1883",
// 		ClientID:    "dev1",
// 		ListenTopic: "/device/1",
// 	})

// 	// 连接 MQTT2
// 	m.Add("mqtt2", MqttConfig{
// 		Broker:      "tcp://127.0.0.1:1884",
// 		ClientID:    "dev2",
// 		ListenTopic: "/device/2",
// 	})

// 	// 注册服务
// 	m.Register("mqtt1", "test.hello", func(req []byte) ([]byte, error) {
// 		fmt.Println("收到:", string(req))
// 		return []byte("hello from mqtt1"), nil
// 	})

// 	// 调用：从 mqtt2 发给 mqtt1 的主题
// 	resp, _ := m.Call("mqtt2", "/device/1", "test.hello", []byte("hi"), 3*time.Second)
// 	fmt.Println("收到响应:", string(resp))

// 	select {}
// }
