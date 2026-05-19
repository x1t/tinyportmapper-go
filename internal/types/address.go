package types

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
)

// AddressFamily 地址族类型
type AddressFamily int

const (
	// IPv4 IPv4 地址族
	IPv4 AddressFamily = syscall.AF_INET
	// IPv6 IPv6 地址族
	IPv6 AddressFamily = syscall.AF_INET6
)

// Address 统一的网络地址结构,支持 IPv4 和 IPv6
// 对应 C++ 中的 address_t 结构
type Address struct {
	Family AddressFamily
	IP     net.IP
	Port   uint16
}

// NewAddress 创建新的地址对象
func NewAddress(family AddressFamily, ip net.IP, port uint16) *Address {
	return &Address{
		Family: family,
		IP:     ip,
		Port:   port,
	}
}

// NewAddressFromString 从字符串解析地址
// 支持格式: "1.2.3.4:80", "[::1]:80", "1.2.3.4", "[::1]"
func NewAddressFromString(s string) (*Address, error) {
	var ip net.IP
	var port uint16
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		if strings.Contains(err.Error(), "missing port") {
			host = s
			port = 0
		} else {
			return nil, fmt.Errorf("解析地址失败: %w", err)
		}
	}

	ip = net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("无效的 IP 地址: %s", host)
	}

	if portStr != "" {
		p, err := strconv.ParseUint(portStr, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("无效的端口号: %s", portStr)
		}
		port = uint16(p)
	}

	family := IPv4
	if ip.To4() == nil {
		family = IPv6
	}

	return &Address{
		Family: family,
		IP:     ip,
		Port:   port,
	}, nil
}

// FromSockaddr 从 sockaddr 构造地址
func FromSockaddr(sa syscall.Sockaddr) *Address {
	switch v := sa.(type) {
	case *syscall.SockaddrInet4:
		ip := net.IPv4(v.Addr[0], v.Addr[1], v.Addr[2], v.Addr[3])
		return &Address{
			Family: IPv4,
			IP:     ip,
			Port:   uint16(v.Port),
		}
	case *syscall.SockaddrInet6:
		ip := make([]byte, 16)
		copy(ip, v.Addr[:])
		return &Address{
			Family: IPv6,
			IP:     net.IP(ip),
			Port:   uint16(v.Port),
		}
	default:
		return nil
	}
}

// ToSockaddr 转换为 syscall.Sockaddr
func (a *Address) ToSockaddr() (syscall.Sockaddr, error) {
	if a.IP.To4() != nil {
		sa := &syscall.SockaddrInet4{
			Port: int(a.Port),
		}
		copy(sa.Addr[:], a.IP.To4())
		return sa, nil
	}

	sa := &syscall.SockaddrInet6{
		Port: int(a.Port),
	}
	copy(sa.Addr[:], a.IP.To16())
	return sa, nil
}

// ToUDPAddr 转换为 net.UDPAddr
func (a *Address) ToUDPAddr() *net.UDPAddr {
	return &net.UDPAddr{
		IP:   a.IP,
		Port: int(a.Port),
		Zone: "",
	}
}

// NewAddressFromUDPAddr 从 net.UDPAddr 创建 Address
func NewAddressFromUDPAddr(udp *net.UDPAddr) *Address {
	family := IPv4
	if udp.IP.To4() == nil {
		family = IPv6
	}
	return &Address{
		Family: family,
		IP:     udp.IP,
		Port:   uint16(udp.Port),
	}
}

// ToTCPAddr 转换为 net.TCPAddr
func (a *Address) ToTCPAddr() *net.TCPAddr {
	return &net.TCPAddr{
		IP:   a.IP,
		Port: int(a.Port),
		Zone: "",
	}
}

// String 返回地址字符串
// IPv6 地址会被方括号包裹: "[::1]:80"
func (a *Address) String() string {
	if a.IP.To4() != nil || a.Family == IPv4 {
		return net.JoinHostPort(a.IP.String(), strconv.Itoa(int(a.Port)))
	}
	return "[" + a.IP.String() + "]:" + strconv.Itoa(int(a.Port))
}

// Equal 比较两个地址是否相等
func (a *Address) Equal(b *Address) bool {
	if a.Family != b.Family {
		return false
	}
	if !a.IP.Equal(b.IP) {
		return false
	}
	return a.Port == b.Port
}

// IsZero 检查地址是否为零值
func (a *Address) IsZero() bool {
	return a.Port == 0 && a.IP == nil
}

// IsIPv4 检查是否为 IPv4 地址
func (a *Address) IsIPv4() bool {
	return a.IP.To4() != nil
}

// IsIPv6 检查是否为 IPv6 地址
func (a *Address) IsIPv6() bool {
	return a.IP.To4() == nil
}

// Size 返回 sockaddr 结构的大小
func (a *Address) Size() int {
	if a.IsIPv4() {
		return SizeofSockaddrInet4
	}
	return SizeofSockaddrInet6
}

// Hash 计算地址的哈希值
// 对应 C++ 中的 sdbm 哈希函数
func (a *Address) Hash() uint32 {
	var data []byte
	if a.IsIPv4() {
		data = make([]byte, 6)
		copy(data[0:4], a.IP.To4())
		data[4] = byte(a.Port >> 8)
		data[5] = byte(a.Port & 0xFF)
	} else {
		data = make([]byte, 18)
		copy(data[0:16], a.IP.To16())
		data[16] = byte(a.Port >> 8)
		data[17] = byte(a.Port & 0xFF)
	}

	hash := uint32(0)
	for _, c := range data {
		hash = ((hash << 5) + hash) ^ uint32(c)
	}
	return hash
}
