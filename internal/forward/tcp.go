package forward

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// TCP buffer 大小，与 C++ 版本一致
const DefaultTCPBufferSize = 16 * 1024 // 16384 bytes = 16KB, 与 C++ 的 max_data_len_tcp 一致

// TCPForwarder TCP 数据转发器
// 对应 C++ 中的 tcp_cb 回调函数
type TCPForwarder struct {
	src      *net.TCPConn
	dst      *net.TCPConn
	buf      []byte
	bufSize  int
	wg       sync.WaitGroup
	done     chan struct{}
	onClose  func()
	closed   int32
	mu       sync.Mutex
	readTimeout time.Duration // 使用配置的超时值，避免硬编码30分钟
}

// NewTCPForwarder 创建新的 TCP 转发器
// readTimeout: 读取超时时间，使用配置的TCPTimeout，避免硬编码30分钟
func NewTCPForwarder(src, dst *net.TCPConn, bufSize int, readTimeout time.Duration, onClose func()) *TCPForwarder {
	if bufSize <= 0 {
		bufSize = DefaultTCPBufferSize // 使用与 C++ 一致的默认值 16KB
	}
	if readTimeout <= 0 {
		readTimeout = 5 * time.Minute // 默认5分钟，与C++行为更接近
	}
	return &TCPForwarder{
		src:         src,
		dst:         dst,
		buf:         make([]byte, bufSize),
		bufSize:     bufSize,
		done:        make(chan struct{}),
		onClose:     onClose,
		readTimeout: readTimeout,
	}
}

// Start 开始双向转发
// 对应 C++ 中的 ev_io_start(loop, &tcp_pair.local.ev)
// 对应 C++ 中的 ev_io_start(loop, &tcp_pair.remote.ev)
func (f *TCPForwarder) Start() {
	f.wg.Add(2)
	go f.copySrcToDst()
	go f.copyDstToSrc()
}

// copySrcToDst 从源复制到目标
// 对应 C++ 中的 EV_READ 处理逻辑
func (f *TCPForwarder) copySrcToDst() {
	defer func() {
		if f.src != nil {
			f.src.CloseRead()
		}
		f.wg.Done()
	}()

	for {
		// 检查 src 是否为 nil
		if f.src == nil {
			return
		}

		// 使用配置的超时值，避免硬编码30分钟
		f.src.SetReadDeadline(time.Now().Add(f.readTimeout))

		n, err := f.src.Read(f.buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return
		}

		// 写入目标
		written, err := f.dst.Write(f.buf[:n])
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// 临时错误,重试
				f.handlePartialWrite(f.buf[:n], written)
			}
			return
		}
		if written != n {
			f.handlePartialWrite(f.buf[written:], n-written)
		}
	}
}

// copyDstToSrc 从目标复制到源
// 对应 C++ 中的 EV_WRITE 处理逻辑
func (f *TCPForwarder) copyDstToSrc() {
	defer func() {
		if f.dst != nil {
			f.dst.CloseWrite()
		}
		f.wg.Done()
	}()

	for {
		// 检查 dst 是否为 nil
		if f.dst == nil {
			return
		}

		// 使用配置的超时值，避免硬编码30分钟
		f.dst.SetReadDeadline(time.Now().Add(f.readTimeout))

		n, err := f.dst.Read(f.buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return
		}

		// 写入源（从远程读取后写入本地连接）
		written, err := f.src.Write(f.buf[:n])
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				f.handlePartialWrite(f.buf[:n], written)
			}
			return
		}
		if written != n {
			f.handlePartialWrite(f.buf[written:], n-written)
		}
	}
}

// handlePartialWrite 处理部分写入
// 对应 C++ 中的 ev.events |= EV_WRITE 处理逻辑
func (f *TCPForwarder) handlePartialWrite(data []byte, written int) {
	total := len(data)
	remaining := total - written
	offset := written
	for remaining > 0 {
		n, err := f.dst.Write(data[offset:])
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return
		}
		offset += n
		remaining -= n
	}
}

// Wait 等待转发完成
// 对应 C++ 中的 ev_run 等待
func (f *TCPForwarder) Wait() {
	f.wg.Wait()
	f.Close()
}

// Close 关闭转发器
func (f *TCPForwarder) Close() {
	if atomic.CompareAndSwapInt32(&f.closed, 0, 1) {
		close(f.done)
		if f.onClose != nil {
			f.onClose()
		}
	}
}

// IsClosed 检查是否已关闭
func (f *TCPForwarder) IsClosed() bool {
	return atomic.LoadInt32(&f.closed) == 1
}

// TCPForwarderPool TCP 转发器池
type TCPForwarderPool struct {
	pool     sync.Pool
	bufSize  int
	maxConns int
}

// NewTCPForwarderPool 创建新的转发器池
func NewTCPForwarderPool(bufSize, maxConns int) *TCPForwarderPool {
	if bufSize <= 0 {
		bufSize = DefaultTCPBufferSize
	}
	return &TCPForwarderPool{
		bufSize:  bufSize,
		maxConns: maxConns,
		pool: sync.Pool{
			New: func() interface{} {
				return &TCPForwarder{
					buf: make([]byte, bufSize),
				}
			},
		},
	}
}

// Get 从池中获取转发器
func (p *TCPForwarderPool) Get() *TCPForwarder {
	return p.pool.Get().(*TCPForwarder)
}

// Put 将转发器归还池中
func (p *TCPForwarderPool) Put(f *TCPForwarder) {
	f.buf = make([]byte, p.bufSize)
	f.done = make(chan struct{})
	f.closed = 0
	p.pool.Put(f)
}

// TCPForwarderManager TCP 转发器管理器
type TCPForwarderManager struct {
	mu        sync.RWMutex
	forwarders map[*TCPForwarder]struct{}
	pool       *TCPForwarderPool
	wg         sync.WaitGroup
}

// NewTCPForwarderManager 创建新的转发器管理器
func NewTCPForwarderManager(bufSize, maxConns int) *TCPForwarderManager {
	return &TCPForwarderManager{
		forwarders: make(map[*TCPForwarder]struct{}),
		pool:       NewTCPForwarderPool(bufSize, maxConns),
	}
}

// Add 添加转发器
func (m *TCPForwarderManager) Add(f *TCPForwarder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forwarders[f] = struct{}{}
}

// Remove 移除转发器
func (m *TCPForwarderManager) Remove(f *TCPForwarder) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.forwarders, f)
}

// Count 返回转发器数量
func (m *TCPForwarderManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.forwarders)
}

// CloseAll 关闭所有转发器
func (m *TCPForwarderManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for f := range m.forwarders {
		f.Close()
	}
	m.forwarders = make(map[*TCPForwarder]struct{})
}

// WaitAll 等待所有转发器完成
func (m *TCPForwarderManager) WaitAll() {
	m.wg.Wait()
}

// Start 启动转发器并添加到管理器
func (m *TCPForwarderManager) Start(f *TCPForwarder, onClose func()) {
	if f == nil {
		return
	}
	f.onClose = func() {
		m.Remove(f)
		onClose()
	}
	m.Add(f)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		f.Start()
	}()
}
