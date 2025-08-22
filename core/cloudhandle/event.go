package cloudhandle

import (
	"context"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/types"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"github.com/rs/zerolog"
)

type MessageInfoWrapper struct {
	Info           CloudMessageInfo
	whatsappClient *WhatsappCloudClient
}

type WAMessageEvent struct {
	*MessageInfoWrapper
	Message  types.CloudValue
	MsgEvent types.CloudEvent

	parsedMessageType             string
	isUndecryptableUpsertSubEvent bool
	postHandle                    func()
}

func (evt *MessageInfoWrapper) AddLogContext(c zerolog.Context) zerolog.Context {
	return c.Str("message_id", evt.Info.ID).Str("sender_id", evt.Info.Sender)
}

func (evt *MessageInfoWrapper) GetPortalKey() networkid.PortalKey {
	return waid.MakePortalKey(evt.Info.Sender)
}

func (evt *MessageInfoWrapper) GetSender() bridgev2.EventSender {
	userID := evt.whatsappClient.UserLogin.UserMXID
	return bridgev2.EventSender{
		IsFromMe:    networkid.UserLoginID(userID) == networkid.UserLoginID(evt.Info.Sender),
		Sender:      waid.MakeUserID(evt.Info.Sender),
		SenderLogin: waid.MakeUserLoginID(string(userID)),
	}
}

func (evt *WAMessageEvent) ConvertMessage(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI) (*bridgev2.ConvertedMessage, error) {
	converted := evt.whatsappClient.Main.MsgConv.ToMatrix(
		ctx, portal, evt.whatsappClient, intent, &evt.Message, &evt.Info, evt.isViewOnce(), nil,
	)

	return converted, nil
}

func (evt *WAMessageEvent) isViewOnce() bool {
	return false // TODO: Implement logic to determine if the message is view once
}

func (evt *WAMessageEvent) GetType() bridgev2.RemoteEventType {
	switch evt.parsedMessageType {
	case "reaction", "encrypted reaction":
		return bridgev2.RemoteEventReaction
	case "reaction remove":
		return bridgev2.RemoteEventReactionRemove
	case "edit":
		return bridgev2.RemoteEventEdit
	case "revoke":
		return bridgev2.RemoteEventMessageRemove
	case "ignore":
		return bridgev2.RemoteEventUnknown
	default:
		return bridgev2.RemoteEventMessageUpsert
	}
}

func (evt *MessageInfoWrapper) GetID() networkid.MessageID {
	return waid.MakeMessageID(evt.Info.Chat, evt.Info.Chat, evt.Info.ID)
}
