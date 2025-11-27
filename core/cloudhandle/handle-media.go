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
	case "video":
		data.MsgType = event.MsgVideo
		data.FileName = "video" + exmime.ExtensionFromMimetype(messageData.Video.MimeType)
		data.Body = ""
		if messageData.Video.Caption != nil {
			data.Body = *messageData.Video.Caption
		}
	case "audio":
		data.MsgType = event.MsgAudio
		data.FileName = "audio" + exmime.ExtensionFromMimetype(messageData.Audio.MimeType)
		data.Body = ""
		// Audio messages don't typically have captions in WhatsApp
	case "document":
		data.MsgType = event.MsgFile
		// Use the original filename if available, otherwise generate one
		if messageData.Document.FileName != "" {
			data.FileName = messageData.Document.FileName
		} else {
			data.FileName = "document" + exmime.ExtensionFromMimetype(messageData.Document.MimeType)
		}
		data.Body = messageData.Document.FileName // Use filename as body for documents
	case "sticker":
		data.MsgType = event.MsgImage // Treat stickers as images in Matrix
		data.FileName = "sticker" + exmime.ExtensionFromMimetype(messageData.Sticker.MimeType)
		data.Body = ""
	default:
		panic(fmt.Errorf("unknown media message type %s", messageData.Type))
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
	// Images
	case data[0] == 0xFF && data[1] == 0xD8:
		return "image/jpeg"
	case data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47:
		return "image/png"
	case data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46:
		return "image/gif"
	case bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 8 && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	case data[0] == 0x42 && data[1] == 0x4D:
		return "image/bmp"
	case bytes.HasPrefix(data, []byte("\x00\x00\x01\x00")):
		return "image/x-icon"
	case bytes.HasPrefix(data, []byte("MM\x00\x2A")) || bytes.HasPrefix(data, []byte("II\x2A\x00")):
		return "image/tiff"

	// Videos
	case bytes.HasPrefix(data, []byte("\x00\x00\x00\x18ftypmp4")) || bytes.HasPrefix(data, []byte("\x00\x00\x00\x20ftypmp4")) || bytes.HasPrefix(data, []byte("\x00\x00\x00 ftypisom")):
		return "video/mp4"
	case bytes.HasPrefix(data, []byte("ftypisom")) || bytes.HasPrefix(data, []byte("ftypmp42")):
		return "video/mp4"
	case bytes.HasPrefix(data, []byte("\x1A\x45\xDF\xA3")):
		return "video/webm"
	case bytes.HasPrefix(data, []byte("FLV")):
		return "video/x-flv"
	case bytes.HasPrefix(data, []byte("\x00\x00\x01\xBA")) || bytes.HasPrefix(data, []byte("\x00\x00\x01\xB3")):
		return "video/mpeg"
	case bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 8 && bytes.Equal(data[8:12], []byte("AVI ")):
		return "video/x-msvideo"
	case bytes.HasPrefix(data, []byte("\x30\x26\xB2\x75\x8E\x66\xCF\x11")):
		return "video/x-ms-wmv"
	case bytes.HasPrefix(data, []byte("ftyp3gp")):
		return "video/3gpp"

	// Audio
	case bytes.HasPrefix(data, []byte("ID3")) || (data[0] == 0xFF && (data[1]&0xE0) == 0xE0):
		return "audio/mpeg"
	case bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 8 && bytes.Equal(data[8:12], []byte("WAVE")):
		return "audio/wav"
	case bytes.HasPrefix(data, []byte("OggS")):
		return "audio/ogg"
	case bytes.HasPrefix(data, []byte("fLaC")):
		return "audio/flac"
	case bytes.HasPrefix(data, []byte("\xFF\xFB")) || bytes.HasPrefix(data, []byte("\xFF\xF3")) || bytes.HasPrefix(data, []byte("\xFF\xF2")):
		return "audio/mpeg"
	case bytes.HasPrefix(data, []byte("FORM")) && len(data) > 8 && bytes.Equal(data[8:12], []byte("AIFF")):
		return "audio/aiff"
	case bytes.HasPrefix(data, []byte("\x30\x26\xB2\x75\x8E\x66\xCF\x11")):
		return "audio/x-ms-wma"
	case bytes.HasPrefix(data, []byte("MAC ")):
		return "audio/x-ape"

	// Documents
	case bytes.HasPrefix(data, []byte("%PDF")):
		return "application/pdf"
	case bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06")) || bytes.HasPrefix(data, []byte("PK\x07\x08")):
		// Could be ZIP, DOCX, XLSX, PPTX, etc. - we'll return zip and let the application determine
		return "application/zip"
	case bytes.HasPrefix(data, []byte("\xD0\xCF\x11\xE0\xA1\xB1\x1A\xE1")):
		return "application/vnd.ms-office" // Old Office formats (DOC, XLS, PPT)
	case bytes.HasPrefix(data, []byte("{\\rtf")):
		return "application/rtf"
	case bytes.HasPrefix(data, []byte("<?xml")) || bytes.HasPrefix(data, []byte("\xEF\xBB\xBF<?xml")):
		return "text/xml"
	case bytes.HasPrefix(data, []byte("<!DOCTYPE html")) || bytes.HasPrefix(data, []byte("<html")):
		return "text/html"
	case bytes.HasPrefix(data, []byte("BZh")):
		return "application/x-bzip2"
	case bytes.HasPrefix(data, []byte("\x1f\x8b\x08")):
		return "application/gzip"
	case bytes.HasPrefix(data, []byte("Rar!")):
		return "application/x-rar-compressed"
	case bytes.HasPrefix(data, []byte("7z\xBC\xAF\x27\x1C")):
		return "application/x-7z-compressed"

	default:
		return "application/octet-stream"
	}
}

// getExtensionFromMimeType returns the file extension for a given MIME type
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	// Images
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/bmp":
		return "bmp"
	case "image/x-icon":
		return "ico"
	case "image/tiff":
		return "tiff"

	// Videos
	case "video/mp4":
		return "mp4"
	case "video/webm":
		return "webm"
	case "video/x-flv":
		return "flv"
	case "video/mpeg":
		return "mpg"
	case "video/x-msvideo":
		return "avi"
	case "video/x-ms-wmv":
		return "wmv"
	case "video/3gpp":
		return "3gp"

	// Audio
	case "audio/mpeg":
		return "mp3"
	case "audio/wav":
		return "wav"
	case "audio/ogg":
		return "ogg"
	case "audio/flac":
		return "flac"
	case "audio/aiff":
		return "aiff"
	case "audio/x-ms-wma":
		return "wma"
	case "audio/x-ape":
		return "ape"

	// Documents
	case "application/pdf":
		return "pdf"
	case "application/zip":
		return "zip"
	case "application/vnd.ms-office":
		return "doc" // Could be doc, xls, ppt - defaulting to doc
	case "application/rtf":
		return "rtf"
	case "text/xml":
		return "xml"
	case "text/html":
		return "html"
	case "application/x-bzip2":
		return "bz2"
	case "application/gzip":
		return "gz"
	case "application/x-rar-compressed":
		return "rar"
	case "application/x-7z-compressed":
		return "7z"

	default:
		// Try to extract extension from MIME type
		parts := strings.Split(mimeType, "/")
		if len(parts) == 2 {
			subtype := parts[1]
			// Handle special cases like "x-" prefix
			subtype = strings.TrimPrefix(subtype, "x-")
			// Handle vnd. prefix (vendor-specific)
			if strings.HasPrefix(subtype, "vnd.") {
				return "bin"
			}
			return subtype
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

	mediaType := parts[0]
	switch mediaType {
	case "image":
		return "image"
	case "video":
		return "video"
	case "audio":
		return "audio"
	case "text", "application":
		// Some text and application types should be treated as documents
		return "document"
	default:
		return "document"
	}
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
