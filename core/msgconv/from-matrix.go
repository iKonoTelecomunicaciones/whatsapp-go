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
	"fmt"
	"image"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/format"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
)

func (mc *MessageConverter) generateContextInfo(
	ctx context.Context,
	replyTo *database.Message,
	portal *bridgev2.Portal,
) *waE2E.ContextInfo {
	contextInfo := &waE2E.ContextInfo{}
	if replyTo != nil {
		msgID, err := waid.ParseMessageID(replyTo.ID)
		if err == nil {
			contextInfo.StanzaID = proto.String(msgID.ID)
			contextInfo.Participant = proto.String(msgID.Sender.String())
			contextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		} else {
			zerolog.Ctx(ctx).Warn().Err(err).
				Stringer("reply_to_event_id", replyTo.MXID).
				Str("reply_to_message_id", string(replyTo.ID)).
				Msg("Failed to parse reply to message ID")
		}
	}
	if portal.Disappear.Timer > 0 {
		contextInfo.Expiration = ptr.Ptr(uint32(portal.Disappear.Timer.Seconds()))
		setAt := portal.Metadata.(*waid.PortalMetadata).DisappearingTimerSetAt
		if setAt > 0 {
			contextInfo.EphemeralSettingTimestamp = ptr.Ptr(setAt)
		}
	}
	return contextInfo
}

func (mc *MessageConverter) ToWhatsApp(
	ctx context.Context,
	client *whatsmeow.Client,
	evt *event.Event,
	content *event.MessageEventContent,
	replyTo,
	threadRoot *database.Message,
	portal *bridgev2.Portal,
) (*waE2E.Message, *whatsmeow.SendRequestExtra, error) {
	ctx = context.WithValue(ctx, contextKeyClient, client)
	ctx = context.WithValue(ctx, contextKeyPortal, portal)
	if evt.Type == event.EventSticker {
		content.MsgType = event.MessageType(event.EventSticker.Type)
	}

	message := &waE2E.Message{}
	contextInfo := mc.generateContextInfo(ctx, replyTo, portal)

	switch content.MsgType {
	case event.MsgText:
		message = mc.constructTextMessage(ctx, content, contextInfo)
	default:
		return nil, nil, fmt.Errorf("%w %s", bridgev2.ErrUnsupportedMessageType, content.MsgType)
	}
	extra := &whatsmeow.SendRequestExtra{}
	if portal.Metadata.(*waid.PortalMetadata).CommunityAnnouncementGroup {
		if threadRoot != nil {
			parsedID, err := waid.ParseMessageID(threadRoot.ID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse message ID: %w", err)
			}
			rootMsgInfo := MessageIDToInfo(client, parsedID)
			message, err = client.EncryptComment(ctx, rootMsgInfo, message)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to encrypt comment: %w", err)
			}
			lid := parsedID.Sender
			if lid.Server == types.DefaultUserServer {
				lid, err = client.Store.LIDs.GetLIDForPN(ctx, parsedID.Sender)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get LID for PN: %w", err)
				}
			}
			extra.Meta = &types.MsgMetaInfo{
				ThreadMessageID:        parsedID.ID,
				ThreadMessageSenderJID: lid,
				DeprecatedLIDSession:   ptr.Ptr(false),
			}
		} else {
			message.MessageContextInfo = &waE2E.MessageContextInfo{
				MessageSecret: random.Bytes(32),
			}
		}
	}
	return message, extra, nil
}

func (mc *MessageConverter) parseText(
	ctx context.Context,
	content *event.MessageEventContent,
) (text string, mentions []string) {
	mentions = make([]string, 0)

	parseCtx := format.NewContext(ctx)
	parseCtx.ReturnData["allowed_mentions"] = content.Mentions
	parseCtx.ReturnData["output_mentions"] = &mentions
	if content.Format == event.FormatHTML {
		text = mc.HTMLParser.Parse(content.FormattedBody, parseCtx)
	} else {
		text = content.Body
	}
	return
}

func (mc *MessageConverter) constructTextMessage(
	ctx context.Context,
	content *event.MessageEventContent,
	contextInfo *waE2E.ContextInfo,
) *waE2E.Message {
	text, mentions := mc.parseText(ctx, content)
	if len(mentions) > 0 {
		contextInfo.MentionedJID = mentions
	}
	etm := &waE2E.ExtendedTextMessage{
		Text:        proto.String(text),
		ContextInfo: contextInfo,
	}
	mc.convertURLPreviewToWhatsApp(ctx, content, etm)

	return &waE2E.Message{ExtendedTextMessage: etm}
}

func (mc *MessageConverter) convertPill(
	displayname, mxid, eventID string, ctx format.Context,
) string {
	if len(mxid) == 0 || mxid[0] != '@' {
		return format.DefaultPillConverter(displayname, mxid, eventID, ctx)
	}
	allowedMentions, _ := ctx.ReturnData["allowed_mentions"].(*event.Mentions)
	if allowedMentions != nil && !allowedMentions.Has(id.UserID(mxid)) {
		return displayname
	}
	var jid types.JID
	ghost, err := mc.Bridge.GetGhostByMXID(ctx.Ctx, id.UserID(mxid))
	if err != nil {
		zerolog.Ctx(ctx.Ctx).Err(err).Str("mxid", mxid).Msg("Failed to get ghost for mention")
		return displayname
	} else if ghost != nil {
		jid = waid.ParseUserID(ghost.ID)
	} else if user, err := mc.Bridge.GetExistingUserByMXID(ctx.Ctx, id.UserID(mxid)); err != nil {
		zerolog.Ctx(ctx.Ctx).Err(err).Str("mxid", mxid).Msg("Failed to get user for mention")
		return displayname
	} else if user != nil {
		portal := getPortal(ctx.Ctx)
		login, _, _ := portal.FindPreferredLogin(ctx.Ctx, user, false)
		if login == nil {
			return displayname
		}
		jid = waid.ParseUserLoginID(login.ID, 0)
	} else {
		return displayname
	}
	mentions := ctx.ReturnData["output_mentions"].(*[]string)
	*mentions = append(*mentions, jid.String())
	return fmt.Sprintf("@%s", jid.User)
}

type PaddedImage struct {
	image.Image
	Size       int
	OffsetX    int
	OffsetY    int
	RealWidth  int
	RealHeight int
}
