package main

import (
	"fmt"
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog/hlog"
)

func get_userLogin(w http.ResponseWriter, r *http.Request) *bridgev2.UserLogin {
	// This function retrieves the user login from the request context.
	userLogin, failed := brmain.Matrix.Provisioning.GetExplicitLoginForRequest(w, r)

	if userLogin != nil || failed {
		return userLogin
	}

	// If the user login is not found in the cache, we can try to fetch it
	userLogin = brmain.Matrix.Provisioning.GetUser(r).GetDefaultLogin()

	return userLogin
}

func validateUserLogin(w http.ResponseWriter, r *http.Request) error {
	userLogin := get_userLogin(w, r)

	if userLogin == nil || userLogin.Metadata == nil {
		return nil
	}

	var metadata *waid.UserLoginMetadata
	var err error

	// Decode the user metadata from the user login
	switch v := userLogin.Metadata.(type) {
	case *waid.UserLoginMetadata:
		metadata = v
	default:
		hlog.FromRequest(r).Error().Interface("v", fmt.Sprintf("%T", v)).Msg("Error decoding user metadata")
		err = fmt.Errorf("Invalid user metadata type: %T", v)
	}

	if err != nil {
		hlog.FromRequest(r).Error().Err(err).Msg("Error decoding user metadata")
		return fmt.Errorf("Invalid user metadata type, please check the format and try again.")
	}

	if metadata.WabaID == "" {
		return nil
	}

	hlog.FromRequest(r).Warn().Msgf(
		"User login [%s] is already registered with WhatsApp Business Account ID [%s].",
		userLogin.UserMXID, metadata.WabaID,
	)
	return fmt.Errorf(
		"User login is already registered with WhatsApp Business Account ID [%s].",
		metadata.WabaID,
	)
}
