package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type response struct {
	Msg string `json:"msg"`
}

func ping(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response{Msg: "Pong"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func registerRoutes(r chi.Router) {
	r.Get("/ping", ping)
}
