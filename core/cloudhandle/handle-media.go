package cloudhandle

import (
	"fmt"
	"time"

	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
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
