package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/dentaku7/mcp-go-sdk"
)

// StdioTransport implements Transport using standard input/output
type StdioTransport struct {
	scanner *bufio.Scanner
	encoder *json.Encoder
	closed  bool
	mu      sync.Mutex
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() mcp.Transport {
	scanner := bufio.NewScanner(os.Stdin)
	// Set a larger buffer size to handle large messages
	const maxCapacity = 1024 * 1024 // 1MB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	return &StdioTransport{
		scanner: scanner,
		encoder: json.NewEncoder(os.Stdout),
		closed:  false,
	}
}

// Send sends a message through stdout
func (t *StdioTransport) Send(data interface{}) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return fmt.Errorf("transport closed")
	}

	return t.encoder.Encode(data)
}

// Receive reads a message from stdin
func (t *StdioTransport) Receive() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil, fmt.Errorf("transport closed")
	}

	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	return t.scanner.Bytes(), nil
}

// Close closes the transport
func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.closed = true
	return nil
}
