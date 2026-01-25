package cleanup

import (
	"sync"
	"time"

	"github.com/x1t/tinyportmapper-go/internal/lru"
	"github.com/x1t/tinyportmapper-go/internal/log"
	"go.uber.org/zap"
)

// Manager 连接清理管理器
// 对应 C++ 中的 conn_manager_tcp_t 和 conn_manager_udp_t
type Manager struct {
	logger        *log.Logger
	tcpLRU        *lru.Cache
	udpLRU        *lru.Cache
	tcpTimeout    time.Duration
	udpTimeout    time.Duration
	clearInterval time.Duration
	clearRatio    int
	minClear      int
	disabled      bool // 是否禁用连接清理，对应 C++ 的 disable_conn_clear
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// Config 清理管理器配置
type Config struct {
	TCPTimeout    time.Duration
	UDPTimeout    time.Duration
	ClearInterval time.Duration
	ClearRatio    int
	MinClear      int
	MaxConns      int
	DisableClear  bool // 是否禁用连接清理，对应 C++ 的 disable_conn_clear
}

// NewManager 创建新的清理管理器
func NewManager(logger *log.Logger, cfg *Config) *Manager {
	return &Manager{
		logger:        logger,
		tcpLRU:        lru.NewCache(cfg.MaxConns),
		udpLRU:        lru.NewCache(cfg.MaxConns),
		tcpTimeout:    cfg.TCPTimeout,
		udpTimeout:    cfg.UDPTimeout,
		clearInterval: cfg.ClearInterval,
		clearRatio:    cfg.ClearRatio,
		minClear:      cfg.MinClear,
		disabled:      cfg.DisableClear, // 使用配置中的禁用清理选项
		stopCh:        make(chan struct{}),
	}
}

// Start 启动清理 goroutine
// 对应 C++ 中的 ev_timer_init + clear_timer_cb
func (m *Manager) Start() {
	m.wg.Add(1)
	go m.cleanupLoop()
}

// Stop 停止清理
func (m *Manager) Stop() {
	select {
	case <-m.stopCh:
		// 已經關閉
		return
	default:
		close(m.stopCh)
	}
	m.wg.Wait()
}

// cleanupLoop 清理循环
func (m *Manager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.clearInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case now := <-ticker.C:
			m.Cleanup(now)
		}
	}
}

// Cleanup 执行清理
// 对应 C++ 中的 clear_inactive0()
func (m *Manager) Cleanup(now time.Time) {
	// 如果禁用连接清理，直接返回
	if m.disabled {
		return
	}

	// 清理 TCP 连接
	tcpRemoved := m.tcpLRU.RemoveExpiredRatio(now, m.clearRatio, m.minClear)
	if tcpRemoved > 0 {
		m.logger.Debug("清理 TCP 连接", zap.Int("count", tcpRemoved))
	}

	// 清理 UDP 连接
	udpRemoved := m.udpLRU.RemoveExpiredRatio(now, m.clearRatio, m.minClear)
	if udpRemoved > 0 {
		m.logger.Debug("清理 UDP 连接", zap.Int("count", udpRemoved))
	}
}

// TouchTCP 更新 TCP 连接的活跃时间
// 对应 C++ 中的 conn_manager_tcp.lru.update()
func (m *Manager) TouchTCP(key interface{}) {
	_ = m.tcpLRU.Touch(key, m.tcpTimeout)
}

// TouchUDP 更新 UDP 连接的活跃时间
// 对应 C++ 中的 conn_manager_udp.lru.update()
func (m *Manager) TouchUDP(key interface{}) {
	_ = m.udpLRU.Touch(key, m.udpTimeout)
}

// AddTCP 添加 TCP 连接
// 对应 C++ 中的 conn_manager_tcp.lru.new_key()
func (m *Manager) AddTCP(key, value interface{}) {
	m.tcpLRU.Put(key, value, m.tcpTimeout)
}

// AddUDP 添加 UDP 连接
// 对应 C++ 中的 conn_manager_udp.lru.new_key()
func (m *Manager) AddUDP(key, value interface{}) {
	m.udpLRU.Put(key, value, m.udpTimeout)
}

// RemoveTCP 移除 TCP 连接
func (m *Manager) RemoveTCP(key interface{}) {
	_ = m.tcpLRU.Remove(key)
}

// RemoveUDP 移除 UDP 连接
func (m *Manager) RemoveUDP(key interface{}) {
	_ = m.udpLRU.Remove(key)
}

// PeekBackTCP 查看最旧的 TCP 连接
// 对应 C++ 中的 lru_collector_t::peek_back()
func (m *Manager) PeekBackTCP() (interface{}, time.Time, bool) {
	return m.tcpLRU.PeekBackTS()
}

// PeekBackUDP 查看最旧的 UDP 连接
func (m *Manager) PeekBackUDP() (interface{}, time.Time, bool) {
	return m.udpLRU.PeekBackTS()
}

// TCPConnCount 返回 TCP 连接数
func (m *Manager) TCPConnCount() int {
	return m.tcpLRU.Len()
}

// UDPConnCount 返回 UDP 连接数
func (m *Manager) UDPConnCount() int {
	return m.udpLRU.Len()
}

// TotalConnCount 返回总连接数
func (m *Manager) TotalConnCount() int {
	return m.tcpLRU.Len() + m.udpLRU.Len()
}

// SetTCPCleanupFn 设置 TCP 清理回调
func (m *Manager) SetTCPCleanupFn(fn func(key, value interface{})) {
	m.tcpLRU.SetCleanupFn(fn)
}

// SetUDPCleanupFn 设置 UDP 清理回调
func (m *Manager) SetUDPCleanupFn(fn func(key, value interface{})) {
	m.udpLRU.SetCleanupFn(fn)
}

// Zap zap 字段别名
type Zap = log.ZapField
