package webui

import (
	"encoding/json"
	"net/http"
)

func returnError(err error, w http.ResponseWriter) {
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(struct {
		Error error
	}{err})
}
