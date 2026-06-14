package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"xeh/internal/core"
)

// StartHTTPServer は <server> タグの内容に基づいてHTTPサーバーを非同期で起動する。
//
// <server> は app.xeh のルート直下に置く特別なタグで、<subframework> の
// タグ群とは異なりレジストリ経由ではなく main.go から直接呼び出される。
func StartHTTPServer(serverConfig core.XehServer, store *core.SharedStore) *http.Server {
	port := strings.TrimSpace(serverConfig.Port)
	if port == "" {
		port = "8080"
	}

	pageHTML := strings.TrimSpace(serverConfig.InnerXML)
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		body := pageHTML
		if body == "" {
			body = "<!DOCTYPE html><html><body><h1>Welcome to xeh server</h1></body></html>"
		}
		_, _ = w.Write([]byte(body))
	})

	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		handleSubmit(w, r, store)
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("[xeh/os] HTTP server '%s' listening on %s", serverConfig.ID, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Error] HTTP server '%s' failed: %v", serverConfig.ID, err)
		}
	}()

	return srv
}

// handleSubmit は /submit へのフォーム送信を受け取り、送信内容と共有メモリの現在値を表示する
func handleSubmit(w http.ResponseWriter, r *http.Request, store *core.SharedStore) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Failed to parse form data."))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!DOCTYPE html><html><body><h1>Form Submission</h1>"))
	_, _ = w.Write([]byte("<p>Received form values:</p><ul>"))
	for key, values := range r.Form {
		_, _ = w.Write([]byte(fmt.Sprintf("<li><strong>%s</strong>: %s</li>", key, strings.Join(values, ", "))))
	}
	_, _ = w.Write([]byte("</ul>"))

	_, _ = w.Write([]byte("<h2>Current Memory Values</h2><ul>"))
	store.Range(func(key string, value interface{}) {
		_, _ = w.Write([]byte(fmt.Sprintf("<li><strong>%s</strong>: %v</li>", key, value)))
	})
	_, _ = w.Write([]byte("</ul><p><a href=\"/\">Back to form</a></p></body></html>"))
}
