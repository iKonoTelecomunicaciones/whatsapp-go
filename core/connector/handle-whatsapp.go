package connector

import (
	"context"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
)

func (whatsappConnector *WhatsappCloudConnector) HandleCloudEvent(
	ctx context.Context, wb_business_id string, wb_value types.CloudValue, body types.CloudEvent,
) map[string]interface{} {
	log := zerolog.Ctx(ctx).With().Str("HandleCloudEvent", body.Object).Logger()

	// If the event is a message, we send a message event to matrix
	userLoginID := wb_value.Messages[0].ID

	portalKey := waid.MakePortalKey(wb_business_id)
	logins, err := whatsappConnector.Bridge.GetUserLoginsInPortal(ctx, portalKey)

	if err != nil {
		log.Error().Err(err).Msg("Error while getting user login by ID")

		return map[string]interface{}{
			"message": "Error while getting user login by ID %(login_id)s",
			"data": map[string]interface{}{
				"login_id": userLoginID,
			},
		}
	}

	if len(logins) == 0 {
		log.Warn().Msgf(
			"Ignoring event because the user login [%s] is not registered.",
			userLoginID,
		)
		return map[string]interface{}{
			"message": "Ignoring event because the user login is not registered.",
		}
	}

	login := logins[0]

	wClient := &WhatsappCloudClient{
		Main:      whatsappConnector,
		UserLogin: login,
	}

	wClient.HandleCloudMessage(ctx, body)
	return nil
}

func (whatsappClient *WhatsappCloudClient) HandleCloudMessage(
	ctx context.Context, event types.CloudEvent,
) {
	log := zerolog.Ctx(ctx).With().Str("HandleCloudMessage", event.Object).Logger()
	log.Info().Msgf("Received Whatsapp Cloud message event: %s", event)

	// Handle the converted cloud message
}
