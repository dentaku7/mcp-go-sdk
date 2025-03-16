package transport

import (
	"encoding/json"
	"io"

	"mcp-go-sdk"
)

// BaseTransport provides common functionality for transports
type BaseTransport struct {
	reader  io.Reader
	writer  io.Writer
	closer  io.Closer
	config  *mcp.TransportConfig
	encoder *json.Encoder
	decoder *json.Decoder
}

// NewBaseTransport creates a new base transport
func NewBaseTransport(r io.Reader, w io.Writer, c io.Closer, config *mcp.TransportConfig) *BaseTransport {
	if config == nil {
		config = mcp.DefaultTransportConfig()
	}

	return &BaseTransport{
		reader:  r,
		writer:  w,
		closer:  c,
		config:  config,
		encoder: json.NewEncoder(w),
		decoder: json.NewDecoder(r),
	}
}

// Send implements Transport.Send
func (t *BaseTransport) Send(data interface{}) error {
	return t.encoder.Encode(data)
}

// Receive implements Transport.Receive
func (t *BaseTransport) Receive() ([]byte, error) {
	var raw json.RawMessage
	if err := t.decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Close implements Transport.Close
func (t *BaseTransport) Close() error {
	if t.closer != nil {
		return t.closer.Close()
	}
	return nil
}
