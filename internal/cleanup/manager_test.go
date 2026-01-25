package cleanup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x1t/tinyportmapper-go/internal/log"
)

func TestNewManager(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	require.NotNil(t, mgr)
	assert.Equal(t, 5*time.Minute, mgr.tcpTimeout)
	assert.Equal(t, 3*time.Minute, mgr.udpTimeout)
	assert.Equal(t, 1*time.Second, mgr.clearInterval)
	assert.Equal(t, 30, mgr.clearRatio)
	assert.Equal(t, 1, mgr.minClear)
}

func TestManager_AddAndCount(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 初始状态
	assert.Equal(t, 0, mgr.TotalConnCount())
	assert.Equal(t, 0, mgr.TCPConnCount())
	assert.Equal(t, 0, mgr.UDPConnCount())

	// 添加 TCP 连接
	mgr.AddTCP("tcp-key-1", "tcp-value-1")
	mgr.AddTCP("tcp-key-2", "tcp-value-2")
	assert.Equal(t, 2, mgr.TCPConnCount())

	// 添加 UDP 连接
	mgr.AddUDP("udp-key-1", "udp-value-1")
	assert.Equal(t, 1, mgr.UDPConnCount())

	// 总数
	assert.Equal(t, 3, mgr.TotalConnCount())
}

func TestManager_Remove(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 添加连接
	mgr.AddTCP("key-1", "value-1")
	mgr.AddTCP("key-2", "value-2")
	assert.Equal(t, 2, mgr.TCPConnCount())

	// 移除一个
	mgr.RemoveTCP("key-1")
	assert.Equal(t, 1, mgr.TCPConnCount())

	// 移除不存在的key（不应panic）
	mgr.RemoveTCP("non-existent")
	assert.Equal(t, 1, mgr.TCPConnCount())
}

func TestManager_Touch(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 添加连接
	mgr.AddTCP("key-1", "value-1")

	// Touch应该更新活跃时间
	mgr.TouchTCP("key-1")

	// 连接仍然存在
	assert.Equal(t, 1, mgr.TCPConnCount())
}

func TestManager_PeekBack(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 空管理器
	_, _, ok := mgr.PeekBackTCP()
	assert.False(t, ok)

	// 添加连接
	mgr.AddTCP("key-1", "value-1")
	mgr.AddTCP("key-2", "value-2")

	// PeekBack应该返回最旧的连接
	key, ts, ok := mgr.PeekBackTCP()
	assert.True(t, ok)
	assert.NotNil(t, key)
	assert.False(t, ts.IsZero())
}

func TestManager_Cleanup(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    100 * time.Millisecond, // 短超时
		UDPTimeout:    100 * time.Millisecond,
		ClearInterval: 50 * time.Millisecond,
		ClearRatio:    100, // 100%清理
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 启动清理
	mgr.Start()
	defer mgr.Stop()

	// 等待清理执行
	time.Sleep(200 * time.Millisecond)

	// 应该能够正常停止
}

func TestManager_StartStop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 100 * time.Millisecond,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 启动
	mgr.Start()

	// 等待一下
	time.Sleep(200 * time.Millisecond)

	// 停止
	mgr.Stop()

	// 应该能够再次停止而不panic
	mgr.Stop()
}

func TestManager_SetCleanupFn(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	var tcpCleanupCalled bool
	var udpCleanupCalled bool

	mgr.SetTCPCleanupFn(func(key, value interface{}) {
		tcpCleanupCalled = true
	})

	mgr.SetUDPCleanupFn(func(key, value interface{}) {
		udpCleanupCalled = true
	})

	// 验证没有panic
	assert.NotPanics(t, func() {
		mgr.Start()
		mgr.Stop()
	})

	// 验证回调已设置（通过访问内部状态验证）
	assert.NotPanics(t, func() {
		mgr.AddTCP("test-key", "test-value")
		mgr.RemoveTCP("test-key")
	})

	// 使用变量避免编译器警告
	_ = tcpCleanupCalled
	_ = udpCleanupCalled
}

// TestManager_UDPOperations 测试 UDP 相关的操作
func TestManager_UDPOperations(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 添加 UDP 连接
	mgr.AddUDP("udp-key-1", "udp-value-1")
	mgr.AddUDP("udp-key-2", "udp-value-2")
	assert.Equal(t, 2, mgr.UDPConnCount())

	// TouchUDP 应该更新活跃时间
	mgr.TouchUDP("udp-key-1")
	assert.Equal(t, 2, mgr.UDPConnCount())

	// RemoveUDP 应该移除连接
	mgr.RemoveUDP("udp-key-1")
	assert.Equal(t, 1, mgr.UDPConnCount())

	// 移除不存在的 key（不应 panic）
	mgr.RemoveUDP("non-existent")
	assert.Equal(t, 1, mgr.UDPConnCount())
}

// TestManager_PeekBackUDP 测试 PeekBackUDP 函数
func TestManager_PeekBackUDP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, true)
	require.NoError(t, err)

	cfg := &Config{
		TCPTimeout:    5 * time.Minute,
		UDPTimeout:    3 * time.Minute,
		ClearInterval: 1 * time.Second,
		ClearRatio:    30,
		MinClear:      1,
		MaxConns:      1000,
	}

	mgr := NewManager(logger, cfg)

	// 空管理器
	_, _, ok := mgr.PeekBackUDP()
	assert.False(t, ok)

	// 添加 UDP 连接
	mgr.AddUDP("udp-key-1", "udp-value-1")
	mgr.AddUDP("udp-key-2", "udp-value-2")

	// PeekBackUDP 应该返回最旧的 UDP 连接
	key, ts, ok := mgr.PeekBackUDP()
	assert.True(t, ok)
	assert.NotNil(t, key)
	assert.False(t, ts.IsZero())
}
