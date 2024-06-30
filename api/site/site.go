package site

import (
	"encoding/json"
	"log"
	"net/http"
	"server/cart"
	"server/error_messages"
	"server/session"

	"github.com/gorilla/csrf"
)

func InitHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/items", retrieveItemCount)
	mux.HandleFunc("/api/retrieve_cart", retrieveCartItems)
	mux.HandleFunc("/api/add_to_cart", addToCart)
	mux.HandleFunc("/api/remove_from_cart", removeFromCart)
	mux.HandleFunc("/api/checkout", removeFromCart)
}

/* Send the number item's in the client's cart in a response */
func retrieveItemCount(w http.ResponseWriter, r *http.Request) {
	/* We do not need to store the user's session id in the database for this
	 * request unless it is there already, so we call BeginSession instead of
	 * RetrieveCart */
	session_id := session.BeginSession(w, r)

	retrieved_items, err := session.RetrieveItems(session_id)
	if err != error_messages.ErrNotExists && err != nil {
		error_bad_request(w, "Failed to retrieve items in retrieveItemCount()", err)
		return
	}

	numItems := len(retrieved_items)
	obj := map[string]int{"items": numItems}
	jsonResp, err := json.Marshal(obj)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

/* Send a list of items in the client's cart in a resposne */
func retrieveCartItems(w http.ResponseWriter, r *http.Request) {
	/* We do not need to store the user's session id in the database for this
	 * request unless it is there already, so we call BeginSession instead of
	 * RetrieveCart */
	session_id := session.BeginSession(w, r)

	retrieved_items, err := session.RetrieveItems(session_id)
	if err != error_messages.ErrNotExists && err != nil {
		error_bad_request(w, "retrieveCartItems: Failed to retrieve items", err)
		return
	}

	items := []cart.CartItem{}
	for _, item := range retrieved_items {
		items = append(items, cart.AddDisplayDetails(item))
	}

	jsonResp, err := json.Marshal(items)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(http.StatusCreated)
	w.Write(jsonResp)
}

/* Add item to client's cart */
func addToCart(w http.ResponseWriter, r *http.Request) {
	item, err := validate_item(r)

	if err != nil {
		error_bad_request(w, "addToCart: Can not decode JSON", err)
		return
	}

	/* session.RetrieveCart will check/create the user's cookie containing
	 * their session id and uses the session id to create/retrieve a shopping
	 * cart entry in the database */
	shopping_cart, err := session.RetrieveCart(w, r)
	if err != nil {
		error_bad_request(w, "addToCart: Failed to retrieve/create session", err)
		return
	}

	retrieved_items, err := cart.Repo.GetItemsBySessionID(shopping_cart.SessionID)
	if err != nil {
		error_bad_request(w, "addToCart: Could not retrieve items", err)
		return
	}

	numItems := len(retrieved_items)
	if numItems >= 8 {
		error_bad_request(w, "addToCart: Too many items are in the user's cart", err)
		return
	}

	item.ShoppingCartID = shopping_cart.ID
	_, err = cart.Repo.CreateItemEntry(*item)

	if err != nil {
		error_bad_request(w, "addToCart: Failed to create item", err)
		return
	}

	log.Printf("%s: Added %s to cart\n", shopping_cart.SessionID, item.Item)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Successful Request"))
}

/* Remove an item from a user's cart items in the database */
func removeFromCart(w http.ResponseWriter, r *http.Request) {
	item, err := validate_item(r)

	if err != nil {
		error_bad_request(w, "removeFromCart: Can not decode JSON", err)
		return
	}

	/* session.RetrieveCart will check/create the user's cookie containing
	 * their session id and uses the session id to create/retrieve a shopping
	 * cart entry in the database */
	shopping_cart, err := session.RetrieveCart(w, r)

	if err != nil {
		error_bad_request(w, "removeFromCart: Failed to retrieve/create session", err)
		return
	}

	item.ShoppingCartID = shopping_cart.ID
	err = cart.Repo.DeleteItem(*item)

	if err != nil {
		error_bad_request(w, "removeFromCart: Failed to delete item", err)
		return
	}

	log.Printf("%s: Removed %s to cart\n", shopping_cart.SessionID, item.Item)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-CSRF-Token", csrf.Token(r))
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Successful Request"))
}

/* Decode JSON object and ensure that each field for the item contains a valid
 * value. */
func validate_item(r *http.Request) (*cart.CartItem, error) {
	decoder := json.NewDecoder(r.Body)

	var item cart.CartItem
	err := decoder.Decode(&item)
	if err != nil {
		return nil, err
	}

	if !contains(cart.Items, item.Item) || !contains(cart.Sizes, item.Size) || !contains(cart.Colors, item.Color) {
		log.Printf("Error in validate_item(): %v\n", error_messages.ErrInvalidItem)
		return nil, error_messages.ErrInvalidItem
	}

	return &item, nil
}

// contains checks if a value is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func error_bad_request(w http.ResponseWriter, print string, err error) {
	log.Printf("Error in %s: %v\n", print, err)
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
}
