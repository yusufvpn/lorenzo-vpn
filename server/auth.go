package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// Default server secret key for Lorenzo VPN authentication
var serverSecret = getEnv("LORENZO_SECRET", "LorenzoStrictLeaderSecret2026")

// App Version Configuration (Controls Force Update)
const (
	CurrentServerVersion = "1.0.0"
	MinimumClientVersion = "1.0.0" // Clients below this version are BLOCKED from connecting!
	UpdateDownloadURL    = "https://github.com/yusufvpn/lorenzo-vpn/releases/latest"
)

type VersionInfo struct {
	AppName          string `json:"app_name"`
	ServerVersion    string `json:"server_version"`
	MinClientVersion string `json:"min_client_version"`
	ForceUpdate      bool   `json:"force_update"`
	DownloadURL      string `json:"download_url"`
	Changelog        string `json:"changelog"`
}

func getVersionInfo(clientVersion string) VersionInfo {
	needsUpdate := isVersionOutdated(clientVersion, MinimumClientVersion)
	return VersionInfo{
		AppName:          "Lorenzo VPN",
		ServerVersion:    CurrentServerVersion,
		MinClientVersion: MinimumClientVersion,
		ForceUpdate:      needsUpdate,
		DownloadURL:      UpdateDownloadURL,
		Changelog:        "تحديث أمني إجباري: تحسين سرعة النفق المشفر وفك حظر الألعاب.",
	}
}

// Compare semantic version (e.g. "1.0.0" vs "1.0.1")
func isVersionOutdated(clientVer, minVer string) bool {
	if clientVer == "" {
		return true
	}
	cParts := strings.Split(clientVer, ".")
	mParts := strings.Split(minVer, ".")
	for i := 0; i < len(cParts) && i < len(mParts); i++ {
		var cVal, mVal int
		fmt.Sscanf(cParts[i], "%d", &cVal)
		fmt.Sscanf(mParts[i], "%d", &mVal)
		if cVal < mVal {
			return true
		}
		if cVal > mVal {
			return false
		}
	}
	return len(cParts) < len(mParts)
}

// Verify client authentication token
func verifyClientToken(token string) bool {
	if token == "" {
		return false
	}
	// Direct token comparison or HMAC validation
	expectedHash := generateTokenHash(serverSecret)
	return token == serverSecret || token == expectedHash
}

func generateTokenHash(secret string) string {
	h := hmac.New(sha256.New, []byte("LORENZO_SALT"))
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum(nil))
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
