package fd

import (
	"errors"
	"sync"
	"sync/atomic"
)

// fd64_t 是 64 位文件描述符标识
// 对应 C++ 中的 fd64_t
type fd64_t uint64

// Manager 文件描述符管理器
// 用于管理 fd64_t 和实际 fd 之间的映射
// 对应 C++ 中的 fd_manager_t
type Manager struct {
	mu          sync.RWMutex
	fd64ToInfo  map[fd64_t]*info
	fdToInfo    map[int]*info
	nextID      uint64
	maxFD       int
	lastFd64    fd64_t
	lastFd      int
}

// info 文件描述符信息
type info struct {
	fd64   fd64_t
	fd     int
	isTCP  bool
	refCnt int32
}

// ErrFDNotFound 文件描述符未找到
var ErrFDNotFound = errors.New("文件描述符未找到")

// ErrFDNotExist 文件描述符不存在
var ErrFDNotExist = errors.New("文件描述符不存在")

// New 创建新的文件描述符管理器
func New(maxFD int) *Manager {
	return &Manager{
		fd64ToInfo: make(map[fd64_t]*info),
		fdToInfo:   make(map[int]*info),
		maxFD:      maxFD,
		lastFd64:   0,
		lastFd:     -1,
	}
}

// Create 创建新的 fd64 标识
// 对应 C++ 中的 fd_manager_t::create()
func (m *Manager) Create(fd int) fd64_t {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成唯一的 fd64
	id := atomic.AddUint64(&m.nextID, 1)

	info := &info{
		fd64:   fd64_t(id),
		fd:     fd,
		isTCP:  false,
		refCnt: 1,
	}

	m.fd64ToInfo[info.fd64] = info
	m.fdToInfo[fd] = info

	return info.fd64
}

// Exist 检查 fd64 是否存在
// 对应 C++ 中的 fd_manager_t::exist()
func (m *Manager) Exist(fd64 fd64_t) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.fd64ToInfo[fd64]
	return ok
}

// ExistInfo 检查 fd64 是否存在信息
// 对应 C++ 中的 fd_manager_t::exist_info()
func (m *Manager) ExistInfo(fd64 fd64_t) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.fd64ToInfo[fd64]
	return ok
}

// ToFD 将 fd64 转换为实际 fd
// 对应 C++ 中的 fd_manager_t::to_fd()
func (m *Manager) ToFD(fd64 fd64_t) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return -1, ErrFDNotFound
	}
	return info.fd, nil
}

// GetInfo 获取 fd64 对应的信息
// 对应 C++ 中的 fd_manager_t::get_info()
func (m *Manager) GetInfo(fd64 fd64_t) (*info, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return nil, ErrFDNotExist
	}
	return info, nil
}

// SetTCP 设置 fd64 对应的连接为 TCP 类型
// 对应 C++ 中的 fd_manager_t::get_info().is_tcp = 1
func (m *Manager) SetTCP(fd64 fd64_t) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return ErrFDNotExist
	}
	info.isTCP = true
	return nil
}

// SetUDP 设置 fd64 对应的连接为 UDP 类型
func (m *Manager) SetUDP(fd64 fd64_t) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return ErrFDNotExist
	}
	info.isTCP = false
	return nil
}

// IsTCP 检查 fd64 对应的连接是否为 TCP
func (m *Manager) IsTCP(fd64 fd64_t) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return false, ErrFDNotExist
	}
	return info.isTCP, nil
}

// IncRef 增加引用计数
func (m *Manager) IncRef(fd64 fd64_t) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return ErrFDNotExist
	}
	atomic.AddInt32(&info.refCnt, 1)
	return nil
}

// DecRef 减少引用计数
func (m *Manager) DecRef(fd64 fd64_t) (int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return 0, ErrFDNotExist
	}
	return atomic.AddInt32(&info.refCnt, -1), nil
}

// RefCnt 获取引用计数
func (m *Manager) RefCnt(fd64 fd64_t) (int32, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return 0, ErrFDNotExist
	}
	return atomic.LoadInt32(&info.refCnt), nil
}

// Close 关闭 fd64 对应的文件描述符
// 对应 C++ 中的 fd_manager_t::fd64_close()
func (m *Manager) Close(fd64 fd64_t) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, ok := m.fd64ToInfo[fd64]
	if !ok {
		return ErrFDNotExist
	}

	// 减少引用计数
	if atomic.AddInt32(&info.refCnt, -1) > 0 {
		return nil // 还有引用,不能关闭
	}

	// 移除映射
	delete(m.fd64ToInfo, fd64)
	delete(m.fdToInfo, info.fd)

	return nil
}

// Count 返回当前管理的文件描述符数量
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.fd64ToInfo)
}
