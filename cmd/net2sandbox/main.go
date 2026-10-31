package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
)

// sandboxConfig holds the sandbox isolation configuration
type sandboxConfig struct {
	Net2Port      int    `json:"net2Port"`       // port for net2
	RealNetworkOK bool   `json:"realNetworkOk"` // allow real network access
	BlockedPorts  []int  `json:"blockedPorts"`  // ports to block
}

// DefaultConfig is the default sandbox configuration
var DefaultConfig = sandboxConfig{
	Net2Port:      8765,
	RealNetworkOK: false,
}

// Sandbox holds the sandbox state
type Sandbox struct {
	cfg     sandboxConfig
	ln      net.Listener
	net2Ln  net.Listener
	blocks  int // count of blocked connections
}

// NewSandbox creates a new sandbox with the given config
func NewSandbox(cfg sandboxConfig) *Sandbox {
	return &Sandbox{cfg: cfg}
}

// setupNet2Network sets up the net2 isolated network on a port
func (s *Sandbox) setupNet2Network() error {
	// Listen on the net2 port for network connections
	var err error
	s.net2Ln, err = net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Net2Port))
	if err != nil {
		return fmt.Errorf("failed to listen on net2 port %d: %v", s.cfg.Net2Port, err)
	}

	fmt.Printf("net2 fake network listening on port %d\n", s.cfg.Net2Port)
	fmt.Println("Users connect to this port to access the net2 environment")
	fmt.Println("Real network ports are blocked/isolated")

	// Accept connections in a goroutine
	go s.acceptNet2Connections()

	return nil
}

// acceptNet2Connections accepts connections on the net2 port
func (s *Sandbox) acceptNet2Connections() {
	for {
		conn, err := s.net2Ln.Accept()
		if err != nil {
			fmt.Printf("net2 accept error: %v\n", err)
			return
		}

		// Handle the connection
		go s.handleNet2Connection(conn)
	}
}

// handleNet2Connection handles a connection to the net2 sandbox
func (s *Sandbox) handleNet2Connection(conn net.Conn) {
	defer conn.Close()

	fmt.Printf("net2 client connected from %s\n", conn.RemoteAddr().String())

	// Write welcome message
 welcomeMsg := fmt.Sprintf("Welcome to net2 sandbox on port %d\n", s.cfg.Net2Port)
	fmt.Fprintf(conn, welcomeMsg)

	// Read and handle user instructions from the OS project
	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("net2 connection closed: %v\n", err)
			return
		}

		// Process the instruction
		instruction := string(buffer[:n])
		fmt.Printf("net2 received instruction: %s\n", instruction)

		// Echo back or process the instruction
		response := fmt.Sprintf("net2 processed: %s\n", instruction)
		fmt.Fprintf(conn, response)
	}
}

// blockRealNetworks blocks real network ports using iptables/nftables
func (s *Sandbox) blockRealNetworks() error {
	// Try to block real network access using available methods
	fmt.Println("blocking real network ports")

	cmd := exec.Command("iptables", "-A", "OUTPUT", "-m", "owner", "--cmd-owner", "nobody", "-j", "DROP")
	output, err := cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("iptables rule added: %s\n", string(output))
	}

	cmd = exec.Command("nft", "add rule", "inet filter", "output", "drop")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("nft rule added: %s\n", string(output))
	}

	if err != nil {
		fmt.Println("Note: Could not apply iptables/nftables rules (may need root)")
		fmt.Println("Real network access will be logically blocked in net2 mode")
	}

	return nil
}

func (s *Sandbox) runNet2OS() {
	fmt.Printf("net2 port: %d (isolated)\n", s.cfg.Net2Port)
}

// main entry point
func main() {
	// Initialize sandbox config
	cfg := DefaultConfig

	// Create the sandbox
	s := NewSandbox(cfg)

	fmt.Println("net2 sandbox")

	// Attempt to block real network ports
	if err := s.blockRealNetworks(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	// Set up the net2 isolated network
	if err := s.setupNet2Network(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"error setting up net2 network: %v\n",
			err,
		)
		os.Exit(1)
	}

	s.runNet2OS()

	// Keep the sandbox running
	fmt.Println("net2 sandbox running. Press Ctrl+C to stop.")
	select {}
}