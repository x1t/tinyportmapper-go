package signal

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/x1t/tinyportmapper-go/internal/log"
)

func TestNewHandler(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	// 创建信号处理器
	handler := NewHandler(logger)
	assert.NotNil(t, handler)
}

func TestHandler_Stop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 直接停止，不运行
	handler.Stop()

	// 应该可以再次停止而不panic
	handler.Stop()
}

func TestHandler_Handle_SIGPIPE(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 使用SIGPIPE测试，它不会导致退出
	assert.NotPanics(t, func() {
		handler.handle(syscall.SIGPIPE)
	})
}

func TestHandler_Handle_SIGTERM(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 使用SIGTERM测试，它应该关闭quit channel而不是调用os.Exit
	assert.NotPanics(t, func() {
		handler.handle(syscall.SIGTERM)
	})

	// 验证quit channel已关闭
	select {
	case <-handler.quit:
		// 预期行为
	default:
		t.Error("quit channel should be closed after SIGTERM")
	}
}

func TestHandler_Handle_SIGINT(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 使用SIGINT测试，它应该关闭quit channel而不是调用os.Exit
	assert.NotPanics(t, func() {
		handler.handle(syscall.SIGINT)
	})

	// 验证quit channel已关闭
	select {
	case <-handler.quit:
		// 预期行为
	default:
		t.Error("quit channel should be closed after SIGINT")
	}
}

func TestHandler_Run_And_Stop(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 在goroutine中运行handler
	go handler.Run()

	// 等待一下让handler启动
	time.Sleep(10 * time.Millisecond)

	// 停止handler
	handler.Stop()

	// 等待handler退出
	handler.Wait()
}

func TestHandler_Wait_NoStart(t *testing.T) {
	logger, err := log.New(log.LogLevelInfo, false, false)
	require.NoError(t, err)

	handler := NewHandler(logger)

	// 在没有运行的情况下调用Wait应该阻塞，但这里我们不测试阻塞
	// 只是验证它可以被调用
	done := make(chan struct{})
	go func() {
		handler.Wait()
		close(done)
	}()

	// 等待一小段时间，确认Wait没有立即返回（因为没有运行）
	select {
	case <-done:
		t.Error("Wait should block when Run() is not called")
	case <-time.After(50 * time.Millisecond):
		// 预期行为：Wait正在阻塞
	}

	// 停止handler（它还没有运行，但应该安全）
	handler.Stop()
}
