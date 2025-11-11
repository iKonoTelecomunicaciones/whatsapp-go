package cloudhandle

import (
	"context"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
)

func (whatsappClient *WhatsappCloudClient) QueueEvent(
	ctx context.Context, event bridgev2.RemoteEvent, portal *bridgev2.Portal,
) {
	log := whatsappClient.UserLogin.Log

	if portal == nil {
		log.Warn().
			Interface("portal_key", event).
			Msg("Portal not found to handle remote event")
		return
	}

	log.Info().Interface("event", event).
		Msgf("Handling remote event in portal %s", portal.PortalKey.ID)

	whatsappClient.UserLogin.QueueRemoteEvent(event)
}
