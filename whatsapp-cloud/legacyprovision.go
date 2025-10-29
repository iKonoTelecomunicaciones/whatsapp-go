package main

import (
	"net/http"

	"go.mau.fi/util/exhttp"
)

func jsonResponse(w http.ResponseWriter, status int, detail map[string]interface{}) {
	exhttp.WriteJSONResponse(w, status, Response{Detail: detail})
}

func legacyProvReceive(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "This endpoint is not implemented yet",
	})
}

func legacyProvVerifyConnection(w http.ResponseWriter, r *http.Request) {
	//
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "This endpoint is not implemented yet",
	})
}
