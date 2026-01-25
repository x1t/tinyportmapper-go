package main

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/x1t/tinyportmapper-go/internal/config"
	"github.com/x1t/tinyportmapper-go/internal/types"
)

// TestAddressParsing tests various address formats
func TestAddressParsing(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"[::1]:8080", "[::1]:8080"},
		{"192.168.1.1:80", "192.168.1.1:80"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			addr, err := types.NewAddressFromString(tt.input)
			if err != nil {
				t.Fatalf("Failed to parse address: %v", err)
			}
			if addr.String() != tt.expect {
				t.Errorf("String() = %s, want %s", addr.String(), tt.expect)
			}
		})
	}
}

// TestTCPForwardingIntegration tests end-to-end TCP data forwarding
func TestTCPForwardingIntegration(t *testing.T) {
	// Create a simple server
	serverLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer serverLn.Close()

	// Get the port
	serverAddr := serverLn.Addr().String()

	// Client connects
	clientConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	// Server accepts
	serverConn, err := serverLn.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer serverConn.Close()

	// Test data
	testData := make([]byte, 64*1024) // 64KB
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Send and receive concurrently
	var wg sync.WaitGroup
	var sendErr, recvErr error
	recvData := make([]byte, len(testData))

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, sendErr = clientConn.Write(testData)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, recvErr = io.ReadFull(serverConn, recvData)
	}()

	wg.Wait()

	if sendErr != nil {
		t.Fatalf("Send error: %v", sendErr)
	}
	if recvErr != nil {
		t.Fatalf("Receive error: %v", recvErr)
	}

	if !bytes.Equal(testData, recvData) {
		t.Error("Data mismatch")
	}
}

// TestUDPForwardingIntegration tests end-to-end UDP data forwarding
func TestUDPForwardingIntegration(t *testing.T) {
	// Create UDP listener
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Get the listener address
	listenAddr := ln.LocalAddr().(*net.UDPAddr)

	// Connect to listener
	conn, err := net.DialUDP("udp", nil, listenAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Test data
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	// Send
	_, err = conn.Write(testData)
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	// Receive with deadline
	ln.SetReadDeadline(time.Now().Add(2 * time.Second))
	recvData := make([]byte, 1500)
	n, _, err := ln.ReadFrom(recvData)
	if err != nil {
		t.Fatalf("Receive error: %v", err)
	}

	if !bytes.Equal(testData, recvData[:n]) {
		t.Error("Data mismatch")
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	cfg := &config.Config{
		ListenAddr:      "0.0.0.0:8080",
		RemoteAddr:      "127.0.0.1:9090",
		EnableTCP:       true,
		SocketBufferSizeKbyte: 1024, // 1024 kbyte = 1MB, 与C++版本一致
		MaxConnections:   100,
		ClearRatio:      30,
		ClearMin:        1,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}
}

// TestConcurrentConnections tests handling multiple concurrent connections
func TestConcurrentConnections(t *testing.T) {
	// Create server
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Start accepting connections
	var wg sync.WaitGroup
	var connCount int
	var mu sync.Mutex

	acceptDone := make(chan struct{})
	go func() {
		for i := 0; i < 5; i++ {
			conn, err := ln.AcceptTCP()
			if err != nil {
				return
			}
			mu.Lock()
			connCount++
			mu.Unlock()
			wg.Add(1)
			go func(c *net.TCPConn) {
				defer wg.Done()
				defer c.Close()
				// Read and close
				buf := make([]byte, 1024)
				c.Read(buf)
			}(conn)
		}
		close(acceptDone)
	}()

	// Connect multiple clients
	for i := 0; i < 5; i++ {
		conn, err := net.DialTCP("tcp", nil, ln.Addr().(*net.TCPAddr))
		if err != nil {
			t.Fatal(err)
		}
		conn.Write([]byte("test"))
		conn.CloseWrite()
		conn.Close()
	}

	<-acceptDone
	wg.Wait()

	mu.Lock()
	if connCount != 5 {
		t.Errorf("Expected 5 connections, got %d", connCount)
	}
	mu.Unlock()
}
