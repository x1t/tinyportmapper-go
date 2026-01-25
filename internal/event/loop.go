package event

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	connPkg "github.com/x1t/tinyportmapper-go/internal/conn"
	"github.com/x1t/tinyportmapper-go/internal/cleanup"
	"github.com/x1t/tinyportmapper-go/internal/config"
	"github.com/x1t/tinyportmapper-go/internal/fd"
	"github.com/x1t/tinyportmapper-go/internal/forward"
	"github.com/x1t/tinyportmapper-go/internal/log"
	"github.com/x1t/tinyportmapper-go/internal/types"
	"go.uber.org/zap"
)

// Loop 事件循环
// 对应 C++ 中的 event_loop() 函数
type Loop struct {
	logger   *log.Logger
	cfg      *config.Config

	// 地址
	localAddr  *types.Address
	remoteAddr *types.Address

	// 监听 socket
	tcpListener *net.TCPListener
	udpConn     *net.UDPConn

	// 远程 UDP 连接
	remoteUDPConn *net.UDPConn

	// 管理器
	fdManager     *fd.Manager
	tcpForwarders *forward.TCPForwarderManager
	udpForwarders *forward.UDPForwarderManager
	udpManager    *connPkg.UDPManager
	cleanupMgr    *cleanup.Manager

	// 状态
	running   int32
	stopCh    chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewLoop 创建新的事件循环
func NewLoop(logger *log.Logger, cfg *config.Config) (*Loop, error) {
	localAddr, err := types.NewAddressFromString(cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("解析监听地址失败: %w", err)
	}

	remoteAddr, err := types.NewAddressFromString(cfg.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("解析远程地址失败: %w", err)
	}

	cleanupCfg := &cleanup.Config{
		TCPTimeout:    cfg.TCPTimeout,
		UDPTimeout:    cfg.UDPTimeout,
		ClearInterval: cfg.ClearInterval,
		ClearRatio:    cfg.ClearRatio,
		MinClear:      1,
		MaxConns:      cfg.MaxConnections,
		DisableClear:  cfg.DisableConnClear, // 传递禁用连接清理配置
	}

	ctx, cancel := context.WithCancel(context.Background())

	loop := &Loop{
		logger:   logger,
		cfg:      cfg,
		localAddr: localAddr,
		remoteAddr: remoteAddr,
		fdManager: fd.New(-1),
		tcpForwarders: forward.NewTCPForwarderManager(cfg.SocketBufferBytes(), cfg.MaxConnections),
		udpForwarders: forward.NewUDPForwarderManager(),
		udpManager:    connPkg.NewUDPManager(cfg.MaxConnections),
		cleanupMgr:    cleanup.NewManager(logger, cleanupCfg),
		stopCh:        make(chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}

	// 设置 TCP 清理回调 - 连接超时时关闭转发器
	loop.cleanupMgr.SetTCPCleanupFn(func(key, value interface{}) {
		if tcpConn, ok := key.(*connPkg.TCPConn); ok {
			logger.Debug("TCP 连接超时,关闭连接",
				zap.String("addr", tcpConn.Addr))
			tcpConn.Close()
		}
	})

	// 设置 UDP 清理回调 - 连接超时时关闭转发器
	loop.cleanupMgr.SetUDPCleanupFn(func(key, value interface{}) {
		if forwarder, ok := value.(*forward.UDPForwarder); ok {
			logger.Debug("UDP 连接超时,关闭转发器",
				zap.String("client", forwarder.GetClientAddr()))
			forwarder.Close()
		}
	})

	return loop, nil
}

// Run 启动事件循环
// 对应 C++ 中的 event_loop() 函数
func (l *Loop) Run() error {
	if atomic.CompareAndSwapInt32(&l.running, 0, 1) {
		defer atomic.StoreInt32(&l.running, 0)
	}

	// 启动清理管理器
	l.cleanupMgr.Start()
	defer l.cleanupMgr.Stop()

	// 启动 TCP 监听
	if l.cfg.EnableTCP {
		if err := l.startTCPListener(); err != nil {
			return fmt.Errorf("启动 TCP 监听失败: %w", err)
		}
		defer l.tcpListener.Close()
	}

	// 启动 UDP 监听
	if l.cfg.EnableUDP {
		if err := l.startUDPListener(); err != nil {
			return fmt.Errorf("启动 UDP 监听失败: %w", err)
		}
		defer l.udpConn.Close()
	}

	l.logger.Info("事件循环启动",
		zap.String("local", l.localAddr.String()),
		zap.String("remote", l.remoteAddr.String()),
		zap.Bool("tcp", l.cfg.EnableTCP),
		zap.Bool("udp", l.cfg.EnableUDP),
	)

	// 等待停止信号
	<-l.stopCh

	l.logger.Info("事件循环停止")
	return nil
}

// startTCPListener 启动 TCP 监听
// 对应 C++ 中的 tcp_accept_cb 回调注册
func (l *Loop) startTCPListener() error {
	tcpAddr := l.localAddr.ToTCPAddr()
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("创建 TCP 监听失败: %w", err)
	}

	// 设置监听参数
	// TCPListener 不支持 SetNoDelay,需要在 accept 后的连接上设置

	l.tcpListener = listener

	// 启动接受连接 goroutine
	l.wg.Add(1)
	go l.acceptTCP()

	return nil
}

// acceptTCP 接受 TCP 连接
// 对应 C++ 中的 tcp_accept_cb 回调函数
func (l *Loop) acceptTCP() {
	defer l.wg.Done()

	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		// 设置非阻塞并设置截止时间
		l.tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond))

		conn, err := l.tcpListener.AcceptTCP()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return
		}

		// 设置连接参数
		conn.SetReadBuffer(l.cfg.SocketBufferBytes())
		conn.SetWriteBuffer(l.cfg.SocketBufferBytes())
		conn.SetNoDelay(true)

		// 检查连接数
		if l.tcpForwarders.Count() >= l.cfg.MaxConnections {
			l.logger.Warn("达到最大连接数,忽略新连接")
			conn.Close()
			continue
		}

		// 连接到远程
		remoteConn, err := net.DialTCP("tcp", nil, l.remoteAddr.ToTCPAddr())
		if err != nil {
			l.logger.Warn("连接远程失败", zap.Error(err))
			conn.Close()
			continue
		}

		// 设置远程连接参数
		remoteConn.SetReadBuffer(l.cfg.SocketBufferBytes())
		remoteConn.SetWriteBuffer(l.cfg.SocketBufferBytes())
		remoteConn.SetNoDelay(true)

		// 创建连接对
		tcpConn := connPkg.NewTCPConn(conn, remoteConn)

		// 创建转发器
		// src是本地连接(tcpConn.Local)，dst是远程连接(tcpConn.Remote)
		forwarder := forward.NewTCPForwarder(tcpConn.Local, tcpConn.Remote, l.cfg.SocketBufferBytes(), l.cfg.TCPTimeout, func() {
			l.cleanupMgr.RemoveTCP(tcpConn)
		})

		// 添加到清理管理器
		l.cleanupMgr.AddTCP(tcpConn, tcpConn)

		// 启动转发
		l.tcpForwarders.Start(forwarder, func() {
			l.logger.Debug("TCP 连接关闭",
				zap.String("addr", tcpConn.Addr),
			)
		})

		l.logger.Info("新 TCP 连接",
			zap.String("local", conn.LocalAddr().String()),
			zap.String("remote", conn.RemoteAddr().String()),
			zap.Int("total", l.tcpForwarders.Count()),
		)
	}
}

// startUDPListener 启动 UDP 监听
// 对应 C++ 中的 udp_accept_cb 回调注册
func (l *Loop) startUDPListener() error {
	udpAddr := l.localAddr.ToUDPAddr()

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("创建 UDP 监听失败: %w", err)
	}

	// 设置缓冲区大小
	conn.SetReadBuffer(l.cfg.SocketBufferBytes())
	conn.SetWriteBuffer(l.cfg.SocketBufferBytes())

	l.udpConn = conn

	// 启动 UDP 处理 goroutine
	l.wg.Add(1)
	go l.handleUDP(conn)

	return nil
}

// handleUDP 处理 UDP 数据包
// 对应 C++ 中的 udp_accept_cb 和 udp_cb 回调函数
// 注意：每个客户端使用独立的 UDP 套接字连接到远程服务器
func (l *Loop) handleUDP(localConn *net.UDPConn) {
	defer l.wg.Done()

	buf := make([]byte, 64*1024) // max_data_len_udp + 200

	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		// 设置截止时间
		localConn.SetReadDeadline(time.Now().Add(1 * time.Minute))

		n, addr, err := localConn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return
		}

		// 检查数据包大小
		if n > 65536 {
			l.logger.Warn("UDP 数据包过大,丢弃",
				zap.Int("size", n),
			)
			continue
		}

		// 解析客户端地址
		clientAddr, ok := addr.(*net.UDPAddr)
		if !ok {
			continue
		}
		clientTypesAddr := types.NewAddressFromUDPAddr(clientAddr)

		// 检查是否已有该客户端的转发器
		forwarder, exists := l.udpForwarders.Get(clientTypesAddr)

		if !exists {
			// 检查连接数
			if l.udpForwarders.Count() >= l.cfg.MaxConnections {
				l.logger.Debug("达到最大 UDP 连接数,忽略")
				continue
			}

			// 为该客户端创建独立的 UDP 套接字（对应 C++ 的 new_connected_udp_fd()）
			remoteUDP, err := net.DialUDP("udp", nil, l.remoteAddr.ToUDPAddr())
			if err != nil {
				l.logger.Warn("创建客户端 UDP 连接失败",
					zap.String("client", clientTypesAddr.String()),
					zap.Error(err),
				)
				continue
			}

			remoteUDP.SetReadBuffer(l.cfg.SocketBufferBytes())
			remoteUDP.SetWriteBuffer(l.cfg.SocketBufferBytes())

			// 获取本地套接字文件描述符
			localFile, err := localConn.File()
			if err != nil {
				l.logger.Warn("获取本地 UDP 连接文件描述符失败")
				remoteUDP.Close()
				continue
			}

			// 获取远程套接字文件描述符
			remoteFile, err := remoteUDP.File()
			if err != nil {
				l.logger.Warn("获取远程 UDP 连接文件描述符失败")
				remoteUDP.Close()
				continue
			}

			// 创建新的 UDP 连接对
			udpConn := connPkg.NewUDPConn(
				clientTypesAddr,
				int(localFile.Fd()),
				int(remoteFile.Fd()),
				uint64(remoteFile.Fd()),
			)

			// 添加到管理器
			if !l.udpManager.Add(udpConn) {
				l.logger.Warn("添加 UDP 连接失败")
				remoteUDP.Close()
				continue
			}

			// 创建转发器
			forwarder = forward.NewUDPForwarder(localConn, remoteUDP, clientTypesAddr, udpConn, 64*1024)
			forwarder.Start()

			// 添加到转发器管理器
			l.udpForwarders.Add(forwarder)

			// 添加到清理管理器
			l.cleanupMgr.AddUDP(forwarder, forwarder)

			l.logger.Info("新 UDP 连接",
				zap.String("client", clientTypesAddr.String()),
				zap.Int("total", l.udpForwarders.Count()),
			)
		}

		// 更新活跃时间
		l.cleanupMgr.TouchUDP(forwarder)

		// 转发到远程（通过 forwarder 转发以保持一致性）
		forwarder.ForwardToRemote(buf[:n])
	}
}

// Stop 停止事件循环
func (l *Loop) Stop() {
	if atomic.CompareAndSwapInt32(&l.running, 1, 0) {
		close(l.stopCh)
		l.cancel()
		if l.remoteUDPConn != nil {
			l.remoteUDPConn.Close()
		}
	}
}

// Wait 等待事件循环完成
func (l *Loop) Wait() {
	l.wg.Wait()
}

// Stats 返回统计信息
type Stats struct {
	TCPConnections int
	UDPConnections int
	Running        bool
}

// Stats 返回当前统计信息
func (l *Loop) Stats() Stats {
	return Stats{
		TCPConnections: l.tcpForwarders.Count(),
		UDPConnections: l.udpForwarders.Count(),
		Running:        atomic.LoadInt32(&l.running) == 1,
	}
}
