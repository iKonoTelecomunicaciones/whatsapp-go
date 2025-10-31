package main

import (
	"encoding/json"
	"net/http"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog/hlog"
)

func receive(w http.ResponseWriter, r *http.Request) {
	// This endpoint is used to receive provision requests from Meta WhatsApp Cloud.
	// It checks if the request is valid and then processes the event accordingly.
	var body types.CloudEvent
	ctx := r.Context()
	err := json.NewDecoder(r.Body).Decode(&body)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error decoding request body")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Invalid request body",
		})
		return
	}

	hlog.FromRequest(r).Info().Msg("Received event from WhatsApp Cloud")
	hlog.FromRequest(r).Info().Interface("body", body).Msg("Event body: ")

	// Get the business id and the value of the event
	wb_business_id := body.Entry[0].ID
	wb_value := body.Entry[0].Changes[0].Value

	// Validate if the app is registered
	app_registered, err := whatsappConnector.DB.CloudRequest.SearchApp(ctx, wb_business_id, "", "")

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error while searching for whatsapp app")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Error while searching for whatsapp app",
		})
		return
	}

	if len(app_registered) == 0 {
		hlog.FromRequest(r).Warn().Msgf(
			"Ignoring event because the whatsapp_app [%s] is not registered.",
			wb_business_id,
		)
		// If the app is not registered, we return a 200 OK response to acknowledge the event
		// and avoid further processing.
		// This is important to prevent WhatsApp from retrying the event.
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Ignoring event because the whatsapp_app is not registered.",
		})
		return
	}

	//Validate if the event is a not a message.
	if wb_value.Messages == nil {
		hlog.FromRequest(r).Warn().Msgf(
			"Ignoring event because the integration type is not supported.",
		)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Integration type not supported. Only messages are supported.",
		})
	}

	handleErr := whatsappConnector.HandleCloudEvent(ctx, wb_business_id, wb_value, body)

	if handleErr != nil {
		hlog.FromRequest(r).Error().Interface(
			"error", handleErr,
		).Msg("Error while handling cloud event")
		jsonResponse(w, http.StatusOK, handleErr)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Message event processed successfully",
	})
	return
}

func verifyConnection(w http.ResponseWriter, r *http.Request) {
	// This endpoint is used to verify the connection with WhatsApp Cloud.
	// It checks the mode and token, and returns the challenge if they match.
	hlog.FromRequest(r).Info().Msg("Received verification request from WhatsApp Cloud")

	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" {
		hlog.FromRequest(r).Info().Msg("Mode is not 'subscribe'... returning error")
		hlog.FromRequest(r).Error().Msg("Invalid mode, it's not 'subscribe'")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Mode is not 'subscribe'",
		})
		return
	}

	if token != brmain.Config.Provisioning.SharedSecret {
		hlog.FromRequest(r).Info().Msg("Invalid verification token... returning error")
		hlog.FromRequest(r).Error().Msg("Invalid verification token")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Invalid verification token",
		})
		return
	}

	hlog.FromRequest(r).Info().Msg("Verification successful, returning challenge")
	// If the verification is successful, we return the challenge.
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}
