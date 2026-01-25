# tinyportmapper-go

高性能端口映射工具，Go 语言重写版本。支持 TCP/UDP 协议转发，性能可达 **6-8 Gbits/sec**。

## 特性

- 高性能数据转发
- TCP/UDP 协议支持
- 灵活的缓冲区配置
- 连接超时管理
- 结构化日志输出
- 静态链接支持

## 安装

### 方式一：二进制下载

从 [Releases](https://github.com/yourusername/tinyportmapper-go/releases) 下载预编译二进制文件。

### 方式二：源码编译

```bash
git clone https://github.com/x1t/tinyportmapper-go.git
cd tinyportmapper-go
make build
sudo make install
```

### 方式三：Docker

```bash
docker run -d --network host --name tinyportmapper \
  x1t/tinyportmapper -l 0.0.0.0:8080 -r 127.0.0.1:80 -t -u
```

## 使用

### 基本用法

```bash
tinyportmapper -l 0.0.0.0:8080 -r 127.0.0.1:80 -t -u
```

### 参数说明

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-l, --listen` | 监听地址 (必填) | - |
| `-r, --remote` | 远程地址 (必填) | - |
| `-t` | 启用 TCP 转发 | false |
| `-u` | 启用 UDP 转发 | false |
| `--sock-buf` | Socket 缓冲区大小 (KB) | 1024 |
| `--tcp-timeout` | TCP 超时时间 (ms) | 360000 |
| `--udp-timeout` | UDP 超时时间 (ms) | 180000 |
| `--max-connections` | 最大连接数 | 20000 |
| `--log-level` | 日志级别 (0-6) | 2 |

### 示例

```bash
# TCP 转发
tinyportmapper -l :8080 -r 10.0.0.1:80 -t

# UDP 转发
tinyportmapper -l :8080 -r 10.0.0.1:53 -u

# 同时启用 TCP 和 UDP
tinyportmapper -l :8080 -r 10.0.0.1:53 -t -u

# 自定义缓冲区
tinyportmapper -l :8080 -r 10.0.0.1:80 -t --sock-buf 2048
```

## 构建

```bash
make build          # 构建二进制文件
make build-static   # 构建静态链接二进制文件
make test           # 运行测试
make test-race      # 竞态检测测试
make lint           # 代码检查
make lint-fix       # 自动修复代码问题
```

## 目录结构

```
tinyportmapper-go/
├── cmd/
│   └── tinyportmapper/   # CLI 入口
├── internal/
│   ├── config/           # 配置管理
│   ├── event/            # 事件循环
│   ├── forward/          # TCP/UDP 转发器
│   ├── conn/             # 连接对抽象
│   ├── cleanup/          # 连接清理
│   ├── lru/              # LRU 缓存
│   ├── log/              # 日志
│   ├── types/            # 类型定义
│   ├── signal/           # 信号处理
│   └── fd/               # 文件描述符
├── pkg/
│   └── utils/            # 工具函数
├── test/
│   ├── integration/      # 集成测试
│   └── load/             # 负载测试
├── Makefile
├── .golangci.yml
└── go.mod
```

## 架构

### 核心组件

- **事件循环** (`internal/event/loop.go`): 协调 TCP/UDP 监听和转发
- **连接对** (`internal/conn/`): 管理本地↔远程连接对
- **转发器** (`internal/forward/`): 实现数据双向转发
- **LRU 缓存** (`internal/lru/`): 连接生命周期管理
- **清理器** (`internal/cleanup/`): 超时连接清理

### 性能优化

- 零拷贝数据传输
- goroutine 并发处理
- 缓冲区池化管理
- 高效超时管理

## 贡献

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -m 'feat: xxx'`)
4. 推送分支 (`git push origin feature/xxx`)
5. 提交 Pull Request

## 许可证

MIT License

## 参考

原项目: [tinyPortMapper](https://github.com/wangyu-/tinyPortMapper)
