package cart

import (
	"database/sql"
	"strings"
)

/* 
    TODO:
    Change the contents of this file to fit your own store's custom SKU structure
    and product lineup. Values are placeholders that I may later externalize to
    the configuration.

    Find and replace: PLACEHOLDER
*/

var (
	Items  = []string{"sweatshirt", "tshirt", "hoodie"}
	Sizes  = []string{"s", "m", "l", "xl", "2xl", "3xl"}
	Colors = []string{"black", "red", "green"}
	// These values are used to charge the user
	ItemtoPrice        = map[string]int64{"sweatshirt": 3000, "tshirt": 3000, "hoodie": 3000}
	ItemtoDisplayPrice = map[string]string{"sweatshirt": "$30", "tshirt": "$30", "hoodie": "$30"}
)

type ShoppingCart struct {
	ID              int64
	SessionID       string
	PaymentIntentID string
}

type CartItem struct {
	ID             int64
	ShoppingCartID int64
	Item           string `json:"id"`
	Size           string `json:"size"`
	Color          string `json:"color"`
	// Last three parameters are strictly for displaying
	// the cart item on the /cart page
	Display struct {
		Name   string `json:"name"`
		ImgSrc string `json:"imgsrc"`
		Price  string `json:"price"`
	} `json:"display"`
}

type SQLiteDatabase struct {
	db *sql.DB
}

// Fills in some of the struct details that are only needed for the /cart page
func AddDisplayDetails(item CartItem) CartItem {
	item.Display.Name = "PLACEHOLDER "
	item.Display.ImgSrc = item.Item + "_" + item.Color
	item.Color = strings.Title(item.Color)
	item.Size = strings.ToUpper(item.Size)

	id := item.Item
	item.Display.Price = ItemtoDisplayPrice[id]
	switch {
	case id == "tshirt":
		item.Display.Name = item.Display.Name + "T-Shirt"
	case id == "hoodie":
		item.Display.Name = item.Display.Name + "Hoodie"
	case id == "sweatshirt":
		item.Display.Name = item.Display.Name + "Sweatshirt"
	}
	return item
}

/*
SKU Naming:

	PLACEHOLDER_{item first initial}__{Size}_{Color first two
	initials}

	ex: PLACEHOLDER_S_S_BL
	PLACEHOLDER_S = PLACEHOLDER_sweatshirt
	S = small
	BL = black
*/
func (item *CartItem) GetSKU() string {
	SKU := "PLACEHOLDER_"
	color := strings.ToUpper(item.Color[:2])
	size := strings.ToUpper(item.Size)

	id := item.Item
	switch {
	case id == "tshirt":
		SKU = SKU + "T_"
	case id == "hoodie":
		SKU = SKU + "H_"
	case id == "sweatshirt":
		SKU = SKU + "S_"
	}

	SKU = SKU + size + "_" + color

	return SKU
}
