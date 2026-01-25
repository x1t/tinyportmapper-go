package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// DefaultSocketBufferSizeKbyte 默认socket缓冲区大小 (kbyte)
	DefaultSocketBufferSizeKbyte = 1024 // 1024 kbyte = 1MB
	// DefaultTCPTimeout 默认TCP超时时间 (毫秒)
	DefaultTCPTimeout = 360000 * time.Millisecond
	// DefaultUDPTimeout 默认UDP超时时间 (毫秒)
	DefaultUDPTimeout = 180000 * time.Millisecond
	// DefaultClearInterval 默认清理间隔 (毫秒)
	DefaultClearInterval = 1000 * time.Millisecond
	// DefaultTimerInterval 默认定时器间隔 (毫秒)
	DefaultTimerInterval = 400 * time.Millisecond
	// DefaultMaxConnections 默认最大连接数
	DefaultMaxConnections = 20000
	// DefaultClearRatio 默认清理比例
	DefaultClearRatio = 30
	// DefaultClearMin 每次最小清理数
	DefaultClearMin = 1
	// DefaultLogLevel 默认日志级别 (与 C++ 版本 log_info=4 一致)
	DefaultLogLevel = 4
)

// Config 配置结构
type Config struct {
	// 基础配置 (与C++版本一致)
	ListenAddr string `mapstructure:"listen"`
	RemoteAddr string `mapstructure:"remote"`
	EnableTCP  bool   `mapstructure:"enable-tcp"`
	EnableUDP  bool   `mapstructure:"enable-udp"`

	// 高级配置
	SocketBufferSizeKbyte int           `mapstructure:"socket-buffer-size"` // 用户输入单位: kbyte (与C++一致)
	TCPTimeout            time.Duration `mapstructure:"tcp-timeout"`
	UDPTimeout            time.Duration `mapstructure:"udp-timeout"`
	ClearInterval         time.Duration `mapstructure:"clear-interval"`
	TimerInterval         time.Duration `mapstructure:"timer-interval"`
	MaxConnections        int           `mapstructure:"max-connections"`
	ClearRatio            int           `mapstructure:"clear-ratio"`
	ClearMin              int           `mapstructure:"clear-min"`

	// 日志配置
	LogLevel     int  `mapstructure:"log-level"`
	LogPosition  bool `mapstructure:"log-position"`
	DisableColor bool `mapstructure:"disable-color"`

	// 高级配置 (与 C++ 版本一致)
	DisableConnClear bool `mapstructure:"disable-conn-clear"` // 是否禁用连接清理
}

// SocketBufferBytes 返回字节单位的缓冲区大小 (用于 SetReadBuffer/SetWriteBuffer)
func (c *Config) SocketBufferBytes() int {
	return c.SocketBufferSizeKbyte * 1024
}

// Default 返回默认配置 (与C++版本一致)
func Default() *Config {
	return &Config{
		EnableTCP:             false,
		EnableUDP:             false,
		SocketBufferSizeKbyte: DefaultSocketBufferSizeKbyte,
		TCPTimeout:            DefaultTCPTimeout,
		UDPTimeout:            DefaultUDPTimeout,
		ClearInterval:         DefaultClearInterval,
		TimerInterval:         DefaultTimerInterval,
		MaxConnections:        DefaultMaxConnections,
		ClearRatio:            DefaultClearRatio,
		ClearMin:              DefaultClearMin,
		LogLevel:              DefaultLogLevel, // 与 C++ 版本一致，默认 log_info=4
		LogPosition:           false,
		DisableColor:          false,
	}
}

// Validate 验证配置 (与C++版本一致)
func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("必须指定监听地址 (-l/--listen)")
	}
	if c.RemoteAddr == "" {
		return fmt.Errorf("必须指定远程地址 (-r/--remote)")
	}
	if !c.EnableTCP && !c.EnableUDP {
		return fmt.Errorf("必须指定 -t (TCP) 或 -u (UDP) 或两者")
	}

	// sock-buf 范围: 10KB - 10MB (kbyte单位，与C++版本一致)
	if c.SocketBufferSizeKbyte < 10 || c.SocketBufferSizeKbyte > 10*1024 {
		return fmt.Errorf("sock-buf 大小必须在 10-10240 kbyte 之间")
	}

	if c.MaxConnections <= 0 {
		return fmt.Errorf("max-connections 必须大于 0")
	}

	if c.ClearRatio <= 0 {
		return fmt.Errorf("clear-ratio 必须大于 0")
	}

	if c.ClearMin <= 0 {
		return fmt.Errorf("clear-min 必须大于 0")
	}

	return nil
}

// InitFlags 初始化命令行参数 (与C++版本一致)
func InitFlags(cmd *cobra.Command) {
	defaultCfg := Default()

	// 基础参数 (短参数与C++一致)
	cmd.Flags().StringP("listen", "l", "", "监听地址, 格式: ip:port 或 [ip]:port")
	cmd.Flags().StringP("remote", "r", "", "远程地址, 格式: ip:port 或 [ip]:port")
	cmd.Flags().BoolP("tcp", "t", false, "启用 TCP 转发")
	cmd.Flags().BoolP("udp", "u", false, "启用 UDP 转发")

	// 帮助参数 (与C++版本一致，C++使用 -h)
	cmd.Flags().BoolP("help", "h", false, "显示帮助信息")

	// 高级参数 (与C++版本一致，单位为kbyte)
	cmd.Flags().Int("sock-buf", defaultCfg.SocketBufferSizeKbyte,
		"Socket 缓冲区大小 (kbyte), 范围 10-10240, 与C++版本一致")
	cmd.Flags().Duration("tcp-timeout", defaultCfg.TCPTimeout,
		"TCP 连接超时时间 (支持 Go Duration 格式，如 300s, 5m, 1h)")
	cmd.Flags().Duration("udp-timeout", defaultCfg.UDPTimeout,
		"UDP 连接超时时间 (支持 Go Duration 格式，如 120s, 2m, 30m)")
	cmd.Flags().Duration("clear-interval", defaultCfg.ClearInterval,
		"清理超时连接的间隔 (毫秒)")
	cmd.Flags().Duration("timer-interval", defaultCfg.TimerInterval,
		"定时器间隔 (毫秒)")
	cmd.Flags().Int("max-connections", defaultCfg.MaxConnections,
		"最大连接数")
	cmd.Flags().Int("clear-ratio", defaultCfg.ClearRatio,
		"每次清理连接的比例")
	cmd.Flags().Int("clear-min", defaultCfg.ClearMin,
		"每次最小清理数")

	// 日志参数 (与C++版本一致)
	cmd.Flags().Int("log-level", defaultCfg.LogLevel,
		"日志级别: 0=never, 1=fatal, 2=error, 3=warn, 4=info, 5=debug, 6=trace")
	cmd.Flags().Bool("log-position", defaultCfg.LogPosition,
		"在日志中显示文件名和行号")
	cmd.Flags().Bool("disable-color", defaultCfg.DisableColor,
		"禁用日志颜色")
	cmd.Flags().Bool("enable-color", !defaultCfg.DisableColor,
		"启用日志颜色")

	// 配置文件支持
	cmd.Flags().String("config", "", "配置文件路径")

	// 连接清理控制 (与 C++ 版本一致)
	cmd.Flags().Bool("disable-conn-clear", false, "禁用连接超时自动清理")
}

// Load 加载配置
// cmd: 必须传入已初始化标志的 cobra.Command 指针
func Load(cmd *cobra.Command) (*Config, error) {
	v := viper.New()

	// 设置默认值
	cfg := Default()
	v.SetDefault("enable-tcp", cfg.EnableTCP)
	v.SetDefault("enable-udp", cfg.EnableUDP)
	v.SetDefault("socket-buffer-size", cfg.SocketBufferSizeKbyte)
	v.SetDefault("tcp-timeout", cfg.TCPTimeout)
	v.SetDefault("udp-timeout", cfg.UDPTimeout)
	v.SetDefault("clear-interval", cfg.ClearInterval)
	v.SetDefault("max-connections", cfg.MaxConnections)
	v.SetDefault("clear-ratio", cfg.ClearRatio)
	v.SetDefault("log-level", cfg.LogLevel)
	v.SetDefault("log-position", cfg.LogPosition)
	v.SetDefault("disable-color", cfg.DisableColor)

	// 直接从 cobra 命令读取所有标志值 (避免 viper BindPFlags 的布尔标志问题)
	flags := cmd.Flags()

	// 基础配置
	if flags.Changed("listen") {
		cfg.ListenAddr, _ = flags.GetString("listen")
	}
	if flags.Changed("remote") {
		cfg.RemoteAddr, _ = flags.GetString("remote")
	}
	cfg.EnableTCP, _ = flags.GetBool("tcp")
	cfg.EnableUDP, _ = flags.GetBool("udp")

	// 高级配置
	cfg.SocketBufferSizeKbyte, _ = flags.GetInt("sock-buf")
	cfg.TCPTimeout, _ = flags.GetDuration("tcp-timeout")
	cfg.UDPTimeout, _ = flags.GetDuration("udp-timeout")
	cfg.ClearInterval, _ = flags.GetDuration("clear-interval")
	cfg.TimerInterval, _ = flags.GetDuration("timer-interval")
	cfg.MaxConnections, _ = flags.GetInt("max-connections")
	cfg.ClearRatio, _ = flags.GetInt("clear-ratio")
	cfg.ClearMin, _ = flags.GetInt("clear-min")

	// 日志配置
	cfg.LogLevel, _ = flags.GetInt("log-level")
	logPos, _ := flags.GetBool("log-position")
	cfg.LogPosition = logPos
	cfg.DisableColor, _ = flags.GetBool("disable-color")

	// 连接清理控制
	cfg.DisableConnClear, _ = flags.GetBool("disable-conn-clear")

	// 如果指定了配置文件，覆盖命令行参数
	configFile, _ := flags.GetString("config")
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 使用配置文件的值覆盖
		if err := v.Unmarshal(cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件失败: %w", err)
		}
	}

	// 环境变量支持 (仅用于未在命令行指定的配置项)
	v.SetEnvPrefix("TINYPORT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// RootCmd 用于绑定的命令
var RootCmd = &cobra.Command{Use: "root"}
