package cloudhandle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
)

// uploadMediaToWhatsApp uploads media to WhatsApp Cloud API and returns the media ID
func (whatsappClient *WhatsappCloudClient) uploadMediaToWhatsApp(
	ctx context.Context, mxcURL id.ContentURIString, info *event.FileInfo,
) (string, error) {
	log := zerolog.Ctx(ctx).With().Str("uploadMediaToWhatsApp", string(mxcURL)).Logger()

	if mxcURL == "" {
		return "", fmt.Errorf("mxc URL is empty")
	}

	// Download media from Matrix
	mediaData, err := whatsappClient.Main.Bridge.Bot.DownloadMedia(ctx, mxcURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to download media from Matrix")
		return "", fmt.Errorf("failed to download media from Matrix: %w", err)
	}

	// Detect MIME type and generate filename
	var mediaType string
	if info != nil && info.MimeType != "" {
		mediaType = info.MimeType
	} else {
		mediaType = detectMimeType(mediaData)
	}

	ext := getExtensionFromMimeType(mediaType)
	mediaName := fmt.Sprintf("media.%s", ext)

	// Convert MIME type to WhatsApp media type
	whatsappMediaType := getWhatsAppMediaType(mediaType)

	log.Debug().
		Str("media_name", mediaName).
		Str("media_type", mediaType).
		Str("whatsapp_media_type", whatsappMediaType).
		Int("media_size", len(mediaData)).
		Msg("Downloaded media from Matrix, uploading to WhatsApp")

	// Upload using our new uploadMedia function
	response, err := whatsappClient.uploadMedia(ctx, mediaData, "whatsapp", mediaName, whatsappMediaType)
	if err != nil {
		return "", fmt.Errorf("failed to upload media to WhatsApp: %w", err)
	}

	return response.ID, nil
}

// SendMessage sends a message to a specific WhatsApp user.
func (whatsappClient *WhatsappCloudClient) SendMessage(
	ctx context.Context, msg *bridgev2.MatrixMessage, messageType event.MessageType,
) (string, error) {
	log := zerolog.Ctx(ctx).With().Str("SendMessage", string(msg.Event.ID)).Logger()

	metadata := whatsappClient.GetMetaData(ctx)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + metadata.PageAccessToken,
	}

	sendMessageURL := fmt.Sprintf(
		"%s/%s/%s/messages",
		*whatsappClient.Main.Config.WhatsApp.CloudURL,
		*whatsappClient.Main.Config.WhatsApp.CloudVersion,
		metadata.BusinessPhoneID,
	)

	var messageData map[string]interface{}
	var cloudMessageType string

	switch messageType {
	case event.MsgText:
		cloudMessageType = "text"
		messageData = map[string]interface{}{
			"preview_url": false,
			"body":        msg.Content.Body,
		}

	case event.MsgImage:
		cloudMessageType = "image"
		// We need to upload the media to WhatsApp first, then use the ID
		mediaID, err := whatsappClient.uploadMediaToWhatsApp(ctx, msg.Content.URL, msg.Content.Info)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload image to WhatsApp")
			whatsappClient.sendMediaUploadFailedNotice(
				ctx, msg.Portal, fmt.Sprintf("Failed to upload image to WhatsApp: %v", err),
			)
			return "", fmt.Errorf("failed to upload image to WhatsApp: %w", err)
		}

		messageData = map[string]interface{}{
			"id": mediaID,
		}

		// Add caption if present
		if msg.Content.Body != "" {
			messageData["caption"] = msg.Content.Body
		}

	case event.MsgVideo:
		cloudMessageType = "video"
		mediaID, err := whatsappClient.uploadMediaToWhatsApp(ctx, msg.Content.URL, msg.Content.Info)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload video to WhatsApp")
			return "", fmt.Errorf("failed to upload video to WhatsApp: %w", err)
		}

		messageData = map[string]interface{}{
			"id": mediaID,
		}

		if msg.Content.Body != "" {
			messageData["caption"] = msg.Content.Body
		}

	case event.MsgAudio:
		cloudMessageType = "audio"
		mediaID, err := whatsappClient.uploadMediaToWhatsApp(ctx, msg.Content.URL, msg.Content.Info)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload audio to WhatsApp")
			return "", fmt.Errorf("failed to upload audio to WhatsApp: %w", err)
		}

		messageData = map[string]interface{}{
			"id": mediaID,
		}

	case event.MsgFile:
		cloudMessageType = "document"
		mediaID, err := whatsappClient.uploadMediaToWhatsApp(ctx, msg.Content.URL, msg.Content.Info)
		if err != nil {
			log.Error().Err(err).Msg("Failed to upload document to WhatsApp")
			return "", fmt.Errorf("failed to upload document to WhatsApp: %w", err)
		}

		messageData = map[string]interface{}{
			"id": mediaID,
		}

		if msg.Content.Body != "" {
			messageData["caption"] = msg.Content.Body
		}

		// Set filename if available
		if msg.Content.FileName != "" {
			messageData["filename"] = msg.Content.FileName
		}

	default:
		log.Error().Msgf("Unsupported message type: %s", messageType)
		return "", fmt.Errorf("unsupported message type: %s", messageType)
	}

	dataToSend := map[string]interface{}{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.Portal.Receiver,
		"type":              cloudMessageType,
		cloudMessageType:    messageData,
	}

	log.Debug().Interface("dataToSend", dataToSend).
		Msgf("Sending message to WhatsApp to %s", msg.Portal.Receiver)

	jsonData, err := json.Marshal(dataToSend)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal message data to JSON")
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, sendMessageURL, bytes.NewReader(jsonData))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request")
		return "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	} else if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return "", fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var respData types.CloudMessageResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w, status: %d", err, resp.StatusCode)
	}

	log.Debug().Msgf("Message sent, status code: %d, response: %+v", resp.StatusCode, respData)
	return respData.Messages[0].ID, nil
}

// handleConvertedMatrixMessage takes a message that has been converted from a Matrix event
// and sends it to WhatsApp. This is currently a placeholder and needs to be implemented.
func (whatsappClient *WhatsappCloudClient) handleConvertedMatrixMessage(
	ctx context.Context,
	msg *bridgev2.MatrixMessage,
) (*bridgev2.MatrixMessageResponse, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	if msg.Event == nil {
		return nil, fmt.Errorf("message event is nil")
	}

	log := zerolog.Ctx(ctx).With().Str("handleConvertedMatrixMessage", string(msg.Event.ID)).Logger()

	log.Debug().Interface("Message", msg.Event).Msgf("Handle Matrix message %s", msg.Event.ID)

	if msg.Portal == nil {
		return nil, fmt.Errorf("failed to get portal from context")
	}

	chatJID, err := waid.ParsePortalID(msg.Portal.ID)
	if err != nil {
		return nil, err
	}

	// Support text and media messages
	supportedTypes := []event.MessageType{
		event.MsgText,
		event.MsgImage,
		event.MsgVideo,
		event.MsgAudio,
		event.MsgFile,
	}

	supported := false
	for _, msgType := range supportedTypes {
		if msg.Content.MsgType == msgType {
			supported = true
			break
		}
	}

	if !supported {
		log.Error().Msgf("Unsupported message type: %s", msg.Content.MsgType)
		return nil, fmt.Errorf("unsupported message type: %s", msg.Content.MsgType)
	}

	resp, err := whatsappClient.SendMessage(ctx, msg, msg.Content.MsgType)
	if err != nil {
		return nil, err
	}

	wrappedMsgID := waid.MakeMessageID(chatJID, string(msg.Event.Sender), resp)
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:        wrappedMsgID,
			SenderID:  networkid.UserID(msg.Event.Sender),
			Timestamp: time.Now(),
		},
		StreamOrder:   time.Now().Unix(),
		RemovePending: networkid.TransactionID(wrappedMsgID),
	}, nil
}
