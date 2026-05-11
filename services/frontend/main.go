package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/wbk/webinar-demo/pkg/httpx"
	"github.com/wbk/webinar-demo/pkg/model"
)

//go:embed templates/*.html
var templatesFS embed.FS

var tmpl *template.Template

var (
	productCatalogURL string
	cartURL           string
	checkoutURL       string
)

type lineItem struct {
	Product  model.Product
	Quantity int
	Subtotal float64
}

func main() {
	port := getenv("PORT", "8080")
	productCatalogURL = getenv("PRODUCT_CATALOG_URL", "http://product-catalog:8081")
	cartURL = getenv("CART_URL", "http://cart:8082")
	checkoutURL = getenv("CHECKOUT_URL", "http://checkout:8083")

	funcs := template.FuncMap{
		"multiply":    func(p float64, q int) float64 { return p * float64(q) },
		"formatPrice": func(p float64) string { return fmt.Sprintf("$%.2f", p) },
	}
	var err error
	tmpl, err = template.New("").Funcs(funcs).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("POST /cart/add", handleCartAdd)
	mux.HandleFunc("GET /cart", handleCart)
	mux.HandleFunc("POST /checkout", handleCheckout)
	mux.HandleFunc("GET /thanks", handleThanks)

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("frontend listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getUID(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("uid")
	if err == nil && c.Value != "" {
		return c.Value
	}
	uid := uuid.NewString()
	http.SetCookie(w, &http.Cookie{
		Name:   "uid",
		Value:  uid,
		Path:   "/",
		MaxAge: 30 * 24 * 60 * 60,
	})
	return uid
}

func cartCount(c model.Cart) int {
	n := 0
	for _, it := range c.Items {
		n += it.Quantity
	}
	return n
}

func renderError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, "<!doctype html><html><body><h1>Error</h1><p>%s</p><p><a href=\"/\">Back</a></p></body></html>", template.HTMLEscapeString(msg))
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	uid := getUID(w, r)
	var products []model.Product
	if err := httpx.GetJSON(productCatalogURL+"/products", &products); err != nil {
		log.Printf("get products: %v", err)
		renderError(w, http.StatusBadGateway, "could not load products")
		return
	}
	var cart model.Cart
	if err := httpx.GetJSON(cartURL+"/carts/"+uid, &cart); err != nil {
		log.Printf("get cart: %v", err)
		cart = model.Cart{UserID: uid}
	}
	data := struct {
		Products  []model.Product
		CartCount int
	}{products, cartCount(cart)}
	if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("render index: %v", err)
	}
}

func handleCartAdd(w http.ResponseWriter, r *http.Request) {
	uid := getUID(w, r)
	if err := r.ParseForm(); err != nil {
		renderError(w, http.StatusBadRequest, "bad form")
		return
	}
	pid := r.FormValue("product_id")
	qty, _ := strconv.Atoi(r.FormValue("quantity"))
	if qty <= 0 {
		qty = 1
	}
	body := model.AddToCartRequest{ProductID: pid, Quantity: qty}
	if err := httpx.PostJSON(cartURL+"/carts/"+uid, body, nil); err != nil {
		log.Printf("add to cart: %v", err)
		renderError(w, http.StatusBadGateway, "could not add to cart")
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleCart(w http.ResponseWriter, r *http.Request) {
	uid := getUID(w, r)
	var cart model.Cart
	if err := httpx.GetJSON(cartURL+"/carts/"+uid, &cart); err != nil {
		log.Printf("get cart: %v", err)
		renderError(w, http.StatusBadGateway, "could not load cart")
		return
	}
	items := make([]lineItem, 0, len(cart.Items))
	var total float64
	for _, it := range cart.Items {
		var p model.Product
		if err := httpx.GetJSON(productCatalogURL+"/products/"+it.ProductID, &p); err != nil {
			log.Printf("get product %s: %v", it.ProductID, err)
			continue
		}
		sub := p.Price * float64(it.Quantity)
		total += sub
		items = append(items, lineItem{Product: p, Quantity: it.Quantity, Subtotal: sub})
	}
	data := struct {
		Items []lineItem
		Total float64
	}{items, total}
	if err := tmpl.ExecuteTemplate(w, "cart.html", data); err != nil {
		log.Printf("render cart: %v", err)
	}
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	uid := getUID(w, r)
	var resp model.CheckoutResponse
	if err := httpx.PostJSON(checkoutURL+"/checkout", model.CheckoutRequest{UserID: uid}, &resp); err != nil {
		log.Printf("checkout: %v", err)
		renderError(w, http.StatusBadGateway, "checkout failed: "+err.Error())
		return
	}
	q := url.Values{}
	q.Set("order", resp.OrderID)
	q.Set("total", fmt.Sprintf("%.2f", resp.Total))
	http.Redirect(w, r, "/thanks?"+q.Encode(), http.StatusSeeOther)
}

func handleThanks(w http.ResponseWriter, r *http.Request) {
	getUID(w, r)
	data := struct {
		Order string
		Total string
	}{r.URL.Query().Get("order"), r.URL.Query().Get("total")}
	if err := tmpl.ExecuteTemplate(w, "thanks.html", data); err != nil {
		log.Printf("render thanks: %v", err)
	}
}
