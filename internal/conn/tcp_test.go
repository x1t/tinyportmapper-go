package conn

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTCPConn(t *testing.T) {
	// 创建一对测试连接
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	// 创建 TCPConn
	tcpConn := NewTCPConn(client, server)

	assert.NotNil(t, tcpConn)
	assert.Equal(t, client, tcpConn.Local)
	assert.Equal(t, server, tcpConn.Remote)
	assert.NotEmpty(t, tcpConn.Addr)
	assert.NotZero(t, tcpConn.LastActive())
}

func TestTCPConn_UpdateActivity(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)
	initialActive := tcpConn.LastActive()

	// 等待一下
	time.Sleep(10 * time.Millisecond)

	// 更新活跃时间
	tcpConn.UpdateActivity()
	newActive := tcpConn.LastActive()

	assert.Greater(t, newActive, initialActive)
}

func TestTCPConn_RefCounting(t *testing.T) {
	// 创建一对测试连接以避免nil问题
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 初始引用计数为0
	assert.Equal(t, int32(0), tcpConn.RefCnt())

	// 增加引用
	tcpConn.IncRef()
	assert.Equal(t, int32(1), tcpConn.RefCnt())

	tcpConn.IncRef()
	assert.Equal(t, int32(2), tcpConn.RefCnt())

	// 减少引用
	assert.False(t, tcpConn.DecRef())
	assert.Equal(t, int32(1), tcpConn.RefCnt())

	assert.True(t, tcpConn.DecRef())
	assert.Equal(t, int32(0), tcpConn.RefCnt())
}

func TestTCPConn_IsClosed(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)

	server, err := listener.AcceptTCP()
	require.NoError(t, err)

	tcpConn := NewTCPConn(client, server)

	assert.False(t, tcpConn.IsClosed())

	tcpConn.Close()
	assert.True(t, tcpConn.IsClosed())

	// 再次关闭不应该panic
	tcpConn.Close()
	assert.True(t, tcpConn.IsClosed())
}

func TestTCPConn_Close(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)

	server, err := listener.AcceptTCP()
	require.NoError(t, err)

	tcpConn := NewTCPConn(client, server)

	// 关闭连接
	err = tcpConn.Close()
	assert.NoError(t, err)
	assert.True(t, tcpConn.IsClosed())

	// 验证连接已关闭
	assert.Nil(t, tcpConn.Local)
	assert.Nil(t, tcpConn.Remote)
}

func TestTCPConn_CloseReadWrite(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 关闭读端
	err = tcpConn.CloseRead()
	assert.NoError(t, err)

	// 关闭写端
	err = tcpConn.CloseWrite()
	assert.NoError(t, err)
}

func TestTCPConn_ReadWriteClosed(t *testing.T) {
	tcpConn := NewTCPConn(nil, nil)

	// 对关闭的连接读取应该返回错误
	_, err := tcpConn.Read(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.ReadFromRemote(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.Write(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.WriteToRemote(make([]byte, 10))
	assert.Error(t, err)
}

func TestTCPConn_RemoteAddr(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 远程地址应该不为nil
	addr := tcpConn.RemoteAddr()
	assert.NotNil(t, addr)
	assert.Equal(t, server.RemoteAddr().String(), addr.String())

	// 本地地址
	localAddr := tcpConn.LocalAddr()
	assert.NotNil(t, localAddr)
}

func TestTCPConn_NilRemote(t *testing.T) {
	tcpConn := NewTCPConn(nil, nil)

	// 测试nil远程连接的方法
	addr := tcpConn.RemoteAddr()
	assert.Nil(t, addr)

	localAddr := tcpConn.LocalAddr()
	assert.Nil(t, localAddr)
}

func TestBufferPool(t *testing.T) {
	// 获取缓冲区
	buf := GetBuffer()
	assert.NotNil(t, buf)

	// 写入一些数据
	buf.WriteString("test data")
	assert.Equal(t, "test data", buf.String())

	// 归还缓冲区
	PutBuffer(buf)

	// 再次获取，应该可以复用
	buf2 := GetBuffer()
	assert.NotNil(t, buf2)
	assert.Empty(t, buf2.String()) // 已经被重置
}

func TestConnectionClosedError(t *testing.T) {
	err := ErrConnectionClosed
	assert.NotNil(t, err)
	assert.Equal(t, "连接已关闭", err.Error())
}

func TestTCPConn_SetReadBuffer(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置读取缓冲区
	err = tcpConn.SetReadBuffer(512 * 1024)
	assert.NoError(t, err)
}

func TestTCPConn_SetWriteBuffer(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置写入缓冲区
	err = tcpConn.SetWriteBuffer(512 * 1024)
	assert.NoError(t, err)
}

func TestTCPConn_SetNoDelay(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置 NoDelay
	err = tcpConn.SetNoDelay(true)
	assert.NoError(t, err)

	err = tcpConn.SetNoDelay(false)
	assert.NoError(t, err)
}

func TestTCPConn_SetDeadline(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置截止时间
	deadline := time.Now().Add(1 * time.Minute)
	err = tcpConn.SetDeadline(deadline)
	assert.NoError(t, err)
}

func TestTCPConn_SetReadDeadline(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置读取截止时间
	deadline := time.Now().Add(1 * time.Minute)
	err = tcpConn.SetReadDeadline(deadline)
	assert.NoError(t, err)
}

func TestTCPConn_SetWriteDeadline(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试设置写入截止时间
	deadline := time.Now().Add(1 * time.Minute)
	err = tcpConn.SetWriteDeadline(deadline)
	assert.NoError(t, err)
}

func TestTCPConn_WithNilLocal(t *testing.T) {
	// 测试nil本地连接的情况
	tcpConn := NewTCPConn(nil, nil)

	// 应该能创建成功
	assert.NotNil(t, tcpConn)

	// 各种操作应该返回错误
	_, err := tcpConn.Read(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.Write(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.ReadFromRemote(make([]byte, 10))
	assert.Error(t, err)

	_, err = tcpConn.WriteToRemote(make([]byte, 10))
	assert.Error(t, err)

	// 关闭应该安全
	err = tcpConn.Close()
	assert.NoError(t, err)

	err = tcpConn.CloseRead()
	assert.NoError(t, err)

	err = tcpConn.CloseWrite()
	assert.NoError(t, err)

	// 设置各种参数应该安全
	err = tcpConn.SetReadBuffer(1024)
	assert.NoError(t, err)

	err = tcpConn.SetWriteBuffer(1024)
	assert.NoError(t, err)

	err = tcpConn.SetNoDelay(true)
	assert.NoError(t, err)

	err = tcpConn.SetDeadline(time.Now().Add(1 * time.Minute))
	assert.NoError(t, err)

	// 地址应该为nil
	assert.Nil(t, tcpConn.RemoteAddr())
	assert.Nil(t, tcpConn.LocalAddr())
}

func TestTCPConn_WithOnlyLocal(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	// 创建只有Local连接的TCPConn
	tcpConn := NewTCPConn(server, nil)
	assert.NotNil(t, tcpConn)

	// RemoteAddr应该为nil
	assert.Nil(t, tcpConn.RemoteAddr())
	assert.NotNil(t, tcpConn.LocalAddr())
}

func TestBufferPool_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup

	// 并发获取和归还缓冲区
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := GetBuffer()
			buf.WriteString("test data")
			PutBuffer(buf)
		}()
	}

	wg.Wait()
}

func TestTCPConn_UpdateActivityMultipleTimes(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)
	initialActive := tcpConn.LastActive()

	// 等待一小段时间
	time.Sleep(10 * time.Millisecond)

	// 更新活跃时间多次
	for i := 0; i < 5; i++ {
		tcpConn.UpdateActivity()
	}

	newActive := tcpConn.LastActive()
	assert.Greater(t, newActive, initialActive)
}

func TestTCPConn_RefCountEdgeCases(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer listener.Close()

	client, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
	require.NoError(t, err)
	defer client.Close()

	server, err := listener.AcceptTCP()
	require.NoError(t, err)
	defer server.Close()

	tcpConn := NewTCPConn(client, server)

	// 测试引用计数从0增加到较大值
	for i := int32(0); i < 100; i++ {
		tcpConn.IncRef()
		assert.Equal(t, i+1, tcpConn.RefCnt())
	}

	// 测试引用计数减少到0
	for i := int32(100); i > 0; i-- {
		result := tcpConn.DecRef()
		if i == 1 {
			assert.True(t, result)
		} else {
			assert.False(t, result)
		}
	}
}
