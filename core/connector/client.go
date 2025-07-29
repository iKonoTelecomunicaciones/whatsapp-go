package connector

import (
	"context"
	"fmt"

	"github.com/iKonoTelecomunicaciones/go/bridgev2"
	"github.com/iKonoTelecomunicaciones/go/bridgev2/networkid"
	"github.com/iKonoTelecomunicaciones/go/event"
)

type WhatsappCloudClient struct {
	Main      *WhatsappCloudConnector
	UserLogin *bridgev2.UserLogin
}

func (whatsappClient *WhatsappCloudClient) Connect(ctx context.Context) {
	// Method to establish a connection to the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

func (whatsappClient *WhatsappCloudClient) Disconnect() {
	// Method to disconnect the whatsappClient.
	// For now, we'll return nil to satisfy the interface.
	return
}

func (whatsappClient *WhatsappCloudClient) GetCapabilities(
	ctx context.Context,
	portal *bridgev2.Portal,
) *event.RoomFeatures {
	// Method to get the capabilities of the whatsappClient.
	return nil
}

func (whatsappClient *WhatsappCloudClient) HandleMatrixMessage(
	ctx context.Context,
	msg *bridgev2.MatrixMessage,
) (*bridgev2.MatrixMessageResponse, error) {
	chatboxMsg, err := whatsappClient.Main.MsgConv.ToWhatsApp(
		ctx,
		msg.Event,
		msg.Content,
		msg.ReplyTo,
		msg.ThreadRoot,
		msg.Portal,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message: %w", err)
	}
	return whatsappClient.handleConvertedMatrixMessage(ctx, chatboxMsg)
}

func (whatsappClient *WhatsappCloudClient) IsThisUser(
	_ context.Context, userID networkid.UserID,
) bool {
	return userID == networkid.UserID(whatsappClient.UserLogin.ID)
}

func (whatsappClient *WhatsappCloudClient) GetUserInfo(
	ctx context.Context, ghost *bridgev2.Ghost,
) (*bridgev2.UserInfo, error) {
	// Method to get user information from the whatsappClient.
	// For now, we'll return an empty UserInfo to satisfy the interface.

	return &bridgev2.UserInfo{}, nil
}

func (whatsappClient *WhatsappCloudClient) getChatInfo(
	ctx context.Context, portalID networkid.PortalID) (wrapped *bridgev2.ChatInfo, err error,
) {
	if portalID == "" {
		return nil, fmt.Errorf("portalID cannot be empty")
	}

	Name := fmt.Sprintf("WhatsApp Cloud Portal: %s", portalID)

	wrapped = &bridgev2.ChatInfo{
		Name: &Name,
	}

	return wrapped, nil
}

func (whatsappClient *WhatsappCloudClient) GetChatInfo(
	ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error,
) {
	if portal.ID == "" {
		return nil, fmt.Errorf("portalID cannot be empty")
	}
	return whatsappClient.getChatInfo(ctx, portal.ID)
}

func (whatsappClient *WhatsappCloudClient) IsLoggedIn() bool {
	return whatsappClient.UserLogin != nil && whatsappClient.UserLogin.Client.IsLoggedIn()
}

func (whatsappClient *WhatsappCloudClient) LogoutRemote(ctx context.Context) {
	// Method to log out the user from the remote service.
	if cli := whatsappClient.UserLogin.Client; cli != nil {
		cli.Disconnect()
	}
	whatsappClient.UserLogin.Client = nil
}
