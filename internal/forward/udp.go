package forward

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/x1t/tinyportmapper-go/internal/conn"
	"github.com/x1t/tinyportmapper-go/internal/types"
)

// UDPForwarder UDP 数据转发器
// 对应 C++ 中的 udp_cb 和 udp_accept_cb 回调函数
type UDPForwarder struct {
	localConn *net.UDPConn // 本地监听连接
	remoteUDP *net.UDPConn // 远程 UDP 连接
	client    *types.Address // 客户端地址
	conn      *conn.UDPConn // UDP 连接对 (可选)

	buf       []byte
	bufSize   int
	closed    int32
	done      chan struct{}
	onClose   func()
	wg        sync.WaitGroup
	mu        sync.Mutex
	lastActive int64 // 最后活跃时间
}

// NewUDPForwarder 创建新的 UDP 转发器
func NewUDPForwarder(localConn, remoteUDP *net.UDPConn, client *types.Address, conn *conn.UDPConn, bufSize int) *UDPForwarder {
	if bufSize <= 0 {
		bufSize = 16 * 1024 // 默认 16KB
	}
	return &UDPForwarder{
		localConn: localConn,
		remoteUDP: remoteUDP,
		client:    client,
		conn:      conn,
		buf:       make([]byte, bufSize),
		bufSize:   bufSize,
		done:      make(chan struct{}),
	}
}

// Start 开始转发 (只处理远程到本地的转发)
// 对应 C++ 中的 ev_io_start(loop, &udp_pair.ev)
// 注意：本地到远程的转发由 handleUDP 直接处理，避免竞争
func (f *UDPForwarder) Start() {
	f.wg.Add(1)
	go f.forwardFromRemote() // 处理远程响应 -> 本地监听
}

// MarkActive 标记为活跃
func (f *UDPForwarder) MarkActive() {
	atomic.StoreInt64(&f.lastActive, time.Now().Unix())
}

// GetLastActive 获取最后活跃时间
func (f *UDPForwarder) GetLastActive() int64 {
	return atomic.LoadInt64(&f.lastActive)
}

// ForwardToRemote 将数据转发到远程服务器
func (f *UDPForwarder) ForwardToRemote(data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, err := f.remoteUDP.Write(data)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
			return
		}
	}
}

// forwardFromRemote 从远程转发到本地
// 对应 C++ 中的 udp_cb 回调函数
func (f *UDPForwarder) forwardFromRemote() {
	defer f.wg.Done()

	for {
		select {
		case <-f.done:
			return
		default:
		}

		// 设置截止时间
		f.remoteUDP.SetReadDeadline(time.Now().Add(1 * time.Minute))

		n, err := f.remoteUDP.Read(f.buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return
		}

		// 标记活跃
		f.MarkActive()

		// 发送到本地监听
		sent, err := f.localConn.WriteTo(f.buf[:n], f.client.ToUDPAddr())
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
		}
		_ = sent
	}
}

// Close 关闭转发器
func (f *UDPForwarder) Close() {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		close(f.done)
		if f.onClose != nil {
			f.onClose()
		}
	}
}

// Wait 等待转发完成
func (f *UDPForwarder) Wait() {
	f.wg.Wait()
}

// IsClosed 检查是否已关闭
func (f *UDPForwarder) IsClosed() bool {
	return atomic.LoadInt32(&f.closed) == 1
}

// GetClientAddr 获取客户端地址字符串
func (f *UDPForwarder) GetClientAddr() string {
	return f.client.String()
}

// UDPForwarderManager UDP 转发器管理器
type UDPForwarderManager struct {
	mu        sync.RWMutex
	forwarders map[string]*UDPForwarder // key: client address string (使用字符串避免指针比较问题)
}

// NewUDPForwarderManager 创建新的 UDP 转发器管理器
func NewUDPForwarderManager() *UDPForwarderManager {
	return &UDPForwarderManager{
		forwarders: make(map[string]*UDPForwarder),
	}
}

// Add 添加转发器
func (m *UDPForwarderManager) Add(f *UDPForwarder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forwarders[f.client.String()] = f
}

// Get 获取转发器
func (m *UDPForwarderManager) Get(client *types.Address) (*UDPForwarder, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	f, ok := m.forwarders[client.String()]
	return f, ok
}

// Remove 移除转发器
func (m *UDPForwarderManager) Remove(client *types.Address) {
	if client == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.forwarders, client.String())
}

// Count 返回转发器数量
func (m *UDPForwarderManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.forwarders)
}

// CloseAll 关闭所有转发器
func (m *UDPForwarderManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, f := range m.forwarders {
		f.Close()
	}
	m.forwarders = make(map[string]*UDPForwarder)
}
