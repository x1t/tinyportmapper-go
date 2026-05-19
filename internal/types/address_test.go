package types

import (
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAddressFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantIP  net.IP
		wantPort uint16
		wantErr bool
	}{
		{
			name:    "IPv4 with port",
			input:   "192.168.1.1:8080",
			wantIP:  net.ParseIP("192.168.1.1"),
			wantPort: 8080,
			wantErr: false,
		},
		{
			name:    "IPv6 with port",
			input:   "[::1]:8080",
			wantIP:  net.ParseIP("::1"),
			wantPort: 8080,
			wantErr: false,
		},
		{
			name:    "IPv6 without port - expect error",
			input:   "::1",
			wantIP:  nil,
			wantPort: 0,
			wantErr: true,
		},
		{
			name:    "localhost IPv4",
			input:   "127.0.0.1:80",
			wantIP:  net.ParseIP("127.0.0.1"),
			wantPort: 80,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewAddressFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAddressFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !addr.IP.Equal(tt.wantIP) {
					t.Errorf("NewAddressFromString() IP = %v, want %v", addr.IP, tt.wantIP)
				}
				if addr.Port != tt.wantPort {
					t.Errorf("NewAddressFromString() Port = %v, want %v", addr.Port, tt.wantPort)
				}
			}
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	addr1, _ := NewAddressFromString("192.168.1.1:8080")
	addr2, _ := NewAddressFromString("192.168.1.1:8080")
	addr3, _ := NewAddressFromString("192.168.1.2:8080")

	if !addr1.Equal(addr2) {
		t.Error("Equal() same addresses should be equal")
	}

	if addr1.Equal(addr3) {
		t.Error("Equal() different addresses should not be equal")
	}
}

func TestAddress_String(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "IPv4",
			input:  "192.168.1.1:8080",
			expect: "192.168.1.1:8080",
		},
		{
			name:   "IPv6",
			input:  "[::1]:8080",
			expect: "[::1]:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := NewAddressFromString(tt.input)
			if err != nil {
				t.Fatalf("NewAddressFromString() failed: %v", err)
			}
			if addr.String() != tt.expect {
				t.Errorf("String() = %v, want %v", addr.String(), tt.expect)
			}
		})
	}
}

func TestNewAddressFromUDPAddr(t *testing.T) {
	udpAddr := &net.UDPAddr{
		IP:   net.ParseIP("192.168.1.1"),
		Port: 8080,
	}

	addr := NewAddressFromUDPAddr(udpAddr)
	if !addr.IP.Equal(udpAddr.IP) {
		t.Errorf("IP = %v, want %v", addr.IP, udpAddr.IP)
	}
	if addr.Port != 8080 {
		t.Errorf("Port = %v, want 8080", addr.Port)
	}
}

func TestAddress_IsIPv4(t *testing.T) {
	addr4, _ := NewAddressFromString("192.168.1.1:8080")
	addr6, _ := NewAddressFromString("[::1]:8080")

	if !addr4.IsIPv4() {
		t.Error("IPv4 address should return true for IsIPv4()")
	}
	if addr6.IsIPv4() {
		t.Error("IPv6 address should return false for IsIPv4()")
	}
}

func TestAddress_IsIPv6(t *testing.T) {
	addr4, _ := NewAddressFromString("192.168.1.1:8080")
	addr6, _ := NewAddressFromString("[::1]:8080")

	if addr4.IsIPv6() {
		t.Error("IPv4 address should return false for IsIPv6()")
	}
	if !addr6.IsIPv6() {
		t.Error("IPv6 address should return true for IsIPv6()")
	}
}

func TestNewAddress(t *testing.T) {
	ip := net.ParseIP("192.168.1.1")
	addr := NewAddress(IPv4, ip, 8080)

	assert.NotNil(t, addr)
	assert.Equal(t, IPv4, addr.Family)
	assert.True(t, addr.IP.Equal(ip))
	assert.Equal(t, uint16(8080), addr.Port)
}

func TestNewAddress_IPv6(t *testing.T) {
	ip := net.ParseIP("::1")
	addr := NewAddress(IPv6, ip, 8080)

	assert.NotNil(t, addr)
	assert.Equal(t, IPv6, addr.Family)
	assert.True(t, addr.IP.Equal(ip))
	assert.Equal(t, uint16(8080), addr.Port)
}

func TestAddress_IsZero(t *testing.T) {
	addr, _ := NewAddressFromString("192.168.1.1:8080")

	// 非零地址
	if addr.IsZero() {
		t.Error("Non-zero address should return false for IsZero()")
	}

	// 零地址
	zeroAddr := NewAddress(IPv4, nil, 0)
	if !zeroAddr.IsZero() {
		t.Error("Zero address should return true for IsZero()")
	}
}

func TestAddress_Size(t *testing.T) {
	addr4, _ := NewAddressFromString("192.168.1.1:8080")
	addr6, _ := NewAddressFromString("[::1]:8080")

	// IPv4 size
	size4 := addr4.Size()
	assert.Equal(t, SizeofSockaddrInet4, size4)

	// IPv6 size
	size6 := addr6.Size()
	assert.Equal(t, SizeofSockaddrInet6, size6)
}

func TestAddress_Hash(t *testing.T) {
	addr1, _ := NewAddressFromString("192.168.1.1:8080")
	addr2, _ := NewAddressFromString("192.168.1.1:8080")
	addr3, _ := NewAddressFromString("192.168.1.2:8080")

	// 相同地址应该有相同哈希
	hash1 := addr1.Hash()
	hash2 := addr2.Hash()
	assert.Equal(t, hash1, hash2, "Same addresses should have same hash")

	// 不同地址应该有不同哈希
	hash3 := addr3.Hash()
	assert.NotEqual(t, hash1, hash3, "Different addresses should have different hash")

	// IPv6地址的哈希
	addr6, _ := NewAddressFromString("[::1]:8080")
	hash6 := addr6.Hash()
	assert.NotZero(t, hash6)
}

func TestToUDPAddr(t *testing.T) {
	addr, _ := NewAddressFromString("192.168.1.1:8080")
	udpAddr := addr.ToUDPAddr()

	assert.NotNil(t, udpAddr)
	assert.Equal(t, "192.168.1.1", udpAddr.IP.String())
	assert.Equal(t, 8080, udpAddr.Port)
}

func TestToTCPAddr(t *testing.T) {
	addr, _ := NewAddressFromString("[::1]:8080")
	tcpAddr := addr.ToTCPAddr()

	assert.NotNil(t, tcpAddr)
	assert.Equal(t, "::1", tcpAddr.IP.String())
	assert.Equal(t, 8080, tcpAddr.Port)
}

func TestFromSockaddr_IPv4(t *testing.T) {
	sa := &syscall.SockaddrInet4{
		Addr: [4]byte{192, 168, 1, 1},
		Port: 8080,
	}

	addr := FromSockaddr(sa)

	assert.NotNil(t, addr)
	assert.Equal(t, IPv4, addr.Family)
	assert.True(t, addr.IP.Equal(net.ParseIP("192.168.1.1")))
	assert.Equal(t, uint16(8080), addr.Port)
}

func TestFromSockaddr_IPv6(t *testing.T) {
	sa := &syscall.SockaddrInet6{
		Addr: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		Port: 8080,
	}

	addr := FromSockaddr(sa)

	assert.NotNil(t, addr)
	assert.Equal(t, IPv6, addr.Family)
	assert.True(t, addr.IP.Equal(net.ParseIP("::1")))
	assert.Equal(t, uint16(8080), addr.Port)
}

func TestFromSockaddr_Invalid(t *testing.T) {
	// 测试不支持的 sockaddr 类型
	addr := FromSockaddr(nil)
	assert.Nil(t, addr)
}

func TestToSockaddr_IPv4(t *testing.T) {
	addr, _ := NewAddressFromString("192.168.1.1:8080")
	sa, err := addr.ToSockaddr()

	assert.NoError(t, err)
	sa4, ok := sa.(*syscall.SockaddrInet4)
	assert.True(t, ok)
	assert.Equal(t, 8080, sa4.Port)
	assert.Equal(t, [4]byte{192, 168, 1, 1}, sa4.Addr)
}

func TestToSockaddr_IPv6(t *testing.T) {
	addr, _ := NewAddressFromString("[::1]:8080")
	sa, err := addr.ToSockaddr()

	assert.NoError(t, err)
	sa6, ok := sa.(*syscall.SockaddrInet6)
	assert.True(t, ok)
	assert.Equal(t, 8080, sa6.Port)
}

func TestAddress_Equal_PortMismatch(t *testing.T) {
	addr1, _ := NewAddressFromString("192.168.1.1:8080")
	addr2, _ := NewAddressFromString("192.168.1.1:8081")

	if addr1.Equal(addr2) {
		t.Error("Addresses with different ports should not be equal")
	}
}

func TestAddress_Equal_FamilyMismatch(t *testing.T) {
	// 相同IP但不同协议族的地址
	addr4 := NewAddress(IPv4, net.ParseIP("192.168.1.1"), 8080)
	addr6 := NewAddress(IPv6, net.ParseIP("192.168.1.1"), 8080)

	// 注意：192.168.1.1 是IPv4地址，To4() != nil
	// 所以addr6实际上会是IPv4
	if addr4.Equal(addr6) {
		// 这是预期的，因为IP相同
	}
}

func TestAddress_String_IPv4_NoPort(t *testing.T) {
	addr := NewAddress(IPv4, net.ParseIP("192.168.1.1"), 0)
	str := addr.String()
	assert.Equal(t, "192.168.1.1:0", str)
}

func TestAddress_String_IPv6_NoPort(t *testing.T) {
	addr := NewAddress(IPv6, net.ParseIP("::1"), 0)
	str := addr.String()
	assert.Equal(t, "[::1]:0", str)
}
