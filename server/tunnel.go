package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  65536,
	WriteBufferSize: 65536,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all cross-origin requests
	},
}

/*
Tunnel Handshake Protocol (Binary Format):
[1 byte: Version = 1]
[1 byte: Network Type (1 = TCP, 2 = UDP)]
[2 bytes: Port (BigEndian)]
[1 byte: Host Length (N)]
[N bytes: Host (Domain or IP)]
[32 bytes: Auth Token / HMAC]
*/

func handleTunnel(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Tunnel] WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	// Read initial handshake packet
	messageType, payload, err := ws.ReadMessage()
	if err != nil || len(payload) < 5 {
		log.Printf("[Tunnel] Invalid handshake payload from %s", r.RemoteAddr)
		return
	}

	if messageType != websocket.BinaryMessage {
		log.Printf("[Tunnel] Expected binary handshake, got %d", messageType)
		return
	}

	protoVer := payload[0]
	if protoVer != 1 {
		log.Printf("[Tunnel] Unsupported protocol version: %d", protoVer)
		return
	}

	netType := payload[1]
	network := "tcp"
	if netType == 2 {
		network = "udp"
	}

	port := binary.BigEndian.Uint16(payload[2:4])
	hostLen := int(payload[4])
	if len(payload) < 5+hostLen {
		log.Printf("[Tunnel] Malformed host header")
		return
	}

	host := string(payload[5 : 5+hostLen])
	offset := 5 + hostLen

	clientVer := ""
	token := ""

	if len(payload) > offset {
		verLen := int(payload[offset])
		offset++
		if len(payload) >= offset+verLen {
			clientVer = string(payload[offset : offset+verLen])
			offset += verLen
		}
	}
	if len(payload) > offset {
		token = string(payload[offset:])
	}

	// 1. Enforce Mandatory Version Check (Server-Side Lockdown)
	if isVersionOutdated(clientVer, MinimumClientVersion) {
		log.Printf("[Security] Outdated client rejected (client ver: '%s', required: '%s') from %s", clientVer, MinimumClientVersion, r.RemoteAddr)
		ws.WriteMessage(websocket.BinaryMessage, []byte{0x03}) // 0x03 = Update Required
		return
	}

	// 2. Verify Auth Token
	if !verifyClientToken(token) {
		log.Printf("[Security] Unauthorized connection attempt from %s for %s:%d", r.RemoteAddr, host, port)
		ws.WriteMessage(websocket.BinaryMessage, []byte{0x00}) // 0x00 = Auth Failed
		return
	}

	targetAddr := fmt.Sprintf("%s:%d", host, port)

	// Dial target destination
	remoteConn, err := net.DialTimeout(network, targetAddr, 8*time.Second)
	if err != nil {
		log.Printf("[Tunnel] Failed to dial target %s (%s): %v", targetAddr, network, err)
		ws.WriteMessage(websocket.BinaryMessage, []byte{0x02}) // 0x02 = Dial Failed
		return
	}
	defer remoteConn.Close()

	// Send success ACK to client
	if err := ws.WriteMessage(websocket.BinaryMessage, []byte{0x01}); err != nil {
		return
	}

	// Bi-directional stream forwarding
	var wg sync.WaitGroup
	wg.Add(2)

	// 1. WebSocket -> Remote Target
	go func() {
		defer wg.Done()
		defer remoteConn.Close()
		for {
			mType, data, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if mType == websocket.BinaryMessage {
				_, err = remoteConn.Write(data)
				if err != nil {
					break
				}
			}
		}
	}()

	// 2. Remote Target -> WebSocket
	go func() {
		defer wg.Done()
		defer ws.Close()
		buf := make([]byte, 32768)
		for {
			n, err := remoteConn.Read(buf)
			if n > 0 {
				writeErr := ws.WriteMessage(websocket.BinaryMessage, buf[:n])
				if writeErr != nil {
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					// Connection ended
				}
				break
			}
		}
	}()

	wg.Wait()
}
