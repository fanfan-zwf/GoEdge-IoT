package tcp_udp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	ErrNotConnected   = errors.New("未建立网络连接")
	ErrInvalidLength  = errors.New("接收长度必须大于0")
	ErrReceiveTimeout = errors.New("接收数据超时")
	ErrQueueFull      = errors.New("发送队列已满")
	ErrClosed         = errors.New("客户端已关闭")
	ErrConnectTimeout = errors.New("建立连接超时")
	ErrWriteFailed    = errors.New("写入数据失败，连接已断开")
)

type ConnType string

const (
	TCP ConnType = "tcp"
	UDP ConnType = "udp"
)

type Config struct {
	Type              ConnType
	RemoteAddr        string
	LocalAddr         string
	Reconnect         bool
	ReconnectInterval time.Duration
	ConnectTimeout    time.Duration
	SendQueueSize     int
}

type Client interface {
	Send(data []byte) error
	Receive(maxLen int, timeout time.Duration) ([]byte, error)
	ClearSendQueue()
	Close() error
	OnConnected(at time.Duration, callback func())
	Reconnect(newCfg *Config) error // 新增
}

type baseClient struct {
	cfg       Config
	cfgMu     sync.Mutex
	mu        sync.RWMutex
	conn      net.Conn
	closed    bool
	sendQueue chan []byte

	callbackOnce  sync.Once
	callbackTimer *time.Timer
	callbackAt    time.Duration
	callbackFunc  func()
}

type TcpClient struct {
	*baseClient
}

type UdpClient struct {
	*baseClient
	udpConn *net.UDPConn
}

func NewClient(cfg Config) Client {
	base := &baseClient{
		cfg:       cfg,
		sendQueue: make(chan []byte, cfg.SendQueueSize),
	}

	var cli Client
	if cfg.Type == TCP {
		cli = &TcpClient{baseClient: base}
	} else {
		cli = &UdpClient{baseClient: base}
	}

	go cli.(interface{ sendWorker() }).sendWorker()
	_ = cli.Reconnect(nil)
	return cli
}

// 统一重连入口（修复类型断言错误）
func (t *TcpClient) Reconnect(newCfg *Config) error {
	return t.reconnectInternal(newCfg, t.dial)
}

func (u *UdpClient) Reconnect(newCfg *Config) error {
	return u.reconnectInternal(newCfg, u.dial)
}

// 内部通用重连逻辑
func (b *baseClient) reconnectInternal(newCfg *Config, dialFunc func() error) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return ErrClosed
	}
	b.mu.Unlock()

	// 更新配置
	if newCfg != nil {
		b.cfgMu.Lock()
		b.cfg = *newCfg
		b.cfgMu.Unlock()
	}

	// 关闭旧连接
	b.mu.Lock()
	if b.conn != nil {
		_ = b.conn.Close()
		b.conn = nil
	}
	b.mu.Unlock()

	// 执行连接
	err := dialFunc()
	if err == nil {
		fmt.Println("✅ 连接/重连成功")
		b.startCallback()
	}
	return err
}

func (b *baseClient) startCallback() {
	if b.callbackFunc == nil {
		return
	}
	b.callbackOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.callbackTimer = time.AfterFunc(b.callbackAt, func() {
			if !b.isClose() && b.getConn() != nil {
				b.callbackFunc()
			}
		})
	})
}

func (b *baseClient) OnConnected(at time.Duration, callback func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.callbackAt = at
	b.callbackFunc = callback
}

func (b *baseClient) ClearSendQueue() {
	for {
		select {
		case <-b.sendQueue:
		default:
			return
		}
	}
}

func (b *baseClient) isClose() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

func (b *baseClient) getConn() net.Conn {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.conn
}

func (b *baseClient) setConn(c net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.conn = c
}

// -------------------------- TCP --------------------------
func (t *TcpClient) dial() error {
	t.cfgMu.Lock()
	cfg := t.cfg
	t.cfgMu.Unlock()

	if t.isClose() {
		return ErrClosed
	}

	dialer := net.Dialer{Timeout: cfg.ConnectTimeout}
	if cfg.LocalAddr != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.LocalAddr)
		if err != nil {
			return err
		}
		dialer.LocalAddr = addr
	}

	conn, err := dialer.Dial("tcp", cfg.RemoteAddr)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return ErrConnectTimeout
		}
		return err
	}

	t.setConn(conn)
	return nil
}

func (t *TcpClient) Send(data []byte) error {
	if t.isClose() {
		return ErrClosed
	}
	select {
	case t.sendQueue <- data:
		return nil
	default:
		return ErrQueueFull
	}
}

func (t *TcpClient) sendWorker() {
	for raw := range t.sendQueue {
		if t.isClose() {
			return
		}

		if t.getConn() == nil {
			t.cfgMu.Lock()
			reconnect := t.cfg.Reconnect
			t.cfgMu.Unlock()

			if !reconnect {
				continue
			}

			fmt.Println("🔌 尝试重连...")
			err := t.Reconnect(nil)
			if err != nil {
				fmt.Printf("重连失败: %v\n", err)
				t.cfgMu.Lock()
				interval := t.cfg.ReconnectInterval
				t.cfgMu.Unlock()
				time.Sleep(interval)
				continue
			}
		}

		conn := t.getConn()
		_, err := conn.Write(raw)
		if err != nil {
			fmt.Printf("发送失败，断开连接: %v\n", err)
			_ = conn.Close()
			t.setConn(nil)
			continue
		}
	}
}

func (t *TcpClient) Receive(maxLen int, timeout time.Duration) ([]byte, error) {
	if maxLen <= 0 {
		return nil, ErrInvalidLength
	}
	conn := t.getConn()
	if conn == nil {
		return nil, ErrNotConnected
	}

	buf := make([]byte, maxLen)
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		_ = conn.SetReadDeadline(time.Time{})
	}

	n, err := conn.Read(buf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, ErrReceiveTimeout
		}
		_ = conn.Close()
		t.setConn(nil)
		return nil, err
	}
	return buf[:n], nil
}

func (t *TcpClient) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true

	if t.conn != nil {
		_ = t.conn.Close()
		t.conn = nil
	}
	if t.callbackTimer != nil {
		t.callbackTimer.Stop()
	}
	close(t.sendQueue)
	return nil
}

// -------------------------- UDP --------------------------
func (u *UdpClient) dial() error {
	u.cfgMu.Lock()
	cfg := u.cfg
	u.cfgMu.Unlock()

	if u.isClose() {
		return ErrClosed
	}

	lAddr, _ := net.ResolveUDPAddr("udp", cfg.LocalAddr)
	rAddr, err := net.ResolveUDPAddr("udp", cfg.RemoteAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", lAddr, rAddr)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return ErrConnectTimeout
		}
		return err
	}

	u.udpConn = conn
	u.setConn(conn)
	return nil
}

func (u *UdpClient) Send(data []byte) error {
	if u.isClose() {
		return ErrClosed
	}
	select {
	case u.sendQueue <- data:
		return nil
	default:
		return ErrQueueFull
	}
}

func (u *UdpClient) sendWorker() {
	for raw := range u.sendQueue {
		if u.isClose() {
			return
		}
		if u.udpConn == nil {
			if err := u.dial(); err != nil {
				fmt.Printf("UDP 初始化失败: %v\n", err)
				continue
			}
		}
		_, err := u.udpConn.Write(raw)
		if err != nil {
			fmt.Printf("UDP 发送失败: %v\n", err)
		}
	}
}

func (u *UdpClient) Receive(maxLen int, timeout time.Duration) ([]byte, error) {
	if maxLen <= 0 {
		return nil, ErrInvalidLength
	}
	if u.udpConn == nil {
		return nil, ErrNotConnected
	}

	buf := make([]byte, maxLen)
	if timeout > 0 {
		_ = u.udpConn.SetReadDeadline(time.Now().Add(timeout))
	} else {
		_ = u.udpConn.SetReadDeadline(time.Time{})
	}

	n, _, err := u.udpConn.ReadFromUDP(buf)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, ErrReceiveTimeout
		}
		return nil, err
	}
	return buf[:n], nil
}

func (u *UdpClient) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.closed {
		return nil
	}
	u.closed = true

	if u.udpConn != nil {
		_ = u.udpConn.Close()
		u.udpConn = nil
	}
	if u.callbackTimer != nil {
		u.callbackTimer.Stop()
	}
	close(u.sendQueue)
	return nil
}

// -------------------------- 使用示例 --------------------------
func main() {
	cfg := Config{
		Type:              TCP,
		RemoteAddr:        "127.0.0.1:8080",
		Reconnect:         true,
		ReconnectInterval: 2 * time.Second,
		ConnectTimeout:    3 * time.Second,
		SendQueueSize:     50,
	}

	cli := NewClient(cfg)
	defer cli.Close()

	cli.OnConnected(3*time.Second, func() {
		fmt.Println("🎉 连接稳定，回调执行！")
	})

	_ = cli.Send([]byte("hello"))

	// 10秒后重连并修改地址
	go func() {
		time.Sleep(10 * time.Second)
		newCfg := cfg
		newCfg.RemoteAddr = "127.0.0.1:8081"
		err := cli.Reconnect(&newCfg)
		if err != nil {
			fmt.Println("重连失败:", err)
		} else {
			fmt.Println("✅ 新配置重连成功")
		}
	}()

	for {
		data, err := cli.Receive(1024, 1*time.Second)
		if err != nil {
			if err == ErrReceiveTimeout {
				continue
			}
			fmt.Println("接收错误:", err)
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Println("收到:", string(data))
	}
}
