package main

import (
	"fmt"
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/cloudhandle"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog/hlog"
)

func startRegistration(
	w http.ResponseWriter, r *http.Request, body types.CloudRegisterAppRequest,
) error {
	// Start the registration process for the WhatsApp app.
	// This function checks if the WhatsApp app is already registered and if not,
	// it creates a new WhatsApp app in the database and starts the login process.
	// It returns an error if the registration fails or if the app is already registered.
	user_id := r.URL.Query().Get("user_id")
	log := hlog.FromRequest(r)

	log.Debug().Msg("Checking if WhatsApp app is already registered for Waba ID")
	app_registered, err := whatsappConnector.DB.CloudRequest.SearchApp(
		r.Context(), body.WabaID, "", "",
	)

	if err != nil {
		log.Error().Err(err).Msg("Error while searching for whatsapp app")
		return fmt.Errorf("Error while searching for whatsapp app.")
	}

	if app_registered != nil && len(app_registered) > 0 {
		log.Warn().Msgf(
			"WhatsApp app [%s] is already registered for Waba ID [%s].",
			body.AppName, body.WabaID,
		)
		return fmt.Errorf("WhatsApp app is already registered for this Waba ID.")
	}

	log.Debug().Msg("Checking if WhatsApp app is already registered for App Phone ID")
	app_registered, err = whatsappConnector.DB.CloudRequest.SearchApp(
		r.Context(), "", body.AppPhoneID, "",
	)

	if err != nil {
		log.Error().Err(err).Msg("Error while searching for whatsapp app")
		return fmt.Errorf("Error while searching for whatsapp app.")
	}

	if app_registered != nil && len(app_registered) > 0 {
		log.Warn().Msgf(
			"WhatsApp app [%s] is already registered for Waba ID [%s].",
			body.AppName, body.WabaID,
		)
		return fmt.Errorf("WhatsApp app is already registered for this Waba ID.")
	}

	log.Info().Msg("Creating new WhatsApp app in the database")
	new_app, err := whatsappConnector.DB.CloudRequest.CreateApp(
		r.Context(), body.AppName, user_id,
		body.WabaID, body.AppPhoneID, body.AccessToken,
	)

	if err != nil {
		log.Error().Err(err).Msg("Error while creating whatsapp app")
		return fmt.Errorf("Error while creating whatsapp app.")
	}

	if new_app == nil {
		log.Error().Msg("Failed to register WhatsApp app, no app returned")
		return fmt.Errorf("Failed to register WhatsApp app, no app returned.")
	}

	log.Info().Interface("BusinessID", new_app.BusinessID).Msg(
		"WhatsApp app registered successfully",
	)

	// Update the user login metadata with the new WhatsApp app details
	log.Info().Msg("Getting user from the request context")
	user := brmain.Matrix.Provisioning.GetUser(r)
	user.ManagementRoom = id.RoomID(body.NoticeRoom)
	user.User.ManagementRoom = id.RoomID(body.NoticeRoom)

	err = user.Save(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Error saving user metadata")
		return fmt.Errorf("Error saving user metadata.")
	}

	log.Info().Msg("Creating login for WhatsApp app")
	login, err := whatsappConnector.CreateAppLogin(r.Context(), user, body)
	if err != nil {
		log.Err(err).Msg("Failed to create login")
		return fmt.Errorf("Failed to create login for WhatsApp app.")
	}

	log.Debug().Msg("Creating login for WhatsApp app")
	waLogin := login.(*cloudhandle.WaCloudLogin)
	waLogin.Timezone = r.URL.Query().Get("tz")

	_, err = waLogin.Start(r.Context())
	if err != nil {
		log.Err(err).Msg("Failed to start login")
		return fmt.Errorf("Failed to start login for WhatsApp app.")
	}

	return nil
}
