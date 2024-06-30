package main

import (
	"log"
	"net/http"
	"os"
	"server/api/external"
	"server/api/site"
	"server/cart"
	"server/config"

	"github.com/gorilla/csrf"
)

func main() {
	config.InitConf()

	// open log file
	logFile, err := os.OpenFile(config.LOGFILE, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Panic(err)
	}
	defer logFile.Close()
	// set log out put
	log.SetOutput(logFile)

	CSRF := csrf.Protect(
		[]byte(config.CSRF_AUTH_TOKEN),
		csrf.SameSite(csrf.SameSiteStrictMode),
		//csrf.Secure(false), // REMOVE IN PRODUCTION
	)

	mux := http.NewServeMux()
	webhook_mux := http.NewServeMux()

	cart.InitDatabase()
	site.InitHandlers(mux)
	external.InitHandlers(mux)
	external.InitWebhook(webhook_mux)
	external.InitPrintifyClient(config.PRINTIFY_API_TOKEN, config.SHOP_ID)

	log.Printf("Beginning to listen on ports 4242 and 4343\n")
	go http.ListenAndServe("localhost:4343", webhook_mux)
	err = http.ListenAndServe("localhost:4242", CSRF(mux))
	log.Fatal(err)
}
