package conn

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/x1t/tinyportmapper-go/internal/types"
)

// UDPConn UDP 连接对
// 对应 C++ 中的 udp_pair_t
type UDPConn struct {
	Addr        *types.Address // 客户端地址
	LocalListen int            // 本地监听 fd
	FD          int            // 远程 UDP fd
	FD64        uint64         // 64 位 fd 标识

	lastActive int64  // 最后活跃时间 (Unix ms)
	refCount   int32  // 引用计数
	closed     int32  // 是否已关闭
}

// NewUDPConn 创建新的 UDP 连接对
// 对应 C++ 中的 udp_accept_cb 中的连接创建逻辑
func NewUDPConn(addr *types.Address, localListen, fd int, fd64 uint64) *UDPConn {
	return &UDPConn{
		Addr:        addr,
		LocalListen: localListen,
		FD:          fd,
		FD64:        fd64,
		lastActive:  time.Now().UnixMilli(),
	}
}

// UpdateActivity 更新最后活跃时间
func (c *UDPConn) UpdateActivity() {
	atomic.StoreInt64(&c.lastActive, time.Now().UnixMilli())
}

// LastActive 获取最后活跃时间
func (c *UDPConn) LastActive() int64 {
	return atomic.LoadInt64(&c.lastActive)
}

// IncRef 增加引用计数
func (c *UDPConn) IncRef() {
	atomic.AddInt32(&c.refCount, 1)
}

// DecRef 减少引用计数
func (c *UDPConn) DecRef() bool {
	return atomic.AddInt32(&c.refCount, -1) == 0
}

// RefCnt 获取引用计数
func (c *UDPConn) RefCnt() int32 {
	return atomic.LoadInt32(&c.refCount)
}

// IsClosed 检查是否已关闭
func (c *UDPConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

// Close 关闭连接
// 对应 C++ 中的 conn_manager_udp.erase()
func (c *UDPConn) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		// 关闭 fd
		if c.FD >= 0 {
			// syscall.Close(c.FD)
			c.FD = -1
		}
	}
	return nil
}

// AddrString 返回地址字符串
// 对应 C++ 中的 udp_pair_t::addr_s
func (c *UDPConn) AddrString() string {
	return c.Addr.String()
}

// UDPManager UDP 连接管理器
// 对应 C++ 中的 conn_manager_udp_t
type UDPManager struct {
	mu            sync.RWMutex
	addrToConn    map[string]*UDPConn
	connList      []*UDPConn
	lastClearTime int64
	maxConns      int
}

// NewUDPManager 创建新的 UDP 连接管理器
func NewUDPManager(maxConns int) *UDPManager {
	return &UDPManager{
		addrToConn: make(map[string]*UDPConn),
		connList:   make([]*UDPConn, 0, maxConns),
		maxConns:   maxConns,
	}
}

// Add 添加 UDP 连接
// 对应 C++ 中的 conn_manager_udp.adress_to_info[tmp_addr] = &udp_pair
func (m *UDPManager) Add(conn *UDPConn) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.connList) >= m.maxConns {
		return false
	}

	addrStr := conn.AddrString()
	if _, exists := m.addrToConn[addrStr]; exists {
		return false
	}

	m.addrToConn[addrStr] = conn
	m.connList = append(m.connList, conn)
	return true
}

// Get 获取 UDP 连接
// 对应 C++ 中的 conn_manager_udp.adress_to_info.find()
func (m *UDPManager) Get(addr *types.Address) (*UDPConn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, ok := m.addrToConn[addr.String()]
	return conn, ok
}

// Remove 移除 UDP 连接
// 对应 C++ 中的 conn_manager_udp.erase()
// 使用 swap-and-pop 优化删除性能
func (m *UDPManager) Remove(conn *UDPConn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	addrStr := conn.AddrString()
	delete(m.addrToConn, addrStr)

	for i, c := range m.connList {
		if c == conn {
			// 使用 swap-and-pop 优化：最后一个元素移到当前位置，然后删除最后一个
			lastIdx := len(m.connList) - 1
			if i != lastIdx {
				m.connList[i] = m.connList[lastIdx]
			}
			m.connList = m.connList[:lastIdx]
			break
		}
	}
}

// Count 返回连接数
func (m *UDPManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connList)
}

// Iter 遍历所有连接 (只读)
func (m *UDPManager) Iter(fn func(conn *UDPConn) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, conn := range m.connList {
		if !fn(conn) {
			break
		}
	}
}

// Len 返回列表长度
func (m *UDPManager) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connList)
}
