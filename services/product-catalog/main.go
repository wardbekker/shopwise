package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@postgres.shop:5432/products?sslmode=disable"
	}

	log.Printf("product-catalog starting on :%s (dsn=%s)", port, dsn)

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
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		rows, err := pool.Query(r.Context(), "SELECT id, name, description, price, currency, stock FROM products ORDER BY id")
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		defer rows.Close()
		out := []model.Product{}
		for rows.Next() {
			var p model.Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.Stock); err != nil {
				httpx.WriteError(w, 500, err.Error())
				return
			}
			out = append(out, p)
		}
		httpx.WriteJSON(w, 200, out)
	})
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var p model.Product
		err := pool.QueryRow(r.Context(),
			"SELECT id, name, description, price, currency, stock FROM products WHERE id=$1", id,
		).Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Currency, &p.Stock)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, 404, "not found")
			return
		}
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		httpx.WriteJSON(w, 200, p)
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
