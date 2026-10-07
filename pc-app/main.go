package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed ui/*
var embeddedUI embed.FS

const (
	ClientVersion     = "1.0.0"
	DefaultServerHost = "lorenzo-vpn-production.up.railway.app"
	DefaultSecret     = "LorenzoStrictLeaderSecret2026"
	HTTPProxyPort     = "10809"
	SOCKSProxyPort    = "10808"
	UIPort            = "10807"
)

type AppState struct {
	mu           sync.Mutex
	isConnected  bool
	proxyServer  *ProxyServer
	tunnelClient *LorenzoTunnelClient
	bytesRx      int64
	bytesTx      int64
}

var state AppState

func main() {
	// Cleanup hook on process termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[Lorenzo] Shutting down, restoring Windows system proxy...")
		_ = disableWindowsProxy()
		os.Exit(0)
	}()

	// 1. Setup local UI and API server
	mux := http.NewServeMux()

	// Embedded Static UI assets
	uiSub, err := fs.Sub(embeddedUI, "ui")
	if err != nil {
		log.Fatalf("Failed to load embedded UI: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(uiSub)))

	// API: Check Force Update
	mux.HandleFunc("/api/check-update", handleCheckUpdate)

	// API: Connect
	mux.HandleFunc("/api/connect", handleConnect)

	// API: Disconnect
	mux.HandleFunc("/api/disconnect", handleDisconnect)

	// API: Poll Stats
	mux.HandleFunc("/api/poll-stats", handlePollStats)

	// API: Heartbeat & Exit
	mux.HandleFunc("/api/heartbeat", handleHeartbeat)
	mux.HandleFunc("/api/exit", handleExit)

	uiAddr := "127.0.0.1:" + UIPort
	listener, err := net.Listen("tcp", uiAddr)
	if err != nil {
		log.Fatalf("Failed to bind UI server on %s: %v", uiAddr, err)
	}

	go func() {
		_ = http.Serve(listener, mux)
	}()

	log.Printf("==================================================")
	log.Printf("👑 Lorenzo VPN Desktop GUI starting on %s", uiAddr)
	log.Printf("==================================================")

	// 2. Launch Native Desktop App Window via Edge App Mode
	launchDesktopApp("http://" + uiAddr)
}

func handleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	apiURL := fmt.Sprintf("https://%s/api/version?v=%s", DefaultServerHost, ClientVersion)
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"force_update": false,
		})
		return
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"force_update": false})
		return
	}
	json.NewEncoder(w).Encode(data)
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.isConnected {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "already_connected": true})
		return
	}

	tunnelURL := fmt.Sprintf("wss://%s/lorenzo-tunnel", DefaultServerHost)
	state.tunnelClient = NewLorenzoTunnelClient(tunnelURL, DefaultSecret, ClientVersion)

	httpAddr := "127.0.0.1:" + HTTPProxyPort
	socksAddr := "127.0.0.1:" + SOCKSProxyPort
	state.proxyServer = NewProxyServer(state.tunnelClient)

	err := state.proxyServer.Start(httpAddr, socksAddr)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	// Set Windows System Proxy
	_ = setWindowsProxy(httpAddr)
	state.isConnected = true

	log.Printf("[VPN] Connected successfully! Proxy: %s", httpAddr)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func handleDisconnect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.proxyServer != nil {
		state.proxyServer.Stop()
		state.proxyServer = nil
	}
	_ = disableWindowsProxy()
	state.isConnected = false

	log.Println("[VPN] Disconnected and proxy restored.")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func handlePollStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rx := int64(0)
	tx := int64(0)
	if state.proxyServer != nil {
		rx = atomic.LoadInt64(&state.proxyServer.bytesRx)
		tx = atomic.LoadInt64(&state.proxyServer.bytesTx)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"connected": state.isConnected,
		"bytes_rx":  rx,
		"bytes_tx":  tx,
		"ping":      102,
	})
}

var (
	lastHeartbeat int64
	hasConnectedUI int32
)

func handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	atomic.StoreInt64(&lastHeartbeat, time.Now().Unix())
	atomic.StoreInt32(&hasConnectedUI, 1)
	w.WriteHeader(http.StatusOK)
}

func handleExit(w http.ResponseWriter, r *http.Request) {
	log.Println("[Lorenzo] Exit requested by UI.")
	_ = disableWindowsProxy()
	os.Exit(0)
}

func launchDesktopApp(url string) {
	atomic.StoreInt64(&lastHeartbeat, time.Now().Unix())

	edgePaths := []string{
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}

	var targetEdge string
	for _, p := range edgePaths {
		if _, err := os.Stat(p); err == nil {
			targetEdge = p
			break
		}
	}

	tempDir := filepath.Join(os.TempDir(), "LorenzoVPN_DesktopProfile")

	if targetEdge != "" {
		cmd := exec.Command(targetEdge,
			"--app="+url,
			"--window-size=440,730",
			"--user-data-dir="+tempDir,
			"--proxy-bypass-list=127.0.0.1;localhost;<local>",
			"--no-first-run",
			"--no-default-browser-check",
		)
		err := cmd.Start()
		if err != nil {
			log.Printf("Failed to start Edge window: %v, falling back...", err)
			_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		}
	} else {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}

	// Watchdog loop: keeps process alive and monitors UI heartbeat
	go func() {
		// Wait 15 seconds grace period for UI to load
		time.Sleep(15 * time.Second)
		for {
			time.Sleep(2 * time.Second)
			if atomic.LoadInt32(&hasConnectedUI) == 1 {
				last := atomic.LoadInt64(&lastHeartbeat)
				if time.Now().Unix()-last > 6 {
					log.Println("[Lorenzo] UI window closed (heartbeat lost), shutting down cleanly...")
					_ = disableWindowsProxy()
					os.Exit(0)
				}
			}
		}
	}()

	// Keep main thread alive
	select {}
}
