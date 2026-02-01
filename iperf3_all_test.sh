#!/bin/bash
# tinyPortMapper Go版本 综合性能测试脚本 (TCP + UDP)
# 用法: ./iperf3_all_test.sh [--debug]

set -e

# --- 脚本定位 ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TCP_TEST="$SCRIPT_DIR/iperf3_tcp_test.sh"
UDP_TEST="$SCRIPT_DIR/iperf3_udp_test.sh"
LOG_DIR="/tmp/iperf3_test"

# --- 颜色定义 ---
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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

# --- 初始检查 ---
if [[ ! -f "$TCP_TEST" || ! -f "$UDP_TEST" ]]; then
    echo_color $RED "❌ 错误: 找不到子测试脚本 ($TCP_TEST 或 $UDP_TEST)"
    exit 1
fi

mkdir -p "$LOG_DIR"
print_header "tinyPortMapper Go 版本 综合性能测试"

# 参数透传
EXTRA_ARGS="$@"

# ==================== 执行子脚本 ====================

echo_color $BLUE ">>> [1/2] 正在开始进行 TCP 性能测试..."
bash "$TCP_TEST" $EXTRA_ARGS
echo ""

echo_color $BLUE ">>> [2/2] 正在开始进行 UDP 性能测试..."
bash "$UDP_TEST" $EXTRA_ARGS
echo ""

# ==================== 深度汇总分析 ====================
print_header "综合性能深度分析报告"

python3 << 'PYEOF'
import json
import os

LOG_DIR = "/tmp/iperf3_test"

def load_json(filename):
    path = os.path.join(LOG_DIR, filename)
    if not os.path.exists(path): return None
    try:
        with open(path, 'r') as f:
            return json.load(f)
    except: return None

def get_bps(data, is_udp=False):
    if not data: return 0
    end = data.get('end', {})
    if is_udp:
        return end.get('sum_received', {}).get('bits_per_second', 0) or end.get('sum_sent', {}).get('bits_per_second', 0)
    return end.get('sum_sent', {}).get('bits_per_second', 0)

def format_bw(bps):
    if bps >= 1e9: return f"{bps/1e9:.2f} Gbps"
    return f"{bps/1e6:.2f} Mbps"

# 加载所有数据
tcp_d = load_json("direct_tcp.json")
tcp_f = load_json("forward_tcp.json")
udp_d = load_json("direct_udp.json")
udp_f = load_json("forward_udp.json")

bps_tcp_d = get_bps(tcp_d)
bps_tcp_f = get_bps(tcp_f)
bps_udp_d = get_bps(udp_d, True)
bps_udp_f = get_bps(udp_f, True)

# 打印对比表
print("\033[1;32m┌" + "─"*76 + "┐\033[0m")
print("\033[1;32m│\033[0m" + " 性能对比概览 ".center(76) + "\033[1;32m│\033[0m")
print("\033[1;32m├" + "─"*25 + "┬" + "─"*25 + "┬" + "─"*24 + "┤\033[0m")
print(f"\033[1;32m│\033[0m 测试模式{'':<17} \033[1;32m│\033[0m 直接连接 (基准){'':<9} \033[1;32m│\033[0m 转发连接 (TPM){'':<9} \033[1;32m│\033[0m")
print("\033[1;32m├" + "─"*25 + "┼" + "─"*25 + "┼" + "─"*24 + "┤\033[0m")

print(f"\033[1;32m│\033[0m TCP 吞吐量{'':<15} \033[1;32m│\033[0m {format_bw(bps_tcp_d):<23} \033[1;32m│\033[0m {format_bw(bps_tcp_f):<22} \033[1;32m│\033[0m")
print(f"\033[1;32m│\033[0m UDP 吞吐量{'':<15} \033[1;32m│\033[0m {format_bw(bps_udp_d):<23} \033[1;32m│\033[0m {format_bw(bps_udp_f):<22} \033[1;32m│\033[0m")
print("\033[1;32m└" + "─"*76 + "┘\033[0m")

# 效率统计
print("\n\033[1;33m📊 效率分析:\033[0m")
if bps_tcp_d > 0:
    ratio_tcp = (bps_tcp_f / bps_tcp_d) * 100
    print(f" - TCP 转发效率: {ratio_tcp:.2f}%")
if bps_udp_d > 0:
    ratio_udp = (bps_udp_f / bps_udp_d) * 100
    print(f" - UDP 转发效率: {ratio_udp:.2f}%")

# 丢包统计
if udp_f:
    loss = udp_f.get('end', {}).get('sum_received', {}).get('lost_percent', 0)
    print(f" - UDP 转发丢包率: {loss:.2f}%")
PYEOF

echo ""
echo_color $GREEN "🎉 综合性能测试全部完成！详细日志在: $LOG_DIR/"
echo ""
