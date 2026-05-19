# tinyportmapper-go

高性能端口映射工具，Go 语言重写版本。支持 TCP/UDP 协议转发，性能可达 **6-8 Gbits/sec**。
跨平台支持 **Linux / macOS / Windows**。

## 特性

- 高性能数据转发（goroutine 并发 + 缓冲区池）
- TCP/UDP 双协议支持
- 连接超时自动清理（LRU 淘汰）
- 结构化日志（zap）
- 跨平台编译（Windows 原生支持）

## 安装

### 源码编译

```bash
git clone https://github.com/x1t/tinyportmapper-go.git
cd tinyportmapper-go

# Linux / macOS
make build
sudo make install

# Windows
go build -o tinyportmapper.exe ./cmd/tinyportmapper
```

### Docker

```bash
docker run -d --network host --name tinyportmapper \
  x1t/tinyportmapper -l 0.0.0.0:8080 -r 127.0.0.1:80 -t -u
```

## 使用

```bash
# TCP 转发
tinyportmapper -l :8080 -r 10.0.0.1:80 -t

# UDP 转发
tinyportmapper -l :8080 -r 10.0.0.1:53 -u

# 同时启用 TCP 和 UDP
tinyportmapper -l :8080 -r 10.0.0.1:53 -t -u

# Windows 示例：将本机 22222 端口流量转发到 SSH 服务
tinyportmapper.exe -l 0.0.0.0:22222 -r 127.0.0.1:22 -t
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-l, --listen` | 监听地址 (必填) | - |
| `-r, --remote` | 远程地址 (必填) | - |
| `-t` | 启用 TCP 转发 | false |
| `-u` | 启用 UDP 转发 | false |
| `--sock-buf` | Socket 缓冲区大小 (KB) | 1024 |
| `--tcp-timeout` | TCP 超时时间 | 6m |
| `--udp-timeout` | UDP 超时时间 | 3m |
| `--max-connections` | 最大连接数 | 20000 |
| `--log-level` | 日志级别 (0-6) | 4 |

## 目录结构

```
tinyportmapper-go/
├── cmd/
│   └── tinyportmapper/   # CLI 入口 (Cobra)
├── internal/
│   ├── config/           # 配置加载与校验
│   ├── event/            # 事件循环 (中央协调器)
│   ├── forward/          # TCP/UDP 转发器
│   ├── conn/             # 连接对抽象
│   ├── cleanup/          # 超时连接清理器
│   ├── lru/              # LRU 缓存 (C++ 原版算法)
│   ├── log/              # 结构化日志 (zap)
│   ├── types/            # 地址类型 (IPv4/IPv6)
│   ├── signal/           # 信号处理
│   └── fd/               # 文件描述符管理
└── test/
    └── integration/      # 集成测试
```

## 架构

### 核心组件

- **事件循环** (`internal/event/loop.go`): 协调 TCP 监听、UDP 接收和转发器生命周期
- **TCP 转发** (`internal/forward/tcp.go`): 每连接 2 个 goroutine 双向数据拷贝
- **UDP 转发** (`internal/forward/udp.go`): 每客户端独立 UDP socket + goroutine
- **连接清理** (`internal/cleanup/manager.go`): 定时按 LRU 淘汰超时连接

## 构建

```bash
# 构建二进制
go build -o tinyportmapper ./cmd/tinyportmapper

# 运行测试
go test ./...
go test -race ./...

# 代码检查
go vet ./...

# 或使用 Makefile（Linux/macOS）
make build
make test
make lint
```

## 参考

原项目: [tinyPortMapper](https://github.com/wangyu-/tinyPortMapper)

## 许可证

MIT License
