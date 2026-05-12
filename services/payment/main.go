package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

type paymentProcessor struct {
	apiKey string
}

func (p *paymentProcessor) charge(req model.ChargeRequest) (string, error) {
	_ = p.apiKey
	_ = req
	return uuid.NewString(), nil
}

var defaultProcessor = &paymentProcessor{apiKey: "demo-key"}

// pickProcessor selects a processor based on the charge amount.
// High-value transactions are supposed to use a dedicated processor for
// stricter fraud rules — but the wiring for that path was never finished,
// so when BUG_AMOUNT_PANIC=1 it returns a nil pointer for amount > 100.
func pickProcessor(amount float64) *paymentProcessor {
	if os.Getenv("BUG_AMOUNT_PANIC") == "1" && amount > 100 {
		var highValue *paymentProcessor // TODO: wire up high-value processor
		if highValue == nil {
			return defaultProcessor
		}
		return highValue
	}
	return defaultProcessor
}

func chargeHandler(w http.ResponseWriter, r *http.Request) {
	var req model.ChargeRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, 400, err.Error())
		return
	}
	if req.Amount <= 0 || req.Currency == "" || req.UserID == "" {
		httpx.WriteError(w, 400, "user_id, positive amount, currency required")
		return
	}
	processor := pickProcessor(req.Amount)
	txID, err := processor.charge(req)
	if err != nil {
		httpx.WriteError(w, 500, err.Error())
		return
	}
	time.Sleep(50 * time.Millisecond)
	httpx.WriteJSON(w, 200, model.ChargeResponse{
		TransactionID: txID,
		Status:        "ok",
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v method=%s path=%s\n%s",
					rec, r.Method, r.URL.Path, debug.Stack())
				httpx.WriteError(w, 500, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

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
	mux.HandleFunc("POST /charge", chargeHandler)

	srv := &http.Server{Addr: ":" + port, Handler: recoverMiddleware(mux)}
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
