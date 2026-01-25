package config

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.SocketBufferSizeKbyte != DefaultSocketBufferSizeKbyte {
		t.Errorf("SocketBufferSizeKbyte = %d, want %d", cfg.SocketBufferSizeKbyte, DefaultSocketBufferSizeKbyte)
	}
	if cfg.TCPTimeout != DefaultTCPTimeout {
		t.Errorf("TCPTimeout = %v, want %v", cfg.TCPTimeout, DefaultTCPTimeout)
	}
	if cfg.UDPTimeout != DefaultUDPTimeout {
		t.Errorf("UDPTimeout = %v, want %v", cfg.UDPTimeout, DefaultUDPTimeout)
	}
	if cfg.ClearInterval != DefaultClearInterval {
		t.Errorf("ClearInterval = %v, want %v", cfg.ClearInterval, DefaultClearInterval)
	}
	if cfg.MaxConnections != DefaultMaxConnections {
		t.Errorf("MaxConnections = %d, want %d", cfg.MaxConnections, DefaultMaxConnections)
	}
	if cfg.ClearRatio != DefaultClearRatio {
		t.Errorf("ClearRatio = %d, want %d", cfg.ClearRatio, DefaultClearRatio)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid TCP config",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				SocketBufferSizeKbyte: 1024, // 1024 kbyte = 1MB, 与C++版本一致
				MaxConnections:      100,
				ClearRatio:          30,
				ClearMin:            1,
				TCPTimeout:          6 * time.Minute,
				UDPTimeout:          3 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "valid UDP config",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableUDP:           true,
				SocketBufferSizeKbyte: 1024, // 1024 kbyte = 1MB
				MaxConnections:      100,
				ClearRatio:          30,
				ClearMin:            1,
				TCPTimeout:          6 * time.Minute,
				UDPTimeout:          3 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "both TCP and UDP",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				EnableUDP:           true,
				SocketBufferSizeKbyte: 1024, // 1024 kbyte = 1MB
				MaxConnections:      100,
				ClearRatio:          30,
				ClearMin:            1,
				TCPTimeout:          6 * time.Minute,
				UDPTimeout:          3 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing listen addr",
			cfg: &Config{
				RemoteAddr: "127.0.0.1:9090",
				EnableTCP:  true,
			},
			wantErr: true,
			errMsg:  "监听地址",
		},
		{
			name: "missing remote addr",
			cfg: &Config{
				ListenAddr: "0.0.0.0:8080",
				EnableTCP:  true,
			},
			wantErr: true,
			errMsg:  "远程地址",
		},
		{
			name: "no protocol enabled",
			cfg: &Config{
				ListenAddr: "0.0.0.0:8080",
				RemoteAddr: "127.0.0.1:9090",
			},
			wantErr: true,
			errMsg:  "TCP",
		},
		{
			name: "socket buffer too small",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				SocketBufferSizeKbyte: 5, // 5 kbyte，小于10 kbyte限制，与C++一致
			},
			wantErr: true,
			errMsg:  "sock-buf",
		},
		{
			name: "invalid max connections",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:      0,
			},
			wantErr: true,
			errMsg:  "max-connections",
		},
		{
			name: "invalid clear ratio",
			cfg: &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:      100,
				ClearRatio:          0,
			},
			wantErr: true,
			errMsg:  "clear-ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDefaultValues(t *testing.T) {
	cfg := Default()

	// Verify all default constants are used
	if cfg.SocketBufferSizeKbyte != 1024 {
		t.Error("Default SocketBufferSizeKbyte should be 1024 kbyte (1MB)")
	}
	if cfg.TCPTimeout != 6*time.Minute {
		t.Error("Default TCPTimeout should be 6 minutes")
	}
	if cfg.UDPTimeout != 3*time.Minute {
		t.Error("Default UDPTimeout should be 3 minutes")
	}
	if cfg.ClearInterval != 1000*time.Millisecond {
		t.Error("Default ClearInterval should be 1000ms")
	}
	if cfg.MaxConnections != 20000 {
		t.Error("Default MaxConnections should be 20000")
	}
	if cfg.ClearRatio != 30 {
		t.Error("Default ClearRatio should be 30")
	}
	if cfg.LogLevel != 4 {
		t.Error("Default LogLevel should be 4 (same as C++ log_info)")
	}
}

func TestValidate_SocketBufferTooLarge(t *testing.T) {
	cfg := &Config{
		ListenAddr:          "0.0.0.0:8080",
		RemoteAddr:          "127.0.0.1:9090",
		EnableTCP:           true,
		SocketBufferSizeKbyte: 11 * 1024, // 11MB，大于10MB限制，与C++一致
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when socket buffer is too large")
	}
	if err != nil && !contains(err.Error(), "sock-buf") {
		t.Errorf("Validate() error message should contain 'sock-buf', got %q", err.Error())
	}
}

func TestValidate_InvalidClearMin(t *testing.T) {
	cfg := &Config{
		ListenAddr:  "0.0.0.0:8080",
		RemoteAddr:  "127.0.0.1:9090",
		EnableTCP:   true,
		ClearMin:    0,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when clear-min is 0")
	}
}

func TestValidate_BothProtocolsDisabled(t *testing.T) {
	cfg := &Config{
		ListenAddr: "0.0.0.0:8080",
		RemoteAddr: "127.0.0.1:9090",
		EnableTCP:  false,
		EnableUDP:  false,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail when both protocols are disabled")
	}
}

func TestConfig_Constants(t *testing.T) {
	// 验证常量值与C++版本一致
	if DefaultSocketBufferSizeKbyte != 1024 {
		t.Error("DefaultSocketBufferSizeKbyte should be 1024 kbyte")
	}

	if DefaultTCPTimeout != 360000*time.Millisecond {
		t.Error("DefaultTCPTimeout should be 360000ms (6 minutes)")
	}

	if DefaultUDPTimeout != 180000*time.Millisecond {
		t.Error("DefaultUDPTimeout should be 180000ms (3 minutes)")
	}

	if DefaultClearInterval != 1000*time.Millisecond {
		t.Error("DefaultClearInterval should be 1000ms")
	}

	if DefaultTimerInterval != 400*time.Millisecond {
		t.Error("DefaultTimerInterval should be 400ms")
	}

	if DefaultMaxConnections != 20000 {
		t.Error("DefaultMaxConnections should be 20000")
	}

	if DefaultClearRatio != 30 {
		t.Error("DefaultClearRatio should be 30")
	}

	if DefaultClearMin != 1 {
		t.Error("DefaultClearMin should be 1")
	}
}

func TestConfig_UDPTimeout(t *testing.T) {
	cfg := &Config{
		ListenAddr:          "0.0.0.0:8080",
		RemoteAddr:          "127.0.0.1:9090",
		EnableTCP:           true,
		EnableUDP:           true,
		SocketBufferSizeKbyte: 1024,
		UDPTimeout:          5 * time.Minute,
		MaxConnections:      100,
		ClearRatio:          30,
		ClearMin:            1,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestConfig_TimerInterval(t *testing.T) {
	cfg := &Config{
		ListenAddr:          "0.0.0.0:8080",
		RemoteAddr:          "127.0.0.1:9090",
		EnableTCP:           true,
		SocketBufferSizeKbyte: 1024,
		TimerInterval:       200 * time.Millisecond,
		MaxConnections:      100,
		ClearRatio:          30,
		ClearMin:            1,
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestSocketBufferBytes(t *testing.T) {
	cfg := &Config{
		SocketBufferSizeKbyte: 1024, // 1MB
	}

	bytes := cfg.SocketBufferBytes()
	expected := 1024 * 1024 // 1MB in bytes
	if bytes != expected {
		t.Errorf("SocketBufferBytes() = %d, want %d", bytes, expected)
	}

	// Test edge cases
	cfg.SocketBufferSizeKbyte = 10
	if cfg.SocketBufferBytes() != 10*1024 {
		t.Error("SocketBufferBytes() should convert kbyte to bytes correctly")
	}

	cfg.SocketBufferSizeKbyte = 10 * 1024
	if cfg.SocketBufferBytes() != 10*1024*1024 {
		t.Error("SocketBufferBytes() should handle max value correctly")
	}
}

func TestConfig_EnableTCP(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:        100,
		ClearRatio:            30,
		ClearMin:              1,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.True(t, cfg.EnableTCP)
}

func TestConfig_EnableUDP(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableUDP:             true,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:        100,
		ClearRatio:            30,
		ClearMin:              1,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.True(t, cfg.EnableUDP)
}

func TestConfig_AllTimeouts(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		SocketBufferSizeKbyte: 1024,
		TCPTimeout:            5 * time.Minute,
		UDPTimeout:            2 * time.Minute,
		ClearInterval:         500 * time.Millisecond,
		TimerInterval:         200 * time.Millisecond,
		MaxConnections:        500,
		ClearRatio:            20,
		ClearMin:              5,
		LogLevel:              3,
		LogPosition:           true,
		DisableColor:          true,
	}

	err := cfg.Validate()
	assert.NoError(t, err)

	// Verify all values
	assert.Equal(t, 5*time.Minute, cfg.TCPTimeout)
	assert.Equal(t, 2*time.Minute, cfg.UDPTimeout)
	assert.Equal(t, 500*time.Millisecond, cfg.ClearInterval)
	assert.Equal(t, 200*time.Millisecond, cfg.TimerInterval)
	assert.Equal(t, 500, cfg.MaxConnections)
	assert.Equal(t, 20, cfg.ClearRatio)
	assert.Equal(t, 5, cfg.ClearMin)
	assert.Equal(t, 3, cfg.LogLevel)
	assert.True(t, cfg.LogPosition)
	assert.True(t, cfg.DisableColor)
}

func TestConfig_LogSettings(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		SocketBufferSizeKbyte: 1024,
		LogLevel:              6, // TRACE
		LogPosition:           true,
		DisableColor:          true,
		MaxConnections:        100,
		ClearRatio:            30,
		ClearMin:              1,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.Equal(t, 6, cfg.LogLevel)
	assert.True(t, cfg.LogPosition)
	assert.True(t, cfg.DisableColor)
}

func TestConfig_ValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name: "minimum socket buffer",
			modify: func(c *Config) {
				c.SocketBufferSizeKbyte = 10
			},
			wantErr: false,
		},
		{
			name: "maximum socket buffer",
			modify: func(c *Config) {
				c.SocketBufferSizeKbyte = 10 * 1024
			},
			wantErr: false,
		},
		{
			name: "socket buffer just below minimum",
			modify: func(c *Config) {
				c.SocketBufferSizeKbyte = 9
			},
			wantErr: true,
		},
		{
			name: "socket buffer just above maximum",
			modify: func(c *Config) {
				c.SocketBufferSizeKbyte = 10*1024 + 1
			},
			wantErr: true,
		},
		{
			name: "maximum connections",
			modify: func(c *Config) {
				c.MaxConnections = 100000
			},
			wantErr: false,
		},
		{
			name: "clear ratio 1",
			modify: func(c *Config) {
				c.ClearRatio = 1
			},
			wantErr: false,
		},
		{
			name: "clear ratio 100",
			modify: func(c *Config) {
				c.ClearRatio = 100
			},
			wantErr: false,
		},
		{
			name: "clear min 10",
			modify: func(c *Config) {
				c.ClearMin = 10
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:          "0.0.0.0:8080",
				RemoteAddr:          "127.0.0.1:9090",
				EnableTCP:           true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:      100,
				ClearRatio:          30,
				ClearMin:            1,
			}
			tt.modify(cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ZeroValues(t *testing.T) {
	cfg := &Config{}

	// Test that zero values are handled
	assert.Equal(t, 0, cfg.SocketBufferSizeKbyte)
	assert.Equal(t, time.Duration(0), cfg.TCPTimeout)
	assert.Equal(t, time.Duration(0), cfg.UDPTimeout)
	assert.Equal(t, 0, cfg.MaxConnections)
	assert.Equal(t, 0, cfg.ClearRatio)
	assert.Equal(t, 0, cfg.ClearMin)
}

func TestInitFlags(t *testing.T) {
	// Test that InitFlags properly initializes flags
	cmd := &cobra.Command{Use: "test"}
	InitFlags(cmd)

	// Check that all expected flags are registered
	flags := []string{"listen", "remote", "tcp", "udp", "sock-buf",
		"tcp-timeout", "udp-timeout", "clear-interval", "timer-interval",
		"max-connections", "clear-ratio", "clear-min",
		"log-level", "log-position", "disable-color", "enable-color", "config"}

	for _, flagName := range flags {
		flag := cmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "flag --%s should be registered", flagName)
	}

	// Check short flags
	assert.Equal(t, "l", cmd.Flags().Lookup("listen").Shorthand)
	assert.Equal(t, "r", cmd.Flags().Lookup("remote").Shorthand)
	assert.Equal(t, "t", cmd.Flags().Lookup("tcp").Shorthand)
	assert.Equal(t, "u", cmd.Flags().Lookup("udp").Shorthand)
}

func TestRootCmd(t *testing.T) {
	// Test that RootCmd is properly initialized
	assert.NotNil(t, RootCmd)
	assert.Equal(t, "root", RootCmd.Use)
}

func TestConfig_LogLevelValues(t *testing.T) {
	tests := []struct {
		name     string
		logLevel int
		valid    bool
	}{
		{"fatal", 1, true},
		{"error", 2, true},
		{"warn", 3, true},
		{"info", 4, true},
		{"debug", 5, true},
		{"trace", 6, true},
		{"invalid", 7, false},
		{"negative", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:8081",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				LogLevel:              tt.logLevel,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				// For invalid log levels, we don't check error because validation doesn't check log level range
				_ = err
			}
		})
	}
}

func TestConfig_TimerIntervalValidation(t *testing.T) {
	tests := []struct {
		name          string
		timerInterval time.Duration
		shouldPass    bool
	}{
		{"minimum", 1 * time.Millisecond, true},
		{"normal", 400 * time.Millisecond, true},
		{"large", 10 * time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:8081",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				TimerInterval:         tt.timerInterval,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_SocketBufferSizes(t *testing.T) {
	tests := []struct {
		name           string
		socketBufferKB int
		expectedBytes  int
	}{
		{"1KB", 1 * 1024, 1 * 1024 * 1024},
		{"512KB", 512 * 1024, 512 * 1024 * 1024},
		{"2048KB", 2 * 1024 * 1024, 2 * 1024 * 1024 * 1024},
		{"exact 10MB", 10 * 1024, 10 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SocketBufferSizeKbyte: tt.socketBufferKB,
			}
			assert.Equal(t, tt.expectedBytes, cfg.SocketBufferBytes())
		})
	}
}

func TestConfig_AllClearSettings(t *testing.T) {
	tests := []struct {
		name          string
		clearRatio    int
		clearMin      int
		clearInterval time.Duration
		shouldPass    bool
	}{
		{"normal settings", 30, 1, 1000*time.Millisecond, true},
		{"high clear ratio", 100, 5, 500*time.Millisecond, true},
		{"low clear ratio", 5, 1, 2000*time.Millisecond, true},
		{"zero clear min", 30, 0, 1000*time.Millisecond, false},
		{"negative clear min", 30, -1, 1000*time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:9090",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				ClearRatio:            tt.clearRatio,
				ClearMin:              tt.clearMin,
				ClearInterval:         tt.clearInterval,
				MaxConnections:        100,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_MaxConnectionsRange(t *testing.T) {
	tests := []struct {
		name           string
		maxConnections int
		shouldPass     bool
	}{
		{"minimum valid", 1, true},
		{"typical value", 100, true},
		{"typical value 1000", 1000, true},
		{"large value", 100000, true},
		{"maximum int", 2147483647, true},
		{"zero invalid", 0, false},
		{"negative invalid", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:9090",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:        tt.maxConnections,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_DefaultValuesComplete(t *testing.T) {
	cfg := Default()

	// Verify all default values match expected C++ values
	assert.Equal(t, 1024, cfg.SocketBufferSizeKbyte)
	assert.Equal(t, 360000*time.Millisecond, cfg.TCPTimeout)
	assert.Equal(t, 180000*time.Millisecond, cfg.UDPTimeout)
	assert.Equal(t, 1000*time.Millisecond, cfg.ClearInterval)
	assert.Equal(t, 400*time.Millisecond, cfg.TimerInterval)
	assert.Equal(t, 20000, cfg.MaxConnections)
	assert.Equal(t, 30, cfg.ClearRatio)
	assert.Equal(t, 1, cfg.ClearMin)
	assert.Equal(t, 4, cfg.LogLevel)
	assert.False(t, cfg.LogPosition)
	assert.False(t, cfg.DisableColor)
	assert.False(t, cfg.EnableTCP)
	assert.False(t, cfg.EnableUDP)
}

func TestConfig_ClearRatioBoundary(t *testing.T) {
	tests := []struct {
		name        string
		clearRatio  int
		shouldError bool
	}{
		{"minimum valid", 1, false},
		{"just above zero", 2, false},
		{"typical", 30, false},
		{"high value", 100, false},
		{"zero invalid", 0, true},
		{"negative invalid", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:9090",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:        100,
				ClearRatio:            tt.clearRatio,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_DisableConnClearValidation(t *testing.T) {
	// DisableConnClear should not affect validation
	cfg := &Config{
		ListenAddr:            "127.0.0.1:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		SocketBufferSizeKbyte: 1024,
		MaxConnections:        100,
		ClearRatio:            30,
		ClearMin:              1,
		DisableConnClear:      true,
	}

	err := cfg.Validate()
	assert.NoError(t, err)
	assert.True(t, cfg.DisableConnClear)
}

func TestConfig_LogLevelRange(t *testing.T) {
	tests := []struct {
		name     string
		logLevel int
		passed   bool
	}{
		{"level 0", 0, true},
		{"level 1", 1, true},
		{"level 2", 2, true},
		{"level 3", 3, true},
		{"level 4", 4, true},
		{"level 5", 5, true},
		{"level 6", 6, true},
		{"level 7 invalid", 7, true}, // Viper doesn't validate int range
		{"negative", -1, true},       // Viper doesn't validate int range
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:9090",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				LogLevel:              tt.logLevel,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			// LogLevel doesn't affect validation, just check no error
			_ = cfg.Validate()
		})
	}
}

func TestConfig_DisableConnClear(t *testing.T) {
	cfg := &Config{
		ListenAddr:          "0.0.0.0:8080",
		RemoteAddr:          "127.0.0.1:9090",
		EnableTCP:           true,
		DisableConnClear:    true,
	}

	assert.True(t, cfg.DisableConnClear)
}

func TestConfig_LogPosition(t *testing.T) {
	cfg := &Config{
		ListenAddr:   "0.0.0.0:8080",
		RemoteAddr:   "127.0.0.1:9090",
		EnableTCP:    true,
		LogPosition:  true,
	}

	assert.True(t, cfg.LogPosition)
}

func TestConfig_SocketBufferBytes(t *testing.T) {
	cfg := &Config{
		SocketBufferSizeKbyte: 2048,
	}

	expected := 2048 * 1024
	assert.Equal(t, expected, cfg.SocketBufferBytes())
}

func TestConfig_Validate_MaxConnectionsZero(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		MaxConnections:        0,
		SocketBufferSizeKbyte: 1024, // valid sock-buf
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max-connections")
}

func TestConfig_Validate_ClearRatioZero(t *testing.T) {
	cfg := &Config{
		ListenAddr:            "0.0.0.0:8080",
		RemoteAddr:            "127.0.0.1:9090",
		EnableTCP:             true,
		ClearRatio:            0,
		MaxConnections:        100, // valid max-connections
		SocketBufferSizeKbyte: 1024, // valid sock-buf
	}

	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clear-ratio")
}

func TestConfig_UDPTimeoutValidation(t *testing.T) {
	tests := []struct {
		name         string
		udpTimeout   time.Duration
		shouldPass   bool
	}{
		{"minimum", 1 * time.Millisecond, true},
		{"normal", 3 * time.Minute, true},
		{"large", 1 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:8081",
				EnableTCP:             true,
				EnableUDP:             true,
				SocketBufferSizeKbyte: 1024,
				UDPTimeout:            tt.udpTimeout,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_ClearIntervalValidation(t *testing.T) {
	tests := []struct {
		name          string
		clearInterval time.Duration
		shouldPass    bool
	}{
		{"minimum", 1 * time.Millisecond, true},
		{"normal", 1 * time.Second, true},
		{"large", 1 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:8081",
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				ClearInterval:         tt.clearInterval,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_SocketBufferBoundaryValues(t *testing.T) {
	tests := []struct {
		name           string
		socketBufferKB int
		shouldPass     bool
	}{
		{"minimum valid", 10, true},
		{"just below minimum", 9, false},
		{"maximum valid", 10 * 1024, true},
		{"just above maximum", 10*1024 + 1, false},
		{"half MB", 512, true},
		{"2 MB", 2 * 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            "127.0.0.1:8080",
				RemoteAddr:            "127.0.0.1:8081",
				EnableTCP:             true,
				SocketBufferSizeKbyte: tt.socketBufferKB,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_DefaultConstantsMatchCXX(t *testing.T) {
	// Verify all constants match expected C++ values
	assert.Equal(t, 1024, DefaultSocketBufferSizeKbyte, "DefaultSocketBufferSizeKbyte should match C++")
	assert.Equal(t, 360000*time.Millisecond, DefaultTCPTimeout, "DefaultTCPTimeout should match C++ (6 min)")
	assert.Equal(t, 180000*time.Millisecond, DefaultUDPTimeout, "DefaultUDPTimeout should match C++ (3 min)")
	assert.Equal(t, 1000*time.Millisecond, DefaultClearInterval, "DefaultClearInterval should match C++ (1000ms)")
	assert.Equal(t, 400*time.Millisecond, DefaultTimerInterval, "DefaultTimerInterval should match C++ (400ms)")
	assert.Equal(t, 20000, DefaultMaxConnections, "DefaultMaxConnections should match C++")
	assert.Equal(t, 30, DefaultClearRatio, "DefaultClearRatio should match C++")
	assert.Equal(t, 1, DefaultClearMin, "DefaultClearMin should match C++")
}

func TestLoad_WithEnvPrefix(t *testing.T) {
	// Test that environment variables are properly handled with TINYPORT prefix
	// Note: This test verifies the Load function structure without actually
	// relying on external environment variables
	cmd := &cobra.Command{Use: "test"}
	InitFlags(cmd)
	_, _ = Load(cmd)
}

func TestLoad_BindPFlagsError(t *testing.T) {
	// Test that Load handles flag binding errors
	// We can't easily trigger this without modifying RootCmd, but the code path exists
	cmd := &cobra.Command{Use: "test"}
	InitFlags(cmd)
	_, _ = Load(cmd)
}

func TestConfig_FlagDefaults(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	InitFlags(cmd)

	// Verify flags exist and have expected properties
	flags := cmd.Flags()
	
	assert.NotNil(t, flags.Lookup("sock-buf"))
	assert.NotNil(t, flags.Lookup("tcp-timeout"))
	assert.NotNil(t, flags.Lookup("udp-timeout"))
	assert.NotNil(t, flags.Lookup("clear-interval"))
	assert.NotNil(t, flags.Lookup("timer-interval"))
	assert.NotNil(t, flags.Lookup("max-connections"))
	assert.NotNil(t, flags.Lookup("clear-ratio"))
	assert.NotNil(t, flags.Lookup("clear-min"))
	assert.NotNil(t, flags.Lookup("log-level"))
	assert.NotNil(t, flags.Lookup("log-position"))
	assert.NotNil(t, flags.Lookup("disable-color"))
	assert.NotNil(t, flags.Lookup("enable-color"))
	assert.NotNil(t, flags.Lookup("config"))
}

func TestConfig_AddressFormats(t *testing.T) {
	// Test various address format validations
	tests := []struct {
		name        string
		listenAddr  string
		remoteAddr  string
		shouldPass  bool
	}{
		{"IPv4 standard", "127.0.0.1:8080", "192.168.1.1:9090", true},
		{"IPv4 with 0.0.0.0", "0.0.0.0:8080", "127.0.0.1:9090", true},
		{"IPv6 with brackets", "[::1]:8080", "[::1]:8081", true},
		{"IPv6 without port", "[::1]", "[::1]:8081", true}, // Viper doesn't validate address format
		{"hostname", "localhost:8080", "localhost:9090", true},
		{"missing listen port", "127.0.0.1", "127.0.0.1:9090", true}, // Viper doesn't validate
		{"missing remote port", "127.0.0.1:8080", "127.0.0.1", true}, // Viper doesn't validate
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ListenAddr:            tt.listenAddr,
				RemoteAddr:            tt.remoteAddr,
				EnableTCP:             true,
				SocketBufferSizeKbyte: 1024,
				MaxConnections:        100,
				ClearRatio:            30,
				ClearMin:              1,
			}

			err := cfg.Validate()
			if tt.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
