package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog/hlog"
)

func receive(w http.ResponseWriter, r *http.Request) {
	// This endpoint is used to receive provision requests from Meta WhatsApp Cloud.
	// It checks if the request is valid and then processes the event accordingly.
	hlog.FromRequest(r).Info().Msg("Received event from WhatsApp Cloud")

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

	hlog.FromRequest(r).Info().Msgf(
		"Searching for user login with ID [%s]. AND bridge ID [%s]",
		wb_business_id, brmain.Bridge.DB.UserLogin.BridgeID,
	)

	userLogin, err := brmain.Bridge.GetExistingUserLoginByID(
		ctx, networkid.UserLoginID(wb_business_id),
	)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error while getting user login")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Error while getting user login",
		})
		return
	}

	if userLogin == nil {
		hlog.FromRequest(r).Error().Msg("User login not found for request")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "User login not found for request",
		})
		return
	}

	//Validate if the event is not a message.
	if wb_value.Messages == nil {
		hlog.FromRequest(r).Warn().Msgf(
			"Ignoring event because the integration type is not supported.",
		)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Integration type not supported. Only messages are supported.",
		})
		return
	}

	domain := brmain.Config.Homeserver.Domain
	userKey := waid.MakeUserKeyUsingBody(body, domain, brmain)
	portal, err := whatsappConnector.GetPortal(ctx, userLogin, brmain, userKey)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error while getting portal")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Error while getting portal",
		})
		return
	}

	if portal == nil {
		hlog.FromRequest(r).Error().Msg("Portal not found to handle remote event")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": "Portal not found to handle remote event",
		})
		return
	}

	wClient := whatsappConnector.GetWhatsappCloudClient(ctx, userLogin)
	err = wClient.HandleCloudMessage(ctx, body, portal)

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error while handling cloud message")
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"message": fmt.Sprintf("Error while handling cloud message: %s", err),
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Message event processed successfully",
	})
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
