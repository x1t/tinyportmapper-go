package types

// 这些常量在 POSIX 和 Windows 上值相同（网络协议标准）
const (
	SizeofSockaddrInet4 = 16 // struct sockaddr_in 的大小
	SizeofSockaddrInet6 = 28 // struct sockaddr_in6 的大小
)
