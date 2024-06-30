package external

/* Handle Printify API connection and calls */

import (
	"fmt"
	"log"
	"server/cart"
	"strings"

	go_printify "github.com/ericdbishop/go-printify"
)

var (
	client  *go_printify.Client
	shop_id int
)

func InitPrintifyClient(api_token string, shopID int) {
	client = go_printify.NewClient(api_token)
	client.UserAgent = "Go"
	shop_id = shopID
}

// Initializes an Order struct for use with go_printify
func formOrderShipping(items []cart.CartItem, client_info *ClientInfo) *go_printify.OrderSubmission {
	line_items := []*go_printify.LineItem{}

	for _, item := range items {
		SKU := item.GetSKU()
		var line_item *go_printify.LineItem = &go_printify.LineItem{
			Sku:      &SKU,
			Quantity: 1,
		}
		line_items = append(line_items, line_item)
	}

	full_name := strings.Split(client_info.Name, " ")

	address_to := &go_printify.AddressTo{
		FirstName: full_name[0],
		Country:   client_info.Address.Country,
		Region:    client_info.Address.State,
		Address1:  client_info.Address.Line1,
		Address2:  client_info.Address.Line2,
		City:      client_info.Address.City,
		Zip:       client_info.Address.PostalCode,
	}
	if len(full_name) > 1 {
		address_to.LastName = full_name[1]
	}

	order := &go_printify.OrderSubmission{
		LineItems: line_items,
		AddressTo: address_to,
	}

	return order
}

func formOrderSubmission(items []cart.CartItem, client_info *ClientInfo) (*go_printify.OrderSubmission, error) {
	order := formOrderShipping(items, client_info)

	order.AddressTo.Email = client_info.Email

	// Order label will be the primary key of a new row in the order table
	// padded out with 0's.
	label_num, err := cart.Repo.CreateOrderEntry(items[0].ShoppingCartID)
	if err != nil {
		log.Printf("formOrderSubmission: Error in CreateOrderEntry(): %v\n", err)
		return nil, err
	}
	order.Label = fmt.Sprintf("%05d", label_num)

	shipping_notification := true
	order.SendShippingNotification = &shipping_notification
	order.ShippingMethod = 1

	return order, nil
}

func GetShippingCost(items []cart.CartItem, client_info *ClientInfo) int64 {
	order := formOrderShipping(items, client_info)

	shipping_cost, err := client.CalculateShippingCosts(shop_id, order)

	if err != nil {
		// Give it another ole' college try
		shipping_cost, err = client.CalculateShippingCosts(shop_id, order)
		if err != nil {
			log.Printf("GetShippingCost: Error calculating shipping cost: client.CalculateShippingCosts(): %v\n", err)
			return 850
		}
	}

	var cost int64 = int64(shipping_cost.Standard)

	return cost
}

func submitOrder(items []cart.CartItem, client_info *ClientInfo) error {
	order, err := formOrderSubmission(items, client_info)

	if err != nil {
		log.Printf("Error forming order struct: external.formOrderSubmission()\n")
		return err
	}

	log.Printf("Submitting order for %s: %s", client_info.PaymentIntentID, order.Label)

	client.SubmitOrder(shop_id, order)

	return nil
}
