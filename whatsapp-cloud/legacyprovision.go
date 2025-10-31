package main

import (
	"encoding/json"
	"net/http"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog/hlog"
)

func registerApp(w http.ResponseWriter, r *http.Request) {
	// Check if the user is already registered. This acd user can be registered because the
	// bridge registers the acd user when it listens that the acd user is invited to the
	// control room
	err := validateUserLogin(w, r)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("User login validation failed")
		jsonResponse(w, http.StatusNotAcceptable, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	// This endpoint is used to register a WhatsApp app with the bridge.
	// It checks if the request is valid and then processes the registration accordingly.
	var body types.CloudRegisterAppRequest
	err = json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error decoding request body")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Invalid request body, please check the format and try again.",
		})
		return
	}

	if body.AppName == "" || body.WabaID == "" || body.AppPhoneID == "" || body.AccessToken == "" {
		hlog.FromRequest(r).Error().Msg("Missing required fields in request body")
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Missing required fields: app_name, waba_id, app_phone_id, access_token.",
		})
		return
	}

	if body.WabaID == body.AppPhoneID {
		hlog.FromRequest(r).Error().Msg("WABA ID and App Phone ID cannot be the same")
		jsonResponse(w, http.StatusBadRequest, map[string]interface{}{
			"message": "WABA ID and App Phone ID cannot be the same.",
		})
		return
	}

	err = startRegistration(w, r, body)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error starting registration process")
		jsonResponse(w, http.StatusInternalServerError, map[string]interface{}{
			"message": err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "WhatsApp app registered successfully",
	})
	return
}
