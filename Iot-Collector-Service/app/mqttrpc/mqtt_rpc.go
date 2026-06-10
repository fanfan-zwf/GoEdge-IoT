package mqttrpc

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"main/Init"
	"main/app/mqttbase"
)

// ====================== RPC 专属配置 ======================
// RpcBindConfig 绑定MQTT实例时需要的RPC配置，ListenTopic在此定义
type RpcBindConfig struct {
	ListenTopic string // RPC响应监听主题，仅RPC使用
}

// ====================== 安全JSON载体 ======================
type SafePayload []byte

func (s SafePayload) MarshalJSON() ([]byte, error) {
	if len(s) == 0 {
		return []byte("{}"), nil
	}
	return s, nil
}

func (s *SafePayload) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*s = []byte("{}")
		return nil
	}
	*s = data
	return nil
}

func (s SafePayload) Bytes() []byte {
	return []byte(s)
}

// ====================== RPC 消息协议 ======================
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

var timeoutErr = fmt.Errorf("ERROR 请求超时")

// ====================== RPC 处理器 & 实例 ======================
type BizHandler func(req []byte) ([]byte, error)

type rpcInstance struct {
	mqttName    string
	listenTopic string // RPC监听主题
	handlers    map[string]BizHandler
	hMutex      sync.RWMutex
	waiters     map[RequestID]chan RpcMessage
	wMutex      sync.Mutex
}

// RpcManager RPC管理器
type RpcManager struct {
	rpcInst map[string]*rpcInstance
	mu      sync.RWMutex
}

func NewRpcManager() *RpcManager {
	return &RpcManager{
		rpcInst: make(map[string]*rpcInstance),
	}
}

// BindMQTT 绑定已有MQTT实例 + 订阅RPC监听主题（使用 mqttbase.Subscribe）
func (r *RpcManager) BindMQTT(mqttName string, cfg RpcBindConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.rpcInst[mqttName]; ok {
		return fmt.Errorf("mqtt[%s] 已绑定RPC", mqttName)
	}

	inst := &rpcInstance{
		mqttName:    mqttName,
		listenTopic: cfg.ListenTopic,
		handlers:    make(map[string]BizHandler),
		waiters:     make(map[RequestID]chan RpcMessage),
	}
	r.rpcInst[mqttName] = inst

	// 【重点】使用 mqttbase 包级 Subscribe 订阅监听主题
	err := mqttbase.Subscribe(mqttName, cfg.ListenTopic, inst.onMsg)
	if err != nil {
		delete(r.rpcInst, mqttName)
		return err
	}
	log.Printf("RPC 绑定MQTT[%s] 成功，监听主题: %s", mqttName, cfg.ListenTopic)
	return nil
}

// onMsg 处理RPC请求/响应消息
func (ri *rpcInstance) onMsg(_, _ string, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RPC消息处理panic: %v", r)
		}
	}()

	var msg RpcMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("RPC消息解析失败: %v", err)
		return
	}

	// 1. 响应：唤醒等待方
	ri.wMutex.Lock()
	ch, ok := ri.waiters[msg.ReqID]
	if ok {
		delete(ri.waiters, msg.ReqID)
	}
	ri.wMutex.Unlock()

	if ok {
		select {
		case ch <- msg:
		default:
			log.Printf("响应通道已满，丢弃 ReqID: %s", msg.ReqID)
		}
		return
	}

	// 2. 请求：执行业务handler
	ri.hMutex.RLock()
	fn, exists := ri.handlers[msg.BizTopic]
	ri.hMutex.RUnlock()
	if !exists {
		return
	}

	go func() {
		respData, errFn := fn(msg.Payload.Bytes())
		if msg.ReplyTopic == "" {
			return
		}

		respMsg := RpcMessage{
			ReqID:      msg.ReqID,
			ReplyTopic: msg.ReplyTopic,
			BizTopic:   msg.BizTopic,
			Payload:    respData,
			Uuid:       Init.Config.APP.Uuid,
		}
		if errFn != nil {
			respMsg.Error = errFn.Error()
		}

		respBytes, err := json.Marshal(respMsg)
		if err != nil {
			log.Printf("RPC响应打包失败: %v", err)
			return
		}
		// 【重点】使用 mqttbase 包级 Send 发送响应
		if err := mqttbase.Send(ri.mqttName, msg.ReplyTopic, respBytes); err != nil {
			log.Printf("RPC响应发送失败: %v", err)
		}
	}()
}

// Register 注册RPC业务接口
func (r *RpcManager) Register(mqttName string, bizTopic string, fn BizHandler) error {
	r.mu.RLock()
	ri, ok := r.rpcInst[mqttName]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("mqtt[%s] 未绑定RPC", mqttName)
	}

	ri.hMutex.Lock()
	defer ri.hMutex.Unlock()
	ri.handlers[bizTopic] = fn
	return nil
}

// Call 发起RPC调用
func (r *RpcManager) Call(
	mqttName string,
	targetTopic string,
	bizTopic string,
	req []byte,
	timeout time.Duration,
) ([]byte, error) {

	r.mu.RLock()
	ri, ok := r.rpcInst[mqttName]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("mqtt[%s] 未绑定RPC", mqttName)
	}

	reqID := NewReqID()
	msg := RpcMessage{
		ReqID:      reqID,
		ReplyTopic: ri.listenTopic,
		BizTopic:   bizTopic,
		Payload:    req,
		Uuid:       Init.Config.APP.Uuid,
	}

	reqBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("请求JSON失败: %w", err)
	}

	ch := make(chan RpcMessage, 1)
	ri.wMutex.Lock()
	ri.waiters[reqID] = ch
	ri.wMutex.Unlock()

	defer func() {
		ri.wMutex.Lock()
		delete(ri.waiters, reqID)
		close(ch)
		ri.wMutex.Unlock()
	}()

	// 【重点】使用 mqttbase 包级 Send 发送RPC请求
	if err := mqttbase.Send(mqttName, targetTopic, reqBytes); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != "" {
			return nil, fmt.Errorf("rpc error: %s", resp.Error)
		}
		if resp.BizTopic != bizTopic || resp.ReqID != reqID {
			return nil, fmt.Errorf("响应校验不通过")
		}
		return resp.Payload.Bytes(), nil

	case <-time.After(timeout):
		return nil, timeoutErr
	}
}

// ====================== JSON 工具函数 ======================
func JsonWrap[T any, R any](req []byte, business func(req T) (R, error)) ([]byte, error) {
	var reqData T
	if err := json.Unmarshal(req, &reqData); err != nil {
		log.Println("JSON解析失败:", err)
		return nil, err
	}
	respData, err := business(reqData)
	if err != nil {
		return nil, err
	}
	return json.Marshal(respData)
}

func JsonCall[Req any, Resp any](
	rpcMgr *RpcManager,
	mqttName string,
	targetTopic string,
	bizTopic string,
	reqData Req,
	respData *Resp,
	timeout time.Duration,
) error {
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		log.Println("请求打包失败:", err)
		return err
	}
	respBytes, err := rpcMgr.Call(mqttName, targetTopic, bizTopic, reqBytes, timeout)
	if err != nil {
		log.Println("RPC调用失败:", err)
		return err
	}
	if err := json.Unmarshal(respBytes, respData); err != nil {
		log.Printf("响应解析失败: %v, content: %s", err, string(respBytes))
		return err
	}
	return nil
}

// 全局RPC单例
var M *RpcManager

func InitGlobalRPC() {
	M = NewRpcManager()
}

// New RPC初始化入口（你原有逻辑保留）
func New() error {
	// 1. 底层MQTT已提前通过 mqttbase.New() 初始化完成
	InitGlobalRPC()

	rpcBindCfg := RpcBindConfig{
		ListenTopic: Init.Config.Mqtt_Rpc.ListenTopic,
	}
	err := M.BindMQTT(Init.Config.Mqtt_Rpc.Example, rpcBindCfg)
	if err != nil {
		log.Fatalf("RPC绑定[%s] 失败: %v", Init.Config.Mqtt_Rpc.Example, err)
		return err
	}

	// 3. 注册业务 handler
	// register()

	return nil
}

// 兼容你原有两套json工具函数
func jsonWrap[T any, R any](req []byte, business func(req T) (R, error)) ([]byte, error) {
	return JsonWrap(req, business)
}

func jsonCall[Req any, Resp any](
	reqData Req,
	respData *Resp,
	broker string,
	topic string,
	method string,
	timeout time.Duration,
) error {
	return JsonCall(M, broker, topic, method, reqData, respData, timeout)
}
