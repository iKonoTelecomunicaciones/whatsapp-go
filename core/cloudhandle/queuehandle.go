package cloudhandle

import (
	"context"
	"fmt"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
)

type MediaResponse struct {
	ID               *string `json:"id"`
	MessagingProduct *string `json:"messaging_product"`
	URL              *string `json:"url"`
	MimeType         *string `json:"mime_type"`
	Hash             *string `json:"hash"`
	FileSize         *int    `json:"file_size"`
	Error            *string `json:"error"`
}

func (whatsappClient *WhatsappCloudClient) QueueEvent(
	ctx context.Context,
	event bridgev2.RemoteEvent,
	message types.CloudMessage,
	portal *bridgev2.Portal,
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

	event.(*WAMessageEvent).parsedMessageType = message.Text.Body

	whatsappClient.UserLogin.QueueRemoteEvent(event)
}

func (whatsappClient *WhatsappCloudClient) QueueMediaEvent(
	ctx context.Context,
	event bridgev2.RemoteEvent,
	message types.CloudMessage,
	portal *bridgev2.Portal,
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

	var media_id *string

	switch message.Type {
	case "image":
		if message.Image == nil || message.Image.ID == "" {
			log.Warn().Msg("Image data is empty or Image ID is empty in the message")
			return
		}
		media_id = &message.Image.ID

	default:
		log.Warn().Str("message_type", message.Type).Msg("Unsupported media type")
		return
	}

	mediaData, err := whatsappClient.GetMediaFromMeta(ctx, *media_id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch image from Meta")
		errorMessage := fmt.Sprintf("Error getting media from Meta, %v", err)
		whatsappClient.sendMediaUploadFailedNotice(ctx, portal, errorMessage)
		return
	}

	mxcURL, err := whatsappClient.UploadMediaToSynapseV3(ctx, mediaData, portal)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload image to Synapse")
		return
	}

	originalEvent := event.(*WAMessageEvent)
	originalEvent.preuploadedMXC = mxcURL
	originalEvent.Message = types.CloudValue{
		MessagingProduct: originalEvent.Message.MessagingProduct,
		Metadata:         originalEvent.Message.Metadata,
		Contacts:         originalEvent.Message.Contacts,
		Messages: []types.CloudMessage{
			{
				From:      message.From,
				ID:        message.ID,
				Type:      message.Type,
				TimeStamp: message.TimeStamp,
				Image: &types.ImageCloud{
					ID:       message.Image.ID,
					MimeType: message.Image.MimeType,
					SHA256:   message.Image.SHA256,
					Caption:  message.Image.Caption,
				},
			},
		},
	}

	log.Info().Str(
		"mxc_url", mxcURL,
	).Msg("Image processed and uploaded successfully, preparing to enqueue event")

	whatsappClient.UserLogin.QueueRemoteEvent(originalEvent)
	return
}
