package model

import "time"

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Stock       int     `json:"stock"`
}

type CartItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Cart struct {
	UserID string     `json:"user_id"`
	Items  []CartItem `json:"items"`
}

type AddToCartRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ChargeRequest struct {
	UserID   string  `json:"user_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type ChargeResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type Order struct {
	ID            string      `json:"id"`
	UserID        string      `json:"user_id"`
	Items         []OrderItem `json:"items"`
	Total         float64     `json:"total"`
	Currency      string      `json:"currency"`
	TransactionID string      `json:"transaction_id"`
	CreatedAt     time.Time   `json:"created_at"`
}

type CreateOrderRequest struct {
	UserID        string      `json:"user_id"`
	Items         []OrderItem `json:"items"`
	Total         float64     `json:"total"`
	Currency      string      `json:"currency"`
	TransactionID string      `json:"transaction_id"`
}

type CheckoutRequest struct {
	UserID string `json:"user_id"`
}

type CheckoutResponse struct {
	OrderID       string  `json:"order_id"`
	Total         float64 `json:"total"`
	Currency      string  `json:"currency"`
	TransactionID string  `json:"transaction_id"`
}
