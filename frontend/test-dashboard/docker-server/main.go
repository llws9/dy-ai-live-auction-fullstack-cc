package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	addr := getenv("DASHBOARD_ADDR", ":80")
	gatewayURL := getenv("GATEWAY_URL", "http://gateway:8080")
	gateway, err := url.Parse(gatewayURL)
	if err != nil {
		log.Fatalf("invalid GATEWAY_URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(gateway)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", proxy.ServeHTTP)
	mux.HandleFunc("/ws/", proxy.ServeHTTP)
	mux.HandleFunc("/", spaHandler("dist"))

	log.Printf("test-dashboard listening %s, gateway=%s", addr, gatewayURL)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func spaHandler(root string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(root, filepath.Clean(r.URL.Path))
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		if !strings.Contains(r.Header.Get("Accept"), "text/html") {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
