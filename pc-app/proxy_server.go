package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
)

type ProxyServer struct {
	tunnelClient *LorenzoTunnelClient
	httpListener net.Listener
	sockListener net.Listener
	activeConns  int64
	bytesRx      int64
	bytesTx      int64
}

func NewProxyServer(tunnel *LorenzoTunnelClient) *ProxyServer {
	return &ProxyServer{
		tunnelClient: tunnel,
	}
}

func (s *ProxyServer) Start(httpAddr, socksAddr string) error {
	var err error
	s.httpListener, err = net.Listen("tcp", httpAddr)
	if err != nil {
		return fmt.Errorf("failed to bind HTTP proxy to %s: %w", httpAddr, err)
	}

	s.sockListener, err = net.Listen("tcp", socksAddr)
	if err != nil {
		s.httpListener.Close()
		return fmt.Errorf("failed to bind SOCKS5 proxy to %s: %w", socksAddr, err)
	}

	go s.runHTTP()
	go s.runSOCKS5()

	return nil
}

func (s *ProxyServer) Stop() {
	if s.httpListener != nil {
		s.httpListener.Close()
	}
	if s.sockListener != nil {
		s.sockListener.Close()
	}
}

// ------------------- HTTP/HTTPS CONNECT Proxy -------------------

func (s *ProxyServer) runHTTP() {
	for {
		conn, err := s.httpListener.Accept()
		if err != nil {
			return
		}
		go s.handleHTTPConn(conn)
	}
}

func (s *ProxyServer) handleHTTPConn(clientConn net.Conn) {
	defer clientConn.Close()
	atomic.AddInt64(&s.activeConns, 1)
	defer atomic.AddInt64(&s.activeConns, -1)

	br := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	var host string
	var port uint16 = 80

	if req.Method == http.MethodConnect {
		// HTTPS Tunnel: CONNECT host:port HTTP/1.1
		h, pStr, err := net.SplitHostPort(req.URL.Host)
		if err == nil {
			host = h
			pInt, _ := strconv.Atoi(pStr)
			port = uint16(pInt)
		} else {
			host = req.URL.Host
			port = 443
		}

		remoteConn, err := s.tunnelClient.DialViaTunnel("tcp", host, port)
		if err != nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
		defer remoteConn.Close()

		// Acknowledge CONNECT
		clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		s.pipe(clientConn, remoteConn)
	} else {
		// Plain HTTP Proxy
		h := req.URL.Hostname()
		pStr := req.URL.Port()
		if pStr != "" {
			pInt, _ := strconv.Atoi(pStr)
			port = uint16(pInt)
		} else if req.URL.Scheme == "https" {
			port = 443
		}
		host = h

		remoteConn, err := s.tunnelClient.DialViaTunnel("tcp", host, port)
		if err != nil {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
			return
		}
		defer remoteConn.Close()

		req.Write(remoteConn)
		s.pipe(clientConn, remoteConn)
	}
}

// ------------------- SOCKS5 Proxy -------------------

func (s *ProxyServer) runSOCKS5() {
	for {
		conn, err := s.sockListener.Accept()
		if err != nil {
			return
		}
		go s.handleSOCKSConn(conn)
	}
}

func (s *ProxyServer) handleSOCKSConn(clientConn net.Conn) {
	defer clientConn.Close()
	atomic.AddInt64(&s.activeConns, 1)
	defer atomic.AddInt64(&s.activeConns, -1)

	buf := make([]byte, 260)
	// 1. SOCKS5 Greeting: [VER, NMETHODS, METHODS...]
	if _, err := io.ReadFull(clientConn, buf[:2]); err != nil || buf[0] != 0x05 {
		return
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(clientConn, buf[:nMethods]); err != nil {
		return
	}

	// Reply NO_AUTH (0x00)
	clientConn.Write([]byte{0x05, 0x00})

	// 2. Request: [VER, CMD, RSV, ATYP, DST.ADDR, DST.PORT]
	if _, err := io.ReadFull(clientConn, buf[:4]); err != nil || buf[0] != 0x05 {
		return
	}

	if buf[1] != 0x01 { // Only TCP CONNECT is supported in SOCKS5 mode
		clientConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Command not supported
		return
	}

	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(clientConn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // Domain
		if _, err := io.ReadFull(clientConn, buf[:1]); err != nil {
			return
		}
		domainLen := int(buf[0])
		if _, err := io.ReadFull(clientConn, buf[:domainLen]); err != nil {
			return
		}
		host = string(buf[:domainLen])
	case 0x04: // IPv6
		if _, err := io.ReadFull(clientConn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		return
	}

	// Port
	if _, err := io.ReadFull(clientConn, buf[:2]); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(buf[:2])

	// Dial via encrypted tunnel
	remoteConn, err := s.tunnelClient.DialViaTunnel("tcp", host, port)
	if err != nil {
		clientConn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0}) // Connection refused
		return
	}
	defer remoteConn.Close()

	// Reply SUCCESS
	clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0x10, 0x80})

	s.pipe(clientConn, remoteConn)
}

// Bi-directional pipe
func (s *ProxyServer) pipe(c1, c2 net.Conn) {
	errCh := make(chan error, 2)

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := c1.Read(buf)
			if n > 0 {
				atomic.AddInt64(&s.bytesTx, int64(n))
				_, wErr := c2.Write(buf[:n])
				if wErr != nil {
					errCh <- wErr
					return
				}
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 32768)
		for {
			n, err := c2.Read(buf)
			if n > 0 {
				atomic.AddInt64(&s.bytesRx, int64(n))
				_, wErr := c1.Write(buf[:n])
				if wErr != nil {
					errCh <- wErr
					return
				}
			}
			if err != nil {
				errCh <- err
				return
			}
		}
	}()

	<-errCh
}
