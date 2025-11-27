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
	case "video":
		if message.Video == nil || message.Video.ID == "" {
			log.Warn().Msg("Video data is empty or Video ID is empty in the message")
			return
		}
		media_id = &message.Video.ID
	case "audio":
		if message.Audio == nil || message.Audio.ID == "" {
			log.Warn().Msg("Audio data is empty or Audio ID is empty in the message")
			return
		}
		media_id = &message.Audio.ID
	case "document":
		if message.Document == nil || message.Document.ID == "" {
			log.Warn().Msg("Document data is empty or Document ID is empty in the message")
			return
		}
		media_id = &message.Document.ID
	case "sticker":
		if message.Sticker == nil || message.Sticker.ID == "" {
			log.Warn().Msg("Sticker data is empty or Sticker ID is empty in the message")
			return
		}
		media_id = &message.Sticker.ID
	default:
		log.Warn().Str("message_type", message.Type).Msg("Unsupported media type")
		return
	}

	mediaData, err := whatsappClient.GetMediaFromMeta(ctx, *media_id)
	if err != nil {
		log.Error().Err(err).Str("media_type", message.Type).Msg("Failed to fetch media from Meta")
		errorMessage := fmt.Sprintf("Error getting media from Meta, %v", err)
		whatsappClient.sendMediaUploadFailedNotice(ctx, portal, errorMessage)
		return
	}

	mxcURL, err := whatsappClient.UploadMediaToSynapseV3(ctx, mediaData, portal)
	if err != nil {
		log.Error().Err(err).Str("media_type", message.Type).Msg("Failed to upload media to Synapse")
		return
	}

	originalEvent := event.(*WAMessageEvent)
	originalEvent.preuploadedMXC = mxcURL

	// Create the message with the appropriate media type
	cloudMessage := types.CloudMessage{
		From:      message.From,
		ID:        message.ID,
		Type:      message.Type,
		TimeStamp: message.TimeStamp,
	}

	// Set the appropriate media field based on message type
	switch message.Type {
	case "image":
		cloudMessage.Image = &types.ImageCloud{
			ID:       message.Image.ID,
			MimeType: message.Image.MimeType,
			SHA256:   message.Image.SHA256,
			Caption:  message.Image.Caption,
		}
	case "sticker":
		cloudMessage.Sticker = &types.StickerCloud{
			ID:       message.Sticker.ID,
			MimeType: message.Sticker.MimeType,
			SHA256:   message.Sticker.SHA256,
			Animated: message.Sticker.Animated,
		}
	case "video":
		cloudMessage.Video = &types.VideoCloud{
			ID:       message.Video.ID,
			MimeType: message.Video.MimeType,
			SHA256:   message.Video.SHA256,
			Caption:  message.Video.Caption,
		}
	case "audio":
		cloudMessage.Audio = &types.AudioCloud{
			ID:       message.Audio.ID,
			MimeType: message.Audio.MimeType,
			SHA256:   message.Audio.SHA256,
			Voice:    message.Audio.Voice,
		}
	case "document":
		cloudMessage.Document = &types.DocumentCloud{
			ID:       message.Document.ID,
			MimeType: message.Document.MimeType,
			SHA256:   message.Document.SHA256,
			FileName: message.Document.FileName,
		}
	}

	originalEvent.Message = types.CloudValue{
		MessagingProduct: originalEvent.Message.MessagingProduct,
		Metadata:         originalEvent.Message.Metadata,
		Contacts:         originalEvent.Message.Contacts,
		Messages:         []types.CloudMessage{cloudMessage},
	}

	log.Info().Str(
		"mxc_url", mxcURL,
	).Str("media_type", message.Type).Msg("Media processed and uploaded successfully, preparing to enqueue event")

	whatsappClient.UserLogin.QueueRemoteEvent(originalEvent)
}
