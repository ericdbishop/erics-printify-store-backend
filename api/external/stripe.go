package external

/* Handle Stripe API connection and calls including checkout */

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"server/config"
	"server/session"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
)

const (
	TSHIRT     = "tshirt"
	SWEATSHIRT = "sweatshirt"
	HOODIE     = "hoodie"
)

type UpdateData struct {
	Status        string `json:"status"`
	ItemsPrice    string `json:"cart"`
	ShippingPrice string `json:"shipping"`
	TotalPrice    string `json:"total"`
}

type ClientInfo struct {
	ClientSecret    string `json:"client_secret"`
	PaymentIntentID string
	Name            string   `json:"name"`
	Address         *Address `json:"address"`
	Email           string   `json:"receipt_email,omitempty"`
}

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`
}

func InitHandlers(mux *http.ServeMux) {
	// This is your test secret API key.
	stripe.Key = config.STRIPE_SECRET

	mux.HandleFunc("/api/create-payment-intent", handleCreatePaymentIntent)
	mux.HandleFunc("/api/address-update", handleUpdate)
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		log.Printf("handleUpdate: Wrong request method: %s\n", r.Method)
		return
	}

	decoder := json.NewDecoder(r.Body)

	var update ClientInfo
	err := decoder.Decode(&update)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("handleUpdate: Could not decode request body: %v\n", err)
		return
	}

	update.PaymentIntentID = strings.Split(update.ClientSecret, "_secret")[0]

	amount, err := session.RetrieveOrderAmountAndItems(update.PaymentIntentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("handleUpdate: Error in RetrieveOrderAmountAndItems(): %v\n", err)
		return
	}

	cart_items, err := session.RetrievePaymentIntentItems(update.PaymentIntentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("handleUpdate: Error in retrievePaymentIntentItems(): %v\n", err)
		return
	}

	shipping_cost := GetShippingCost(cart_items, &update)

	cart_total := amount

	amount += shipping_cost

	log.Printf("Updating cost for %s: cart=%d, shipping=%d, total=%d\n", update.PaymentIntentID, cart_total, shipping_cost, amount)

	pi, err := updatePaymentIntentAmount(update.PaymentIntentID, amount)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("handleUpdate: error from updatePaymentIntentAmount: %v\n", err)
		return
	}

	data := UpdateData{
		Status:        string(pi.Status),
		ItemsPrice:    fmt.Sprintf("%.2f", float64(cart_total)/100),
		ShippingPrice: fmt.Sprintf("%.2f", float64(shipping_cost)/100),
		TotalPrice:    fmt.Sprintf("%.2f", float64(amount)/100),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	session_id := session.BeginSession(w, r)
	order_amount, err := session.RetrieveOrderAmount(w, session_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error retrieving order amount for session id: %s\n", session_id)
		return
	}

	// Check for an existing PaymentIntent ID for the user
	paymentintent_id, err := session.RetrievePaymentIntentID(session_id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error retrieving PaymentIntent for session id: %s\n", session_id)
		return
	}

	// Create a PaymentIntent with amount and currency
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(order_amount),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	var payment_intent_exists bool = (paymentintent_id != "")

	var pi *stripe.PaymentIntent
	if payment_intent_exists {
		pi, err = updatePaymentIntentAmount(paymentintent_id, order_amount)
		log.Printf("pi Exists: %v", pi.ClientSecret)
	} else {
		pi, err = paymentintent.New(params)
		log.Printf("pi.New: %v", pi.ClientSecret)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("pi error: %v", err)
		return
	}

	// Store pi ID if it is a new payment intent.
	if !payment_intent_exists {
		//log.Printf("\nAdding payment intent to shopping cart, PI ID: %s\n", pi.ID)
		err = session.AddPaymentIntentID(session_id, pi.ID)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Printf("session.AddPaymentIntentID: %v", err)
			return
		}
	}

	log.Printf("SessionID %s, PaymentIntent %s cart total (without shipping): %d\n", session_id, pi.ID, order_amount)

	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	writeJSON(w, struct {
		ClientSecret string `json:"clientSecret"`
	}{
		ClientSecret: pi.ClientSecret,
	})
}

func updatePaymentIntentAmount(paymentintent_id string, amount int64) (*stripe.PaymentIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount: stripe.Int64(amount),
	}

	pi, err := paymentintent.Update(
		paymentintent_id,
		params,
	)

	return pi, err
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}
