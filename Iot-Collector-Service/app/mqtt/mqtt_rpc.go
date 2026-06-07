package mqtt

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"main/Init"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// ====================== 安全 JSON 载体：永不报错 ======================
type SafePayload []byte

// 序列化：空时自动返回合法的 JSON {}，不会断裂
func (s SafePayload) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("{}"), nil
	}
	return s, nil
}

// 反序列化：兼容空、null、正常数据
func (s *SafePayload) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*s = []byte("{}")
		return nil
	}
	*s = data
	return nil
}

// 方便使用：转成普通 []byte
func (s SafePayload) Bytes() []byte {
	return []byte(s)
}

var timeoutErr = fmt.Errorf("ERROR 请求超时")

// ====================== 消息结构 ======================
type RequestID string

type RpcMessage struct {
	ReqID      RequestID   `json:"reqId"`
	ReplyTopic string      `json:"replyTopic"`
	BizTopic   string      `json:"bizTopic"`
	Payload    SafePayload `json:"payload"`
	Uuid       string      `json:"uuid"`
	Error      string      `json:"error"`
}

func NewReqID() RequestID {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return RequestID(fmt.Sprintf("%s %x", time.Now().Format(time.RFC3339Nano), b))
}

// ====================== MQTT 配置 ======================
type MqttConfig struct {
	Broker            string
	Username          string
	Password          string
	ClientID          string
	SetCleanSession   bool
	SetAutoReconnect  bool
	SetConnectTimeout time.Duration
	SetWriteTimeout   time.Duration
	SetKeepAlive      time.Duration
	ListenTopic       string
}

// ====================== MQTT 实例 ======================
type BizHandler func(req []byte) ([]byte, error)

type mqttInstance struct {
	client      mqtt.Client
	listenTopic string
	handlers    map[string]BizHandler
	hMutex      sync.RWMutex
	waiters     map[RequestID]chan RpcMessage
	wMutex      sync.Mutex
}

// ====================== MQTT 管理器 ======================
type MqttManager struct {
	inst map[string]*mqttInstance
	mu   sync.RWMutex
}

func NewMqttManager() *MqttManager {
	return &MqttManager{
		inst: make(map[string]*mqttInstance),
	}
}

// 添加MQTT连接
func (m *MqttManager) Add(name string, cfg MqttConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Broker)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetClientID(cfg.ClientID)
	opts.SetCleanSession(cfg.SetCleanSession)
	opts.SetAutoReconnect(cfg.SetAutoReconnect)
	opts.SetConnectTimeout(cfg.SetConnectTimeout)
	opts.SetWriteTimeout(cfg.SetWriteTimeout)
	opts.SetKeepAlive(cfg.SetKeepAlive)

	// 连接丢失处理
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		log.Printf("MQTT 连接断开: %v", err)
	})

	// 重连成功处理
	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		log.Printf("MQTT 尝试重连...")
	})

	// 连接成功后自动订阅
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		log.Printf("MQTT 连接成功，订阅主题: %s", cfg.ListenTopic)
		client.Subscribe(cfg.ListenTopic, 1, m.inst[name].onMsg)
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	inst := &mqttInstance{
		client:      client,
		listenTopic: cfg.ListenTopic,
		handlers:    make(map[string]BizHandler),
		waiters:     make(map[RequestID]chan RpcMessage),
	}

	m.inst[name] = inst
	log.Println("MQTT 已初始化:", name, " 监听主题:", cfg.ListenTopic)
	return nil
}

// 收到消息
func (inst *mqttInstance) onMsg(c mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("消息处理 panic: %v", r)
		}
	}()

	var m RpcMessage
	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
		log.Printf("消息 JSON 解析失败: %v", err)
		return
	}

	// 1. 处理响应：唤醒等待的调用方
	inst.wMutex.Lock()
	ch, ok := inst.waiters[m.ReqID]
	if ok {
		delete(inst.waiters, m.ReqID)
	}
	inst.wMutex.Unlock()

	if ok {
		// 非阻塞发送，防止超时后通道无人接收
		select {
		case ch <- m:
		default:
			log.Printf("响应通道已满，丢弃消息 ReqID: %s", m.ReqID)
		}
		return
	}

	// 2. 处理请求：执行本地业务
	inst.hMutex.RLock()
	fn, exists := inst.handlers[m.BizTopic]
	inst.hMutex.RUnlock()
	if !exists {
		return
	}

	// 异步执行业务，不阻塞消息循环
	go func() {
		respData, errFn := fn(m.Payload.Bytes())
		if m.ReplyTopic == "" {
			return
		}

		respMsg := RpcMessage{
			ReqID:      m.ReqID,
			ReplyTopic: m.ReplyTopic,
			BizTopic:   m.BizTopic,
			Payload:    respData,
			Uuid:       Init.Config.APP.Uuid,
		}
		if errFn != nil {
			respMsg.Error = errFn.Error()
		}

		respBytes, err := json.Marshal(respMsg)
		if err != nil {
			log.Printf("响应 JSON 打包失败: %v", err)
			return
		}
		inst.client.Publish(m.ReplyTopic, 1, false, respBytes)
	}()
}

// 注册业务接口
func (m *MqttManager) Register(mqttName string, bizTopic string, fn BizHandler) error {
	m.mu.RLock()
	inst, ok := m.inst[mqttName]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("mqtt %s 不存在", mqttName)
	}
	inst.hMutex.Lock()
	defer inst.hMutex.Unlock()
	inst.handlers[bizTopic] = fn
	return nil
}

// ====================== RPC 调用（支持并发）======================
func (m *MqttManager) Call(
	mqttName string,
	targetTopic string,
	bizTopic string,
	req []byte,
	timeout time.Duration,
) ([]byte, error) {

	m.mu.RLock()
	inst, ok := m.inst[mqttName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mqtt %s 不存在", mqttName)
	}

	reqID := NewReqID()
	msg := RpcMessage{
		ReqID:      reqID,
		ReplyTopic: inst.listenTopic,
		BizTopic:   bizTopic,
		Payload:    req,
		Uuid:       Init.Config.APP.Uuid,
	}

	reqBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("请求 JSON 错误: %v", err)
	}

	// 带缓冲通道，防止阻塞
	ch := make(chan RpcMessage, 1)

	// 注册等待者
	inst.wMutex.Lock()
	inst.waiters[reqID] = ch
	inst.wMutex.Unlock()

	// 确保退出时清理 waiter 和 channel
	defer func() {
		inst.wMutex.Lock()
		delete(inst.waiters, reqID)
		close(ch)
		inst.wMutex.Unlock()
	}()

	// 发送请求
	token := inst.client.Publish(targetTopic, 1, false, reqBytes)
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("发布失败: %v", token.Error())
	}

	// 等待响应或超时
	select {
	case resp := <-ch:
		if resp.Error != "" {
			return nil, fmt.Errorf("%s", resp.Error)
		}
		if resp.BizTopic != bizTopic {
			return nil, fmt.Errorf("业务主题不匹配")
		}
		if resp.ReqID != reqID {
			return nil, fmt.Errorf("请求 ID 不匹配")
		}
		return resp.Payload.Bytes(), nil

	case <-time.After(timeout):
		return nil, timeoutErr
	}
}

// 关闭所有连接
func (m *MqttManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, inst := range m.inst {
		inst.client.Disconnect(200)
		log.Printf("MQTT %s 已关闭", name)
	}
}

// ====================== 全局单例 ======================
var M *MqttManager

func New() error {
	if !Init.Config.Mqtt.Enable {
		return nil
	}
	M = NewMqttManager()
	err := M.Add(Init.Config.Mqtt.Broker, MqttConfig{
		Broker:            Init.Config.Mqtt.Broker,
		Username:          Init.Config.Mqtt.Username,
		Password:          Init.Config.Mqtt.Password,
		ClientID:          Init.Config.Mqtt.ClientID,
		SetCleanSession:   Init.Config.Mqtt.SetCleanSession,
		SetAutoReconnect:  Init.Config.Mqtt.SetAutoReconnect,
		SetConnectTimeout: Init.Config.Mqtt.SetConnectTimeout,
		SetWriteTimeout:   Init.Config.Mqtt.SetWriteTimeout,
		SetKeepAlive:      Init.Config.Mqtt.SetKeepAlive,
		ListenTopic:       Init.Config.Mqtt.ListenTopic,
	})
	if err != nil {
		log.Fatalf("MQTT 初始化失败: %v", err)
		return err
	}

	register()
	return nil
}

// ====================== 工具函数 ======================
// 服务端：自动 JSON 解析/打包
func jsonWrap[T any, R any](req []byte, business func(req T) (R, error)) ([]byte, error) {
	var reqData T
	if err := json.Unmarshal(req, &reqData); err != nil {
		log.Println("ERROR JSON 解析失败:", err)
		return nil, err
	}

	respData, err := business(reqData)
	if err != nil {
		return nil, err
	}

	return json.Marshal(respData)
}

// 客户端：自动 JSON 调用
func jsonCall[Req any, Resp any](
	reqData Req,
	respData *Resp,
	broker string,
	topic string,
	method string,
	timeout time.Duration,
) error {
	if !Init.Config.Mqtt.Enable {
		return fmt.Errorf("ERROR Mqtt未启用")
	}
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR 请求打包失败:", err)
		return err
	}

	respBytes, err := M.Call(broker, topic, method, reqBytes, timeout)
	if err != nil {
		log.Println("ERROR RPC 调用失败:", err)
		return err
	}

	if err := json.Unmarshal(respBytes, respData); err != nil {
		log.Println("ERROR 响应解析失败:", err, "内容:", string(respBytes))
		return err
	}
	return nil
}

// ====================== 自定义自由 MQTT 收发（含取消订阅）======================

// Send 自定义发送：broker名称 + 目标topic + 数据
func Send(broker string, topic string, data []byte) error {
	M.mu.RLock()
	inst, ok := M.inst[broker]
	M.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	token := inst.client.Publish(topic, 1, false, data)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Subscribe 自定义订阅：broker + topic + 回调函数
func Subscribe(broker string, topic string, callback func(data []byte)) error {
	M.mu.RLock()
	inst, ok := M.inst[broker]
	M.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	token := inst.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("订阅回调 panic: %v", r)
			}
		}()
		callback(msg.Payload())
	})
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	log.Printf("订阅成功: broker=%s, topic=%s", broker, topic)
	return nil
}

// Subscribe_Close 取消订阅
func Subscribe_Close(broker string, topic string) error {
	M.mu.RLock()
	inst, ok := M.inst[broker]
	M.mu.RUnlock()

	if !ok {
		return fmt.Errorf("mqtt 实例不存在: %s", broker)
	}

	token := inst.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	log.Printf("取消订阅成功: broker=%s, topic=%s", broker, topic)
	return nil
}
