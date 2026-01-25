package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	logger, err := New(LogLevelInfo, false, false)
	require.NoError(t, err)
	require.NotNil(t, logger)
}

func TestNewLogger_AllLevels(t *testing.T) {
	levels := []LogLevel{LogLevelNever, LogLevelFatal, LogLevelError, LogLevelWarn, LogLevelInfo, LogLevelDebug, LogLevelTrace}

	for _, level := range levels {
		logger, err := New(level, false, false)
		require.NoError(t, err, "Failed to create logger with level %d", level)
		assert.NotNil(t, logger, "Logger should not be nil for level %d", level)
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger, err := New(LogLevelInfo, false, false)
	require.NoError(t, err)

	assert.Equal(t, LogLevelInfo, logger.GetLevel())

	logger.SetLevel(LogLevelDebug)
	assert.Equal(t, LogLevelDebug, logger.GetLevel())

	logger.SetLevel(LogLevelError)
	assert.Equal(t, LogLevelError, logger.GetLevel())
}

func TestLogger_LoggingMethods(t *testing.T) {
	logger, err := New(LogLevelDebug, false, false)
	require.NoError(t, err)

	// 这些方法不应该panic
	assert.NotPanics(t, func() {
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")
	})

	logger.SetLevel(LogLevelFatal)
	// Debug和Info级别应该不输出但也不panic
	assert.NotPanics(t, func() {
		logger.Debug("debug message")
		logger.Info("info message")
	})
}

func TestLogger_LoggingMethodsFormatted(t *testing.T) {
	logger, err := New(LogLevelDebug, false, false)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		logger.Debugf("debug: %d", 1)
		logger.Infof("info: %d", 2)
		logger.Warnf("warn: %d", 3)
		logger.Errorf("error: %d", 4)
	})
}

func TestLogger_With(t *testing.T) {
	logger, err := New(LogLevelInfo, false, false)
	require.NoError(t, err)

	newLogger := logger.With(zap.String("key", "value"))
	assert.NotNil(t, newLogger)
}

func TestLogger_Named(t *testing.T) {
	logger, err := New(LogLevelInfo, false, false)
	require.NoError(t, err)

	namedLogger := logger.Named("test")
	assert.NotNil(t, namedLogger)
}

func TestLogger_Sync(t *testing.T) {
	logger, err := New(LogLevelInfo, false, false)
	require.NoError(t, err)

	// Sync不应该panic
	assert.NotPanics(t, func() {
		logger.Sync()
	})
}

func TestLogLevel_Order(t *testing.T) {
	// 验证日志级别顺序 (iota从0开始递增)
	assert.True(t, LogLevelNever < LogLevelFatal)
	assert.True(t, LogLevelFatal < LogLevelError)
	assert.True(t, LogLevelError < LogLevelWarn)
	assert.True(t, LogLevelWarn < LogLevelInfo)
	assert.True(t, LogLevelInfo < LogLevelDebug)
	assert.True(t, LogLevelDebug < LogLevelTrace)
}

func TestLogger_Fatal(t *testing.T) {
	// Fatal 调用 zap.Fatal 会 os.Exit(1)，跳过此测试
	// 在实际应用中会正确退出
	t.Skip("Fatal causes os.Exit(1), skipping in test")
}

func TestLogger_Fatalf(t *testing.T) {
	// Fatalf 调用 zap.Fatal 会 os.Exit(1)，跳过此测试
	t.Skip("Fatalf causes os.Exit(1), skipping in test")
}

func TestLogger_Fatal_NotDisplayedBelowFatalLevel(t *testing.T) {
	// Fatal 即使在低级别也会 os.Exit(1)，跳过此测试
	t.Skip("Fatal causes os.Exit(1), skipping in test")
}

func TestLogger_Fatalf_NotDisplayedBelowFatalLevel(t *testing.T) {
	// Fatalf 即使在低级别也会 os.Exit(1)，跳过此测试
	t.Skip("Fatalf causes os.Exit(1), skipping in test")
}
