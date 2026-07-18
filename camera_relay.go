package main

import "net/http"

// NewCameraRelayHandler exposes only health and validated camera-frame routes.
// It is intentionally separate from the full application so Tailscale Funnel
// cannot expose bus, access, or Telegram APIs from the development PC.
func NewCameraRelayHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /camera/{id}/frame", func(w http.ResponseWriter, r *http.Request) {
		(&Server{}).handleCameraFrame(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://287287287.xyz")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}
