#!/bin/bash
# tinyPortMapper Go版本 iperf3 性能测试脚本
# 用法: ./iperf3_test.sh

set -e

TINYPORTMAPPER="/root/tinyportmapper/tinyportmapper-go/tinyportmapper-go"
LISTEN_PORT=3322
SERVER_PORT=5201
TEST_DURATION=5
PARALLEL_STREAMS=4
LOG_DIR="/tmp/iperf3_test"

mkdir -p "$LOG_DIR"

cleanup() {
    pkill -9 iperf3 2>/dev/null || true
    pkill -9 tinyportmapper-go 2>/dev/null || true
}
trap cleanup EXIT

pkill -9 iperf3 2>/dev/null || true
pkill -9 tinyportmapper-go 2>/dev/null || true
sleep 1

echo "启动 iperf3 服务器..."
iperf3 -s -p $SERVER_PORT &
sleep 2

echo "=== 第一轮: 直接连接测试 ==="
iperf3 -c 127.0.0.1 -p $SERVER_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > $LOG_DIR/direct_tcp.json

echo "启动 tinyportmapper..."
$TINYPORTMAPPER -l 127.0.0.1:$LISTEN_PORT -r 127.0.0.1:$SERVER_PORT -t 2>&1 &
sleep 2

echo "=== 第二轮: 转发测试 ==="
iperf3 -c 127.0.0.1 -p $LISTEN_PORT -t $TEST_DURATION -P $PARALLEL_STREAMS --json > $LOG_DIR/forward_tcp.json

pkill -9 -f "tinyportmapper-go.*-l.*$LISTEN_PORT" 2>/dev/null

echo ""
echo "=== 测试结果对比 ==="
python3 << 'PYEOF'
import json
import os

LOG_DIR = "/tmp/iperf3_test"

def parse_result(json_file, key="sum_sent"):
    try:
        with open(os.path.join(LOG_DIR, json_file), 'r') as f:
            data = json.load(f)
        end = data.get('end', {})
        if key in end and 'bits_per_second' in end[key]:
            bits = end[key]['bits_per_second']
            if bits > 1e9:
                return f"{bits/1e9:.2f} Gbits/sec"
            else:
                return f"{bits/1e6:.2f} Mbits/sec"
        return "N/A"
    except Exception as e:
        return f"Error: {e}"

TCP_DIRECT = parse_result("direct_tcp.json")
TCP_FORWARD = parse_result("forward_tcp.json")

print()
print("┌─────────────────────────────────────────────────────────────┐")
print("│                        TCP 测试结果                          │")
print("├─────────────────────────────────────────────────────────────┤")
print("│  直接连接 (5201)    │  转发连接 (3322)                      │")
print("├─────────────────────────────────────────────────────────────┤")
print(f"│  发送: {TCP_DIRECT:<15} │  发送: {TCP_FORWARD:<28} │")
print("└─────────────────────────────────────────────────────────────┘")

if "Gbits" in TCP_DIRECT and "Gbits" in TCP_FORWARD:
    direct_tcp_val = float(TCP_DIRECT.split()[0])
    forward_tcp_val = float(TCP_FORWARD.split()[0])
    ratio = forward_tcp_val / direct_tcp_val * 100
    print(f"\n性能对比: TCP 转发性能是直接的 {ratio:.1f}%")
PYEOF

echo ""
echo "测试完成!"
