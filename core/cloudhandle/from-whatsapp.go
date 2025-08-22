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
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
	_ "golang.org/x/image/webp"
)

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyIntent
	contextKeyPortal
)

func getPortal(ctx context.Context) *bridgev2.Portal {
	return ctx.Value(contextKeyPortal).(*bridgev2.Portal)
}

var failedCommentPart = &bridgev2.ConvertedMessagePart{
	Type: event.EventMessage,
	Content: &event.MessageEventContent{
		Body:    "Failed to decrypt comment",
		MsgType: event.MsgNotice,
	},
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

	log.Debug().Msgf("Getting contextInfo: %v", contextInfo)

	return cm
}
