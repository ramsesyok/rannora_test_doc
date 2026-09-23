// Stateful teaching API. All data is in memory; never connects to a real shop.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type order struct {
	ID       string `json:"id"`
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
	Status   string `json:"status"`
}

type shop struct {
	sync.Mutex
	stock  int
	nextID int
	orders map[string]*order
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Print(err)
	}
}

func problem(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"code": code})
}

func (s *shop) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()
	log.Printf("%s %s", r.Method, r.URL.Path)
	switch {
	case r.Method == "POST" && r.URL.Path == "/test/reset":
		s.stock = 5
		s.orders = map[string]*order{}
		// Keep the sequence across resets: scenarios must capture the returned ID.
		writeJSON(w, 200, map[string]any{"sku": "PEN-001", "available": s.stock, "orderCount": len(s.orders)})
	case r.Method == "GET" && r.URL.Path == "/inventory/PEN-001":
		writeJSON(w, 200, map[string]any{"sku": "PEN-001", "available": s.stock})
	case r.Method == "POST" && r.URL.Path == "/orders":
		var input struct {
			SKU      string `json:"sku"`
			Quantity int    `json:"quantity"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			problem(w, 400, "INVALID_REQUEST")
			return
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF || input.SKU != "PEN-001" || input.Quantity < 1 {
			problem(w, 400, "INVALID_REQUEST")
			return
		}
		if input.Quantity > s.stock {
			problem(w, 409, "INSUFFICIENT_STOCK")
			return
		}
		s.nextID++
		item := &order{ID: fmt.Sprintf("ORD-%06d", s.nextID), SKU: input.SKU, Quantity: input.Quantity, Status: "confirmed"}
		s.orders[item.ID] = item
		s.stock -= input.Quantity
		writeJSON(w, 201, item)
	default:
		// Path matching is delegated to a small ServeMux to avoid loose suffix matching.
		mux := http.NewServeMux()
		mux.HandleFunc("GET /orders/{orderId}", func(w http.ResponseWriter, r *http.Request) {
			item, exists := s.orders[r.PathValue("orderId")]
			if !exists {
				problem(w, 404, "ORDER_NOT_FOUND")
				return
			}
			writeJSON(w, 200, item)
		})
		mux.HandleFunc("POST /orders/{orderId}/cancel", func(w http.ResponseWriter, r *http.Request) {
			item, exists := s.orders[r.PathValue("orderId")]
			if !exists {
				problem(w, 404, "ORDER_NOT_FOUND")
				return
			}
			if item.Status == "cancelled" {
				problem(w, 409, "ORDER_ALREADY_CANCELLED")
				return
			}
			item.Status = "cancelled"
			s.stock += item.Quantity
			writeJSON(w, 200, item)
		})
		mux.ServeHTTP(w, r)
	}
}

func main() {
	s := &shop{stock: 5, orders: map[string]*order{}}
	server := &http.Server{Addr: "127.0.0.1:18081", Handler: s, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Order workflow sample: http://%s (Ctrl+C to stop)", server.Addr)
	log.Fatal(server.ListenAndServe())
}
