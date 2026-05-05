package mqtt_rpc

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

var timeoutErr = fmt.Errorf("ERROR 请求超时")

// ====================== 【正确】消息结构 ======================
type RequestID string

type RpcMessage struct {
	ReqID      RequestID       // 唯一ID，防冲突
	ReplyTopic string          // ✅ 回复主题（必须）
	BizTopic   string          // 业务接口名
	Payload    json.RawMessage // 业务数据
	Uuid       string          // 发送方uuid
	Error      string          // 执行错误
}

func NewReqID() RequestID {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return RequestID(fmt.Sprintf("%x", b))
}

// ====================== MQTT 配置 ======================
type MqttConfig struct {
	Broker            string
	Username          string
	Password          string
	ClientID          string
	SetCleanSession   bool          // 清洁会话（重启不接收离线消息）
	SetAutoReconnect  bool          // 自动重连（必须开）
	SetConnectTimeout time.Duration // 连接超时
	SetWriteTimeout   time.Duration // 写超时
	SetKeepAlive      time.Duration // 心跳保活

	ListenTopic string // 我订阅的主题（别人发给我）
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

	opts.SetCleanSession(cfg.SetCleanSession)     // 清洁会话（重启不接收离线消息）
	opts.SetAutoReconnect(cfg.SetAutoReconnect)   // 自动重连（必须开）
	opts.SetConnectTimeout(cfg.SetConnectTimeout) // 连接超时
	opts.SetWriteTimeout(cfg.SetWriteTimeout)     // 写超时
	opts.SetKeepAlive(cfg.SetKeepAlive)           // 心跳保活

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	inst := &mqttInstance{
		client:      client,
		listenTopic: cfg.ListenTopic,
		handlers:    make(map[string]BizHandler),
		waiters:     make(map[RequestID]chan RpcMessage),
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
	err := json.Unmarshal(msg.Payload(), &m)
	if err != nil {
		log.Printf("ERROR 收到消息 处理json失败 %s", err)
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
		ch <- m
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
	respData, errFn := fn(m.Payload)
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
		respMsg.Error = errFn.Error() // 现在绝对安全
	}

	respBytes, err := json.Marshal(respMsg)
	if err != nil {
		log.Printf("ERROR 收到消息 处理json失败 %s", err)
		return
	}
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

// 传入：mqttName 驱动名称  targetTopic 目标主题  bizTopic 业务主题  reqData 请求数据  timeout 超时时间
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
		ReplyTopic: inst.listenTopic, // 我在哪收响应
		BizTopic:   bizTopic,
		Payload:    req,
		Uuid:       Init.Config.APP.Uuid,
	}
	reqBytes, err := json.Marshal(msg)
	if err != nil {
		err = fmt.Errorf("ERROR JSON转化错误%s", err)
		return nil, err
	}

	ch := make(chan RpcMessage, 1)

	inst.wMutex.Lock()
	inst.waiters[reqID] = ch
	inst.wMutex.Unlock()

	// 超时
	go func() {
		time.Sleep(timeout)
		inst.wMutex.Lock()
		delete(inst.waiters, reqID)
		inst.wMutex.Unlock()
		ch <- RpcMessage{Error: timeoutErr.Error()}
	}()

	// 发给对方主题
	inst.client.Publish(targetTopic, 1, false, reqBytes).Wait()

	resp := <-ch
	if resp.Error == timeoutErr.Error() {
		return nil, timeoutErr
	}

	if resp.BizTopic != bizTopic {
		return nil, fmt.Errorf("ERROR 业务接口 不一致")
	}

	if resp.ReqID != reqID {
		return nil, fmt.Errorf("ERROR 唯一ID 不一致")
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}

	return resp.Payload, nil
}

func (m *MqttManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.inst {
		inst.client.Disconnect(200)
	}
}

var M *MqttManager

func New() (err error) {
	M = NewMqttManager()
	M.Add(Init.Config.Mqtt_Rpc.Broker, MqttConfig{
		Broker:            Init.Config.Mqtt_Rpc.Broker,
		Username:          Init.Config.Mqtt_Rpc.Username,
		Password:          Init.Config.Mqtt_Rpc.Password,
		ClientID:          Init.Config.Mqtt_Rpc.ClientID,
		SetCleanSession:   Init.Config.Mqtt_Rpc.SetCleanSession,   // 清洁会话（重启不接收离线消息）
		SetAutoReconnect:  Init.Config.Mqtt_Rpc.SetAutoReconnect,  // 自动重连（必须开）
		SetConnectTimeout: Init.Config.Mqtt_Rpc.SetConnectTimeout, // 连接超时
		SetWriteTimeout:   Init.Config.Mqtt_Rpc.SetWriteTimeout,   // 写超时
		SetKeepAlive:      Init.Config.Mqtt_Rpc.SetKeepAlive,      // 心跳保活

		ListenTopic: Init.Config.Mqtt_Rpc.ListenTopic,
	})
	register()

	// resp, _ := M.Call("mqtt2", "/device/1", "test.hello", []byte("hi"), 3*time.Second)
	return
}

func jsonWrap[T any, R any](req []byte, business func(req T) (R, error)) (rep []byte, err error) {

	// 1. 自动反序列化 JSON → 请求结构体
	var reqData T
	if err = json.Unmarshal(req, &reqData); err != nil {
		log.Println("ERROR JSON解析失败：", err)
		return
	}

	// 2. 执行业务逻辑（你只需要写这里）
	var respData R
	respData, err = business(reqData)

	// 3. 自动序列化 结构体 → JSON
	rep, err = json.Marshal(respData)
	if err != nil {
		log.Println("ERROR JSON转换失败：", err)
		return
	}

	return
}

// rpcCall 客户端RPC调用包装：自动 JSON + 加解密 + 发送 + 解析
func jsonCall[Req any, Resp any](
	reqData Req,
	respData *Resp,
	broker string,
	topic string,
	method string,
	timeout time.Duration,
) (err error) {
	// 1. 序列化请求
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("ERROR 请求打包失败：", err)
	}

	// 2. 调用 MQTT RPC
	respBytes, err := M.Call(broker, topic, method, reqBytes, timeout)
	if err != nil {
		log.Println("ERROR RPC调用失败：", err)
	}

	// 3. 反序列化到响应结构体
	if err := json.Unmarshal(respBytes, respData); err != nil {
		log.Printf("ERROR 响应解析失败：%s decBytes=%s", err, string(respBytes))
	}

	return
}
