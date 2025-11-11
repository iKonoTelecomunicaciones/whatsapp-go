package cloudhandle

import (
	"context"
	"fmt"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/format"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/connector/whatsappclouddb"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
)

type AnimatedStickerConfig struct {
	Target string `yaml:"target"`
	Args   struct {
		Width  int `yaml:"width"`
		Height int `yaml:"height"`
		FPS    int `yaml:"fps"`
	} `yaml:"args"`
}

type MessageConverter struct {
	Bridge                *bridgev2.Bridge
	DB                    *whatsappclouddb.Database
	MaxFileSize           int64
	HTMLParser            *format.HTMLParser
	AnimatedStickerConfig AnimatedStickerConfig
	FetchURLPreviews      bool
	ExtEvPolls            bool
	DisableViewOnce       bool
	DirectMedia           bool
	OldMediaSuffix        string
}

type WhatsappCloudClient struct {
	Main      *WhatsappCloudConnector
	UserLogin *bridgev2.UserLogin
}

// ToMatrix converts an incoming WhatsApp Cloud message into a `ConvertedMessage`
// that can be processed by the bridge and sent to Matrix. It determines the
// message type and calls the appropriate conversion helper function.
func (mc *MessageConverter) ToMatrix(
	ctx context.Context,
	portal *bridgev2.Portal,
	client *WhatsappCloudClient,
	intent bridgev2.MatrixAPI,
	waMsg *types.CloudValue,
	info *CloudMessageInfo,
	isViewOnce bool,
	previouslyConvertedPart *bridgev2.ConvertedMessagePart,
) *bridgev2.ConvertedMessage {
	log := zerolog.Ctx(ctx).With().Str("MessageConverter", waMsg.Messages[0].ID).Logger()
	ctx = context.WithValue(ctx, contextKeyClient, client)
	ctx = context.WithValue(ctx, contextKeyIntent, intent)
	ctx = context.WithValue(ctx, contextKeyPortal, portal)

	var part *bridgev2.ConvertedMessagePart
	var status_part *bridgev2.ConvertedMessagePart
	var contextInfo *CloudMessageInfo

	switch {
	case waMsg.Messages[0].Text.Body != "":
		part, contextInfo = mc.convertTextMessage(ctx, waMsg)
	default:
		part, contextInfo = mc.convertUnknownMessage(ctx, waMsg)
	}

	part.Content.Mentions = &event.Mentions{}
	if part.DBMetadata == nil {
		part.DBMetadata = &waid.MessageMetadata{}
	}
	// Convert info.Sender from string to uint16
	var senderDeviceID uint16
	fmt.Sscanf(info.Sender, "%d", &senderDeviceID)

	if part.DBMetadata == nil {
		part.DBMetadata = &waid.MessageMetadata{}
	}
	dbMeta := part.DBMetadata.(*waid.MessageMetadata)
	dbMeta.SenderDeviceID = senderDeviceID

	parts_to_send := []*bridgev2.ConvertedMessagePart{part}
	if status_part != nil {
		parts_to_send = append([]*bridgev2.ConvertedMessagePart{status_part}, parts_to_send...)
	}

	cm := &bridgev2.ConvertedMessage{
		Parts: parts_to_send,
	}

	log.Debug().Msgf("contextInfo: %v", contextInfo)

	return cm
}
