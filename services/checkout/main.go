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

	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

var ready atomic.Bool

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	productURL := os.Getenv("PRODUCT_CATALOG_URL")
	cartURL := os.Getenv("CART_URL")
	paymentURL := os.Getenv("PAYMENT_URL")
	orderURL := os.Getenv("ORDER_URL")
	log.Printf("checkout starting on :%s (product=%s cart=%s payment=%s order=%s)",
		port, productURL, cartURL, paymentURL, orderURL)
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
	mux.HandleFunc("POST /checkout", func(w http.ResponseWriter, r *http.Request) {
		var req model.CheckoutRequest
		if err := httpx.ReadJSON(r, &req); err != nil {
			httpx.WriteError(w, 400, err.Error())
			return
		}
		if req.UserID == "" {
			httpx.WriteError(w, 400, "user_id required")
			return
		}

		var cart model.Cart
		if err := httpx.GetJSON(cartURL+"/carts/"+req.UserID, &cart); err != nil {
			httpx.WriteError(w, 502, "cart fetch: "+err.Error())
			return
		}
		if len(cart.Items) == 0 {
			httpx.WriteError(w, 400, "cart empty")
			return
		}

		items := make([]model.OrderItem, 0, len(cart.Items))
		var total float64
		for _, ci := range cart.Items {
			var p model.Product
			if err := httpx.GetJSON(productURL+"/products/"+ci.ProductID, &p); err != nil {
				httpx.WriteError(w, 502, "product fetch: "+err.Error())
				return
			}
			items = append(items, model.OrderItem{
				ProductID: p.ID,
				Name:      p.Name,
				Quantity:  ci.Quantity,
				UnitPrice: p.Price,
			})
			total += p.Price * float64(ci.Quantity)
		}

		var charge model.ChargeResponse
		if err := httpx.PostJSON(paymentURL+"/charge", model.ChargeRequest{
			UserID:   req.UserID,
			Amount:   total,
			Currency: "USD",
		}, &charge); err != nil {
			httpx.WriteError(w, 502, "charge: "+err.Error())
			return
		}

		var order model.Order
		if err := httpx.PostJSON(orderURL+"/orders", model.CreateOrderRequest{
			UserID:        req.UserID,
			Items:         items,
			Total:         total,
			Currency:      "USD",
			TransactionID: charge.TransactionID,
		}, &order); err != nil {
			httpx.WriteError(w, 502, "order: "+err.Error())
			return
		}

		if err := httpx.DeleteReq(cartURL + "/carts/" + req.UserID); err != nil {
			log.Printf("warning: cart delete failed: %v", err)
		}

		httpx.WriteJSON(w, 200, model.CheckoutResponse{
			OrderID:       order.ID,
			Total:         total,
			Currency:      "USD",
			TransactionID: charge.TransactionID,
		})
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
