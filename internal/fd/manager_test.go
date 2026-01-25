package fd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	// 创建文件描述符管理器
	mgr := New(-1)
	assert.NotNil(t, mgr)
	assert.Equal(t, 0, mgr.Count())
}

func TestManager_Create(t *testing.T) {
	mgr := New(-1)

	// 创建测试文件以获取真实fd
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	// 创建 fd64
	fd64 := mgr.Create(int(file.Fd()))

	// 验证创建成功
	assert.True(t, mgr.Exist(fd64))
	assert.Equal(t, 1, mgr.Count())

	// 验证可以转换为真实fd
	fd, err := mgr.ToFD(fd64)
	require.NoError(t, err)
	assert.Equal(t, int(file.Fd()), fd)
}

func TestManager_Exist(t *testing.T) {
	mgr := New(-1)

	// 初始不存在
	assert.False(t, mgr.Exist(1))

	// 创建
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	assert.True(t, mgr.Exist(fd64))
}

func TestManager_ToFD(t *testing.T) {
	mgr := New(-1)

	// 查找不存在的fd64
	_, err := mgr.ToFD(999)
	assert.Error(t, err)
	assert.Equal(t, ErrFDNotFound, err)

	// 创建并查找
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	fd, err := mgr.ToFD(fd64)
	require.NoError(t, err)
	assert.Equal(t, int(file.Fd()), fd)
}

func TestManager_RefCounting(t *testing.T) {
	mgr := New(-1)

	// 创建
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))

	// 初始引用计数为1
	cnt, err := mgr.RefCnt(fd64)
	require.NoError(t, err)
	assert.Equal(t, int32(1), cnt)

	// 增加引用
	err = mgr.IncRef(fd64)
	require.NoError(t, err)
	cnt, err = mgr.RefCnt(fd64)
	require.NoError(t, err)
	assert.Equal(t, int32(2), cnt)

	// 减少引用
	cnt, err = mgr.DecRef(fd64)
	require.NoError(t, err)
	assert.Equal(t, int32(1), cnt)

	// 再减少一个应该回到0
	cnt, err = mgr.DecRef(fd64)
	require.NoError(t, err)
	assert.Equal(t, int32(0), cnt)
}

func TestManager_Close(t *testing.T) {
	mgr := New(-1)

	// 创建
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	assert.Equal(t, 1, mgr.Count())

	// 关闭
	err = mgr.Close(fd64)
	require.NoError(t, err)
	assert.Equal(t, 0, mgr.Count())

	// 再次关闭应该返回错误
	err = mgr.Close(fd64)
	assert.Error(t, err)
}

func TestManager_SetTCP(t *testing.T) {
	mgr := New(-1)

	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))

	// 初始是UDP
	isTCP, err := mgr.IsTCP(fd64)
	require.NoError(t, err)
	assert.False(t, isTCP)

	// 设置为TCP
	err = mgr.SetTCP(fd64)
	require.NoError(t, err)

	isTCP, err = mgr.IsTCP(fd64)
	require.NoError(t, err)
	assert.True(t, isTCP)
}

// TestManager_ExistInfo 测试 ExistInfo 函数
func TestManager_ExistInfo(t *testing.T) {
	mgr := New(-1)

	// 初始不存在
	assert.False(t, mgr.ExistInfo(1))

	// 创建
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	assert.True(t, mgr.ExistInfo(fd64))
}

// TestManager_GetInfo 测试 GetInfo 函数
func TestManager_GetInfo(t *testing.T) {
	mgr := New(-1)

	// 获取不存在的 fd64 信息
	_, err := mgr.GetInfo(999)
	assert.Error(t, err)
	assert.Equal(t, ErrFDNotExist, err)

	// 创建并获取
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	info, err := mgr.GetInfo(fd64)
	require.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, fd64, info.fd64)
}

// TestManager_SetUDP 测试 SetUDP 函数
func TestManager_SetUDP(t *testing.T) {
	mgr := New(-1)

	// 设置不存在的 fd64
	err := mgr.SetUDP(999)
	assert.Error(t, err)
	assert.Equal(t, ErrFDNotExist, err)

	// 创建并设置 UDP
	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))

	// 先设置为 TCP
	err = mgr.SetTCP(fd64)
	require.NoError(t, err)

	isTCP, err := mgr.IsTCP(fd64)
	require.NoError(t, err)
	assert.True(t, isTCP)

	// 再设置为 UDP
	err = mgr.SetUDP(fd64)
	require.NoError(t, err)

	isTCP, err = mgr.IsTCP(fd64)
	require.NoError(t, err)
	assert.False(t, isTCP)
}

func TestManager_CloseRefCount(t *testing.T) {
	mgr := New(-1)

	file, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	defer os.Remove(file.Name())
	defer file.Close()

	fd64 := mgr.Create(int(file.Fd()))
	mgr.IncRef(fd64) // 引用计数变为2

	// 关闭时引用计数>0，不应该真正关闭
	err = mgr.Close(fd64)
	require.NoError(t, err)
	assert.Equal(t, 1, mgr.Count()) // 仍然存在

	// 再关闭一次，引用计数变为0，应该关闭
	err = mgr.Close(fd64)
	require.NoError(t, err)
	assert.Equal(t, 0, mgr.Count()) // 已移除
}
