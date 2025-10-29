package connector

import (
	"context"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
)

func (whatsappClient *WhatsappCloudClient) handleCloudMessage(
	ctx context.Context, event types.CloudEvent,
) {
	// Handle the converted cloud message
}
