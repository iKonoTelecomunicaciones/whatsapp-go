package cloudhandle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exmime"
)

type FailedMediaKeys struct {
	Key        []byte `json:"key"`
	Length     uint64 `json:"length"`
	Type       string `json:"type"`
	SHA256     []byte `json:"sha256"`
	EncSHA256  []byte `json:"enc_sha256"`
	DirectPath string `json:"direct_path,omitempty"`
	MimeType   string `json:"mime_type,omitempty"`
}

type PreparedMedia struct {
	Type                       event.Type `json:"type"`
	*event.MessageEventContent `json:"content"`
	Extra                      map[string]any    `json:"extra"`
	FailedKeys                 *FailedMediaKeys  `json:"whatsapp_media"`
	MentionedJID               []string          `json:"mentioned_jid,omitempty"`
	TypeDescription            string            `json:"type_description"`
	ContextInfo                *CloudMessageInfo `json:"context_info,omitempty"`
}

func prepareMediaMessage(rawMsg *types.CloudValue) *PreparedMedia {
	messageData := rawMsg.Messages[0]
	contact := rawMsg.Contacts[0]

	extraInfo := map[string]any{}
	data := &PreparedMedia{
		Type: event.EventMessage,
		MessageEventContent: &event.MessageEventContent{
			Info: &event.FileInfo{},
		},
		Extra: map[string]any{
			"info": extraInfo,
		},
	}

	switch messageData.Type {
	case "image":
		data.MsgType = event.MsgImage
		data.FileName = "image" + exmime.ExtensionFromMimetype(messageData.Image.MimeType)
		data.Body = ""
		if messageData.Image.Caption != nil {
			data.Body = *messageData.Image.Caption
		}
	default:
		panic(fmt.Errorf("unknown media message type %T", rawMsg))
	}

	data.ContextInfo = &CloudMessageInfo{
		ID:        string(messageData.ID),
		Sender:    string(messageData.From),
		Type:      string(messageData.Type),
		Timestamp: time.Now(),
		PushName:  string(contact.Profile.Name),
	}

	return data
}

// detectMimeType detects the MIME type of media data based on its content
func detectMimeType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// Check for common image types by magic bytes
	switch {
	case data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "image/gif"
	case bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 8 && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// getExtensionFromMimeType returns the file extension for a given MIME type
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "video/mp4":
		return "mp4"
	case "audio/mpeg":
		return "mp3"
	case "audio/ogg":
		return "ogg"
	default:
		// Try to extract extension from MIME type
		parts := strings.Split(mimeType, "/")
		if len(parts) == 2 {
			return parts[1]
		}
		return "bin"
	}
}

// getWhatsAppMediaType converts a MIME type to WhatsApp media type
func getWhatsAppMediaType(mimeType string) string {
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return "document"
	}

	return parts[0]
}

// MediaUploadResponse represents the response from WhatsApp Cloud API media upload
type MediaUploadResponse struct {
	ID string `json:"id"`
}

// uploadMedia uploads media to WhatsApp Cloud API
// This method should be called on a WhatsappCloudClient instance, not MessageConverter
// fileType should be the WhatsApp media category: "image", "video", "audio", or "document"
func (client *WhatsappCloudClient) uploadMedia(
	ctx context.Context,
	dataFile []byte,
	messagingProduct string,
	fileName string,
	fileType string,
) (*MediaUploadResponse, error) {
	log := zerolog.Ctx(ctx).With().Str("uploadMedia", fileName).Logger()

	// Detect the actual MIME type of the file content
	actualMimeType := detectMimeType(dataFile)

	log.Debug().
		Str("messaging_product", messagingProduct).
		Str("file_name", fileName).
		Str("file_type", fileType).
		Str("detected_mime_type", actualMimeType).
		Int("file_size", len(dataFile)).
		Msg("Uploading media to WhatsApp API")

	// Get metadata from the client
	metadata := client.GetMetaData(ctx)
	if metadata == nil {
		return nil, fmt.Errorf("user metadata not found")
	}

	// Set the URL to upload media to WhatsApp API
	uploadMediaURL := fmt.Sprintf(
		"%s/%s/%s/media",
		*client.Main.Config.WhatsApp.CloudURL,
		*client.Main.Config.WhatsApp.CloudVersion,
		metadata.BusinessPhoneID,
	)

	log.Debug().Str("upload_url", uploadMediaURL).Msg("Uploading media to WhatsApp API")

	// Create multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add messaging_product field
	err := writer.WriteField("messaging_product", messagingProduct)
	if err != nil {
		return nil, fmt.Errorf("failed to add messaging_product field: %w", err)
	}

	// Add type field
	err = writer.WriteField("type", fileType)
	if err != nil {
		return nil, fmt.Errorf("failed to add type field: %w", err)
	}

	// Add the media file with proper MIME type
	fileHeader := textproto.MIMEHeader{}
	fileHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	fileHeader.Set("Content-Type", actualMimeType)

	part, err := writer.CreatePart(fileHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file part: %w", err)
	}

	_, err = part.Write(dataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to write media data: %w", err)
	}

	// Close the multipart writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", uploadMediaURL, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+metadata.PageAccessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Unexpected status code from WhatsApp API")
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	// Parse response
	var uploadResp MediaUploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Info().Str("media_id", uploadResp.ID).Msg("Media uploaded to WhatsApp successfully")
	return &uploadResp, nil
}
