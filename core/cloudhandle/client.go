package cloudhandle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog"
)

var mediaTypes = []string{"image", "video", "audio", "document"}
var validMessagesTypes = append([]string{"text"}, mediaTypes...)

// Connect handles establishing the connection for the WhatsApp client.
// Currently, this function is a stub and simply returns to satisfy the interface.
func (whatsappClient *WhatsappCloudClient) Connect(ctx context.Context) {
	// Method to establish a connection to the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

// Disconnect handles terminating the connection for the WhatsApp client.
// Currently, this function is a stub and simply returns to satisfy the interface.
func (whatsappClient *WhatsappCloudClient) Disconnect() {
	// Method to disconnect the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

// GetCapabilities returns the features and capabilities of a specific room (portal).
// This allows the bridge to know what types of actions and messages are supported in that room.
func (whatsappClient *WhatsappCloudClient) GetCapabilities(
	ctx context.Context,
	portal *bridgev2.Portal,
) *event.RoomFeatures {
	return &event.RoomFeatures{
		ID: string(portal.ID),
	}
}

// HandleMatrixMessage processes a message coming from Matrix.
// It converts the Matrix message to a WhatsApp-compatible format and sends it.
func (whatsappClient *WhatsappCloudClient) HandleMatrixMessage(
	ctx context.Context,
	msg *bridgev2.MatrixMessage,
) (*bridgev2.MatrixMessageResponse, error) {
	whatsappMessage, err := whatsappClient.Main.MsgConv.ToWhatsApp(
		ctx,
		msg.Event,
		msg.Content,
		msg.ReplyTo,
		msg.ThreadRoot,
		msg.Portal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}
	return whatsappClient.handleConvertedMatrixMessage(ctx, whatsappMessage)
}

// IsThisUser checks if a Matrix User ID corresponds to this WhatsApp client.
// This is useful for preventing message loops and identifying the bot's own messages.
func (whatsappClient *WhatsappCloudClient) IsThisUser(
	_ context.Context, userID networkid.UserID,
) bool {
	return userID == networkid.UserID(whatsappClient.UserLogin.ID)
}

// GetUserInfo gets the information of a user (ghost) on WhatsApp.
// For now, it returns an empty struct to satisfy the interface.
func (whatsappClient *WhatsappCloudClient) GetUserInfo(
	ctx context.Context, ghost *bridgev2.Ghost,
) (*bridgev2.UserInfo, error) {
	// Method to get user information from the whatsappClient.
	// For now, we'll return an empty UserInfo to satisfy the interface.

	return &bridgev2.UserInfo{}, nil
}

// getChatInfo is an internal function to get chat information from a portal ID.
func (whatsappClient *WhatsappCloudClient) getChatInfo(
	ctx context.Context, portalID networkid.PortalID) (wrapped *bridgev2.ChatInfo, err error,
) {
	if portalID == "" {
		return nil, fmt.Errorf("portalID cannot be empty")
	}

	Name := fmt.Sprintf("WhatsApp Cloud Portal: %s", portalID)

	wrapped = &bridgev2.ChatInfo{
		Name: &Name,
	}

	return wrapped, nil
}

// GetChatInfo gets information about a specific chat (portal).
// It uses the internal getChatInfo function to perform the task.
func (whatsappClient *WhatsappCloudClient) GetChatInfo(
	ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error,
) {
	if portal.ID == "" {
		return nil, fmt.Errorf("portalID cannot be empty")
	}
	return whatsappClient.getChatInfo(ctx, portal.ID)
}

// IsLoggedIn checks if the client is logged in and has an active connection.
func (whatsappClient *WhatsappCloudClient) IsLoggedIn() bool {
	return whatsappClient.UserLogin != nil && whatsappClient.UserLogin.Client.IsLoggedIn()
}

// LogoutRemote handles logging out the user from the remote WhatsApp service.
func (whatsappClient *WhatsappCloudClient) LogoutRemote(ctx context.Context) {
	// Method to log out the user from the remote service.
	if cli := whatsappClient.UserLogin.Client; cli != nil {
		cli.Disconnect()
	}
	whatsappClient.UserLogin.Client = nil
}

// HandleCloudMessage processes an incoming message from the WhatsApp Cloud API.
// It identifies the message type, converts it to an internal event, and queues it for processing.
func (whatsappClient *WhatsappCloudClient) HandleCloudMessage(
	ctx context.Context, event types.CloudEvent, portal *bridgev2.Portal,
) error {
	log := zerolog.Ctx(ctx).With().Str("HandleCloudMessage", event.Object).Logger()
	log.Info().Msgf("Received Whatsapp Cloud message event: %s", event)

	if len(event.Entry) > 1 {
		log.Warn().Msg("Ignoring event because it contains multiple entries")
		return fmt.Errorf("ignoring event because it contains multiple entries")
	}

	eventEntry := event.Entry[0]

	if len(eventEntry.Changes) > 1 {
		log.Warn().Msg("Ignoring event because it contains multiple changes")
		return fmt.Errorf("ignoring event because it contains multiple changes")
	}

	eventChange := eventEntry.Changes[0]

	if eventChange.Value.Messages == nil || len(eventChange.Value.Messages) == 0 {
		log.Warn().Msg("Ignoring event because it contains no messages")
		return fmt.Errorf("ignoring event because it contains no messages")
	}

	messages := eventChange.Value.Messages

	for _, messageData := range messages {
		log.Info().Msgf(
			"Processing message ID: %s of type: %s",
			messageData.ID, messageData.Type,
		)

		if messageData.ID == "" {
			log.Warn().Msg("Ignoring event because the message data is empty")
			return fmt.Errorf("ignoring event because the message data is empty")
		}

		messageType := messageData.Type
		messageID := messageData.ID

		var eventToQueue bridgev2.RemoteEvent
		messageInfo := CloudMessageInfo{
			ID:     messageID,
			Type:   messageType,
			Sender: messageData.From,
			MessageSource: MessageSource{
				Chat:           string(portal.ID),
				Sender:         messageData.From,
				IsFromMe:       false,
				IsGroup:        false,
				AddressingMode: "pn",
			},
		}

		eventToQueue = &WAMessageEvent{
			MessageInfoWrapper: &MessageInfoWrapper{
				Info:           messageInfo,
				whatsappClient: whatsappClient,
			},
			Message:  eventChange.Value,
			MsgEvent: event,
		}

		log.Error().Msgf("messageType: %v", messageType)

		if !slices.Contains(validMessagesTypes, messageType) {
			log.Warn().Msgf("Unsupported message type: %s", messageType)
			return fmt.Errorf("unsupported message type: %s", messageType)
		}

		log.Info().Msgf("Queued event for processing: %s", messageID)
		switch {
		case messageType == "text":
			whatsappClient.QueueEvent(ctx, eventToQueue, messageData, portal)
		case slices.Contains(mediaTypes, messageType):
			whatsappClient.QueueMediaEvent(ctx, eventToQueue, messageData, portal)
		default:
			log.Warn().Msgf("Ignoring unsupported message type: %s", messageType)
			return fmt.Errorf("ignoring unsupported message type: %s", messageType)
		}

		log.Info().Msg("Successfully handled cloud message event")

	}

	// Return nil to indicate successful handling
	return nil
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
