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
	"html"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	_ "golang.org/x/image/webp"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
)

type contextKey int

const (
	contextKeyClient contextKey = iota
	contextKeyIntent
	contextKeyPortal
)

func getClient(ctx context.Context) *whatsmeow.Client {
	return ctx.Value(contextKeyClient).(*whatsmeow.Client)
}

func getIntent(ctx context.Context) bridgev2.MatrixAPI {
	return ctx.Value(contextKeyIntent).(bridgev2.MatrixAPI)
}

func getPortal(ctx context.Context) *bridgev2.Portal {
	return ctx.Value(contextKeyPortal).(*bridgev2.Portal)
}

func (mc *MessageConverter) getBasicUserInfo(
	ctx context.Context, user types.JID,
) (id.UserID, string, error) {
	ghost, err := mc.Bridge.GetGhostByID(ctx, networkid.UserID(waid.MakeUserID(user)))
	if err != nil {
		return "", "", fmt.Errorf("failed to get ghost by ID: %w", err)
	}
	var pnJID types.JID
	if user.Server == types.DefaultUserServer {
		pnJID = user
	} else if user.Server == types.HiddenUserServer {
		cli := getClient(ctx)
		if user.User == cli.Store.GetLID().User {
			pnJID = cli.Store.GetJID()
		} else {
			pnJID, err = cli.Store.LIDs.GetPNForLID(ctx, user)
			if err != nil {
				zerolog.Ctx(ctx).Err(err).
					Stringer("lid", user).
					Msg("Failed to get PN for LID in mention bridging")
			}
		}
	}
	if !pnJID.IsEmpty() {
		login := mc.Bridge.GetCachedUserLoginByID(networkid.UserLoginID(waid.MakeUserLoginID(pnJID)))
		if login != nil {
			return login.UserMXID, ghost.Name, nil
		}
	}
	return ghost.Intent.GetMXID(), ghost.Name, nil
}

func (mc *MessageConverter) addMentions(
	ctx context.Context, mentionedJID []string, into *event.MessageEventContent,
) {
	if len(mentionedJID) == 0 {
		return
	}
	into.EnsureHasHTML()
	for _, jid := range mentionedJID {
		parsed, err := types.ParseJID(jid)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Str("jid", jid).Msg("Failed to parse mentioned JID")
			continue
		}
		mxid, displayname, err := mc.getBasicUserInfo(ctx, parsed)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Str("jid", jid).Msg("Failed to get user info")
			continue
		}
		into.Mentions.UserIDs = append(into.Mentions.UserIDs, mxid)
		mentionText := "@" + parsed.User
		into.Body = strings.ReplaceAll(into.Body, mentionText, displayname)
		into.FormattedBody = strings.ReplaceAll(
			into.FormattedBody,
			mentionText,
			fmt.Sprintf(
				`<a href="%s">%s</a>`, mxid.URI().MatrixToURL(), html.EscapeString(displayname),
			),
		)
	}
}

var failedCommentPart = &bridgev2.ConvertedMessagePart{
	Type: event.EventMessage,
	Content: &event.MessageEventContent{
		Body:    "Failed to decrypt comment",
		MsgType: event.MsgNotice,
	},
}

func (mc *MessageConverter) ToMatrix(
	ctx context.Context,
	portal *bridgev2.Portal,
	client *whatsmeow.Client,
	intent bridgev2.MatrixAPI,
	waMsg *waE2E.Message,
	info *types.MessageInfo,
	isViewOnce bool,
	previouslyConvertedPart *bridgev2.ConvertedMessagePart,
) *bridgev2.ConvertedMessage {
	ctx = context.WithValue(ctx, contextKeyClient, client)
	ctx = context.WithValue(ctx, contextKeyIntent, intent)
	ctx = context.WithValue(ctx, contextKeyPortal, portal)

	var part *bridgev2.ConvertedMessagePart
	var status_part *bridgev2.ConvertedMessagePart
	var contextInfo *waE2E.ContextInfo
	switch {
	case waMsg.Conversation != nil:
		part, contextInfo = mc.convertTextMessage(ctx, waMsg)
	default:
		part, contextInfo = mc.convertUnknownMessage(ctx, waMsg)
	}

	part.Content.Mentions = &event.Mentions{}
	if part.DBMetadata == nil {
		part.DBMetadata = &waid.MessageMetadata{}
	}
	dbMeta := part.DBMetadata.(*waid.MessageMetadata)
	dbMeta.SenderDeviceID = info.Sender.Device
	if info.IsIncomingBroadcast() {
		dbMeta.BroadcastListJID = &info.Chat
		if part.Extra == nil {
			part.Extra = map[string]any{}
		}
		part.Extra["fi.mau.whatsapp.source_broadcast_list"] = info.Chat.String()
	}
	mc.addMentions(ctx, contextInfo.GetMentionedJID(), part.Content)

	parts_to_send := []*bridgev2.ConvertedMessagePart{part}
	if status_part != nil {
		parts_to_send = append([]*bridgev2.ConvertedMessagePart{status_part}, parts_to_send...)
	}

	cm := &bridgev2.ConvertedMessage{
		Parts: parts_to_send,
	}

	if contextInfo.GetStanzaID() != "" && status_part == nil {
		pcp, _ := types.ParseJID(contextInfo.GetParticipant())
		chat, _ := types.ParseJID(contextInfo.GetRemoteJID())
		if chat.IsEmpty() {
			portalID := networkid.PortalID(portal.ID)
			chat, _ = waid.ParsePortalID(portalID)
		}
		cm.ReplyTo = &networkid.MessageOptionalPartID{
			MessageID: networkid.MessageID(waid.MakeMessageID(chat, pcp, contextInfo.GetStanzaID())),
		}
	}
	if contextInfo.GetIsForwarded() {
		hasCaption := part.Content.FileName != "" && part.Content.FileName != part.Content.Body
		isMedia := part.Content.MsgType.IsMedia()
		isText := part.Content.MsgType.IsText()
		if isMedia && !hasCaption {
			part.Content.FileName = part.Content.Body
			part.Content.Body = "↷ Forwarded"
			part.Content.Format = event.FormatHTML
			part.Content.FormattedBody = "<p data-mx-forwarded-notice><em>↷ Forwarded</em></p>"
		} else if isText || isMedia {
			part.Content.EnsureHasHTML()
			part.Content.Body = "↷ Forwarded\n\n" + part.Content.Body
			part.Content.FormattedBody = "<p data-mx-forwarded-notice><em>↷ Forwarded</em></p>" + part.Content.FormattedBody
		}
	}
	commentTarget := waMsg.GetEncCommentMessage().GetTargetMessageKey()
	if commentTarget == nil {
		commentTarget = waMsg.GetCommentMessage().GetTargetMessageKey()
	}
	if commentTarget != nil {
		pcp, _ := types.ParseJID(commentTarget.GetParticipant())
		chat, _ := types.ParseJID(commentTarget.GetRemoteJID())
		if chat.IsEmpty() {
			chat, _ = waid.ParsePortalID(portal.ID)
		}
		cm.ThreadRoot = ptr.Ptr(waid.MakeMessageID(chat, pcp, commentTarget.GetID()))
	}

	return cm
}
