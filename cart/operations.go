package cart

import (
	"database/sql"
	"errors"
	"log"
	"server/error_messages"

	"github.com/mattn/go-sqlite3"
)

func NewSQLiteDatabase(db *sql.DB) *SQLiteDatabase {
	return &SQLiteDatabase{
		db: db,
	}
}

func (r *SQLiteDatabase) Migrate() error {
	query := `
    CREATE TABLE IF NOT EXISTS shopping_cart(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        session_id TEXT NOT NULL UNIQUE,
		payment_intent_id TEXT
    );
    CREATE TABLE IF NOT EXISTS cart_item(
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        shopping_cart_id INTEGER NOT NULL,
        item TEXT NOT NULL,
        size TEXT NOT NULL,
        color TEXT NOT NULL,
		FOREIGN KEY (shopping_cart_id)
			REFERENCES shopping_cart (id)
			ON DELETE CASCADE
    );
    CREATE TABLE IF NOT EXISTS order_label(
        label INTEGER PRIMARY KEY AUTOINCREMENT,
        shopping_cart_id INTEGER NOT NULL,
		FOREIGN KEY (shopping_cart_id)
			REFERENCES shopping_cart (id)
			ON DELETE CASCADE
    );
    `

	_, err := r.db.Exec(query)
	return err
}

/**********/
/* CREATE */
/**********/

func (r *SQLiteDatabase) CreateCartEntry(session_id string) (*ShoppingCart, error) {
	var shopping_cart ShoppingCart = ShoppingCart{SessionID: session_id}

	res, err := r.db.Exec("INSERT INTO shopping_cart(session_id, payment_intent_id) values(?, ?)", shopping_cart.SessionID, "")
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return nil, error_messages.ErrDuplicate
			}
		}
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	shopping_cart.ID = id

	return &shopping_cart, nil
}

func (r *SQLiteDatabase) CreateItemEntry(item CartItem) (*CartItem, error) {
	res, err := r.db.Exec("INSERT INTO cart_item(shopping_cart_id, item, size, color) values(?,?,?,?)", item.ShoppingCartID, item.Item, item.Size, item.Color)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return nil, error_messages.ErrDuplicate
			}
		}
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	item.ID = id

	return &item, nil
}

func (r *SQLiteDatabase) CreateOrderEntry(shopping_cart_id int64) (int64, error) {
	res, err := r.db.Exec("INSERT INTO order_label(shopping_cart_id) values(?)", shopping_cart_id)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
				return -1, error_messages.ErrDuplicate
			}
		}
		return -1, err
	}

	label, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}

	return label, nil
}

/**********/
/* UPDATE */
/**********/

func (r *SQLiteDatabase) UpdatePaymentIntentID(session_id string, paymentintent_id string) error {
	shopping_cart, err := r.GetCartBySessionID(session_id)
	if err != nil {
		return error_messages.ErrNotExists
	}
	return r.updateCart(shopping_cart.ID, "payment_intent_id", paymentintent_id)
}

func (r *SQLiteDatabase) UpdateSessionID(session_id string, new_session_id string) error {
	shopping_cart, err := r.GetCartBySessionID(session_id)
	if err != nil {
		return error_messages.ErrNotExists
	}
	return r.updateCart(shopping_cart.ID, "session_id", new_session_id)
}

func (r *SQLiteDatabase) updateCart(id int64, column string, newval string) error {
	res, err := r.db.Exec("UPDATE shopping_cart SET "+column+" = ? WHERE id = ?", newval, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return error_messages.ErrUpdateFailed
	}

	return nil
}

/*******/
/* GET */
/*******/

// Return a user's ShoppingCart struct based on their session id.
func (r *SQLiteDatabase) GetCartBySessionID(session_id string) (*ShoppingCart, error) {
	return r.getCartByColumn("session_id", session_id)
}

// Return a user's ShoppingCart struct based on their payment intent id.
func (r *SQLiteDatabase) GetCartByPaymentIntentID(payment_intent_id string) (*ShoppingCart, error) {
	return r.getCartByColumn("payment_intent_id", payment_intent_id)
}

// Return a shopping cart struct based on a specific column
func (r *SQLiteDatabase) getCartByColumn(col_title string, col_val string) (*ShoppingCart, error) {
	row := r.db.QueryRow("SELECT * FROM shopping_cart WHERE "+col_title+" = ?", col_val)

	//fmt.Printf("Retrieving cart where %s == %s\n", col_title, col_val)
	var shopping_cart ShoppingCart
	if err := row.Scan(&shopping_cart.ID, &shopping_cart.SessionID, &shopping_cart.PaymentIntentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, error_messages.ErrNotExists
		}
		return nil, err
	}
	return &shopping_cart, nil
}

// Returns a slice of items in the user's cart
func (r *SQLiteDatabase) GetItemsBySessionID(session_id string) ([]CartItem, error) {
	cart, err := r.GetCartBySessionID(session_id)
	if err != nil {
		if err != error_messages.ErrNotExists {
			log.Printf("GetCartBySessionID: %v", err)
		}
		return nil, err
	}

	items, err := r.getItemsByShoppingCartID(cart.ID)

	return items, err
}

func (r *SQLiteDatabase) getItemsByShoppingCartID(id int64) ([]CartItem, error) {
	rows, err := r.db.Query("SELECT * FROM cart_item WHERE shopping_cart_id = ?", id)
	if err != nil {
		log.Printf("Error in getItemsByShoppingCartID(): %v\n", err)
		return nil, err
	}

	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ID, &item.ShoppingCartID, &item.Item, &item.Size, &item.Color); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, error_messages.ErrNotExists
			}
			return nil, err
		}

		items = append(items, item)
	}

	err = rows.Err()
	if err != nil {
		log.Printf("Error in getItemsByShoppingCartID(): %v\n", err)
		return nil, err
	}

	return items, nil
}

func (r *SQLiteDatabase) AllCarts() ([]ShoppingCart, error) {
	rows, err := r.db.Query("SELECT * FROM shopping_cart")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all []ShoppingCart
	for rows.Next() {
		var shopping_cart ShoppingCart
		if err := rows.Scan(&shopping_cart.ID, &shopping_cart.SessionID, &shopping_cart.PaymentIntentID); err != nil {
			return nil, err
		}
		all = append(all, shopping_cart)
	}
	return all, nil
}

/**********/
/* DELETE */
/**********/

func (r *SQLiteDatabase) DeleteCart(session_id string) error {
	res, err := r.db.Exec("DELETE FROM shopping_cart WHERE session_id = ?", session_id)
	err = r.checkDeleteError(res, err)
	return err
}

// res, err := r.db.Exec("DELETE FROM cart_item WHERE shopping_cart_id = ? AND item = ? AND size = ? AND color = ?", item.ShoppingCartID, item.Item, item.Size, item.Color)
func (r *SQLiteDatabase) DeleteItem(item CartItem) error {
	var id int64 = -1
	items, err := r.getItemsByShoppingCartID(item.ShoppingCartID)
	for _, cart_item := range items {
		if cart_item.Item == item.Item && cart_item.Size == item.Size && cart_item.Color == item.Color {
			id = cart_item.ID
		}
	}
	if err != nil {
		return err
	} else if id == -1 {
		return error_messages.ErrNotExists
	}

	res, err := r.db.Exec("DELETE FROM cart_item WHERE id = ?", id)
	err = r.checkDeleteError(res, err)
	return err
}

func (r *SQLiteDatabase) checkDeleteError(res sql.Result, err error) error {
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return error_messages.ErrDeleteFailed
	}

	return err
}
