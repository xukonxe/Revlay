package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/xukonxe/revlay/internal/color"
)

// Manager handles the lifecycle of the proxy and watches for changes.
type Manager struct {
	listenAddr  string
	stateFile   string
	proxy       *TCPProxy
	initialPort int
}

// NewManager creates a new proxy manager.
func NewManager(listenPort, initialPort int, stateFile string) *Manager {
	return &Manager{
		listenAddr:  fmt.Sprintf(":%d", listenPort),
		stateFile:   stateFile,
		initialPort: initialPort,
	}
}

// Start runs the proxy and begins watching the state file for changes.
func (m *Manager) Start() error {
	// Ensure state directory exists
	if err := os.MkdirAll(filepath.Dir(m.stateFile), 0755); err != nil {
		return fmt.Errorf("could not create state directory: %w", err)
	}

	// Read initial target port from state file, or use default
	targetPort, err := m.readStateFile()
	if err != nil {
		log.Print(color.Yellow(fmt.Sprintf("Could not read state file, falling back to initial port %d. Error: %v", m.initialPort, err)))
		targetPort = m.initialPort
		// Write initial state
		if err := m.writeStateFile(targetPort); err != nil {
			log.Print(color.Red(fmt.Sprintf("Failed to write initial state file: %v", err)))
		}
	}

	m.proxy = NewTCPProxy(fmt.Sprintf("127.0.0.1:%d", targetPort))
	if err := m.proxy.Start(m.listenAddr); err != nil {
		return fmt.Errorf("could not start proxy: %w", err)
	}
	log.Print(color.Green(fmt.Sprintf("Proxy listening on %s, forwarding to 127.0.0.1:%d", m.listenAddr, targetPort)))

	return m.watchStateFile()
}

func (m *Manager) watchStateFile() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	defer watcher.Close()

	// Watch the directory containing the state file
	watchDir := filepath.Dir(m.stateFile)
	if err := watcher.Add(watchDir); err != nil {
		return fmt.Errorf("failed to watch state directory '%s': %w", watchDir, err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// We only care about writes to our specific state file
			if event.Name == m.stateFile && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				log.Println(color.Cyan("State file changed, attempting to switch proxy target..."))
				newPort, err := m.readStateFile()
				if err != nil {
					log.Print(color.Red(fmt.Sprintf("Error reading state file on change: %v", err)))
					continue
				}
				m.proxy.SwitchTarget(fmt.Sprintf("127.0.0.1:%d", newPort))
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Print(color.Red(fmt.Sprintf("File watcher error: %v", err)))
		}
	}
}

func (m *Manager) readStateFile() (int, error) {
	content, err := os.ReadFile(m.stateFile)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0, fmt.Errorf("state file content is not a valid port: %w", err)
	}
	return port, nil
}

func (m *Manager) writeStateFile(port int) error {
	return os.WriteFile(m.stateFile, []byte(strconv.Itoa(port)), 0644)
}

// TCPProxy is a thread-safe TCP proxy.
type TCPProxy struct {
	listener   net.Listener
	targetAddr string
	mu         sync.RWMutex
}

// NewTCPProxy creates a new TCPProxy.
func NewTCPProxy(initialTarget string) *TCPProxy {
	return &TCPProxy{
		targetAddr: initialTarget,
	}
}

// Start initializes the proxy listener and starts accepting connections.
func (p *TCPProxy) Start(listenAddr string) error {
	var err error
	p.listener, err = net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	go p.acceptLoop()
	return nil
}

func (p *TCPProxy) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			// TODO: Better error handling for closed listeners.
			log.Print(color.Red(fmt.Sprintf("Failed to accept connection: %v", err)))
			time.Sleep(1 * time.Second)
			continue
		}
		go p.handleConnection(conn)
	}
}

// SwitchTarget safely changes the proxy's target address.
func (p *TCPProxy) SwitchTarget(newTargetAddr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.targetAddr != newTargetAddr {
		log.Print(color.Green(fmt.Sprintf("Proxy switching target from %s to %s", p.targetAddr, newTargetAddr)))
		p.targetAddr = newTargetAddr
	}
}

func (p *TCPProxy) getTargetAddr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.targetAddr
}

func (p *TCPProxy) handleConnection(conn net.Conn) {
	defer conn.Close()
	targetAddr := p.getTargetAddr()

	dialer := net.Dialer{Timeout: 5 * time.Second}
	targetConn, err := dialer.Dial("tcp", targetAddr)
	if err != nil {
		log.Print(color.Red(fmt.Sprintf("Failed to connect to target %s: %v", targetAddr, err)))
		return
	}
	defer targetConn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(targetConn, conn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(conn, targetConn)
	}()

	wg.Wait()
}
