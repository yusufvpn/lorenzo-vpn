package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type LorenzoTunnelClient struct {
	ServerURL string
	AuthToken string
	ClientVer string
	dialer    *websocket.Dialer
}

func NewLorenzoTunnelClient(serverURL, authToken, clientVer string) *LorenzoTunnelClient {
	return &LorenzoTunnelClient{
		ServerURL: serverURL,
		AuthToken: authToken,
		ClientVer: clientVer,
		dialer: &websocket.Dialer{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			HandshakeTimeout: 10 * time.Second,
		},
	}
}

// DialViaTunnel establishes an encrypted tunnel connection to target (host:port)
func (c *LorenzoTunnelClient) DialViaTunnel(network, host string, port uint16) (net.Conn, error) {
	u, err := url.Parse(c.ServerURL)
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	header.Set("User-Agent", "LorenzoVPN-PC/"+c.ClientVer)

	ws, resp, err := c.dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("handshake failed with HTTP %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	// Prepare Handshake binary payload:
	// [1 byte: Version=1]
	// [1 byte: NetType (1=TCP, 2=UDP)]
	// [2 bytes: Port (BigEndian)]
	// [1 byte: HostLen]
	// [Host bytes]
	// [1 byte: ClientVerLen]
	// [ClientVer bytes]
	// [AuthToken bytes]
	netType := byte(1)
	if network == "udp" {
		netType = byte(2)
	}

	hostBytes := []byte(host)
	verBytes := []byte(c.ClientVer)
	tokenBytes := []byte(c.AuthToken)

	payload := make([]byte, 5+len(hostBytes)+1+len(verBytes)+len(tokenBytes))
	payload[0] = 1 // Proto version
	payload[1] = netType
	binary.BigEndian.PutUint16(payload[2:4], port)
	payload[4] = byte(len(hostBytes))
	copy(payload[5:], hostBytes)

	offset := 5 + len(hostBytes)
	payload[offset] = byte(len(verBytes))
	offset++
	copy(payload[offset:], verBytes)
	offset += len(verBytes)

	copy(payload[offset:], tokenBytes)

	// Send Handshake
	if err := ws.WriteMessage(websocket.BinaryMessage, payload); err != nil {
		ws.Close()
		return nil, fmt.Errorf("failed to send handshake: %w", err)
	}

	// Read 1-byte handshake response
	ws.SetReadDeadline(time.Now().Add(8 * time.Second))
	msgType, res, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil || msgType != websocket.BinaryMessage || len(res) == 0 {
		ws.Close()
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}

	switch res[0] {
	case 0x01: // Success!
		return newWSConnWrapper(ws), nil
	case 0x00:
		ws.Close()
		return nil, fmt.Errorf("authentication failed: invalid Lorenzo secret token")
	case 0x02:
		ws.Close()
		return nil, fmt.Errorf("target dial failed: server could not connect to %s:%d", host, port)
	case 0x03:
		ws.Close()
		return nil, fmt.Errorf("UPDATE_REQUIRED: client version %s is outdated! Server enforced mandatory update", c.ClientVer)
	default:
		ws.Close()
		return nil, fmt.Errorf("unknown handshake response: 0x%02x", res[0])
	}
}

// WSConnWrapper adapts a websocket.Conn to a standard net.Conn
type WSConnWrapper struct {
	ws       *websocket.Conn
	reader   io.Reader
	readMu   sync.Mutex
	writeMu  sync.Mutex
	closed   bool
	closeMu  sync.Mutex
}

func newWSConnWrapper(ws *websocket.Conn) *WSConnWrapper {
	return &WSConnWrapper{ws: ws}
}

func (c *WSConnWrapper) Read(b []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for {
		if c.reader != nil {
			n, err := c.reader.Read(b)
			if err == io.EOF {
				c.reader = nil
				if n > 0 {
					return n, nil
				}
				continue
			}
			return n, err
		}

		mType, data, err := c.ws.ReadMessage()
		if err != nil {
			return 0, err
		}
		if mType == websocket.BinaryMessage {
			c.reader = bytes.NewReader(data)
			n, err := c.reader.Read(b)
			if err == io.EOF {
				c.reader = nil
			}
			return n, nil
		}
	}
}

func (c *WSConnWrapper) Write(b []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	err := c.ws.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *WSConnWrapper) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.ws.Close()
}

func (c *WSConnWrapper) LocalAddr() net.Addr                { return c.ws.LocalAddr() }
func (c *WSConnWrapper) RemoteAddr() net.Addr               { return c.ws.RemoteAddr() }
func (c *WSConnWrapper) SetDeadline(t time.Time) error      { return nil }
func (c *WSConnWrapper) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *WSConnWrapper) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }
