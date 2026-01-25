package log

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLevel 日志级别
type LogLevel int

const (
	// LogLevelNever 从不输出
	LogLevelNever LogLevel = iota
	// LogLevelFatal 致命错误
	LogLevelFatal
	// LogLevelError 错误
	LogLevelError
	// LogLevelWarn 警告
	LogLevelWarn
	// LogLevelInfo 信息 (默认)
	LogLevelInfo
	// LogLevelDebug 调试
	LogLevelDebug
	// LogLevelTrace 跟踪
	LogLevelTrace
)

// zapLevel 转换日志级别到 zapcore.Level
var zapLevelMap = map[LogLevel]zapcore.Level{
	LogLevelNever: zapcore.DebugLevel,
	LogLevelFatal: zapcore.FatalLevel,
	LogLevelError: zapcore.ErrorLevel,
	LogLevelWarn:  zapcore.WarnLevel,
	LogLevelInfo:  zapcore.InfoLevel,
	LogLevelDebug: zapcore.DebugLevel,
	LogLevelTrace: zapcore.DebugLevel,
}

// Logger 日志器封装
type Logger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
	mu     sync.RWMutex
	level  LogLevel
}

// ZapField 用于向后兼容
type ZapField = zap.Field

// New 创建新的日志器
func New(level LogLevel, showPosition bool, enableColor bool) (*Logger, error) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel(enableColor),
		EncodeTime:     encodeTime(),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   encodeCaller(showPosition),
	}

	zapLevel := zapcore.InfoLevel
	if l, ok := zapLevelMap[level]; ok {
		zapLevel = l
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.Lock(os.Stdout),
		zapLevel,
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.FatalLevel))

	return &Logger{
		logger: logger,
		sugar:  logger.Sugar(),
		level:  level,
	}, nil
}

// encodeLevel 编码日志级别
func encodeLevel(enableColor bool) zapcore.LevelEncoder {
	return func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		var colorCode string
		switch l {
		case zapcore.FatalLevel:
			colorCode = "31" // 红色
		case zapcore.ErrorLevel:
			colorCode = "31" // 红色
		case zapcore.WarnLevel:
			colorCode = "33" // 黄色
		case zapcore.InfoLevel:
			colorCode = "32" // 绿色
		case zapcore.DebugLevel:
			colorCode = "36" // 青色
		default:
			colorCode = "37" // 白色
		}

		if enableColor {
			enc.AppendString(fmt.Sprintf("\033[%sm%s\033[0m", colorCode, l.String()))
		} else {
			enc.AppendString(l.String())
		}
	}
}

// encodeTime 编码时间
func encodeTime() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("15:04:05.000"))
	}
}

// encodeCaller 编码调用位置
func encodeCaller(showPosition bool) zapcore.CallerEncoder {
	if showPosition {
		return zapcore.FullCallerEncoder
	}
	// 返回 nil 表示禁用 caller 编码
	return nil
}

// Sync 同步日志
func (l *Logger) Sync() {
	_ = l.logger.Sync()
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Debug 调试日志
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	if l.shouldLog(LogLevelDebug) {
		l.logger.Debug(msg, fields...)
	}
}

// Info 信息日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	if l.shouldLog(LogLevelInfo) {
		l.logger.Info(msg, fields...)
	}
}

// Warn 警告日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	if l.shouldLog(LogLevelWarn) {
		l.logger.Warn(msg, fields...)
	}
}

// Error 错误日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	if l.shouldLog(LogLevelError) {
		l.logger.Error(msg, fields...)
	}
}

// Fatal 致命日志
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

// Debugf 格式化调试日志
func (l *Logger) Debugf(template string, args ...interface{}) {
	if l.shouldLog(LogLevelDebug) {
		l.sugar.Debugf(template, args...)
	}
}

// Infof 格式化信息日志
func (l *Logger) Infof(template string, args ...interface{}) {
	if l.shouldLog(LogLevelInfo) {
		l.sugar.Infof(template, args...)
	}
}

// Warnf 格式化警告日志
func (l *Logger) Warnf(template string, args ...interface{}) {
	if l.shouldLog(LogLevelWarn) {
		l.sugar.Warnf(template, args...)
	}
}

// Errorf 格式化错误日志
func (l *Logger) Errorf(template string, args ...interface{}) {
	if l.shouldLog(LogLevelError) {
		l.sugar.Errorf(template, args...)
	}
}

// Fatalf 格式化致命日志
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// shouldLog 检查是否应该记录该级别的日志
func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level <= l.level
}

// With 创建带有字段的子日志器
func (l *Logger) With(fields ...zap.Field) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 转换 zap.Field 为 interface{} 用于 sugar.With
	args := make([]interface{}, len(fields))
	for i, f := range fields {
		args[i] = f
	}

	newLogger := &Logger{
		logger: l.logger.With(fields...),
		sugar:  l.sugar.With(args...),
		level:  l.level,
	}
	return newLogger
}

// Named 添加命名空间
func (l *Logger) Named(name string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		logger: l.logger.Named(name),
		sugar:  l.sugar.Named(name),
		level:  l.level,
	}
	return newLogger
}
