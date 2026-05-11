package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	log.Printf("payment starting on :%s", port)
	ready.Store(true)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /charge", func(w http.ResponseWriter, r *http.Request) {
		var req model.ChargeRequest
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteError(w, 400, err.Error())
			return
		}
		if req.Amount <= 0 || req.Currency == "" || req.UserID == "" {
			httpx.WriteError(w, 400, "user_id, positive amount, currency required")
			return
		}
		time.Sleep(50 * time.Millisecond)
		resp := model.ChargeResponse{
			TransactionID: uuid.NewString(),
			Status:        "ok",
		}
		httpx.WriteJSON(w, 200, resp)
	})

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
