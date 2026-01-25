package conn

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/x1t/tinyportmapper-go/internal/types"
)

func TestNewUDPConn(t *testing.T) {
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	assert.NotNil(t, conn)
	assert.Equal(t, addr, conn.Addr)
	assert.Equal(t, 10, conn.LocalListen)
	assert.Equal(t, 20, conn.FD)
	assert.Equal(t, uint64(21), conn.FD64)
	assert.NotZero(t, conn.LastActive())
}

func TestUDPConn_UpdateActivity(t *testing.T) {
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	initialActive := conn.LastActive()
	
	// 等待一下确保时间戳变化
	time.Sleep(2 * time.Millisecond)
	
	// 更新活跃时间
	conn.UpdateActivity()
	newActive := conn.LastActive()

	assert.Greater(t, newActive, initialActive)
}

func TestUDPConn_RefCounting(t *testing.T) {
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	// 初始引用计数为0
	assert.Equal(t, int32(0), conn.RefCnt())

	// 增加引用
	conn.IncRef()
	assert.Equal(t, int32(1), conn.RefCnt())

	conn.IncRef()
	assert.Equal(t, int32(2), conn.RefCnt())

	// 减少引用
	assert.False(t, conn.DecRef())
	assert.Equal(t, int32(1), conn.RefCnt())

	assert.True(t, conn.DecRef())
	assert.Equal(t, int32(0), conn.RefCnt())
}

func TestUDPConn_IsClosed(t *testing.T) {
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	assert.False(t, conn.IsClosed())

	conn.Close()
	assert.True(t, conn.IsClosed())

	// 再次关闭不应该改变状态
	conn.Close()
	assert.True(t, conn.IsClosed())
}

func TestUDPConn_Close(t *testing.T) {
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	err := conn.Close()
	assert.NoError(t, err)
	assert.Equal(t, -1, conn.FD)
}

func TestUDPConn_AddrString(t *testing.T) {
	addr, _ := types.NewAddressFromString("192.168.1.1:8080")
	conn := NewUDPConn(addr, 10, 20, 21)

	assert.Equal(t, "192.168.1.1:8080", conn.AddrString())
}

func TestNewUDPManager(t *testing.T) {
	manager := NewUDPManager(100)

	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())
}

func TestUDPManager_Add(t *testing.T) {
	manager := NewUDPManager(10)
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	// 添加连接
	result := manager.Add(conn)
	assert.True(t, result)
	assert.Equal(t, 1, manager.Count())

	// 重复添加应该失败
	result = manager.Add(conn)
	assert.False(t, result)
	assert.Equal(t, 1, manager.Count())
}

func TestUDPManager_Add_MaxConns(t *testing.T) {
	manager := NewUDPManager(2)

	// 添加两个连接
	addr1 := types.NewAddress(types.IPv4, nil, 8080)
	conn1 := NewUDPConn(addr1, 10, 20, 21)
	assert.True(t, manager.Add(conn1))

	addr2 := types.NewAddress(types.IPv4, nil, 8081)
	conn2 := NewUDPConn(addr2, 11, 22, 23)
	assert.True(t, manager.Add(conn2))

	// 第三个应该失败
	addr3 := types.NewAddress(types.IPv4, nil, 8082)
	conn3 := NewUDPConn(addr3, 12, 23, 24)
	assert.False(t, manager.Add(conn3))
}

func TestUDPManager_Get(t *testing.T) {
	manager := NewUDPManager(10)
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	manager.Add(conn)

	result, ok := manager.Get(addr)
	assert.True(t, ok)
	assert.Equal(t, conn, result)

	// 获取不存在的地址
	addr2, _ := types.NewAddressFromString("192.168.1.1:9999")
	result, ok = manager.Get(addr2)
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestUDPManager_Remove(t *testing.T) {
	manager := NewUDPManager(10)
	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)

	manager.Add(conn)
	assert.Equal(t, 1, manager.Count())

	manager.Remove(conn)
	assert.Equal(t, 0, manager.Count())

	// 再次移除应该安全
	manager.Remove(conn)
	assert.Equal(t, 0, manager.Count())
}

func TestUDPManager_Count(t *testing.T) {
	manager := NewUDPManager(10)

	assert.Equal(t, 0, manager.Count())

	for i := 0; i < 5; i++ {
		addr := types.NewAddress(types.IPv4, nil, uint16(8080+i))
		conn := NewUDPConn(addr, 10+i, 20+i, uint64(21+i))
		manager.Add(conn)
	}

	assert.Equal(t, 5, manager.Count())
}

func TestUDPManager_Iter(t *testing.T) {
	manager := NewUDPManager(10)

	// 空迭代器应该安全
	count := 0
	manager.Iter(func(conn *UDPConn) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count)

	// 添加几个连接
	for i := 0; i < 3; i++ {
		addr := types.NewAddress(types.IPv4, nil, uint16(8080+i))
		conn := NewUDPConn(addr, 10+i, 20+i, uint64(21+i))
		manager.Add(conn)
	}

	// 迭代
	count = 0
	manager.Iter(func(conn *UDPConn) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count)

	// 提前终止迭代
	count = 0
	manager.Iter(func(conn *UDPConn) bool {
		count++
		return count < 2 // 只迭代前两个
	})
	assert.Equal(t, 2, count)
}

func TestUDPManager_Len(t *testing.T) {
	manager := NewUDPManager(10)
	assert.Equal(t, 0, manager.Len())

	addr := types.NewAddress(types.IPv4, nil, 8080)
	conn := NewUDPConn(addr, 10, 20, 21)
	manager.Add(conn)

	assert.Equal(t, 1, manager.Len())
}

func TestUDPManager_DuplicateAddress(t *testing.T) {
	manager := NewUDPManager(10)
	addr := types.NewAddress(types.IPv4, nil, uint16(8080))

	// 使用相同地址创建两个连接
	conn1 := NewUDPConn(addr, 10, 20, uint64(21))
	conn2 := NewUDPConn(addr, 11, 22, uint64(23))

	// 第一个应该成功
	assert.True(t, manager.Add(conn1))

	// 第二个应该失败（地址冲突）
	assert.False(t, manager.Add(conn2))
}
