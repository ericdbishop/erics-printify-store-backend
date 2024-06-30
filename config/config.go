package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	PRINTIFY_API_TOKEN    = ""
	SHOP_ID               = 0
	STRIPE_SECRET         = ""
	STRIPE_WEBHOOK_SECRET = ""
	CSRF_AUTH_TOKEN       = ""
	LOGFILE               = ""
)

func InitConf() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file")
	}

	PRINTIFY_API_TOKEN = os.Getenv("PRINTIFY_API_TOKEN")

	SHOP_ID, err = strconv.Atoi(os.Getenv("SHOP_ID"))
	if err != nil {
		log.Fatal("SHOP_ID could not be converted to int")
	}

	STRIPE_SECRET = os.Getenv("STRIPE_SECRET")

	STRIPE_WEBHOOK_SECRET = os.Getenv("STRIPE_WEBHOOK_SECRET")

	CSRF_AUTH_TOKEN = os.Getenv("CSRF_AUTH_TOKEN")

	LOGFILE = os.Getenv("LOGFILE")
}
