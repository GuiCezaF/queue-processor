package routes

import (
	"encoding/json"
	"net/http"
)

type sendResponse struct {
	Msg string `json:"msg"`
}

func ping(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(sendResponse{Msg: "Pong"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
