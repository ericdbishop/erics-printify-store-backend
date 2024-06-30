package external

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"server/cart"
	"server/config"
	"server/session"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
)

func InitWebhook(mux *http.ServeMux) {
	// This is your test secret API key.
	stripe.Key = config.STRIPE_SECRET

	mux.HandleFunc("/webhook", handleWebhook)
}

func handleWebhook(w http.ResponseWriter, req *http.Request) {
	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("handleWebhook: Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		log.Printf("Error in handleWebhook: Webhook error while parsing basic request. %v\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Replace this endpoint secret with your endpoint's unique secret
	// If you are testing with the CLI, find the secret by running 'stripe listen'
	// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
	// at https://dashboard.stripe.com/webhooks
	endpointSecret := config.STRIPE_WEBHOOK_SECRET
	signatureHeader := req.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		log.Printf("Error in handleWebhook: Webhook signature verification failed. %v\n", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}
	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Successful payment amount for %s: %d.\n", paymentIntent.ID, paymentIntent.Amount)
		// Inform user their order will be on the way.
		err = handlePaymentIntentSucceeded(paymentIntent)
		if err != nil {
			log.Printf("Error in handlePaymentIntentSucceeded: %v\n", err)
			return
		}
	case "payment_intent.failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Failed payment for %s: %d.\n", paymentIntent.ID, paymentIntent.Amount)
	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Failed payment for %s: %d.\n", paymentIntent.ID, paymentIntent.Amount)
	}

	w.WriteHeader(http.StatusOK)
}

func handlePaymentIntentSucceeded(payment_intent stripe.PaymentIntent) error {
	items, err := session.RetrievePaymentIntentItems(payment_intent.ID)

	if err != nil {
		log.Printf("handlePaymentIntentSucceeded: Error retrieving client items after succesful payment: %v\n", err)
		return err
	}
	client_info := formClientInfo(payment_intent)

	err = submitOrder(items, client_info)
	if err != nil {
		log.Printf("handlePaymentIntentSucceeded: Error in handling order submission: %v\n", err)
		return err
	} else {
		shopping_cart, err := cart.Repo.GetCartByPaymentIntentID(client_info.PaymentIntentID)
		if err != nil {
			log.Printf("handlePaymentIntentSucceeded: Could not retrieve cart to clear session id: %v\n", err)
			return nil
		}
		// replace user's session id with another random session id so their
		// cart will be cleared for them, but it won't be deleted from the db.
		err = cart.Repo.UpdateSessionID(shopping_cart.SessionID, session.SessionId())
		if err != nil {
			log.Printf("handlePaymentIntentSucceeded: Could not clear session id: %v\n", err)
		}
	}

	return nil
}

func formClientInfo(payment_intent stripe.PaymentIntent) *ClientInfo {
	addr := Address{
		Line1:      payment_intent.Shipping.Address.Line1,
		Line2:      payment_intent.Shipping.Address.Line2,
		City:       payment_intent.Shipping.Address.City,
		Country:    payment_intent.Shipping.Address.Country,
		PostalCode: payment_intent.Shipping.Address.PostalCode,
		State:      payment_intent.Shipping.Address.State,
	}
	return &ClientInfo{
		ClientSecret:    payment_intent.ClientSecret,
		PaymentIntentID: payment_intent.ID,
		Name:            payment_intent.Shipping.Name,
		Address:         &addr,
		Email:           payment_intent.ReceiptEmail,
	}
}
