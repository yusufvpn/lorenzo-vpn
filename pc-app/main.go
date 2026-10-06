package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	ClientVersion     = "1.0.0"
	DefaultServerHost = "lorenzo-vpn-production.up.railway.app"
	DefaultSecret     = "LorenzoStrictLeaderSecret2026"
	HTTPProxyPort     = "10809"
	SOCKSProxyPort    = "10808"
)

type VersionResponse struct {
	AppName          string `json:"app_name"`
	ServerVersion    string `json:"server_version"`
	MinClientVersion string `json:"min_client_version"`
	ForceUpdate      bool   `json:"force_update"`
	DownloadURL      string `json:"download_url"`
	Changelog        string `json:"changelog"`
}

func checkForceUpdate(serverHost string) (*VersionResponse, error) {
	apiURL := fmt.Sprintf("https://%s/api/version?v=%s", serverHost, ClientVersion)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ver VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&ver); err != nil {
		return nil, err
	}
	return &ver, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	} else {
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func main() {
	serverHost := DefaultServerHost
	secret := DefaultSecret

	fmt.Println("==================================================================")
	fmt.Println("       👑 LORENZO VPN CLIENT (The Strict Leader) 👑               ")
	fmt.Println("==================================================================")
	fmt.Printf(" [•] إصدار التطبيق: v%s\n", ClientVersion)
	fmt.Printf(" [•] السيرفر: %s (أمستردام، أوروبا)\n", serverHost)
	fmt.Println("------------------------------------------------------------------")
	fmt.Println(" [*] جاري فحص الحماية والتحديثات الأمنية من السيرفر...")

	// 1. Mandatory Force Update Check
	verInfo, err := checkForceUpdate(serverHost)
	if err != nil {
		fmt.Printf(" [!] تنبيه: تعذر الوصول لواجهة فحص التحديثات (%v)، سيتم المتابعة...\n", err)
	} else if verInfo.ForceUpdate {
		fmt.Println("\n==================================================================")
		fmt.Println(" 🛑🛑🛑 [تحديث إجباري مطلوب - MANDATORY UPDATE REQUIRED] 🛑🛑🛑")
		fmt.Println("==================================================================")
		fmt.Printf(" ⚠️ نسختك الحالية (v%s) أصبحت قديمة وغير مدعومة أمنياً!\n", ClientVersion)
		fmt.Printf(" 🔒 أدنى نسخة مطلوبة للاتصال: v%s\n", verInfo.MinClientVersion)
		fmt.Printf(" 📝 تفاصيل التحديث: %s\n", verInfo.Changelog)
		fmt.Println("------------------------------------------------------------------")
		fmt.Println(" 🚀 جاري فتح رابط التحديث المباشر الآن...")
		openBrowser(verInfo.DownloadURL)
		fmt.Println(" ❌ تم قفل الاتصال. يرجى تثبيت النسخة الجديدة والمحاولة مجدداً.")
		fmt.Println("==================================================================")
		os.Exit(1)
	} else {
		fmt.Println(" [✓] تم التحقق من أمان النسخة: مصرح بالاتصال 100%.")
	}

	// 2. Setup Tunnel Client
	tunnelURL := fmt.Sprintf("wss://%s/lorenzo-tunnel", serverHost)
	tunnelClient := NewLorenzoTunnelClient(tunnelURL, secret, ClientVersion)

	// 3. Start Local Proxy Server
	httpAddr := "127.0.0.1:" + HTTPProxyPort
	socksAddr := "127.0.0.1:" + SOCKSProxyPort
	proxyServer := NewProxyServer(tunnelClient)

	err = proxyServer.Start(httpAddr, socksAddr)
	if err != nil {
		log.Fatalf(" [!] فشل تشغيل البروكسي المحلي: %v", err)
	}
	defer proxyServer.Stop()

	// 4. Configure Windows System Proxy
	err = setWindowsProxy(httpAddr)
	if err != nil {
		fmt.Printf(" [!] تنبيه: تعذر ضبط بروكسي الويندوز تلقائياً (%v)، يمكنك استخدامه يدوياً على %s\n", err, httpAddr)
	} else {
		fmt.Printf(" [✓] تم تفعيل البروكسي للنظام بالكامل بنجاح: %s\n", httpAddr)
	}

	// 5. Cleanup Hook on Exit
	cleanupCh := make(chan os.Signal, 1)
	signal.Notify(cleanupCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cleanupCh
		fmt.Println("\n [*] جاري إيقاف الاتصال واستعادة إعدادات الإنترنت الطبيعية...")
		_ = disableWindowsProxy()
		os.Exit(0)
	}()

	fmt.Println("==================================================================")
	fmt.Println(" 🟢 الحالة: النفق المشفر نشط الآن بنجاح! (CONNECTED)")
	fmt.Printf(" 🌐 SOCKS5 Port: %s | HTTP/HTTPS Port: %s\n", SOCKSProxyPort, HTTPProxyPort)
	fmt.Println(" 🛡️ جميع البرامج والمتصفحات والألعاب تمر الآن عبر سيرفر أمستردام المشفر.")
	fmt.Println("==================================================================")
	fmt.Println(" اضغط [Enter] في أي وقت لقطع الاتصال واستعادة الإعدادات...")

	// Wait for user enter key
	var input string
	fmt.Scanln(&input)

	fmt.Println(" [*] جاري قطع الاتصال...")
	_ = disableWindowsProxy()
	fmt.Println(" [✓] تم قطع الاتصال واستعادة الإنترنت العادي بنجاح. مع السلامة!")
}
