package signal

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/x1t/tinyportmapper-go/internal/log"
	"go.uber.org/zap"
)

// Handler 信号处理器
// 对应 C++ 中的 sigterm_cb, sigint_cb, sigpipe_cb
type Handler struct {
	logger   *log.Logger
	quit     chan struct{}
	sigCh    chan os.Signal
	exited   chan struct{} // 退出通知 channel
	closeMu  sync.Mutex    // 保护close操作
}

// NewHandler 创建新的信号处理器
func NewHandler(logger *log.Logger) *Handler {
	return &Handler{
		logger: logger,
		quit:   make(chan struct{}),
		sigCh:  make(chan os.Signal, 1),
		exited: make(chan struct{}),
	}
}

// Run 启动信号处理循环
func (h *Handler) Run() {
	// 注册信号处理
	signal.Notify(h.sigCh, syscall.SIGTERM, syscall.SIGINT, syscall.SIGPIPE)

	defer close(h.exited)

	for {
		select {
		case <-h.quit:
			signal.Stop(h.sigCh)
			close(h.sigCh)
			return
		case sig := <-h.sigCh:
			h.handle(sig)
		}
	}
}

// Wait 等待信号处理器退出
func (h *Handler) Wait() {
	<-h.exited
}

// handle 处理信号
func (h *Handler) handle(sig os.Signal) {
	switch s := sig.(type) {
	case syscall.Signal:
		switch s {
		case syscall.SIGTERM:
			h.logger.Info("收到 SIGTERM 信号, 准备退出")
			h.safeCloseQuit()
		case syscall.SIGINT:
			h.logger.Info("收到 SIGINT 信号, 准备退出")
			h.safeCloseQuit()
		case syscall.SIGPIPE:
			h.logger.Debug("收到 SIGPIPE 信号, 忽略")
		default:
			h.logger.Warn("收到未知信号", zap.Any("signal", sig))
		}
	default:
		h.logger.Warn("收到未知信号类型", zap.Any("signal", sig))
	}
}

// safeCloseQuit 安全地关闭quit channel，防止重复关闭
func (h *Handler) safeCloseQuit() {
	h.closeMu.Lock()
	defer h.closeMu.Unlock()
	select {
	case <-h.quit:
		// 已經關閉
		return
	default:
		close(h.quit)
	}
}

// Stop 停止信号处理
func (h *Handler) Stop() {
	h.safeCloseQuit()
}
