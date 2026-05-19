# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tinyportmapper-go** 是 tinyPortMapper 的 Go 语言重写版本，一个高性能端口映射工具，支持 TCP/UDP 协议转发，性能可达 6-8 Gbits/sec。支持 Linux/macOS/Windows 跨平台编译。

## Build & Test

```bash
# 构建
go build -o tinyportmapper ./cmd/tinyportmapper   # 构建二进制
go build ./...                                      # 检查全部编译

# 测试
go test ./...                                       # 运行所有测试
go test -v -run TestLoop_Run_OnlyTCP ./internal/event/  # 单一测试
go test -race ./...                                 # 竞态检测

# 代码质量
go vet ./...                                        # 静态分析
gofmt -w .                                         # 格式化
golangci-lint run ./...                             # lint（需安装）
```

## Architecture

### 数据流

```
Client → [TCPListener/UDPConn] → Loop → Forwarder → Remote Server
         ↑                           ↓                           ↑
         └──── TCPForwarder ◄────────┘        Remote → Forwarder ─┘
               (双向 goroutine copy)

Client → [UDPConn] → handleUDP → ForwardToRemote → Remote Server
         ↑                                              ↓
         └──── forwardFromRemote (per-client goroutine) ─┘
               (每个客户端独立 UDP socket)
```

### 核心模块

| 模块 | 文件 | 职责 |
|------|------|------|
| **Loop** | `internal/event/loop.go` | 中央协调器，管理 TCP 监听/UDP 接收/转发器/清理器生命周期 |
| **TCPForwarder** | `internal/forward/tcp.go` | 每连接两个 goroutine（src→dst + dst→src）做双向数据拷贝 |
| **UDPForwarder** | `internal/forward/udp.go` | 每个客户端一个 goroutine 从 remote 读回写 local |
| **TCPConn** | `internal/conn/tcp.go` | 本地↔远程 TCP 连接对 + 缓冲区池 (`sync.Pool`) |
| **UDPConn** | `internal/conn/udp.go` | UDP 连接对 + `UDPManager` 地址→连接映射 |
| **LRU Cache** | `internal/lru/cache.go` | 双向链表 + map，按活跃时间淘汰（C++ 原版算法） |
| **Cleanup** | `internal/cleanup/manager.go` | 定时 ticker 清理超时连接 |
| **Config** | `internal/config/config.go` | Cobra flags + viper 配置加载 |
| **Signal** | `internal/signal/handler.go` | SIGINT/SIGTERM 优雅退出 |
| **Types** | `internal/types/address.go` | 统一地址抽象（IPv4/IPv6） |

### 关键设计决策

- **TCP 转发**：每连接 2 个 goroutine 各自独立 copy，一方关闭写端时对面自动传播关闭（`CloseRead` / `CloseWrite`）
- **UDP 转发**：每个客户端 IP:port 一个独立的 `*net.UDPConn` 连到 remote，通过 `WriteTo` 回写 local 监听 socket
- **连接清理**：`cleanup.Manager` 内部两个 `lru.Cache`（TCP/UDP 分离），定时按 `clearRatio` 比例淘汰最久未活跃的连接
- **跨平台**：使用 Go 标准库 `net`，fd 操作限于 `(*net.UDPConn).File().Fd()`（Windows 返回 SOCKET handle）；`syscall` 常量层使用本地常量或 build tags

### 配置常量

| 常量 | 默认值 | 说明 |
|------|--------|------|
| `DefaultSocketBufferSizeKbyte` | 1024 | 缓冲区 1MB |
| `DefaultTCPTimeout` | 6min | TCP 超时 |
| `DefaultUDPTimeout` | 3min | UDP 超时 |
| `DefaultMaxConnections` | 20000 | 最大连接数 |
| `DefaultClearRatio` | 30 | 每次清理比例 1/30 |

## Coding Conventions

- 单文件控制在 200-500 行
- 并发安全：`sync.RWMutex` + `sync/atomic` 组合使用
- 日志：`zap.Field` 结构化字段，不用 fmt 打印
- 错误处理：显式 `if err != nil`，用 `fmt.Errorf("上下文: %w", err)` 包装
- 不写 `// TODO` 注释——必须立刻完成
