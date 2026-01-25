# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**tinyportmapper-go** 是 tinyPortMapper 的 Go 语言重写版本，一个高性能端口映射工具，支持 TCP/UDP 协议转发，性能可达 6-8 Gbits/sec。

## Build Commands

```bash
make build              # 构建二进制文件
make build-static       # 构建静态链接二进制文件
make run                # 构建并运行示例
make install            # 安装到 /usr/local/bin
```

## Test Commands

```bash
make test               # 运行所有单元测试
make test-race          # 运行竞态检测测试
make test-coverage      # 生成覆盖率报告
make bench              # 运行性能测试
make check              # 代码检查 + 测试
```

## Linting

```bash
make lint               # 运行代码检查
make lint-fix           # 自动修复代码问题
make security           # 安全检查
make fmt                # 格式化代码
```

## Architecture

### Core Components

- **事件循环驱动** (`internal/event/loop.go`): `Loop` 结构是核心，协调 TCP/UDP 监听和转发
- **连接对抽象** (`internal/conn/`): `TCPConn`、`UDPConn` 管理本地↔远程连接对
- **转发器模式** (`internal/forward/`): `TCPForwarder` 和 `UDPForwarder` 实现数据转发
- **LRU 缓存** (`internal/lru/`): 管理连接生命周期和超时淘汰
- **连接清理器** (`internal/cleanup/`): 定时清理超时连接

### CLI Entry Point

`cmd/tinyportmapper/main.go` 使用 Cobra 框架，支持 `-l` 监听地址和 `-r` 远程地址参数。

## Key Configuration Constants

| 常量 | 默认值 | 说明 |
|------|--------|------|
| `DefaultSocketBufferSizeKbyte` | 1024 | 默认缓冲区 1MB |
| `DefaultTCPTimeout` | 360000ms | TCP 超时 6 分钟 |
| `DefaultUDPTimeout` | 180000ms | UDP 超时 3 分钟 |
| `DefaultMaxConnections` | 20000 | 最大连接数 |

## Coding Conventions

- Go 文件控制在 **200-500 行**之间
- 使用 `sync.RWMutex` 和 `sync/atomic` 实现并发安全
- 使用 `zap.Field` 进行结构化日志
