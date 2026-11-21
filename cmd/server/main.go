package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bliveks/blpay-go/app"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("BlPay listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, app.HandlerFunc()))
}
