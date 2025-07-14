package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockBackend creates a simple HTTP test server that replies with its port.
func createMockBackend(t *testing.T) (*httptest.Server, int) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The address is 127.0.0.1:PORT, we extract the port
		port := strings.Split(r.Host, ":")[1]
		fmt.Fprint(w, port)
	}))

	// Get the port
	addr, ok := server.Listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return server, addr.Port
}

func TestTCPProxy_Forwarding(t *testing.T) {
	backend, backendPort := createMockBackend(t)
	defer backend.Close()

	proxy := NewTCPProxy(backend.Listener.Addr().String())

	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		proxy.listener = proxyListener
		proxy.acceptLoop()
	}()
	defer proxyListener.Close()

	proxyAddr := proxyListener.Addr().String()

	// Connect to the proxy
	conn, err := net.Dial("tcp", proxyAddr)
	require.NoError(t, err)
	defer conn.Close()

	// Send a dummy HTTP request
	_, err = conn.Write([]byte("GET / HTTP/1.0\r\nHost: " + backend.Listener.Addr().String() + "\r\n\r\n"))
	require.NoError(t, err)

	// Read the response
	response, err := io.ReadAll(conn)
	require.NoError(t, err)

	// The response should contain the port number from the backend
	assert.Contains(t, string(response), strconv.Itoa(backendPort))
}

func TestManager_Switching(t *testing.T) {
	// Backend 1 (initial)
	backend1, port1 := createMockBackend(t)
	defer backend1.Close()

	// Backend 2 (target for switch)
	backend2, port2 := createMockBackend(t)
	defer backend2.Close()

	// Create a temporary state file
	tmpDir, err := os.MkdirTemp("", "proxy-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)
	stateFile := filepath.Join(tmpDir, "active_port")

	// Start the manager
	proxyListenPort := 8999 // A fixed port for testing the proxy listener
	manager := NewManager(proxyListenPort, port1, stateFile)
	go func() {
		err := manager.Start()
		// We expect an error when the watcher is closed, so we can ignore it in the test context.
		if err != nil && !strings.Contains(err.Error(), "bad file descriptor") {
			assert.NoError(t, err)
		}
	}()

	// Give the manager a moment to start up
	time.Sleep(200 * time.Millisecond)

	// --- Test 1: Forwarding to initial backend ---
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d", proxyListenPort))
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, strconv.Itoa(port1), string(body), "Should initially proxy to backend 1")

	// --- Test 2: Switch target by writing to state file ---
	err = os.WriteFile(stateFile, []byte(strconv.Itoa(port2)), 0644)
	require.NoError(t, err)

	// Give the file watcher a moment to detect the change and switch the proxy
	time.Sleep(200 * time.Millisecond)

	// --- Test 3: Forwarding to the new backend ---
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d", proxyListenPort))
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, strconv.Itoa(port2), string(body), "Should proxy to backend 2 after switch")
}
