package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/x1t/tinyportmapper-go/internal/config"
	"github.com/x1t/tinyportmapper-go/internal/event"
	"github.com/x1t/tinyportmapper-go/internal/log"
	"go.uber.org/zap"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tinyportmapper",
		Short: "高性能端口映射工具",
		Long:  `tinyportmapper-go 是 tinyPortMapper 的 Go 语言重写版本，支持 TCP/UDP 端口转发。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cmd)
			if err != nil {
				return err
			}

			logger, err := log.New(log.LogLevel(cfg.LogLevel), cfg.LogPosition, !cfg.DisableColor)
			if err != nil {
				return err
			}

			loop, err := event.NewLoop(logger, cfg)
			if err != nil {
				return err
			}

			// 信号处理：监听 SIGINT / SIGTERM（Windows 和 Unix 都支持）
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			// 启动转发事件循环（非阻塞）
			go func() {
				if err := loop.Run(); err != nil {
					logger.Error("事件循环异常退出", zap.Error(err))
				}
			}()

			logger.Info("tinyportmapper 已启动",
				zap.String("version", version),
				zap.String("listen", cfg.ListenAddr),
				zap.String("remote", cfg.RemoteAddr),
				zap.Bool("tcp", cfg.EnableTCP),
				zap.Bool("udp", cfg.EnableUDP),
			)

			// 等待退出信号
			sig := <-sigCh
			logger.Info("收到退出信号，正在关闭...", zap.String("signal", sig.String()))

			loop.Stop()
			loop.Wait()
			logger.Info("tinyportmapper 已停止")
			return nil
		},
	}

	config.InitFlags(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
