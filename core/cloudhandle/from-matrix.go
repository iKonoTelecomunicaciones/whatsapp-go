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
	"image"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/go/format"
	"github.com/iKonoTelecomunicaciones/go/id"
	"github.com/rs/zerolog"

	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
)

// ToWhatsApp converts a Matrix event into a WhatsApp-compatible message format.
// It handles different message types and prepares the message for sending.
func (mc *MessageConverter) ToWhatsApp(
	ctx context.Context,
	evt *event.Event,
	content *event.MessageEventContent,
	replyTo,
	threadRoot *database.Message,
	portal *bridgev2.Portal,
) (*bridgev2.MatrixMessage, error) {
	if evt.Type == event.EventSticker {
		content.MsgType = event.MessageType(event.EventSticker.Type)
	}

	message := &bridgev2.MatrixMessage{}

	switch content.MsgType {
	case event.MsgText:
		message = mc.constructTextMessage(ctx, content, evt, portal)
	case event.MessageType(event.EventSticker.Type), event.MsgImage, event.MsgVideo, event.MsgAudio, event.MsgFile:
		zerolog.Ctx(ctx).Debug().Str("msgtype", string(content.MsgType)).Msg("Processing media message")
		zerolog.Ctx(ctx).Error().Str("content.URL", string(content.URL)).Msg("Processing media message")
		media_id, err := mc.GetMediaIDWhatsApp(ctx, string(content.URL))

		if err != nil {
			return nil, fmt.Errorf("failed to get media for WhatsApp: %w", err)
		}

		message = mc.constructMediaMessage(ctx, content, evt, portal, media_id)
	default:
		return nil, fmt.Errorf("%w %s", bridgev2.ErrUnsupportedMessageType, content.MsgType)
	}

	return message, nil
}

// parseText extracts the plain text from a message's content,
// parsing HTML if available and extracting any user mentions.
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

// constructTextMessage builds a text message object from the given content.
// It parses the text and mentions, then wraps them in a MatrixMessage struct.
func (mc *MessageConverter) constructTextMessage(
	ctx context.Context,
	content *event.MessageEventContent,
	evt *event.Event,
	portal *bridgev2.Portal,
) *bridgev2.MatrixMessage {
	text, mentions := mc.parseText(ctx, content)
	if len(mentions) > 0 || len(text) > 0 {
		zerolog.Ctx(ctx).Debug().
			Strs("mentions", mentions).
			Msg("Found mentions in text message")
	}

	content.Body = text
	matrix_message := &bridgev2.MatrixMessage{}
	matrix_message.Event = evt
	matrix_message.Portal = portal
	matrix_message.Content = content

	return matrix_message
}

// Given a mxc url, it gets the media from the mxc server.
//
// Parameters:
//
//	ctx : The context of the request.
//	mxc: The mxc url of the media to retrieve.
//
// Returns:
//
//	The media file as a byte slice,
//	The original filename of the media,
func (mc *MessageConverter) getMedia(ctx context.Context, mxc string) ([]byte, string, error) {
	return nil, "", nil
}

// Given a mxc url, it gets the media and update it to meta to get an id.
//
// Parameters:
//
//	ctx : The context of the request.
//	mxc : The mxc url of the media.
//
// Returns:
//
//	The id of the media or an error if something went wrong.
func (mc *MessageConverter) GetMediaIDWhatsApp(
	ctx context.Context,
	mxc string,
) (string, error) {
	// Implement media retrieval logic here
	return "", nil
}

func (mc *MessageConverter) constructMediaMessage(
	ctx context.Context,
	content *event.MessageEventContent,
	evt *event.Event,
	portal *bridgev2.Portal,
	media_id string,
) *bridgev2.MatrixMessage {
	//var caption string
	//if content.FileName != "" && content.Body != content.FileName {
	//	caption, contextInfo.MentionedJID = mc.parseText(ctx, content)
	//}
	switch content.MsgType {
	//case event.MessageType(event.EventSticker.Type):
	//	width := uint32(content.Info.Width)
	//	height := uint32(content.Info.Height)

	//	return &waE2E.Message{
	//		StickerMessage: &waE2E.StickerMessage{
	//			Width:  &width,
	//			Height: &height,

	//			ContextInfo:   contextInfo,
	//			PngThumbnail:  thumbnail,
	//			DirectPath:    proto.String(uploaded.DirectPath),
	//			MediaKey:      uploaded.MediaKey,
	//			Mimetype:      proto.String(mime),
	//			FileEncSHA256: uploaded.FileEncSHA256,
	//			FileSHA256:    uploaded.FileSHA256,
	//			FileLength:    proto.Uint64(uploaded.FileLength),
	//			URL:           proto.String(uploaded.URL),
	//		},
	//	}
	//case event.MsgAudio:
	//	waveform, seconds := getAudioInfo(content)

	//	return &waE2E.Message{
	//		AudioMessage: &waE2E.AudioMessage{
	//			Seconds:  &seconds,
	//			Waveform: waveform,
	//			PTT:      proto.Bool(content.MSC3245Voice != nil),

	//			URL:           proto.String(uploaded.URL),
	//			DirectPath:    proto.String(uploaded.DirectPath),
	//			MediaKey:      uploaded.MediaKey,
	//			Mimetype:      proto.String(mime),
	//			FileEncSHA256: uploaded.FileEncSHA256,
	//			FileSHA256:    uploaded.FileSHA256,
	//			FileLength:    proto.Uint64(uploaded.FileLength),
	//			ContextInfo:   contextInfo,
	//		},
	//	}
	//case event.MsgImage:
	//	width := uint32(content.Info.Width)
	//	height := uint32(content.Info.Height)

	//	return &waE2E.Message{
	//		ImageMessage: &waE2E.ImageMessage{
	//			Width:  &width,
	//			Height: &height,

	//			Caption:       proto.String(caption),
	//			JPEGThumbnail: thumbnail,
	//			URL:           proto.String(uploaded.URL),
	//			DirectPath:    proto.String(uploaded.DirectPath),
	//			MediaKey:      uploaded.MediaKey,
	//			Mimetype:      proto.String(mime),
	//			FileEncSHA256: uploaded.FileEncSHA256,
	//			FileSHA256:    uploaded.FileSHA256,
	//			FileLength:    proto.Uint64(uploaded.FileLength),
	//			ContextInfo:   contextInfo,
	//		},
	//	}
	//case event.MsgVideo:
	//	isGIF := mime == "video/mp4" && (content.Info.MimeType == "image/gif" || content.Info.MauGIF)

	//	width := uint32(content.Info.Width)
	//	height := uint32(content.Info.Height)
	//	seconds := uint32(content.Info.Duration / 1000)

	//	return &waE2E.Message{
	//		VideoMessage: &waE2E.VideoMessage{
	//			GifPlayback: proto.Bool(isGIF),
	//			Width:       &width,
	//			Height:      &height,
	//			Seconds:     &seconds,

	//			Caption:       proto.String(caption),
	//			JPEGThumbnail: thumbnail,
	//			URL:           proto.String(uploaded.URL),
	//			DirectPath:    proto.String(uploaded.DirectPath),
	//			MediaKey:      uploaded.MediaKey,
	//			Mimetype:      proto.String(mime),
	//			FileEncSHA256: uploaded.FileEncSHA256,
	//			FileSHA256:    uploaded.FileSHA256,
	//			FileLength:    proto.Uint64(uploaded.FileLength),
	//			ContextInfo:   contextInfo,
	//		},
	//	}
	//case event.MsgFile:
	//	fileName := content.FileName
	//	if fileName == "" {
	//		fileName = content.Body
	//	}

	//	msg := &waE2E.Message{
	//		DocumentMessage: &waE2E.DocumentMessage{
	//			FileName: proto.String(fileName),

	//			Caption:       proto.String(caption),
	//			JPEGThumbnail: thumbnail,
	//			URL:           proto.String(uploaded.URL),
	//			DirectPath:    proto.String(uploaded.DirectPath),
	//			MediaKey:      uploaded.MediaKey,
	//			Mimetype:      proto.String(mime),
	//			FileEncSHA256: uploaded.FileEncSHA256,
	//			FileSHA256:    uploaded.FileSHA256,
	//			FileLength:    proto.Uint64(uploaded.FileLength),
	//			ContextInfo:   contextInfo,
	//		},
	//	}
	//	if msg.GetDocumentMessage().GetCaption() != "" {
	//		msg.DocumentWithCaptionMessage = &waE2E.FutureProofMessage{
	//			Message: &waE2E.Message{
	//				DocumentMessage: msg.DocumentMessage,
	//			},
	//		}
	//		msg.DocumentMessage = nil
	//	}
	//	return msg
	default:
		return nil
	}
}

// convertPill handles the conversion of a Matrix user mention (a "pill")
// into a format that WhatsApp can understand, typically an @-mention with a JID.
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
	var jid string
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
	*mentions = append(*mentions, jid)
	return fmt.Sprintf("@%s", jid)
}

type PaddedImage struct {
	image.Image
	Size       int
	OffsetX    int
	OffsetY    int
	RealWidth  int
	RealHeight int
}
