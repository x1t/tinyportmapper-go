#!/bin/bash
# tinyPortMapper Go版本 iperf3 TCP 性能测试脚本
# 用法: ./iperf3_tcp_test.sh [--debug]

set -e

# --- 配置参数 ---
TINYPORTMAPPER="/root/tinyportmapper/tinyportmapper-go/tinyportmapper-go"
LISTEN_PORT=3322
SERVER_PORT=5201
TEST_DURATION=5
PARALLEL_STREAMS=4
LOG_DIR="/tmp/iperf3_test"
DEBUG_MODE=false

# --- 颜色定义 ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# --- 工具函数 ---
echo_color() {
    echo -e "${1}${2}${NC}"
}

print_header() {
    echo ""
    echo_color $CYAN "╔════════════════════════════════════════════════════════════════╗"
    printf "${CYAN}║%*s%s%*s║${NC}\n" $(((64-${#1})/2)) "" "$1" $(((64-${#1}+1)/2)) ""
    echo_color $CYAN "╚════════════════════════════════════════════════════════════════╝"
    echo ""
}

# 解析命令行参数
for arg in "$@"; do
    case $arg in
        --debug)
            DEBUG_MODE=true
            ;;
        --help|-h)
            echo "用法: $0 [--debug]"
            echo ""
            echo "选项:"
            echo "  --debug    启用详细实时输出模式"
            echo "  --help     显示此帮助信息"
            echo ""
            exit 0
            ;;
    esac
done

# --- 清理逻辑 ---
cleanup() {
    if [ "$DEBUG_MODE" = false ]; then
        echo ""
        echo_color $BLUE "🧹 正在清理进程..."
    fi
    pkill -9 -f "tinyportmapper-go.*-l.*$LISTEN_PORT" 2>/dev/null || true
    pkill -9 -f "iperf3.*-s.*-p.*$SERVER_PORT" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# --- 初始准备 ---
mkdir -p "$LOG_DIR"
if [ "$DEBUG_MODE" = false ]; then
    print_header "tinyPortMapper Go TCP 专项性能测试"
fi

# 预清理
cleanup

# ==================== 测试开始 ====================

# [1/4] 启动服务器
echo_color $BLUE "🚀 [1/4] 启动 iperf3 TCP 服务器 (端口 $SERVER_PORT)..."
if [ "$DEBUG_MODE" = true ]; then
    iperf3 -s -p $SERVER_PORT &
else
    iperf3 -s -p $SERVER_PORT > /dev/null 2>&1 &
fi
SERVER_PID=$!
sleep 3

# 检查服务器是否启动 (使用 PID 检查)
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo_color $RED "❌ 错误: iperf3 服务器启动失败 (PID: $SERVER_PID)!"
    exit 1
fi
echo_color $GREEN "  ✓ iperf3 服务器已就绪"

# [2/4] 基准测试
echo_color $BLUE "🚀 [2/4] 开始基准测试 (直接连接)..."
if [ "$DEBUG_MODE" = true ]; then
    iperf3 -c 127.0.0.1 -p $SERVER_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS 2>&1 | tee "$LOG_DIR/direct_output.txt"
    # 同时生成 JSON 供后续分析
    iperf3 -c 127.0.0.1 -p $SERVER_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > "$LOG_DIR/direct_tcp.json" 2>/dev/null || true
else
    iperf3 -c 127.0.0.1 -p $SERVER_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > "$LOG_DIR/direct_tcp.json"
fi
echo_color $GREEN "  ✓ 基准测试完成"

# 为了下一轮，重启服务器（iperf3 默认接受一次连接后可能需要重置，特别是某些版本）
kill $SERVER_PID 2>/dev/null || true
sleep 1
iperf3 -s -p $SERVER_PORT > /dev/null 2>&1 &
SERVER_PID=$!
sleep 1

# [3/4] 启动 tinyPortMapper
echo_color $BLUE "🚀 [3/4] 启动 tinyPortMapper (转发: $LISTEN_PORT -> $SERVER_PORT)..."
if [ "$DEBUG_MODE" = true ]; then
    $TINYPORTMAPPER -l 127.0.0.1:$LISTEN_PORT -r 127.0.0.1:$SERVER_PORT -t &
else
    $TINYPORTMAPPER -l 127.0.0.1:$LISTEN_PORT -r 127.0.0.1:$SERVER_PORT -t > /dev/null 2>&1 &
fi
TPM_PID=$!
sleep 2

if ! kill -0 $TPM_PID 2>/dev/null; then
    echo_color $RED "❌ 错误: tinyPortMapper 启动失败!"
    exit 1
fi
echo_color $GREEN "  ✓ tinyPortMapper 已就绪 (PID: $TPM_PID)"

# [4/4] 转发测试
echo_color $BLUE "🚀 [4/4] 开始转发性能测试..."
if [ "$DEBUG_MODE" = true ]; then
    iperf3 -c 127.0.0.1 -p $LISTEN_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS 2>&1 | tee "$LOG_DIR/forward_output.txt"
    iperf3 -c 127.0.0.1 -p $LISTEN_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > "$LOG_DIR/forward_tcp.json" 2>/dev/null || true
else
    iperf3 -c 127.0.0.1 -p $LISTEN_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > "$LOG_DIR/forward_tcp.json"
fi
echo_color $GREEN "  ✓ 转发测试完成"

# ==================== 结果分析 ====================
if [ "$DEBUG_MODE" = false ]; then
    print_header "TCP 测试结果汇总"
    python3 << 'PYEOF'
import json
import os

LOG_DIR = "/tmp/iperf3_test"

def load_json(filename):
    path = os.path.join(LOG_DIR, filename)
    if not os.path.exists(path): return None, "未找到日志"
    try:
        with open(path, 'r') as f:
            data = json.load(f)
            if "error" in data: return None, data["error"]
            return data, None
    except: return None, "格式错误"

def format_bw(bps):
    if bps >= 1e9: return f"{bps/1e9:.2f} Gbps"
    return f"{bps/1e6:.2f} Mbps"

d_raw, d_err = load_json("direct_tcp.json")
f_raw, f_err = load_json("forward_tcp.json")

bps_d = d_raw.get('end', {}).get('sum_sent', {}).get('bits_per_second', 0) if d_raw else 0
bps_f = f_raw.get('end', {}).get('sum_sent', {}).get('bits_per_second', 0) if f_raw else 0

print("\033[1;32m┌" + "─"*60 + "┐\033[0m")
line_d = f" 直接连接 (基准): {format_bw(bps_d)}" if not d_err else f" 直接连接: ❌ {d_err[:30]}"
line_f = f" 转发连接 (TPM):  {format_bw(bps_f)}" if not f_err else f" 转发连接: ❌ {f_err[:30]}"
print(f"\033[1;32m│\033[0m {line_d:<58} \033[1;32m│\033[0m")
print(f"\033[1;32m│\033[0m {line_f:<58} \033[1;32m│\033[0m")

if bps_d > 0 and bps_f > 0:
    ratio = (bps_f / bps_d) * 100
    status = "🔥 极佳" if ratio > 90 else ("✅ 优秀" if ratio > 75 else "⚠️ 一般")
    print("\033[1;32m├" + "─"*60 + "┤\033[0m")
    print(f"\033[1;32m│\033[0m 转发效率: {ratio:.2f}%  ({status})" + " "*(60-24-len(status)) + "\033[1;32m│\033[0m")
print("\033[1;32m└" + "─"*60 + "┘\033[0m")
PYEOF
fi

echo ""
echo_color $GREEN "✨ TCP 性能测试全部完成！"
echo ""
