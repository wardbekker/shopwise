package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

func cartKey(userID string) string { return "cart:" + userID }

func loadCart(ctx context.Context, rdb *redis.Client, userID string) (model.Cart, error) {
	cart := model.Cart{UserID: userID, Items: []model.CartItem{}}
	val, err := rdb.Get(ctx, cartKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return cart, nil
	}
	if err != nil {
		return cart, err
	}
	if err := json.Unmarshal([]byte(val), &cart); err != nil {
		return cart, err
	}
	if cart.Items == nil {
		cart.Items = []model.CartItem{}
	}
	return cart, nil
}

func saveCart(ctx context.Context, rdb *redis.Client, cart model.Cart) error {
	b, err := json.Marshal(cart)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, cartKey(cart.UserID), b, 0).Err()
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "redis.shop:6379"
	}
	log.Printf("cart starting on :%s (redis=%s)", port, addr)

	rdb := redis.NewClient(&redis.Options{Addr: addr})

	go func() {
		for i := 0; i < 30; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := rdb.Ping(ctx).Err()
			cancel()
			if err == nil {
				ready.Store(true)
				log.Printf("redis connected")
				return
			}
			log.Printf("redis not ready (attempt %d): %v", i+1, err)
			time.Sleep(2 * time.Second)
		}
		log.Fatalf("redis unreachable after retries")
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
	mux.HandleFunc("GET /carts/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		cart, err := loadCart(r.Context(), rdb, userID)
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		httpx.WriteJSON(w, 200, cart)
	})
	mux.HandleFunc("POST /carts/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		var req model.AddToCartRequest
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteError(w, 400, err.Error())
			return
		}
		if req.ProductID == "" || req.Quantity <= 0 {
			httpx.WriteError(w, 400, "product_id and positive quantity required")
			return
		}
		cart, err := loadCart(r.Context(), rdb, userID)
		if err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		found := false
		for i := range cart.Items {
			if cart.Items[i].ProductID == req.ProductID {
				cart.Items[i].Quantity += req.Quantity
				found = true
				break
			}
		}
		if !found {
			cart.Items = append(cart.Items, model.CartItem{ProductID: req.ProductID, Quantity: req.Quantity})
		}
		if err := saveCart(r.Context(), rdb, cart); err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		httpx.WriteJSON(w, 200, cart)
	})
	mux.HandleFunc("DELETE /carts/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userID := r.PathValue("userId")
		if err := rdb.Del(r.Context(), cartKey(userID)).Err(); err != nil {
			httpx.WriteError(w, 500, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
	_ = rdb.Close()
}
