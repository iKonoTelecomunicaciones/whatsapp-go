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

package msgconv

import (
	"context"
	"encoding/base64"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/event"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func (mc *MessageConverter) convertTextMessage(
	ctx context.Context, msg *waE2E.Message,
) (part *bridgev2.ConvertedMessagePart, contextInfo *waE2E.ContextInfo) {
	part = &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    msg.GetConversation(),
		},
	}
	if len(msg.GetExtendedTextMessage().GetText()) > 0 {
		part.Content.Body = msg.GetExtendedTextMessage().GetText()
	}
	contextInfo = msg.GetExtendedTextMessage().GetContextInfo()
	mc.parseFormatting(part.Content, false, false)
	part.Content.BeeperLinkPreviews = mc.convertURLPreviewToBeeper(ctx, msg.GetExtendedTextMessage())
	return
}

func (mc *MessageConverter) convertUnknownMessage(ctx context.Context, msg *waE2E.Message) (*bridgev2.ConvertedMessagePart, *waE2E.ContextInfo) {
	data, _ := proto.Marshal(msg)
	encodedMsg := base64.StdEncoding.EncodeToString(data)
	extra := make(map[string]any)
	if len(encodedMsg) < 16*1024 {
		extra["fi.mau.whatsapp.unsupported_message_data"] = encodedMsg
	}
	return &bridgev2.ConvertedMessagePart{
		Type: event.EventMessage,
		Content: &event.MessageEventContent{
			MsgType: event.MsgNotice,
			Body:    "Unknown message type, please view it on the WhatsApp app",
		},
		Extra: extra,
	}, nil
}
