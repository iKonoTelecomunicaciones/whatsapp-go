package cloudhandle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
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

func (whatsappClient *WhatsappCloudClient) GetMediaFromMeta(ctx context.Context, mediaID string) ([]byte, error) {
	log := whatsappClient.UserLogin.Log

	metaData := whatsappClient.GetMetaData(ctx)
	if metaData == nil {
		log.Error().Msg("User metadata not found")
		return nil, fmt.Errorf("user metadata not found")
	}

	baseURL := *whatsappClient.Main.Config.WhatsApp.CloudURL
	mediaURL := fmt.Sprintf("%s/%s?access_token=%s", baseURL, mediaID, metaData.PageAccessToken)

	log.Info().Str("media_url", mediaURL).Msg("Fetching media from Meta")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", metaData.PageAccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch media from Meta")
		return nil, fmt.Errorf("failed to fetch media from Meta: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error().Int("status_code", resp.StatusCode).Msg("Unexpected status code from Meta API")
		return nil, fmt.Errorf("unexpected status code %d from Meta API", resp.StatusCode)
	}

	var mediaResponse MediaResponse
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse)

	if err != nil {
		log.Error().Err(err).Msg("Failed to decode media response from Meta")
		return nil, fmt.Errorf("failed to decode media response: %w", err)
	}

	if mediaResponse.Error != nil && *mediaResponse.Error != "" {
		log.Error().Str("error", *mediaResponse.Error).Msg("Error in media response from Meta")
		return nil, fmt.Errorf("Meta API error: %s", mediaResponse.Error)
	}

	mediaReq, err := http.NewRequestWithContext(ctx, http.MethodGet, *mediaResponse.URL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create media content request")
		return nil, fmt.Errorf("failed to create media content request: %w", err)
	}

	mediaReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", metaData.PageAccessToken))
	mediaResp, err := http.DefaultClient.Do(mediaReq)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch media content")
		return nil, fmt.Errorf("failed to fetch media content: %w", err)
	}

	if mediaResp.StatusCode < 200 || mediaResp.StatusCode >= 300 {
		log.Error().Int("status_code", mediaResp.StatusCode).Msg("Unexpected status code from media URL")
		return nil, fmt.Errorf("unexpected status code %d from media URL", mediaResp.StatusCode)
	}

	mediaData, err := io.ReadAll(mediaResp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read media content")
		return nil, fmt.Errorf("failed to read media content: %w", err)
	}

	log.Info().Int("media_size", len(mediaData)).Msg("Media fetched from Meta successfully")
	return mediaData, nil
}

func (whatsappClient *WhatsappCloudClient) UploadMediaToSynapseV3(ctx context.Context, mediaData []byte, portal *bridgev2.Portal) (string, error) {
	log := whatsappClient.UserLogin.Log

	if len(mediaData) == 0 {
		log.Error().Msg("Media data is empty, cannot upload to Synapse")
		whatsappClient.sendMediaUploadFailedNotice(ctx, portal, "Media data is empty, cannot upload to Synapse")
		return "", fmt.Errorf("Media data is empty")
	}

	intent := whatsappClient.Main.Bridge.Bot

	contentType := "application/octet-stream"
	if len(mediaData) >= 4 {
		// Detect some common types by magic bytes
		switch {
		case mediaData[0] == 0xFF && mediaData[1] == 0xD8:
			contentType = "image/jpeg"
		case mediaData[0] == 0x89 && mediaData[1] == 0x50 && mediaData[2] == 0x4E && mediaData[3] == 0x47:
			contentType = "image/png"
		case mediaData[0] == 0x47 && mediaData[1] == 0x49 && mediaData[2] == 0x46:
			contentType = "image/gif"
		}
	}

	mxcURL, _, err := intent.UploadMedia(ctx, portal.MXID, mediaData, fmt.Sprintf("whatsapp-media-%d", len(mediaData)), contentType)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload media to Matrix")
		whatsappClient.sendMediaUploadFailedNotice(ctx, portal, fmt.Sprintf("Failed to upload media, error: %v", err))
		return "", fmt.Errorf("failed to upload media to Matrix: %w", err)
	}

	log.Info().Int("media_size", len(mediaData)).Str("mxc_url", string(mxcURL)).Msg("Media uploaded to Synapse successfully")
	return string(mxcURL), nil
}

func (whatsappClient *WhatsappCloudClient) sendMediaUploadFailedNotice(ctx context.Context, portal *bridgev2.Portal, errorMessage string) {
	log := whatsappClient.UserLogin.Log

	content := &event.Content{
		Parsed: &event.MessageEventContent{
			MsgType: event.MsgNotice,
			Body:    fmt.Sprintf("Failed to upload media from WhatsApp: %s", errorMessage),
		},
	}

	intent := whatsappClient.Main.Bridge.Bot
	_, err := intent.SendMessage(ctx, portal.MXID, event.EventMessage, content, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send media upload failure notice to portal")
	}
}
