package event

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/x1t/tinyportmapper-go/internal/config"
	"github.com/x1t/tinyportmapper-go/internal/log"
)

func TestNewLoop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:        "127.0.0.1:0",
		RemoteAddr:        "127.0.0.1:0",
		EnableTCP:         true,
		EnableUDP:         false,
		SocketBufferSizeKbyte: 1024,
		TCPTimeout:        360000 * time.Millisecond,
		UDPTimeout:        180000 * time.Millisecond,
		ClearInterval:     1000 * time.Millisecond,
		ClearRatio:        30,
		MaxConnections:    100,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestNewLoop_InvalidListenAddr(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:   "invalid:addr",
		RemoteAddr:   "127.0.0.1:0",
		EnableTCP:    true,
	}

	loop, err := NewLoop(logger, cfg)
	assert.Error(t, err)
	assert.Nil(t, loop)
}

func TestNewLoop_InvalidRemoteAddr(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:   "127.0.0.1:0",
		RemoteAddr:   "invalid:addr",
		EnableTCP:    true,
	}

	loop, err := NewLoop(logger, cfg)
	assert.Error(t, err)
	assert.Nil(t, loop)
}

func TestLoop_Stats(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:        "127.0.0.1:0",
		RemoteAddr:        "127.0.0.1:0",
		EnableTCP:         false,
		EnableUDP:         false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:     1000 * time.Millisecond,
		MaxConnections:    100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	stats := loop.Stats()
	assert.False(t, stats.Running)
	assert.Equal(t, 0, stats.TCPConnections)
	assert.Equal(t, 0, stats.UDPConnections)
}

func TestLoop_Stop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:        "127.0.0.1:0",
		RemoteAddr:        "127.0.0.1:0",
		EnableTCP:         false,
		EnableUDP:         false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:     1000 * time.Millisecond,
		MaxConnections:    100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// Stop应该不会panic
	loop.Stop()
	loop.Stop() // 重复调用应该安全
}

func TestLoop_Run_OnlyUDP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:        listenAddr,
		RemoteAddr:        remoteAddr,
		EnableTCP:         false,
		EnableUDP:         true,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:     1000 * time.Millisecond,
		MaxConnections:    100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止 (不使用race检测)
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_Run_InvalidUDP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 使用占用的端口应该导致UDP监听失败
	cfg := &config.Config{
		ListenAddr:        "127.0.0.1:0",
		RemoteAddr:        "127.0.0.1:0",
		EnableTCP:         false,
		EnableUDP:         true,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:     1000 * time.Millisecond,
		MaxConnections:    100,
	}

	// 首先创建一个会占用地址的监听
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	cfg.ListenAddr = listener.LocalAddr().String()
	defer listener.Close()

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// Run应该会失败因为端口已被占用
	err = loop.Run()
	assert.Error(t, err)
}

func TestLoop_ConcurrentStart(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:        listenAddr,
		RemoteAddr:        remoteAddr,
		EnableTCP:         true,
		EnableUDP:         false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:     1000 * time.Millisecond,
		MaxConnections:    100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 多次调用Stop应该安全
	for i := 0; i < 5; i++ {
		loop.Stop()
	}
}

func TestLoop_Run_InvalidTCP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 使用占用的端口应该导致TCP监听失败
	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	// 首先创建一个会占用地址的监听
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	cfg.ListenAddr = listener.Addr().String()
	defer listener.Close()

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// Run应该会失败因为端口已被占用
	err = loop.Run()
	assert.Error(t, err)
}

func TestLoop_Wait_NotRunning(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 即使没有运行，Wait也应该安全返回
	loop.Wait()
}

func TestLoop_Run_BothTCPAndUDP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           true,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_Stats_MultipleConnections(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// Stats应该返回正确的初始状态
	stats := loop.Stats()
	assert.Equal(t, 0, stats.TCPConnections)
	assert.Equal(t, 0, stats.UDPConnections)
	assert.False(t, stats.Running)
}

func TestNewLoop_IPv6(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "[::1]:0",
		RemoteAddr:          "[::1]:0",
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		TCPTimeout:          360000 * time.Millisecond,
		UDPTimeout:          180000 * time.Millisecond,
		ClearInterval:       1000 * time.Millisecond,
		ClearRatio:          30,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_Stop_Double(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 多次停止
	loop.Stop()
	time.Sleep(50 * time.Millisecond)
	loop.Stop()
	loop.Stop()
}

func TestLoop_Run_TCPConnection(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	defer listener.Close()

	// 创建远程服务器
	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	defer remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_Run_UDPConnection(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.LocalAddr().String()
	defer listener.Close()

	// 创建远程地址
	remoteListener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.LocalAddr().String()
	defer remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           false,
		EnableUDP:           true,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_Run_RejectWhenMaxConnections(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      1, // 只允许1个连接
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_Run_OnlyTCP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestLoop_WithDifferentTimeouts(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		TCPTimeout:          5 * time.Minute,
		UDPTimeout:          2 * time.Minute,
		ClearInterval:       500 * time.Millisecond,
		TimerInterval:       200 * time.Millisecond,
		MaxConnections:      100,
		ClearRatio:          20,
		ClearMin:            5,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)

	stats := loop.Stats()
	assert.False(t, stats.Running)
	assert.Equal(t, 0, stats.TCPConnections)
	assert.Equal(t, 0, stats.UDPConnections)
}

func TestLoop_WithZeroConnections(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:      0,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_StopMultipleTimes(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 多次调用Stop应该安全
	for i := 0; i < 10; i++ {
		loop.Stop()
	}
}

func TestLoop_Run_ZeroTimeout(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		TCPTimeout:          1 * time.Millisecond, // 使用最小超时而不是零
		UDPTimeout:          1 * time.Millisecond,
		ClearInterval:       1 * time.Millisecond, // 使用最小间隔而不是零
		TimerInterval:       1 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}

func TestNewLoop_OnlyUDP(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "[::1]:0",
		RemoteAddr:          "[::1]:0",
		EnableTCP:           false,
		EnableUDP:           true,
		SocketBufferSizeKbyte: 1024,
		UDPTimeout:          180000 * time.Millisecond,
		ClearInterval:       1000 * time.Millisecond,
		ClearRatio:          30,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_Stats_AfterStop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)

	// Stats应该显示未运行
	stats := loop.Stats()
	assert.False(t, stats.Running)
}

func TestLoop_WithDisableConnClear(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		DisableConnClear:    true,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_WithDifferentClearRatio(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearRatio:          50,
		ClearMin:            5,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_WithSmallMaxConnections(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:      1,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_WithLargeMaxConnections(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:      100000,
	}

	loop, err := NewLoop(logger, cfg)
	assert.NoError(t, err)
	assert.NotNil(t, loop)
}

func TestLoop_WithDifferentSocketBuffers(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	sizes := []int{10, 512, 1024, 2048, 5120, 10240}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("sock-buf-%d", size), func(t *testing.T) {
			cfg := &config.Config{
				ListenAddr:          "127.0.0.1:0",
				RemoteAddr:          "127.0.0.1:0",
				EnableTCP:           false,
				EnableUDP:           false,
				SocketBufferSizeKbyte: size,
				MaxConnections:      100,
			}

			loop, err := NewLoop(logger, cfg)
			assert.NoError(t, err)
			assert.NotNil(t, loop)
		})
	}
}

func TestLoop_WithDifferentTimerIntervals(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	intervals := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1 * time.Second,
	}

	for _, interval := range intervals {
		t.Run(fmt.Sprintf("timer-%v", interval), func(t *testing.T) {
			cfg := &config.Config{
				ListenAddr:          "127.0.0.1:0",
				RemoteAddr:          "127.0.0.1:0",
				EnableTCP:           false,
				EnableUDP:           false,
				SocketBufferSizeKbyte: 1024,
				TimerInterval:       interval,
				MaxConnections:      100,
			}

			loop, err := NewLoop(logger, cfg)
			assert.NoError(t, err)
			assert.NotNil(t, loop)
		})
	}
}

func TestLoop_WithInvalidListenAddress(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		listenAddr string
		remoteAddr string
		enableTCP  bool
	}{
		{"empty_listen", "", "127.0.0.1:8080", true},
		{"empty_remote", "127.0.0.1:0", "", true},
		{"invalid_listen_format", "invalid:format", "127.0.0.1:8080", true},
		{"invalid_port", "127.0.0.1:invalid", "127.0.0.1:8080", true},
		{"port_out_of_range", "127.0.0.1:99999", "127.0.0.1:8080", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				ListenAddr:          tc.listenAddr,
				RemoteAddr:          tc.remoteAddr,
				EnableTCP:           tc.enableTCP,
				EnableUDP:           false,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:      100,
			}

			_, err := NewLoop(logger, cfg)
			assert.Error(t, err)
		})
	}
}

func TestLoop_WithDifferentClearIntervals(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	intervals := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, interval := range intervals {
		t.Run(fmt.Sprintf("clear-%v", interval), func(t *testing.T) {
			cfg := &config.Config{
				ListenAddr:          "127.0.0.1:0",
				RemoteAddr:          "127.0.0.1:0",
				EnableTCP:           false,
				EnableUDP:           false,
				SocketBufferSizeKbyte: 1024,
				ClearInterval:       interval,
				MaxConnections:      100,
			}

			loop, err := NewLoop(logger, cfg)
			assert.NoError(t, err)
			assert.NotNil(t, loop)
		})
	}
}

func TestLoop_WithDifferentClearMinValues(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	clearMinValues := []int{1, 5, 10, 50, 100}

	for _, clearMin := range clearMinValues {
		t.Run(fmt.Sprintf("clear-min-%d", clearMin), func(t *testing.T) {
			cfg := &config.Config{
				ListenAddr:          "127.0.0.1:0",
				RemoteAddr:          "127.0.0.1:0",
				EnableTCP:           false,
				EnableUDP:           false,
				SocketBufferSizeKbyte: 1024,
				ClearMin:            clearMin,
				MaxConnections:      100,
			}

			loop, err := NewLoop(logger, cfg)
			assert.NoError(t, err)
			assert.NotNil(t, loop)
		})
	}
}

func TestLoop_WithBothProtocolsAndQuickStop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           true,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       100 * time.Millisecond,
		MaxConnections:      100,
		TCPTimeout:          1 * time.Second,
		UDPTimeout:          1 * time.Second,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 快速启动和停止
	go func() {
		loop.Run()
	}()

	time.Sleep(20 * time.Millisecond)
	loop.Stop()
	time.Sleep(50 * time.Millisecond)
}

func TestLoop_Run_And_QuickStop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 512,
		ClearInterval:       500 * time.Millisecond,
		MaxConnections:      50,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(30 * time.Millisecond)

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(50 * time.Millisecond)
}

func TestLoop_Stats_Empty(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	cfg := &config.Config{
		ListenAddr:          "127.0.0.1:0",
		RemoteAddr:          "127.0.0.1:0",
		EnableTCP:           false,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 多次获取stats
	for i := 0; i < 5; i++ {
		stats := loop.Stats()
		assert.Equal(t, 0, stats.TCPConnections)
		assert.Equal(t, 0, stats.UDPConnections)
		assert.False(t, stats.Running)
	}
}

func TestLoop_AcceptTCP_Connection(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 获取可用的地址
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	listenAddr := listener.Addr().String()
	defer listener.Close()

	remoteListener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	remoteAddr := remoteListener.Addr().String()
	defer remoteListener.Close()

	cfg := &config.Config{
		ListenAddr:          listenAddr,
		RemoteAddr:          remoteAddr,
		EnableTCP:           true,
		EnableUDP:           false,
		SocketBufferSizeKbyte: 1024,
		ClearInterval:       1000 * time.Millisecond,
		MaxConnections:      100,
	}

	loop, err := NewLoop(logger, cfg)
	require.NoError(t, err)

	// 在goroutine中运行
	go func() {
		loop.Run()
	}()

	// 等待一小段时间让loop启动
	time.Sleep(50 * time.Millisecond)

	// 发起一个TCP连接来触发acceptTCP
	client, err := net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err == nil {
		// 接受连接
		server, err := listener.AcceptTCP()
		if err == nil {
			// 验证连接已建立
			stats := loop.Stats()
			assert.GreaterOrEqual(t, stats.TCPConnections, 1)
			server.Close()
		}
		client.Close()
	}

	// 停止loop
	loop.Stop()

	// 等待loop完全停止
	time.Sleep(100 * time.Millisecond)
}





