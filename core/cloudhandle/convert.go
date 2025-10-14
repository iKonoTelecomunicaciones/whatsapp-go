// mautrix-whatsapp - A Matrix-WhatsApp puppeting bridge.
// Copyright (C) 2024 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package cloudhandle

import (
	"context"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/rs/zerolog"
)

func (mc *MessageConverter) convertMediaMessage(
	ctx context.Context,
	msg *types.CloudValue,
	typeName string,
	client *WhatsappCloudClient,
	intent bridgev2.MatrixAPI,
	roomID *id.RoomID,
) (part *bridgev2.ConvertedMessagePart, contextInfo *CloudMessageInfo) {
	log := zerolog.Ctx(ctx).With().Str("ConvertMediaMessage", msg.Messages[0].ID).Logger()
	if msg.Messages == nil || len(msg.Messages) == 0 || len(msg.Messages) > 1 {
		log.Warn().Msg("No messages found in CloudValue or multiple messages found")
		return nil, nil
	}

	if msg.Contacts == nil || len(msg.Contacts) == 0 || len(msg.Contacts) > 1 {
		log.Warn().Msg("No contacts found in CloudValue or multiple contacts found")
		return nil, nil
	}

	var mxcURL string

	if preuploadedMXC := ctx.Value("preuploadedMXC"); preuploadedMXC != nil {
		mxcURL = preuploadedMXC.(string)
		log.Info().Str("mxc_url", mxcURL).Msg("Using preuploadedMXC from context")
	} else {
		log.Warn().Msg("No preuploadedMXC found in context, media will not be available")
		return nil, nil
	}

	preparedMedia := prepareMediaMessage(msg)
	preparedMedia.TypeDescription = typeName
	preparedMedia.URL = id.ContentURIString(mxcURL)

	if preparedMedia.FileName != "" && preparedMedia.Body != preparedMedia.FileName {
		mc.parseFormatting(preparedMedia.MessageEventContent, false, false)
	}
	contextInfo = preparedMedia.ContextInfo
	part = &bridgev2.ConvertedMessagePart{
		Type:    event.EventMessage,
		Content: preparedMedia.MessageEventContent,
	}

	return
}

// convertTextMessage converts a WhatsApp Cloud API text message into a bridge-compatible format.
// It extracts the message body, timestamp, sender information, and other metadata.
func (mc *MessageConverter) convertTextMessage(
	ctx context.Context, msg *types.CloudValue,
) (part *bridgev2.ConvertedMessagePart, contextInfo *CloudMessageInfo) {
	part = &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgText,
		},
	}
	if len(msg.Messages[0].Text.Body) > 0 {
		part.Content.Body = msg.Messages[0].Text.Body
	}
	var timestamp time.Time
	var err error
	if msg.Messages[0].TimeStamp != "" {
		timestamp, err = time.Parse(time.RFC3339, msg.Messages[0].TimeStamp)
		if err != nil {
			timestamp = time.Time{}
		}
	}
	contextInfo = &CloudMessageInfo{
		ID:        string(msg.Messages[0].ID),
		Sender:    string(msg.Messages[0].From),
		Type:      string(msg.Messages[0].Type),
		PushName:  string(msg.Contacts[0].Profile.Name),
		Timestamp: timestamp,
		Category:  string(msg.Messages[0].Type),
	}

	mc.parseFormatting(part.Content, false, false)
	return
}

// convertUnknownMessage handles messages of an unknown or unsupported type.
// It returns a generic notice message to inform the user to check the message on their device.
func (mc *MessageConverter) convertUnknownMessage(
	ctx context.Context, msg *types.CloudValue,
) (*bridgev2.ConvertedMessagePart, *CloudMessageInfo) {

	return &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgNotice,
			Body:    "Unknown message type, please view it on the WhatsApp app",
		},
	}, nil
}
