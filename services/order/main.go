package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres.shop:5432/orders?sslmode=disable"
	}
	log.Printf("order starting on :%s (dsn=%s)", port, dsn)

	var pool *pgxpool.Pool
	go func() {
		for i := 0; i < 30; i++ {
			p, err := pgxpool.New(context.Background(), dsn)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err = p.Ping(ctx)
				cancel()
				if err == nil {
					pool = p
					ready.Store(true)
					log.Printf("postgres connected")
					return
				}
				p.Close()
			}
			log.Printf("postgres not ready (attempt %d): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		log.Fatalf("postgres unreachable after retries")
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateOrderRequest
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteError(w, 400, err.Error())
			return
		}
		if req.UserID == "" || len(req.Items) == 0 || req.TransactionID == "" {
			httpx.WriteError(w, 400, "user_id, items, transaction_id required")
			return
		}
		order := model.Order{
			ID:            uuid.NewString(),
			UserID:        req.UserID,
			Items:         req.Items,
			Total:         req.Total,
			Currency:      req.Currency,
			TransactionID: req.TransactionID,
			CreatedAt:     time.Now().UTC(),
		}
		itemsJSON, err := json.Marshal(order.Items)
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		_, err = pool.Exec(r.Context(),
			`INSERT INTO orders (id, user_id, items, total, currency, transaction_id, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			order.ID, order.UserID, itemsJSON, order.Total, order.Currency, order.TransactionID, order.CreatedAt,
		)
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		httpx.WriteJSON(w, 200, order)
	})
	mux.HandleFunc("GET /orders/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		rows, err := pool.Query(r.Context(),
			`SELECT id, user_id, items, total, currency, transaction_id, created_at
			 FROM orders WHERE user_id=$1 ORDER BY created_at DESC LIMIT 50`, userID)
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		out := []model.Order{}
		for rows.Next() {
			var o model.Order
			var itemsJSON []byte
			if err := rows.Scan(&o.ID, &o.UserID, &itemsJSON, &o.Total, &o.Currency, &o.TransactionID, &o.CreatedAt); err != nil {
				httpx.WriteError(w, 500, err.Error())
				return
			}
			if err := json.Unmarshal(itemsJSON, &o.Items); err != nil {
				httpx.WriteError(w, 500, err.Error())
				return
			}
			out = append(out, o)
		}
		httpx.WriteJSON(w, 200, out)
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
	if pool != nil {
		pool.Close()
	}
}
