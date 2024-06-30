package session

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"server/cart"
	"server/error_messages"
	"time"
)

// If the user did not send a valid cookie, create one for them.
func RetrieveCart(w http.ResponseWriter, r *http.Request) (*cart.ShoppingCart, error) {
	var shopping_cart *cart.ShoppingCart

	session_id := BeginSession(w, r)
	// Retrieve database entry
	shopping_cart, err := cart.Repo.GetCartBySessionID(session_id)
	if err == error_messages.ErrNotExists {
		// Create new session and cart record
		shopping_cart, err = cart.Repo.CreateCartEntry(session_id)
		if err != nil {
			log.Printf("Error: RetrieveCart: Could not create new cart entry for %s, error: %v\n", session_id, err)
			return nil, err
		}
	}

	return shopping_cart, nil
}

// BeginSession creates a new user session ID and stores it in the user's
// cookie if the user doesn't have one yet.
func BeginSession(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("session")
	// Along with checking if cookie exists, make sure the length is valid
	if err != nil || len(cookie.Value) != 44 {
		// Create cookie and attach it to the server response
		session_id := SessionId()
		setSessionCookie(w, session_id)
		log.Printf("New session cookie created: %s\n", session_id)
		return session_id
	} else {
		return cookie.Value
	}
}

func RetrieveItems(session_id string) ([]cart.CartItem, error) {
	retrieved_items, err := cart.Repo.GetItemsBySessionID(session_id)
	if err != nil {
		if err == error_messages.ErrNotExists {
			// User's session id has not been saved to backend yet.
			retrieved_items = []cart.CartItem{}
		} else {
			log.Printf("Failure in session.RetrieveItems: %v\n", err)
			return nil, err
		}
	}
	return retrieved_items, err
}

// Called when a PaymentIntent is created in stripe.go
func RetrieveOrderAmount(w http.ResponseWriter, session_id string) (int64, error) {
	var amount int64 = 0
	retrieved_items, err := RetrieveItems(session_id)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return amount, err
	}

	for _, item := range retrieved_items {
		amount += cart.ItemtoPrice[item.Item]
	}

	return amount, err
}

func RetrievePaymentIntentItems(payment_intent_id string) ([]cart.CartItem, error) {
	shopping_cart, err := cart.Repo.GetCartByPaymentIntentID(payment_intent_id)

	if err != nil {
		log.Printf("RetrievePaymentIntentItems: Failed to retrive cart with pi id: %v\n", err)
		return nil, err
	}

	retrieved_items, err := RetrieveItems(shopping_cart.SessionID)

	if err != nil {
		return nil, err
	}

	return retrieved_items, nil
}

// Called when a PaymentIntent is created in stripe.go
func RetrieveOrderAmountAndItems(payment_intent_id string) (int64, error) {
	var amount int64 = 0

	retrieved_items, err := RetrievePaymentIntentItems(payment_intent_id)

	if err != nil {
		return amount, err
	}

	for _, item := range retrieved_items {
		amount += cart.ItemtoPrice[item.Item]
	}

	return amount, err
}

// Called after a PaymentIntent is created in stripe.go to store a user's payment intent
func AddPaymentIntentID(session_id string, paymentintent_id string) error {
	err := cart.Repo.UpdatePaymentIntentID(session_id, paymentintent_id)

	if err != nil {
		log.Printf("AddPaymentIntentID: Failed to add id to cart: %v\n", err)
	}

	return err
}

// Returns empty string if there is no payment intent id stored for the cart
func RetrievePaymentIntentID(session_id string) (string, error) {
	shopping_cart, err := cart.Repo.GetCartBySessionID(session_id)
	if err != nil {
		return "", error_messages.ErrNotExists
	}
	return shopping_cart.PaymentIntentID, nil
}

// Create and set the user's cookie in the http response
func setSessionCookie(w http.ResponseWriter, session_id string) {
	expiration := time.Now().Add(7 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:     "session",
		Value:    session_id,
		Expires:  expiration,
		SameSite: 3, // "strict"
	}
	http.SetCookie(w, &cookie)
}

// Generate a random session id
func SessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}
