package conn

import (
	"bytes"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCPConn TCP 连接对
// 对应 C++ 中的 tcp_pair_t
type TCPConn struct {
	Local  *net.TCPConn  // 本地连接
	Remote *net.TCPConn  // 远程连接
	Addr   string        // 远端地址字符串

	lastActive int64  // 最后活跃时间 (Unix ms)
	refCount   int32  // 引用计数
	closed     int32  // 是否已关闭
	mu         sync.Mutex
}

// NewTCPConn 创建新的 TCP 连接对
// 对应 C++ 中的 tcp_accept_cb 中的连接创建逻辑
func NewTCPConn(local, remote *net.TCPConn) *TCPConn {
	addr := ""
	if remote != nil {
		addr = remote.RemoteAddr().String()
	}
	return &TCPConn{
		Local:      local,
		Remote:     remote,
		Addr:       addr,
		lastActive: time.Now().UnixMilli(),
	}
}

// UpdateActivity 更新最后活跃时间
// 对应 C++ 中的 lru.update()
func (c *TCPConn) UpdateActivity() {
	atomic.StoreInt64(&c.lastActive, time.Now().UnixMilli())
}

// LastActive 获取最后活跃时间
func (c *TCPConn) LastActive() int64 {
	return atomic.LoadInt64(&c.lastActive)
}

// IncRef 增加引用计数
func (c *TCPConn) IncRef() {
	atomic.AddInt32(&c.refCount, 1)
}

// DecRef 减少引用计数,返回是否应该释放
func (c *TCPConn) DecRef() bool {
	return atomic.AddInt32(&c.refCount, -1) == 0
}

// RefCnt 获取引用计数
func (c *TCPConn) RefCnt() int32 {
	return atomic.LoadInt32(&c.refCount)
}

// IsClosed 检查是否已关闭
func (c *TCPConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

// Close 关闭连接对
// 对应 C++ 中的 conn_manager_tcp.erase()
func (c *TCPConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		var err1, err2 error
		if c.Local != nil {
			err1 = c.Local.Close()
			c.Local = nil
		}
		if c.Remote != nil {
			err2 = c.Remote.Close()
			c.Remote = nil
		}
		if err1 != nil {
			return err1
		}
		return err2
	}
	return nil
}

// CloseRead 关闭读端
func (c *TCPConn) CloseRead() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Local != nil {
		return c.Local.CloseRead()
	}
	return nil
}

// CloseWrite 关闭写端
func (c *TCPConn) CloseWrite() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Remote != nil {
		return c.Remote.CloseWrite()
	}
	return nil
}

// Read 从本地连接读取
func (c *TCPConn) Read(b []byte) (int, error) {
	if c.Local == nil {
		return 0, ErrConnectionClosed
	}
	return c.Local.Read(b)
}

// ReadFromRemote 从远程连接读取
func (c *TCPConn) ReadFromRemote(b []byte) (int, error) {
	if c.Remote == nil {
		return 0, ErrConnectionClosed
	}
	return c.Remote.Read(b)
}

// Write 写入本地连接
func (c *TCPConn) Write(b []byte) (int, error) {
	if c.Local == nil {
		return 0, ErrConnectionClosed
	}
	return c.Local.Write(b)
}

// WriteToRemote 写入远程连接
func (c *TCPConn) WriteToRemote(b []byte) (int, error) {
	if c.Remote == nil {
		return 0, ErrConnectionClosed
	}
	return c.Remote.Write(b)
}

// SetReadBuffer 设置读取缓冲区大小
// 对应 C++ 中的 set_buf_size()
func (c *TCPConn) SetReadBuffer(bytes int) error {
	if c.Local != nil {
		if err := c.Local.SetReadBuffer(bytes); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetReadBuffer(bytes)
	}
	return nil
}

// SetWriteBuffer 设置写入缓冲区大小
func (c *TCPConn) SetWriteBuffer(bytes int) error {
	if c.Local != nil {
		if err := c.Local.SetWriteBuffer(bytes); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetWriteBuffer(bytes)
	}
	return nil
}

// SetNoDelay 设置 Nagle 算法
func (c *TCPConn) SetNoDelay(noDelay bool) error {
	if c.Local != nil {
		if err := c.Local.SetNoDelay(noDelay); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetNoDelay(noDelay)
	}
	return nil
}

// SetDeadline 设置读写截止时间
func (c *TCPConn) SetDeadline(t time.Time) error {
	if c.Local != nil {
		if err := c.Local.SetDeadline(t); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetDeadline(t)
	}
	return nil
}

// SetReadDeadline 设置读取截止时间
func (c *TCPConn) SetReadDeadline(t time.Time) error {
	// 依次设置本地和远程的读取截止时间
	if c.Local != nil {
		if err := c.Local.SetReadDeadline(t); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetReadDeadline(t)
	}
	return nil
}

// SetWriteDeadline 设置写入截止时间
func (c *TCPConn) SetWriteDeadline(t time.Time) error {
	if c.Local != nil {
		if err := c.Local.SetWriteDeadline(t); err != nil {
			return err
		}
	}
	if c.Remote != nil {
		return c.Remote.SetWriteDeadline(t)
	}
	return nil
}

// RemoteAddr 返回远程地址
func (c *TCPConn) RemoteAddr() net.Addr {
	if c.Remote != nil {
		return c.Remote.RemoteAddr()
	}
	return nil
}

// LocalAddr 返回本地地址
func (c *TCPConn) LocalAddr() net.Addr {
	if c.Local != nil {
		return c.Local.LocalAddr()
	}
	return nil
}

// BufferPool TCP 数据缓冲区池
// 对应 C++ 中的 tcp_info_t::data[max_data_len_tcp+200]
var BufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer 从池中获取缓冲区
func GetBuffer() *bytes.Buffer {
	return BufferPool.Get().(*bytes.Buffer)
}

// PutBuffer 将缓冲区归还池中
func PutBuffer(b *bytes.Buffer) {
	b.Reset()
	BufferPool.Put(b)
}

// ErrConnectionClosed 连接已关闭错误
var ErrConnectionClosed = &ConnectionClosedError{}

// ConnectionClosedError 连接已关闭错误
type ConnectionClosedError struct{}

func (e *ConnectionClosedError) Error() string {
	return "连接已关闭"
}
