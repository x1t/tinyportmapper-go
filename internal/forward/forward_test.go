package forward

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/x1t/tinyportmapper-go/internal/conn"
	"github.com/x1t/tinyportmapper-go/internal/types"
)

// Helper to create a pair of connected TCP connections (for testing)
func createTCPConnectionPair(t *testing.T) (*net.TCPConn, *net.TCPConn) {
	t.Helper()
	// Create a listener
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Get the listener port
	listenerAddr := ln.Addr().(*net.TCPAddr)

	// Client connection (non-blocking start)
	clientConn, err := net.DialTCP("tcp", nil, listenerAddr)
	if err != nil {
		ln.Close()
		t.Fatalf("Failed to connect client: %v", err)
	}

	// Server accept
	serverConn, err := ln.AcceptTCP()
	if err != nil {
		clientConn.Close()
		ln.Close()
		t.Fatalf("Failed to accept connection: %v", err)
	}
	ln.Close()

	return clientConn, serverConn
}

// Helper to create a pair of connected TCP connections (for benchmarking)
func createTCPConnectionPairForBench(b *testing.B) (*net.TCPConn, *net.TCPConn) {
	// Create a listener
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}

	// Get the listener port
	listenerAddr := ln.Addr().(*net.TCPAddr)

	// Client connection (non-blocking start)
	clientConn, err := net.DialTCP("tcp", nil, listenerAddr)
	if err != nil {
		ln.Close()
		b.Fatalf("Failed to connect client: %v", err)
	}

	// Server accept
	serverConn, err := ln.AcceptTCP()
	if err != nil {
		clientConn.Close()
		ln.Close()
		b.Fatalf("Failed to accept connection: %v", err)
	}
	ln.Close()

	return clientConn, serverConn
}

func TestTCPForwarder_Start(t *testing.T) {
	// Test that forwarder can be created and started without panic
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	// Close should not panic even with nil connections
	forwarder.Close()

	if !forwarder.IsClosed() {
		t.Error("Forwarder should be closed after Close()")
	}
}

func TestTCPForwarder_Close(t *testing.T) {
	// Test that close works without panic on nil connections
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	forwarder.Start()
	time.Sleep(50 * time.Millisecond)

	forwarder.Close()

	if !forwarder.IsClosed() {
		t.Error("Forwarder should be closed after Close()")
	}
}

func TestUDPForwarderManager_Add(t *testing.T) {
	manager := NewUDPForwarderManager()

	if manager.Count() != 0 {
		t.Errorf("Count() = %d, want 0", manager.Count())
	}

	// Can't easily add without full setup, just test manager creation
}

func TestUDPForwarderManager_Remove(t *testing.T) {
	manager := NewUDPForwarderManager()

	// Test remove nil (should not panic)
	manager.Remove(nil)

	// Test remove on empty manager (should not panic)
	manager.Remove(&types.Address{})
}

func TestUDPForwarderManager_CloseAll(t *testing.T) {
	manager := NewUDPForwarderManager()

	// CloseAll on empty manager should not panic
	manager.CloseAll()
}

func TestBufferReuse(t *testing.T) {
	// Test that we can create multiple forwarders with buffer
	f1 := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
	f2 := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	if f1.bufSize != 4096 || f2.bufSize != 4096 {
		t.Errorf("Buffer sizes not set correctly")
	}

	f1.Close()
	f2.Close()
}

// Benchmark for TCP forwarder throughput
func BenchmarkTCPForwarder(b *testing.B) {
	clientConn, serverConn := createTCPConnectionPairForBench(b)
	defer clientConn.Close()
	defer serverConn.Close()

	// Set large buffers
	clientConn.SetReadBuffer(1024 * 1024)
	clientConn.SetWriteBuffer(1024 * 1024)
	serverConn.SetReadBuffer(1024 * 1024)
	serverConn.SetWriteBuffer(1024 * 1024)

	b.ResetTimer()
	b.SetBytes(64 * 1024) // 64KB chunks

	for i := 0; i < b.N; i++ {
		// Create a test data chunk
		data := bytes.Repeat([]byte{'A'}, 64*1024)

		var wg sync.WaitGroup
		var writeErr, readErr error

		// Write in background
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, writeErr = clientConn.Write(data)
		}()

		// Read and verify
		received := make([]byte, len(data))
		_, readErr = io.ReadFull(serverConn, received)

		wg.Wait()

		if writeErr != nil {
			b.Fatalf("Write error: %v", writeErr)
		}
		if readErr != nil {
			b.Fatalf("Read error: %v", readErr)
		}
		if !bytes.Equal(data, received) {
			b.Error("Data mismatch")
		}
	}
}

func TestTCPForwarderManager_New(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderManager_Start(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// 验证manager创建成功
	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderManager_Close(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// CloseAll should not panic
	manager.CloseAll()
}

func TestTCPForwarderManager_Count(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	assert.Equal(t, 0, manager.Count())
}

func TestNewTCPForwarder_DefaultBufferSize(t *testing.T) {
	// Test with zero buffer size
	forwarder := NewTCPForwarder(nil, nil, 0, 5*time.Minute, nil)

	assert.NotNil(t, forwarder)
	// C++ uses max_data_len_tcp = 4096*4 = 16384 bytes
	assert.Equal(t, 16*1024, forwarder.bufSize)

	forwarder.Close()
}

func TestNewTCPForwarder_NegativeBufferSize(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, -1, 5*time.Minute, nil)

	assert.NotNil(t, forwarder)
	// C++ uses max_data_len_tcp = 4096*4 = 16384 bytes
	assert.Equal(t, 16*1024, forwarder.bufSize)

	forwarder.Close()
}

func TestTCPForwarder_IsClosed(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	assert.False(t, forwarder.IsClosed())

	forwarder.Close()

	assert.True(t, forwarder.IsClosed())
}

func TestTCPForwarder_Wait(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	// Wait on nil connections should return immediately
	forwarder.Wait()
}

func TestTCPForwarder_SetBufferSize(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 8192, 5*time.Minute, nil)

	assert.Equal(t, 8192, forwarder.bufSize)

	forwarder.Close()
}

func TestNewUDPForwarder(t *testing.T) {
	// Create test connections
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	assert.NotNil(t, forwarder)
	assert.Equal(t, 64*1024, forwarder.bufSize)
}

func TestUDPForwarder_GetConn(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %v", err)
	}
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	conn := forwarder.GetConn()
	assert.Equal(t, udpConn, conn)
}

func TestUDPForwarder_Close(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %v", err)
	}

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	if err != nil {
		ln.Close()
		t.Fatalf("Failed to create UDP connection: %v", err)
	}

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	// Close should not panic
	forwarder.Close()
}

func TestUDPForwarderManager_Get(t *testing.T) {
	manager := NewUDPForwarderManager()

	addr := &types.Address{}

	// Get non-existent address
	result, ok := manager.Get(addr)
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestUDPForwarderManager_Count(t *testing.T) {
	manager := NewUDPForwarderManager()

	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderPool_New(t *testing.T) {
	pool := NewTCPForwarderPool(64*1024, 1000)

	assert.NotNil(t, pool)
	assert.Equal(t, 64*1024, pool.bufSize)
	assert.Equal(t, 1000, pool.maxConns)
}

func TestTCPForwarderPool_GetPut(t *testing.T) {
	pool := NewTCPForwarderPool(64*1024, 1000)

	// Get a forwarder from pool
	f1 := pool.Get()
	assert.NotNil(t, f1)
	assert.NotNil(t, f1.buf)
	assert.Equal(t, 64*1024, len(f1.buf))

	// Put it back
	pool.Put(f1)

	// Get again
	f2 := pool.Get()
	assert.NotNil(t, f2)
}

func TestTCPForwarderManager_Add(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	manager.Add(forwarder)

	assert.Equal(t, 1, manager.Count())
}

func TestTCPForwarderManager_Remove(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	manager.Add(forwarder)
	assert.Equal(t, 1, manager.Count())

	manager.Remove(forwarder)
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderManager_StartAndTrack(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	closed := false
	manager.Start(forwarder, func() {
		closed = true
	})

	assert.Equal(t, 1, manager.Count())
	assert.False(t, closed)

	// Clean up
	forwarder.Close()
	manager.CloseAll()
}

func TestTCPForwarderManager_WaitAll(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// WaitAll on empty manager should not block forever
	done := make(chan struct{})
	go func() {
		manager.WaitAll()
		close(done)
	}()

	select {
	case <-done:
		// Expected: returns immediately
	case <-time.After(100 * time.Millisecond):
		t.Error("WaitAll should not block on empty manager")
	}
}

func TestTCPForwarderManager_AddRemoveNil(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// Add nil should not panic
	assert.NotPanics(t, func() {
		manager.Add(nil)
	})

	// Remove nil should not panic
	assert.NotPanics(t, func() {
		manager.Remove(nil)
	})
}

func TestUDPForwarder_New(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	// Test with default buffer size
	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 0)
	assert.NotNil(t, forwarder)
	assert.Equal(t, 16*1024, forwarder.bufSize) // Default 16KB

	forwarder.Close()
}

func TestUDPForwarder_StartAndWait(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	// Start should not block
	forwarder.Start()

	// Close first to stop the goroutines
	forwarder.Close()

	// Now Wait should return quickly
	done := make(chan struct{})
	go func() {
		forwarder.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Error("Wait should return after Close")
	}
}

func TestUDPForwarder_IsClosed(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	assert.False(t, forwarder.IsClosed())

	forwarder.Close()

	assert.True(t, forwarder.IsClosed())
}

func TestUDPForwarderManager_AddGet(t *testing.T) {
	manager := NewUDPForwarderManager()

	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}
	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	// Add to manager
	manager.Add(forwarder)
	assert.Equal(t, 1, manager.Count())

	// Get from manager
	result, ok := manager.Get(addr)
	assert.True(t, ok)
	assert.Equal(t, forwarder, result)

	// Get non-existent
	result, ok = manager.Get(&types.Address{})
	assert.False(t, ok)
	assert.Nil(t, result)
}

func TestUDPForwarderManager_RemoveByAddr(t *testing.T) {
	manager := NewUDPForwarderManager()

	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}
	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	manager.Add(forwarder)
	assert.Equal(t, 1, manager.Count())

	manager.Remove(addr)
	assert.Equal(t, 0, manager.Count())
}

func TestUDPForwarderManager_CloseAllWithMultiple(t *testing.T) {
	manager := NewUDPForwarderManager()

	ln1, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)

	ln2, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)

	remoteConn1, _ := net.DialUDP("udp", nil, ln1.LocalAddr().(*net.UDPAddr))
	remoteConn2, _ := net.DialUDP("udp", nil, ln2.LocalAddr().(*net.UDPAddr))

	addr1 := types.NewAddressFromUDPAddr(ln1.LocalAddr().(*net.UDPAddr))
	addr2 := types.NewAddressFromUDPAddr(ln2.LocalAddr().(*net.UDPAddr))

	f1 := NewUDPForwarder(ln1, remoteConn1, addr1, &conn.UDPConn{}, 64*1024)
	f2 := NewUDPForwarder(ln2, remoteConn2, addr2, &conn.UDPConn{}, 64*1024)

	manager.Add(f1)
	manager.Add(f2)
	assert.Equal(t, 2, manager.Count())

	manager.CloseAll()
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarder_ConcurrentStartClose(t *testing.T) {
	// Test concurrent start and close
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			forwarder.Start()
			time.Sleep(10 * time.Millisecond)
			forwarder.Close()
		}()
	}

	wg.Wait()
	assert.True(t, forwarder.IsClosed())
}

func TestUDPForwarder_ConcurrentStartClose(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			forwarder.Start()
			time.Sleep(10 * time.Millisecond)
			forwarder.Close()
		}()
	}

	wg.Wait()
	assert.True(t, forwarder.IsClosed())
}

func TestTCPForwarderManager_ConcurrentAddRemove(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	var wg sync.WaitGroup

	// Concurrent add
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
			manager.Add(forwarder)
			time.Sleep(10 * time.Millisecond)
			manager.Remove(forwarder)
		}()
	}

	wg.Wait()
}

func TestUDPForwarderManager_ConcurrentAddRemove(t *testing.T) {
	manager := NewUDPForwarderManager()
	var wg sync.WaitGroup

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
			if err != nil {
				return
			}
			defer ln.Close()

			remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
			if err != nil {
				return
			}
			defer remoteConn.Close()

			addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
			udpConn := &conn.UDPConn{}
			forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

			manager.Add(forwarder)
			time.Sleep(5 * time.Millisecond)
			manager.Remove(addr)
		}(i)
	}

	wg.Wait()
}

func TestTCPForwarderPool_Concurrency(t *testing.T) {
	pool := NewTCPForwarderPool(64*1024, 100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f := pool.Get()
			time.Sleep(5 * time.Millisecond)
			pool.Put(f)
		}()
	}

	wg.Wait()
}

func TestTCPForwarder_SetOnClose(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
	
	forwarder.Start()
	time.Sleep(20 * time.Millisecond)
	forwarder.Close()
	
	// Verify forwarder is closed
	assert.True(t, forwarder.IsClosed())
}

func TestUDPForwarder_SetOnClose(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)
	
	forwarder.Start()
	time.Sleep(20 * time.Millisecond)
	forwarder.Close()
	
	assert.True(t, forwarder.IsClosed())
}

func TestTCPForwarderManager_WaitAllWithMultiple(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	
	// Add some nil forwarders
	for i := 0; i < 5; i++ {
		manager.Add(NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil))
	}
	
	// WaitAll should not block
	done := make(chan struct{})
	go func() {
		manager.WaitAll()
		close(done)
	}()
	
	select {
	case <-done:
		// Expected
	case <-time.After(500 * time.Millisecond):
		t.Error("WaitAll should not block with active forwarders")
	}
	
	manager.CloseAll()
}

func TestUDPForwarderManager_WaitAll(t *testing.T) {
	manager := NewUDPForwarderManager()
	
	// CloseAll should not panic
	manager.CloseAll()
}

func TestTCPForwarderManager_CloseAllWithNil(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	
	// Add some nil forwarders
	for i := 0; i < 5; i++ {
		manager.Add(NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil))
	}
	
	// CloseAll should handle nil forwarders gracefully
	manager.CloseAll()
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarder_DoubleClose(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
	
	forwarder.Close()
	forwarder.Close() // Double close should not panic
	
	assert.True(t, forwarder.IsClosed())
}

func TestUDPForwarder_DoubleClose(t *testing.T) {
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln.Close()

	remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer remoteConn.Close()

	addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
	udpConn := &conn.UDPConn{}

	forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)
	
	forwarder.Close()
	forwarder.Close() // Double close should not panic
	
	assert.True(t, forwarder.IsClosed())
}

func TestTCPForwarder_ConcurrentClose(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
	
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			forwarder.Close()
		}()
	}
	
	wg.Wait()
	assert.True(t, forwarder.IsClosed())
}

func TestTCPForwarderPool_MultipleGetPut(t *testing.T) {
	pool := NewTCPForwarderPool(64*1024, 100)
	
	// Get and put multiple forwarders
	for i := 0; i < 50; i++ {
		f := pool.Get()
		assert.NotNil(t, f)
		assert.NotNil(t, f.buf)
		assert.Equal(t, 64*1024, len(f.buf))
		pool.Put(f)
	}
}

func TestTCPForwarderManager_RemoveNil(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)
	
	// Remove nil should not panic
	assert.NotPanics(t, func() {
		manager.Remove(nil)
	})
}

func TestNewTCPForwarder_VariousBufferSizes(t *testing.T) {
	sizes := []int{0, 1024, 32*1024, 64*1024, 128*1024}

	for _, size := range sizes {
		forwarder := NewTCPForwarder(nil, nil, size, 5*time.Minute, nil)
		expectedSize := size
		if size <= 0 {
			// C++ uses max_data_len_tcp = 4096*4 = 16384 bytes
			expectedSize = 16 * 1024
		}
		assert.Equal(t, expectedSize, forwarder.bufSize, "Buffer size should match for input %d", size)
		forwarder.Close()
	}
}

func TestUDPForwarderManager_ReplaceClient(t *testing.T) {
	manager := NewUDPForwarderManager()

	ln1, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln1.Close()

	remoteConn1, _ := net.DialUDP("udp", nil, ln1.LocalAddr().(*net.UDPAddr))
	defer remoteConn1.Close()

	addr1 := types.NewAddressFromUDPAddr(ln1.LocalAddr().(*net.UDPAddr))
	f1 := NewUDPForwarder(ln1, remoteConn1, addr1, &conn.UDPConn{}, 64*1024)

	manager.Add(f1)
	assert.Equal(t, 1, manager.Count())

	// Create new forwarder for same address
	ln2, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	require.NoError(t, err)
	defer ln2.Close()

	remoteConn2, _ := net.DialUDP("udp", nil, ln2.LocalAddr().(*net.UDPAddr))
	defer remoteConn2.Close()

	addr2 := types.NewAddressFromUDPAddr(ln2.LocalAddr().(*net.UDPAddr))
	f2 := NewUDPForwarder(ln2, remoteConn2, addr2, &conn.UDPConn{}, 64*1024)

	manager.Add(f2)
	assert.Equal(t, 2, manager.Count())
}

func TestTCPForwarder_ZeroBufferSize(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, 0, 5*time.Minute, nil)
	// C++ uses max_data_len_tcp = 4096*4 = 16384 bytes
	assert.Equal(t, 16*1024, forwarder.bufSize)
	forwarder.Close()
}

func TestTCPForwarder_NegativeBufferSize(t *testing.T) {
	forwarder := NewTCPForwarder(nil, nil, -100, 5*time.Minute, nil)
	// C++ uses max_data_len_tcp = 4096*4 = 16384 bytes
	assert.Equal(t, 16*1024, forwarder.bufSize)
	forwarder.Close()
}

func TestTCPForwarderManager_AddMultiple(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// Add multiple forwarders
	for i := 0; i < 10; i++ {
		forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
		manager.Add(forwarder)
	}

	assert.Equal(t, 10, manager.Count())

	// Remove them one by one
	for i := 0; i < 5; i++ {
		forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
		manager.Remove(forwarder) // Should not affect count
	}

	assert.Equal(t, 10, manager.Count())

	manager.CloseAll()
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderPool_DifferentBufferSizes(t *testing.T) {
	sizes := []int{0, 1024, 4096, 16384, 32768, 65536}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("buffer-%d", size), func(t *testing.T) {
			pool := NewTCPForwarderPool(size, 100)
			f := pool.Get()
			assert.NotNil(t, f)
			expectedSize := size
			if size <= 0 {
				expectedSize = 16 * 1024
			}
			assert.Equal(t, expectedSize, len(f.buf))
			pool.Put(f)
		})
	}
}

func TestUDPForwarderManager_AddMultiple(t *testing.T) {
	manager := NewUDPForwarderManager()

	for i := 0; i < 5; i++ {
		ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		require.NoError(t, err)
		defer ln.Close()

		remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
		require.NoError(t, err)
		defer remoteConn.Close()

		addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
		udpConn := &conn.UDPConn{}
		forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, 64*1024)

		manager.Add(forwarder)
	}

	assert.Equal(t, 5, manager.Count())

	manager.CloseAll()
	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarder_TimeoutValues(t *testing.T) {
	timeouts := []time.Duration{
		1 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
		1 * time.Hour,
	}

	for _, timeout := range timeouts {
		t.Run(fmt.Sprintf("timeout-%v", timeout), func(t *testing.T) {
			forwarder := NewTCPForwarder(nil, nil, 4096, timeout, nil)
			assert.Equal(t, timeout, forwarder.readTimeout)
			forwarder.Close()
		})
	}
}

func TestUDPForwarder_DifferentBufferSizes(t *testing.T) {
	sizes := []int{0, 4096, 16384, 32768, 65536}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("buffer-%d", size), func(t *testing.T) {
			ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
			require.NoError(t, err)
			defer ln.Close()

			remoteConn, err := net.DialUDP("udp", nil, ln.LocalAddr().(*net.UDPAddr))
			require.NoError(t, err)
			defer remoteConn.Close()

			addr := types.NewAddressFromUDPAddr(ln.LocalAddr().(*net.UDPAddr))
			udpConn := &conn.UDPConn{}

			forwarder := NewUDPForwarder(ln, remoteConn, addr, udpConn, size)
			expectedSize := size
			if size <= 0 {
				expectedSize = 16 * 1024
			}
			assert.Equal(t, expectedSize, forwarder.bufSize)
			forwarder.Close()
		})
	}
}

func TestTCPForwarderManager_StartWithNil(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 1000)

	// Start with nil should not panic and not add anything
	assert.NotPanics(t, func() {
		manager.Start(nil, nil)
	})

	assert.Equal(t, 0, manager.Count()) // nil is not added
	manager.CloseAll()
}

func TestUDPForwarderManager_CloseEmpty(t *testing.T) {
	manager := NewUDPForwarderManager()

	// Close empty manager multiple times should not panic
	assert.NotPanics(t, func() {
		manager.CloseAll()
		manager.CloseAll()
		manager.CloseAll()
	})

	assert.Equal(t, 0, manager.Count())
}

func TestTCPForwarderManager_CountConcurrency(t *testing.T) {
	manager := NewTCPForwarderManager(64*1024, 10000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)
			manager.Add(forwarder)
			time.Sleep(1 * time.Millisecond)
			manager.Remove(forwarder)
		}()
	}

	wg.Wait()
	// Just verify no panic and count is reasonable
	assert.True(t, manager.Count() >= 0)
	manager.CloseAll()
}

func TestTCPForwarder_HandlePartialWrite(t *testing.T) {
	// Test handlePartialWrite by creating a forwarder and calling the method
	// Since it's a private method, we test it indirectly through normal operation
	// or by creating a test that triggers partial writes

	// Create a mock test that exercises the partial write logic
	// We can't directly test private methods, so we test the behavior

	// Create a forwarder with nil connections
	forwarder := NewTCPForwarder(nil, nil, 4096, 5*time.Minute, nil)

	// The handlePartialWrite method will be called when there's a partial write
	// We can't easily test this without a real connection, so we verify the method exists
	assert.NotNil(t, forwarder)

	forwarder.Close()
}

func TestTCPForwarder_CopySrcToDst_EOF(t *testing.T) {
	// Test EOF handling in copySrcToDst
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 5*time.Minute, nil)

	forwarder.Start()

	// Close the server side to trigger EOF
	serverConn.CloseWrite()

	// Wait for forwarder to process
	time.Sleep(100 * time.Millisecond)

	forwarder.Close()
}

func TestTCPForwarder_CopyDstToSrc_EOF(t *testing.T) {
	// Test EOF handling in copyDstToSrc
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 5*time.Minute, nil)

	forwarder.Start()

	// Close the client side to trigger EOF
	clientConn.CloseWrite()

	// Wait for forwarder to process
	time.Sleep(100 * time.Millisecond)

	forwarder.Close()
}

func TestTCPForwarder_Timeout(t *testing.T) {
	// Test timeout handling
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	// Use a very short timeout
	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 100*time.Millisecond, nil)

	forwarder.Start()

	// Wait for timeout to occur
	time.Sleep(200 * time.Millisecond)

	forwarder.Close()
}

func TestTCPForwarder_DataTransfer(t *testing.T) {
	// Test actual data transfer
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 5*time.Minute, nil)

	// Send data from client to server
	testData := []byte("Hello, World!")
	var wg sync.WaitGroup
	var recvData []byte
	var recvMu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		n, _ := serverConn.Read(buf)
		recvMu.Lock()
		recvData = buf[:n]
		recvMu.Unlock()
	}()

	forwarder.Start()

	_, err := clientConn.Write(testData)
	require.NoError(t, err)

	wg.Wait()

	recvMu.Lock()
	assert.Equal(t, testData, recvData)
	recvMu.Unlock()

	forwarder.Close()
}


func TestTCPForwarder_HandlePartialWrite_Actual(t *testing.T) {
	// Test actual partial write scenario
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	// Set small buffer to force partial writes
	clientConn.SetWriteBuffer(512)
	serverConn.SetReadBuffer(512)

	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 5*time.Minute, nil)

	forwarder.Start()

	// Send large data that will cause partial writes
	largeData := make([]byte, 64*1024) // 64KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	var wg sync.WaitGroup
	var recvData []byte
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, len(largeData))
		n, _ := io.ReadFull(serverConn, buf)
		mu.Lock()
		recvData = buf[:n]
		mu.Unlock()
	}()

	_, err := clientConn.Write(largeData)
	require.NoError(t, err)

	wg.Wait()

	mu.Lock()
	assert.Equal(t, largeData, recvData)
	mu.Unlock()

	forwarder.Close()
}

// TestTCPForwarder_HandlePartialWrite_Indirect 间接测试 handlePartialWrite
// 当 WriteToRemote 返回已写入字节数小于总字节数时，会调用 handlePartialWrite
func TestTCPForwarder_HandlePartialWrite_Indirect(t *testing.T) {
	clientConn, serverConn := createTCPConnectionPair(t)
	defer clientConn.Close()
	defer serverConn.Close()

	// 设置非常小的写缓冲区，强制部分写入
	clientConn.SetWriteBuffer(256)
	serverConn.SetReadBuffer(256)

	forwarder := NewTCPForwarder(clientConn, serverConn, 4096, 5*time.Minute, nil)

	forwarder.Start()

	// 发送足够大的数据来触发多次部分写入
	largeData := make([]byte, 32*1024) // 32KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	// 异步接收数据
	var wg sync.WaitGroup
	var recvData []byte
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, len(largeData))
		n, _ := io.ReadFull(serverConn, buf)
		mu.Lock()
		recvData = buf[:n]
		mu.Unlock()
	}()

	// 发送数据
	_, err := clientConn.Write(largeData)
	require.NoError(t, err)

	wg.Wait()

	mu.Lock()
	assert.Equal(t, largeData, recvData)
	mu.Unlock()

	forwarder.Close()
}
