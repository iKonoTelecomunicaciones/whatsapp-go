package connector

import (
	"context"
	"errors"
	"time"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/database"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
	"github.com/iKonoTelecomunicaciones/whatsapp-go/core/waid"
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
) (*bridgev2.MatrixMessageResponse, error) {

	// TODO: Send the message to WhatsApp
	var pickedMessageID networkid.MessageID

	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:        pickedMessageID,
			SenderID:  waid.MakeUserID(""),
			Timestamp: time.Now(),
			Metadata: &waid.MessageMetadata{
				SenderDeviceID: 0,
			},
		},
		StreamOrder:   time.Now().Unix(),
		RemovePending: networkid.TransactionID(pickedMessageID),
	}, nil
}
