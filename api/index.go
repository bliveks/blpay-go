package handler

import (
	"net/http"

	"github.com/bliveks/blpay-go/app"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	app.Handler(w, r)
}
