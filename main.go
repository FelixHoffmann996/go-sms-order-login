package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type loginRequest struct {
	OrderID   string `json:"order_id"`
	Phone     string `json:"phone"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	service := &orderLogin{
		codes: newSMSClient(key),
		orders: map[string]Order{
			"ord_1042": {
				ID: "ord_1042", Phone: "+15551234567", Checkout: "paid",
				Fulfillment: "packed", Receipt: "rcpt_1042",
				Updates: []string{"payment accepted", "packing started"},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/code", func(w http.ResponseWriter, r *http.Request) {
		var input loginRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if input.OrderID == "" || input.Phone == "" || input.RequestID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id, phone, and request_id are required"})
			return
		}
		if err := service.sendCode(r.Context(), input.OrderID, input.Phone, input.RequestID); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "code_sent"})
	})
	mux.HandleFunc("POST /login/verify", func(w http.ResponseWriter, r *http.Request) {
		var input loginRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
			return
		}
		if input.OrderID == "" || input.Phone == "" || input.Code == "" || input.RequestID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id, phone, code, and request_id are required"})
			return
		}
		order, err := service.verifyAndRead(r.Context(), input.OrderID, input.Phone, input.Code, input.RequestID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, order)
	})

	log.Println("order login listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
