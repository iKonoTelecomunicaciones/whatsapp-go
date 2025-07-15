package connector

import (
	"context"
	"errors"
	"strings"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

var ErrBroadcastSendDisabled = bridgev2.WrapErrorInStatus(
	errors.New(
		"sending status messages is disabled",
	)).WithErrorAsMessage().WithIsCertain(true).WithSendNotice(true).WithErrorReason(
	event.MessageStatusUnsupported,
)
var ErrBroadcastReactionUnsupported = bridgev2.WrapErrorInStatus(errors.New(
	"reacting to status messages is not currently supported",
)).WithErrorAsMessage().WithIsCertain(true).WithSendNotice(true).WithErrorReason(
	event.MessageStatusUnsupported,
)

func (whatsappClient *WhatsappCloudClient) handleConvertedMatrixMessage(
	ctx context.Context,
	msg *bridgev2.MatrixMessage,
	waMsg *waE2E.Message,
	req *whatsmeow.SendRequestExtra,
) (*bridgev2.MatrixMessageResponse, error) {
	if req == nil {
		req = &whatsmeow.SendRequestExtra{}
	}
	if strings.HasPrefix(string(msg.InputTransactionID), whatsmeow.WebMessageIDPrefix) {
		req.ID = types.MessageID(msg.InputTransactionID)
	} else {
		req.ID = whatsappClient.Client.GenerateMessageID()
	}

	chatJID, err := waid.ParsePortalID(msg.Portal.ID)
	if err != nil {
		return nil, err
	}
	if chatJID == types.StatusBroadcastJID && whatsappClient.Main.Config.DisableStatusBroadcastSend {
		return nil, ErrBroadcastSendDisabled
	}
	wrappedMsgID := waid.MakeMessageID(chatJID, whatsappClient.JID, req.ID)
	wrappedMsgID2 := waid.MakeMessageID(chatJID, whatsappClient.GetStore().GetLID(), req.ID)
	msg.AddPendingToIgnore(networkid.TransactionID(wrappedMsgID))
	msg.AddPendingToIgnore(networkid.TransactionID(wrappedMsgID2))
	resp, err := whatsappClient.Client.SendMessage(ctx, chatJID, waMsg, *req)
	if err != nil {
		return nil, err
	}
	var pickedMessageID networkid.MessageID
	if resp.Sender == whatsappClient.GetStore().GetLID() {
		pickedMessageID = wrappedMsgID2
		msg.RemovePending(networkid.TransactionID(wrappedMsgID))
	} else {
		pickedMessageID = wrappedMsgID
		msg.RemovePending(networkid.TransactionID(wrappedMsgID2))
	}
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:        pickedMessageID,
			SenderID:  waid.MakeUserID(resp.Sender),
			Timestamp: resp.Timestamp,
			Metadata: &waid.MessageMetadata{
				SenderDeviceID: whatsappClient.JID.Device,
			},
		},
		StreamOrder:   resp.Timestamp.Unix(),
		RemovePending: networkid.TransactionID(pickedMessageID),
	}, nil
}
