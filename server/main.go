package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// 1. Static Web Dashboard (Camouflage & Admin UI)
	fs := http.FileServer(http.Dir("./web"))
	mux.Handle("/", fs)

	// 2. Encrypted VPN Tunnel Endpoint
	mux.HandleFunc("/lorenzo-tunnel", handleTunnel)

	// 3. Mandatory Update / Version Verification API
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		clientVer := r.URL.Query().Get("v")
		info := getVersionInfo(clientVer)
		json.NewEncoder(w).Encode(info)
	})

	// 4. Server Health Check API
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "online",
			"server_name": "Lorenzo VPN Core",
			"region":      "Europe (Amsterdam)",
			"timestamp":   time.Now().Unix(),
		})
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  0, // Unlimited for persistent tunnel connections
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("==================================================")
	log.Printf("👑 Lorenzo VPN Server starting on port %s", port)
	log.Printf("🔒 Secret Key: %s...", serverSecret[:min(len(serverSecret), 6)])
	log.Printf("🚀 Protocol: Lorenzo Encrypted Tunnel v1.0")
	log.Printf("==================================================")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
